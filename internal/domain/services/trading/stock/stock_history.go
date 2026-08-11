package stock

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/udaypt/trading-app/config"
	"github.com/udaypt/trading-app/internal/domain/usecase/trading"
)

// Stock Historical Chart Structs
type ChartResponse struct {
	Chart struct {
		Result []struct {
			Timestamp  []int64 `json:"timestamp"`
			Indicators struct {
				Quote []struct {
					Close []float64 `json:"close"`
				} `json:"quote"`
			} `json:"indicators"`
		} `json:"result"`
	} `json:"chart"`
}
type History struct {
	// repo *postgres.DBRepository
}

func NewHistory() *History {
	return &History{}
}

func (h *History) GetAssetType() trading.AssetType {
	return trading.Stock
}

func (h *History) Fetch(ctx context.Context, symbol string, days int) ([]trading.PricePoint, error) {
	return byExternalAPI(ctx, symbol, days)
}

// StockHistoryAPIURL is the stock price-history endpoint. Exported so it
// can be redirected to an httptest server from this and other packages' tests.
var StockHistoryAPIURL = config.STOCK_HISTORY_API_URL

func byExternalAPI(ctx context.Context, symbol string, days int) ([]trading.PricePoint, error) {
	rangeParam := "1mo"
	if days > 30 && days <= 90 {
		rangeParam = "3mo"
	} else if days > 90 && days <= 365 {
		rangeParam = "1y"
	} else if days > 365 {
		rangeParam = "5y"
	}

	urlStr := fmt.Sprintf(StockHistoryAPIURL, symbol, rangeParam)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var raw ChartResponse
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}

	if len(raw.Chart.Result) == 0 {
		return nil, fmt.Errorf("no chart result for %s", symbol)
	}

	result := raw.Chart.Result[0]
	timestamps := result.Timestamp
	if len(result.Indicators.Quote) == 0 {
		return nil, fmt.Errorf("missing price indicators")
	}
	closes := result.Indicators.Quote[0].Close

	var points []trading.PricePoint
	for i, ts := range timestamps {
		if i >= len(closes) || closes[i] == 0 {
			continue // Skip market holidays
		}
		t := time.Unix(ts, 0).UTC()
		points = append(points, trading.PricePoint{
			Date:  t.Format("2006-01-02"),
			Price: closes[i],
		})
	}

	if len(points) > days {
		points = points[len(points)-days:]
	}

	return points, nil
}
