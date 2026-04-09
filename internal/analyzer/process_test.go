package analyzer

import (
	"context"
	"errors"
	"testing"
)

func TestAnalyzeJob_Process_nilReceiver(t *testing.T) {
	var job *AnalyzeJob
	err := job.Process(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAnalyzeJob_Process_rejectsLoopback(t *testing.T) {
	job := &AnalyzeJob{URL: "http://127.0.0.1/"}
	err := job.Process(context.Background())
	if !errors.Is(err, ErrDisallowedHost) {
		t.Fatalf("err = %v, want ErrDisallowedHost", err)
	}
}
