package webassets

import (
	"embed"
	"net/http"
)

// Files contains frontend static assets embedded into the Go binary.
//
//go:embed index.html app.js
var Files embed.FS

// NewFileHandler creates a new HTTP handler for serving the web assets.
//
// Returns:
//   - http.Handler: The HTTP handler.
func NewFileHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			data, err := Files.ReadFile("index.html")
			if err != nil {
				http.Error(w, "failed to load web page", http.StatusInternalServerError)
				return
			}
			_, _ = w.Write(data)
			return
		}

		http.FileServer(http.FS(Files)).ServeHTTP(w, r)
	})
}
