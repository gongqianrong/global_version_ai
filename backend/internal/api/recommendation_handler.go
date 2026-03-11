package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rakutao/collection-gateway/internal/domain"
	"github.com/rakutao/collection-gateway/internal/repo"
)

// PreferenceStore abstracts preference repository for testability.
type PreferenceStore interface {
	SetPreferences(ctx context.Context, userID int64, categories []string) ([]domain.UserPreference, error)
	GetPreferences(ctx context.Context, userID int64) ([]domain.UserPreference, error)
}

// BrowsingStore abstracts browsing repository for testability.
type BrowsingStore interface {
	Record(ctx context.Context, rec *domain.BrowsingRecord) error
}

// SearchHistoryStore abstracts search history repository for testability.
type SearchHistoryStore interface {
	Record(ctx context.Context, rec *domain.SearchRecord) error
}

// RecService abstracts recommendation service for testability.
type RecService interface {
	GetRecommendations(ctx context.Context, userID int64, listType string, refresh bool) ([]domain.RecommendationList, error)
	InvalidateCache(ctx context.Context, userID int64)
}

// RecommendationHandler handles recommendation-related HTTP endpoints.
type RecommendationHandler struct {
	preferences PreferenceStore
	browsing    BrowsingStore
	searchHist  SearchHistoryStore
	recService  RecService
}

// NewRecommendationHandler creates a new RecommendationHandler.
func NewRecommendationHandler(
	prefs PreferenceStore,
	browsing BrowsingStore,
	searchHist SearchHistoryStore,
	recSvc RecService,
) *RecommendationHandler {
	return &RecommendationHandler{
		preferences: prefs,
		browsing:    browsing,
		searchHist:  searchHist,
		recService:  recSvc,
	}
}

// HandleSetPreferences handles POST /api/v1/preferences.
func (h *RecommendationHandler) HandleSetPreferences(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())

	var req struct {
		Categories []string `json:"categories"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ErrorWithCode(w, r, http.StatusBadRequest, 40002, "invalid request body")
		return
	}
	if len(req.Categories) == 0 {
		ErrorWithCode(w, r, http.StatusBadRequest, 40002, "categories cannot be empty")
		return
	}

	prefs, err := h.preferences.SetPreferences(r.Context(), userID, req.Categories)
	if err != nil {
		ErrorWithCode(w, r, http.StatusInternalServerError, 50001, "failed to set preferences")
		return
	}

	// Invalidate recommendation cache on preference change
	if h.recService != nil {
		h.recService.InvalidateCache(r.Context(), userID)
	}

	Success(w, r, map[string]interface{}{
		"preferences": prefs,
	})
}

// HandleGetPreferences handles GET /api/v1/preferences.
func (h *RecommendationHandler) HandleGetPreferences(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())

	prefs, err := h.preferences.GetPreferences(r.Context(), userID)
	if err != nil {
		ErrorWithCode(w, r, http.StatusInternalServerError, 50001, "failed to get preferences")
		return
	}
	if prefs == nil {
		prefs = []domain.UserPreference{}
	}

	Success(w, r, map[string]interface{}{
		"preferences": prefs,
	})
}

// HandleTrackView handles POST /api/v1/track/view.
func (h *RecommendationHandler) HandleTrackView(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())

	var req struct {
		ProductID string `json:"productId"`
		Category  string `json:"category"`
		Brand     string `json:"brand"`
		SellerID  string `json:"sellerId"`
		Platform  string `json:"platform"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ErrorWithCode(w, r, http.StatusBadRequest, 40002, "invalid request body")
		return
	}
	if req.ProductID == "" {
		ErrorWithCode(w, r, http.StatusBadRequest, 40002, "productId is required")
		return
	}

	// Fire-and-forget: don't block the response
	rec := &domain.BrowsingRecord{
		UserID:    userID,
		ProductID: req.ProductID,
		Category:  req.Category,
		Brand:     req.Brand,
		SellerID:  req.SellerID,
		Platform:  req.Platform,
	}
	go func() {
		ctx := context.Background()
		if err := h.browsing.Record(ctx, rec); err != nil {
			log.Printf("[track] record view for user %d product %s: %v", userID, req.ProductID, err)
		}
	}()

	Success(w, r, nil)
}

// HandleTrackSearch handles POST /api/v1/track/search.
func (h *RecommendationHandler) HandleTrackSearch(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())

	var req struct {
		Keyword   string `json:"keyword"`
		KeywordJA string `json:"keywordJa"`
		Platform  string `json:"platform"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ErrorWithCode(w, r, http.StatusBadRequest, 40002, "invalid request body")
		return
	}
	if req.Keyword == "" {
		ErrorWithCode(w, r, http.StatusBadRequest, 40002, "keyword is required")
		return
	}

	rec := &domain.SearchRecord{
		UserID:    userID,
		Keyword:   req.Keyword,
		KeywordJA: req.KeywordJA,
		Platform:  req.Platform,
	}
	go func() {
		ctx := context.Background()
		if err := h.searchHist.Record(ctx, rec); err != nil {
			log.Printf("[track] record search for user %d keyword %s: %v", userID, req.Keyword, err)
		}
	}()

	Success(w, r, nil)
}

// HandleGetRecommendations handles GET /api/v1/recommendations.
func (h *RecommendationHandler) HandleGetRecommendations(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	refresh := r.URL.Query().Get("refresh") == "true"

	lists, err := h.recService.GetRecommendations(r.Context(), userID, "", refresh)
	if err != nil {
		ErrorWithCode(w, r, http.StatusInternalServerError, 50001, "failed to get recommendations")
		return
	}
	if lists == nil {
		lists = []domain.RecommendationList{}
	}

	Success(w, r, map[string]interface{}{
		"lists": lists,
	})
}

// HandleGetRecommendationList handles GET /api/v1/recommendations/{listType}.
func (h *RecommendationHandler) HandleGetRecommendationList(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	listType := chi.URLParam(r, "listType")
	refresh := r.URL.Query().Get("refresh") == "true"

	validTypes := map[string]bool{
		"for_you": true, "browsing": true, "sellers": true, "new_arrivals": true,
	}
	if !validTypes[listType] {
		ErrorWithCode(w, r, http.StatusBadRequest, 40002, "invalid list type")
		return
	}

	lists, err := h.recService.GetRecommendations(r.Context(), userID, listType, refresh)
	if err != nil {
		ErrorWithCode(w, r, http.StatusInternalServerError, 50001, "failed to get recommendations")
		return
	}
	if lists == nil {
		lists = []domain.RecommendationList{}
	}

	Success(w, r, map[string]interface{}{
		"lists": lists,
	})
}

// RecordBrowseAsync records a product view asynchronously (called from ProductHandler).
func RecordBrowseAsync(browseRepo *repo.BrowsingRepo, userID int64, product *domain.UnifiedProduct) {
	if browseRepo == nil || userID == 0 || product == nil {
		return
	}

	category := ""
	if len(product.Categories) > 0 {
		category = product.Categories[0]
	}
	brandName := ""
	if product.Brand != nil {
		brandName = product.Brand.Name
	}

	rec := &domain.BrowsingRecord{
		UserID:    userID,
		ProductID: product.ID,
		Category:  category,
		Brand:     brandName,
		SellerID:  product.Seller.SellerID,
		Platform:  product.SourcePlatform,
	}

	go func() {
		ctx := context.Background()
		_ = browseRepo.Record(ctx, rec)
	}()
}
