package analyzer

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/praminda/link_analyzer/internal/appconfig"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestFetchHTML_nilClient(t *testing.T) {
	u, _ := url.Parse("https://example.com/")
	_, err := fetchHTML(context.Background(), nil, u, 1024, appconfig.DefaultAnalyzer.UserAgent)
	if !errors.Is(err, ErrNilHTTPClient) {
		t.Fatalf("err = %v, want ErrNilHTTPClient", err)
	}
}

func TestFetchHTML_nonOKStatus(t *testing.T) {
	client := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Header:     http.Header{"Content-Type": []string{"text/html"}},
				Body:       io.NopCloser(strings.NewReader("")),
			}, nil
		}),
	}
	u, _ := url.Parse("https://example.com/")
	_, err := fetchHTML(context.Background(), client, u, 1024, appconfig.DefaultAnalyzer.UserAgent)
	if !errors.Is(err, ErrFetchStatus) {
		t.Fatalf("err = %v, want ErrFetchStatus", err)
	}
}

func TestFetchHTML_ok(t *testing.T) {
	client := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.Method != http.MethodGet {
				t.Fatalf("method = %q", r.Method)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"text/html; charset=utf-8"}},
				Body:       io.NopCloser(strings.NewReader("<html></html>")),
			}, nil
		}),
	}
	u, err := url.Parse("https://example.com/")
	if err != nil {
		t.Fatal(err)
	}
	body, err := fetchHTML(context.Background(), client, u, 1024, appconfig.DefaultAnalyzer.UserAgent)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "<html></html>" {
		t.Fatalf("body = %q", body)
	}
}

func TestFetchHTML_notHTML(t *testing.T) {
	client := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader("{}")),
			}, nil
		}),
	}
	u, _ := url.Parse("https://example.com/")
	_, err := fetchHTML(context.Background(), client, u, 1024, appconfig.DefaultAnalyzer.UserAgent)
	if !errors.Is(err, ErrNotHTML) {
		t.Fatalf("err = %v, want ErrNotHTML", err)
	}
}

func TestFetchHTML_tooLarge(t *testing.T) {
	client := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"text/html"}},
				Body:       io.NopCloser(strings.NewReader(strings.Repeat("a", 20))),
			}, nil
		}),
	}
	u, _ := url.Parse("https://example.com/")
	_, err := fetchHTML(context.Background(), client, u, 10, appconfig.DefaultAnalyzer.UserAgent)
	if !errors.Is(err, ErrBodyTooLarge) {
		t.Fatalf("err = %v, want ErrBodyTooLarge", err)
	}
}

func TestIsHTMLContentType(t *testing.T) {
	if !isHTMLContentType("") {
		t.Fatal("empty should be allowed")
	}
	if !isHTMLContentType("text/html") {
		t.Fatal("text/html")
	}
	if !isHTMLContentType("TEXT/HTML ; charset=utf-8") {
		t.Fatal("with charset")
	}
	if isHTMLContentType("application/json") {
		t.Fatal("json not html")
	}
}
