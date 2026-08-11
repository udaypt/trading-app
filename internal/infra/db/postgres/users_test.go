package postgres

import (
	"database/sql"
	"errors"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDBRepository_CreateNewUser(t *testing.T) {
	t.Run("returns generated id", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		mock.ExpectQuery("INSERT INTO users").
			WithArgs("user@example.com", "hashed-pwd").
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(42)))

		id, err := repo.CreateNewUser("user@example.com", "hashed-pwd")
		require.NoError(t, err)
		assert.Equal(t, int64(42), id)
	})

	t.Run("duplicate email returns error and zero id", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		mock.ExpectQuery("INSERT INTO users").
			WillReturnError(errors.New("duplicate key value violates unique constraint"))

		id, err := repo.CreateNewUser("dupe@example.com", "hashed-pwd")
		assert.Error(t, err)
		assert.Equal(t, int64(0), id)
	})
}

func TestDBRepository_GetCredential(t *testing.T) {
	t.Run("returns id and hash for known email", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		mock.ExpectQuery("SELECT id, password_hash FROM users").
			WithArgs("user@example.com").
			WillReturnRows(sqlmock.NewRows([]string{"id", "password_hash"}).AddRow(int64(7), "hashed-pwd"))

		id, hash, err := repo.GetCredential("user@example.com")
		require.NoError(t, err)
		assert.Equal(t, int64(7), id)
		assert.Equal(t, "hashed-pwd", hash)
	})

	t.Run("unknown email returns error", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		mock.ExpectQuery("SELECT id, password_hash FROM users").
			WithArgs("missing@example.com").
			WillReturnError(sql.ErrNoRows)

		id, hash, err := repo.GetCredential("missing@example.com")
		assert.ErrorIs(t, err, sql.ErrNoRows)
		assert.Equal(t, int64(0), id)
		assert.Empty(t, hash)
	})
}
