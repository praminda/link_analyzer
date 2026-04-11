package jobs

import (
	"github.com/praminda/link_analyzer/internal/analyzer"
	"github.com/saravanasai/goqueue"
)

func init() {
	goqueue.RegisterJob("AnalyzeJob", func() goqueue.Job {
		return &analyzer.AnalyzeJob{}
	})
}
