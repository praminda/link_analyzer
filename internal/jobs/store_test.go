package jobs

import (
	"path/filepath"
	"testing"

	"github.com/praminda/link_analyzer/internal/analyzer"
)

func TestStore_JobLifecycle(t *testing.T) {
	path := filepath.Join(t.TempDir(), "jobs.sqlite")
	s, err := NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()

	const id, u = "job-a", "https://example.com/"
	s.CreateQueued(id, u)
	r, ok := s.Get(id)
	if !ok {
		t.Fatal("expected row")
	}
	if r.Status != StatusQueued || r.URL != u {
		t.Fatalf("queued: %+v", r)
	}

	s.SetRunning(id)
	r, ok = s.Get(id)
	if !ok || r.Status != StatusRunning {
		t.Fatalf("running: %+v ok=%v", r, ok)
	}

	want := analyzer.AnalyzeResponse{
		HTMLVersion: "5",
		PageTitle:   "T",
		HeadingCounts: analyzer.HeadingCounts{
			Heading1: 1,
		},
		ExternalLinks: 2,
		InternalLinks: 3,
		IsLoginPage:   true,
	}
	s.SetCompleted(id, want)
	r, ok = s.Get(id)
	if !ok || r.Status != StatusCompleted {
		t.Fatalf("completed: %+v ok=%v", r, ok)
	}
	if r.Result != want {
		t.Fatalf("result mismatch: %+v", r.Result)
	}
}

func TestStore_SetFailed(t *testing.T) {
	path := filepath.Join(t.TempDir(), "jobs.sqlite")
	s, err := NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()

	s.CreateQueued("j2", "https://example.org/")
	s.SetFailed("j2", &analyzer.AnalyzeError{Code: "x", Message: "m"})
	r, ok := s.Get("j2")
	if !ok || r.Status != StatusFailed || r.ErrorCode != "x" || r.ErrorMessage != "m" {
		t.Fatalf("failed: %+v ok=%v", r, ok)
	}
}

func TestStore_ReopenKeepsRows(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "db.sqlite")

	s1, err := NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	s1.CreateQueued("persist", "https://a.test/")
	_ = s1.Close()

	s2, err := NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s2.Close() }()

	r, ok := s2.Get("persist")
	if !ok || r.URL != "https://a.test/" {
		t.Fatalf("after reopen: %+v ok=%v", r, ok)
	}
}
