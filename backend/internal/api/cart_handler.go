package api

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/rakutao/collection-gateway/internal/i18n"
	"github.com/rakutao/collection-gateway/internal/repo"
	"github.com/rakutao/collection-gateway/internal/translate"
)

// CartHandler handles shopping cart endpoints.
type CartHandler struct {
	carts      *repo.CartRepo
	esFetcher  ProductFetcher
	translator translate.Translator
}

// NewCartHandler creates a CartHandler.
func NewCartHandler(carts *repo.CartRepo, esFetcher ProductFetcher, tr translate.Translator) *CartHandler {
	return &CartHandler{carts: carts, esFetcher: esFetcher, translator: tr}
}

type addToCartRequest struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

type updateCartRequest struct {
	Quantity int `json:"quantity"`
}

// cartProductResponse is a cart item enriched with product data.
type cartProductResponse struct {
	ProductID       string `json:"product_id"`
	Quantity        int    `json:"quantity"`
	PriceAtAdd      int64  `json:"price_at_add"`
	CurrentPrice    int64  `json:"current_price"`
	PriceChanged    bool   `json:"price_changed"`
	Title           string `json:"title"`
	TitleOriginal   string `json:"title_original"`
	Image           string `json:"image"`
	Platform        string `json:"platform"`
	Status          string `json:"status"`
	SellerID        string `json:"seller_id"`
	SellerName      string `json:"seller_name"`
	IsTranslated    bool   `json:"is_translated"`
}

type cartSellerGroup struct {
	SellerID   string                `json:"seller_id"`
	SellerName string                `json:"seller_name"`
	Items      []cartProductResponse `json:"items"`
}

// HandleListCart handles GET /api/v1/cart.
func (h *CartHandler) HandleListCart(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	items, err := h.carts.ListByUser(r.Context(), userID)
	if err != nil {
		ErrorWithCode(w, r, http.StatusInternalServerError, 50001, "failed to list cart")
		return
	}

	if len(items) == 0 {
		Success(w, r, map[string]interface{}{"groups": []interface{}{}, "total_items": 0})
		return
	}

	lang := i18n.FromContext(r.Context())
	langStr := string(lang)

	// Fetch products from ES concurrently (bounded to 10).
	type enrichedItem struct {
		item    repo.CartItem
		resp    cartProductResponse
	}
	results := make([]enrichedItem, len(items))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 10)

	for idx, ci := range items {
		results[idx] = enrichedItem{item: ci}
		wg.Add(1)
		go func(i int, cartItem repo.CartItem) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			resp := cartProductResponse{
				ProductID:  cartItem.ProductID,
				Quantity:   cartItem.Quantity,
				PriceAtAdd: cartItem.PriceAtAdd,
			}

			product, err := h.esFetcher.GetProduct(r.Context(), cartItem.ProductID)
			if err != nil {
				log.Printf("[cart] product %s not found in ES: %v", cartItem.ProductID, err)
				resp.Title = cartItem.ProductID
				resp.TitleOriginal = cartItem.ProductID
				resp.Status = "unknown"
				results[i].resp = resp
				return
			}

			resp.CurrentPrice = product.PriceJPY
			resp.PriceChanged = product.PriceJPY != cartItem.PriceAtAdd
			resp.Platform = product.SourcePlatform
			resp.Status = product.Status
			resp.SellerID = product.Seller.SellerID
			resp.SellerName = product.Seller.SellerName

			if len(product.Images) > 0 {
				resp.Image = product.Images[0]
			}

			// Title with translation.
			resp.TitleOriginal = product.Title
			resp.Title = product.Title
			if t, ok := product.TitleTranslated[langStr]; ok && t != "" {
				resp.Title = t
				resp.IsTranslated = true
			} else if lang != i18n.LangJA && h.translator != nil && product.Title != "" {
				if translated, err := h.translator.Translate(r.Context(), product.Title, "ja", langStr); err == nil {
					resp.Title = translated
					resp.IsTranslated = true
				}
			}

			results[i].resp = resp
		}(idx, ci)
	}
	wg.Wait()

	// Group by seller.
	sellerMap := map[string]*cartSellerGroup{}
	var sellerOrder []string
	totalItems := 0
	for _, ei := range results {
		totalItems += ei.resp.Quantity
		sid := ei.resp.SellerID
		if sid == "" {
			sid = "_unknown"
		}
		if _, exists := sellerMap[sid]; !exists {
			sellerMap[sid] = &cartSellerGroup{
				SellerID:   ei.resp.SellerID,
				SellerName: ei.resp.SellerName,
			}
			sellerOrder = append(sellerOrder, sid)
		}
		sellerMap[sid].Items = append(sellerMap[sid].Items, ei.resp)
	}

	groups := make([]cartSellerGroup, 0, len(sellerOrder))
	for _, sid := range sellerOrder {
		groups = append(groups, *sellerMap[sid])
	}

	Success(w, r, map[string]interface{}{
		"groups":      groups,
		"total_items": totalItems,
	})
}

// HandleAddToCart handles POST /api/v1/cart.
func (h *CartHandler) HandleAddToCart(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())

	var req addToCartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ErrorWithCode(w, r, http.StatusBadRequest, 40002, "invalid request body")
		return
	}
	if req.ProductID == "" {
		ErrorWithCode(w, r, http.StatusBadRequest, 40002, "product_id is required")
		return
	}
	if req.Quantity <= 0 {
		req.Quantity = 1
	}

	// Verify product exists and get current price.
	product, err := h.esFetcher.GetProduct(r.Context(), req.ProductID)
	if err != nil {
		ErrorWithCode(w, r, http.StatusNotFound, 40401, "product not found")
		return
	}
	if !product.IsAvailable() {
		ErrorWithCode(w, r, http.StatusBadRequest, 40003, "product is not available")
		return
	}

	item, err := h.carts.Add(r.Context(), userID, req.ProductID, req.Quantity, product.PriceJPY)
	if err != nil {
		ErrorWithCode(w, r, http.StatusInternalServerError, 50001, "failed to add to cart")
		return
	}

	Success(w, r, item)
}

// HandleUpdateQuantity handles PUT /api/v1/cart/{productID}.
func (h *CartHandler) HandleUpdateQuantity(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	productID := chi.URLParam(r, "productID")

	var req updateCartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ErrorWithCode(w, r, http.StatusBadRequest, 40002, "invalid request body")
		return
	}
	if req.Quantity <= 0 {
		ErrorWithCode(w, r, http.StatusBadRequest, 40003, "quantity must be > 0")
		return
	}

	err := h.carts.UpdateQuantity(r.Context(), userID, productID, req.Quantity)
	if err != nil {
		if err == pgx.ErrNoRows {
			ErrorWithCode(w, r, http.StatusNotFound, 40401, "cart item not found")
			return
		}
		ErrorWithCode(w, r, http.StatusInternalServerError, 50001, "failed to update cart")
		return
	}

	Success(w, r, map[string]interface{}{"product_id": productID, "quantity": req.Quantity})
}

// HandleRemoveFromCart handles DELETE /api/v1/cart/{productID}.
func (h *CartHandler) HandleRemoveFromCart(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	productID := chi.URLParam(r, "productID")

	err := h.carts.Remove(r.Context(), userID, productID)
	if err != nil {
		if err == pgx.ErrNoRows {
			ErrorWithCode(w, r, http.StatusNotFound, 40401, "cart item not found")
			return
		}
		ErrorWithCode(w, r, http.StatusInternalServerError, 50001, "failed to remove from cart")
		return
	}

	Success(w, r, map[string]string{"product_id": productID, "status": "removed"})
}
