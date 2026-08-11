package mutualfund

import (
	"context"
	"fmt"

	"github.com/udaypt/trading-app/internal/infra/db/postgres"
)

// MFStoreLoader reads the mutual-fund master list already cached in
// Postgres. It never talks to the external API — that's MFStoreSyncer's job.
type MFStoreLoader struct {
	repo *postgres.DBRepository
}

func NewMFStoreLoader(repo *postgres.DBRepository) *MFStoreLoader {
	return &MFStoreLoader{repo: repo}
}

// Load returns the schemes cached in Postgres, erroring if the DB read
// fails or no schemes are stored yet.
func (l *MFStoreLoader) Load(ctx context.Context) ([]Scheme, error) {
	dbSchemes, err := l.repo.GetAllMFSchemes()
	if err != nil {
		return nil, fmt.Errorf("failed to load mutual fund schemes from postgres: %w", err)
	}
	if len(dbSchemes) == 0 {
		return nil, fmt.Errorf("no mutual fund schemes found in postgres")
	}

	return convertFromDBRecords(dbSchemes), nil
}
