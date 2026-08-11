package mfstore

import (
	"context"
	"fmt"
	"log"
	"time"

	mutualfund "github.com/udaypt/trading-app/internal/domain/services/trading/mutual_fund"
	"github.com/udaypt/trading-app/internal/infra/db/postgres"
)

// syncMaxAttempts is the initial attempt plus 3 progressive retries.
const syncMaxAttempts = 4

// syncRetryBaseWait is overridden in tests to keep retry-path tests fast.
var syncRetryBaseWait = 1 * time.Second

// Syncer syncs the mutual-fund master list from the external API
// into Postgres, retrying transient failures with progressive backoff.
type Syncer struct {
	provider *Provider
	repo     *postgres.DBRepository
}

func NewSyncer(provider *Provider, repo *postgres.DBRepository) *Syncer {
	return &Syncer{provider: provider, repo: repo}
}

// Sync fetches the master list from the external API (retrying on failure),
// persists it to Postgres asynchronously, and returns the fresh schemes.
func (s *Syncer) Sync(ctx context.Context) ([]mutualfund.Scheme, error) {
	log.Println("[Syncer] Fetching latest mutual fund master list from API...")

	schemes, err := s.fetchWithRetry(ctx)
	if err != nil {
		return nil, err
	}

	log.Printf("[Syncer] Downloaded %d schemes from API. Persisting to PostgreSQL...", len(schemes))

	dbRecords := convertToDBRecords(schemes)

	// Persist to PostgreSQL asynchronously so callers aren't blocked on it.
	go func() {
		if err := s.repo.BulkUpsertMFSchemes(dbRecords); err != nil {
			log.Printf("[Syncer Error] Failed to persist schemes to PostgreSQL: %v", err)
		} else {
			log.Printf("[Syncer] Successfully persisted %d schemes to PostgreSQL.", len(dbRecords))
		}
	}()

	return schemes, nil
}

// fetchWithRetry calls the provider, retrying on any failure (network
// error, non-200 status, or malformed body) with progressively increasing
// backoff between attempts: 1s, 2s, then 4s.
func (s *Syncer) fetchWithRetry(ctx context.Context) ([]mutualfund.Scheme, error) {
	var lastErr error

	for attempt := 1; attempt <= syncMaxAttempts; attempt++ {
		schemes, err := s.provider.Fetch(ctx)
		if err == nil {
			return schemes, nil
		}

		lastErr = err
		log.Printf("[Syncer] Attempt %d/%d to fetch mutual fund master list failed: %v", attempt, syncMaxAttempts, err)

		if attempt == syncMaxAttempts {
			break
		}

		wait := syncRetryBaseWait * time.Duration(1<<(attempt-1)) // 1s, 2s, 4s, ...
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(wait):
		}
	}

	return nil, fmt.Errorf("failed to fetch mutual fund master list after %d attempts: %w", syncMaxAttempts, lastErr)
}

// StartBackgroundSync runs Sync on a ticker, pushing every successful
// result to onSync, until ctx is canceled.
func (s *Syncer) StartBackgroundSync(ctx context.Context, interval time.Duration, onSync func([]mutualfund.Scheme)) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Println("[Syncer] Stopping background sync worker.")
				return
			case <-ticker.C:
				log.Println("[Syncer] Executing scheduled sync...")
				syncCtx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
				schemes, err := s.Sync(syncCtx)
				if err != nil {
					log.Printf("[Syncer Error] Scheduled sync failed: %v", err)
				} else {
					onSync(schemes)
				}
				cancel()
			}
		}
	}()
}
