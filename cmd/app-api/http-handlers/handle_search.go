package httphandlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	mf "github.com/udaypt/trading-app/internal/domain/services/trading/mutual_fund"
	"github.com/udaypt/trading-app/internal/domain/services/trading/stock"
	"github.com/udaypt/trading-app/internal/domain/usecase/trading"
	"github.com/udaypt/trading-app/internal/httphandler"
	"github.com/udaypt/trading-app/internal/infra/db/postgres"
)

// SearchAPIResponse is the JSON wrapper for GET /api/v1/search
type SearchAPIResponse struct {
	Status string                 `json:"status"`
	Query  string                 `json:"query"`
	Count  int                    `json:"count"`
	Data   []trading.SearchResult `json:"data"`
	Error  string                 `json:"error,omitempty"`
}

type Search struct {
	mfStore *mf.MFStore
	repo    *postgres.DBRepository
}

func NewSearch(store *mf.MFStore, repo *postgres.DBRepository) *Search {
	return &Search{
		mfStore: store,
		repo:    repo,
	}
}

// func HandleHttp(writer http.ResponseWriter, request *http.Request, handle http.HandlerFunc) {
func (app *Search) Handle(w http.ResponseWriter, r *http.Request) {
	httphandler.HandleHttp(w, r, app.handleSearch)
}

// HandleSearch GET /api/v1/search?q=...
func (app *Search) handleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(SearchAPIResponse{
			Status: "error",
			Error:  "Query parameter 'q' is required",
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	var stockResults []trading.SearchResult
	var mfResults []trading.SearchResult

	// Execute Stock search in Goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		var err error
		stockResults, err = stock.SearchStocks(ctx, query, 5)
		if err != nil {
			log.Printf("[Warn] Stock search warning: %v", err)
		}
	}()

	// Execute Mutual Fund search in Goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		mfResults = app.mfStore.Search(query, 5)
	}()

	wg.Wait()

	combined := append(stockResults, mfResults...)

	json.NewEncoder(w).Encode(SearchAPIResponse{
		Status: "success",
		Query:  query,
		Count:  len(combined),
		Data:   combined,
	})
}
