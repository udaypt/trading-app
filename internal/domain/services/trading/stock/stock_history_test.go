package stock

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/udaypt/trading-app/internal/domain/usecase/trading"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func withStockHistoryServer(t *testing.T, handler http.HandlerFunc) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	orig := StockHistoryAPIURL
	StockHistoryAPIURL = srv.URL + "/%s?range=%s"
	t.Cleanup(func() { StockHistoryAPIURL = orig })
}

func TestHistory_StockGetAssetType(t *testing.T) {
	h := NewHistory()
	assert.Equal(t, trading.Stock, h.GetAssetType())
}

func TestHistory_StockFetch(t *testing.T) {
	t.Run("parses timestamps and closes, skipping zero-close holidays", func(t *testing.T) {
		withStockHistoryServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{"chart":{"result":[{
				"timestamp":[1704067200,1704153600,1704240000],
				"indicators":{"quote":[{"close":[100.5,0,102.25]}]}
			}]}}`))
		})

		h := NewHistory()
		points, err := h.Fetch(context.Background(), "RELIANCE.NS", 30)
		require.NoError(t, err)
		require.Len(t, points, 2)
		assert.Equal(t, 100.5, points[0].Price)
		assert.Equal(t, 102.25, points[1].Price)
	})

	t.Run("truncates to the requested number of most recent days", func(t *testing.T) {
		withStockHistoryServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{"chart":{"result":[{
				"timestamp":[1704067200,1704153600,1704240000],
				"indicators":{"quote":[{"close":[1,2,3]}]}
			}]}}`))
		})

		h := NewHistory()
		points, err := h.Fetch(context.Background(), "RELIANCE.NS", 2)
		require.NoError(t, err)
		require.Len(t, points, 2)
		assert.Equal(t, 2.0, points[0].Price)
		assert.Equal(t, 3.0, points[1].Price)
	})

	t.Run("empty chart result returns error", func(t *testing.T) {
		withStockHistoryServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{"chart":{"result":[]}}`))
		})

		h := NewHistory()
		_, err := h.Fetch(context.Background(), "RELIANCE.NS", 30)
		assert.Error(t, err)
	})

	t.Run("missing indicators returns error", func(t *testing.T) {
		withStockHistoryServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{"chart":{"result":[{"timestamp":[1704067200],"indicators":{"quote":[]}}]}}`))
		})

		h := NewHistory()
		_, err := h.Fetch(context.Background(), "RELIANCE.NS", 30)
		assert.Error(t, err)
	})

	t.Run("malformed json returns error", func(t *testing.T) {
		withStockHistoryServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`not json`))
		})

		h := NewHistory()
		_, err := h.Fetch(context.Background(), "RELIANCE.NS", 30)
		assert.Error(t, err)
	})

	t.Run("invalid URL from a malformed symbol returns a request-build error", func(t *testing.T) {
		orig := StockHistoryAPIURL
		StockHistoryAPIURL = "http://example.com/%s?range=%s"
		t.Cleanup(func() { StockHistoryAPIURL = orig })

		h := NewHistory()
		_, err := h.Fetch(context.Background(), "bad\nsymbol", 30)
		assert.Error(t, err)
	})

	t.Run("unreachable server returns a transport error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		closedURL := srv.URL
		srv.Close()

		orig := StockHistoryAPIURL
		StockHistoryAPIURL = closedURL + "/%s?range=%s"
		t.Cleanup(func() { StockHistoryAPIURL = orig })

		h := NewHistory()
		_, err := h.Fetch(context.Background(), "RELIANCE.NS", 30)
		assert.Error(t, err)
	})

	t.Run("range param scales with requested days", func(t *testing.T) {
		var gotRange string
		withStockHistoryServer(t, func(w http.ResponseWriter, r *http.Request) {
			gotRange = r.URL.Query().Get("range")
			w.Write([]byte(`{"chart":{"result":[{"timestamp":[],"indicators":{"quote":[{"close":[]}]}}]}}`))
		})

		h := NewHistory()

		cases := []struct {
			days      int
			wantRange string
		}{
			{days: 10, wantRange: "1mo"},
			{days: 60, wantRange: "3mo"},
			{days: 200, wantRange: "1y"},
			{days: 400, wantRange: "5y"},
		}
		for _, c := range cases {
			_, err := h.Fetch(context.Background(), "RELIANCE.NS", c.days)
			require.NoError(t, err)
			assert.Equal(t, c.wantRange, gotRange, "days=%d", c.days)
		}
	})
}
