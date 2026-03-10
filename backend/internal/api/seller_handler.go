package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/rakutao/collection-gateway/internal/repo"
)

// SellerFollowStore abstracts seller-follow persistence for testing.
type SellerFollowStore interface {
	Follow(ctx context.Context, userID int64, sellerID, sellerName string) (*repo.FollowedSeller, error)
	Unfollow(ctx context.Context, userID int64, sellerID string) error
	ListByUser(ctx context.Context, userID int64, limit, offset int) ([]repo.FollowedSeller, int64, error)
	BatchIsFollowing(ctx context.Context, userID int64, sellerIDs []string) (map[string]bool, error)
}

// SellerHandler handles seller follow/unfollow endpoints.
type SellerHandler struct {
	repo SellerFollowStore
}

// NewSellerHandler creates a SellerHandler.
func NewSellerHandler(r SellerFollowStore) *SellerHandler {
	return &SellerHandler{repo: r}
}

type followRequest struct {
	SellerName string `json:"seller_name"`
}

// HandleFollow handles POST /api/v1/sellers/{sellerID}/follow.
func (h *SellerHandler) HandleFollow(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	sellerID := chi.URLParam(r, "sellerID")

	if sellerID == "" {
		ErrorWithCode(w, r, http.StatusBadRequest, 40002, "seller_id is required")
		return
	}

	var body followRequest
	if r.Body != nil {
		json.NewDecoder(r.Body).Decode(&body)
	}

	item, err := h.repo.Follow(r.Context(), userID, sellerID, body.SellerName)
	if err != nil {
		ErrorWithCode(w, r, http.StatusInternalServerError, 50001, "failed to follow seller")
		return
	}

	Success(w, r, item)
}

// HandleUnfollow handles DELETE /api/v1/sellers/{sellerID}/follow.
func (h *SellerHandler) HandleUnfollow(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	sellerID := chi.URLParam(r, "sellerID")

	err := h.repo.Unfollow(r.Context(), userID, sellerID)
	if err != nil {
		if err == pgx.ErrNoRows {
			ErrorWithCode(w, r, http.StatusNotFound, 40401, "follow record not found")
			return
		}
		ErrorWithCode(w, r, http.StatusInternalServerError, 50001, "failed to unfollow seller")
		return
	}

	Success(w, r, map[string]string{"seller_id": sellerID, "status": "unfollowed"})
}

// HandleListFollowed handles GET /api/v1/sellers/followed?page=1&page_size=20.
func (h *SellerHandler) HandleListFollowed(w http.ResponseWriter, r *http.Request) {
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

	items, total, err := h.repo.ListByUser(r.Context(), userID, pageSize, offset)
	if err != nil {
		ErrorWithCode(w, r, http.StatusInternalServerError, 50001, "failed to list followed sellers")
		return
	}

	if items == nil {
		items = []repo.FollowedSeller{}
	}

	Success(w, r, map[string]interface{}{
		"items": items,
		"total": total,
		"page":  page,
	})
}

// HandleCheckFollowed handles GET /api/v1/sellers/check?ids=id1,id2,id3.
func (h *SellerHandler) HandleCheckFollowed(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())

	idsStr := r.URL.Query().Get("ids")
	if idsStr == "" {
		ErrorWithCode(w, r, http.StatusBadRequest, 40002, "ids is required")
		return
	}

	ids := strings.Split(idsStr, ",")
	cleaned := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id != "" {
			cleaned = append(cleaned, id)
		}
	}
	if len(cleaned) == 0 {
		ErrorWithCode(w, r, http.StatusBadRequest, 40002, "ids is required")
		return
	}

	result, err := h.repo.BatchIsFollowing(r.Context(), userID, cleaned)
	if err != nil {
		ErrorWithCode(w, r, http.StatusInternalServerError, 50001, "failed to check followed sellers")
		return
	}

	fullResult := make(map[string]bool, len(cleaned))
	for _, id := range cleaned {
		fullResult[id] = result[id]
	}

	Success(w, r, fullResult)
}
