package main

import (
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	appcfg "github.com/praminda/link_analyzer/internal/appconfig"
	apphttp "github.com/praminda/link_analyzer/internal/http"
	"github.com/praminda/link_analyzer/internal/jobs"
	"github.com/praminda/link_analyzer/internal/logging"
	"github.com/saravanasai/goqueue"
	"github.com/saravanasai/goqueue/config"
)

func main() {
	cfg, err := appcfg.Load()
	if err != nil {
		slog.Error("Invalid configuration", "error", err)
		os.Exit(1)
	}

	logger := logging.New(cfg.Log)
	slog.SetDefault(logger)

	dbPath := cfg.Jobs.DBPath
	if err := os.MkdirAll(filepath.Dir(dbPath), os.ModePerm); err != nil {
		logger.Error("Failed to create job database directory", "error", err, "dir", filepath.Dir(dbPath))
		os.Exit(1)
	}
	jobStore, err := jobs.NewStore(dbPath)
	if err != nil {
		logger.Error("Failed to open job database", "error", err, "path", dbPath)
		os.Exit(1)
	}
	defer jobStore.Close()

	gqCfg := config.NewInMemoryConfig().
		WithMaxRetryAttempts(cfg.Queue.MaxRetryAttempts).
		WithMaxWorkers(cfg.Queue.MaxWorkers).
		WithConcurrencyLimit(cfg.Queue.ConcurrencyLimit)
	q, err := goqueue.NewQueueWithDefaults(cfg.Queue.Name, gqCfg)
	if err != nil {
		logger.Error("Failed to create job queue", "error", err)
		os.Exit(1)
	}

	srv := &apphttp.Server{
		Queue:       q,
		Jobs:        jobStore,
		WorkerCount: cfg.Queue.WorkerCount,
		Analyzer:    &cfg.Analyzer,
	}
	mux := apphttp.WithRequestLogging(apphttp.NewRouter(srv))
	logger.Info("Server starting", "addr", cfg.HTTP.Addr)
	if err := http.ListenAndServe(cfg.HTTP.Addr, mux); err != nil {
		logger.Error("Server failed", "error", err)
		os.Exit(1)
	}
}
