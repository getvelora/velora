package libraries

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type stubStore struct {
	libraries []Library
	listErr   error
	createErr error
}

func (s *stubStore) List(ctx context.Context) ([]Library, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	if s.libraries == nil {
		return []Library{}, nil
	}
	return s.libraries, nil
}

func (s *stubStore) Create(ctx context.Context, name, path string) (Library, error) {
	if s.createErr != nil {
		return Library{}, s.createErr
	}
	now := time.Now().UTC()
	lib := Library{
		ID:        int64(len(s.libraries) + 1),
		Name:      name,
		Path:      path,
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.libraries = append(s.libraries, lib)
	return lib, nil
}

func (s *stubStore) Get(ctx context.Context, id int64) (Library, error) {
	for _, library := range s.libraries {
		if library.ID == id {
			return library, nil
		}
	}
	return Library{}, ErrNotFound
}

func TestHandlerGetReturnsLibrariesAsArray(t *testing.T) {
	t.Parallel()

	store := &stubStore{libraries: []Library{{ID: 1, Name: "Movies", Path: "/media/movies"}}}
	rec := callHandler(store, http.MethodGet, "")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected Content-Type application/json, got %q", got)
	}
	var body []Library
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if len(body) != 1 || body[0].Name != "Movies" {
		t.Fatalf("unexpected body: %+v", body)
	}
}

func TestHandlerGetEmptyReturnsEmptyArrayNotNull(t *testing.T) {
	t.Parallel()

	rec := callHandler(&stubStore{}, http.MethodGet, "")

	body := strings.TrimSpace(rec.Body.String())
	if body != "[]" {
		t.Fatalf("expected empty array body, got %q", body)
	}
}

func TestHandlerPostCreatesLibraryAndReturns201(t *testing.T) {
	t.Parallel()

	store := &stubStore{}
	rec := callHandler(store, http.MethodPost, `{"name":"Movies","path":"/media/movies"}`)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", rec.Code, rec.Body.String())
	}
	var body Library
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.ID == 0 || body.Name != "Movies" || body.Path != "/media/movies" {
		t.Fatalf("unexpected body: %+v", body)
	}
	if len(store.libraries) != 1 {
		t.Fatalf("expected one stored library, got %d", len(store.libraries))
	}
}

func TestHandlerPostMissingFieldsReturns400(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"empty name": `{"name":"","path":"/media/movies"}`,
		"empty path": `{"name":"Movies","path":""}`,
		"only name":  `{"name":"Movies"}`,
		"junk":       `not json`,
	}
	for label, body := range cases {
		t.Run(label, func(t *testing.T) {
			rec := callHandler(&stubStore{}, http.MethodPost, body)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestHandlerPostRejectsPathOutsideMediaRoot(t *testing.T) {
	t.Parallel()

	rec := callHandler(
		&stubStore{},
		http.MethodPost,
		`{"name":"Movies","path":"/etc/movies"}`,
	)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandlerPostCleansContainedPath(t *testing.T) {
	t.Parallel()

	store := &stubStore{}
	rec := callHandler(
		store,
		http.MethodPost,
		`{"name":"Movies","path":"/media/./movies"}`,
	)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", rec.Code, rec.Body.String())
	}
	if store.libraries[0].Path != "/media/movies" {
		t.Fatalf("expected clean path, got %q", store.libraries[0].Path)
	}
}

func TestHandlerPostDuplicatePathReturns409(t *testing.T) {
	t.Parallel()

	store := &stubStore{createErr: ErrPathAlreadyExists}
	rec := callHandler(store, http.MethodPost, `{"name":"Movies","path":"/media/movies"}`)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", rec.Code)
	}
}

func TestHandlerPostStoreErrorReturns500(t *testing.T) {
	t.Parallel()

	store := &stubStore{createErr: errors.New("disk on fire")}
	rec := callHandler(store, http.MethodPost, `{"name":"Movies","path":"/media/movies"}`)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestHandlerDisallowsOtherMethods(t *testing.T) {
	t.Parallel()

	rec := callHandler(&stubStore{}, http.MethodDelete, "")
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
	if got := rec.Header().Get("Allow"); got != "GET, POST" {
		t.Fatalf("expected Allow header, got %q", got)
	}
}

func callHandler(store Store, method, body string) *httptest.ResponseRecorder {
	var reqBody *strings.Reader
	if body != "" {
		reqBody = strings.NewReader(body)
	}
	var req *http.Request
	if reqBody != nil {
		req = httptest.NewRequest(method, "/api/libraries", reqBody)
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, "/api/libraries", http.NoBody)
	}
	rec := httptest.NewRecorder()
	NewHandler(store, "/media").ServeHTTP(rec, req)
	return rec
}
