package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/rakutao/collection-gateway/internal/cache"
	"github.com/rakutao/collection-gateway/internal/domain"
	"github.com/rakutao/collection-gateway/internal/repo"
)

// Signal weight multipliers.
const (
	signalPreference = 5.0
	signalOrder      = 4.0
	signalFavorite   = 3.0
	signalSeller     = 3.0
	signalCart        = 2.5
	signalSearch     = 2.0
	signalBrowse     = 1.0
)

const (
	recCacheTTL   = 30 * time.Minute
	recCachePrefix = "rec:"
)

// RecommendationService computes user weights and fetches personalized recommendations.
type RecommendationService struct {
	prefRepo     *repo.PreferenceRepo
	browseRepo   *repo.BrowsingRepo
	searchRepo   *repo.SearchHistoryRepo
	weightRepo   *repo.RecWeightRepo
	favRepo      FavoriteReader
	cartRepo     CartReader
	orderRepo    OrderReader
	sellerRepo   SellerReader
	esClient     *elasticsearch.Client
	esIndex      string
	cache        *cache.Client
}

// FavoriteReader reads user favorites (for weight computation).
type FavoriteReader interface {
	ListByUser(ctx context.Context, userID int64, limit, offset int) ([]repo.FavoriteItem, int64, error)
}

// CartReader reads user cart items (for weight computation).
type CartReader interface {
	ListByUser(ctx context.Context, userID int64) ([]repo.CartItem, error)
}

// OrderReader reads user orders (for weight computation).
type OrderReader interface {
	ListByUser(ctx context.Context, userID int64, state int, limit, offset int) ([]domain.Order, int64, error)
	GetDetailsByOrderIDs(ctx context.Context, orderIDs []int64) (map[int64][]domain.OrderDetail, error)
}

// SellerReader reads user's followed sellers.
type SellerReader interface {
	ListByUser(ctx context.Context, userID int64, limit, offset int) ([]repo.FollowedSeller, int64, error)
}

// NewRecommendationService creates a new RecommendationService.
func NewRecommendationService(
	prefRepo *repo.PreferenceRepo,
	browseRepo *repo.BrowsingRepo,
	searchRepo *repo.SearchHistoryRepo,
	weightRepo *repo.RecWeightRepo,
	favRepo FavoriteReader,
	cartRepo CartReader,
	orderRepo OrderReader,
	sellerRepo SellerReader,
	esClient *elasticsearch.Client,
	esIndex string,
	cacheClient *cache.Client,
) *RecommendationService {
	return &RecommendationService{
		prefRepo:   prefRepo,
		browseRepo: browseRepo,
		searchRepo: searchRepo,
		weightRepo: weightRepo,
		favRepo:    favRepo,
		cartRepo:   cartRepo,
		orderRepo:  orderRepo,
		sellerRepo: sellerRepo,
		esClient:   esClient,
		esIndex:    esIndex,
		cache:      cacheClient,
	}
}

// timeDecay returns a decay factor based on days since event.
// decay = 1.0 / (1.0 + daysSince * 0.03)
func timeDecay(t time.Time) float32 {
	days := float64(time.Since(t).Hours()) / 24.0
	if days < 0 {
		days = 0
	}
	return float32(1.0 / (1.0 + days*0.03))
}

// weightAccumulator accumulates weights per dimension+value.
type weightAccumulator map[string]float32 // key = "dimension:value"

func (a weightAccumulator) add(dimension, value string, w float32) {
	if value == "" {
		return
	}
	key := dimension + ":" + value
	a[key] += w
}

