package stock

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"trading-dashboard/internal/domain/usecase/trading"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func withStockSearchServer(t *testing.T, handler http.HandlerFunc) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	orig := StockSearchAPIURL
	StockSearchAPIURL = srv.URL
	t.Cleanup(func() { StockSearchAPIURL = orig })
}

func TestSearchStocks(t *testing.T) {
	t.Run("keeps only NSE/BSE symbols and prefers long name", func(t *testing.T) {
		withStockSearchServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{"quotes":[
				{"symbol":"RELIANCE.NS","shortname":"RELIANCE","longname":"Reliance Industries Ltd","exchDisp":"NSE"},
				{"symbol":"AAPL","shortname":"Apple","longname":"Apple Inc"},
				{"symbol":"TATASTEEL.BO","shortname":"TATA STEEL","longname":"","exchDisp":"BSE"}
			]}`))
		})

		results, err := SearchStocks(context.Background(), "reliance", 10)
		require.NoError(t, err)
		require.Len(t, results, 2)

		assert.Equal(t, "RELIANCE.NS", results[0].ID)
		assert.Equal(t, "Reliance Industries Ltd", results[0].Name)
		assert.Equal(t, trading.AssetType(trading.Stock), results[0].Type)

		assert.Equal(t, "TATASTEEL.BO", results[1].ID)
		assert.Equal(t, "TATA STEEL", results[1].Name) // falls back to shortname
	})

	t.Run("limit truncates results", func(t *testing.T) {
		withStockSearchServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{"quotes":[
				{"symbol":"A.NS","shortname":"A"},
				{"symbol":"B.NS","shortname":"B"},
				{"symbol":"C.NS","shortname":"C"}
			]}`))
		})

		results, err := SearchStocks(context.Background(), "x", 2)
		require.NoError(t, err)
		assert.Len(t, results, 2)
	})

	t.Run("no matching exchange suffix returns empty", func(t *testing.T) {
		withStockSearchServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{"quotes":[{"symbol":"AAPL","shortname":"Apple"}]}`))
		})

		results, err := SearchStocks(context.Background(), "apple", 10)
		require.NoError(t, err)
		assert.Empty(t, results)
	})

	t.Run("non-200 status returns error", func(t *testing.T) {
		withStockSearchServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		})

		_, err := SearchStocks(context.Background(), "x", 10)
		assert.Error(t, err)
	})

	t.Run("malformed json returns error", func(t *testing.T) {
		withStockSearchServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`not json`))
		})

		_, err := SearchStocks(context.Background(), "x", 10)
		assert.Error(t, err)
	})

	t.Run("invalid base URL returns a parse error", func(t *testing.T) {
		orig := StockSearchAPIURL
		StockSearchAPIURL = "http://example.com/\n"
		t.Cleanup(func() { StockSearchAPIURL = orig })

		_, err := SearchStocks(context.Background(), "x", 10)
		assert.Error(t, err)
	})

	t.Run("unreachable server returns a transport error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		closedURL := srv.URL
		srv.Close()

		orig := StockSearchAPIURL
		StockSearchAPIURL = closedURL
		t.Cleanup(func() { StockSearchAPIURL = orig })

		_, err := SearchStocks(context.Background(), "x", 10)
		assert.Error(t, err)
	})

	t.Run("sends query as q param", func(t *testing.T) {
		var gotQuery string
		withStockSearchServer(t, func(w http.ResponseWriter, r *http.Request) {
			gotQuery = r.URL.Query().Get("q")
			w.Write([]byte(`{"quotes":[]}`))
		})

		_, err := SearchStocks(context.Background(), "hdfc bank", 10)
		require.NoError(t, err)
		assert.Equal(t, "hdfc bank", gotQuery)
	})
}
