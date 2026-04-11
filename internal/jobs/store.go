package jobs

import (
	"github.com/praminda/link_analyzer/internal/analyzer"
)

// Status is the coarse lifecycle of an application job (HTTP polling).
type Status string

const (
	StatusQueued    Status = "queued"
	StatusRunning   Status = "running"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
)

// Record is a snapshot of one analyze job for the API layer.
type Record struct {
	Status       Status
	URL          string
	Result       analyzer.AnalyzeResponse
	ErrorCode    string
	ErrorMessage string
}

// Store is a sqlite backed registry of job status and outcomes.
type Store struct {
}

// NewStore returns a new job store.
func NewStore() *Store {
	return &Store{}
}

// CreateQueued inserts a new job in queued state.
func (s *Store) CreateQueued(jobID, url string) {
}

// SetRunning marks a job as running.
func (s *Store) SetRunning(jobID string) {
}

// SetCompleted stores the analyzer result.
func (s *Store) SetCompleted(jobID string, out analyzer.AnalyzeResponse) {
}

// SetFailed records failure.
func (s *Store) SetFailed(jobID string, ae *analyzer.AnalyzeError) {
}

// Get returns a copy of the record and whether it existed.
func (s *Store) Get(jobID string) (Record, bool) {
	return Record{}, false
}
