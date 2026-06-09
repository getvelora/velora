package mediafiles

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/getvelora/velora/apps/server/internal/libraries"
	"github.com/getvelora/velora/apps/server/internal/storage"
)

type LibraryStore interface {
	Get(ctx context.Context, id int64) (libraries.Library, error)
}

type Store interface {
	List(ctx context.Context, libraryID int64) ([]File, error)
	Reconcile(
		ctx context.Context,
		libraryID int64,
		discovered []DiscoveredFile,
		scannedAt time.Time,
	) (ReconcileSummary, error)
}

type DiscoverFunc func(ctx context.Context, root string) (DiscoveryResult, error)

type ScanResponse struct {
	LibraryID int64 `json:"libraryId"`
	ReconcileSummary
	Ignored     int       `json:"ignored"`
	StartedAt   time.Time `json:"startedAt"`
	CompletedAt time.Time `json:"completedAt"`
}

type Handler struct {
	libraries LibraryStore
	files     Store
	mediaRoot string
	discover  DiscoverFunc

	mu      sync.Mutex
	running map[int64]struct{}
}

func NewHandler(
	libraryStore LibraryStore,
	fileStore Store,
	mediaRoot string,
	discover DiscoverFunc,
) *Handler {
	return &Handler{
		libraries: libraryStore,
		files:     fileStore,
		mediaRoot: mediaRoot,
		discover:  discover,
		running:   map[int64]struct{}{},
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	libraryID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || libraryID <= 0 {
		writeMediaError(w, http.StatusNotFound, "library not found")
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.handleList(w, r, libraryID)
	case http.MethodPost:
		h.handleScan(w, r, libraryID)
	default:
		w.Header().Set("Allow", "GET, POST")
		writeMediaError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *Handler) handleList(w http.ResponseWriter, r *http.Request, libraryID int64) {
	if _, ok := h.getLibrary(w, r, libraryID); !ok {
		return
	}
	files, err := h.files.List(r.Context(), libraryID)
	if err != nil {
		log.Printf("media files: list library %d failed: %v", libraryID, err)
		writeMediaError(w, http.StatusInternalServerError, "failed to list media files")
		return
	}
	if files == nil {
		files = []File{}
	}
	writeMediaJSON(w, http.StatusOK, files)
}

func (h *Handler) handleScan(w http.ResponseWriter, r *http.Request, libraryID int64) {
	library, ok := h.getLibrary(w, r, libraryID)
	if !ok {
		return
	}
	if err := storage.ValidateMediaPath(h.mediaRoot, library.Path); err != nil {
		log.Printf("media files: unsafe library %d path %q: %v", libraryID, library.Path, err)
		writeMediaError(w, http.StatusInternalServerError, "library path is invalid")
		return
	}
	if err := storage.ValidateResolvedMediaPath(h.mediaRoot, library.Path); err != nil {
		log.Printf("media files: unavailable library %d path %q: %v", libraryID, library.Path, err)
		writeMediaError(w, http.StatusInternalServerError, "library path is unavailable")
		return
	}
	if !h.startScan(libraryID) {
		writeMediaError(w, http.StatusConflict, "library scan already in progress")
		return
	}
	defer h.finishScan(libraryID)

	startedAt := time.Now().UTC()
	discovery, err := h.discover(r.Context(), library.Path)
	if err != nil {
		log.Printf("media files: scan library %d failed: %v", libraryID, err)
		writeMediaError(w, http.StatusInternalServerError, "failed to scan library")
		return
	}
	summary, err := h.files.Reconcile(
		r.Context(),
		libraryID,
		discovery.Files,
		startedAt,
	)
	if err != nil {
		log.Printf("media files: reconcile library %d failed: %v", libraryID, err)
		writeMediaError(w, http.StatusInternalServerError, "failed to save library scan")
		return
	}

	writeMediaJSON(w, http.StatusOK, ScanResponse{
		LibraryID:        libraryID,
		ReconcileSummary: summary,
		Ignored:          discovery.Ignored,
		StartedAt:        startedAt,
		CompletedAt:      time.Now().UTC(),
	})
}

func (h *Handler) getLibrary(
	w http.ResponseWriter,
	r *http.Request,
	libraryID int64,
) (libraries.Library, bool) {
	library, err := h.libraries.Get(r.Context(), libraryID)
	if errors.Is(err, libraries.ErrNotFound) {
		writeMediaError(w, http.StatusNotFound, "library not found")
		return libraries.Library{}, false
	}
	if err != nil {
		log.Printf("media files: get library %d failed: %v", libraryID, err)
		writeMediaError(w, http.StatusInternalServerError, "failed to load library")
		return libraries.Library{}, false
	}
	return library, true
}

func (h *Handler) startScan(libraryID int64) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, running := h.running[libraryID]; running {
		return false
	}
	h.running[libraryID] = struct{}{}
	return true
}

func (h *Handler) finishScan(libraryID int64) {
	h.mu.Lock()
	delete(h.running, libraryID)
	h.mu.Unlock()
}

func writeMediaJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeMediaError(w http.ResponseWriter, status int, message string) {
	writeMediaJSON(w, status, map[string]string{"error": message})
}
