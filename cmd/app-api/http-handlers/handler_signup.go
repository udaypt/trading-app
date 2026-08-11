package httphandlers

import (
	"encoding/json"
	"github.com/udaypt/trading-app/internal/httphandler"
	"github.com/udaypt/trading-app/internal/infra/db/postgres"
	"github.com/udaypt/trading-app/internal/security"
	"net/http"
)

type SignUp struct {
	repo *postgres.DBRepository
}

func NewSignUp(repo *postgres.DBRepository) *SignUp {
	return &SignUp{
		repo: repo,
	}
}

func (app *SignUp) Handle(w http.ResponseWriter, r *http.Request) {
	httphandler.HandleHttp(w, r, app.handleSignUp)
}

// Endpoint: POST /api/v1/auth/register
func (app *SignUp) handleSignUp(w http.ResponseWriter, r *http.Request) {
	security.EnableCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	var req security.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(security.AuthResponse{Status: "error", Error: "Invalid JSON body"})
		return
	}

	hashedPassword, err := security.HashPassword(req.Password)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	_, err = app.repo.CreateNewUser(req.Email, hashedPassword)
	if err != nil {
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(security.AuthResponse{Status: "error", Error: "Account already exists!"})
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(security.AuthResponse{Status: "success"})
}
