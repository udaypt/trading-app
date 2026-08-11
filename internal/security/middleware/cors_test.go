package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/udaypt/trading-app/config"

	"github.com/stretchr/testify/assert"
)

func TestCors(t *testing.T) {
	nextCalled := func(called *bool) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			*called = true
			w.WriteHeader(http.StatusOK)
		}
	}

	t.Run("sets CORS headers and calls next", func(t *testing.T) {
		var called bool
		handler := Cors(nextCalled(&called))

		req := httptest.NewRequest(http.MethodGet, "/api/v1/search", nil)
		rec := httptest.NewRecorder()
		handler(rec, req)

		assert.True(t, called)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, config.CORS_ALLOWED_ORIGIN, rec.Header().Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "GET, POST, PUT, DELETE, OPTIONS", rec.Header().Get("Access-Control-Allow-Methods"))
		assert.Equal(t, "Content-Type, Authorization", rec.Header().Get("Access-Control-Allow-Headers"))
		assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	})

	t.Run("sets CORS headers on OPTIONS preflight and still calls next", func(t *testing.T) {
		var called bool
		handler := Cors(nextCalled(&called))

		req := httptest.NewRequest(http.MethodOptions, "/api/v1/search", nil)
		rec := httptest.NewRecorder()
		handler(rec, req)

		assert.True(t, called)
		assert.Equal(t, config.CORS_ALLOWED_ORIGIN, rec.Header().Get("Access-Control-Allow-Origin"))
	})

	t.Run("headers are present even when next rejects the request", func(t *testing.T) {
		reject := func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		}
		handler := Cors(reject)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/search", nil)
		rec := httptest.NewRecorder()
		handler(rec, req)

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		assert.Equal(t, config.CORS_ALLOWED_ORIGIN, rec.Header().Get("Access-Control-Allow-Origin"))
	})
}
