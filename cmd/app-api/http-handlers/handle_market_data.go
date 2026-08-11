package httphandlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"

	tracing_svc "trading-dashboard/internal/domain/services/trading"
	"trading-dashboard/internal/domain/usecase/trading"
	"trading-dashboard/internal/httphandler"
	"trading-dashboard/internal/infra/db/postgres"
	"trading-dashboard/internal/security"
)

// MarketDataResponse is the JSON wrapper for GET /api/v1/market-data
type MarketDataResponse struct {
	Status string               `json:"status"`
	ID     string               `json:"id"`
	Type   string               `json:"type"`
	Days   int                  `json:"days"`
	Count  int                  `json:"count"`
	Data   []trading.PricePoint `json:"data"`
	Error  string               `json:"error,omitempty"`
}

type MarketData struct {
	repo *postgres.DBRepository
}

func NewMarketData(repo *postgres.DBRepository) *MarketData {
	return &MarketData{
		repo: repo,
	}
}

// func HandleHttp(writer http.ResponseWriter, request *http.Request, handle http.HandlerFunc) {
func (app *MarketData) Handle(w http.ResponseWriter, r *http.Request) {
	httphandler.HandleHttp(w, r, app.handleMarketData)
}

func (app *MarketData) handleMarketData(w http.ResponseWriter, r *http.Request) {
	security.EnableCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	id := strings.TrimSpace(r.URL.Query().Get("id"))
	assetType := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("type")))
	daysStr := r.URL.Query().Get("days")

	days := 30
	if daysStr != "" {
		if parsed, err := strconv.Atoi(daysStr); err == nil && parsed > 0 {
			days = parsed
		}
	}

	if id == "" || assetType == "" || !slices.Contains(trading.AssetTypes, assetType) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(MarketDataResponse{
			Status: "error",
			Error:  "Query parameters 'id' and 'type' are required",
		})
		return
	}

	history := tracing_svc.NewPriceHistory(assetType, app.repo)

	points, err := history.Get(r.Context(), id, days)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(MarketDataResponse{
			Status: "error",
			Error:  fmt.Sprintf("Failed to fetch historical data: %v", err),
		})
		return
	}

	json.NewEncoder(w).Encode(MarketDataResponse{
		Status: "success",
		ID:     id,
		Type:   assetType,
		Days:   days,
		Count:  len(points),
		Data:   points,
	})
}
