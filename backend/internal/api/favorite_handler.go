package api

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/rakutao/collection-gateway/internal/repo"
	"github.com/rakutao/collection-gateway/internal/translate"
)

// FavoriteStore abstracts favorite persistence for testing.
type FavoriteStore interface {
	Add(ctx context.Context, userID int64, productID string) (*repo.FavoriteItem, error)
	Remove(ctx context.Context, userID int64, productID string) error
	ListByUser(ctx context.Context, userID int64, limit, offset int) ([]repo.FavoriteItem, int64, error)
	BatchIsFavorited(ctx context.Context, userID int64, productIDs []string) (map[string]bool, error)
}

// FavoriteHandler handles favorites endpoints.
type FavoriteHandler struct {
	favorites  FavoriteStore
	esFetcher  ProductFetcher
	translator translate.Translator
}

// NewFavoriteHandler creates a FavoriteHandler.
func NewFavoriteHandler(favorites FavoriteStore, esFetcher ProductFetcher, tr translate.Translator) *FavoriteHandler {
	return &FavoriteHandler{favorites: favorites, esFetcher: esFetcher, translator: tr}
}

type favoriteProductResponse struct {
	ProductID     string `json:"product_id"`
	Title         string `json:"title"`
	TitleOriginal string `json:"title_original"`
	Image         string `json:"image"`
	PriceJPY      int64  `json:"price_jpy"`
	Platform      string `json:"platform"`
	Status        string `json:"status"`
	IsTranslated  bool   `json:"is_translated"`
	AddedAt       string `json:"added_at"`
}

// HandleListFavorites handles GET /api/v1/favorites.
func (h *FavoriteHandler) HandleListFavorites(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page <= 0 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	items, total, err := h.favorites.ListByUser(r.Context(), userID, pageSize, offset)
	if err != nil {
		ErrorWithCode(w, r, http.StatusInternalServerError, 50001, "failed to list favorites")
		return
	}

	if len(items) == 0 {
		Success(w, r, map[string]interface{}{
			"items": []interface{}{},
			"total": total,
			"page":  page,
		})
		return
	}

	// Enrich with product data concurrently.
	results := make([]favoriteProductResponse, len(items))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 10)

	for idx, fi := range items {
		wg.Add(1)
		go func(i int, fav repo.FavoriteItem) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			resp := favoriteProductResponse{
				ProductID: fav.ProductID,
				AddedAt:   fav.AddedAt.Format("2006-01-02T15:04:05Z"),
			}

			product, err := h.esFetcher.GetProduct(r.Context(), fav.ProductID)
			if err != nil {
				log.Printf("[favorites] product %s not found in ES: %v", fav.ProductID, err)
				resp.Title = fav.ProductID
				resp.TitleOriginal = fav.ProductID
				resp.Status = "unknown"
				results[i] = resp
				return
			}

			resp.PriceJPY = product.PriceJPY
			resp.Platform = product.SourcePlatform
			resp.Status = product.Status
			if len(product.Images) > 0 {
				resp.Image = product.Images[0]
			}

			resp.TitleOriginal = product.Title
			resp.Title = product.Title
			// TODO: translation disabled — always return Japanese original
			// if t, ok := product.TitleTranslated[langStr]; ok && t != "" {
			// 	resp.Title = t
			// 	resp.IsTranslated = true
			// }

			results[i] = resp
		}(idx, fi)
	}
	wg.Wait()

	Success(w, r, map[string]interface{}{
		"items": results,
		"total": total,
		"page":  page,
	})
}

// HandleAddFavorite handles POST /api/v1/favorites/{productID}.
func (h *FavoriteHandler) HandleAddFavorite(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	productID := chi.URLParam(r, "productID")

	if productID == "" {
		ErrorWithCode(w, r, http.StatusBadRequest, 40002, "product_id is required")
		return
	}

	// Verify product exists.
	if _, err := h.esFetcher.GetProduct(r.Context(), productID); err != nil {
		ErrorWithCode(w, r, http.StatusNotFound, 40401, "product not found")
		return
	}

	fav, err := h.favorites.Add(r.Context(), userID, productID)
	if err != nil {
		ErrorWithCode(w, r, http.StatusInternalServerError, 50001, "failed to add favorite")
		return
	}

	Success(w, r, fav)
}

// HandleRemoveFavorite handles DELETE /api/v1/favorites/{productID}.
func (h *FavoriteHandler) HandleRemoveFavorite(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	productID := chi.URLParam(r, "productID")

	err := h.favorites.Remove(r.Context(), userID, productID)
	if err != nil {
		if err == pgx.ErrNoRows {
			ErrorWithCode(w, r, http.StatusNotFound, 40401, "favorite not found")
			return
		}
		ErrorWithCode(w, r, http.StatusInternalServerError, 50001, "failed to remove favorite")
		return
	}

	Success(w, r, map[string]string{"product_id": productID, "status": "removed"})
}

// HandleCheckFavorites handles GET /api/v1/favorites/check?product_ids=id1,id2.
func (h *FavoriteHandler) HandleCheckFavorites(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())

	idsStr := r.URL.Query().Get("product_ids")
	if idsStr == "" {
		ErrorWithCode(w, r, http.StatusBadRequest, 40002, "product_ids is required")
		return
	}

	ids := strings.Split(idsStr, ",")
	// Trim whitespace.
	cleaned := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id != "" {
			cleaned = append(cleaned, id)
		}
	}
	if len(cleaned) == 0 {
		ErrorWithCode(w, r, http.StatusBadRequest, 40002, "product_ids is required")
		return
	}

	result, err := h.favorites.BatchIsFavorited(r.Context(), userID, cleaned)
	if err != nil {
		ErrorWithCode(w, r, http.StatusInternalServerError, 50001, "failed to check favorites")
		return
	}

	// Return full map with false for non-favorited items.
	fullResult := make(map[string]bool, len(cleaned))
	for _, id := range cleaned {
		fullResult[id] = result[id]
	}

	Success(w, r, fullResult)
}
