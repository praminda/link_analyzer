package jobs

import "github.com/praminda/link_analyzer/internal/analyzer"

// StoreNotifier implements [analyzer.JobRunNotifier] by updating a [Store].
type StoreNotifier struct {
	store *Store
}

// NewStoreNotifier returns a notifier that writes to store. Store must not be nil.
func NewStoreNotifier(store *Store) *StoreNotifier {
	return &StoreNotifier{store: store}
}

func (sn *StoreNotifier) OnRunStarted(jobID string) {
	if sn == nil || sn.store == nil {
		return
	}
	sn.store.SetRunning(jobID)
}

func (sn *StoreNotifier) OnRunSucceeded(jobID string, result analyzer.AnalyzeResponse) {
	if sn == nil || sn.store == nil {
		return
	}
	sn.store.SetCompleted(jobID, result)
}

func (sn *StoreNotifier) OnRunFailed(jobID string, err *analyzer.AnalyzeError) {
	if sn == nil || sn.store == nil {
		return
	}
	sn.store.SetFailed(jobID, err)
}
