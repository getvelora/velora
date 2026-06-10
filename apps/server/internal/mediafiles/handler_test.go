package mediafiles

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
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
	reconciled       []DiscoveredFile
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
	_ context.Context,
	_ int64,
	discovered []DiscoveredFile,
	_ time.Time,
) (ReconcileSummary, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reconcileCalls++
	s.reconciled = append([]DiscoveredFile(nil), discovered...)
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
		func(context.Context, string) (ProbeResult, error) {
			return ProbeResult{}, nil
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
		func(context.Context, string) (ProbeResult, error) {
			return ProbeResult{}, nil
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
		func(context.Context, string) (ProbeResult, error) {
			return ProbeResult{}, nil
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
		func(context.Context, string) (ProbeResult, error) {
			return ProbeResult{}, nil
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
		func(context.Context, string) (ProbeResult, error) {
			return ProbeResult{}, nil
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

func TestHandlerProbesOnlyFilesNeedingInspection(t *testing.T) {
	t.Parallel()
	mediaRoot := t.TempDir()
	modified := time.Date(2026, time.June, 1, 10, 0, 0, 0, time.UTC)
	store := &stubFileStore{files: []File{
		{
			Path:       "ready.mkv",
			Size:       100,
			ModifiedAt: modified,
			Status:     StatusAvailable,
			Inspection: Inspection{Status: InspectionStatusReady},
		},
		{
			Path:       "failed.mkv",
			Size:       100,
			ModifiedAt: modified,
			Status:     StatusAvailable,
			Inspection: Inspection{Status: InspectionStatusError},
		},
		{
			Path:       "changed.mkv",
			Size:       90,
			ModifiedAt: modified,
			Status:     StatusAvailable,
			Inspection: Inspection{Status: InspectionStatusReady},
		},
	}}
	var probed []string
	handler := NewHandler(
		stubLibraryStore{library: libraries.Library{ID: 7, Path: mediaRoot}},
		store,
		mediaRoot,
		func(context.Context, string) (DiscoveryResult, error) {
			return DiscoveryResult{Files: []DiscoveredFile{
				{Path: "ready.mkv", Size: 100, ModifiedAt: modified},
				{Path: "failed.mkv", Size: 100, ModifiedAt: modified},
				{Path: "changed.mkv", Size: 101, ModifiedAt: modified},
				{Path: "new.mkv", Size: 100, ModifiedAt: modified},
			}}, nil
		},
		func(_ context.Context, path string) (ProbeResult, error) {
			probed = append(probed, path)
			return ProbeResult{Format: Format{Name: "matroska,webm"}}, nil
		},
	)

	req := httptest.NewRequest(http.MethodPost, "/api/libraries/7/scan", http.NoBody)
	req.SetPathValue("id", "7")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if len(probed) != 3 {
		t.Fatalf("expected failed, changed, and new files to be probed, got %v", probed)
	}
	for _, item := range store.reconciled {
		if item.Path == "ready.mkv" && item.Inspection.Attempted {
			t.Fatal("expected ready unchanged file to retain inspection")
		}
		if item.Path != "ready.mkv" && !item.Inspection.Attempted {
			t.Fatalf("expected %s to be inspected", item.Path)
		}
	}
}

func TestHandlerRecordsProbeFailureAndContinues(t *testing.T) {
	t.Parallel()
	mediaRoot := t.TempDir()
	store := &stubFileStore{reconcileSummary: ReconcileSummary{
		Discovered:  1,
		Added:       1,
		Probed:      1,
		ProbeFailed: 1,
	}}
	handler := NewHandler(
		stubLibraryStore{library: libraries.Library{ID: 7, Path: mediaRoot}},
		store,
		mediaRoot,
		func(context.Context, string) (DiscoveryResult, error) {
			return DiscoveryResult{Files: []DiscoveredFile{{Path: "broken.mkv"}}}, nil
		},
		func(context.Context, string) (ProbeResult, error) {
			return ProbeResult{}, errors.New(mediaRoot + "/broken.mkv: invalid data\nmore detail")
		},
	)

	req := httptest.NewRequest(http.MethodPost, "/api/libraries/7/scan", http.NoBody)
	req.SetPathValue("id", "7")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if store.reconcileCalls != 1 || len(store.reconciled) != 1 {
		t.Fatalf("expected reconciliation, got %+v", store)
	}
	got := store.reconciled[0].Inspection
	if !got.Attempted || got.Error == "" ||
		strings.Contains(got.Error, mediaRoot) ||
		!strings.Contains(got.Error, "broken.mkv") {
		t.Fatalf("unexpected stored probe error %q", got.Error)
	}
}

func TestHandlerRuntimeProbeFailureDoesNotReconcile(t *testing.T) {
	t.Parallel()
	mediaRoot := t.TempDir()
	store := &stubFileStore{}
	handler := NewHandler(
		stubLibraryStore{library: libraries.Library{ID: 7, Path: mediaRoot}},
		store,
		mediaRoot,
		func(context.Context, string) (DiscoveryResult, error) {
			return DiscoveryResult{Files: []DiscoveredFile{{Path: "movie.mkv"}}}, nil
		},
		func(context.Context, string) (ProbeResult, error) {
			return ProbeResult{}, fmt.Errorf("%w: executable missing", ErrProbeRuntime)
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

func TestHandlerCanceledProbeDoesNotReconcile(t *testing.T) {
	t.Parallel()
	mediaRoot := t.TempDir()
	store := &stubFileStore{}
	started := make(chan struct{})
	handler := NewHandler(
		stubLibraryStore{library: libraries.Library{ID: 7, Path: mediaRoot}},
		store,
		mediaRoot,
		func(context.Context, string) (DiscoveryResult, error) {
			return DiscoveryResult{Files: []DiscoveredFile{{Path: "movie.mkv"}}}, nil
		},
		func(ctx context.Context, _ string) (ProbeResult, error) {
			close(started)
			<-ctx.Done()
			return ProbeResult{}, ctx.Err()
		},
	)

	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest(http.MethodPost, "/api/libraries/7/scan", http.NoBody).
		WithContext(ctx)
	req.SetPathValue("id", "7")
	rec := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		handler.ServeHTTP(rec, req)
		close(done)
	}()

	<-started
	cancel()
	<-done

	if store.reconcileCalls != 0 {
		t.Fatalf("expected no reconciliation, got %d calls", store.reconcileCalls)
	}
}
