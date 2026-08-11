package httphandlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	mf "trading-dashboard/internal/domain/services/trading/mutual_fund"
	"trading-dashboard/internal/domain/services/trading/stock"
	"trading-dashboard/internal/infra/db/postgres"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSearch_Handle(t *testing.T) {
	t.Run("OPTIONS preflight returns 200", func(t *testing.T) {
		app := NewSearch(nil, nil)

		req := httptest.NewRequest(http.MethodOptions, "/api/v1/search", nil)
		rec := httptest.NewRecorder()
		app.Handle(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("missing q param returns 400", func(t *testing.T) {
		app := NewSearch(nil, nil)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/search", nil)
		rec := httptest.NewRecorder()
		app.Handle(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		var resp SearchAPIResponse
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
		assert.Equal(t, "error", resp.Status)
	})

	t.Run("blank q param returns 400", func(t *testing.T) {
		app := NewSearch(nil, nil)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/search?q=%20%20", nil)
		rec := httptest.NewRecorder()
		app.Handle(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("combines stock and mutual fund results", func(t *testing.T) {
		stockSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{"quotes":[{"symbol":"HDFCBANK.NS","shortname":"HDFC Bank","exchDisp":"NSE"}]}`))
		}))
		defer stockSrv.Close()
		origSearchURL := stock.StockSearchAPIURL
		stock.StockSearchAPIURL = stockSrv.URL
		defer func() { stock.StockSearchAPIURL = origSearchURL }()

		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()
		mock.ExpectQuery("SELECT scheme_code, scheme_name FROM mutual_fund_schemes").
			WillReturnRows(sqlmock.NewRows([]string{"scheme_code", "scheme_name"}).
				AddRow(1, "HDFC Top 100 Fund"))

		repo := postgres.NewDBRepositoryWithDB(db)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		mfStore, err := mf.NewMFStore(ctx, repo)
		require.NoError(t, err)

		app := NewSearch(mfStore, repo)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/search?q=hdfc", nil)
		rec := httptest.NewRecorder()
		app.Handle(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var resp SearchAPIResponse
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
		assert.Equal(t, "success", resp.Status)
		assert.Equal(t, 2, resp.Count)
	})

	t.Run("stock search failure still returns mutual fund results", func(t *testing.T) {
		stockSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		defer stockSrv.Close()
		origSearchURL := stock.StockSearchAPIURL
		stock.StockSearchAPIURL = stockSrv.URL
		defer func() { stock.StockSearchAPIURL = origSearchURL }()

		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()
		mock.ExpectQuery("SELECT scheme_code, scheme_name FROM mutual_fund_schemes").
			WillReturnRows(sqlmock.NewRows([]string{"scheme_code", "scheme_name"}).
				AddRow(1, "HDFC Top 100 Fund"))

		repo := postgres.NewDBRepositoryWithDB(db)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		mfStore, err := mf.NewMFStore(ctx, repo)
		require.NoError(t, err)

		app := NewSearch(mfStore, repo)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/search?q=hdfc", nil)
		rec := httptest.NewRecorder()
		app.Handle(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var resp SearchAPIResponse
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
		assert.Equal(t, "success", resp.Status)
		assert.Equal(t, 1, resp.Count) // stock search warning is logged, not surfaced
	})
}
