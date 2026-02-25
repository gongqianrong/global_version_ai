package domain

import (
	"testing"
)

func TestAdapterCaps_CanRealtimeSearch_BothTrue(t *testing.T) {
	caps := AdapterCaps{SupportsSearch: true, SupportsRealtime: true}
	if !caps.CanRealtimeSearch() {
		t.Error("CanRealtimeSearch() should return true when both SupportsSearch and SupportsRealtime are true")
	}
}

func TestAdapterCaps_CanRealtimeSearch_RealtimeFalse(t *testing.T) {
	caps := AdapterCaps{SupportsSearch: true, SupportsRealtime: false}
	if caps.CanRealtimeSearch() {
		t.Error("CanRealtimeSearch() should return false when SupportsRealtime is false")
	}
}

func TestAdapterCaps_CanRealtimeSearch_SearchFalse(t *testing.T) {
	caps := AdapterCaps{SupportsSearch: false, SupportsRealtime: true}
	if caps.CanRealtimeSearch() {
		t.Error("CanRealtimeSearch() should return false when SupportsSearch is false")
	}
}

func TestAdapterCaps_CanRealtimeSearch_BothFalse(t *testing.T) {
	caps := AdapterCaps{SupportsSearch: false, SupportsRealtime: false}
	if caps.CanRealtimeSearch() {
		t.Error("CanRealtimeSearch() should return false when both are false")
	}
}

func TestRawProduct_Fields(t *testing.T) {
	raw := RawProduct{
		Platform: "mercari",
		RawID:    "m999",
		RawData: map[string]interface{}{
			"title": "Test Item",
			"price": 1500,
		},
		Normalized: nil,
	}

	if raw.Platform != "mercari" {
		t.Errorf("Platform = %q, want %q", raw.Platform, "mercari")
	}
	if raw.RawID != "m999" {
		t.Errorf("RawID = %q, want %q", raw.RawID, "m999")
	}
	if raw.RawData["title"] != "Test Item" {
		t.Errorf("RawData[\"title\"] = %v, want %q", raw.RawData["title"], "Test Item")
	}
	if raw.Normalized != nil {
		t.Error("Normalized should be nil")
	}
}

func TestRawProduct_WithNormalized(t *testing.T) {
	product := &UnifiedProduct{
		ID:     "mercari_m999",
		Title:  "Test Item",
		Status: StatusAvailable,
	}
	raw := RawProduct{
		Platform:   "mercari",
		RawID:      "m999",
		RawData:    map[string]interface{}{"title": "Test Item"},
		Normalized: product,
	}

	if raw.Normalized == nil {
		t.Fatal("Normalized should not be nil")
	}
	if raw.Normalized.ID != "mercari_m999" {
		t.Errorf("Normalized.ID = %q, want %q", raw.Normalized.ID, "mercari_m999")
	}
}

func TestAdapterCaps_AllFields(t *testing.T) {
	caps := AdapterCaps{
		SupportsSearch:       true,
		SupportsRealtime:     true,
		SupportsBatchCollect: true,
		HasBrandField:        true,
		HasCategoryField:     false,
		MaxQPS:               10,
	}

	if !caps.SupportsBatchCollect {
		t.Error("SupportsBatchCollect should be true")
	}
	if !caps.HasBrandField {
		t.Error("HasBrandField should be true")
	}
	if caps.HasCategoryField {
		t.Error("HasCategoryField should be false")
	}
	if caps.MaxQPS != 10 {
		t.Errorf("MaxQPS = %d, want %d", caps.MaxQPS, 10)
	}
}

func TestHealthStatus_Fields(t *testing.T) {
	h := HealthStatus{Status: "healthy", Message: "all good"}
	if h.Status != "healthy" {
		t.Errorf("Status = %q, want %q", h.Status, "healthy")
	}
	if h.Message != "all good" {
		t.Errorf("Message = %q, want %q", h.Message, "all good")
	}
}

func TestCollectParams_Fields(t *testing.T) {
	cp := CollectParams{
		Categories: []string{"bags", "shoes"},
		MaxItems:   100,
		SinceTime:  1700000000,
	}
	if len(cp.Categories) != 2 {
		t.Errorf("len(Categories) = %d, want %d", len(cp.Categories), 2)
	}
	if cp.MaxItems != 100 {
		t.Errorf("MaxItems = %d, want %d", cp.MaxItems, 100)
	}
	if cp.SinceTime != 1700000000 {
		t.Errorf("SinceTime = %d, want %d", cp.SinceTime, 1700000000)
	}
}

func TestSearchResult_Fields(t *testing.T) {
	sr := SearchResult{
		Products: []RawProduct{
			{Platform: "mercari", RawID: "m1"},
			{Platform: "mercari", RawID: "m2"},
		},
		Total:   50,
		HasMore: true,
	}
	if len(sr.Products) != 2 {
		t.Errorf("len(Products) = %d, want %d", len(sr.Products), 2)
	}
	if sr.Total != 50 {
		t.Errorf("Total = %d, want %d", sr.Total, 50)
	}
	if !sr.HasMore {
		t.Error("HasMore should be true")
	}
}
