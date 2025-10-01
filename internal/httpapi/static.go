package httpapi

import (
	"embed"
	"io"
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

//go:embed static/*
var staticFiles embed.FS

// StaticHandler serves the embedded static files
func StaticHandler() http.Handler {
	// Get the embedded filesystem
	fsys, err := fs.Sub(staticFiles, "static")
	if err != nil {
		panic(err)
	}

	// Create a custom handler that serves index.html for SPA routes
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if the requested file exists
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}

		// Try to open the file
		file, err := fsys.Open(path)
		if err != nil {
			// If file doesn't exist, serve index.html for SPA routing
			indexFile, err := fsys.Open("index.html")
			if err != nil {
				http.NotFound(w, r)
				return
			}
			defer indexFile.Close()

			// Set content type
			w.Header().Set("Content-Type", "text/html")
			http.ServeContent(w, r, "index.html", time.Time{}, indexFile.(io.ReadSeeker))
			return
		}
		defer file.Close()

		// File exists, serve it normally
		http.ServeContent(w, r, filepath.Base(path), time.Time{}, file.(io.ReadSeeker))
	})
}
