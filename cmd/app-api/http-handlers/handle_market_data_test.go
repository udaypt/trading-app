package httphandlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	mf "trading-dashboard/internal/domain/services/trading/mutual_fund"
	"trading-dashboard/internal/domain/services/trading/stock"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The success path builds a PriceHistory service via NewPriceHistory, which
// selects stock.NewHistory() or mf.NewHistory() based on the asset type.
// Those hit stock.StockHistoryAPIURL / mf.MFHistoryAPIURL, both exported
// test seams, so they can be redirected to an httptest server here.

func TestMarketData_Handle(t *testing.T) {
	t.Run("OPTIONS preflight returns 200", func(t *testing.T) {
		app := NewMarketData(nil)

		req := httptest.NewRequest(http.MethodOptions, "/api/v1/market-data", nil)
		rec := httptest.NewRecorder()
		app.Handle(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("missing id returns 400", func(t *testing.T) {
		app := NewMarketData(nil)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/market-data?type=STOCK", nil)
		rec := httptest.NewRecorder()
		app.Handle(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		var resp MarketDataResponse
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
		assert.Equal(t, "error", resp.Status)
	})

	t.Run("missing type returns 400", func(t *testing.T) {
		app := NewMarketData(nil)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/market-data?id=RELIANCE.NS", nil)
		rec := httptest.NewRecorder()
		app.Handle(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("unsupported type returns 400", func(t *testing.T) {
		app := NewMarketData(nil)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/market-data?id=X&type=BOND", nil)
		rec := httptest.NewRecorder()
		app.Handle(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("non-numeric days does not panic and still validates type", func(t *testing.T) {
		app := NewMarketData(nil)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/market-data?id=X&type=BOND&days=abc", nil)
		rec := httptest.NewRecorder()
		app.Handle(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("STOCK success returns price points", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{"chart":{"result":[{
				"timestamp":[1704067200,1704153600],
				"indicators":{"quote":[{"close":[100.5,101.5]}]}
			}]}}`))
		}))
		defer srv.Close()
		origURL := stock.StockHistoryAPIURL
		stock.StockHistoryAPIURL = srv.URL + "/%s?range=%s"
		defer func() { stock.StockHistoryAPIURL = origURL }()

		repo, mock := newRepoWithMock(t)
		mock.ExpectQuery("SELECT last_nday_fetched_date").
			WithArgs("RELIANCE.NS").
			WillReturnError(errors.New("no row"))

		app := NewMarketData(repo)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/market-data?id=RELIANCE.NS&type=STOCK&days=5", nil)
		rec := httptest.NewRecorder()
		app.Handle(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var resp MarketDataResponse
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
		assert.Equal(t, "success", resp.Status)
		assert.Equal(t, 2, resp.Count)
		assert.Len(t, resp.Data, 2)
	})

	t.Run("MUTUAL_FUND success returns price points", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{"data":[{"date":"01-01-2024","nav":"11.5"}]}`))
		}))
		defer srv.Close()
		origURL := mf.MFHistoryAPIURL
		mf.MFHistoryAPIURL = srv.URL + "/%s"
		defer func() { mf.MFHistoryAPIURL = origURL }()

		repo, mock := newRepoWithMock(t)
		mock.ExpectQuery("SELECT last_nday_fetched_date").
			WithArgs("119551").
			WillReturnError(errors.New("no row"))

		app := NewMarketData(repo)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/market-data?id=119551&type=MUTUAL_FUND", nil)
		rec := httptest.NewRecorder()
		app.Handle(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var resp MarketDataResponse
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
		assert.Equal(t, "success", resp.Status)
		assert.Equal(t, 1, resp.Count)
	})

	t.Run("upstream fetch failure returns 500", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`not json`))
		}))
		defer srv.Close()
		origURL := stock.StockHistoryAPIURL
		stock.StockHistoryAPIURL = srv.URL + "/%s?range=%s"
		defer func() { stock.StockHistoryAPIURL = origURL }()

		repo, mock := newRepoWithMock(t)
		mock.ExpectQuery("SELECT last_nday_fetched_date").
			WithArgs("RELIANCE.NS").
			WillReturnError(errors.New("no row"))

		app := NewMarketData(repo)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/market-data?id=RELIANCE.NS&type=STOCK", nil)
		rec := httptest.NewRecorder()
		app.Handle(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		var resp MarketDataResponse
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
		assert.Equal(t, "error", resp.Status)
	})
}
