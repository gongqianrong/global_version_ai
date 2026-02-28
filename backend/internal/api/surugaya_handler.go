package api

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/rakutao/collection-gateway/internal/adapter/surugaya"
)

// SurugayaHandler handles Surugaya-specific extension endpoints.
type SurugayaHandler struct {
	client *surugaya.Client
}

// NewSurugayaHandler creates a SurugayaHandler.
func NewSurugayaHandler(client *surugaya.Client) *SurugayaHandler {
	return &SurugayaHandler{client: client}
}

// HandleProductReviews handles GET /api/v1/surugaya/products/{id}/reviews.
func (h *SurugayaHandler) HandleProductReviews(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		ErrorWithCode(w, r, http.StatusBadRequest, 40002, "missing product ID")
		return
	}

	data, err := h.client.GetProductExtend(r.Context(), id)
	if err != nil {
		ErrorWithCode(w, r, http.StatusServiceUnavailable, 50003, err.Error())
		return
	}

	Success(w, r, data)
}

// HandleProductStores handles GET /api/v1/surugaya/products/{id}/stores.
func (h *SurugayaHandler) HandleProductStores(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		ErrorWithCode(w, r, http.StatusBadRequest, 40002, "missing product ID")
		return
	}

	data, err := h.client.GetProductStores(r.Context(), id)
	if err != nil {
		ErrorWithCode(w, r, http.StatusServiceUnavailable, 50003, err.Error())
		return
	}

	Success(w, r, data)
}

// HandleDiscounts handles GET /api/v1/surugaya/discounts.
func (h *SurugayaHandler) HandleDiscounts(w http.ResponseWriter, r *http.Request) {
	data, err := h.client.GetDiscount(r.Context())
	if err != nil {
		ErrorWithCode(w, r, http.StatusServiceUnavailable, 50003, err.Error())
		return
	}

	Success(w, r, data)
}

// HandleUserComments handles GET /api/v1/surugaya/comments.
func (h *SurugayaHandler) HandleUserComments(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	userID := q.Get("user_id")
	if userID == "" {
		ErrorWithCode(w, r, http.StatusBadRequest, 40002, "missing required parameter: user_id")
		return
	}

	pageNum, _ := strconv.Atoi(q.Get("page"))
	if pageNum <= 0 {
		pageNum = 1
	}
	pageSize, _ := strconv.Atoi(q.Get("page_size"))
	if pageSize <= 0 {
		pageSize = 20
	}
	sortType := q.Get("sort")
	if sortType == "" {
		sortType = "-created_at"
	}

	data, err := h.client.GetUserComments(r.Context(), surugaya.UserCommentsParams{
		UserID:   userID,
		PageNum:  pageNum,
		PageSize: pageSize,
		SortType: sortType,
	})
	if err != nil {
		ErrorWithCode(w, r, http.StatusServiceUnavailable, 50003, err.Error())
		return
	}

	Success(w, r, data)
}

// HandleCampaigns handles GET /api/v1/surugaya/campaigns.
func (h *SurugayaHandler) HandleCampaigns(w http.ResponseWriter, r *http.Request) {
	data, err := h.client.GetCampaigns(r.Context())
	if err != nil {
		ErrorWithCode(w, r, http.StatusServiceUnavailable, 50003, err.Error())
		return
	}

	Success(w, r, data)
}

// HandleCampaignDetail handles GET /api/v1/surugaya/campaigns/detail.
func (h *SurugayaHandler) HandleCampaignDetail(w http.ResponseWriter, r *http.Request) {
	detailURL := r.URL.Query().Get("url")
	if detailURL == "" {
		ErrorWithCode(w, r, http.StatusBadRequest, 40002, "missing required parameter: url")
		return
	}

	data, err := h.client.GetCampaignDetail(r.Context(), detailURL)
	if err != nil {
		ErrorWithCode(w, r, http.StatusServiceUnavailable, 50003, err.Error())
		return
	}

	Success(w, r, data)
}

// HandleCategories handles GET /api/v1/surugaya/categories.
func (h *SurugayaHandler) HandleCategories(w http.ResponseWriter, r *http.Request) {
	data, err := h.client.GetCategories(r.Context())
	if err != nil {
		ErrorWithCode(w, r, http.StatusServiceUnavailable, 50003, err.Error())
		return
	}

	Success(w, r, data)
}

// HandleSubCategories handles GET /api/v1/surugaya/categories/{id}.
func (h *SurugayaHandler) HandleSubCategories(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		ErrorWithCode(w, r, http.StatusBadRequest, 40002, "missing category ID")
		return
	}

	data, err := h.client.GetSubCategories(r.Context(), id)
	if err != nil {
		ErrorWithCode(w, r, http.StatusServiceUnavailable, 50003, err.Error())
		return
	}

	Success(w, r, data)
}
