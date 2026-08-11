package stock

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
	"trading-dashboard/config"

	"trading-dashboard/internal/domain/usecase/trading"
)

// Yahoo Stock Search Structs
type YahooQuote struct {
	Symbol    string  `json:"symbol"`
	ShortName string  `json:"shortname"`
	LongName  string  `json:"longname"`
	Exchange  string  `json:"exchDisp"`
	Score     float64 `json:"score"` // float64 handles numbers like 20009.0 safely
}

type YahooSearchResponse struct {
	Quotes []YahooQuote `json:"quotes"`
}

// StockSearchAPIURL is the stock search endpoint. Exported so it can be
// redirected to an httptest server from this and other packages' tests.
var StockSearchAPIURL = config.STOCK_SEARCH_API_URL

func SearchStocks(ctx context.Context, query string, limit int) ([]trading.SearchResult, error) {
	u, err := url.Parse(StockSearchAPIURL)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Add("q", query)
	params.Add("quotesCount", "10")
	params.Add("newsCount", "0")
	u.RawQuery = params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("yahoo search returned status %d", resp.StatusCode)
	}

	var raw YahooSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}

	var results []trading.SearchResult
	for _, q := range raw.Quotes {
		if strings.HasSuffix(q.Symbol, ".NS") || strings.HasSuffix(q.Symbol, ".BO") {
			displayName := q.LongName
			if displayName == "" {
				displayName = q.ShortName
			}

			results = append(results, trading.SearchResult{
				ID:       q.Symbol,
				Name:     displayName,
				Type:     trading.AssetType(trading.Stock),
				Symbol:   q.Symbol,
				Exchange: q.Exchange,
			})
			if limit > 0 && len(results) >= limit {
				break
			}
		}
	}
	return results, nil
}
