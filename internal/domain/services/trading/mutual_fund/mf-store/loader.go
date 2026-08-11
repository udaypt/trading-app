package mfstore

import (
	"context"
	"fmt"

	mutualfund "github.com/udaypt/trading-app/internal/domain/services/trading/mutual_fund"
	"github.com/udaypt/trading-app/internal/infra/db/postgres"
)

// Loader reads the mutual-fund master list already cached in
// Postgres. It never talks to the external API — that's Syncer's job.
type Loader struct {
	repo *postgres.DBRepository
}

func NewLoader(repo *postgres.DBRepository) *Loader {
	return &Loader{repo: repo}
}

// Load returns the schemes cached in Postgres, erroring if the DB read
// fails or no schemes are stored yet.
func (l *Loader) Load(ctx context.Context) ([]mutualfund.Scheme, error) {
	dbSchemes, err := l.repo.GetAllMFSchemes()
	if err != nil {
		return nil, fmt.Errorf("failed to load mutual fund schemes from postgres: %w", err)
	}
	if len(dbSchemes) == 0 {
		return nil, fmt.Errorf("no mutual fund schemes found in postgres")
	}

	return convertFromDBRecords(dbSchemes), nil
}
