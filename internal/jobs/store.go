package jobs

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"path/filepath"
	"time"

	"github.com/praminda/link_analyzer/internal/analyzer"

	_ "modernc.org/sqlite" // register driver name "sqlite"
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

// Store persists job lifecycle and results in SQLite.
type Store struct {
	db *sql.DB
}

// NewStore opens (or creates) a SQLite database at path and applies schema.
// path should be a filesystem path to the database file.
func NewStore(path string) (*Store, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("jobs store path: %w", err)
	}
	u := url.URL{
		Scheme:   "file",
		Path:     filepath.ToSlash(abs),
		RawQuery: "mode=rwc&_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)",
	}
	db, err := sql.Open("sqlite", u.String())
	if err != nil {
		return nil, fmt.Errorf("jobs store open: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetConnMaxLifetime(0)

	s := &Store{db: db}
	if err := s.migrate(context.Background()); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("jobs store migrate: %w", err)
	}
	return s, nil
}

func (s *Store) migrate(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS analyze_jobs (
	job_id TEXT PRIMARY KEY NOT NULL,
	status TEXT NOT NULL,
	url TEXT NOT NULL,
	result_json TEXT,
	error_code TEXT,
	error_message TEXT,
	created_at INTEGER NOT NULL,
	updated_at INTEGER NOT NULL
);
`)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
CREATE INDEX IF NOT EXISTS idx_analyze_jobs_updated_at ON analyze_jobs(updated_at);
`)
	return err
}

// Close releases the database handle.
func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

// CreateQueued inserts a new job in queued state (or resets an existing id to queued).
func (s *Store) CreateQueued(jobID, url string) {
	if s == nil || s.db == nil {
		return
	}
	now := time.Now().Unix()
	_, err := s.db.ExecContext(context.Background(), `
INSERT INTO analyze_jobs (job_id, status, url, created_at, updated_at)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(job_id) DO UPDATE SET
	status = excluded.status,
	url = excluded.url,
	result_json = NULL,
	error_code = NULL,
	error_message = NULL,
	updated_at = excluded.updated_at
`, jobID, string(StatusQueued), url, now, now)
	if err != nil {
		slog.Default().Error("jobs store: CreateQueued", "error", err, "job_id", jobID)
	}
}

// SetRunning marks a job as running.
func (s *Store) SetRunning(jobID string) {
	if s == nil || s.db == nil {
		return
	}
	now := time.Now().Unix()
	res, err := s.db.ExecContext(context.Background(), `
UPDATE analyze_jobs SET status = ?, updated_at = ? WHERE job_id = ?
`, string(StatusRunning), now, jobID)
	if err != nil {
		slog.Default().Error("jobs store: SetRunning", "error", err, "job_id", jobID)
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		slog.Default().Warn("jobs store: SetRunning no row", "job_id", jobID)
	}
}

// SetCompleted stores the analyzer result.
func (s *Store) SetCompleted(jobID string, out analyzer.AnalyzeResponse) {
	if s == nil || s.db == nil {
		return
	}
	b, err := json.Marshal(out)
	if err != nil {
		slog.Default().Error("jobs store: SetCompleted marshal", "error", err, "job_id", jobID)
		return
	}
	now := time.Now().Unix()
	_, err = s.db.ExecContext(context.Background(), `
UPDATE analyze_jobs
SET status = ?, result_json = ?, error_code = NULL, error_message = NULL, updated_at = ?
WHERE job_id = ?
`, string(StatusCompleted), b, now, jobID)
	if err != nil {
		slog.Default().Error("jobs store: SetCompleted", "error", err, "job_id", jobID)
	}
}

// SetFailed records failure.
func (s *Store) SetFailed(jobID string, ae *analyzer.AnalyzeError) {
	if s == nil || s.db == nil {
		return
	}
	code := "unknown_error"
	msg := "analysis failed"
	if ae != nil {
		code = ae.Code
		if ae.Message != "" {
			msg = ae.Message
		}
	}
	now := time.Now().Unix()
	_, err := s.db.ExecContext(context.Background(), `
UPDATE analyze_jobs
SET status = ?, result_json = NULL, error_code = ?, error_message = ?, updated_at = ?
WHERE job_id = ?
`, string(StatusFailed), code, msg, now, jobID)
	if err != nil {
		slog.Default().Error("jobs store: SetFailed", "error", err, "job_id", jobID)
	}
}

// Get returns a copy of the record and whether it existed.
func (s *Store) Get(jobID string) (Record, bool) {
	if s == nil || s.db == nil {
		return Record{}, false
	}
	var (
		statusStr    string
		url          string
		resultJSON   sql.NullString
		errorCode    sql.NullString
		errorMessage sql.NullString
	)
	err := s.db.QueryRowContext(context.Background(), `
SELECT status, url, result_json, error_code, error_message
FROM analyze_jobs WHERE job_id = ?
`, jobID).Scan(&statusStr, &url, &resultJSON, &errorCode, &errorMessage)
	if errors.Is(err, sql.ErrNoRows) {
		return Record{}, false
	}
	if err != nil {
		slog.Default().Error("jobs store: Get", "error", err, "job_id", jobID)
		return Record{}, false
	}

	rec := Record{
		Status:       Status(statusStr),
		URL:          url,
		ErrorCode:    errorCode.String,
		ErrorMessage: errorMessage.String,
	}
	if resultJSON.Valid && resultJSON.String != "" {
		if err := json.Unmarshal([]byte(resultJSON.String), &rec.Result); err != nil {
			slog.Default().Error("jobs store: Get unmarshal result", "error", err, "job_id", jobID)
		}
	}
	return rec, true
}
