package libraries

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/getvelora/velora/apps/server/internal/storage"
)

// Store is the persistence contract the handler depends on. Repository
// satisfies it; tests can substitute an in-memory stub.
type Store interface {
	List(ctx context.Context) ([]Library, error)
	Create(ctx context.Context, name, path string) (Library, error)
}

type createRequest struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// NewHandler returns an http.Handler serving /api/libraries. GET lists the
// configured libraries; POST creates a new one. Other methods return 405.
func NewHandler(store Store, mediaRoot string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleList(store, w, r)
		case http.MethodPost:
			handleCreate(store, mediaRoot, w, r)
		default:
			w.Header().Set("Allow", "GET, POST")
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	})
}

func handleList(store Store, w http.ResponseWriter, r *http.Request) {
	items, err := store.List(r.Context())
	if err != nil {
		log.Printf("libraries: list failed: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to list libraries")
		return
	}
	// Defensive: a nil slice serializes as `null`, but the API contract is a
	// JSON array. Repository.List already returns []Library{}, but coerce
	// here so any Store implementation is safe to use.
	if items == nil {
		items = []Library{}
	}
	writeJSON(w, http.StatusOK, items)
}

func handleCreate(store Store, mediaRoot string, w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Path = strings.TrimSpace(req.Path)
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.Path == "" {
		writeError(w, http.StatusBadRequest, "path is required")
		return
	}
	if err := storage.ValidateMediaPath(mediaRoot, req.Path); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	req.Path = filepath.Clean(req.Path)

	lib, err := store.Create(r.Context(), req.Name, req.Path)
	if err != nil {
		if errors.Is(err, ErrPathAlreadyExists) {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		log.Printf("libraries: create failed: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to create library")
		return
	}
	writeJSON(w, http.StatusCreated, lib)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
