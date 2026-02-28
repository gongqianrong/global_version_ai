package surugaya

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rakutao/collection-gateway/internal/domain"
)

func TestPlatformID(t *testing.T) {
	a := New("http://localhost:9999", nil)
	if got := a.PlatformID(); got != "surugaya" {
		t.Errorf("PlatformID() = %q, want %q", got, "surugaya")
	}
}

func TestCapabilities(t *testing.T) {
	a := New("http://localhost:9999", nil)
	caps := a.Capabilities()
	if !caps.SupportsSearch {
		t.Error("expected SupportsSearch = true")
	}
	if !caps.SupportsRealtime {
		t.Error("expected SupportsRealtime = true")
	}
	if !caps.HasBrandField {
		t.Error("expected HasBrandField = true")
	}
}

// mockSearchResponse returns a realistic Surugaya search API response.
func mockSearchResponse() apiResponse {
	brand := "[バンダイ] "
	data := searchData{
		Item: []searchItem{
			{
				ID:          "663043159",
				Title:       "FW GUNDAM CONVERGE SB アーガマ級強襲用宇宙巡洋艦 ニカーヤ",
				Brand:       &brand,
				Category:    "食玩　トレーディングフィギュア",
				Condition:   "keyword_match",
				Link:        "https://www.suruga-ya.jp/product/detail/663043159",
				Pic:         "https://cdn.suruga-ya.jp/pics_webp/boxart_m/663043159m.jpg.webp",
				ReleaseDate: "2024-03-31 00:00:00",
				Sale:        []saleEntry{{Price: 4950, Type: "定価"}},
				State:       strPtr("品切れ"),
				StoreTag:    "(1点の中古品)",
			},
			{
				ID:       "GU375453",
				Title:    "1-27[SR]：機動戦士Gundam GQuuuuuuX キービジュアル",
				Brand:    &brand,
				Category: "アニメ系トレカ",
				Link:     "https://www.suruga-ya.jp/product/detail/GU375453",
				Pic:      "https://cdn.suruga-ya.jp/pics_webp/boxart_m/gu375453m.jpg.webp",
				Sale:     []saleEntry{{Price: 580, Type: "中古"}},
				State:    nil,
				StoreTag: "(5点の中古品)",
			},
		},
		MaxNum: 2,
		Total:  150,
	}

	dataBytes, _ := json.Marshal(data)
	return apiResponse{Code: 200, Data: dataBytes, Msg: "成功！"}
}

func TestSearch_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/suruga/product/search" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		q := r.URL.Query()
		if got := q.Get("search_word"); got != "ガンダム" {
			t.Errorf("search_word = %q, want %q", got, "ガンダム")
		}
		if got := q.Get("safe_search_enable"); got != "1" {
			t.Errorf("safe_search_enable = %q, want %q", got, "1")
		}

		resp := mockSearchResponse()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	a := New(srv.URL, nil)
	result, err := a.Search(context.Background(), domain.SearchQuery{
		Keyword:   "gundam",
		KeywordJA: "ガンダム",
		Page:      1,
		PageSize:  20,
	})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}

	if result.Total != 150 {
		t.Errorf("Total = %d, want 150", result.Total)
	}
	if len(result.Products) != 2 {
		t.Fatalf("len(Products) = %d, want 2", len(result.Products))
	}

	// First item: sold out.
	p0 := result.Products[0]
	if p0.Platform != "surugaya" {
		t.Errorf("Platform = %q", p0.Platform)
	}
	if p0.RawID != "663043159" {
		t.Errorf("RawID = %q", p0.RawID)
	}
	if p0.RawData["price"] != 4950.0 {
		t.Errorf("price = %v", p0.RawData["price"])
	}
	if p0.RawData["sale_type"] != "定価" {
		t.Errorf("sale_type = %v", p0.RawData["sale_type"])
	}
	if p0.RawData["status"] != "品切れ" {
		t.Errorf("status = %v, want 品切れ", p0.RawData["status"])
	}
	if p0.RawData["brand_name"] != "[バンダイ]" {
		t.Errorf("brand_name = %v", p0.RawData["brand_name"])
	}
	if p0.RawData["source_url"] != "https://www.suruga-ya.jp/product/detail/663043159" {
		t.Errorf("source_url = %v", p0.RawData["source_url"])
	}
	if p0.RawData["seller_id"] != "surugaya" {
		t.Errorf("seller_id = %v", p0.RawData["seller_id"])
	}

	// Second item: available (state=null).
	p1 := result.Products[1]
	if _, hasStatus := p1.RawData["status"]; hasStatus {
		t.Errorf("expected no status key for available item, got %v", p1.RawData["status"])
	}
	if p1.RawData["price"] != 580.0 {
		t.Errorf("price = %v", p1.RawData["price"])
	}
}

