package api

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/rakutao/collection-gateway/internal/adapter/surugaya"
	"github.com/rakutao/collection-gateway/internal/i18n"
	"github.com/rakutao/collection-gateway/internal/translate"
)

// SurugayaHandler handles Surugaya-specific extension endpoints.
type SurugayaHandler struct {
	client     *surugaya.Client
	translator translate.Translator
}

// NewSurugayaHandler creates a SurugayaHandler.
func NewSurugayaHandler(client *surugaya.Client, tr translate.Translator) *SurugayaHandler {
	return &SurugayaHandler{client: client, translator: tr}
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

	lang := i18n.FromContext(r.Context())
	if lang != i18n.LangJA && h.translator != nil {
		h.translateExtendData(r.Context(), data, string(lang))
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

	lang := i18n.FromContext(r.Context())
	if lang != i18n.LangJA && h.translator != nil {
		h.translateCampaignData(r.Context(), data, string(lang))
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

	lang := i18n.FromContext(r.Context())
	if lang != i18n.LangJA && h.translator != nil {
		h.translateCampaignDetailData(r.Context(), data, string(lang))
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

// --- translation helpers ---

// translateField is a pointer to a string field that needs translation.
type translateField struct {
	ptr *string
}

// batchTranslateFields concurrently translates all non-empty string fields.
func (h *SurugayaHandler) batchTranslateFields(ctx context.Context, fields []translateField, lang string) {
	if len(fields) == 0 {
		return
	}

	sem := make(chan struct{}, 10)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, f := range fields {
		if *f.ptr == "" {
			continue
		}
		wg.Add(1)
		go func(p *string, original string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			translated, err := h.translator.Translate(ctx, original, "ja", lang)
			if err != nil {
				log.Printf("[translate] surugaya field error →%s: %v", lang, err)
				return
			}
			mu.Lock()
			*p = translated
			mu.Unlock()
		}(f.ptr, *f.ptr)
	}

	wg.Wait()
}

// translateExtendData translates reviews and similar product names.
func (h *SurugayaHandler) translateExtendData(ctx context.Context, data *surugaya.ExtendData, lang string) {
	var fields []translateField
	for i := range data.Comments {
		fields = append(fields, translateField{&data.Comments[i].Title})
		fields = append(fields, translateField{&data.Comments[i].Comment})
	}
	for i := range data.OtherList {
		fields = append(fields, translateField{&data.OtherList[i].Name})
	}
	h.batchTranslateFields(ctx, fields, lang)
}

// translateCampaignData translates campaign titles and details.
func (h *SurugayaHandler) translateCampaignData(ctx context.Context, data *surugaya.CampaignData, lang string) {
	var fields []translateField
	for i := range data.Campaigns {
		fields = append(fields, translateField{&data.Campaigns[i].Title})
		fields = append(fields, translateField{&data.Campaigns[i].Detail})
	}
	h.batchTranslateFields(ctx, fields, lang)
}

// translateCampaignDetailData translates campaign detail entries and group items.
func (h *SurugayaHandler) translateCampaignDetailData(ctx context.Context, data *surugaya.CampaignDetailData, lang string) {
	var fields []translateField
	for i := range data.Campaigns {
		fields = append(fields, translateField{&data.Campaigns[i].Title})
		fields = append(fields, translateField{&data.Campaigns[i].Desc})
	}
	for i := range data.Groups {
		fields = append(fields, translateField{&data.Groups[i].Title})
		for j := range data.Groups[i].Items {
			fields = append(fields, translateField{&data.Groups[i].Items[j].Name})
		}
	}
	h.batchTranslateFields(ctx, fields, lang)
}
