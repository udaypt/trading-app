package trading

import (
	"context"
	"errors"
	"testing"
	"time"

	"trading-dashboard/internal/domain/usecase/trading"
	"trading-dashboard/internal/infra/db/postgres"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeAssetAPI struct {
	assetType trading.AssetType
	fetchFn   func(ctx context.Context, schemeCode string, days int) ([]trading.PricePoint, error)
	calls     int
}

func (f *fakeAssetAPI) GetAssetType() trading.AssetType {
	return f.assetType
}

func (f *fakeAssetAPI) Fetch(ctx context.Context, schemeCode string, days int) ([]trading.PricePoint, error) {
	f.calls++
	return f.fetchFn(ctx, schemeCode, days)
}

func newRepoForTest(t *testing.T) (*postgres.DBRepository, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return postgres.NewDBRepositoryWithDB(db), mock
}

func TestNewPriceHistory(t *testing.T) {
	repo, _ := newRepoForTest(t)

	t.Run("stock asset type selects stock history API", func(t *testing.T) {
		ph := NewPriceHistory(string(trading.Stock), repo)
		require.NotNil(t, ph)
		assert.Equal(t, trading.Stock, ph.assetAPI.GetAssetType())
	})

	t.Run("mutual fund asset type selects mutual fund history API", func(t *testing.T) {
		ph := NewPriceHistory(string(trading.MutualFund), repo)
		require.NotNil(t, ph)
		assert.Equal(t, trading.MutualFund, ph.assetAPI.GetAssetType())
	})

	t.Run("unknown asset type panics", func(t *testing.T) {
		assert.Panics(t, func() {
			NewPriceHistory("BOND", repo)
		})
	})
}

func TestPriceHistory_Get(t *testing.T) {
	apiPoints := []trading.PricePoint{{Date: "2024-02-01", Price: 42}}

	t.Run("serves from cache when db data is stale enough and present", func(t *testing.T) {
		repo, mock := newRepoForTest(t)
		staleDate := time.Now().AddDate(0, 0, -60).Format(time.RFC3339)
		mock.ExpectQuery("SELECT last_nday_fetched_date").
			WithArgs("RELIANCE.NS").
			WillReturnRows(sqlmock.NewRows([]string{"last_nday_fetched_date"}).AddRow(staleDate))

		cachedPoints := []trading.PricePoint{{Date: "2024-01-01", Price: 99}}
		rows := sqlmock.NewRows([]string{"price_date", "price"}).AddRow("2024-01-01T00:00:00Z", 99.0)
		mock.ExpectQuery("SELECT price_date, price").WithArgs("RELIANCE.NS", 30).WillReturnRows(rows)

		fake := &fakeAssetAPI{assetType: trading.Stock, fetchFn: func(ctx context.Context, id string, d int) ([]trading.PricePoint, error) {
			t.Fatal("assetAPI.Fetch should not be called on cache hit")
			return nil, nil
		}}
		ph := &PriceHistory{repo: repo, assetAPI: fake}

		points, err := ph.Get(context.Background(), "RELIANCE.NS", 30)
		require.NoError(t, err)
		assert.Equal(t, cachedPoints, points)
		assert.Equal(t, 0, fake.calls)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("falls back to API when GetLastNDaysDate errors", func(t *testing.T) {
		repo, mock := newRepoForTest(t)
		mock.ExpectQuery("SELECT last_nday_fetched_date").
			WithArgs("RELIANCE.NS").
			WillReturnError(errors.New("no such row"))
		mock.ExpectExec("INSERT INTO assets").WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectBegin()
		prep := mock.ExpectPrepare("INSERT INTO price_history")
		prep.ExpectExec().WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		fake := &fakeAssetAPI{assetType: trading.Stock, fetchFn: func(ctx context.Context, id string, d int) ([]trading.PricePoint, error) {
			return apiPoints, nil
		}}
		ph := &PriceHistory{repo: repo, assetAPI: fake}

		points, err := ph.Get(context.Background(), "RELIANCE.NS", 30)
		require.NoError(t, err)
		assert.Equal(t, apiPoints, points)
		assert.Equal(t, 1, fake.calls)

		assert.Eventually(t, func() bool {
			return mock.ExpectationsWereMet() == nil
		}, time.Second, 10*time.Millisecond, "expected async persistence to run")
	})

	t.Run("skips db read and calls API when cached date is not stale enough", func(t *testing.T) {
		repo, mock := newRepoForTest(t)
		recentDate := time.Now().AddDate(0, 0, -1).Format(time.RFC3339)
		mock.ExpectQuery("SELECT last_nday_fetched_date").
			WithArgs("RELIANCE.NS").
			WillReturnRows(sqlmock.NewRows([]string{"last_nday_fetched_date"}).AddRow(recentDate))
		// Registered but should remain unfulfilled: GetPriceHistory must not be called.
		mock.ExpectQuery("SELECT price_date, price").
			WithArgs("RELIANCE.NS", 30).
			WillReturnRows(sqlmock.NewRows([]string{"price_date", "price"}).AddRow("2024-01-01T00:00:00Z", 1.0))

		fake := &fakeAssetAPI{assetType: trading.Stock, fetchFn: func(ctx context.Context, id string, d int) ([]trading.PricePoint, error) {
			return apiPoints, nil
		}}
		ph := &PriceHistory{repo: repo, assetAPI: fake}

		points, err := ph.Get(context.Background(), "RELIANCE.NS", 30)
		require.NoError(t, err)
		assert.Equal(t, apiPoints, points)
		assert.Equal(t, 1, fake.calls)
		assert.Error(t, mock.ExpectationsWereMet(), "GetPriceHistory should not have been called")
	})

	t.Run("falls back to API when GetPriceHistory errors", func(t *testing.T) {
		repo, mock := newRepoForTest(t)
		staleDate := time.Now().AddDate(0, 0, -60).Format(time.RFC3339)
		mock.ExpectQuery("SELECT last_nday_fetched_date").
			WithArgs("RELIANCE.NS").
			WillReturnRows(sqlmock.NewRows([]string{"last_nday_fetched_date"}).AddRow(staleDate))
		mock.ExpectQuery("SELECT price_date, price").
			WithArgs("RELIANCE.NS", 30).
			WillReturnError(errors.New("db error"))

		fake := &fakeAssetAPI{assetType: trading.Stock, fetchFn: func(ctx context.Context, id string, d int) ([]trading.PricePoint, error) {
			return apiPoints, nil
		}}
		ph := &PriceHistory{repo: repo, assetAPI: fake}

		points, err := ph.Get(context.Background(), "RELIANCE.NS", 30)
		require.NoError(t, err)
		assert.Equal(t, apiPoints, points)
		assert.Equal(t, 1, fake.calls)
	})

	t.Run("falls back to API when cached rows are empty", func(t *testing.T) {
		repo, mock := newRepoForTest(t)
		staleDate := time.Now().AddDate(0, 0, -60).Format(time.RFC3339)
		mock.ExpectQuery("SELECT last_nday_fetched_date").
			WithArgs("RELIANCE.NS").
			WillReturnRows(sqlmock.NewRows([]string{"last_nday_fetched_date"}).AddRow(staleDate))
		mock.ExpectQuery("SELECT price_date, price").
			WithArgs("RELIANCE.NS", 30).
			WillReturnRows(sqlmock.NewRows([]string{"price_date", "price"}))

		fake := &fakeAssetAPI{assetType: trading.Stock, fetchFn: func(ctx context.Context, id string, d int) ([]trading.PricePoint, error) {
			return apiPoints, nil
		}}
		ph := &PriceHistory{repo: repo, assetAPI: fake}

		points, err := ph.Get(context.Background(), "RELIANCE.NS", 30)
		require.NoError(t, err)
		assert.Equal(t, apiPoints, points)
		assert.Equal(t, 1, fake.calls)
	})

	t.Run("propagates error when the external API fetch fails", func(t *testing.T) {
		repo, mock := newRepoForTest(t)
		mock.ExpectQuery("SELECT last_nday_fetched_date").
			WithArgs("RELIANCE.NS").
			WillReturnError(errors.New("no row"))

		fake := &fakeAssetAPI{assetType: trading.Stock, fetchFn: func(ctx context.Context, id string, d int) ([]trading.PricePoint, error) {
			return nil, errors.New("external api down")
		}}
		ph := &PriceHistory{repo: repo, assetAPI: fake}

		points, err := ph.Get(context.Background(), "RELIANCE.NS", 30)
		assert.Error(t, err)
		assert.Nil(t, points)
	})

	t.Run("saves mutual fund points under the AMC exchange", func(t *testing.T) {
		repo, mock := newRepoForTest(t)
		mock.ExpectQuery("SELECT last_nday_fetched_date").
			WithArgs("119551").
			WillReturnError(errors.New("no row"))
		mock.ExpectExec("INSERT INTO assets").
			WithArgs("119551", "119551", string(trading.MutualFund), "AMC", sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectBegin()
		prep := mock.ExpectPrepare("INSERT INTO price_history")
		prep.ExpectExec().WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		fake := &fakeAssetAPI{assetType: trading.MutualFund, fetchFn: func(ctx context.Context, id string, d int) ([]trading.PricePoint, error) {
			return apiPoints, nil
		}}
		ph := &PriceHistory{repo: repo, assetAPI: fake}

		points, err := ph.Get(context.Background(), "119551", 30)
		require.NoError(t, err)
		assert.Equal(t, apiPoints, points)

		assert.Eventually(t, func() bool {
			return mock.ExpectationsWereMet() == nil
		}, time.Second, 10*time.Millisecond, "expected async persistence with the AMC exchange to run")
	})
}
