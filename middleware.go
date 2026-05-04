package main

import (
	"context"
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// contextKey is a custom type to avoid key collisions in request context
type contextKey string

const userContextKey contextKey = "user"

// Claims defines what we store inside the JWT token
type Claims struct {
	UserID int    `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// getJWTSecret reads the secret from env — always use env, never hardcode in production
func getJWTSecret() []byte {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "examdash_dev_secret" // fallback for local dev only
	}
	return []byte(secret)
}

// AuthMiddleware protects routes — rejects requests with no valid JWT token
func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")

		// Must have "Bearer <token>" format
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			writeError(w, http.StatusUnauthorized, "unauthorized", "missing or invalid Authorization header")
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		claims := &Claims{}

		token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
			return getJWTSecret(), nil
		})

		if err != nil || !token.Valid {
			writeError(w, http.StatusUnauthorized, "unauthorized", "invalid or expired token")
			return
		}

		// Attach the claims to the request context so handlers can read them
		ctx := context.WithValue(r.Context(), userContextKey, claims)
		next(w, r.WithContext(ctx))
	}
}

// ReviewerOnly middleware — only allows users with role "reviewer"
func ReviewerOnly(next http.HandlerFunc) http.HandlerFunc {
	return AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		claims := getClaimsFromContext(r)
		if claims == nil || claims.Role != "reviewer" {
			writeError(w, http.StatusForbidden, "forbidden", "only reviewers can access this route")
			return
		}
		next(w, r)
	})
}

// getClaimsFromContext pulls the JWT claims out of the request context
func getClaimsFromContext(r *http.Request) *Claims {
	claims, _ := r.Context().Value(userContextKey).(*Claims)
	return claims
}

// generateToken creates a signed JWT string from claims — used by handlers and tests
func generateToken(claims *Claims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(getJWTSecret())
}