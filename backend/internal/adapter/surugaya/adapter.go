package surugaya

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/rakutao/collection-gateway/internal/domain"
)

// Adapter implements domain.PlatformAdapter for Surugaya,
// proxying requests to the Surugaya domestic API backend.
type Adapter struct {
	client *Client
}

// New creates a Surugaya adapter that proxies to the given API URL.
func New(baseURL string, httpClient *http.Client) *Adapter {
	return &Adapter{
		client: NewClient(baseURL, httpClient),
	}
}

// PlatformID returns "surugaya".
func (a *Adapter) PlatformID() string {
	return "surugaya"
}

// Capabilities returns the adapter's declared capabilities.
func (a *Adapter) Capabilities() domain.AdapterCaps {
	return domain.AdapterCaps{
		SupportsSearch:       true,
		SupportsRealtime:     true,
		SupportsBatchCollect: false,
		HasBrandField:        true, // Surugaya search returns brand
		HasCategoryField:     true,
		MaxQPS:               10,
	}
}

// Search performs a keyword search via the Surugaya API and returns raw products.
func (a *Adapter) Search(ctx context.Context, query domain.SearchQuery) (*domain.SearchResult, error) {
	keyword := query.KeywordJA
	if keyword == "" {
		keyword = query.Keyword
	}

	category := ""
	if len(query.Categories) > 0 {
		category = query.Categories[0]
	}

	// Default to safe search on (hide adult content).
	safeSearch := "1"
	if query.ContentRating == domain.ContentRatingR18 || query.ContentRating == "all" {
		safeSearch = "0"
	}

	params := SearchParams{
		SearchWord:       keyword,
		Page:             query.Page,
		Category:         category,
		SafeSearchEnable: safeSearch,
		InStock:          "On",
		RankBy:           mapSortBy(query.SortBy),
		SaleClassified:   mapConditionToSaleClassified(query.Condition),
		PriceMin:         query.PriceMin,
		PriceMax:         query.PriceMax,
	}

	data, err := a.client.Search(ctx, params)
	if err != nil {
		return nil, err
	}

	products := make([]domain.RawProduct, 0, len(data.Item))
	for _, it := range data.Item {
		products = append(products, searchItemToRawProduct(it))
	}

	hasMore := int64(data.MaxNum) < data.Total

	return &domain.SearchResult{
		Products: products,
		Total:    data.Total,
		HasMore:  hasMore,
	}, nil
}

// GetProduct fetches a single product detail by goods ID.
func (a *Adapter) GetProduct(ctx context.Context, productID string) (*domain.RawProduct, error) {
	detail, err := a.client.GetProductDetail(ctx, productID)
	if err != nil {
		return nil, err
	}
	rp := detailToRawProduct(productID, detail)
	return &rp, nil
}

// BatchCollect is not implemented for Surugaya.
func (a *Adapter) BatchCollect(_ context.Context, _ domain.CollectParams) (<-chan domain.RawProduct, error) {
	return nil, fmt.Errorf("surugaya: BatchCollect not implemented")
}

// HealthCheck checks the API health.
func (a *Adapter) HealthCheck(ctx context.Context) domain.HealthStatus {
	if err := a.client.Health(ctx); err != nil {
		return domain.HealthStatus{
			Status:  "unhealthy",
			Message: err.Error(),
		}
	}
	return domain.HealthStatus{
		Status:  "healthy",
		Message: "surugaya API is reachable",
	}
}

// --- Exposed client accessors for API layer ---

// Client returns the underlying API client for direct endpoint access.
func (a *Adapter) Client() *Client {
	return a.client
}

// --- Data conversion ---

// searchItemToRawProduct converts a search result item to domain.RawProduct.
func searchItemToRawProduct(it searchItem) domain.RawProduct {
	rawData := map[string]interface{}{
		"title":      it.Title,
		"source_url": it.Link,
	}

	// Price: use first sale entry price.
	if len(it.Sale) > 0 {
		rawData["price"] = it.Sale[0].Price
		rawData["sale_type"] = it.Sale[0].Type
	}
	// All sale entries for display.
	if len(it.Sale) > 1 {
		sales := make([]map[string]interface{}, len(it.Sale))
		for i, s := range it.Sale {
			sales[i] = map[string]interface{}{
				"price": s.Price,
				"type":  s.Type,
			}
		}
		rawData["sales"] = sales
	}

	// Brand (nullable).
	if it.Brand != nil && strings.TrimSpace(*it.Brand) != "" {
		rawData["brand_name"] = strings.TrimSpace(*it.Brand)
	}

	if it.Category != "" {
		rawData["category"] = it.Category
	}
	if it.Condition != "" {
		rawData["condition"] = it.Condition
	}
	if it.Pic != "" {
		rawData["images"] = []string{it.Pic}
	}
	if it.ReleaseDate != "" {
		rawData["release_date"] = it.ReleaseDate
	}
	if it.StoreTag != "" {
		rawData["store_tag"] = it.StoreTag
	}

	// Status: "品切れ" = sold out, null = available.
	if it.State != nil && *it.State != "" {
		rawData["status"] = *it.State
	}

	// Surugaya is a store — seller is always Surugaya.
	rawData["seller_id"] = "surugaya"
	rawData["seller_name"] = "駿河屋"

	return domain.RawProduct{
		Platform: "surugaya",
		RawID:    it.ID,
		RawData:  rawData,
	}
}

