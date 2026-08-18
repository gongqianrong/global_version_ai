// backend/cmd/gateway/main.go
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/rakutao/collection-gateway/internal/adapter/surugaya"
	yahoo "github.com/rakutao/collection-gateway/internal/adapter/yahoo_auction"
	"github.com/rakutao/collection-gateway/internal/aiclient"
	"github.com/rakutao/collection-gateway/internal/api"
	"github.com/rakutao/collection-gateway/internal/auth"
	"github.com/rakutao/collection-gateway/internal/brand"
	"github.com/rakutao/collection-gateway/internal/cache"
	"github.com/rakutao/collection-gateway/internal/db"
	"github.com/rakutao/collection-gateway/internal/email"
	"github.com/rakutao/collection-gateway/internal/domain"
	"github.com/rakutao/collection-gateway/internal/filter"
	"github.com/rakutao/collection-gateway/internal/normalizer"
	"github.com/rakutao/collection-gateway/internal/output"
	"github.com/rakutao/collection-gateway/internal/registry"
	"github.com/rakutao/collection-gateway/internal/repo"
	"github.com/rakutao/collection-gateway/internal/search"
	"github.com/rakutao/collection-gateway/internal/service"
	"github.com/rakutao/collection-gateway/internal/translate"
)

func main() {
	// --- Build dependencies ---

	// AI service client (translation + brand extraction).
	aiServiceURL := os.Getenv("AI_SERVICE_URL")
	if aiServiceURL == "" {
		aiServiceURL = "http://localhost:8000"
	}
	aiClient := aiclient.New(aiServiceURL, nil)

	// Keyword filter (political keywords blacklist).
	keywordFilter := filter.NewKeywordFilter(
		[]string{"天安门", "六四", "法轮功"},
		[]string{"政治", "独立运动"},
	)

	// Search gateway with AI-powered translation.
	gateway := search.NewGateway(aiClient, keywordFilter)

	// Brand pipeline (rule-based + AI extraction).
	brandRegistry := brand.DefaultRegistry()
	brandPipeline := brand.NewPipeline(brandRegistry, aiClient)
	brandAdapter := brand.NewPipelineAdapter(brandPipeline)

	// Normalizer with brand extraction.
	norm := normalizer.New(brandAdapter)

	// Platform registry.
	reg := registry.New()

	// Register Yahoo Auction adapter.
	yahooURL := os.Getenv("YAHOO_AUCTION_API_URL")
	if yahooURL == "" {
		yahooURL = "http://localhost:3001"
	}
	reg.Register(registry.PlatformMeta{
		ID:     "yahoo_auction",
		Name:   "ヤフオク",
		NameEN: "Yahoo Auctions",
		Type:   registry.TypeDomesticProxy,
		Status: registry.StatusActive,
		Caps: domain.AdapterCaps{
			SupportsSearch:   true,
			SupportsRealtime: true,
			HasCategoryField: true,
			MaxQPS:           10,
		},
	}, yahoo.New(yahooURL, nil))

	// Register Surugaya adapter.
	surugayaURL := os.Getenv("SURUGAYA_API_URL")
	if surugayaURL == "" {
		surugayaURL = "http://153.231.197.185"
	}
	reg.Register(registry.PlatformMeta{
		ID:     "surugaya",
		Name:   "駿河屋",
		NameEN: "Surugaya",
		Type:   registry.TypeDomesticProxy,
		Status: registry.StatusActive,
		Caps: domain.AdapterCaps{
			SupportsSearch:   true,
			SupportsRealtime: true,
			HasCategoryField: true,
			MaxQPS:           10,
		},
	}, surugaya.New(surugayaURL, nil))

	// Platform search service (bridges adapters to API layer).
	platformService := api.NewPlatformSearchService(reg, norm)

	// Stream manager for real-time search.
	streamManager := api.NewStreamManager()
	defer streamManager.Stop()

	// --- Elasticsearch client ---
	esURL := os.Getenv("ELASTICSEARCH_URL")
	if esURL == "" {
		esURL = "http://localhost:9200"
	}
	esIndexName := os.Getenv("ES_INDEX_NAME")
	if esIndexName == "" {
		esIndexName = "rakutao_products"
	}

	esClient, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: []string{esURL},
	})
	if err != nil {
		log.Fatalf("elasticsearch client: %v", err)
	}

	esSearcher := search.NewESSearcher(esClient, esIndexName)
	esFetcher := search.NewESProductFetcher(esClient, esIndexName)

	// --- Output pipeline (write search results to ES) ---
	esSink := output.NewESSink(esClient, esIndexName)
	outputRouter := output.NewRouter(esSink)
	mallTranslateURL := os.Getenv("MALL_TRANSLATE_URL")
	if mallTranslateURL == "" {
		mallTranslateURL = "http://cloud.gamestudio.tech/mall-translate"
	}
	translator := translate.NewMall(mallTranslateURL, nil)
	translateSink := output.NewTranslateSink(translator)
	productWriter := &asyncProductWriter{router: outputRouter, translateSink: translateSink, esFetcher: esFetcher}

	// --- Build handlers ---
	searchHandler := api.NewSearchHandler(gateway, esSearcher, streamManager)
	realtimeHandler := api.NewRealtimeHandler(streamManager, platformService, platformService, productWriter)
	productHandler := api.NewProductHandler(esFetcher, platformService, translator, productWriter)
	healthHandler := api.NewHealthHandler(reg)
	platformSearchHandler := api.NewPlatformSearchHandler(platformService, productWriter, esFetcher, translator)

	// Surugaya extension handler (direct client access).
	surugayaAdapter, _ := reg.GetAdapter("surugaya")
	surugayaHandler := api.NewSurugayaHandler(surugayaAdapter.(*surugaya.Adapter).Client(), translator)

	// --- Redis cache ---
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}

	var cacheMiddleware func(http.Handler) http.Handler
	cacheClient, err := cache.New(redisURL)
	if err != nil {
		log.Printf("WARNING: Redis cache disabled (invalid URL): %v", err)
	} else {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		if pingErr := cacheClient.Ping(ctx); pingErr != nil {
			log.Printf("WARNING: Redis not reachable, caching disabled: %v", pingErr)
			cacheClient = nil
		} else {
			cacheMiddleware = cache.Middleware(cacheClient)
		}
		cancel()
	}
	if cacheClient != nil {
		defer cacheClient.Close()
	}

	// --- SMTP email sender ---
	var emailSender *email.Sender
	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")
	smtpUser := os.Getenv("SMTP_USER")
	smtpPass := os.Getenv("SMTP_PASS")
	if smtpHost != "" && smtpUser != "" && smtpPass != "" {
		if smtpPort == "" {
			smtpPort = "587"
		}
		emailSender = email.NewSender(smtpHost, smtpPort, smtpUser, smtpPass)
		log.Printf("  SMTP:              %s:%s", smtpHost, smtpPort)
	} else {
		log.Printf("  SMTP:              disabled (SMTP_HOST/USER/PASS not set)")
	}

	// --- PostgreSQL + JWT + Auth/Cart/Favorite/OAuth handlers ---
	var (
		jwtManager            *auth.JWTManager
		authHandler           *api.AuthHandler
		oauthHandler          *api.OAuthHandler
		cartHandler           *api.CartHandler
		favoriteHandler       *api.FavoriteHandler
		sellerHandler         *api.SellerHandler
		walletHandler         *api.WalletHandler
		rechargeHandler       *api.RechargeHandler
		waybillHandler        *api.WaybillHandler
		orderHandler          *api.OrderHandler
		orderLinkHandler      *api.OrderLinkHandler
		globalOrderHandler    *api.GlobalOrderHandler
		recommendationHandler *api.RecommendationHandler
		adminChecker          api.UserAdminChecker
	)

	databaseURL := os.Getenv("DATABASE_URL")
	jwtSecret := os.Getenv("JWT_SECRET")

	if databaseURL != "" && jwtSecret != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		pgDB, err := db.New(ctx, databaseURL)
		cancel()
		if err != nil {
			log.Fatalf("postgres: %v", err)
		}
		defer pgDB.Close()

		ctx2, cancel2 := context.WithTimeout(context.Background(), 2*time.Second)
		if err := pgDB.Ping(ctx2); err != nil {
			log.Fatalf("postgres ping: %v", err)
		}
		cancel2()
		log.Printf("  PostgreSQL:        connected")

		jwtManager = auth.NewJWTManager(jwtSecret, 72*time.Hour)

		userRepo := repo.NewUserRepo(pgDB)
		cartRepo := repo.NewCartRepo(pgDB)
		favRepo := repo.NewFavoriteRepo(pgDB)
		sellerFollowRepo := repo.NewSellerFollowRepo(pgDB)

		authHandler = api.NewAuthHandler(userRepo, jwtManager, cacheClient, emailSender)
		cartHandler = api.NewCartHandler(cartRepo, esFetcher, translator)
		favoriteHandler = api.NewFavoriteHandler(favRepo, esFetcher, translator)
		sellerHandler = api.NewSellerHandler(sellerFollowRepo)

		walletRepo := repo.NewWalletRepo(pgDB)
		walletHandler = api.NewWalletHandler(walletRepo)
		adminChecker = walletRepo

		orderRepo := repo.NewOrderRepo(pgDB)
		orderSvc := service.NewOrderService(esFetcher, walletRepo, orderRepo, cartRepo)
		orderHandler = api.NewOrderHandler(orderSvc, orderRepo)

		// Global order sync (for international version)
		globalOrderRepo := repo.NewGlobalOrderRepo(pgDB)
		globalOrderSvc := service.NewGlobalOrderService(globalOrderRepo, orderRepo)
		globalOrderHandler = api.NewGlobalOrderHandler(globalOrderSvc)

		waybillRepo := repo.NewWaybillRepo(pgDB)
		waybillHandler = api.NewWaybillHandler(waybillRepo, waybillRepo, walletRepo)

		orderLinkRepo := repo.NewOrderLinkRepo(pgDB)
		orderLinkHandler = api.NewOrderLinkHandler(orderLinkRepo, walletRepo, orderRepo)

		rechargeRepo := repo.NewRechargeRepo(pgDB)
		rechargeHandler = api.NewRechargeHandler(rechargeRepo, walletRepo, nil)

		// Background: auto-cancel unpaid orders after 30 minutes.
		go func() {
			ticker := time.NewTicker(1 * time.Minute)
			defer ticker.Stop()
			for range ticker.C {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				n, err := orderRepo.CancelExpired(ctx, 30*time.Minute)
				cancel()
				if err != nil {
					log.Printf("[order] auto-cancel error: %v", err)
				} else if n > 0 {
					log.Printf("[order] auto-cancelled %d expired orders", n)
				}
			}
		}()

		// OAuth (Google + Apple)
		oauthRepo := repo.NewOAuthRepo(pgDB)
		googleClientID := os.Getenv("GOOGLE_CLIENT_ID")
		appleBundleID := os.Getenv("APPLE_BUNDLE_ID")
		if googleClientID != "" || appleBundleID != "" {
			oauthHandler = api.NewOAuthHandler(userRepo, oauthRepo, jwtManager, googleClientID, appleBundleID)
			log.Printf("  OAuth:             enabled (google=%v, apple=%v)", googleClientID != "", appleBundleID != "")
		}

		// Recommendation system
		prefRepo := repo.NewPreferenceRepo(pgDB)
		browseRepo := repo.NewBrowsingRepo(pgDB)
		searchHistRepo := repo.NewSearchHistoryRepo(pgDB)
		recWeightRepo := repo.NewRecWeightRepo(pgDB)

		recSvc := service.NewRecommendationService(
			prefRepo, browseRepo, searchHistRepo, recWeightRepo,
			favRepo, cartRepo, orderRepo, sellerFollowRepo,
			esClient, esIndexName, cacheClient,
		)
		recommendationHandler = api.NewRecommendationHandler(prefRepo, browseRepo, searchHistRepo, recSvc)
		// Inject browseRepo into productHandler for async browse tracking
		productHandler.SetBrowseTracker(browseRepo)
		log.Printf("  Recommendations:   enabled")

		// Background: recompute recommendation weights every 30 minutes.
		go func() {
			ticker := time.NewTicker(30 * time.Minute)
			defer ticker.Stop()
			for range ticker.C {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
				recSvc.RecomputeActiveUsers(ctx)
				cancel()
			}
		}()
	} else {
		log.Printf("  PostgreSQL:        disabled (DATABASE_URL or JWT_SECRET not set)")
	}

	// --- Build router ---
	router := api.NewRouter(api.RouterConfig{
		SearchHandler:         searchHandler,
		RealtimeHandler:       realtimeHandler,
		ProductHandler:        productHandler,
		HealthHandler:         healthHandler,
		PlatformSearchHandler: platformSearchHandler,
		SurugayaHandler:       surugayaHandler,
		AuthHandler:           authHandler,
		OAuthHandler:          oauthHandler,
		CartHandler:           cartHandler,
		FavoriteHandler:       favoriteHandler,
		SellerHandler:         sellerHandler,
		WalletHandler:         walletHandler,
		RechargeHandler:       rechargeHandler,
		WaybillHandler:        waybillHandler,
		OrderHandler:          orderHandler,
		OrderLinkHandler:      orderLinkHandler,
		GlobalOrderHandler:    globalOrderHandler,
		RecommendationHandler: recommendationHandler,
		AdminChecker:          adminChecker,
		JWTManager:            jwtManager,
		CacheMiddleware:       cacheMiddleware,
	})

	// --- Start server ---
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	addr := fmt.Sprintf(":%s", port)
	log.Printf("Rakutao Collection Gateway starting on %s", addr)
	log.Printf("  AI Service:        %s", aiServiceURL)
	log.Printf("  Yahoo Auction API: %s", yahooURL)
	log.Printf("  Surugaya API:      %s", surugayaURL)
	log.Printf("  Elasticsearch:     %s (index: %s)", esURL, esIndexName)
	if cacheClient != nil {
		log.Printf("  Redis Cache:       %s", redisURL)
	} else {
		log.Printf("  Redis Cache:       disabled")
	}

	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

