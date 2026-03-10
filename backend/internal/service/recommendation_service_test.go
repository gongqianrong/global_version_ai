package service

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/rakutao/collection-gateway/internal/domain"
)

func TestTimeDecay(t *testing.T) {
	tests := []struct {
		name     string
		when     time.Time
		wantMin  float32
		wantMax  float32
	}{
		{"now", time.Now(), 0.99, 1.01},
		{"1 day ago", time.Now().Add(-24 * time.Hour), 0.96, 0.98},
		{"30 days ago", time.Now().Add(-30 * 24 * time.Hour), 0.50, 0.55},
		{"90 days ago", time.Now().Add(-90 * 24 * time.Hour), 0.26, 0.28},
		{"future", time.Now().Add(24 * time.Hour), 0.99, 1.01},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := timeDecay(tt.when)
			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("timeDecay() = %f, want [%f, %f]", got, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestWeightAccumulator(t *testing.T) {
	acc := make(weightAccumulator)

	acc.add("category", "フィギュア", 5.0)
	acc.add("category", "フィギュア", 1.0)
	acc.add("brand", "Bandai", 3.0)
	acc.add("category", "", 10.0) // empty value should be skipped

	if got := acc["category:フィギュア"]; got != 6.0 {
		t.Errorf("category:フィギュア = %f, want 6.0", got)
	}
	if got := acc["brand:Bandai"]; got != 3.0 {
		t.Errorf("brand:Bandai = %f, want 3.0", got)
	}
	if _, exists := acc["category:"]; exists {
		t.Error("empty value should not be accumulated")
	}
}

func TestParseProductID(t *testing.T) {
	tests := []struct {
		input      string
		wantPlat   string
		wantSource string
	}{
		{"surugaya_12345", "surugaya", "12345"},
		{"yahoo_auction_67890", "yahoo", "auction_67890"},
		{"nounderscore", "", "nounderscore"},
		{"", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			plat, src := parseProductID(tt.input)
			if plat != tt.wantPlat || src != tt.wantSource {
				t.Errorf("parseProductID(%q) = (%q, %q), want (%q, %q)", tt.input, plat, src, tt.wantPlat, tt.wantSource)
			}
		})
	}
}

func TestKeysFromMap(t *testing.T) {
	m := map[string]float32{"a": 1.0, "b": 2.0, "c": 3.0}
	keys := keysFromMap(m)
	if len(keys) != 3 {
		t.Fatalf("len = %d, want 3", len(keys))
	}
	keySet := map[string]bool{}
	for _, k := range keys {
		keySet[k] = true
	}
	for _, expected := range []string{"a", "b", "c"} {
		if !keySet[expected] {
			t.Errorf("missing key %q", expected)
		}
	}
}

func TestAvgWeight(t *testing.T) {
	if got := avgWeight(nil); got != 1.0 {
		t.Errorf("avgWeight(nil) = %f, want 1.0", got)
	}
	if got := avgWeight(map[string]float32{}); got != 1.0 {
		t.Errorf("avgWeight({}) = %f, want 1.0", got)
	}
	m := map[string]float32{"a": 2.0, "b": 4.0}
	if got := avgWeight(m); got != 3.0 {
		t.Errorf("avgWeight = %f, want 3.0", got)
	}
}

func TestTopNKeys(t *testing.T) {
	m := map[string]float32{
		"low":    1.0,
		"mid":    5.0,
		"high":   10.0,
		"higher": 15.0,
	}

	top2 := topNKeys(m, 2)
	if len(top2) != 2 {
		t.Fatalf("len = %d, want 2", len(top2))
	}
	if top2[0] != "higher" {
		t.Errorf("top2[0] = %q, want \"higher\"", top2[0])
	}
	if top2[1] != "high" {
		t.Errorf("top2[1] = %q, want \"high\"", top2[1])
	}

	// Request more than available
	all := topNKeys(m, 10)
	if len(all) != 4 {
		t.Fatalf("len = %d, want 4", len(all))
	}
}

func TestBuildESQuery_ForYou(t *testing.T) {
	svc := &RecommendationService{esIndex: "products"}
	weights := []domain.RecWeight{
		{Dimension: "category", Value: "フィギュア", Weight: 5.0},
		{Dimension: "brand", Value: "Bandai", Weight: 3.0},
	}
	query := svc.buildESQuery(weights, "for_you")
	if len(query) == 0 {
		t.Fatal("query should not be empty")
	}
	// Verify it's valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal(query, &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if parsed["size"].(float64) != 6 {
		t.Errorf("size = %v, want 6", parsed["size"])
	}
}

func TestBuildESQuery_NewArrivals(t *testing.T) {
	svc := &RecommendationService{esIndex: "products"}
	weights := []domain.RecWeight{
		{Dimension: "category", Value: "ゲーム", Weight: 5.0},
	}
	query := svc.buildESQuery(weights, "new_arrivals")
	var parsed map[string]interface{}
	if err := json.Unmarshal(query, &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	// Should sort by listed_at desc
	sorts := parsed["sort"].([]interface{})
	sortObj := sorts[0].(map[string]interface{})
	if _, ok := sortObj["listed_at"]; !ok {
		t.Error("new_arrivals should sort by listed_at")
	}
}

func TestBuildESQuery_Sellers(t *testing.T) {
	svc := &RecommendationService{esIndex: "products"}
	weights := []domain.RecWeight{
		{Dimension: "seller", Value: "seller123", Weight: 3.0},
	}
	query := svc.buildESQuery(weights, "sellers")
	var parsed map[string]interface{}
	if err := json.Unmarshal(query, &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	// Should have seller_id filter
	boolQ := parsed["query"].(map[string]interface{})["bool"].(map[string]interface{})
	filters := boolQ["filter"].([]interface{})
	found := false
	for _, f := range filters {
		fm := f.(map[string]interface{})
		if terms, ok := fm["terms"]; ok {
			if _, ok := terms.(map[string]interface{})["seller_id"]; ok {
				found = true
			}
		}
	}
	if !found {
		t.Error("sellers query should filter by seller_id")
	}
}

func TestBuildESQuery_EmptyWeights(t *testing.T) {
	svc := &RecommendationService{esIndex: "products"}
	query := svc.buildESQuery(nil, "for_you")
	var parsed map[string]interface{}
	if err := json.Unmarshal(query, &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
}
