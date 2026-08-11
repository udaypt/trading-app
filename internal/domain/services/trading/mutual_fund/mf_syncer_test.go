package mutualfund

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

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

func TestMFStoreSyncer_Sync(t *testing.T) {
	t.Run("success populates return value and persists via repo", func(t *testing.T) {
		payload := []Scheme{
			{SchemeCode: 10, SchemeName: "Test Fund One"},
			{SchemeCode: 11, SchemeName: "Test Fund Two"},
		}
		client := withMFSyncServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(payload)
		})

		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectBegin()
		prep := mock.ExpectPrepare("INSERT INTO mutual_fund_schemes")
		prep.ExpectExec().WithArgs(10, "Test Fund One").WillReturnResult(sqlmock.NewResult(1, 1))
		prep.ExpectExec().WithArgs(11, "Test Fund Two").WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		repo := postgres.NewDBRepositoryWithDB(db)
		syncer := &MFStoreSyncer{provider: &MFStoreProvider{client: client}, repo: repo}

		schemes, err := syncer.Sync(context.Background())
		require.NoError(t, err)
		require.Len(t, schemes, 2)
		assert.Equal(t, "Test Fund One", schemes[0].SchemeName)

		// The persistence happens asynchronously in a goroutine; give it a
		// moment to complete before checking mock expectations.
		assert.Eventually(t, func() bool {
			return mock.ExpectationsWereMet() == nil
		}, time.Second, 10*time.Millisecond)
	})

	t.Run("logs but does not fail when async persistence errors", func(t *testing.T) {
		payload := []Scheme{{SchemeCode: 20, SchemeName: "Persist Failure Fund"}}
		client := withMFSyncServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(payload)
		})

		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectBegin().WillReturnError(assert.AnError)

		repo := postgres.NewDBRepositoryWithDB(db)
		syncer := &MFStoreSyncer{provider: &MFStoreProvider{client: client}, repo: repo}

		schemes, err := syncer.Sync(context.Background())
		require.NoError(t, err) // the sync call itself succeeds; persistence is fire-and-forget
		require.Len(t, schemes, 1)

		assert.Eventually(t, func() bool {
			return mock.ExpectationsWereMet() == nil
		}, time.Second, 10*time.Millisecond)
	})

	t.Run("propagates error when fetching exhausts retries", func(t *testing.T) {
		client := withMFSyncServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		})

		syncer := &MFStoreSyncer{provider: &MFStoreProvider{client: client}}
		_, err := syncer.Sync(context.Background())
		assert.Error(t, err)
	})
}

func TestMFStoreSyncer_fetchWithRetry(t *testing.T) {
	t.Run("succeeds after transient failures within the retry budget", func(t *testing.T) {
		var attempts int32
		client := withMFSyncServer(t, func(w http.ResponseWriter, r *http.Request) {
			if atomic.AddInt32(&attempts, 1) < 3 {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode([]Scheme{{SchemeCode: 1, SchemeName: "Recovered Fund"}})
		})

		syncer := &MFStoreSyncer{provider: &MFStoreProvider{client: client}}
		schemes, err := syncer.fetchWithRetry(context.Background())
		require.NoError(t, err)
		require.Len(t, schemes, 1)
		assert.Equal(t, "Recovered Fund", schemes[0].SchemeName)
		assert.EqualValues(t, 3, atomic.LoadInt32(&attempts))
	})

	t.Run("gives up after exhausting all progressive retries", func(t *testing.T) {
		var attempts int32
		client := withMFSyncServer(t, func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&attempts, 1)
			w.WriteHeader(http.StatusServiceUnavailable)
		})

		syncer := &MFStoreSyncer{provider: &MFStoreProvider{client: client}}
		_, err := syncer.fetchWithRetry(context.Background())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "after 4 attempts")
		assert.EqualValues(t, syncMaxAttempts, atomic.LoadInt32(&attempts))
	})

	t.Run("stops retrying once the context is canceled during backoff", func(t *testing.T) {
		origWait := syncRetryBaseWait
		syncRetryBaseWait = 200 * time.Millisecond // long enough to cancel mid-wait
		defer func() { syncRetryBaseWait = origWait }()

		var attempts int32
		client := withMFSyncServer(t, func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&attempts, 1)
			w.WriteHeader(http.StatusServiceUnavailable)
		})

		ctx, cancel := context.WithCancel(context.Background())
		syncer := &MFStoreSyncer{provider: &MFStoreProvider{client: client}}

		go func() {
			time.Sleep(20 * time.Millisecond) // after attempt 1 fails, while backoff is sleeping
			cancel()
		}()

		_, err := syncer.fetchWithRetry(ctx)
		assert.ErrorIs(t, err, context.Canceled)
		assert.EqualValues(t, 1, atomic.LoadInt32(&attempts))
	})
}

func TestMFStoreSyncer_StartBackgroundSync(t *testing.T) {
	// The scheduled sync fails (500), which exercises both the ticker
	// machinery and the "scheduled sync failed" log branch in one pass; the
	// success path of Sync itself is already covered directly by
	// TestMFStoreSyncer_Sync.
	client := withMFSyncServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	syncer := &MFStoreSyncer{provider: &MFStoreProvider{client: client}}

	// Redirect log output so we can deterministically detect when the
	// background goroutine has actually returned. A plain sleep-based wait
	// would race with the ticker goroutine's own reads of MFSyncAPIURL;
	// observing its log lines gives a real happens-before edge (via the log
	// package's internal mutex) before we touch shared state.
	var logBuf syncBuffer
	origLogOutput := log.Writer()
	log.SetOutput(&logBuf)
	defer log.SetOutput(origLogOutput)

	ctx, cancel := context.WithCancel(context.Background())
	syncer.StartBackgroundSync(ctx, 10*time.Millisecond, func([]Scheme) {
		t.Fatal("onSync should not be called when the scheduled sync fails")
	})

	assert.Eventually(t, func() bool {
		return strings.Contains(logBuf.String(), "Scheduled sync failed")
	}, time.Second, 5*time.Millisecond, "expected the ticker to trigger a scheduled sync")

	cancel()
	require.Eventually(t, func() bool {
		return strings.Contains(logBuf.String(), "Stopping background sync worker")
	}, time.Second, 5*time.Millisecond, "expected background sync worker to stop")
}

func TestMFStoreSyncer_StartBackgroundSync_CallsOnSync(t *testing.T) {
	payload := []Scheme{{SchemeCode: 42, SchemeName: "Ticked Fund"}}
	client := withMFSyncServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(payload)
	})

	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := postgres.NewDBRepositoryWithDB(db)

	syncer := &MFStoreSyncer{provider: &MFStoreProvider{client: client}, repo: repo}

	var gotSchemes []Scheme
	var mu sync.Mutex
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	syncer.StartBackgroundSync(ctx, 10*time.Millisecond, func(schemes []Scheme) {
		mu.Lock()
		defer mu.Unlock()
		gotSchemes = schemes
	})

	assert.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(gotSchemes) == 1
	}, time.Second, 5*time.Millisecond, "expected onSync to be called with fresh schemes")
}
