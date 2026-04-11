package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	apphttp "github.com/praminda/link_analyzer/internal/http"
	"github.com/praminda/link_analyzer/internal/jobs"
	"github.com/praminda/link_analyzer/internal/logging"
	"github.com/saravanasai/goqueue"
	"github.com/saravanasai/goqueue/config"
)

func main() {
	logger := logging.New()
	slog.SetDefault(logger)

	jobStore := jobs.NewStore()
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
	if err := q.StartWorkers(context.Background(), workerCount); err != nil {
		logger.Error("Failed to start queue workers", "error", err)
		os.Exit(1)
	}

	srv := &apphttp.Server{Queue: q, Jobs: jobStore}
	mux := apphttp.WithRequestLogging(apphttp.NewRouter(srv))
	addr := ":8080"
	logger.Info("Server starting", "addr", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		logger.Error("Server failed", "error", err)
		os.Exit(1)
	}
}
