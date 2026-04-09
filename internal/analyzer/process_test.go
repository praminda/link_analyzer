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
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"text/html"}},
				Body:       io.NopCloser(strings.NewReader("<html>ok</html>")),
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
	if got := string(job.RawHTML()); got != "<html>ok</html>" {
		t.Fatalf("RawHTML = %q", got)
	}
}
