package middleware

import (
	"net/http"
	"os"
	"strings"
)

// RequireAdminAuth validates env-based static token (ADMIN_TOKEN) only. Fully isolated from citizen/authority auth.
// Missing or mismatch → 403. No role framework; single token for trusted internal operator.
func RequireAdminAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		adminToken := os.Getenv("ADMIN_TOKEN")
		if adminToken == "" {
			respondWithError(w, http.StatusForbidden, "Forbidden", "Admin access not configured")
			return
		}
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			respondWithError(w, http.StatusForbidden, "Forbidden", "Authorization header required")
			return
		}
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			respondWithError(w, http.StatusForbidden, "Forbidden", "Invalid authorization format")
			return
		}
		if parts[1] != adminToken {
			respondWithError(w, http.StatusForbidden, "Forbidden", "Invalid admin token")
			return
		}
		next.ServeHTTP(w, r)
	})
}
