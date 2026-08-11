package postgres

import (
	"fmt"
)

type SchemeRecord struct {
	SchemeCode int    `json:"schemeCode"`
	SchemeName string `json:"schemeName"`
}

// GetAllMFSchemes retrieves all mutual fund schemes stored in PostgreSQL
func (r *DBRepository) GetAllMFSchemes() ([]SchemeRecord, error) {
	query := `SELECT scheme_code, scheme_name FROM mutual_fund_schemes;`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var schemes []SchemeRecord
	for rows.Next() {
		var s SchemeRecord
		if err := rows.Scan(&s.SchemeCode, &s.SchemeName); err != nil {
			return nil, err
		}
		schemes = append(schemes, s)
	}
	return schemes, nil
}

// BulkUpsertMFSchemes inserts or updates scheme records in PostgreSQL
func (r *DBRepository) BulkUpsertMFSchemes(schemes []SchemeRecord) error {
	if len(schemes) == 0 {
		return nil
	}

	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO mutual_fund_schemes (scheme_code, scheme_name, updated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (scheme_code) DO UPDATE SET
			scheme_name = EXCLUDED.scheme_name,
			updated_at = NOW();
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, s := range schemes {
		if _, err := stmt.Exec(s.SchemeCode, s.SchemeName); err != nil {
			return fmt.Errorf("failed to upsert scheme %d: %w", s.SchemeCode, err)
		}
	}

	return tx.Commit()
}
