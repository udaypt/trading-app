package trading

import (
	"context"
	"fmt"
	"log"
	"time"

	mf "github.com/udaypt/trading-app/internal/domain/services/trading/mutual_fund"
	"github.com/udaypt/trading-app/internal/domain/services/trading/stock"
	"github.com/udaypt/trading-app/internal/domain/usecase/trading"
	"github.com/udaypt/trading-app/internal/infra/db/postgres"
)

type PriceHistory struct {
	repo     *postgres.DBRepository
	assetAPI trading.PricingHistoryAPI
}

func NewPriceHistory(assetType string, repo *postgres.DBRepository) *PriceHistory {
	var assetAPI trading.PricingHistoryAPI
	if assetType == string(trading.AssetType(trading.Stock)) {
		assetAPI = stock.NewHistory()
	} else if assetType == string(trading.AssetType(trading.MutualFund)) {
		assetAPI = mf.NewHistory()
	} else {
		panic(fmt.Sprintf("Invalid asset Type %s", assetType))
	}

	return &PriceHistory{
		repo:     repo,
		assetAPI: assetAPI,
	}
}

func (ph *PriceHistory) Get(ctx context.Context, schemeCode string, days int) ([]trading.PricePoint, error) {
	// Try from DB first
	lastNDayDate, err := ph.repo.GetLastNDaysDate(schemeCode)
	if err != nil {
		log.Println("[Cache Miss] Could not fetch last ndays date from postgres", err.Error())

		return ph.fetchFromAPI(ctx, schemeCode, days)
	}

	oldTime, _ := time.Parse(time.RFC3339, lastNDayDate)

	nDaysDate := time.Now().AddDate(0, 0, -days)

	if lastNDayDate != "" && (oldTime.Before(nDaysDate) || oldTime.Equal(nDaysDate)) {
		dbPoints, err := ph.repo.GetPriceHistory(schemeCode, days)
		if err != nil {
			log.Println("[DB Error] Could not fetch chached data from postgres")

			return ph.fetchFromAPI(ctx, schemeCode, days)
		}

		if len(dbPoints) > 0 {
			log.Println("[Cached Data] Serving cached entries")
			return dbPoints, nil
		}
	}

	return ph.fetchFromAPI(ctx, schemeCode, days)
}

// fetch asset pricing details from an external api
func (ph *PriceHistory) fetchFromAPI(ctx context.Context, schemeCode string, days int) ([]trading.PricePoint, error) {
	log.Printf("[Pricing API] Trying to fetch %s from external api...\n", ph.assetAPI.GetAssetType())
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	points, err := ph.assetAPI.Fetch(ctx, schemeCode, days)
	if err != nil {
		return nil, err
	}

	// Save fresh data to postgres via separate goroutine
	go func() {
		// Ensure Asset metadata exists first
		exchange := "NSE"

		if ph.assetAPI.GetAssetType() == trading.MutualFund {
			exchange = "AMC"
		}

		log.Println("[Info] Upserting assets data...")
		nDaysDate := time.Now().AddDate(0, 0, -days).Format("2006-01-02")

		err = ph.repo.UpsertAsset(schemeCode, schemeCode, string(ph.assetAPI.GetAssetType()), exchange, nDaysDate)
		if err != nil {
			log.Printf("[Warn] failed to save assets record in posgres %s", err.Error())
		}

		// Save prices to PostgreSQL
		if err := ph.repo.BulkUpsertPriceHistory(schemeCode, points); err != nil {
			log.Printf("[Warn] Failed to save price history to Postgres: %v", err)
		} else {
			log.Printf("[Postgres] Successfully stored %d price records for %s", len(points), schemeCode)
		}
	}()

	return points, nil
}
