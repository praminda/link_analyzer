package analyzer

// JobRunNotifier receives lifecycle signals for an [AnalyzeJob]. Optional; when nil, no callbacks run.
type JobRunNotifier interface {
	OnRunStarted(jobID string)
	OnRunSucceeded(jobID string, result AnalyzeResponse)
	OnRunFailed(jobID string, err *AnalyzeError)
}
