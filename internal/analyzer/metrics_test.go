package analyzer

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
)

func TestAnalyzeLinkMetrics_HeadFallbackToGet(t *testing.T) {
	lookup := func(ctx context.Context, host string) ([]net.IPAddr, error) {
		return []net.IPAddr{{IP: net.ParseIP("8.8.8.8")}}, nil
	}
	client := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.String() == "https://other.com/fallback" && r.Method == http.MethodHead {
				return &http.Response{
					StatusCode: http.StatusMethodNotAllowed,
					Header:     http.Header{"Content-Type": []string{"text/html"}},
					Body:       io.NopCloser(strings.NewReader("")),
				}, nil
			}
			if r.URL.String() == "https://other.com/fallback" && r.Method == http.MethodGet {
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     http.Header{"Content-Type": []string{"text/html"}},
					Body:       io.NopCloser(strings.NewReader("")),
				}, nil
			}
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Header:     http.Header{"Content-Type": []string{"text/html"}},
				Body:       io.NopCloser(strings.NewReader("")),
			}, nil
		}),
	}
	base, _ := url.Parse("https://example.com/page")
	links := []string{"https://other.com/fallback"}

	got, err := generateLinkMetrics(context.Background(), nil, client, lookup, base, links)
	if err != nil {
		t.Fatal(err)
	}
	if got.external != 1 || got.internal != 0 || got.inaccessible != 0 {
		t.Fatalf("metrics = %+v", got)
	}
}

func TestGenerateLinkMetrics_DedupesProbesForDuplicateURLs(t *testing.T) {
	lookup := func(ctx context.Context, host string) ([]net.IPAddr, error) {
		return []net.IPAddr{{IP: net.ParseIP("8.8.8.8")}}, nil
	}
	var requests atomic.Int32
	client := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			requests.Add(1)
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"text/html"}},
				Body:       io.NopCloser(strings.NewReader("")),
			}, nil
		}),
	}
	base, _ := url.Parse("https://example.com/page")
	links := []string{
		"https://dup.example/x",
		"https://dup.example/x",
		"https://dup.example/x",
	}
	got, err := generateLinkMetrics(context.Background(), nil, client, lookup, base, links)
	if err != nil {
		t.Fatal(err)
	}
	if got.external != 3 || got.internal != 0 || got.inaccessible != 0 {
		t.Fatalf("metrics = %+v (want 3 external, 0 inaccessible)", got)
	}
	if n := requests.Load(); n != 1 {
		t.Fatalf("HTTP requests = %d want 1 (single probe for duplicate URL)", n)
	}
}

func TestGenerateLinkMetrics_InaccessibleCountedPerOccurrence(t *testing.T) {
	lookup := func(ctx context.Context, host string) ([]net.IPAddr, error) {
		return []net.IPAddr{{IP: net.ParseIP("8.8.8.8")}}, nil
	}
	var requests atomic.Int32
	client := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			requests.Add(1)
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Header:     http.Header{"Content-Type": []string{"text/html"}},
				Body:       io.NopCloser(strings.NewReader("")),
			}, nil
		}),
	}
	base, _ := url.Parse("https://example.com/page")
	links := []string{
		"https://dead.example/y",
		"https://dead.example/y",
	}
	got, err := generateLinkMetrics(context.Background(), nil, client, lookup, base, links)
	if err != nil {
		t.Fatal(err)
	}
	if got.external != 2 || got.inaccessible != 2 {
		t.Fatalf("metrics = %+v (want 2 external, 2 inaccessible)", got)
	}
	if n := requests.Load(); n != 1 {
		t.Fatalf("HTTP requests = %d want 1", n)
	}
}
