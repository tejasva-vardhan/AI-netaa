package middleware

import (
	"context"
	"finalneta/service"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// AuthMiddleware validates JWT token and extracts user_id
type AuthMiddleware struct {
	userService *service.UserService
	jwtSecret   []byte
}

// NewAuthMiddleware creates a new auth middleware
func NewAuthMiddleware(userService *service.UserService, jwtSecret string) *AuthMiddleware {
	return &AuthMiddleware{
		userService: userService,
		jwtSecret:   []byte(jwtSecret),
	}
}

// RequireAuth middleware validates JWT token and sets user_id in context
func (m *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			respondWithError(w, http.StatusUnauthorized, "Unauthorized", "Authorization header required. Please verify your phone number first.")
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
			respondWithError(w, http.StatusUnauthorized, "Unauthorized", "Invalid or expired token. Please verify your phone number again.")
			return
		}

		if !token.Valid {
			respondWithError(w, http.StatusUnauthorized, "Unauthorized", "Invalid token. Please verify your phone number again.")
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			respondWithError(w, http.StatusUnauthorized, "Unauthorized", "Invalid token claims.")
			return
		}
		// Reject authority tokens on citizen endpoints (token must be citizen-scoped only).
		if at, _ := claims["actor_type"].(string); at == "authority" {
			respondWithError(w, http.StatusUnauthorized, "Unauthorized", "Authority token not accepted for this endpoint.")
			return
		}

		userIDFloat, ok := claims["user_id"].(float64)
		if !ok {
			respondWithError(w, http.StatusUnauthorized, "Unauthorized", "Invalid token: user_id not found.")
			return
		}

		userID := int64(userIDFloat)

		// Verify user exists and phone is verified
		if m.userService != nil {
			exists, err := m.userService.VerifyUserExists(userID)
			if err != nil || !exists {
				respondWithError(w, http.StatusUnauthorized, "Unauthorized", "User not found. Please verify your phone number first.")
				return
			}

			phoneVerified, err := m.userService.VerifyUserPhoneVerified(userID)
			if err != nil || !phoneVerified {
				respondWithError(w, http.StatusUnauthorized, "Unauthorized", "Phone not verified. Please verify your phone number first.")
				return
			}
		}

		// Set user_id in context
		ctx := context.WithValue(r.Context(), "user_id", userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Helper function for error responses
func respondWithError(w http.ResponseWriter, statusCode int, errorType, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json := fmt.Sprintf(`{"error":"%s","message":"%s","code":%d}`, errorType, message, statusCode)
	w.Write([]byte(json))
}
