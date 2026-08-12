package httphandlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/udaypt/trading-app/internal/security"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSignUp_Handle(t *testing.T) {
	t.Run("OPTIONS preflight returns 200 without touching the repo", func(t *testing.T) {
		repo, _ := newRepoWithMock(t)
		app := NewSignUp(repo)

		req := httptest.NewRequest(http.MethodOptions, "/api/v1/auth/register", nil)
		rec := httptest.NewRecorder()
		app.Handle(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("invalid JSON body returns 400", func(t *testing.T) {
		repo, _ := newRepoWithMock(t)
		app := NewSignUp(repo)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString("not json"))
		rec := httptest.NewRecorder()
		app.Handle(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("password over bcrypt's 72-byte limit returns 500", func(t *testing.T) {
		repo, _ := newRepoWithMock(t)
		app := NewSignUp(repo)

		body, _ := json.Marshal(security.SignUpRequest{
			Email:    "user@example.com",
			Password: strings.Repeat("a", 73),
			Name:     "Test user",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
		rec := httptest.NewRecorder()
		app.Handle(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("existing account returns 409", func(t *testing.T) {
		repo, mock := newRepoWithMock(t)
		mock.ExpectQuery("INSERT INTO users").WillReturnError(errors.New("duplicate key"))
		app := NewSignUp(repo)

		body, _ := json.Marshal(security.SignUpRequest{Email: "dupe@example.com", Password: "pwd", Name: "Test name"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
		rec := httptest.NewRecorder()
		app.Handle(rec, req)

		assert.Equal(t, http.StatusConflict, rec.Code)
	})

	t.Run("new account returns 201", func(t *testing.T) {
		repo, mock := newRepoWithMock(t)
		mock.ExpectQuery("INSERT INTO users").
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(1)))
		app := NewSignUp(repo)

		body, _ := json.Marshal(security.SignUpRequest{Email: "new@example.com", Password: "pwd", Name: "test user"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
		rec := httptest.NewRecorder()
		app.Handle(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Code)
		var resp security.AuthResponse
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
		assert.Equal(t, "success", resp.Status)
	})
}
