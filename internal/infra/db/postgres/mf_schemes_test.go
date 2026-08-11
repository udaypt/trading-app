package postgres

import (
	"errors"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDBRepository_GetAllMFSchemes(t *testing.T) {
	t.Run("returns mapped rows", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		rows := sqlmock.NewRows([]string{"scheme_code", "scheme_name"}).
			AddRow(1, "Fund A").
			AddRow(2, "Fund B")
		mock.ExpectQuery("SELECT scheme_code, scheme_name FROM mutual_fund_schemes").WillReturnRows(rows)

		schemes, err := repo.GetAllMFSchemes()
		require.NoError(t, err)
		require.Len(t, schemes, 2)
		assert.Equal(t, SchemeRecord{SchemeCode: 1, SchemeName: "Fund A"}, schemes[0])
	})

	t.Run("propagates query error", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		mock.ExpectQuery("SELECT scheme_code, scheme_name FROM mutual_fund_schemes").WillReturnError(errors.New("db down"))

		_, err := repo.GetAllMFSchemes()
		assert.Error(t, err)
	})

	t.Run("empty table returns nil slice, no error", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		rows := sqlmock.NewRows([]string{"scheme_code", "scheme_name"})
		mock.ExpectQuery("SELECT scheme_code, scheme_name FROM mutual_fund_schemes").WillReturnRows(rows)

		schemes, err := repo.GetAllMFSchemes()
		require.NoError(t, err)
		assert.Empty(t, schemes)
	})

	t.Run("propagates row scan error", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		rows := sqlmock.NewRows([]string{"scheme_code", "scheme_name"}).
			AddRow("not-an-int", "Fund A")
		mock.ExpectQuery("SELECT scheme_code, scheme_name FROM mutual_fund_schemes").WillReturnRows(rows)

		_, err := repo.GetAllMFSchemes()
		assert.Error(t, err)
	})
}

func TestDBRepository_BulkUpsertMFSchemes(t *testing.T) {
	t.Run("no-op on empty input", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		err := repo.BulkUpsertMFSchemes(nil)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("commits after upserting each record", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		mock.ExpectBegin()
		prep := mock.ExpectPrepare("INSERT INTO mutual_fund_schemes")
		prep.ExpectExec().WithArgs(1, "Fund A").WillReturnResult(sqlmock.NewResult(1, 1))
		prep.ExpectExec().WithArgs(2, "Fund B").WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		err := repo.BulkUpsertMFSchemes([]SchemeRecord{
			{SchemeCode: 1, SchemeName: "Fund A"},
			{SchemeCode: 2, SchemeName: "Fund B"},
		})
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("rolls back on exec error", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		mock.ExpectBegin()
		prep := mock.ExpectPrepare("INSERT INTO mutual_fund_schemes")
		prep.ExpectExec().WithArgs(1, "Fund A").WillReturnError(errors.New("constraint violation"))
		mock.ExpectRollback()

		err := repo.BulkUpsertMFSchemes([]SchemeRecord{{SchemeCode: 1, SchemeName: "Fund A"}})
		assert.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates transaction begin error", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		mock.ExpectBegin().WillReturnError(errors.New("connection lost"))

		err := repo.BulkUpsertMFSchemes([]SchemeRecord{{SchemeCode: 1, SchemeName: "Fund A"}})
		assert.Error(t, err)
	})

	t.Run("propagates statement prepare error", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		mock.ExpectBegin()
		mock.ExpectPrepare("INSERT INTO mutual_fund_schemes").WillReturnError(errors.New("syntax error"))
		mock.ExpectRollback()

		err := repo.BulkUpsertMFSchemes([]SchemeRecord{{SchemeCode: 1, SchemeName: "Fund A"}})
		assert.Error(t, err)
	})
}