func TestSearch_PassesFilters(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if got := q.Get("category"); got != "ゲーム" {
			t.Errorf("category = %q, want %q", got, "ゲーム")
		}
		if got := q.Get("price"); got != "[1000,5000]" {
			t.Errorf("price = %q, want %q", got, "[1000,5000]")
		}
		if got := q.Get("rankBy"); got != "price:ascending" {
			t.Errorf("rankBy = %q, want %q", got, "price:ascending")
		}

		data := searchData{Item: []searchItem{}, Total: 0}
		dataBytes, _ := json.Marshal(data)
		resp := apiResponse{Code: 200, Data: dataBytes, Msg: "成功！"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	a := New(srv.URL, nil)
	_, err := a.Search(context.Background(), domain.SearchQuery{
		Keyword:    "テスト",
		Categories: []string{"ゲーム"},
		PriceMin:   1000,
		PriceMax:   5000,
		SortBy:     "price_asc",
		Page:       1,
		PageSize:   20,
	})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
}

func TestSearch_AdultContentFilter(t *testing.T) {
	tests := []struct {
		name       string
		rating     string
		wantSafe   string
	}{
		{"general blocks adult", domain.ContentRatingGeneral, "1"},
		{"r18 allows adult", domain.ContentRatingR18, "0"},
		{"all allows adult", "all", "0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				got := r.URL.Query().Get("safe_search_enable")
				if got != tt.wantSafe {
					t.Errorf("safe_search_enable = %q, want %q", got, tt.wantSafe)
				}
				data := searchData{Item: []searchItem{}, Total: 0}
				dataBytes, _ := json.Marshal(data)
				resp := apiResponse{Code: 200, Data: dataBytes, Msg: "成功！"}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(resp)
			}))
			defer srv.Close()

			a := New(srv.URL, nil)
			_, err := a.Search(context.Background(), domain.SearchQuery{
				Keyword:       "テスト",
				ContentRating: tt.rating,
				Page:          1,
				PageSize:      20,
			})
			if err != nil {
				t.Fatalf("Search error: %v", err)
			}
		})
	}
}

func TestSearch_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	a := New(srv.URL, nil)
	_, err := a.Search(context.Background(), domain.SearchQuery{Keyword: "test", Page: 1, PageSize: 20})
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestSearch_APIBusinessError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := apiResponse{Code: 500, Msg: "服务异常"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	a := New(srv.URL, nil)
	_, err := a.Search(context.Background(), domain.SearchQuery{Keyword: "test", Page: 1, PageSize: 20})
	if err == nil {
		t.Fatal("expected error for business error response")
	}
}

func TestGetProduct_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/suruga/product/detail/663043159" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		detail := detailData{
			BuyState:    true,
			Title:       "FW GUNDAM CONVERGE SB ニカーヤ",
			Desc:        "プレミアムバンダイ限定",
			StateDetail: "美品",
			ImgList:     []string{"img1.jpg", "img2.jpg"},
			Tags:        []string{"ガンダム", "食玩"},
			Category: categoryTree{
				Classify:   "食玩",
				ClassifyID: 100,
				Category: &categoryTree{
					Classify:   "トレーディングフィギュア",
					ClassifyID: 200,
				},
			},
			ShopSimpleInfo: shopInfo{
				ShopName: "駿河屋",
				ShopPic:  "shop.jpg",
			},
			Types: []productType{
				{
					Price:         4950,
					State:         "中古",
					Stock:         3,
					LimitPurchase: 1,
					CartID:        "cart123",
					TenpoCD:       "400490",
					BranchNumber:  "br001",
					Kubun:         "中古",
				},
			},
		}

		dataBytes, _ := json.Marshal(detail)
		resp := apiResponse{Code: 200, Data: dataBytes, Msg: "成功！"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	a := New(srv.URL, nil)
	rp, err := a.GetProduct(context.Background(), "663043159")
	if err != nil {
		t.Fatalf("GetProduct error: %v", err)
	}

	if rp.RawID != "663043159" {
		t.Errorf("RawID = %q", rp.RawID)
	}
	if rp.RawData["title"] != "FW GUNDAM CONVERGE SB ニカーヤ" {
		t.Errorf("title = %v", rp.RawData["title"])
	}
	if rp.RawData["price"] != 4950.0 {
		t.Errorf("price = %v", rp.RawData["price"])
	}
	if rp.RawData["description"] != "プレミアムバンダイ限定" {
		t.Errorf("description = %v", rp.RawData["description"])
	}
	if rp.RawData["state_detail"] != "美品" {
		t.Errorf("state_detail = %v", rp.RawData["state_detail"])
	}
	if rp.RawData["buy_state"] != true {
		t.Errorf("buy_state = %v", rp.RawData["buy_state"])
	}
	if rp.RawData["category"] != "食玩" {
		t.Errorf("category = %v", rp.RawData["category"])
	}
	if rp.RawData["seller_name"] != "駿河屋" {
		t.Errorf("seller_name = %v", rp.RawData["seller_name"])
	}

	// Check images.
	imgs, ok := rp.RawData["images"].([]string)
	if !ok || len(imgs) != 2 {
		t.Errorf("images = %v", rp.RawData["images"])
	}

	// Check source URL.
	if rp.RawData["source_url"] != "https://www.suruga-ya.jp/product/detail/663043159" {
		t.Errorf("source_url = %v", rp.RawData["source_url"])
	}

	// Check category path.
	catPath, ok := rp.RawData["category_path"].([]string)
	if !ok || len(catPath) != 2 {
		t.Errorf("category_path = %v", rp.RawData["category_path"])
	}
}

