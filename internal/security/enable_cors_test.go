package security

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnableCORS(t *testing.T) {
	w := httptest.NewRecorder()
	EnableCORS(w)

	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "GET, POST, PUT, DELETE, OPTIONS", w.Header().Get("Access-Control-Allow-Methods"))
	assert.Equal(t, "Content-Type, Authorization", w.Header().Get("Access-Control-Allow-Headers"))
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
}
