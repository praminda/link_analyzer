package analyzer

import "context"

// AnalyzeJob runs link analysis for a single URL. Intended to be used as
// one instance per request.
type AnalyzeJob struct {
	URL string
}

// Process performs analysis. Core steps (fetch, parse, link checks)
func (j *AnalyzeJob) Process(ctx context.Context) error {
	return nil
}
