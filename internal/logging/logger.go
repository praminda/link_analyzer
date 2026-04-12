package logging

import (
	"log/slog"
	"os"
	"strings"

	"github.com/praminda/link_analyzer/internal/appconfig"
)

// New builds the default application logger from Options.
func New(opts appconfig.LogConfig) *slog.Logger {
	level := parseLevel(opts.Level)
	hOpts := &slog.HandlerOptions{Level: level}

	if opts.UseJSON {
		return slog.New(slog.NewJSONHandler(os.Stdout, hOpts))
	}
	return slog.New(slog.NewTextHandler(os.Stdout, hOpts))
}

func parseLevel(raw string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
