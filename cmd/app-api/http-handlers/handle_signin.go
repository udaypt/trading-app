package httphandlers

import (
	"encoding/json"
	"net/http"

	"github.com/udaypt/trading-app/internal/httphandler"
	"github.com/udaypt/trading-app/internal/infra/db/postgres"
	"github.com/udaypt/trading-app/internal/security"
)

type SignIn struct {
	repo *postgres.DBRepository
}

func NewSignIn(repo *postgres.DBRepository) *SignIn {
	return &SignIn{
		repo: repo,
	}
}

// func HandleHttp(writer http.ResponseWriter, request *http.Request, handle http.HandlerFunc) {
func (app *SignIn) Handle(w http.ResponseWriter, r *http.Request) {
	httphandler.HandleHttp(w, r, app.handleSignIn)
}

// Endpoint: POST /api/v1/auth/login
func (app *SignIn) handleSignIn(w http.ResponseWriter, r *http.Request) {
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

	userID, name, passwordHash, lastLogin, err := app.repo.AuthenticateUser(req.Email)
	if err != nil || !security.CheckPasswordHash(req.Password, passwordHash) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(security.AuthResponse{Status: "error", Error: "Invalid credentials"})
		return
	}

	// GenerateJWT only errors if the signing key isn't []byte, and the
	// package's key is hardcoded as []byte — unreachable without changing
	// GenerateJWT's signature to accept an injectable key.
	token, err := security.GenerateJWT(userID, req.Email, name, lastLogin)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(security.AuthResponse{Status: "success", Token: token})
}
