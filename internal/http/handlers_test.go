package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAnalyzeHandler_InvalidJSONReturnsStructuredError(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/links/analyze", strings.NewReader(`not json`))
	req = req.WithContext(context.WithValue(req.Context(), requestIDContextKey, "test-request-id"))
	rec := httptest.NewRecorder()

	AnalyzeHandler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
	assertErrorEnvelope(t, rec.Body.Bytes(), "invalid_json_body")
}

func TestAnalyzeHandler_EmptyURLReturnsStructuredError(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/links/analyze", strings.NewReader(`{}`))
	req = req.WithContext(context.WithValue(req.Context(), requestIDContextKey, "test-request-id"))
	rec := httptest.NewRecorder()

	AnalyzeHandler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
	assertErrorEnvelope(t, rec.Body.Bytes(), "url_required")
}

func TestAnalyzeHandler_InvalidURLReturnsStructuredError(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/links/analyze", strings.NewReader(`{"url":"ftp://example.com"}`))
	req = req.WithContext(context.WithValue(req.Context(), requestIDContextKey, "test-request-id"))
	rec := httptest.NewRecorder()

	AnalyzeHandler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
	assertErrorEnvelope(t, rec.Body.Bytes(), "url_validation_failed")
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
