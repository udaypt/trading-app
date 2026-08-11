package postgres

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
