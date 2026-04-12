package main

import (
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	apphttp "github.com/praminda/link_analyzer/internal/http"
	"github.com/praminda/link_analyzer/internal/jobs"
	"github.com/praminda/link_analyzer/internal/logging"
	"github.com/saravanasai/goqueue"
	"github.com/saravanasai/goqueue/config"
)

func main() {
	logger := logging.New()
	slog.SetDefault(logger)

	dbPath := os.Getenv("JOB_DB_PATH")
	if dbPath == "" {
		dbPath = filepath.Join("data", "jobs.sqlite")
	}
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		logger.Error("Failed to create job database directory", "error", err, "dir", filepath.Dir(dbPath))
		os.Exit(1)
	}
	jobStore, err := jobs.NewStore(dbPath)
	if err != nil {
		logger.Error("Failed to open job database", "error", err, "path", dbPath)
		os.Exit(1)
	}
	defer jobStore.Close()

	// TODO: Make these values configurable
	cfg := config.NewInMemoryConfig().
		WithMaxRetryAttempts(1).
		WithMaxWorkers(4).
		WithConcurrencyLimit(8)
	q, err := goqueue.NewQueueWithDefaults("link-analyze", cfg)
	if err != nil {
		logger.Error("Failed to create job queue", "error", err)
		os.Exit(1)
	}
	workerCount := min(2, cfg.MaxWorkers)

	srv := &apphttp.Server{
		Queue:       q,
		Jobs:        jobStore,
		WorkerCount: workerCount,
	}
	mux := apphttp.WithRequestLogging(apphttp.NewRouter(srv))
	addr := ":8080"
	logger.Info("Server starting", "addr", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		logger.Error("Server failed", "error", err)
		os.Exit(1)
	}
}
