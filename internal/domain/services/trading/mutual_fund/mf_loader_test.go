package mutualfund

import (
	"context"
	"errors"
	"testing"

	"github.com/udaypt/trading-app/internal/infra/db/postgres"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMFStoreLoader_Load(t *testing.T) {
	t.Run("returns schemes cached in postgres", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		rows := sqlmock.NewRows([]string{"scheme_code", "scheme_name"}).
			AddRow(1, "Existing Fund")
		mock.ExpectQuery("SELECT scheme_code, scheme_name FROM mutual_fund_schemes").WillReturnRows(rows)

		loader := NewMFStoreLoader(postgres.NewDBRepositoryWithDB(db))
		schemes, err := loader.Load(context.Background())
		require.NoError(t, err)
		require.Len(t, schemes, 1)
		assert.Equal(t, "Existing Fund", schemes[0].SchemeName)
	})

	t.Run("errors when postgres has no schemes yet", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectQuery("SELECT scheme_code, scheme_name FROM mutual_fund_schemes").
			WillReturnRows(sqlmock.NewRows([]string{"scheme_code", "scheme_name"}))

		loader := NewMFStoreLoader(postgres.NewDBRepositoryWithDB(db))
		_, err = loader.Load(context.Background())
		assert.Error(t, err)
	})

	t.Run("propagates db error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectQuery("SELECT scheme_code, scheme_name FROM mutual_fund_schemes").
			WillReturnError(errors.New("db down"))

		loader := NewMFStoreLoader(postgres.NewDBRepositoryWithDB(db))
		_, err = loader.Load(context.Background())
		assert.Error(t, err)
	})
}
