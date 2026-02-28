// backend/cmd/gateway/main.go
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/rakutao/collection-gateway/internal/adapter/surugaya"
	yahoo "github.com/rakutao/collection-gateway/internal/adapter/yahoo_auction"
	"github.com/rakutao/collection-gateway/internal/aiclient"
	"github.com/rakutao/collection-gateway/internal/api"
	"github.com/rakutao/collection-gateway/internal/brand"
	"github.com/rakutao/collection-gateway/internal/domain"
	"github.com/rakutao/collection-gateway/internal/filter"
	"github.com/rakutao/collection-gateway/internal/normalizer"
	"github.com/rakutao/collection-gateway/internal/registry"
	"github.com/rakutao/collection-gateway/internal/search"
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

	// --- Build handlers ---
	searchHandler := api.NewSearchHandler(gateway, esSearcher, streamManager)
	realtimeHandler := api.NewRealtimeHandler(streamManager, platformService, platformService)
	productHandler := api.NewProductHandler(esFetcher, platformService)
	healthHandler := api.NewHealthHandler(reg)
	platformSearchHandler := api.NewPlatformSearchHandler(platformService)

	// Surugaya extension handler (direct client access).
	surugayaAdapter, _ := reg.GetAdapter("surugaya")
	surugayaHandler := api.NewSurugayaHandler(surugayaAdapter.(*surugaya.Adapter).Client())

	// --- Build router ---
	router := api.NewRouter(api.RouterConfig{
		SearchHandler:         searchHandler,
		RealtimeHandler:       realtimeHandler,
		ProductHandler:        productHandler,
		HealthHandler:         healthHandler,
		PlatformSearchHandler: platformSearchHandler,
		SurugayaHandler:       surugayaHandler,
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

	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
