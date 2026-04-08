package http

import (
	"net/http"

	webassets "github.com/praminda/link_analyzer/web"
)

// NewRouter creates a new HTTP router for the application.
// It handles both API and web routes.
//
// Returns:
//   - http.Handler: The HTTP router.
func NewRouter() http.Handler {
	rootMux := http.NewServeMux()
	apiMux := http.NewServeMux()

	// API routes
	apiMux.HandleFunc("POST /api/v1/links/analyze", AnalyzeHandler)
	rootMux.Handle("/api/", apiMux)

	// Web routes
	rootMux.Handle("/", webassets.NewFileHandler())

	return rootMux
}
