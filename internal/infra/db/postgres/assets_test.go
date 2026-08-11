package postgres

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/udaypt/trading-app/internal/domain/usecase/trading"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newMockRepo(t *testing.T) (*DBRepository, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return NewDBRepositoryWithDB(db), mock
}

func TestDBRepository_UpsertAsset(t *testing.T) {
	t.Run("executes upsert with given args", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		mock.ExpectExec("INSERT INTO assets").
			WithArgs("RELIANCE.NS", "RELIANCE.NS", "STOCK", "NSE", "2024-01-01").
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.UpsertAsset("RELIANCE.NS", "RELIANCE.NS", "STOCK", "NSE", "2024-01-01")
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates db error", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		mock.ExpectExec("INSERT INTO assets").WillReturnError(errors.New("boom"))

		err := repo.UpsertAsset("id", "name", "STOCK", "NSE", "2024-01-01")
		assert.Error(t, err)
	})
}

func TestDBRepository_GetLastNDaysDate(t *testing.T) {
	t.Run("returns date on match", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		mock.ExpectQuery("SELECT last_nday_fetched_date").
			WithArgs("RELIANCE.NS").
			WillReturnRows(sqlmock.NewRows([]string{"last_nday_fetched_date"}).AddRow("2024-01-01T00:00:00Z"))

		date, err := repo.GetLastNDaysDate("RELIANCE.NS")
		require.NoError(t, err)
		assert.Equal(t, "2024-01-01T00:00:00Z", date)
	})

	t.Run("returns error when no rows", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		mock.ExpectQuery("SELECT last_nday_fetched_date").
			WithArgs("UNKNOWN").
			WillReturnError(sql.ErrNoRows)

		_, err := repo.GetLastNDaysDate("UNKNOWN")
		assert.ErrorIs(t, err, sql.ErrNoRows)
	})
}

func TestDBRepository_SearchAssets(t *testing.T) {
	t.Run("returns matches scoped to asset type and query", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		rows := sqlmock.NewRows([]string{"id", "name", "asset_type", "exchange"}).
			AddRow("RELIANCE.NS", "Reliance Industries Ltd", "STOCK", "NSE")
		mock.ExpectQuery("SELECT id, name, asset_type, exchange").
			WithArgs("STOCK", "%reliance%", 5).
			WillReturnRows(rows)

		results, err := repo.SearchAssets("STOCK", "reliance", 5)
		require.NoError(t, err)
		require.Len(t, results, 1)
		assert.Equal(t, "RELIANCE.NS", results[0].ID)
		assert.Equal(t, "RELIANCE.NS", results[0].Symbol)
		assert.Equal(t, "Reliance Industries Ltd", results[0].Name)
		assert.Equal(t, trading.AssetType("STOCK"), results[0].Type)
		assert.Equal(t, "NSE", results[0].Exchange)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns empty slice on no match", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		mock.ExpectQuery("SELECT id, name, asset_type, exchange").
			WithArgs("STOCK", "%unknown%", 5).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "asset_type", "exchange"}))

		results, err := repo.SearchAssets("STOCK", "unknown", 5)
		require.NoError(t, err)
		assert.Empty(t, results)
	})

	t.Run("propagates db error", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		mock.ExpectQuery("SELECT id, name, asset_type, exchange").
			WillReturnError(errors.New("boom"))

		_, err := repo.SearchAssets("STOCK", "reliance", 5)
		assert.Error(t, err)
	})

	t.Run("propagates row scan error", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		rows := sqlmock.NewRows([]string{"id", "name", "asset_type", "exchange"}).
			AddRow("RELIANCE.NS", "Reliance Industries Ltd", "STOCK", "NSE").
			RowError(0, errors.New("scan boom"))
		mock.ExpectQuery("SELECT id, name, asset_type, exchange").
			WillReturnRows(rows)

		_, err := repo.SearchAssets("STOCK", "reliance", 5)
		assert.Error(t, err)
	})
}

func TestDBRepository_UpsertStocksMetadata(t *testing.T) {
	t.Run("executes upsert without touching last_nday_fetched_date", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		mock.ExpectExec("INSERT INTO assets").
			WithArgs("RELIANCE.NS", "Reliance Industries Ltd", "STOCK", "NSE").
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.UpsertStocksMetadata("RELIANCE.NS", "Reliance Industries Ltd", "NSE")
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates db error", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		mock.ExpectExec("INSERT INTO assets").WillReturnError(errors.New("boom"))

		err := repo.UpsertStocksMetadata("id", "name", "NSE")
		assert.Error(t, err)
	})
}
