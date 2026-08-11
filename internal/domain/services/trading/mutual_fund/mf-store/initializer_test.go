package mfstore

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	mutualfund "github.com/udaypt/trading-app/internal/domain/services/trading/mutual_fund"
	"github.com/udaypt/trading-app/internal/infra/db/postgres"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitializer_Initialize(t *testing.T) {
	t.Run("serves schemes already cached in postgres without calling the API", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		rows := sqlmock.NewRows([]string{"scheme_code", "scheme_name"}).
			AddRow(1, "Existing Fund")
		mock.ExpectQuery("SELECT scheme_code, scheme_name FROM mutual_fund_schemes").WillReturnRows(rows)

		repo := postgres.NewDBRepositoryWithDB(db)
		client := withMFSyncServer(t, func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("external API should not be called when postgres already has data")
		})

		store := mutualfund.NewMFStore()
		loader := NewLoader(repo)
		syncer := &Syncer{provider: &Provider{client: client}, repo: repo}
		initializer := NewInitializer(store, loader, syncer)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		got := initializer.Initialize(ctx)
		require.Same(t, store, got)
		assert.Len(t, got.Search("existing", 10), 1)
	})

	t.Run("falls back to the API when postgres has no schemes", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectQuery("SELECT scheme_code, scheme_name FROM mutual_fund_schemes").
			WillReturnRows(sqlmock.NewRows([]string{"scheme_code", "scheme_name"}))

		repo := postgres.NewDBRepositoryWithDB(db)
		payload := []mutualfund.Scheme{{SchemeCode: 99, SchemeName: "Seeded Fund"}}
		client := withMFSyncServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(payload)
		})

		store := mutualfund.NewMFStore()
		loader := NewLoader(repo)
		syncer := &Syncer{provider: &Provider{client: client}, repo: repo}
		initializer := NewInitializer(store, loader, syncer)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		got := initializer.Initialize(ctx)
		assert.Len(t, got.Search("seeded", 10), 1)
	})

	t.Run("panics when neither postgres nor the API can produce data", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectQuery("SELECT scheme_code, scheme_name FROM mutual_fund_schemes").
			WillReturnRows(sqlmock.NewRows([]string{"scheme_code", "scheme_name"}))

		repo := postgres.NewDBRepositoryWithDB(db)
		client := withMFSyncServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		})

		store := mutualfund.NewMFStore()
		loader := NewLoader(repo)
		syncer := &Syncer{provider: &Provider{client: client}, repo: repo}
		initializer := NewInitializer(store, loader, syncer)

		assert.Panics(t, func() {
			initializer.Initialize(context.Background())
		})
	})
}