// detailToRawProduct converts a product detail response to domain.RawProduct.
func detailToRawProduct(goodsID string, d *detailData) domain.RawProduct {
	rawData := map[string]interface{}{
		"title":     d.Title,
		"buy_state": d.BuyState,
	}

	// Price: use first available type's price.
	if len(d.Types) > 0 {
		rawData["price"] = d.Types[0].Price
		rawData["sale_type"] = d.Types[0].Kubun
		rawData["stock"] = d.Types[0].Stock
		rawData["status"] = d.Types[0].State
	} else {
		// No purchase types (sold out). Try list price from detail map.
		if listPrice, ok := d.Detail["定価"]; ok {
			rawData["price"] = listPrice
		}
		rawData["status"] = "品切れ"
	}

	// All purchase types for display.
	if len(d.Types) > 0 {
		types := make([]map[string]interface{}, len(d.Types))
		for i, t := range d.Types {
			types[i] = map[string]interface{}{
				"price":          t.Price,
				"state":          t.State,
				"state_code":     t.StateCode,
				"stock":          t.Stock,
				"limit_purchase": t.LimitPurchase,
				"kubun":          t.Kubun,
				"cart_id":        t.CartID,
				"tenpo_cd":       t.TenpoCD,
				"branch_number":  t.BranchNumber,
			}
		}
		rawData["types"] = types
	}

	if d.Desc != "" {
		rawData["description"] = d.Desc
	}
	if d.StateDetail != "" {
		rawData["state_detail"] = d.StateDetail
	}
	if len(d.ImgList) > 0 {
		rawData["images"] = d.ImgList
	}
	if len(d.Tags) > 0 {
		rawData["tags_raw"] = d.Tags
	}

	// Category tree.
	if d.Category.Classify != "" {
		rawData["category"] = d.Category.Classify
		// Build full category path.
		var categories []string
		cat := &d.Category
		for cat != nil {
			if cat.Classify != "" {
				categories = append(categories, cat.Classify)
			}
			cat = cat.Category
		}
		if len(categories) > 0 {
			rawData["category_path"] = categories
		}
	}

	// Detail key-value pairs.
	if len(d.Detail) > 0 {
		rawData["detail_kv"] = d.Detail
	}

	// Shop info.
	if d.ShopSimpleInfo.ShopName != "" {
		rawData["seller_id"] = "surugaya"
		rawData["seller_name"] = d.ShopSimpleInfo.ShopName
		if d.ShopSimpleInfo.Score != "" {
			rawData["seller_score"] = d.ShopSimpleInfo.Score
		}
		if d.ShopSimpleInfo.ShopPic != "" {
			rawData["seller_pic"] = d.ShopSimpleInfo.ShopPic
		}
	} else {
		rawData["seller_id"] = "surugaya"
		rawData["seller_name"] = "駿河屋"
	}

	// Other branches.
	if len(d.OtherBranch) > 0 {
		branches := make([]map[string]interface{}, len(d.OtherBranch))
		for i, b := range d.OtherBranch {
			branches[i] = map[string]interface{}{
				"id":    b.ID,
				"state": b.State,
				"price": b.Price,
			}
		}
		rawData["other_branches"] = branches
	}

	// Restock notification info.
	if d.Nyuka.Rem != "" {
		rawData["nyuka"] = map[string]interface{}{
			"rem":      d.Nyuka.Rem,
			"tenpo_cd": d.Nyuka.TenpoCD,
		}
	}

	// Source URL: build from goods ID.
	rawData["source_url"] = "https://www.suruga-ya.jp/product/detail/" + goodsID

	return domain.RawProduct{
		Platform: "surugaya",
		RawID:    goodsID,
		RawData:  rawData,
	}
}

// --- Mapping helpers ---

// mapSortBy converts domain sort strings to Surugaya API rankBy values.
func mapSortBy(sort string) string {
	switch sort {
	case "price_asc":
		return "price:ascending"
	case "price_desc":
		return "price:descending"
	case "newest":
		return "modificationTime:descending"
	case "release_date_desc":
		return "release_date(int):descending"
	case "release_date_asc":
		return "release_date(int):ascending"
	default:
		return "relavancy(int)"
	}
}

// mapConditionToSaleClassified converts domain condition filters to Surugaya's
// sale_classified parameter values.
func mapConditionToSaleClassified(conditions []string) []string {
	if len(conditions) == 0 {
		return nil
	}
	mapping := map[string]string{
		domain.ConditionNew: "新品",
	}
	var result []string
	seen := map[string]bool{}
	hasUsed := false
	for _, c := range conditions {
		if sc, ok := mapping[c]; ok {
			if !seen[sc] {
				result = append(result, sc)
				seen[sc] = true
			}
		} else {
			// Any non-new condition maps to "中古" (used).
			hasUsed = true
		}
	}
	if hasUsed && !seen["中古"] {
		result = append(result, "中古")
	}
	return result
}
