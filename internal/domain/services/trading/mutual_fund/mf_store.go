package mutualfund

import (
	"fmt"
	"strings"
	"sync"

	"github.com/udaypt/trading-app/internal/domain/usecase/trading"
)

type Scheme struct {
	SchemeCode int    `json:"schemeCode"`
	SchemeName string `json:"schemeName"`
}

// MFStore is an in-memory index of the mutual-fund master list, searchable
// by name. It holds no knowledge of where schemes come from or how they get
// refreshed — see MFStoreProvider, MFStoreSyncer, MFStoreLoader, and
// MFStoreInitializer for that.
type MFStore struct {
	mu      sync.RWMutex
	schemes []Scheme
}

func NewMFStore() *MFStore {
	return &MFStore{}
}

// SetSchemes atomically replaces the in-memory scheme list.
func (s *MFStore) SetSchemes(schemes []Scheme) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.schemes = schemes
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
