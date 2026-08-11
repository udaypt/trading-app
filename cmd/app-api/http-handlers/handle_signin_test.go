package httphandlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"trading-dashboard/internal/infra/db/postgres"
	"trading-dashboard/internal/security"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newRepoWithMock(t *testing.T) (*postgres.DBRepository, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return postgres.NewDBRepositoryWithDB(db), mock
}

func TestSignIn_Handle(t *testing.T) {
	t.Run("OPTIONS preflight returns 200 without touching the repo", func(t *testing.T) {
		repo, _ := newRepoWithMock(t)
		app := NewSignIn(repo)

		req := httptest.NewRequest(http.MethodOptions, "/api/v1/auth/login", nil)
		rec := httptest.NewRecorder()
		app.Handle(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("invalid JSON body returns 400", func(t *testing.T) {
		repo, _ := newRepoWithMock(t)
		app := NewSignIn(repo)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString("not json"))
		rec := httptest.NewRecorder()
		app.Handle(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		var resp security.AuthResponse
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
		assert.Equal(t, "error", resp.Status)
	})

	t.Run("unknown email returns 401", func(t *testing.T) {
		repo, mock := newRepoWithMock(t)
		mock.ExpectQuery("SELECT id, password_hash FROM users").
			WithArgs("missing@example.com").
			WillReturnError(errors.New("no rows"))
		app := NewSignIn(repo)

		body, _ := json.Marshal(security.LoginRequest{Email: "missing@example.com", Password: "pwd"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
		rec := httptest.NewRecorder()
		app.Handle(rec, req)

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("wrong password returns 401", func(t *testing.T) {
		repo, mock := newRepoWithMock(t)
		hash, err := security.HashPassword("correct-password")
		require.NoError(t, err)
		mock.ExpectQuery("SELECT id, password_hash FROM users").
			WithArgs("user@example.com").
			WillReturnRows(sqlmock.NewRows([]string{"id", "password_hash"}).AddRow(int64(1), hash))
		app := NewSignIn(repo)

		body, _ := json.Marshal(security.LoginRequest{Email: "user@example.com", Password: "wrong-password"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
		rec := httptest.NewRecorder()
		app.Handle(rec, req)

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("correct credentials return a token", func(t *testing.T) {
		repo, mock := newRepoWithMock(t)
		hash, err := security.HashPassword("correct-password")
		require.NoError(t, err)
		mock.ExpectQuery("SELECT id, password_hash FROM users").
			WithArgs("user@example.com").
			WillReturnRows(sqlmock.NewRows([]string{"id", "password_hash"}).AddRow(int64(1), hash))
		app := NewSignIn(repo)

		body, _ := json.Marshal(security.LoginRequest{Email: "user@example.com", Password: "correct-password"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
		rec := httptest.NewRecorder()
		app.Handle(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var resp security.AuthResponse
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
		assert.Equal(t, "success", resp.Status)
		assert.NotEmpty(t, resp.Token)
	})
}
