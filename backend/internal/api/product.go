package api

import (
	"context"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rakutao/collection-gateway/internal/domain"
	"github.com/rakutao/collection-gateway/internal/i18n"
	"github.com/rakutao/collection-gateway/internal/translate"
)

// ProductFetcher retrieves a product by its unified ID.
// This interface abstracts ES cache + adapter fallback for testability.
type ProductFetcher interface {
	GetProduct(ctx context.Context, id string) (*domain.UnifiedProduct, error)
}

// ProductHandler handles product detail requests.
// It tries the primary fetcher (ES) first, then falls back to the platform adapter.
// If the product has no translation for the requested language, it translates on-the-fly
// and dispatches the updated product to ES asynchronously.
type ProductHandler struct {
	fetcher       ProductFetcher
	fallback      ProductFetcher
	translator    translate.Translator
	productWriter ProductWriter
}

// NewProductHandler creates a ProductHandler with an optional fallback fetcher.
func NewProductHandler(fetcher ProductFetcher, fallback ProductFetcher, tr translate.Translator, pw ProductWriter) *ProductHandler {
	return &ProductHandler{fetcher: fetcher, fallback: fallback, translator: tr, productWriter: pw}
}

// HandleGetProduct handles GET /api/v1/products/{id}.
func (h *ProductHandler) HandleGetProduct(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		ErrorWithCode(w, r, http.StatusBadRequest, 40002, "missing product ID")
		return
	}

	product, err := h.fetcher.GetProduct(r.Context(), id)
	if err != nil && h.fallback != nil {
		product, err = h.fallback.GetProduct(r.Context(), id)
	}
	if err != nil {
		ErrorWithCode(w, r, http.StatusNotFound, 40401, "product not found")
		return
	}

	lang := i18n.FromContext(r.Context())

	// Translate on-the-fly if missing for the requested language.
	if lang != i18n.LangJA && h.translator != nil {
		changed := h.translateProductIfNeeded(r.Context(), product, string(lang))
		if changed && h.productWriter != nil {
			go h.productWriter.Dispatch(context.Background(), []domain.UnifiedProduct{*product})
		}
	}

	resp := buildProductResponse(product, lang)
	Success(w, r, resp)
}

// translateProductIfNeeded translates title and description on-the-fly
// if the product doesn't have a translation for the given language.
// Returns true if any new translation was added.
func (h *ProductHandler) translateProductIfNeeded(ctx context.Context, p *domain.UnifiedProduct, lang string) bool {
	changed := false

	if p.TitleTranslated == nil {
		p.TitleTranslated = make(map[string]string)
	}
	if p.DescTranslated == nil {
		p.DescTranslated = make(map[string]string)
	}

	if _, ok := p.TitleTranslated[lang]; !ok && p.Title != "" {
		translated, err := h.translator.Translate(ctx, p.Title, "ja", lang)
		if err != nil {
			log.Printf("[translate] on-the-fly title error for %s→%s: %v", p.ID, lang, err)
		} else {
			p.TitleTranslated[lang] = translated
			changed = true
		}
	}

	if _, ok := p.DescTranslated[lang]; !ok && p.Description != "" {
		translated, err := h.translator.Translate(ctx, p.Description, "ja", lang)
		if err != nil {
			log.Printf("[translate] on-the-fly desc error for %s→%s: %v", p.ID, lang, err)
		} else {
			p.DescTranslated[lang] = translated
			changed = true
		}
	}

	return changed
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
// the translated title/description for the given language and localizing
// status, condition, shipping, and content rating labels.
func buildProductResponse(p *domain.UnifiedProduct, lang i18n.Lang) ProductResponse {
	title := p.Title
	titleOriginal := p.Title
	description := p.Description
	descOriginal := p.Description
	isTranslated := false

	langStr := string(lang)

	// All 4 text fields use translated version when available, skip if empty.
	if t, ok := p.TitleTranslated[langStr]; ok && t != "" {
		title = t
		isTranslated = true
	}
	if t, ok := p.TitleTranslated[langStr]; ok && t != "" {
		titleOriginal = t
	}
	if d, ok := p.DescTranslated[langStr]; ok && d != "" {
		description = d
	}
	if d, ok := p.DescTranslated[langStr]; ok && d != "" {
		descOriginal = d
	}

	return ProductResponse{
		ID:                  p.ID,
		Platform:            p.SourcePlatform,
		Title:               title,
		TitleOriginal:       titleOriginal,
		Description:         description,
		DescriptionOriginal: descOriginal,
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
