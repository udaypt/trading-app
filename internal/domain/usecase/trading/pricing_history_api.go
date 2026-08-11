package trading

import (
	"context"
)

type PricingHistoryAPI interface {
	Fetch(ctx context.Context, schemeCode string, days int) ([]PricePoint, error)
	GetAssetType() AssetType
}
