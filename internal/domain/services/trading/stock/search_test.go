package stock

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/udaypt/trading-app/internal/infra/db/postgres"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newSearchRepoForTest(t *testing.T) (*postgres.DBRepository, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return postgres.NewDBRepositoryWithDB(db), mock
}

func TestSearch_Search(t *testing.T) {
	t.Run("serves from db cache without calling the external API", func(t *testing.T) {
		repo, mock := newSearchRepoForTest(t)
		rows := sqlmock.NewRows([]string{"id", "name", "asset_type", "exchange"}).
			AddRow("RELIANCE.NS", "Reliance Industries Ltd", "STOCK", "NSE")
		mock.ExpectQuery("SELECT id, name, asset_type, exchange").
			WithArgs("STOCK", "%reliance%", 5).
			WillReturnRows(rows)

		called := false
		withStockSearchServer(t, func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.Write([]byte(`{"quotes":[]}`))
		})

		s := NewSearch(repo)
		results, err := s.Search(context.Background(), "reliance", 5)
		require.NoError(t, err)
		require.Len(t, results, 1)
		assert.Equal(t, "RELIANCE.NS", results[0].ID)
		assert.False(t, called, "external API should not be called on cache hit")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("falls back to the API and caches results when db has no match", func(t *testing.T) {
		repo, mock := newSearchRepoForTest(t)
		mock.ExpectQuery("SELECT id, name, asset_type, exchange").
			WithArgs("STOCK", "%reliance%", 5).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "asset_type", "exchange"}))
		mock.ExpectExec("INSERT INTO assets").
			WithArgs("RELIANCE.NS", "Reliance Industries Ltd", "STOCK", "NSE").
			WillReturnResult(sqlmock.NewResult(0, 1))

		withStockSearchServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{"quotes":[
				{"symbol":"RELIANCE.NS","shortname":"RELIANCE","longname":"Reliance Industries Ltd","exchDisp":"NSE"}
			]}`))
		})

		s := NewSearch(repo)
		results, err := s.Search(context.Background(), "reliance", 5)
		require.NoError(t, err)
		require.Len(t, results, 1)
		assert.Equal(t, "RELIANCE.NS", results[0].ID)

		assert.Eventually(t, func() bool {
			return mock.ExpectationsWereMet() == nil
		}, time.Second, 10*time.Millisecond, "expected async metadata caching to run")
	})

	t.Run("falls back to the API when db search errors", func(t *testing.T) {
		repo, mock := newSearchRepoForTest(t)
		mock.ExpectQuery("SELECT id, name, asset_type, exchange").
			WillReturnError(errors.New("db down"))
		mock.ExpectExec("INSERT INTO assets").
			WillReturnResult(sqlmock.NewResult(0, 1))

		withStockSearchServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{"quotes":[
				{"symbol":"RELIANCE.NS","shortname":"RELIANCE","longname":"Reliance Industries Ltd","exchDisp":"NSE"}
			]}`))
		})

		s := NewSearch(repo)
		results, err := s.Search(context.Background(), "reliance", 5)
		require.NoError(t, err)
		require.Len(t, results, 1)
	})

	t.Run("propagates error when external API fails on cache miss", func(t *testing.T) {
		repo, mock := newSearchRepoForTest(t)
		mock.ExpectQuery("SELECT id, name, asset_type, exchange").
			WithArgs("STOCK", "%reliance%", 5).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "asset_type", "exchange"}))

		withStockSearchServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		})

		s := NewSearch(repo)
		results, err := s.Search(context.Background(), "reliance", 5)
		assert.Error(t, err)
		assert.Nil(t, results)
	})

	t.Run("does not persist when the API returns no results", func(t *testing.T) {
		repo, mock := newSearchRepoForTest(t)
		mock.ExpectQuery("SELECT id, name, asset_type, exchange").
			WithArgs("STOCK", "%unknown%", 5).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "asset_type", "exchange"}))

		withStockSearchServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{"quotes":[]}`))
		})

		s := NewSearch(repo)
		results, err := s.Search(context.Background(), "unknown", 5)
		require.NoError(t, err)
		assert.Empty(t, results)

		time.Sleep(20 * time.Millisecond)
		assert.NoError(t, mock.ExpectationsWereMet(), "no upsert should have been attempted")
	})
}
