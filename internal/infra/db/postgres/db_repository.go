package postgres

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/udaypt/trading-app/config"

	_ "github.com/lib/pq"
)

type DBRepository struct {
	db *sql.DB
}

// dbConnectionString defaults to the local dev connection string but can be
// overridden by the DB_CONNECTION_STRING env var, which docker-compose sets
// to point the backend container at the "db" service instead of localhost.
// Also overridden directly in tests to exercise the connect/ping error
// branches without a real Postgres instance.
var dbConnectionString = envOrDefault("DB_CONNECTION_STRING", config.DB_CONNECTION_STRING)

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func NewDBRepository() (*DBRepository, error) {
	// sql.Open only validates the driver name, never the DSN — lib/pq
	// connects lazily on first use — so this branch is unreachable with the
	// "postgres" driver registered via the blank import above.
	db, err := sql.Open("postgres", dbConnectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to db: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping db: %w", err)
	}

	return &DBRepository{db: db}, nil
}

// NewDBRepositoryWithDB builds a DBRepository around an already-open *sql.DB.
// Useful for wiring an alternate connection pool, and for tests that inject
// a sqlmock-backed *sql.DB.
func NewDBRepositoryWithDB(db *sql.DB) *DBRepository {
	return &DBRepository{db: db}
}
