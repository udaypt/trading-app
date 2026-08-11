package mutualfund

import (
	"testing"

	"github.com/udaypt/trading-app/internal/domain/usecase/trading"

	"github.com/stretchr/testify/assert"
)

func newTestStore(schemes []Scheme) *MFStore {
	store := NewMFStore()
	store.SetSchemes(schemes)
	return store
}

func TestMFStore_Search(t *testing.T) {
	schemes := []Scheme{
		{SchemeCode: 1, SchemeName: "HDFC Top 100 Fund"},
		{SchemeCode: 2, SchemeName: "SBI Blue Chip Fund"},
		{SchemeCode: 3, SchemeName: "HDFC Small Cap Fund"},
	}

	tests := []struct {
		name      string
		query     string
		limit     int
		wantCount int
		wantIDs   []string
	}{
		{
			name:      "empty query returns nil",
			query:     "   ",
			limit:     10,
			wantCount: 0,
		},
		{
			name:      "single token matches all with substring",
			query:     "hdfc",
			limit:     10,
			wantCount: 2,
			wantIDs:   []string{"1", "3"},
		},
		{
			name:      "case insensitive match",
			query:     "SBI",
			limit:     10,
			wantCount: 1,
			wantIDs:   []string{"2"},
		},
		{
			name:      "multi-token requires all tokens present",
			query:     "hdfc small",
			limit:     10,
			wantCount: 1,
			wantIDs:   []string{"3"},
		},
		{
			name:      "no match returns empty",
			query:     "nonexistent",
			limit:     10,
			wantCount: 0,
		},
		{
			name:      "limit truncates results",
			query:     "fund",
			limit:     1,
			wantCount: 1,
		},
		{
			name:      "zero limit means unlimited",
			query:     "fund",
			limit:     0,
			wantCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newTestStore(schemes)
			results := store.Search(tt.query, tt.limit)
			assert.Len(t, results, tt.wantCount)
			if tt.wantIDs != nil {
				gotIDs := make([]string, 0, len(results))
				for _, r := range results {
					gotIDs = append(gotIDs, r.ID)
					assert.Equal(t, trading.AssetType(trading.MutualFund), r.Type)
					assert.Equal(t, "AMC", r.Exchange)
				}
				assert.ElementsMatch(t, tt.wantIDs, gotIDs)
			}
		})
	}
}

func TestMFStore_Search_EmptyStore(t *testing.T) {
	store := newTestStore(nil)
	results := store.Search("anything", 10)
	assert.Empty(t, results)
}

func TestMFStore_SetSchemes(t *testing.T) {
	store := NewMFStore()
	assert.Empty(t, store.Search("anything", 10))

	store.SetSchemes([]Scheme{{SchemeCode: 5, SchemeName: "Newly Loaded Fund"}})
	results := store.Search("newly", 10)
	assert.Len(t, results, 1)
	assert.Equal(t, "5", results[0].ID)

	store.SetSchemes([]Scheme{{SchemeCode: 6, SchemeName: "Replacement Fund"}})
	assert.Empty(t, store.Search("newly", 10))
	assert.Len(t, store.Search("replacement", 10), 1)
}
