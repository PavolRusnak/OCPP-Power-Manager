package httpapi

import (
	"embed"
	"io/fs"
	"net/http"
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

	return http.FileServer(http.FS(fsys))
}
