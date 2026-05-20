// Package web serves the bundled Velora web client.
package web

import (
	"net/http"
	"path/filepath"
)

// NewSPAHandler returns an http.Handler that serves the built web client from
// distDir. Requests for paths with file extensions go through the file server
// (so missing assets correctly 404). Requests without an extension fall back
// to index.html so client-side SPA routes resolve on deep-link refresh.
func NewSPAHandler(distDir string) http.Handler {
	fileServer := http.FileServer(http.Dir(distDir))
	indexPath := filepath.Join(distDir, "index.html")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if filepath.Ext(r.URL.Path) != "" {
			fileServer.ServeHTTP(w, r)
			return
		}
		http.ServeFile(w, r, indexPath)
	})
}
