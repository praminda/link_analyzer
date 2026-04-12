// Package appconfig loads process-wide settings from environment variables with defaults.
package appconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Config holds settings for HTTP, persistence, queue, analyzer, and logging.
type Config struct {
	HTTP     HTTPConfig
	Jobs     JobsConfig
	Queue    QueueConfig
	Analyzer AnalyzerConfig
	Log      LogConfig
}

// HTTPConfig is the API and embedded static file server.
type HTTPConfig struct {
	// Addr is the TCP listen address (e.g. ":8080", "127.0.0.1:3000").
	Addr string
}

// JobsConfig is the SQLite job store.
type JobsConfig struct {
	// DBPath is the path to the SQLite database file.
	DBPath string
}

// QueueConfig is the in-process goqueue instance.
type QueueConfig struct {
	// Name is the logical queue name passed to goqueue.
	Name             string
	MaxRetryAttempts int
	MaxWorkers       int
	ConcurrencyLimit int
	// WorkerCount is how many worker goroutines StartWorkers uses (lazy on first job).
	WorkerCount int
}

// AnalyzerConfig controls HTTP fetch behavior for link analysis jobs
type AnalyzerConfig struct {
	MaxBodyBytes int64         `json:"max_body_bytes,omitempty"`
	FetchTimeout time.Duration `json:"fetch_timeout,omitempty"`
	MaxRedirects int           `json:"max_redirects,omitempty"`
	UserAgent    string        `json:"user_agent,omitempty"`
}

// DefaultAnalyzer is the baseline when env is unset or when a job has no overrides.
var DefaultAnalyzer = AnalyzerConfig{
	MaxBodyBytes: 2 << 20,
	FetchTimeout: 30 * time.Second,
	MaxRedirects: 5,
	UserAgent:    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.14; rv:145.0) Gecko/20090101 Firefox/145.0",
}

// FetchLimits is concrete HTTP fetch tuning parameters.
type FetchLimits struct {
	Timeout      time.Duration
	MaxRedirects int
	MaxBodyBytes int64
	UserAgent    string
}

// ResolveFetch overlays optional job-level *AnalyzerConfig on [DefaultAnalyzer].
// Nil c or zero fields in c keep the baseline value for that field.
func ResolveFetch(c *AnalyzerConfig) FetchLimits {
	cfg := DefaultAnalyzer
	if c != nil {
		if c.FetchTimeout > 0 {
			cfg.FetchTimeout = c.FetchTimeout
		}
		if c.MaxRedirects > 0 {
			cfg.MaxRedirects = c.MaxRedirects
		}
		if c.MaxBodyBytes > 0 {
			cfg.MaxBodyBytes = c.MaxBodyBytes
		}
		if ua := strings.TrimSpace(c.UserAgent); ua != "" {
			cfg.UserAgent = ua
		}
	}
	return FetchLimits{
		Timeout:      cfg.FetchTimeout,
		MaxRedirects: cfg.MaxRedirects,
		MaxBodyBytes: cfg.MaxBodyBytes,
		UserAgent:    cfg.UserAgent,
	}
}

// LogConfig selects slog output shape and level.
type LogConfig struct {
	Level   string
	UseJSON bool
}

// Load reads configuration from the environment. Missing values use documented defaults.
func Load() (*Config, error) {
	cfg := &Config{
		HTTP: HTTPConfig{
			Addr: getenv("HTTP_ADDR", ":8080"),
		},
		Jobs: JobsConfig{
			DBPath: getenv("JOB_DB_PATH", filepath.Join("data", "jobs.sqlite")),
		},
		Queue: QueueConfig{
			Name:             getenv("QUEUE_NAME", "link-analyze"),
			MaxRetryAttempts: getenvInt("QUEUE_MAX_RETRY_ATTEMPTS", 1),
			MaxWorkers:       getenvInt("QUEUE_MAX_WORKERS", 4),
			ConcurrencyLimit: getenvInt("QUEUE_CONCURRENCY_LIMIT", 8),
			WorkerCount:      getenvInt("QUEUE_WORKER_COUNT", 2),
		},
		Analyzer: AnalyzerConfig{
			MaxBodyBytes: getenvInt64("ANALYZER_MAX_BODY_BYTES", DefaultAnalyzer.MaxBodyBytes),
			FetchTimeout: time.Duration(getenvInt("ANALYZER_FETCH_TIMEOUT_SEC", int(DefaultAnalyzer.FetchTimeout.Seconds()))) * time.Second,
			MaxRedirects: getenvInt("ANALYZER_MAX_REDIRECTS", DefaultAnalyzer.MaxRedirects),
			UserAgent:    getenv("ANALYZER_USER_AGENT", DefaultAnalyzer.UserAgent),
		},
		Log: LogConfig{
			Level:   getenv("LOG_LEVEL", "info"),
			UseJSON: strings.EqualFold(strings.TrimSpace(os.Getenv("APP_ENV")), "production"),
		},
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) validate() error {
	if c.HTTP.Addr == "" {
		return fmt.Errorf("HTTP_ADDR must not be empty")
	}
	if c.Jobs.DBPath == "" {
		return fmt.Errorf("JOB_DB_PATH must not be empty")
	}
	if c.Queue.Name == "" {
		return fmt.Errorf("QUEUE_NAME must not be empty")
	}
	if c.Queue.MaxRetryAttempts < 0 {
		return fmt.Errorf("QUEUE_MAX_RETRY_ATTEMPTS must be >= 0")
	}
	if c.Queue.MaxWorkers < 1 {
		return fmt.Errorf("QUEUE_MAX_WORKERS must be >= 1")
	}
	if c.Queue.ConcurrencyLimit < 1 {
		return fmt.Errorf("QUEUE_CONCURRENCY_LIMIT must be >= 1")
	}
	if c.Queue.WorkerCount < 1 {
		return fmt.Errorf("QUEUE_WORKER_COUNT must be >= 1")
	}
	if c.Queue.WorkerCount > c.Queue.MaxWorkers {
		return fmt.Errorf("QUEUE_WORKER_COUNT (%d) must be <= QUEUE_MAX_WORKERS (%d)", c.Queue.WorkerCount, c.Queue.MaxWorkers)
	}
	if c.Analyzer.MaxBodyBytes < 1 {
		return fmt.Errorf("ANALYZER_MAX_BODY_BYTES must be >= 1")
	}
	if c.Analyzer.FetchTimeout < time.Second {
		return fmt.Errorf("ANALYZER_FETCH_TIMEOUT_SEC must be >= 1")
	}
	if c.Analyzer.MaxRedirects < 1 {
		return fmt.Errorf("ANALYZER_MAX_REDIRECTS must be >= 1")
	}
	return nil
}

func getenv(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func getenvInt(key string, def int) int {
	s := strings.TrimSpace(os.Getenv(key))
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}

func getenvInt64(key string, def int64) int64 {
	s := strings.TrimSpace(os.Getenv(key))
	if s == "" {
		return def
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return def
	}
	return n
}
