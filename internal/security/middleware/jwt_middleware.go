package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/udaypt/trading-app/internal/security"
)

type contextKey string

const UserContextKey contextKey = "user_claims"

func JWTMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		security.EnableCORS(w)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(security.AuthResponse{
				Status: "error",
				Error:  "Authorization header is required",
			})
			return
		}

		// Header format should be: Bearer <TOKEN>
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(security.AuthResponse{
				Status: "error",
				Error:  "Invalid authorization header format",
			})
			return
		}

		tokenStr := parts[1]
		claims, err := security.ValidateJWT(tokenStr)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(security.AuthResponse{
				Status: "error",
				Error:  "Token expired or invalid",
			})
			return
		}

		// Store user claims in context
		ctx := context.WithValue(r.Context(), UserContextKey, claims)
		next(w, r.WithContext(ctx))
	}
}
