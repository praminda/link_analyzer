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

func AnalyzeHandler(httpRes stdhttp.ResponseWriter, httpReq *stdhttp.Request) {
	logger := slog.With("request_id", httpReq.Context().Value(requestIDContextKey).(string))

	var req AnalyzeRequest
	if err := json.NewDecoder(httpReq.Body).Decode(&req); err != nil {
		logger.Warn("Invalid analyze request body", "error", err)
		writeAPIError(httpRes, stdhttp.StatusBadRequest, "invalid_json_body", "request body must be valid JSON")
		return
	}
	if req.URL == "" {
		logger.Warn("Invalid analyze request body", "error", "empty url")
		writeAPIError(httpRes, stdhttp.StatusBadRequest, "url_required", "url is required")
		return
	}
	logger.Info("Starting analysis", "url", req.URL)

	job := &analyzer.AnalyzeJob{URL: req.URL}
	if err := job.Process(httpReq.Context()); err != nil {
		logger.Error("Analysis failed", "url", req.URL, "error", err)
		writeAnalyzeError(httpRes, err)
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

	httpRes.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(httpRes).Encode(resp); err != nil {
		logger.Error("Failed to encode analyze response", "error", err)
	}
}

func writeAPIError(httpRes stdhttp.ResponseWriter, status int, code, message string) {
	httpRes.Header().Set("Content-Type", "application/json")
	httpRes.WriteHeader(status)
	_ = json.NewEncoder(httpRes).Encode(errorEnvelope{Error: errorPayload{Code: code, Message: message}})
}

func writeAnalyzeError(httpRes stdhttp.ResponseWriter, err error) {
	status := stdhttp.StatusInternalServerError
	code := "analysis_failed"
	message := "failed to analyze URL"

	if analyzeErr, ok := errors.AsType[*analyzer.AnalyzeError](err); ok {
		status = analyzeErr.HTTPStatus
		code = analyzeErr.Code
		message = analyzeErr.Message
	}

	writeAPIError(httpRes, status, code, message)
}
