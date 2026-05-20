package health

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandlerReturnsOKWhenDatabaseCheckPasses(t *testing.T) {
	t.Parallel()

	handler := NewHandler(func() error {
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var body Response
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if body.Status != "ok" {
		t.Fatalf("expected status ok, got %q", body.Status)
	}

	if body.Database != "ok" {
		t.Fatalf("expected database ok, got %q", body.Database)
	}
}

func TestHandlerReturnsDegradedWhenDatabaseCheckFails(t *testing.T) {
	t.Parallel()

	handler := NewHandler(func() error {
		return errors.New("ping failed")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", rec.Code)
	}

	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected Content-Type application/json on degraded response, got %q", got)
	}

	var body Response
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if body.Status != "degraded" {
		t.Fatalf("expected status degraded, got %q", body.Status)
	}

	if body.Database != "error" {
		t.Fatalf("expected database error, got %q", body.Database)
	}
}
