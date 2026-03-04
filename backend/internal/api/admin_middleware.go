package api

import (
	"context"
	"net/http"
)

// UserAdminChecker checks whether a user has admin privileges.
type UserAdminChecker interface {
	IsAdmin(ctx context.Context, userID int64) (bool, error)
}

// AdminRequired returns middleware that rejects non-admin users with 403.
// Must be used after AuthRequired so that UserIDFromContext is available.
func AdminRequired(checker UserAdminChecker) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := UserIDFromContext(r.Context())
			if userID == 0 {
				ErrorWithCode(w, r, http.StatusUnauthorized, 40100, "missing token")
				return
			}

			isAdmin, err := checker.IsAdmin(r.Context(), userID)
			if err != nil {
				ErrorWithCode(w, r, http.StatusInternalServerError, 50001, "failed to check admin status")
				return
			}
			if !isAdmin {
				ErrorWithCode(w, r, http.StatusForbidden, 40300, "admin access required")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
