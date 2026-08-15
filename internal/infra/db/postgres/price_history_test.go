package postgres

import (
	"errors"
	"testing"
	"time"

	"github.com/udaypt/trading-app/internal/domain/usecase/trading"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDBRepository_BulkUpsertPriceHistory(t *testing.T) {
	t.Run("no-op on empty input", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		err := repo.BulkUpsertPriceHistory("RELIANCE.NS", nil)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("upserts each point and commits", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		mock.ExpectBegin()
		prep := mock.ExpectPrepare("INSERT INTO price_history")
		prep.ExpectExec().WithArgs("RELIANCE.NS", "2024-01-01", 100.5).WillReturnResult(sqlmock.NewResult(1, 1))
		prep.ExpectExec().WithArgs("RELIANCE.NS", "2024-01-02", 101.0).WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		err := repo.BulkUpsertPriceHistory("RELIANCE.NS", []trading.PricePoint{
			{Date: "2024-01-01", Price: 100.5},
			{Date: "2024-01-02", Price: 101.0},
		})
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("rolls back on exec error", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		mock.ExpectBegin()
		prep := mock.ExpectPrepare("INSERT INTO price_history")
		prep.ExpectExec().WithArgs("RELIANCE.NS", "2024-01-01", 100.5).WillReturnError(errors.New("boom"))
		mock.ExpectRollback()

		err := repo.BulkUpsertPriceHistory("RELIANCE.NS", []trading.PricePoint{{Date: "2024-01-01", Price: 100.5}})
		assert.Error(t, err)
	})

	t.Run("propagates transaction begin error", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		mock.ExpectBegin().WillReturnError(errors.New("connection lost"))

		err := repo.BulkUpsertPriceHistory("RELIANCE.NS", []trading.PricePoint{{Date: "2024-01-01", Price: 100.5}})
		assert.Error(t, err)
	})

	t.Run("propagates statement prepare error", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		mock.ExpectBegin()
		mock.ExpectPrepare("INSERT INTO price_history").WillReturnError(errors.New("syntax error"))
		mock.ExpectRollback()

		err := repo.BulkUpsertPriceHistory("RELIANCE.NS", []trading.PricePoint{{Date: "2024-01-01", Price: 100.5}})
		assert.Error(t, err)
	})
}

func TestDBRepository_GetPriceHistory(t *testing.T) {
	t.Run("truncates timestamps and reverses to chronological order", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		rows := sqlmock.NewRows([]string{"price_date", "price"}).
			AddRow("2024-01-03T00:00:00Z", 103.0).
			AddRow("2024-01-02T00:00:00Z", 102.0).
			AddRow("2024-01-01T00:00:00Z", 101.0)
		mock.ExpectQuery("SELECT price_date, price").WithArgs("RELIANCE.NS", 3).WillReturnRows(rows)

		points, err := repo.GetPriceHistory("RELIANCE.NS", 3)
		require.NoError(t, err)
		require.Len(t, points, 3)

		assert.Equal(t, "2024-01-01", points[0].Date)
		assert.Equal(t, 101.0, points[0].Price)
		assert.Equal(t, "2024-01-03", points[2].Date)
	})

	t.Run("propagates query error", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		mock.ExpectQuery("SELECT price_date, price").WillReturnError(errors.New("db down"))

		_, err := repo.GetPriceHistory("RELIANCE.NS", 3)
		assert.Error(t, err)
	})

	t.Run("no rows returns empty slice", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		rows := sqlmock.NewRows([]string{"price_date", "price"})
		mock.ExpectQuery("SELECT price_date, price").WillReturnRows(rows)

		points, err := repo.GetPriceHistory("RELIANCE.NS", 3)
		require.NoError(t, err)
		assert.Empty(t, points)
	})

	t.Run("propagates row scan error", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		rows := sqlmock.NewRows([]string{"price_date", "price"}).
			AddRow("2024-01-01T00:00:00Z", "not-a-float")
		mock.ExpectQuery("SELECT price_date, price").WillReturnRows(rows)

		_, err := repo.GetPriceHistory("RELIANCE.NS", 3)
		assert.Error(t, err)
	})
}

func TestDBRepository_GetPriceHistoryUpdatedAt(t *testing.T) {
	t.Run("returns the most recent update timestamp", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		want := time.Date(2024, 1, 3, 12, 0, 0, 0, time.UTC)
		mock.ExpectQuery("SELECT MAX\\(updated_at\\)").
			WithArgs("RELIANCE.NS").
			WillReturnRows(sqlmock.NewRows([]string{"max"}).AddRow(want))

		got, err := repo.GetPriceHistoryUpdatedAt("RELIANCE.NS")
		require.NoError(t, err)
		assert.True(t, want.Equal(got))
	})

	t.Run("propagates query error", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		mock.ExpectQuery("SELECT MAX\\(updated_at\\)").
			WithArgs("RELIANCE.NS").
			WillReturnError(errors.New("db down"))

		_, err := repo.GetPriceHistoryUpdatedAt("RELIANCE.NS")
		assert.Error(t, err)
	})

	t.Run("errors when the asset has no cached rows", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		mock.ExpectQuery("SELECT MAX\\(updated_at\\)").
			WithArgs("UNKNOWN").
			WillReturnRows(sqlmock.NewRows([]string{"max"}).AddRow(nil))

		_, err := repo.GetPriceHistoryUpdatedAt("UNKNOWN")
		assert.Error(t, err)
	})
}
