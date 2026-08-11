package mutualfund

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
	"trading-dashboard/config"
	"trading-dashboard/internal/domain/usecase/trading"
	"trading-dashboard/internal/infra/db/postgres"
)

type Scheme struct {
	SchemeCode int    `json:"schemeCode"`
	SchemeName string `json:"schemeName"`
}

// MUTUAL FUND In-memory store for caching the master list
type MFStore struct {
	mu      sync.RWMutex
	schemes []Scheme
	client  *http.Client
	repo    *postgres.DBRepository
}

func NewMFStore(ctx context.Context, repo *postgres.DBRepository) (*MFStore, error) {
	store := &MFStore{
		client: &http.Client{Timeout: 15 * time.Second},
		repo:   repo,
	}

	// Load existing schemes from PostgreSQL into Memory
	dbSchemes, err := repo.GetAllMFSchemes()
	if err == nil && len(dbSchemes) > 0 {
		store.schemes = convertFromDBRecords(dbSchemes)
		log.Printf("[MFStore] Loaded %d schemes from PostgreSQL into RAM cache.", len(store.schemes))
	} else {
		// If DB is empty, fetch immediately from API to seed Database & Cache
		log.Println("[MFStore] DB empty or unreadable. Seeding from API...")
		if err := store.syncAPIWithDB(ctx); err != nil {
			log.Printf("[Warn] Initial API seed failed: %v", err)
		}
	}

	// Start background ticker worker to update DB & RAM every 24 hours
	store.startBackgroundSync(ctx, 24*time.Hour)

	return store, nil
}

func (s *MFStore) Search(query string, limit int) []trading.SearchResult {
	tokens := strings.Fields(strings.ToLower(query))
	if len(tokens) == 0 {
		return nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []trading.SearchResult
	for _, scheme := range s.schemes {
		lowerName := strings.ToLower(scheme.SchemeName)
		matchesAll := true

		for _, token := range tokens {
			if !strings.Contains(lowerName, token) {
				matchesAll = false
				break
			}
		}

		if matchesAll {
			results = append(results, trading.SearchResult{
				ID:       fmt.Sprintf("%d", scheme.SchemeCode),
				Name:     scheme.SchemeName,
				Type:     trading.AssetType(trading.MutualFund),
				Symbol:   "",
				Exchange: "AMC",
			})
			if limit > 0 && len(results) >= limit {
				break
			}
		}
	}
	return results
}

// MFSyncAPIURL is the mutual-fund master-list endpoint. Exported so it can
// be redirected to an httptest server from this and other packages' tests.
var MFSyncAPIURL = config.MF_API_BASE_URL

func (s *MFStore) syncAPIWithDB(ctx context.Context) error {
	log.Println("[MFStore] Fetching latest mutual fund master list from API...")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, MFSyncAPIURL, nil)
	if err != nil {
		return err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("http fetch error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("mfapi returned status code: %d", resp.StatusCode)
	}

	var apiSchemes []Scheme
	if err := json.NewDecoder(resp.Body).Decode(&apiSchemes); err != nil {
		return fmt.Errorf("failed to decode scheme payload: %w", err)
	}

	log.Printf("[MFStore] Downloaded %d schemes from API. Persisting to PostgreSQL...", len(apiSchemes))

	// Convert domain struct to DB records
	dbRecords := convertToDBRecords(apiSchemes)

	// Persist to PostgreSQL asynchronously to avoid blocking initialization
	go func() {
		if err := s.repo.BulkUpsertMFSchemes(dbRecords); err != nil {
			log.Printf("[MFStore Error] Failed to persist schemes to PostgreSQL: %v", err)
		} else {
			log.Printf("[MFStore] Successfully persisted %d schemes to PostgreSQL.", len(dbRecords))
		}
	}()

	// Update RAM cache safely with Write Lock
	s.mu.Lock()
	s.schemes = apiSchemes
	s.mu.Unlock()

	return nil
}

func (s *MFStore) startBackgroundSync(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Println("[MFStore] Stopping background sync worker.")
				return
			case <-ticker.C:
				log.Println("[MFStore] Executing 24-hour scheduled sync...")
				syncCtx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
				if err := s.syncAPIWithDB(syncCtx); err != nil {
					log.Printf("[MFStore Error] Scheduled sync failed: %v", err)
				}
				cancel()
			}
		}
	}()
}