func TestGetProduct_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	a := New(srv.URL, nil)
	_, err := a.GetProduct(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for 404 response")
	}
}

func TestHealthCheck_Healthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		data := searchData{Item: []searchItem{}, Total: 0}
		dataBytes, _ := json.Marshal(data)
		resp := apiResponse{Code: 200, Data: dataBytes}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	a := New(srv.URL, nil)
	status := a.HealthCheck(context.Background())
	if status.Status != "healthy" {
		t.Errorf("Status = %q, want %q", status.Status, "healthy")
	}
}

func TestHealthCheck_Unhealthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "down", http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	a := New(srv.URL, nil)
	status := a.HealthCheck(context.Background())
	if status.Status != "unhealthy" {
		t.Errorf("Status = %q, want %q", status.Status, "unhealthy")
	}
}

// --- searchItemToRawProduct tests ---

func TestSearchItemToRawProduct_SoldOut(t *testing.T) {
	state := "品切れ"
	brand := "[バンダイ] "
	it := searchItem{
		ID:          "sg-001",
		Title:       "テスト商品",
		Brand:       &brand,
		Category:    "フィギュア",
		Link:        "https://www.suruga-ya.jp/product/detail/sg-001",
		Pic:         "https://cdn.suruga-ya.jp/img.jpg",
		ReleaseDate: "2024-06-15 00:00:00",
		Sale:        []saleEntry{{Price: 3000, Type: "中古"}},
		State:       &state,
		StoreTag:    "(1点の中古品)",
	}

	rp := searchItemToRawProduct(it)
	if rp.RawData["status"] != "品切れ" {
		t.Errorf("status = %v, want 品切れ", rp.RawData["status"])
	}
	if rp.RawData["brand_name"] != "[バンダイ]" {
		t.Errorf("brand_name = %v", rp.RawData["brand_name"])
	}
	if rp.RawData["price"] != 3000.0 {
		t.Errorf("price = %v", rp.RawData["price"])
	}
}

func TestSearchItemToRawProduct_Available(t *testing.T) {
	it := searchItem{
		ID:    "sg-002",
		Title: "テスト",
		Link:  "https://www.suruga-ya.jp/product/detail/sg-002",
		Sale:  []saleEntry{{Price: 500, Type: "新品"}},
		State: nil, // null = available
	}

	rp := searchItemToRawProduct(it)
	if _, has := rp.RawData["status"]; has {
		t.Errorf("expected no status for available item, got %v", rp.RawData["status"])
	}
	if rp.RawData["sale_type"] != "新品" {
		t.Errorf("sale_type = %v", rp.RawData["sale_type"])
	}
}

func TestSearchItemToRawProduct_NoBrand(t *testing.T) {
	it := searchItem{
		ID:    "sg-003",
		Title: "テスト",
		Link:  "link",
		Brand: nil,
		Sale:  []saleEntry{{Price: 100, Type: "中古"}},
	}

	rp := searchItemToRawProduct(it)
	if _, has := rp.RawData["brand_name"]; has {
		t.Errorf("expected no brand_name for nil brand")
	}
}

func TestSearchItemToRawProduct_MultipleSales(t *testing.T) {
	it := searchItem{
		ID:    "sg-004",
		Title: "テスト",
		Link:  "link",
		Sale: []saleEntry{
			{Price: 4320, Type: "中古"},
			{Price: 2860, Type: "定価"},
		},
	}

	rp := searchItemToRawProduct(it)
	// Primary price is first entry.
	if rp.RawData["price"] != 4320.0 {
		t.Errorf("price = %v", rp.RawData["price"])
	}
	// Should have sales array.
	sales, ok := rp.RawData["sales"].([]map[string]interface{})
	if !ok || len(sales) != 2 {
		t.Errorf("sales = %v", rp.RawData["sales"])
	}
}

// --- mapSortBy tests ---

func TestMapSortBy(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"price_asc", "price:ascending"},
		{"price_desc", "price:descending"},
		{"newest", "modificationTime:descending"},
		{"release_date_desc", "release_date(int):descending"},
		{"", "relavancy(int)"},
		{"unknown", "relavancy(int)"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := mapSortBy(tt.input)
			if got != tt.want {
				t.Errorf("mapSortBy(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// --- mapConditionToSaleClassified tests ---

func TestMapConditionToSaleClassified(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{"nil", nil, nil},
		{"empty", []string{}, nil},
		{"new only", []string{domain.ConditionNew}, []string{"新品"}},
		{"used conditions", []string{domain.ConditionGood, domain.ConditionFair}, []string{"中古"}},
		{"mixed", []string{domain.ConditionNew, domain.ConditionGood}, []string{"新品", "中古"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapConditionToSaleClassified(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("len = %d, want %d; got %v", len(got), len(tt.want), got)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("got[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

// --- helper ---

func strPtr(s string) *string {
	return &s
}
