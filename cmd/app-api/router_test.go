package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	httphandlers "trading-dashboard/cmd/app-api/http-handlers"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newRouter registers routes on the process-global http.DefaultServeMux, so
// it must only be invoked once per test binary. All route assertions live in
// this single test to respect that constraint.
func TestNewRouter_RegistersAllEndpoints(t *testing.T) {
	params := routerParams{
		SignIn:     httphandlers.NewSignIn(nil),
		SignUp:     httphandlers.NewSignUp(nil),
		Search:     httphandlers.NewSearch(nil, nil),
		MarketData: httphandlers.NewMarketData(nil),
	}

	router, err := newRouter(params)
	require.NoError(t, err)
	require.NotNil(t, router)

	// Every registered endpoint answers an OPTIONS preflight with 200 without
	// requiring a backing DB or network call, secured or not.
	for _, path := range []string{
		basePath + "/auth/register",
		basePath + "/auth/login",
		basePath + "/search",
		basePath + "/market-data",
	} {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodOptions, path, nil)
			rec := httptest.NewRecorder()
			http.DefaultServeMux.ServeHTTP(rec, req)
			assert.Equal(t, http.StatusOK, rec.Code)
		})
	}

	t.Run("secured endpoint rejects unauthenticated GET", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, basePath+"/search?q=x", nil)
		rec := httptest.NewRecorder()
		http.DefaultServeMux.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("unregistered path returns 404", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/not-a-route", nil)
		rec := httptest.NewRecorder()
		http.DefaultServeMux.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}
