package http

import (
	"net/http"

	webassets "github.com/praminda/link_analyzer/web"
)

// NewRouter creates a new HTTP router for the application.
// It handles both API and web routes. srv must not be nil.
func NewRouter(srv *Server) http.Handler {
	if srv == nil {
		panic("http.NewRouter: nil Server")
	}
	rootMux := http.NewServeMux()
	apiMux := http.NewServeMux()

	// API routes
	apiMux.HandleFunc("POST /api/v1/links/analyze", srv.handleAnalyze)
	apiMux.HandleFunc("GET /api/v1/jobs/{jobId}", srv.handleJobStatus)
	rootMux.Handle("/api/", apiMux)

	// Web routes
	rootMux.Handle("/", webassets.NewFileHandler())

	return rootMux
}
