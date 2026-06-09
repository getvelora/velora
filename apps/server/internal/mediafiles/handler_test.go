package mediafiles

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/getvelora/velora/apps/server/internal/libraries"
)

type stubLibraryStore struct {
	library libraries.Library
	err     error
}

func (s stubLibraryStore) Get(context.Context, int64) (libraries.Library, error) {
	if s.err != nil {
		return libraries.Library{}, s.err
	}
	return s.library, nil
}

type stubFileStore struct {
	mu               sync.Mutex
	files            []File
	reconcileSummary ReconcileSummary
	reconcileCalls   int
	listErr          error
	reconcileErr     error
}

func (s *stubFileStore) List(context.Context, int64) ([]File, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	return s.files, nil
}

func (s *stubFileStore) Reconcile(
	context.Context,
	int64,
	[]DiscoveredFile,
	time.Time,
) (ReconcileSummary, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reconcileCalls++
	return s.reconcileSummary, s.reconcileErr
}

func TestHandlerListsFiles(t *testing.T) {
	t.Parallel()
	store := &stubFileStore{files: []File{{ID: 1, LibraryID: 7, Path: "movie.mkv"}}}
	handler := NewHandler(
		stubLibraryStore{library: libraries.Library{ID: 7, Path: "/media/movies"}},
		store,
		"/media",
		func(context.Context, string) (DiscoveryResult, error) {
			return DiscoveryResult{}, nil
		},
	)
	req := httptest.NewRequest(http.MethodGet, "/api/libraries/7/files", http.NoBody)
	req.SetPathValue("id", "7")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var files []File
	if err := json.NewDecoder(rec.Body).Decode(&files); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(files) != 1 || files[0].Path != "movie.mkv" {
		t.Fatalf("unexpected files: %+v", files)
	}
}

func TestHandlerScansLibraryAndReturnsSummary(t *testing.T) {
	t.Parallel()
	mediaRoot := t.TempDir()
	store := &stubFileStore{reconcileSummary: ReconcileSummary{
		Discovered: 1,
		Added:      1,
	}}
	handler := NewHandler(
		stubLibraryStore{library: libraries.Library{ID: 7, Path: mediaRoot}},
		store,
		mediaRoot,
		func(context.Context, string) (DiscoveryResult, error) {
			return DiscoveryResult{
				Files:   []DiscoveredFile{{Path: "movie.mkv", Size: 100}},
				Ignored: 2,
			}, nil
		},
	)
	req := httptest.NewRequest(http.MethodPost, "/api/libraries/7/scan", http.NoBody)
	req.SetPathValue("id", "7")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var response ScanResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.LibraryID != 7 || response.Added != 1 || response.Ignored != 2 {
		t.Fatalf("unexpected response: %+v", response)
	}
	if response.StartedAt.IsZero() || response.CompletedAt.Before(response.StartedAt) {
		t.Fatalf("unexpected timestamps: %+v", response)
	}
}

func TestHandlerReturns404ForUnknownLibrary(t *testing.T) {
	t.Parallel()
	handler := NewHandler(
		stubLibraryStore{err: libraries.ErrNotFound},
		&stubFileStore{},
		"/media",
		func(context.Context, string) (DiscoveryResult, error) {
			return DiscoveryResult{}, nil
		},
	)
	req := httptest.NewRequest(http.MethodPost, "/api/libraries/99/scan", http.NoBody)
	req.SetPathValue("id", "99")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestHandlerFailedDiscoveryDoesNotReconcile(t *testing.T) {
	t.Parallel()
	mediaRoot := t.TempDir()
	store := &stubFileStore{}
	handler := NewHandler(
		stubLibraryStore{library: libraries.Library{ID: 7, Path: mediaRoot}},
		store,
		mediaRoot,
		func(context.Context, string) (DiscoveryResult, error) {
			return DiscoveryResult{}, errors.New("walk failed")
		},
	)
	req := httptest.NewRequest(http.MethodPost, "/api/libraries/7/scan", http.NoBody)
	req.SetPathValue("id", "7")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
	if store.reconcileCalls != 0 {
		t.Fatalf("expected no reconciliation, got %d calls", store.reconcileCalls)
	}
}

func TestHandlerRejectsConcurrentScanOfSameLibrary(t *testing.T) {
	t.Parallel()
	mediaRoot := t.TempDir()
	started := make(chan struct{})
	release := make(chan struct{})
	handler := NewHandler(
		stubLibraryStore{library: libraries.Library{ID: 7, Path: mediaRoot}},
		&stubFileStore{},
		mediaRoot,
		func(context.Context, string) (DiscoveryResult, error) {
			close(started)
			<-release
			return DiscoveryResult{}, nil
		},
	)

	firstDone := make(chan *httptest.ResponseRecorder)
	go func() {
		req := httptest.NewRequest(http.MethodPost, "/api/libraries/7/scan", http.NoBody)
		req.SetPathValue("id", "7")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		firstDone <- rec
	}()
	<-started

	req := httptest.NewRequest(http.MethodPost, "/api/libraries/7/scan", http.NoBody)
	req.SetPathValue("id", "7")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", rec.Code)
	}

	close(release)
	if first := <-firstDone; first.Code != http.StatusOK {
		t.Fatalf("expected first scan 200, got %d", first.Code)
	}
}
