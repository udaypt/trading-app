package stock

import (
	"context"
	"log"

	"github.com/udaypt/trading-app/internal/domain/usecase/trading"
	"github.com/udaypt/trading-app/internal/infra/db/postgres"
)

type Search struct {
	repo *postgres.DBRepository
}

func NewSearch(repo *postgres.DBRepository) *Search {
	return &Search{repo: repo}
}

// Search serves cached stock matches from Postgres when available, and
// otherwise falls back to the external search API, caching the fresh
// results for next time.
func (s *Search) Search(ctx context.Context, query string, limit int) ([]trading.SearchResult, error) {
	cached, err := s.repo.SearchAssets(string(trading.Stock), query, limit)
	if err != nil {
		log.Println("[Cache Miss] Could not search assets from postgres", err.Error())
	} else if len(cached) > 0 {
		log.Println("[Cached Data] Serving cached stock search results")
		return cached, nil
	}

	results, err := SearchStocks(ctx, query, limit)
	if err != nil {
		return nil, err
	}

	// Save fresh metadata to postgres via separate goroutine
	go func() {
		for _, r := range results {
			if err := s.repo.UpsertStocksMetadata(r.ID, r.Name, r.Exchange); err != nil {
				log.Printf("[Warn] failed to save stock metadata in postgres: %s", err.Error())
			}
		}
	}()

	return results, nil
}
