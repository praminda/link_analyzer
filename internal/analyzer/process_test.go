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

func TestAnalyzeJob_Process_nilReceiver(t *testing.T) {
	var job *AnalyzeJob
	err := job.Process(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAnalyzeJob_Process_rejectsLoopback(t *testing.T) {
	job := &AnalyzeJob{URL: "http://127.0.0.1/"}
	err := job.Process(context.Background())
	if !errors.Is(err, ErrDisallowedHost) {
		t.Fatalf("err = %v, want ErrDisallowedHost", err)
	}
}

func TestAnalyzeJob_Process_successWithMocks(t *testing.T) {
	lookup := func(ctx context.Context, host string) ([]net.IPAddr, error) {
		return []net.IPAddr{{IP: net.ParseIP("8.8.8.8")}}, nil
	}
	client := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			switch r.URL.String() {
			case "https://example.com/page":
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     http.Header{"Content-Type": []string{"text/html"}},
					Body: io.NopCloser(strings.NewReader(
						"<!DOCTYPE html><html><head><title>ok</title></head><body>" +
							"<h1>x</h1>" +
							"<a href=\"/ok\">ok</a>" +
							"<a href=\"https://other.com/bad\">bad</a>" +
							"</body></html>",
					)),
				}, nil
			case "https://example.com/ok":
				if r.Method == http.MethodHead {
					return &http.Response{
						StatusCode: http.StatusOK,
						Header:     http.Header{"Content-Type": []string{"text/html"}},
						Body:       io.NopCloser(strings.NewReader("")),
					}, nil
				}
			case "https://other.com/bad":
				if r.Method == http.MethodHead {
					return &http.Response{
						StatusCode: http.StatusNotFound,
						Header:     http.Header{"Content-Type": []string{"text/html"}},
						Body:       io.NopCloser(strings.NewReader("")),
					}, nil
				}
			}
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Header:     http.Header{"Content-Type": []string{"text/html"}},
				Body:       io.NopCloser(strings.NewReader("unexpected")),
			}, nil
		}),
	}
	job := &AnalyzeJob{
		URL:        "https://example.com/page",
		lookup:     lookup,
		httpClient: client,
	}
	if err := job.Process(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := string(job.RawHTML()); got == "" {
		t.Fatal("RawHTML should not be empty")
	}
	if got := job.Response().PageTitle; got != "ok" {
		t.Fatalf("Response.PageTitle = %q", got)
	}
	if got := job.Response().HeadingCounts.Heading1; got != 1 {
		t.Fatalf("Response.Heading1 = %d", got)
	}
	if got := len(job.ResolvedLinks()); got != 2 {
		t.Fatalf("ResolvedLinks len = %d", got)
	}
	if got := job.Response().InternalLinks; got != 1 {
		t.Fatalf("Response.InternalLinks = %d", got)
	}
	if got := job.Response().ExternalLinks; got != 1 {
		t.Fatalf("Response.ExternalLinks = %d", got)
	}
	if got := job.Response().InaccessibleLinks; got != 1 {
		t.Fatalf("Response.InaccessibleLinks = %d", got)
	}
}
