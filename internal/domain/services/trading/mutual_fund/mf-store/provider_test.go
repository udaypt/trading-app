package mfstore

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	mutualfund "github.com/udaypt/trading-app/internal/domain/services/trading/mutual_fund"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func withMFSyncServer(t *testing.T, handler http.HandlerFunc) *http.Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	orig := MFSyncAPIURL
	MFSyncAPIURL = srv.URL
	t.Cleanup(func() { MFSyncAPIURL = orig })

	return srv.Client()
}

func TestProvider_Fetch(t *testing.T) {
	t.Run("decodes a successful response", func(t *testing.T) {
		payload := []mutualfund.Scheme{
			{SchemeCode: 10, SchemeName: "Test Fund One"},
			{SchemeCode: 11, SchemeName: "Test Fund Two"},
		}
		client := withMFSyncServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(payload)
		})

		provider := &Provider{client: client}
		schemes, err := provider.Fetch(context.Background())
		require.NoError(t, err)
		assert.Len(t, schemes, 2)
		assert.Equal(t, "Test Fund One", schemes[0].SchemeName)
	})

	t.Run("non-200 status returns error", func(t *testing.T) {
		client := withMFSyncServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		})

		provider := &Provider{client: client}
		_, err := provider.Fetch(context.Background())
		assert.Error(t, err)
	})

	t.Run("malformed json returns error", func(t *testing.T) {
		client := withMFSyncServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("not json"))
		})

		provider := &Provider{client: client}
		_, err := provider.Fetch(context.Background())
		assert.Error(t, err)
	})

	t.Run("invalid URL returns a request-build error", func(t *testing.T) {
		orig := MFSyncAPIURL
		MFSyncAPIURL = "http://example.com/\n"
		t.Cleanup(func() { MFSyncAPIURL = orig })

		provider := &Provider{client: &http.Client{Timeout: time.Second}}
		_, err := provider.Fetch(context.Background())
		assert.Error(t, err)
	})

	t.Run("unreachable server returns a transport error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		closedURL := srv.URL
		srv.Close()

		orig := MFSyncAPIURL
		MFSyncAPIURL = closedURL
		t.Cleanup(func() { MFSyncAPIURL = orig })

		provider := &Provider{client: &http.Client{Timeout: time.Second}}
		_, err := provider.Fetch(context.Background())
		assert.Error(t, err)
	})
}

func TestNewProvider(t *testing.T) {
	provider := NewProvider()
	require.NotNil(t, provider)
	require.NotNil(t, provider.client)
}