// ComputeWeights computes recommendation weights for a single user from all signal sources.
func (s *RecommendationService) ComputeWeights(ctx context.Context, userID int64) error {
	acc := make(weightAccumulator)

	// 1. User preferences (no decay)
	prefs, err := s.prefRepo.GetPreferences(ctx, userID)
	if err != nil {
		return fmt.Errorf("prefs: %w", err)
	}
	for _, p := range prefs {
		acc.add("category", p.Category, signalPreference*p.Weight)
	}

	// 2. Browsing history (with decay)
	browses, err := s.browseRepo.ListRecent(ctx, userID, 90)
	if err != nil {
		return fmt.Errorf("browse: %w", err)
	}
	for _, b := range browses {
		d := timeDecay(b.ViewedAt)
		acc.add("category", b.Category, signalBrowse*d)
		acc.add("brand", b.Brand, signalBrowse*d)
		acc.add("seller", b.SellerID, signalBrowse*d)
		acc.add("platform", b.Platform, signalBrowse*d)
	}

	// 3. Search history (with decay)
	searches, err := s.searchRepo.ListRecent(ctx, userID, 90)
	if err != nil {
		return fmt.Errorf("search: %w", err)
	}
	for _, sr := range searches {
		d := timeDecay(sr.CreatedAt)
		kw := sr.KeywordJA
		if kw == "" {
			kw = sr.Keyword
		}
		acc.add("keyword", kw, signalSearch*d)
		acc.add("platform", sr.Platform, signalSearch*d)
	}

	// 4. Favorites (with decay)
	favItems, _, err := s.favRepo.ListByUser(ctx, userID, 200, 0)
	if err != nil {
		return fmt.Errorf("favorites: %w", err)
	}
	for _, f := range favItems {
		d := timeDecay(f.AddedAt)
		// We only have productID, extract platform from ID (format: platform_sourceID)
		platform, _ := parseProductID(f.ProductID)
		acc.add("platform", platform, signalFavorite*d)
	}

	// 5. Cart (with decay)
	cartItems, err := s.cartRepo.ListByUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("cart: %w", err)
	}
	for _, c := range cartItems {
		d := timeDecay(c.UpdatedAt)
		platform, _ := parseProductID(c.ProductID)
		acc.add("platform", platform, signalCart*d)
	}

	// 6. Orders (with decay)
	orders, _, err := s.orderRepo.ListByUser(ctx, userID, -1, 100, 0)
	if err != nil {
		return fmt.Errorf("orders: %w", err)
	}
	if len(orders) > 0 {
		orderIDs := make([]int64, len(orders))
		for i, o := range orders {
			orderIDs[i] = o.ID
		}
		detailMap, err := s.orderRepo.GetDetailsByOrderIDs(ctx, orderIDs)
		if err != nil {
			return fmt.Errorf("order details: %w", err)
		}
		for _, o := range orders {
			d := timeDecay(o.CreatedAt)
			for _, det := range detailMap[o.ID] {
				acc.add("seller", det.SellerID, signalOrder*d)
				acc.add("platform", det.Platform, signalOrder*d)
			}
		}
	}

	// 7. Followed sellers (with decay)
	sellers, _, err := s.sellerRepo.ListByUser(ctx, userID, 200, 0)
	if err != nil {
		return fmt.Errorf("sellers: %w", err)
	}
	for _, fs := range sellers {
		d := timeDecay(fs.FollowedAt)
		acc.add("seller", fs.SellerID, signalSeller*d)
	}

	// Convert accumulator to RecWeight slice
	weights := make([]domain.RecWeight, 0, len(acc))
	for key, w := range acc {
		parts := strings.SplitN(key, ":", 2)
		if len(parts) != 2 || w < 0.01 {
			continue
		}
		weights = append(weights, domain.RecWeight{
			UserID:     userID,
			SignalType: "combined",
			Dimension:  parts[0],
			Value:      parts[1],
			Weight:     w,
		})
	}

	if err := s.weightRepo.UpsertWeights(ctx, userID, weights); err != nil {
		return fmt.Errorf("upsert weights: %w", err)
	}

	// Invalidate recommendation cache
	if s.cache != nil {
		_ = s.cache.Del(ctx, recCachePrefix+fmt.Sprintf("%d", userID))
	}

	return nil
}

// GetRecommendations returns recommendation lists for a user.
// If listType is empty, returns all list types. Otherwise returns the specified type.
func (s *RecommendationService) GetRecommendations(ctx context.Context, userID int64, listType string, refresh bool) ([]domain.RecommendationList, error) {
	cacheKey := recCachePrefix + fmt.Sprintf("%d", userID)
	if listType != "" {
		cacheKey += ":" + listType
	}

	// Check cache (unless refresh requested)
	if !refresh && s.cache != nil {
		data, err := s.cache.Get(ctx, cacheKey)
		if err == nil && data != nil {
			var lists []domain.RecommendationList
			if json.Unmarshal(data, &lists) == nil {
				return lists, nil
			}
		}
	}

	// Load weights
	weights, err := s.weightRepo.GetWeights(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get weights: %w", err)
	}

	types := []string{"for_you", "browsing", "sellers", "new_arrivals"}
	if listType != "" {
		types = []string{listType}
	}

	titles := map[string]string{
		"for_you":      "おすすめ",
		"browsing":     "閲覧履歴から",
		"sellers":      "フォロー中のセラー",
		"new_arrivals": "新着アイテム",
	}

	var lists []domain.RecommendationList
	for _, lt := range types {
		items, err := s.fetchRecommendations(ctx, userID, weights, lt)
		if err != nil {
			log.Printf("[rec] fetch %s for user %d: %v", lt, userID, err)
			continue
		}
		if len(items) == 0 {
			continue
		}
		// Mark the first item as highlighted
		items[0].IsHighlighted = true

		lists = append(lists, domain.RecommendationList{
			Type:  lt,
			Title: titles[lt],
			Items: items,
		})
	}

	// Cache result
	if s.cache != nil && len(lists) > 0 {
		data, _ := json.Marshal(lists)
		_ = s.cache.Set(ctx, cacheKey, data, recCacheTTL)
	}

	return lists, nil
}

