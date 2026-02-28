package api

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rakutao/collection-gateway/internal/domain"
)

// ProductFetcher retrieves a product by its unified ID.
// This interface abstracts ES cache + adapter fallback for testability.
type ProductFetcher interface {
	GetProduct(ctx context.Context, id string) (*domain.UnifiedProduct, error)
}

// ProductHandler handles product detail requests.
type ProductHandler struct {
	fetcher ProductFetcher
}

// NewProductHandler creates a ProductHandler.
func NewProductHandler(fetcher ProductFetcher) *ProductHandler {
	return &ProductHandler{fetcher: fetcher}
}

// HandleGetProduct handles GET /api/v1/products/{id}.
func (h *ProductHandler) HandleGetProduct(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		ErrorWithCode(w, r, http.StatusBadRequest, 40002, "missing product ID")
		return
	}

	product, err := h.fetcher.GetProduct(r.Context(), id)
	if err != nil {
		ErrorWithCode(w, r, http.StatusNotFound, 40401, "product not found")
		return
	}

	lang := r.URL.Query().Get("lang")
	if lang == "" {
		lang = "zh-TW"
	}

	resp := buildProductResponse(product, lang)
	Success(w, r, resp)
}

// ProductResponse is the API response for a product detail.
type ProductResponse struct {
	ID                  string           `json:"id"`
	Platform            string           `json:"platform"`
	Title               string           `json:"title"`
	TitleOriginal       string           `json:"title_original"`
	Description         string           `json:"description"`
	DescriptionOriginal string           `json:"description_original"`
	Images              []string         `json:"images"`
	PriceJPY            int64            `json:"price_jpy"`
	ServiceFeeJPY       int64            `json:"service_fee_jpy"`
	OriginalPrice       int64            `json:"original_price"`
	ShippingType        string           `json:"shipping_type"`
	ShippingFeeJPY      int64            `json:"shipping_fee_jpy"`
	Brand               *domain.Brand    `json:"brand,omitempty"`
	Categories          []string         `json:"categories"`
	Condition           string           `json:"condition"`
	Status              string           `json:"status"`
	Quantity            int              `json:"quantity"`
	Seller              domain.SellerInfo `json:"seller"`
	Variants            []domain.Variant `json:"variants,omitempty"`
	ContentRating       string           `json:"content_rating"`
	ListedAt            string           `json:"listed_at"`
	IsTranslated        bool             `json:"is_translated"`
}

// buildProductResponse maps a UnifiedProduct to the API response, selecting
// the translated title/description for the given language.
func buildProductResponse(p *domain.UnifiedProduct, lang string) ProductResponse {
	title := p.Title
	description := p.Description
	isTranslated := false

	if t, ok := p.TitleTranslated[lang]; ok && t != "" {
		title = t
		isTranslated = true
	}
	if d, ok := p.DescTranslated[lang]; ok && d != "" {
		description = d
	}

	return ProductResponse{
		ID:                  p.ID,
		Platform:            p.SourcePlatform,
		Title:               title,
		TitleOriginal:       p.Title,
		Description:         description,
		DescriptionOriginal: p.Description,
		Images:              p.Images,
		PriceJPY:            p.PriceJPY,
		ServiceFeeJPY:       p.ServiceFeeJPY,
		OriginalPrice:       p.OriginalPrice,
		ShippingType:        p.ShippingType,
		ShippingFeeJPY:      p.ShippingFeeJPY,
		Brand:               p.Brand,
		Categories:          p.Categories,
		Condition:           p.Condition,
		Status:              p.Status,
		Quantity:            p.Quantity,
		Seller:              p.Seller,
		Variants:            p.Variants,
		ContentRating:       p.ContentRating,
		ListedAt:            p.ListedAt.Format("2006-01-02T15:04:05Z"),
		IsTranslated:        isTranslated,
	}
}
