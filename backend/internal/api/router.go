package api

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
)

// RouterConfig holds the dependencies needed to build the API router.
type RouterConfig struct {
	SearchHandler         *SearchHandler
	RealtimeHandler       *RealtimeHandler
	ProductHandler        *ProductHandler
	HealthHandler         *HealthHandler
	PlatformSearchHandler *PlatformSearchHandler
	SurugayaHandler       *SurugayaHandler
}

// NewRouter creates a chi router with all API routes and middleware.
func NewRouter(cfg RouterConfig) *chi.Mux {
	r := chi.NewRouter()

	// Global middleware
	r.Use(Recovery)
	r.Use(RequestID)
	r.Use(Logger)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type", "Authorization"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// Health check (outside /api/v1 group)
	r.Get("/health", cfg.HealthHandler.HandleHealth)

	// API v1
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/search", cfg.SearchHandler.HandleSearch)
		r.Get("/search/stream/{streamID}", cfg.RealtimeHandler.HandleStream)
		r.Get("/products/{id}", cfg.ProductHandler.HandleGetProduct)
		r.Get("/platform/search", cfg.PlatformSearchHandler.HandleSearch)

		// Surugaya extension endpoints.
		r.Route("/surugaya", func(r chi.Router) {
			r.Get("/products/{id}/reviews", cfg.SurugayaHandler.HandleProductReviews)
			r.Get("/products/{id}/stores", cfg.SurugayaHandler.HandleProductStores)
			r.Get("/discounts", cfg.SurugayaHandler.HandleDiscounts)
			r.Get("/comments", cfg.SurugayaHandler.HandleUserComments)
			r.Get("/campaigns", cfg.SurugayaHandler.HandleCampaigns)
			r.Get("/campaigns/detail", cfg.SurugayaHandler.HandleCampaignDetail)
		})
	})

	return r
}
