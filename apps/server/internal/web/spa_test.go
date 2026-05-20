package web

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestSPAHandlerServesIndexForRoutePaths(t *testing.T) {
	dir := writeFixture(t, map[string]string{
		"index.html": "<html>app</html>",
	})

	rec := callHandler(t, dir, "/libraries/123")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for SPA route, got %d", rec.Code)
	}
	if got := rec.Body.String(); got != "<html>app</html>" {
		t.Fatalf("expected index.html body, got %q", got)
	}
}

func TestSPAHandlerServesAssetWhenExtensionMatches(t *testing.T) {
	dir := writeFixture(t, map[string]string{
		"index.html":    "<html>app</html>",
		"assets/app.js": "console.log(1)",
	})

	rec := callHandler(t, dir, "/assets/app.js")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if got := rec.Body.String(); got != "console.log(1)" {
		t.Fatalf("expected asset body, got %q", got)
	}
}

func TestSPAHandlerReturns404ForMissingAsset(t *testing.T) {
	dir := writeFixture(t, map[string]string{
		"index.html": "<html>app</html>",
	})

	rec := callHandler(t, dir, "/missing.js")

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for missing asset, got %d", rec.Code)
	}
}

func writeFixture(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, body := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	return dir
}

func callHandler(t *testing.T, dir, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	NewSPAHandler(dir).ServeHTTP(rec, req)
	return rec
}
