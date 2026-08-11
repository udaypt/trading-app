package postgres

func (r *DBRepository) CreateNewUser(email, pwdHash string) (int64, error) {
	var userID int64
	query := `INSERT INTO users (email, password_hash) VALUES ($1, $2) RETURNING id`
	err := r.db.QueryRow(query, email, pwdHash).Scan(&userID)
	if err != nil {
		return int64(0), err
	}

	return userID, nil
}

func (r *DBRepository) GetCredential(email string) (int64, string, error) {
	var userID int64
	var passwordHash string
	query := `SELECT id, password_hash FROM users WHERE email = $1`
	err := r.db.QueryRow(query, email).Scan(&userID, &passwordHash)
	if err != nil {
		return int64(0), "", err
	}

	return userID, passwordHash, nil
}
