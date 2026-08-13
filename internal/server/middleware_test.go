package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// passthrough returns a handler that records whether it ran, so a test can
// distinguish "rejected by the middleware" from "reached the handler".
func passthrough() (http.Handler, *bool) {
	reached := false
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reached = true
		w.WriteHeader(http.StatusOK)
	})
	return h, &reached
}

func TestLocalhostOnlyRejectsForeignHost(t *testing.T) {
	next, reached := passthrough()
	handler := localhostOnly(next)

	req := httptest.NewRequest(http.MethodGet, "http://evil.com/api/diff", nil)
	req.Host = "evil.com"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", rec.Code)
	}
	if *reached {
		t.Error("handler ran for a foreign Host")
	}
}

func TestLocalhostOnlyAllowsLocalHost(t *testing.T) {
	next, reached := passthrough()
	handler := localhostOnly(next)

	req := httptest.NewRequest(http.MethodGet, "http://localhost:8799/api/diff", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	if !*reached {
		t.Error("handler did not run for a local Host")
	}
}

func TestLocalhostOnlyRejectsForeignOriginPost(t *testing.T) {
	next, reached := passthrough()
	handler := localhostOnly(next)

	req := httptest.NewRequest(http.MethodPost, "http://localhost:8799/api/shutdown", nil)
	req.Header.Set("Origin", "https://evil.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", rec.Code)
	}
	if *reached {
		t.Error("handler ran for a foreign-Origin POST")
	}
}

// The Origin header is absent on same-origin POSTs and some simple requests,
// so an absent Origin must not be rejected.
func TestLocalhostOnlyAllowsAbsentOriginPost(t *testing.T) {
	next, reached := passthrough()
	handler := localhostOnly(next)

	req := httptest.NewRequest(http.MethodPost, "http://localhost:8799/api/shutdown", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	if !*reached {
		t.Error("handler did not run for an absent-Origin POST")
	}
}
