package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/rakutao/collection-gateway/internal/auth"
)

// RouterConfig holds the dependencies needed to build the API router.
type RouterConfig struct {
	SearchHandler         *SearchHandler
	RealtimeHandler       *RealtimeHandler
	ProductHandler        *ProductHandler
	HealthHandler         *HealthHandler
	PlatformSearchHandler *PlatformSearchHandler
	SurugayaHandler       *SurugayaHandler
	AuthHandler           *AuthHandler
	OAuthHandler          *OAuthHandler
	CartHandler           *CartHandler
	FavoriteHandler       *FavoriteHandler
	SellerHandler         *SellerHandler
	WalletHandler         *WalletHandler
	OrderHandler          *OrderHandler
	AdminChecker          UserAdminChecker
	JWTManager            *auth.JWTManager
	CacheMiddleware       func(http.Handler) http.Handler
}

// NewRouter creates a chi router with all API routes and middleware.
func NewRouter(cfg RouterConfig) *chi.Mux {
	r := chi.NewRouter()

	r.Use(Recovery)
	r.Use(RequestID)
	r.Use(Language)
	r.Use(Logger)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type", "Authorization", "Accept-Language"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	r.Get("/health", cfg.HealthHandler.HandleHealth)

	// Swagger UI
	r.Get("/swagger", http.RedirectHandler("/swagger/", http.StatusMovedPermanently).ServeHTTP)
	r.Get("/swagger/", HandleSwaggerUI)
	r.Get("/swagger/openapi.yaml", HandleOpenAPISpec)

	r.Route("/api/v1", func(r chi.Router) {
		// Public endpoints with cache.
		r.Group(func(r chi.Router) {
			if cfg.CacheMiddleware != nil {
				r.Use(cfg.CacheMiddleware)
			}
			r.Get("/search", cfg.SearchHandler.HandleSearch)
			r.Get("/search/stream/{streamID}", cfg.RealtimeHandler.HandleStream)
			r.Get("/products/{id}", cfg.ProductHandler.HandleGetProduct)
			r.Get("/platform/search", cfg.PlatformSearchHandler.HandleSearch)

			r.Route("/surugaya", func(r chi.Router) {
				r.Get("/products/{id}/reviews", cfg.SurugayaHandler.HandleProductReviews)
				r.Get("/products/{id}/stores", cfg.SurugayaHandler.HandleProductStores)
				r.Get("/discounts", cfg.SurugayaHandler.HandleDiscounts)
				r.Get("/comments", cfg.SurugayaHandler.HandleUserComments)
				r.Get("/campaigns", cfg.SurugayaHandler.HandleCampaigns)
				r.Get("/campaigns/detail", cfg.SurugayaHandler.HandleCampaignDetail)
				r.Get("/categories", cfg.SurugayaHandler.HandleCategories)
				r.Get("/categories/{id}", cfg.SurugayaHandler.HandleSubCategories)
			})
		})

		// Auth endpoints (public, no cache).
		if cfg.AuthHandler != nil || cfg.OAuthHandler != nil {
			r.Route("/auth", func(r chi.Router) {
				if cfg.AuthHandler != nil {
					r.Post("/send-code", cfg.AuthHandler.HandleSendCode)
					r.Post("/register", cfg.AuthHandler.HandleRegister)
					r.Post("/login", cfg.AuthHandler.HandleLogin)
				}
				if cfg.OAuthHandler != nil {
					r.Post("/google", cfg.OAuthHandler.HandleGoogle)
					r.Post("/apple", cfg.OAuthHandler.HandleApple)
				}
			})
		}

		// Protected endpoints (require JWT).
		if cfg.JWTManager != nil {
			r.Group(func(r chi.Router) {
				r.Use(AuthRequired(cfg.JWTManager))

				if cfg.CartHandler != nil {
					r.Get("/cart", cfg.CartHandler.HandleListCart)
					r.Post("/cart", cfg.CartHandler.HandleAddToCart)
					r.Put("/cart/{productID}", cfg.CartHandler.HandleUpdateQuantity)
					r.Delete("/cart/{productID}", cfg.CartHandler.HandleRemoveFromCart)
				}

				if cfg.FavoriteHandler != nil {
					r.Get("/favorites", cfg.FavoriteHandler.HandleListFavorites)
					r.Post("/favorites/{productID}", cfg.FavoriteHandler.HandleAddFavorite)
					r.Delete("/favorites/{productID}", cfg.FavoriteHandler.HandleRemoveFavorite)
					r.Get("/favorites/check", cfg.FavoriteHandler.HandleCheckFavorites)
				}

				if cfg.SellerHandler != nil {
					r.Post("/sellers/{sellerID}/follow", cfg.SellerHandler.HandleFollow)
					r.Delete("/sellers/{sellerID}/follow", cfg.SellerHandler.HandleUnfollow)
					r.Get("/sellers/followed", cfg.SellerHandler.HandleListFollowed)
					r.Get("/sellers/check", cfg.SellerHandler.HandleCheckFollowed)
				}

				if cfg.WalletHandler != nil {
					r.Get("/wallet/balance", cfg.WalletHandler.HandleGetBalance)
					r.Get("/wallet/transactions", cfg.WalletHandler.HandleListTransactions)
				}

				if cfg.OrderHandler != nil {
					r.Post("/order/settlement", cfg.OrderHandler.HandleSettlement)
					r.Post("/order/confirm", cfg.OrderHandler.HandleConfirm)
					r.Post("/order/pay", cfg.OrderHandler.HandlePay)
					r.Post("/order/cancel", cfg.OrderHandler.HandleCancelOrder)
					r.Get("/order/{orderNumber}", cfg.OrderHandler.HandleGetOrder)
					r.Get("/orders", cfg.OrderHandler.HandleListOrders)
				}
			})
		}

		// Admin endpoints (require JWT + admin).
		if cfg.JWTManager != nil && cfg.WalletHandler != nil && cfg.AdminChecker != nil {
			r.Route("/admin", func(r chi.Router) {
				r.Use(AuthRequired(cfg.JWTManager))
				r.Use(AdminRequired(cfg.AdminChecker))
				r.Post("/wallet/adjust", cfg.WalletHandler.HandleAdminAdjust)
			})
		}
	})

	return r
}
