package trading

import "context"

type AssetType string

const (
	Stock      AssetType = "STOCK"
	MutualFund AssetType = "MUTUAL_FUND"
)

var AssetTypes = []string{
	string(Stock), string(MutualFund),
}

// PricePoint represents one day in a time-series chart
type PricePoint struct {
	Date  string  `json:"date"`  // YYYY-MM-DD
	Price float64 `json:"price"` // Closing Stock Price or Mutual Fund NAV
}

type SearchResult struct {
	ID       string    `json:"id"`       // Symbol or SchemeCode
	Name     string    `json:"name"`     // Company name or Fund name
	Type     AssetType `json:"type"`     // STOCK or MUTUAL_FUND
	Symbol   string    `json:"symbol"`   // e.g., "RELIANCE.NS"
	Exchange string    `json:"exchange"` // NSE, BSE, or AMC
}

type StockSearchProvider interface {
	SearchStocks(ctx context.Context, query string) ([]SearchResult, error)
}

type MFSearchProvider interface {
	SearchMutualFunds(ctx context.Context, query string) ([]SearchResult, error)
}
