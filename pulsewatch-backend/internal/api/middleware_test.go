package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogging(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusTeapot)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)

	Logging(next).ServeHTTP(w, req)

	assert.True(t, called, "Logging must delegate to the wrapped handler")
	assert.Equal(t, http.StatusTeapot, w.Code, "Logging must not alter the response")
}

func TestCORS_PreflightOptionsShortCircuits(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/api/monitors", nil)

	CORS(next).ServeHTTP(w, req)

	assert.False(t, called, "OPTIONS must not reach the wrapped handler")
	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "POST")
	assert.Contains(t, w.Header().Get("Access-Control-Allow-Headers"), "Content-Type")
}

func TestCORS_OtherMethodsCallThrough(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/monitors", nil)

	CORS(next).ServeHTTP(w, req)

	assert.True(t, called, "CORS must delegate non-OPTIONS requests")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
}
