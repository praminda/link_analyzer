package analyzer

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
)

func TestAnalyzeJob_Process_InvalidURLReturnsAnalyzeError(t *testing.T) {
	job := &AnalyzeJob{URL: "ftp://example.com"}
	err := job.Process(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	var ae *AnalyzeError
	if !errors.As(err, &ae) {
		t.Fatalf("expected AnalyzeError, got %T", err)
	}
	if ae.HTTPStatus != http.StatusBadRequest {
		t.Fatalf("status = %d", ae.HTTPStatus)
	}
	if ae.Code != "url_validation_failed" {
		t.Fatalf("code = %q", ae.Code)
	}
}

func TestAnalyzeJob_Process_UpstreamStatusDetails(t *testing.T) {
	lookup := func(ctx context.Context, host string) ([]net.IPAddr, error) {
		return []net.IPAddr{{IP: net.ParseIP("8.8.8.8")}}, nil
	}
	client := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Header:     http.Header{"Content-Type": []string{"text/html"}},
				Body:       io.NopCloser(strings.NewReader("not found")),
			}, nil
		}),
	}
	job := &AnalyzeJob{
		URL:        "https://example.com",
		lookup:     lookup,
		httpClient: client,
	}
	err := job.Process(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	var ae *AnalyzeError
	if !errors.As(err, &ae) {
		t.Fatalf("expected AnalyzeError, got %T", err)
	}
	if ae.HTTPStatus != http.StatusBadGateway {
		t.Fatalf("status = %d", ae.HTTPStatus)
	}
	if ae.Code != "fetch_http_404" {
		t.Fatalf("code = %q", ae.Code)
	}
}
