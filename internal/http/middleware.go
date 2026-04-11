package http

import (
	"context"
	"log/slog"
	stdhttp "net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

type contextKey string

const requestIDContextKey contextKey = "request_id"

type statusRecorder struct {
	stdhttp.ResponseWriter
	statusCode int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

func WithRequestLogging(next stdhttp.Handler) stdhttp.Handler {
	return stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {

		// we don't want to log web requests. only logging API requests
		if !strings.HasPrefix(r.URL.Path, "/api") {
			next.ServeHTTP(w, r)
			return
		}
		start := time.Now()
		// For log correlation if needed in the future or after deployment
		requestID := r.Header.Get("X-Request-Id")
		if requestID == "" {
			requestID = newRequestID()
		}

		w.Header().Set("X-Request-Id", requestID)
		ctx := context.WithValue(r.Context(), requestIDContextKey, requestID)
		r = r.WithContext(ctx)

		rec := &statusRecorder{
			ResponseWriter: w,
			statusCode:     stdhttp.StatusOK,
		}

		next.ServeHTTP(rec, r)

		slog.Debug("http_request",
			"request_id", requestID,
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.statusCode,
			"duration_ms", time.Since(start).Milliseconds(),
			"remote_addr", r.RemoteAddr,
		)
	})
}

func newRequestID() string {
	reqId, err := uuid.NewRandom()
	if err != nil {
		// This is not critical. Defaulting and no need to return and error
		// from the request for this scenario
		return "request-id-unavailable"
	}
	return reqId.String()
}
