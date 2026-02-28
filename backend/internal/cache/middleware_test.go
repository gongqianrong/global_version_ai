package cache

import (
	"net/url"
	"testing"
	"time"
)

func TestCacheKey_BasicPath(t *testing.T) {
	got := cacheKey("/api/v1/search", url.Values{
		"keyword": {"gundam"},
		"page":    {"1"},
	})
	want := "cache:/api/v1/search?keyword=gundam&page=1"
	if got != want {
		t.Errorf("cacheKey = %q, want %q", got, want)
	}
}

func TestCacheKey_SortedParams(t *testing.T) {
	got := cacheKey("/api/v1/search", url.Values{
		"page":    {"1"},
		"keyword": {"gundam"},
		"brand":   {"bandai"},
	})
	want := "cache:/api/v1/search?brand=bandai&keyword=gundam&page=1"
	if got != want {
		t.Errorf("cacheKey = %q, want %q", got, want)
	}
}

func TestCacheKey_ExcludesRequestID(t *testing.T) {
	got := cacheKey("/api/v1/search", url.Values{
		"keyword":    {"gundam"},
		"request_id": {"abc123"},
	})
	want := "cache:/api/v1/search?keyword=gundam"
	if got != want {
		t.Errorf("cacheKey = %q, want %q", got, want)
	}
}

func TestCacheKey_NoParams(t *testing.T) {
	got := cacheKey("/api/v1/surugaya/discounts", url.Values{})
	want := "cache:/api/v1/surugaya/discounts"
	if got != want {
		t.Errorf("cacheKey = %q, want %q", got, want)
	}
}

func TestCacheKey_ProductWithID(t *testing.T) {
	got := cacheKey("/api/v1/products/surugaya_663043159", url.Values{
		"lang": {"zh-TW"},
	})
	want := "cache:/api/v1/products/surugaya_663043159?lang=zh-TW"
	if got != want {
		t.Errorf("cacheKey = %q, want %q", got, want)
	}
}

func TestTTLForPath(t *testing.T) {
	tests := []struct {
		path string
		want time.Duration
	}{
		// Not cached: health, stream, search, comments.
		{"/health", 0},
		{"/api/v1/search/stream/abc123", 0},
		{"/api/v1/search", 0},
		{"/api/v1/platform/search", 0},
		{"/api/v1/surugaya/comments", 0},
		{"/unknown/path", 0},
		// Cached: product detail, surugaya extensions, discounts, campaigns.
		{"/api/v1/products/surugaya_12345", TTLProductDetail},
		{"/api/v1/surugaya/products/123/reviews", TTLSurugayaDefault},
		{"/api/v1/surugaya/products/123/stores", TTLSurugayaDefault},
		{"/api/v1/surugaya/campaigns/detail", TTLSurugayaDefault},
		{"/api/v1/surugaya/discounts", TTLSurugayaStatic},
		{"/api/v1/surugaya/campaigns", TTLSurugayaStatic},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := ttlForPath(tt.path)
			if got != tt.want {
				t.Errorf("ttlForPath(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}
