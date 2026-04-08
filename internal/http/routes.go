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
	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("/api/v1/analyze", AnalyzeHandler)

	// Web routes
	mux.Handle("/", webassets.NewFileHandler())

	return mux
}
