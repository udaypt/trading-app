package mutualfund

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
	"trading-dashboard/config"
	"trading-dashboard/internal/domain/usecase/trading"
)

// MFapi Historical NAV Structs
type MFNAVItem struct {
	Date string `json:"date"` // "DD-MM-YYYY"
	NAV  string `json:"nav"`
}

type MFHistoricalResponse struct {
	Data []MFNAVItem `json:"data"`
}

type History struct {
}

func NewHistory() *History {
	return &History{}
}

func (h *History) GetAssetType() trading.AssetType {
	return trading.MutualFund
}

func (h *History) Fetch(ctx context.Context, schemeCode string, days int) ([]trading.PricePoint, error) {
	return byExternalAPI(ctx, schemeCode, days)
}

// MFHistoryAPIURL is the mutual-fund NAV history endpoint. Exported so it
// can be redirected to an httptest server from this and other packages' tests.
var MFHistoryAPIURL = config.MF_HISTORY_API_URL

func byExternalAPI(ctx context.Context, schemeCode string, days int) ([]trading.PricePoint, error) {
	urlStr := fmt.Sprintf(MFHistoryAPIURL, schemeCode)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var raw MFHistoricalResponse
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}

	var points []trading.PricePoint
	limit := days
	if len(raw.Data) < limit {
		limit = len(raw.Data)
	}

	for i := 0; i < limit; i++ {
		item := raw.Data[i]
		navVal, err := strconv.ParseFloat(item.NAV, 64)
		if err != nil {
			continue
		}

		parsedTime, err := time.Parse("02-01-2006", item.Date)
		formattedDate := item.Date
		if err == nil {
			formattedDate = parsedTime.Format("2006-01-02")
		}

		points = append(points, trading.PricePoint{
			Date:  formattedDate,
			Price: navVal,
		})
	}

	// Reverse to chronological order (oldest -> newest) for rendering charts
	for i, j := 0, len(points)-1; i < j; i, j = i+1, j-1 {
		points[i], points[j] = points[j], points[i]
	}

	return points, nil
}
