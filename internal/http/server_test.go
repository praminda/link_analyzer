package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/praminda/link_analyzer/internal/jobs"
	"github.com/saravanasai/goqueue"
	"github.com/saravanasai/goqueue/config"
)

func newTestServer(t *testing.T) *Server {
	t.Helper()
	cfg := config.NewInMemoryConfig().
		WithMaxRetryAttempts(1).
		WithMaxWorkers(2).
		WithConcurrencyLimit(2)
	qName := strings.ReplaceAll(t.Name(), "/", "-")
	q, err := goqueue.NewQueue(qName, cfg, 200*time.Millisecond)
	if err != nil {
		t.Fatalf("queue: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = q.Shutdown(ctx)
	})
	dbPath := filepath.Join(t.TempDir(), "jobs.sqlite")
	st, err := jobs.NewStore(dbPath)
	if err != nil {
		t.Fatalf("jobs store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return &Server{Queue: q, Jobs: st, WorkerCount: 1}
}

func testAPIHandler(t *testing.T) http.Handler {
	return WithRequestLogging(NewRouter(newTestServer(t)))
}

func TestAnalyzeHandler_InvalidJSONReturnsStructuredError(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/links/analyze", strings.NewReader(`not json`))
	req = req.WithContext(context.WithValue(req.Context(), requestIDContextKey, "test-request-id"))
	rec := httptest.NewRecorder()

	testAPIHandler(t).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
	assertErrorEnvelope(t, rec.Body.Bytes(), "invalid_json_body")
}

func TestAnalyzeHandler_EmptyURLReturnsStructuredError(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/links/analyze", strings.NewReader(`{}`))
	req = req.WithContext(context.WithValue(req.Context(), requestIDContextKey, "test-request-id"))
	rec := httptest.NewRecorder()

	testAPIHandler(t).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
	assertErrorEnvelope(t, rec.Body.Bytes(), "url_required")
}

func TestAnalyzeHandler_InvalidURLReturnsStructuredError(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/links/analyze", strings.NewReader(`{"url":"ftp://example.com"}`))
	req = req.WithContext(context.WithValue(req.Context(), requestIDContextKey, "test-request-id"))
	rec := httptest.NewRecorder()

	testAPIHandler(t).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
	assertErrorEnvelope(t, rec.Body.Bytes(), "url_validation_failed")
}

func TestJobStatusHandler_NotFound(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/does-not-exist", nil)
	req = req.WithContext(context.WithValue(req.Context(), requestIDContextKey, "test-request-id"))
	rec := httptest.NewRecorder()

	testAPIHandler(t).ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d", rec.Code)
	}
	assertErrorEnvelope(t, rec.Body.Bytes(), "job_not_found")
}

func assertErrorEnvelope(t *testing.T, body []byte, wantCode string) {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	errObj, ok := payload["error"].(map[string]any)
	if !ok {
		t.Fatalf("missing error object: %v", payload)
	}
	if errObj["code"] != wantCode {
		t.Fatalf("error.code = %v, want %q", errObj["code"], wantCode)
	}
	if _, ok := errObj["message"].(string); !ok {
		t.Fatalf("error.message not string: %T", errObj["message"])
	}
	if _, ok := errObj["details"]; ok {
		t.Fatalf("error.details should not be present")
	}
}
