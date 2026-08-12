package postgres

import (
	"database/sql"
	"time"
)

func (r *DBRepository) CreateNewUser(email, pwdHash, name string) (int64, error) {
	var userID int64
	query := `INSERT INTO users (email, password_hash, name) VALUES ($1, $2, $3) RETURNING id`
	err := r.db.QueryRow(query, email, pwdHash, name).Scan(&userID)
	if err != nil {
		return int64(0), err
	}

	return userID, nil
}

func (r *DBRepository) AuthenticateUser(email string) (int64, string, string, time.Time, error) {
	var userID int64
	var passwordHash string
	var name *string
	var previousLastLogin sql.NullTime
	query := `SELECT id, name, password_hash, last_login FROM users WHERE email = $1`
	err := r.db.QueryRow(query, email).Scan(&userID, &name, &passwordHash, &previousLastLogin)
	if err != nil {
		return int64(0), "", "", time.Time{}, err
	}

	// Update last_login timestamp to current time for future log-ins
	updateQuery := `UPDATE users SET last_login = NOW() WHERE id = $1`
	_, _ = r.db.Exec(updateQuery, userID)
	var userName string
	if name == nil {
		userName = ""
	} else {
		userName = *name
	}
	return userID, userName, passwordHash, previousLastLogin.Time, nil
}

// AuthenticateUser retrieves credentials, records previous last_login, and updates last_login to NOW()
/*func (r *DBRepository) AuthenticateUser(email string) (int64, string, string, time.Time, error) {
	var userID int64
	var name, passwordHash string
	var previousLastLogin sql.NullTime

	// 1. Get user details
	query := `SELECT id, name, password_hash, last_login FROM users WHERE email = $1`
	err := r.db.QueryRow(query, email).Scan(&userID, &name, &passwordHash, &previousLastLogin)
	if err != nil {
		return 0, "", "", time.Time{}, err
	}

	// 2. Update last_login timestamp to current time for future log-ins
	updateQuery := `UPDATE users SET last_login = NOW() WHERE id = $1`
	_, _ = r.db.Exec(updateQuery, userID)

	// Return the previous last_login timestamp (zero time if first time logging in)
	return userID, name, passwordHash, previousLastLogin.Time, nil
}
*/
