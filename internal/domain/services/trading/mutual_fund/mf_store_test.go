package mutualfund

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/udaypt/trading-app/internal/domain/usecase/trading"
	"github.com/udaypt/trading-app/internal/infra/db/postgres"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMain speeds up the whole package's retry-path tests by shrinking the
// progressive backoff from seconds to milliseconds; individual tests that
// need to observe real backoff timing (e.g. mid-wait cancellation) restore
// a larger value locally.
func TestMain(m *testing.M) {
	syncRetryBaseWait = time.Millisecond
	os.Exit(m.Run())
}

// syncBuffer is a concurrency-safe io.Writer, used to capture log output
// from a background goroutine without racing on a plain bytes.Buffer.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func newTestStore(schemes []Scheme) *MFStore {
	return &MFStore{
		schemes: schemes,
		client:  &http.Client{Timeout: time.Second},
	}
}

func TestMFStore_Search(t *testing.T) {
	schemes := []Scheme{
		{SchemeCode: 1, SchemeName: "HDFC Top 100 Fund"},
		{SchemeCode: 2, SchemeName: "SBI Blue Chip Fund"},
		{SchemeCode: 3, SchemeName: "HDFC Small Cap Fund"},
	}

	tests := []struct {
		name      string
		query     string
		limit     int
		wantCount int
		wantIDs   []string
	}{
		{
			name:      "empty query returns nil",
			query:     "   ",
			limit:     10,
			wantCount: 0,
		},
		{
			name:      "single token matches all with substring",
			query:     "hdfc",
			limit:     10,
			wantCount: 2,
			wantIDs:   []string{"1", "3"},
		},
		{
			name:      "case insensitive match",
			query:     "SBI",
			limit:     10,
			wantCount: 1,
			wantIDs:   []string{"2"},
		},
		{
			name:      "multi-token requires all tokens present",
			query:     "hdfc small",
			limit:     10,
			wantCount: 1,
			wantIDs:   []string{"3"},
		},
		{
			name:      "no match returns empty",
			query:     "nonexistent",
			limit:     10,
			wantCount: 0,
		},
		{
			name:      "limit truncates results",
			query:     "fund",
			limit:     1,
			wantCount: 1,
		},
		{
			name:      "zero limit means unlimited",
			query:     "fund",
			limit:     0,
			wantCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newTestStore(schemes)
			results := store.Search(tt.query, tt.limit)
			assert.Len(t, results, tt.wantCount)
			if tt.wantIDs != nil {
				gotIDs := make([]string, 0, len(results))
				for _, r := range results {
					gotIDs = append(gotIDs, r.ID)
					assert.Equal(t, trading.AssetType(trading.MutualFund), r.Type)
					assert.Equal(t, "AMC", r.Exchange)
				}
				assert.ElementsMatch(t, tt.wantIDs, gotIDs)
			}
		})
	}
}

func TestMFStore_Search_EmptyStore(t *testing.T) {
	store := newTestStore(nil)
	results := store.Search("anything", 10)
	assert.Empty(t, results)
}

