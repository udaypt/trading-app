package postgres

import (
	"github.com/udaypt/trading-app/internal/domain/usecase/trading"
)

// UpsertAsset inserts asset metadata if it doesn't already exist
func (r *DBRepository) UpsertAsset(id, name, assetType, exchange string, lastNDaysDate string) error {
	query := `
		INSERT INTO assets (id, name, asset_type, exchange, last_nday_fetched_date)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			exchange = EXCLUDED.exchange,
			last_nday_fetched_date = EXCLUDED.last_nday_fetched_date;
	`
	//fmt.Println("INSERTING DATA TO assets TABLE:", id, name, assetType, days)
	//lastNDate := time.Now().AddDate(0, 0, -days).Format("2006-01-02")

	_, err := r.db.Exec(query, id, name, assetType, exchange, lastNDaysDate)
	return err
}

func (r *DBRepository) GetLastNDaysDate(id string) (string, error) {
	query := `
		SELECT last_nday_fetched_date
		FROM assets
		WHERE id = $1
	`
	var date string
	err := r.db.QueryRow(query, id).Scan(&date)
	if err != nil {
		return "", err
	}

	return date, nil
}

// SearchAssets looks up cached assets of the given type whose id or name
// matches the query, so callers can serve a search from the DB before
// falling back to an external API.
func (r *DBRepository) SearchAssets(assetType, query string, limit int) ([]trading.SearchResult, error) {
	sqlQuery := `
		SELECT id, name, asset_type, exchange
		FROM assets
		WHERE asset_type = $1 AND (id ILIKE $2 OR name ILIKE $2)
		ORDER BY name
		LIMIT $3
	`
	rows, err := r.db.Query(sqlQuery, assetType, "%"+query+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []trading.SearchResult
	for rows.Next() {
		var res trading.SearchResult
		var rowAssetType string
		if err := rows.Scan(&res.ID, &res.Name, &rowAssetType, &res.Exchange); err != nil {
			return nil, err
		}
		res.Type = trading.AssetType(rowAssetType)
		res.Symbol = res.ID
		results = append(results, res)
	}

	return results, rows.Err()
}

// UpsertStocksMetadata caches stock search-result metadata without touching
// last_nday_fetched_date, which price-history caching owns.
func (r *DBRepository) UpsertStocksMetadata(id, name, exchange string) error {
	query := `
		INSERT INTO assets (id, name, asset_type, exchange)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			exchange = EXCLUDED.exchange;
	`
	_, err := r.db.Exec(query, id, name, string(trading.Stock), exchange)
	return err
}
