package main

import (
	"log/slog"
	"net/http"
	"os"

	apphttp "github.com/praminda/link_analyzer/internal/http"
	"github.com/praminda/link_analyzer/internal/logging"
)

func main() {
	logger := logging.New()
	slog.SetDefault(logger)

	mux := apphttp.WithRequestLogging(apphttp.NewRouter())
	addr := ":8080"
	logger.Info("Server starting", "addr", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		logger.Error("Server failed", "error", err)
		os.Exit(1)
	}
}
