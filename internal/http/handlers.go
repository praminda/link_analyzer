package http

import (
	"encoding/json"
	"log/slog"
	stdhttp "net/http"
)

func AnalyzeHandler(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	logger := slog.With("request_id", r.Context().Value(requestIDContextKey).(string))

	var req AnalyzeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.URL == "" {
		logger.Warn("Invalid analyze request body", "error", err)
		stdhttp.Error(w, "Invalid json body", stdhttp.StatusBadRequest)
		return
	}
	logger.Info("Starting analysis", "url", req.URL)

	resp := AnalyzeResponse{
		HTMLVersion: "HTML5",
		PageTitle:   "Dummy Page Title",
		HeadingCounts: HeadingCounts{
			Heading1: 4,
			Heading2: 2,
			Heading3: 0,
			Heading4: 0,
			Heading5: 0,
			Heading6: 0,
		},
		ExternalLinks:     12,
		InternalLinks:     8,
		InaccessibleLinks: 1,
		IsLoginPage:       false,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		logger.Error("Failed to encode analyze response", "error", err)
	}
}
