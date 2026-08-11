package postgres

import (
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDBRepositoryWithDB(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewDBRepositoryWithDB(db)
	require.NotNil(t, repo)
	assert.Same(t, db, repo.db)

	mock.ExpectQuery("SELECT last_nday_fetched_date").
		WithArgs("X").
		WillReturnRows(sqlmock.NewRows([]string{"last_nday_fetched_date"}).AddRow("2024-01-01"))

	date, err := repo.GetLastNDaysDate("X")
	require.NoError(t, err)
	assert.Equal(t, "2024-01-01", date)
}

func TestNewDBRepository_PingFailure(t *testing.T) {
	orig := dbConnectionString
	// Port 1 is reserved and nothing listens there; connect_timeout keeps
	// this fast instead of waiting on the OS default TCP timeout.
	dbConnectionString = "postgres://user:pass@127.0.0.1:1/nosuchdb?sslmode=disable&connect_timeout=1"
	t.Cleanup(func() { dbConnectionString = orig })

	repo, err := NewDBRepository()
	assert.Error(t, err)
	assert.Nil(t, repo)
}
