package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAnalyzeHandler_InvalidURLReturnsStructuredError(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/links/analyze", strings.NewReader(`{"url":"ftp://example.com"}`))
	req = req.WithContext(context.WithValue(req.Context(), requestIDContextKey, "test-request-id"))
	rec := httptest.NewRecorder()

	AnalyzeHandler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	errObj, ok := payload["error"].(map[string]any)
	if !ok {
		t.Fatalf("missing error object: %v", payload)
	}
	if errObj["code"] != "url_validation_failed" {
		t.Fatalf("error.code = %v", errObj["code"])
	}
	if _, ok := errObj["message"].(string); !ok {
		t.Fatalf("error.message not string: %T", errObj["message"])
	}
	if _, ok := errObj["details"]; ok {
		t.Fatalf("error.details should not be present")
	}
}
