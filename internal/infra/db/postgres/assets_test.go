package postgres

import (
	"database/sql"
	"errors"
	"testing"

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
