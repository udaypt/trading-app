package postgres

import (
	"fmt"
	"github.com/udaypt/trading-app/internal/domain/usecase/trading"
)

// BulkUpsertPriceHistory efficiently stores/updates price data points in PostgreSQL
func (r *DBRepository) BulkUpsertPriceHistory(assetID string, points []trading.PricePoint) error {
	if len(points) == 0 {
		return nil
	}

	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO price_history (asset_id, price_date, price)
		VALUES ($1, $2, $3)
		ON CONFLICT (asset_id, price_date) DO UPDATE SET
			price = EXCLUDED.price;
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, pt := range points {
		_, err := stmt.Exec(assetID, pt.Date, pt.Price)
		if err != nil {
			return fmt.Errorf("failed inserting point %s: %w", pt.Date, err)
		}
	}

	return tx.Commit()
}

// GetPriceHistory retrieves the last N days of prices from PostgreSQL
func (r *DBRepository) GetPriceHistory(assetID string, days int) ([]trading.PricePoint, error) {
	query := `
		SELECT price_date, price 
		FROM price_history 
		WHERE asset_id = $1 
		ORDER BY price_date DESC 
		LIMIT $2;
	`
	rows, err := r.db.Query(query, assetID, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []trading.PricePoint
	for rows.Next() {
		var date string
		var price float64
		if err := rows.Scan(&date, &price); err != nil {
			return nil, err
		}
		// Truncate timestamp string to YYYY-MM-DD
		if len(date) >= 10 {
			date = date[:10]
		}
		points = append(points, trading.PricePoint{Date: date, Price: price})
	}

	// Reverse to chronological order (oldest -> newest) for charting
	for i, j := 0, len(points)-1; i < j; i, j = i+1, j-1 {
		points[i], points[j] = points[j], points[i]
	}

	return points, nil
}
