package middleware

import (
	"context"
	"finalneta/service"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// AuthorityAuthMiddleware validates JWT token for authority/officer authentication
type AuthorityAuthMiddleware struct {
	authorityService *service.AuthorityService
	jwtSecret        []byte
}

// NewAuthorityAuthMiddleware creates a new authority auth middleware
func NewAuthorityAuthMiddleware(authorityService *service.AuthorityService, jwtSecret string) *AuthorityAuthMiddleware {
	return &AuthorityAuthMiddleware{
		authorityService: authorityService,
		jwtSecret:        []byte(jwtSecret),
	}
}

// RequireAuthorityAuth middleware validates JWT token and sets officer_id in context
func (m *AuthorityAuthMiddleware) RequireAuthorityAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			respondWithError(w, http.StatusUnauthorized, "Unauthorized", "Authorization header required")
			return
		}

		// Check Bearer prefix
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			respondWithError(w, http.StatusUnauthorized, "Unauthorized", "Invalid authorization format. Expected: Bearer <token>")
			return
		}

		tokenString := parts[1]

		// Parse and validate token
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// Validate signing method
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return m.jwtSecret, nil
		})

		if err != nil {
			respondWithError(w, http.StatusUnauthorized, "Unauthorized", "Invalid or expired token")
			return
		}

		if !token.Valid {
			respondWithError(w, http.StatusUnauthorized, "Unauthorized", "Invalid token")
			return
		}

		// Extract officer_id from claims
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			respondWithError(w, http.StatusUnauthorized, "Unauthorized", "Invalid token claims")
			return
		}

		// Check actor_type to ensure this is an authority token
		actorType, ok := claims["actor_type"].(string)
		if !ok || actorType != "authority" {
			respondWithError(w, http.StatusUnauthorized, "Unauthorized", "Invalid token type - authority token required")
			return
		}

		officerIDFloat, ok := claims["officer_id"].(float64)
		if !ok {
			respondWithError(w, http.StatusUnauthorized, "Unauthorized", "Invalid token: officer_id not found")
			return
		}

		officerID := int64(officerIDFloat)
		authorityLevel := 1
		if lv, ok := claims["authority_level"].(float64); ok {
			authorityLevel = int(lv)
		}

		if m.authorityService != nil {
			exists, err := m.authorityService.VerifyOfficerExists(officerID)
			if err != nil || !exists {
				respondWithError(w, http.StatusUnauthorized, "Unauthorized", "Officer not found or inactive")
				return
			}
		}

		ctx := context.WithValue(r.Context(), "officer_id", officerID)
		ctx = context.WithValue(ctx, "authority_level", authorityLevel)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
