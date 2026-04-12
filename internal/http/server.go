package http

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	stdhttp "net/http"
	"sync"

	"github.com/praminda/link_analyzer/internal/analyzer"
	"github.com/praminda/link_analyzer/internal/appconfig"
	"github.com/praminda/link_analyzer/internal/jobs"
	"github.com/saravanasai/goqueue/queue"
)

type errorEnvelope struct {
	Error errorPayload `json:"error"`
}

type errorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Server holds collaborators for HTTP handlers (queue + job store).
type Server struct {
	Queue *queue.Queue
	Jobs  *jobs.Store

	// Analyzer holds the analyzer configurations.
	Analyzer *appconfig.AnalyzerConfig

	// WorkerCount is the number of goqueue worker goroutines. Workers are started
	// lazily on the first successful analyze enqueue to avoid an issue with goqueue's memory store.
	WorkerCount int

	workerMu       sync.Mutex
	workersStarted bool
}

func (s *Server) ensureWorkersStarted() error {
	s.workerMu.Lock()
	defer s.workerMu.Unlock()
	if s.workersStarted {
		return nil
	}
	if s.Queue == nil {
		return errors.New("job queue is not configured")
	}
	n := max(s.WorkerCount, 1)
	if err := s.Queue.StartWorkers(context.Background(), n); err != nil {
		return err
	}
	s.workersStarted = true
	return nil
}

func (s *Server) handleAnalyze(httpRes stdhttp.ResponseWriter, httpReq *stdhttp.Request) {
	reqID, _ := httpReq.Context().Value(requestIDContextKey).(string)
	logger := slog.With("request_id", reqID)

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
	if err := analyzer.ValidateAnalyzeURL(httpReq.Context(), req.URL); err != nil {
		logger.Warn("URL validation failed", "url", req.URL, "error", err)
		writeValidateURLError(httpRes, err)
		return
	}

	// JobID matches request id for correlation (see middleware X-Request-Id).
	jobID := reqID
	if jobID == "" {
		jobID = "request-id-unavailable"
	}
	logger.Info("Enqueue analysis", "url", req.URL, "job_id", jobID)
	jobLog := logger.With("job_id", jobID)

	s.Jobs.CreateQueued(jobID, req.URL)
	job := &analyzer.AnalyzeJob{
		URL:      req.URL,
		JobID:    jobID,
		Log:      jobLog,
		Notifier: jobs.NewStoreNotifier(s.Jobs),
	}
	job.Analyzer = s.Analyzer
	if err := s.Queue.Dispatch(job); err != nil {
		logger.Error("Failed to dispatch job", "error", err)
		s.Jobs.SetFailed(jobID, &analyzer.AnalyzeError{
			HTTPStatus: stdhttp.StatusInternalServerError,
			Code:       "enqueue_failed",
			Message:    "failed to enqueue analysis job",
		})
		writeAPIError(httpRes, stdhttp.StatusInternalServerError, "enqueue_failed", "failed to enqueue analysis job")
		return
	}

	if err := s.ensureWorkersStarted(); err != nil {
		logger.Error("Failed to start job workers", "error", err)
		writeAPIError(httpRes, stdhttp.StatusInternalServerError, "workers_unavailable", "failed to start background workers")
		return
	}

	httpRes.Header().Set("Content-Type", "application/json")
	httpRes.WriteHeader(stdhttp.StatusAccepted)
	if err := json.NewEncoder(httpRes).Encode(AnalyzeAcceptedResponse{JobID: jobID}); err != nil {
		logger.Error("Failed to encode analyze accepted response", "error", err)
	}
}

func (s *Server) handleJobStatus(httpRes stdhttp.ResponseWriter, httpReq *stdhttp.Request) {
	reqID, _ := httpReq.Context().Value(requestIDContextKey).(string)
	logger := slog.With("request_id", reqID)

	jobID := httpReq.PathValue("jobId")
	if jobID == "" {
		writeAPIError(httpRes, stdhttp.StatusBadRequest, "job_id_required", "job id is required")
		return
	}

	rec, ok := s.Jobs.Get(jobID)
	if !ok {
		logger.Warn("Job not found", "job_id", jobID)
		writeAPIError(httpRes, stdhttp.StatusNotFound, "job_not_found", "no job exists for this id")
		return
	}

	out := JobStatusResponse{Status: string(rec.Status)}
	switch rec.Status {
	case jobs.StatusCompleted:
		r := analyzerResultToDTO(rec.Result)
		out.Result = &r
	case jobs.StatusFailed:
		out.Error = &JobStatusError{Code: rec.ErrorCode, Message: rec.ErrorMessage}
	}

	httpRes.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(httpRes).Encode(out); err != nil {
		logger.Error("Failed to encode job status response", "error", err)
	}
}

func analyzerResultToDTO(out analyzer.AnalyzeResponse) AnalyzeResponse {
	return AnalyzeResponse{
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
}

func writeValidateURLError(httpRes stdhttp.ResponseWriter, err error) {
	switch {
	case errors.Is(err, analyzer.ErrInvalidURL), errors.Is(err, analyzer.ErrDisallowedHost):
		writeAPIError(httpRes, stdhttp.StatusBadRequest, "url_validation_failed", err.Error())
	default:
		writeAPIError(httpRes, stdhttp.StatusBadRequest, "url_validation_failed", "invalid URL")
	}
}

func writeAPIError(httpRes stdhttp.ResponseWriter, status int, code, message string) {
	httpRes.Header().Set("Content-Type", "application/json")
	httpRes.WriteHeader(status)
	_ = json.NewEncoder(httpRes).Encode(errorEnvelope{Error: errorPayload{Code: code, Message: message}})
}