func TestMFStore_syncAPIWithDB(t *testing.T) {
	t.Run("success populates cache and persists via repo", func(t *testing.T) {
		payload := []Scheme{
			{SchemeCode: 10, SchemeName: "Test Fund One"},
			{SchemeCode: 11, SchemeName: "Test Fund Two"},
		}
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(payload)
		}))
		defer srv.Close()

		origURL := MFSyncAPIURL
		MFSyncAPIURL = srv.URL
		defer func() { MFSyncAPIURL = origURL }()

		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectBegin()
		prep := mock.ExpectPrepare("INSERT INTO mutual_fund_schemes")
		prep.ExpectExec().WithArgs(10, "Test Fund One").WillReturnResult(sqlmock.NewResult(1, 1))
		prep.ExpectExec().WithArgs(11, "Test Fund Two").WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		repo := postgres.NewDBRepositoryWithDB(db)
		store := &MFStore{client: srv.Client(), repo: repo}

		err = store.syncAPIWithDB(context.Background())
		require.NoError(t, err)

		assert.Len(t, store.schemes, 2)
		assert.Equal(t, "Test Fund One", store.schemes[0].SchemeName)

		// The persistence happens asynchronously in a goroutine; give it a
		// moment to complete before checking mock expectations.
		assert.Eventually(t, func() bool {
			return mock.ExpectationsWereMet() == nil
		}, time.Second, 10*time.Millisecond)
	})

	t.Run("non-200 status returns error and leaves cache untouched", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer srv.Close()

		origURL := MFSyncAPIURL
		MFSyncAPIURL = srv.URL
		defer func() { MFSyncAPIURL = origURL }()

		store := &MFStore{client: srv.Client()}
		err := store.syncAPIWithDB(context.Background())
		assert.Error(t, err)
		assert.Empty(t, store.schemes)
	})

	t.Run("malformed json returns error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("not json"))
		}))
		defer srv.Close()

		origURL := MFSyncAPIURL
		MFSyncAPIURL = srv.URL
		defer func() { MFSyncAPIURL = origURL }()

		store := &MFStore{client: srv.Client()}
		err := store.syncAPIWithDB(context.Background())
		assert.Error(t, err)
	})

	t.Run("invalid URL returns a request-build error", func(t *testing.T) {
		origURL := MFSyncAPIURL
		MFSyncAPIURL = "http://example.com/\n"
		defer func() { MFSyncAPIURL = origURL }()

		store := &MFStore{client: &http.Client{Timeout: time.Second}}
		err := store.syncAPIWithDB(context.Background())
		assert.Error(t, err)
	})

	t.Run("unreachable server returns a transport error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		closedURL := srv.URL
		srv.Close()

		origURL := MFSyncAPIURL
		MFSyncAPIURL = closedURL
		defer func() { MFSyncAPIURL = origURL }()

		store := &MFStore{client: &http.Client{Timeout: time.Second}}
		err := store.syncAPIWithDB(context.Background())
		assert.Error(t, err)
	})

	t.Run("logs but does not fail when async persistence errors", func(t *testing.T) {
		payload := []Scheme{{SchemeCode: 20, SchemeName: "Persist Failure Fund"}}
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(payload)
		}))
		defer srv.Close()

		origURL := MFSyncAPIURL
		MFSyncAPIURL = srv.URL
		defer func() { MFSyncAPIURL = origURL }()

		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectBegin().WillReturnError(assert.AnError)

		repo := postgres.NewDBRepositoryWithDB(db)
		store := &MFStore{client: srv.Client(), repo: repo}

		err = store.syncAPIWithDB(context.Background())
		require.NoError(t, err) // the sync call itself succeeds; persistence is fire-and-forget

		assert.Eventually(t, func() bool {
			return mock.ExpectationsWereMet() == nil
		}, time.Second, 10*time.Millisecond)
	})
}

func TestMFStore_fetchSchemesWithRetry(t *testing.T) {
	t.Run("succeeds after transient failures within the retry budget", func(t *testing.T) {
		var attempts int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if atomic.AddInt32(&attempts, 1) < 3 {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode([]Scheme{{SchemeCode: 1, SchemeName: "Recovered Fund"}})
		}))
		defer srv.Close()

		origURL := MFSyncAPIURL
		MFSyncAPIURL = srv.URL
		defer func() { MFSyncAPIURL = origURL }()

		store := &MFStore{client: srv.Client()}
		schemes, err := store.fetchSchemesWithRetry(context.Background())
		require.NoError(t, err)
		require.Len(t, schemes, 1)
		assert.Equal(t, "Recovered Fund", schemes[0].SchemeName)
		assert.EqualValues(t, 3, atomic.LoadInt32(&attempts))
	})

	t.Run("gives up after exhausting all progressive retries", func(t *testing.T) {
		var attempts int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&attempts, 1)
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		defer srv.Close()

		origURL := MFSyncAPIURL
		MFSyncAPIURL = srv.URL
		defer func() { MFSyncAPIURL = origURL }()

		store := &MFStore{client: srv.Client()}
		_, err := store.fetchSchemesWithRetry(context.Background())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "after 4 attempts")
		assert.EqualValues(t, syncMaxAttempts, atomic.LoadInt32(&attempts))
	})

	t.Run("stops retrying once the context is canceled during backoff", func(t *testing.T) {
		origWait := syncRetryBaseWait
		syncRetryBaseWait = 200 * time.Millisecond // long enough to cancel mid-wait
		defer func() { syncRetryBaseWait = origWait }()

		var attempts int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&attempts, 1)
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		defer srv.Close()

		origURL := MFSyncAPIURL
		MFSyncAPIURL = srv.URL
		defer func() { MFSyncAPIURL = origURL }()

		ctx, cancel := context.WithCancel(context.Background())
		store := &MFStore{client: srv.Client()}

		go func() {
			time.Sleep(20 * time.Millisecond) // after attempt 1 fails, while backoff is sleeping
			cancel()
		}()

		_, err := store.fetchSchemesWithRetry(ctx)
		assert.ErrorIs(t, err, context.Canceled)
		assert.EqualValues(t, 1, atomic.LoadInt32(&attempts))
	})
}

