package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/udaypt/trading-app/internal/security"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJWTMiddleware(t *testing.T) {
	nextCalled := func(called *bool) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			*called = true
			w.WriteHeader(http.StatusOK)
		}
	}

	t.Run("OPTIONS preflight bypasses auth", func(t *testing.T) {
		var called bool
		handler := JWTMiddleware(nextCalled(&called))

		req := httptest.NewRequest(http.MethodOptions, "/api/v1/search", nil)
		rec := httptest.NewRecorder()
		handler(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.False(t, called)
		assert.Equal(t, "*", rec.Header().Get("Access-Control-Allow-Origin"))
	})

	t.Run("missing Authorization header is rejected", func(t *testing.T) {
		var called bool
		handler := JWTMiddleware(nextCalled(&called))

		req := httptest.NewRequest(http.MethodGet, "/api/v1/search", nil)
		rec := httptest.NewRecorder()
		handler(rec, req)

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		assert.False(t, called)
	})

	t.Run("malformed Authorization header is rejected", func(t *testing.T) {
		var called bool
		handler := JWTMiddleware(nextCalled(&called))

		req := httptest.NewRequest(http.MethodGet, "/api/v1/search", nil)
		req.Header.Set("Authorization", "Basic abc123")
		rec := httptest.NewRecorder()
		handler(rec, req)

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		assert.False(t, called)
	})

	t.Run("invalid token is rejected", func(t *testing.T) {
		var called bool
		handler := JWTMiddleware(nextCalled(&called))

		req := httptest.NewRequest(http.MethodGet, "/api/v1/search", nil)
		req.Header.Set("Authorization", "Bearer not-a-real-token")
		rec := httptest.NewRecorder()
		handler(rec, req)

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		assert.False(t, called)
	})

	t.Run("valid token calls next with claims in context", func(t *testing.T) {
		token, err := security.GenerateJWT(9, "user@example.com")
		require.NoError(t, err)

		var gotClaims *security.Claims
		next := func(w http.ResponseWriter, r *http.Request) {
			gotClaims, _ = r.Context().Value(UserContextKey).(*security.Claims)
			w.WriteHeader(http.StatusOK)
		}
		handler := JWTMiddleware(next)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/search", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		handler(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		require.NotNil(t, gotClaims)
		assert.Equal(t, int64(9), gotClaims.UserID)
		assert.Equal(t, "user@example.com", gotClaims.Email)
	})
}
