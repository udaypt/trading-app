package mutualfund

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/udaypt/trading-app/internal/domain/usecase/trading"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func withMFHistoryServer(t *testing.T, handler http.HandlerFunc) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	orig := MFHistoryAPIURL
	MFHistoryAPIURL = srv.URL + "/%s"
	t.Cleanup(func() { MFHistoryAPIURL = orig })
}

func TestHistory_GetAssetType(t *testing.T) {
	h := NewHistory()
	assert.Equal(t, trading.MutualFund, h.GetAssetType())
}

func TestHistory_Fetch(t *testing.T) {
	t.Run("parses, converts dates, and orders oldest to newest", func(t *testing.T) {
		withMFHistoryServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{"data":[
				{"date":"03-01-2024","nav":"12.5"},
				{"date":"02-01-2024","nav":"12.0"},
				{"date":"01-01-2024","nav":"11.5"}
			]}`))
		})

		h := NewHistory()
		points, err := h.Fetch(context.Background(), "12345", 30)
		require.NoError(t, err)
		require.Len(t, points, 3)

		assert.Equal(t, "2024-01-01", points[0].Date)
		assert.Equal(t, 11.5, points[0].Price)
		assert.Equal(t, "2024-01-03", points[2].Date)
		assert.Equal(t, 12.5, points[2].Price)
	})

	t.Run("days limits how many raw entries are considered", func(t *testing.T) {
		withMFHistoryServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{"data":[
				{"date":"03-01-2024","nav":"3"},
				{"date":"02-01-2024","nav":"2"},
				{"date":"01-01-2024","nav":"1"}
			]}`))
		})

		h := NewHistory()
		points, err := h.Fetch(context.Background(), "12345", 2)
		require.NoError(t, err)
		require.Len(t, points, 2)
		assert.Equal(t, "2024-01-02", points[0].Date)
		assert.Equal(t, "2024-01-03", points[1].Date)
	})

	t.Run("skips entries with unparsable NAV", func(t *testing.T) {
		withMFHistoryServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{"data":[
				{"date":"02-01-2024","nav":"not-a-number"},
				{"date":"01-01-2024","nav":"11.5"}
			]}`))
		})

		h := NewHistory()
		points, err := h.Fetch(context.Background(), "12345", 30)
		require.NoError(t, err)
		require.Len(t, points, 1)
		assert.Equal(t, "2024-01-01", points[0].Date)
	})

	t.Run("keeps raw date string when date format is unrecognized", func(t *testing.T) {
		withMFHistoryServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{"data":[{"date":"2024/01/01","nav":"11.5"}]}`))
		})

		h := NewHistory()
		points, err := h.Fetch(context.Background(), "12345", 30)
		require.NoError(t, err)
		require.Len(t, points, 1)
		assert.Equal(t, "2024/01/01", points[0].Date)
	})

	t.Run("empty data returns no points", func(t *testing.T) {
		withMFHistoryServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{"data":[]}`))
		})

		h := NewHistory()
		points, err := h.Fetch(context.Background(), "12345", 30)
		require.NoError(t, err)
		assert.Empty(t, points)
	})

	t.Run("malformed json returns error", func(t *testing.T) {
		withMFHistoryServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`not json`))
		})

		h := NewHistory()
		_, err := h.Fetch(context.Background(), "12345", 30)
		assert.Error(t, err)
	})

	t.Run("invalid URL from a malformed scheme code returns a request-build error", func(t *testing.T) {
		orig := MFHistoryAPIURL
		MFHistoryAPIURL = "http://example.com/%s"
		t.Cleanup(func() { MFHistoryAPIURL = orig })

		h := NewHistory()
		// A control character makes the formatted URL invalid input to
		// http.NewRequestWithContext.
		_, err := h.Fetch(context.Background(), "bad\ncode", 30)
		assert.Error(t, err)
	})

	t.Run("unreachable server returns a transport error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		closedURL := srv.URL
		srv.Close() // nothing is listening anymore

		orig := MFHistoryAPIURL
		MFHistoryAPIURL = closedURL + "/%s"
		t.Cleanup(func() { MFHistoryAPIURL = orig })

		h := NewHistory()
		_, err := h.Fetch(context.Background(), "12345", 30)
		assert.Error(t, err)
	})
}
