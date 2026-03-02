package cache

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// TTL constants for different endpoint categories.
const (
	TTLProductDetail   = 10 * time.Minute
	TTLSurugayaDefault = 30 * time.Minute
	TTLSurugayaStatic  = 1 * time.Hour
	TTLFallback        = 24 * time.Hour // Fallback cache for upstream outages.
)

// Middleware returns chi-compatible middleware that caches GET responses in Redis.
//
// Two-layer caching strategy:
//   - Hot cache (short TTL): serves fresh data quickly.
//   - Fallback cache (24h TTL): serves stale data when upstream is down.
//
// If Redis is unavailable, requests pass through without caching.
func Middleware(c *Client) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only cache GET requests.
			if r.Method != http.MethodGet {
				next.ServeHTTP(w, r)
				return
			}

			// Determine TTL for this path. 0 means "do not cache".
			ttl := ttlForPath(r.URL.Path)
			if ttl == 0 {
				next.ServeHTTP(w, r)
				return
			}

			key := cacheKey(r.URL.Path, r.URL.Query(), r.Header.Get("Accept-Language"))
			fallbackKey := "fallback:" + key[len("cache:"):]

			// Try hot cache lookup.
			cached, err := c.Get(r.Context(), key)
			if err != nil {
				log.Printf("[CACHE] redis get error: %v", err)
			}
			if cached != nil {
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.Header().Set("X-Cache", "HIT")
				w.WriteHeader(http.StatusOK)
				w.Write(cached)
				return
			}

			// Cache miss: buffer the response (don't write through).
			cw := &captureWriter{
				ResponseWriter: w,
				status:         http.StatusOK,
				body:           &bytes.Buffer{},
			}
			next.ServeHTTP(cw, r)

			// Check if handler returned a successful response.
			bodyBytes := cw.body.Bytes()
			isSuccess := cw.status == http.StatusOK
			if isSuccess {
				var envelope struct {
					Code int `json:"code"`
				}
				if json.Unmarshal(bodyBytes, &envelope) != nil || envelope.Code != 0 {
					isSuccess = false
				}
			}

			if isSuccess {
				// Success: store in hot cache + fallback cache, then write response.
				if err := c.Set(r.Context(), key, bodyBytes, ttl); err != nil {
					log.Printf("[CACHE] redis set error: %v", err)
				}
				if err := c.Set(r.Context(), fallbackKey, bodyBytes, TTLFallback); err != nil {
					log.Printf("[CACHE] redis fallback set error: %v", err)
				}
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.Header().Set("X-Cache", "MISS")
				w.WriteHeader(http.StatusOK)
				w.Write(bodyBytes)
				return
			}

			// Failure: try fallback cache (stale data is better than error).
			stale, err := c.Get(r.Context(), fallbackKey)
			if err != nil {
				log.Printf("[CACHE] redis fallback get error: %v", err)
			}
			if stale != nil {
				log.Printf("[CACHE] serving stale fallback for %s", r.URL.Path)
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.Header().Set("X-Cache", "STALE")
				w.WriteHeader(http.StatusOK)
				w.Write(stale)
				return
			}

			// No fallback available: write original error response.
			w.Header().Set("X-Cache", "MISS")
			w.WriteHeader(cw.status)
			w.Write(bodyBytes)
		})
	}
}

// captureWriter wraps http.ResponseWriter to buffer the response
// without writing through. The middleware decides what to send after
// the handler completes.
type captureWriter struct {
	http.ResponseWriter
	status      int
	body        *bytes.Buffer
	wroteHeader bool
}

func (cw *captureWriter) WriteHeader(code int) {
	if !cw.wroteHeader {
		cw.status = code
		cw.wroteHeader = true
	}
}

func (cw *captureWriter) Write(b []byte) (int, error) {
	return cw.body.Write(b)
}

// cacheKey builds a deterministic cache key from path, sorted query parameters,
// and the Accept-Language header value so different languages are cached separately.
// The "request_id" parameter is excluded.
func cacheKey(path string, params url.Values, acceptLang string) string {
	filtered := make(url.Values)
	for k, v := range params {
		if k == "request_id" {
			continue
		}
		filtered[k] = v
	}

	keys := make([]string, 0, len(filtered))
	for k := range filtered {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	sb.WriteString("cache:")
	sb.WriteString(path)
	if len(keys) > 0 {
		sb.WriteByte('?')
		first := true
		for _, k := range keys {
			vals := filtered[k]
			sort.Strings(vals)
			for _, v := range vals {
				if !first {
					sb.WriteByte('&')
				}
				sb.WriteString(url.QueryEscape(k))
				sb.WriteByte('=')
				sb.WriteString(url.QueryEscape(v))
				first = false
			}
		}
	}

	if acceptLang != "" {
		sb.WriteString("|lang=")
		sb.WriteString(acceptLang)
	}

	return sb.String()
}

// ttlForPath returns the cache TTL for a URL path.
// Returns 0 for paths that should not be cached.
// Only hot-spot data is cached to avoid excessive Redis memory usage.
func ttlForPath(path string) time.Duration {
	switch {
	// Discounts and campaigns: 1 hour (shared globally, rarely changes).
	case path == "/api/v1/surugaya/discounts":
		return TTLSurugayaStatic
	case path == "/api/v1/surugaya/campaigns":
		return TTLSurugayaStatic

	// Product detail: 10 minutes (same product viewed by many users).
	case strings.HasPrefix(path, "/api/v1/products/"):
		return TTLProductDetail

	// Surugaya product extensions: 30 minutes (reviews, stores, campaign detail).
	// Excludes /comments (per-user, low hit rate).
	case path == "/api/v1/surugaya/campaigns/detail":
		return TTLSurugayaDefault
	case strings.HasPrefix(path, "/api/v1/surugaya/products/"):
		return TTLSurugayaDefault

	// Categories: 1 hour (menu structure rarely changes).
	case strings.HasPrefix(path, "/api/v1/surugaya/categories"):
		return TTLSurugayaStatic

	default:
		return 0
	}
}
