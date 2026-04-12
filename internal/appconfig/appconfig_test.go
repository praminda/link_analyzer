package appconfig

import (
	"testing"
	"time"
)

func TestLoad_EnvOverride(t *testing.T) {
	t.Setenv("HTTP_ADDR", ":0")
	t.Setenv("JOB_DB_PATH", "/tmp/x.sqlite")
	t.Setenv("QUEUE_NAME", "q-test")
	t.Setenv("QUEUE_MAX_RETRY_ATTEMPTS", "2")
	t.Setenv("QUEUE_MAX_WORKERS", "3")
	t.Setenv("QUEUE_CONCURRENCY_LIMIT", "6")
	t.Setenv("QUEUE_WORKER_COUNT", "2")
	t.Setenv("ANALYZER_MAX_BODY_BYTES", "1048576")
	t.Setenv("ANALYZER_FETCH_TIMEOUT_SEC", "45")
	t.Setenv("ANALYZER_MAX_REDIRECTS", "3")
	t.Setenv("ANALYZER_USER_AGENT", "LinkAnalyzerTest/1.0")
	t.Setenv("ANALYZER_LINK_CHECK_WORKERS", "7")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("APP_ENV", "production")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HTTP.Addr != ":0" {
		t.Fatalf("HTTP.Addr = %q", cfg.HTTP.Addr)
	}
	if cfg.Jobs.DBPath != "/tmp/x.sqlite" {
		t.Fatalf("Jobs.DBPath = %q", cfg.Jobs.DBPath)
	}
	if cfg.Queue.Name != "q-test" || cfg.Queue.MaxRetryAttempts != 2 {
		t.Fatalf("Queue: %+v", cfg.Queue)
	}
	want := AnalyzerConfig{
		MaxBodyBytes:     1048576,
		FetchTimeout:     45 * time.Second,
		MaxRedirects:     3,
		UserAgent:        "LinkAnalyzerTest/1.0",
		LinkCheckWorkers: 7,
	}
	if cfg.Analyzer != want {
		t.Fatalf("Analyzer: %+v, want %+v", cfg.Analyzer, want)
	}
	if cfg.Log.Level != "debug" || !cfg.Log.UseJSON {
		t.Fatalf("Log: %+v", cfg.Log)
	}
}

func TestResolveFetch_nilUsesDefaults(t *testing.T) {
	got := ResolveFetch(nil)
	want := FetchLimits{
		Timeout:            DefaultAnalyzer.FetchTimeout,
		MaxRedirects:       DefaultAnalyzer.MaxRedirects,
		MaxBodyBytes:       DefaultAnalyzer.MaxBodyBytes,
		UserAgent:          DefaultAnalyzer.UserAgent,
		LinkCheckWorkers:   DefaultAnalyzer.LinkCheckWorkers,
	}
	if got != want {
		t.Fatalf("ResolveFetch(nil) = %+v, want %+v", got, want)
	}
}

func TestResolveFetch_overrideDefault(t *testing.T) {
	got := ResolveFetch(&AnalyzerConfig{
		FetchTimeout: 99 * time.Second,
		MaxRedirects: 0,
		MaxBodyBytes: 100,
		UserAgent:    "",
	})
	if got.Timeout != 99*time.Second {
		t.Fatalf("Timeout = %v", got.Timeout)
	}
	if got.MaxRedirects != DefaultAnalyzer.MaxRedirects {
		t.Fatalf("MaxRedirects = %d, want default %d", got.MaxRedirects, DefaultAnalyzer.MaxRedirects)
	}
	if got.MaxBodyBytes != 100 {
		t.Fatalf("MaxBodyBytes = %d", got.MaxBodyBytes)
	}
	if got.UserAgent != DefaultAnalyzer.UserAgent {
		t.Fatalf("UserAgent not defaulted")
	}
	if got.LinkCheckWorkers != DefaultAnalyzer.LinkCheckWorkers {
		t.Fatalf("LinkCheckWorkers = %d, want default %d", got.LinkCheckWorkers, DefaultAnalyzer.LinkCheckWorkers)
	}
}

func TestLoad_LinkCheckWorkersAbove256Errors(t *testing.T) {
	t.Setenv("ANALYZER_LINK_CHECK_WORKERS", "300")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLoad_WorkerCountAboveMaxWorkersErrors(t *testing.T) {
	t.Setenv("QUEUE_MAX_WORKERS", "2")
	t.Setenv("QUEUE_WORKER_COUNT", "5")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error")
	}
}