// asyncProductWriter enriches products with translations and dispatches them
// to the output router for persistence (e.g. Elasticsearch).
// It first checks ES for existing translations to avoid redundant API calls.
type asyncProductWriter struct {
	router        *output.Router
	translateSink *output.TranslateSink
	esFetcher     *search.ESProductFetcher
}

func (w *asyncProductWriter) Dispatch(ctx context.Context, products []domain.UnifiedProduct) {
	// Step 1: Fetch existing translations from ES to reuse them.
	if w.esFetcher != nil {
		ids := make([]string, len(products))
		for i, p := range products {
			ids[i] = p.ID
		}
		existing, err := w.esFetcher.BulkGetTranslations(ctx, ids)
		if err == nil && len(existing) > 0 {
			for i := range products {
				tr, ok := existing[products[i].ID]
				if !ok {
					continue
				}
				// Merge existing translations into the product.
				if products[i].TitleTranslated == nil {
					products[i].TitleTranslated = make(map[string]string)
				}
				for lang, text := range tr.TitleTranslated {
					products[i].TitleTranslated[lang] = text
				}
				if products[i].DescTranslated == nil {
					products[i].DescTranslated = make(map[string]string)
				}
				for lang, text := range tr.DescTranslated {
					products[i].DescTranslated[lang] = text
				}
			}
		}
	}

	// Step 2: Only translate products/languages missing translations.
	if w.translateSink != nil {
		w.translateSink.Enrich(ctx, products)
	}

	// Step 3: Write to ES with complete translations.
	w.router.Dispatch(ctx, products)
}
