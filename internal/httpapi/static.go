package httpapi

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// StaticHandler serves static files from the filesystem
func StaticHandler() http.Handler {
	// Create a custom handler that serves files from the static directory
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if the requested file exists
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}

		// Build the full file path
		fullPath := filepath.Join("internal", "httpapi", "static", path)

		// Check if file exists
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			// If file doesn't exist, serve index.html for SPA routing
			indexPath := filepath.Join("internal", "httpapi", "static", "index.html")
			if _, err := os.Stat(indexPath); os.IsNotExist(err) {
				http.NotFound(w, r)
				return
			}
			http.ServeFile(w, r, indexPath)
			return
		}

		// File exists, serve it normally
		http.ServeFile(w, r, fullPath)
	})
}