// fetchRecommendations queries ES for one recommendation list type.
func (s *RecommendationService) fetchRecommendations(ctx context.Context, userID int64, weights []domain.RecWeight, listType string) ([]domain.RecommendedProduct, error) {
	query := s.buildESQuery(weights, listType)

	res, err := s.esClient.Search(
		s.esClient.Search.WithContext(ctx),
		s.esClient.Search.WithIndex(s.esIndex),
		s.esClient.Search.WithBody(bytes.NewReader(query)),
	)
	if err != nil {
		return nil, fmt.Errorf("es search: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("es search: status %s", res.Status())
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("es read: %w", err)
	}

	var esResp struct {
		Hits struct {
			Hits []struct {
				Source json.RawMessage `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.Unmarshal(data, &esResp); err != nil {
		return nil, fmt.Errorf("es decode: %w", err)
	}

	items := make([]domain.RecommendedProduct, 0, len(esResp.Hits.Hits))
	for _, hit := range esResp.Hits.Hits {
		var doc struct {
			ID             string   `json:"id"`
			Title          string   `json:"title"`
			Images         []string `json:"images"`
			PriceJPY       int64    `json:"price_jpy"`
			SourcePlatform string   `json:"source_platform"`
			BrandName      string   `json:"brand_name"`
			Condition      string   `json:"condition"`
		}
		if json.Unmarshal(hit.Source, &doc) != nil {
			continue
		}
		image := ""
		if len(doc.Images) > 0 {
			image = doc.Images[0]
		}
		items = append(items, domain.RecommendedProduct{
			ID:        doc.ID,
			Title:     doc.Title,
			Image:     image,
			PriceJpy:  doc.PriceJPY,
			Platform:  doc.SourcePlatform,
			Brand:     doc.BrandName,
			Condition: doc.Condition,
		})
	}

	return items, nil
}

// buildESQuery constructs an ES query based on weights and list type.
func (s *RecommendationService) buildESQuery(weights []domain.RecWeight, listType string) []byte {
	// Group weights by dimension
	categoryWeights := map[string]float32{}
	brandWeights := map[string]float32{}
	sellerWeights := map[string]float32{}
	keywordWeights := map[string]float32{}

	for _, w := range weights {
		switch w.Dimension {
		case "category":
			categoryWeights[w.Value] = w.Weight
		case "brand":
			brandWeights[w.Value] = w.Weight
		case "seller":
			sellerWeights[w.Value] = w.Weight
		case "keyword":
			keywordWeights[w.Value] = w.Weight
		}
	}

	// Filter: always available products
	filters := []interface{}{
		map[string]interface{}{
			"term": map[string]interface{}{
				"status": "available",
			},
		},
	}

	var should []interface{}

	switch listType {
	case "browsing":
		// Focus on categories and brands from browsing
		should = buildShouldFromMaps(categoryWeights, brandWeights, nil, nil)

	case "sellers":
		// Focus on followed sellers
		if len(sellerWeights) > 0 {
			sellerIDs := keysFromMap(sellerWeights)
			filters = append(filters, map[string]interface{}{
				"terms": map[string]interface{}{
					"seller_id": sellerIDs,
				},
			})
		}

	case "new_arrivals":
		// Filter by preferred categories, sort by newest
		if len(categoryWeights) > 0 {
			cats := keysFromMap(categoryWeights)
			filters = append(filters, map[string]interface{}{
				"terms": map[string]interface{}{
					"categories": cats,
				},
			})
		}

	default: // "for_you"
		should = buildShouldFromMaps(categoryWeights, brandWeights, sellerWeights, keywordWeights)
	}

	boolQuery := map[string]interface{}{
		"filter": filters,
	}

	if len(should) > 0 {
		boolQuery["should"] = should
		boolQuery["minimum_should_match"] = 1
	}

	// Sort
	var sort []interface{}
	if listType == "new_arrivals" {
		sort = []interface{}{
			map[string]interface{}{
				"listed_at": map[string]interface{}{"order": "desc"},
			},
		}
	} else {
		sort = []interface{}{
			map[string]interface{}{
				"_score": map[string]interface{}{"order": "desc"},
			},
		}
	}

	body := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": boolQuery,
		},
		"sort": sort,
		"size": 6,
	}

	data, _ := json.Marshal(body)
	return data
}

// buildShouldFromMaps creates ES should clauses from weight maps.
func buildShouldFromMaps(cats, brands, sellers, keywords map[string]float32) []interface{} {
	var should []interface{}

	if len(cats) > 0 {
		catList := keysFromMap(cats)
		avgBoost := avgWeight(cats)
		should = append(should, map[string]interface{}{
			"terms": map[string]interface{}{
				"categories": catList,
				"boost":      math.Round(float64(avgBoost)*10) / 10,
			},
		})
	}

	if len(brands) > 0 {
		brandList := keysFromMap(brands)
		avgBoost := avgWeight(brands)
		should = append(should, map[string]interface{}{
			"terms": map[string]interface{}{
				"brand_name": brandList,
				"boost":      math.Round(float64(avgBoost)*10) / 10,
			},
		})
	}

	if len(sellers) > 0 {
		sellerList := keysFromMap(sellers)
		avgBoost := avgWeight(sellers)
		should = append(should, map[string]interface{}{
			"terms": map[string]interface{}{
				"seller_id": sellerList,
				"boost":     math.Round(float64(avgBoost)*10) / 10,
			},
		})
	}

	// Add keyword multi_match for top keywords
	if len(keywords) > 0 {
		// Take top 5 keywords by weight
		topKW := topNKeys(keywords, 5)
		for _, kw := range topKW {
			should = append(should, map[string]interface{}{
				"multi_match": map[string]interface{}{
					"query":  kw,
					"fields": []string{"title^2", "description"},
				},
			})
		}
	}

	return should
}

// RecomputeActiveUsers recomputes weights for all recently active users.
func (s *RecommendationService) RecomputeActiveUsers(ctx context.Context) {
	userIDs, err := s.weightRepo.GetActiveUserIDs(ctx, 7)
	if err != nil {
		log.Printf("[rec] get active users: %v", err)
		return
	}

	for _, uid := range userIDs {
		if err := s.ComputeWeights(ctx, uid); err != nil {
			log.Printf("[rec] compute weights for user %d: %v", uid, err)
		}
	}

	if len(userIDs) > 0 {
		log.Printf("[rec] recomputed weights for %d active users", len(userIDs))
	}
}

// InvalidateCache removes the recommendation cache for a user.
func (s *RecommendationService) InvalidateCache(ctx context.Context, userID int64) {
	if s.cache != nil {
		_ = s.cache.Del(ctx, recCachePrefix+fmt.Sprintf("%d", userID))
	}
}

// --- helpers ---

func parseProductID(id string) (platform, sourceID string) {
	idx := strings.Index(id, "_")
	if idx < 0 {
		return "", id
	}
	return id[:idx], id[idx+1:]
}

func keysFromMap(m map[string]float32) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func avgWeight(m map[string]float32) float32 {
	if len(m) == 0 {
		return 1.0
	}
	var sum float32
	for _, v := range m {
		sum += v
	}
	return sum / float32(len(m))
}

func topNKeys(m map[string]float32, n int) []string {
	type kv struct {
		key string
		val float32
	}
	sorted := make([]kv, 0, len(m))
	for k, v := range m {
		sorted = append(sorted, kv{k, v})
	}
	// Simple selection sort for small n
	for i := 0; i < len(sorted) && i < n; i++ {
		maxIdx := i
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j].val > sorted[maxIdx].val {
				maxIdx = j
			}
		}
		sorted[i], sorted[maxIdx] = sorted[maxIdx], sorted[i]
	}
	result := make([]string, 0, n)
	for i := 0; i < len(sorted) && i < n; i++ {
		result = append(result, sorted[i].key)
	}
	return result
}
