package httphandler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/dig"

	"github.com/stretchr/testify/assert"
)

func TestHandleHttp(t *testing.T) {
	var called bool
	handle := func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusTeapot)
	}

	req := httptest.NewRequest(http.MethodGet, "/anything", nil)
	rec := httptest.NewRecorder()

	HandleHttp(rec, req, handle)

	assert.True(t, called)
	assert.Equal(t, http.StatusTeapot, rec.Code)
}

func TestRunHTTPServer_UnresolvableHandlerDependency(t *testing.T) {
	// An empty container can't resolve router.Handler, so container.Invoke
	// fails before http.ListenAndServe is ever reached — this exercises the
	// error-wrapping path without starting a real, blocking server.
	container := dig.New()

	err := RunHTTPServer(container, ":0")
	assert.Error(t, err)
}
