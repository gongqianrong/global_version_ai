package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/rakutao/collection-gateway/internal/auth"
)

// userIDCtxKey is the context key for the authenticated user ID.
const userIDCtxKey contextKey = "user_id"

// AuthRequired returns middleware that validates a Bearer JWT token and
// injects the user ID into the request context.
func AuthRequired(jm *auth.JWTManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header == "" {
				ErrorWithCode(w, r, http.StatusUnauthorized, 40100, "missing token")
				return
			}
			tokenStr, ok := strings.CutPrefix(header, "Bearer ")
			if !ok || tokenStr == "" {
				ErrorWithCode(w, r, http.StatusUnauthorized, 40100, "invalid token format")
				return
			}
			claims, err := jm.Validate(tokenStr)
			if err != nil {
				ErrorWithCode(w, r, http.StatusUnauthorized, 40101, "invalid or expired token")
				return
			}
			ctx := context.WithValue(r.Context(), userIDCtxKey, claims.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserIDFromContext extracts the authenticated user ID from the context.
// Returns 0 if not present (should not happen behind AuthRequired middleware).
func UserIDFromContext(ctx context.Context) int64 {
	id, _ := ctx.Value(userIDCtxKey).(int64)
	return id
}
