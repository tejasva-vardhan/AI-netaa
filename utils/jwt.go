package utils

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// GenerateJWT generates a JWT token for authenticated user
func GenerateJWT(userID int64, secret []byte, expiresInHours int) (string, error) {
	expiresAt := time.Now().Add(time.Duration(expiresInHours) * time.Hour)

	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     expiresAt.Unix(),
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

// GenerateAuthorityJWT generates a JWT token for authenticated authority/officer; token is scoped to authority only (citizen endpoints must reject it).
func GenerateAuthorityJWT(officerID int64, authorityLevel int, secret []byte, expiresInHours int) (string, error) {
	expiresAt := time.Now().Add(time.Duration(expiresInHours) * time.Hour)
	claims := jwt.MapClaims{
		"officer_id":      officerID,
		"authority_level": authorityLevel,
		"actor_type":      "authority",
		"exp":             expiresAt.Unix(),
		"iat":             time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}