func TestNewMFStore(t *testing.T) {
	t.Run("loads schemes from DB when present", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		rows := sqlmock.NewRows([]string{"scheme_code", "scheme_name"}).
			AddRow(1, "Existing Fund")
		mock.ExpectQuery("SELECT scheme_code, scheme_name FROM mutual_fund_schemes").WillReturnRows(rows)

		repo := postgres.NewDBRepositoryWithDB(db)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		store, err := NewMFStore(ctx, repo)
		require.NoError(t, err)
		require.NotNil(t, store)

		results := store.Search("existing", 10)
		assert.Len(t, results, 1)
	})

	t.Run("seeds from API when DB has no schemes", func(t *testing.T) {
		payload := []Scheme{{SchemeCode: 99, SchemeName: "Seeded Fund"}}
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(payload)
		}))
		defer srv.Close()

		origURL := MFSyncAPIURL
		MFSyncAPIURL = srv.URL
		defer func() { MFSyncAPIURL = origURL }()

		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectQuery("SELECT scheme_code, scheme_name FROM mutual_fund_schemes").
			WillReturnRows(sqlmock.NewRows([]string{"scheme_code", "scheme_name"}))

		repo := postgres.NewDBRepositoryWithDB(db)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		store, err := NewMFStore(ctx, repo)
		require.NoError(t, err)

		results := store.Search("seeded", 10)
		assert.Len(t, results, 1)
	})

	t.Run("logs but does not fail when the initial API seed also fails", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer srv.Close()

		origURL := MFSyncAPIURL
		MFSyncAPIURL = srv.URL
		defer func() { MFSyncAPIURL = origURL }()

		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectQuery("SELECT scheme_code, scheme_name FROM mutual_fund_schemes").
			WillReturnError(assert.AnError)

		repo := postgres.NewDBRepositoryWithDB(db)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		store, err := NewMFStore(ctx, repo)
		require.NoError(t, err) // seed failure is logged, not returned
		assert.Empty(t, store.schemes)
	})
}

func TestMFStore_startBackgroundSync(t *testing.T) {
	// The scheduled sync fails (500), which exercises both the ticker
	// machinery and the "scheduled sync failed" log branch in one pass; the
	// success path of syncAPIWithDB itself is already covered directly by
	// TestMFStore_syncAPIWithDB.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	origURL := MFSyncAPIURL
	MFSyncAPIURL = srv.URL

	store := &MFStore{client: srv.Client()}

	// Redirect log output so we can deterministically detect when the
	// background goroutine has actually returned. A plain sleep-based wait
	// would race with the ticker goroutine's own reads of MFSyncAPIURL;
	// observing its log lines gives a real happens-before edge (via the log
	// package's internal mutex) before we touch shared state.
	var logBuf syncBuffer
	origLogOutput := log.Writer()
	log.SetOutput(&logBuf)

	ctx, cancel := context.WithCancel(context.Background())
	store.startBackgroundSync(ctx, 10*time.Millisecond)

	assert.Eventually(t, func() bool {
		return strings.Contains(logBuf.String(), "Scheduled sync failed")
	}, time.Second, 5*time.Millisecond, "expected the ticker to trigger a scheduled sync")

	cancel()
	require.Eventually(t, func() bool {
		return strings.Contains(logBuf.String(), "Stopping background sync worker")
	}, time.Second, 5*time.Millisecond, "expected background sync worker to stop")

	log.SetOutput(origLogOutput)
	MFSyncAPIURL = origURL
}
