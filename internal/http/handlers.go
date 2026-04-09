package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	stdhttp "net/http"

	"github.com/praminda/link_analyzer/internal/analyzer"
)

type errorEnvelope struct {
	Error errorPayload `json:"error"`
}

type errorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func AnalyzeHandler(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	logger := slog.With("request_id", r.Context().Value(requestIDContextKey).(string))

	var req AnalyzeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.URL == "" {
		logger.Warn("Invalid analyze request body", "error", err)
		stdhttp.Error(w, "Invalid json body", stdhttp.StatusBadRequest)
		return
	}
	logger.Info("Starting analysis", "url", req.URL)

	job := &analyzer.AnalyzeJob{URL: req.URL}
	if err := job.Process(r.Context()); err != nil {
		logger.Error("Analysis failed", "url", req.URL, "error", err)
		writeAnalyzeError(w, err)
		return
	}
	out := job.Response()
	resp := AnalyzeResponse{
		HTMLVersion: out.HTMLVersion,
		PageTitle:   out.PageTitle,
		HeadingCounts: HeadingCounts{
			Heading1: out.HeadingCounts.Heading1,
			Heading2: out.HeadingCounts.Heading2,
			Heading3: out.HeadingCounts.Heading3,
			Heading4: out.HeadingCounts.Heading4,
			Heading5: out.HeadingCounts.Heading5,
			Heading6: out.HeadingCounts.Heading6,
		},
		ExternalLinks:     out.ExternalLinks,
		InternalLinks:     out.InternalLinks,
		InaccessibleLinks: out.InaccessibleLinks,
		IsLoginPage:       out.IsLoginPage,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		logger.Error("Failed to encode analyze response", "error", err)
	}
}

func writeAnalyzeError(w stdhttp.ResponseWriter, err error) {
	status := stdhttp.StatusInternalServerError
	payload := errorPayload{
		Code:    "analysis_failed",
		Message: "failed to analyze URL",
	}

	if analyzeErr, ok := errors.AsType[*analyzer.AnalyzeError](err); ok {
		status = analyzeErr.HTTPStatus
		payload.Code = analyzeErr.Code
		payload.Message = analyzeErr.Message
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(errorEnvelope{Error: payload})
}
