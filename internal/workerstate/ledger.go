package workerstate

import (
	"context"
	"errors"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/storage"
	"time"
)

type Record struct {
	EvidenceComplete  bool                `json:"evidenceComplete"`
	EvidenceAbandoned bool                `json:"evidenceAbandoned"`
	ReconcileAttempts int                 `json:"reconcileAttempts"`
	RecoveredResult   *execution.Result   `json:"recoveredResult,omitempty"`
	Reference         execution.Reference `json:"reference"`
	State             string              `json:"state"`
	ResultSaved       bool                `json:"resultSaved"`
	Recovered         bool                `json:"recovered"`
	EvidenceErrors    []string            `json:"evidenceErrors"`
	Error             string              `json:"error,omitempty"`
	CleanupAttempts   int                 `json:"cleanupAttempts"`
	CreatedAt         time.Time           `json:"createdAt"`
	UpdatedAt         time.Time           `json:"updatedAt"`
	NextCheck         time.Time           `json:"nextCheck"`
	Claim             string              `json:"claimToken,omitempty"`
	ClaimUntil        *time.Time          `json:"claimUntil,omitempty"`
}

type Job struct {
	Run       string `json:"runId"`
	Execution string `json:"executionId"`
}
type Ledger struct{ Store *storage.Store }

func Bucket(run string) string { return "executions/" + run }
func (l Ledger) Installation(ctx context.Context, create bool) (string, error) {
	for range 8 {
		r, err := l.Store.Get(ctx, "runtime", "execution-installation")
		if err == nil {
			return storage.Decode[string](r)
		}
		if !errors.Is(err, storage.ErrNotFound) || !create {
			return "", err
		}
		id := domain.NewID()
		change, err := storage.Put("runtime", "execution-installation", "", 0, id)
		if err != nil {
			return "", err
		}
		if err := l.Store.Commit(ctx, change); errors.Is(err, storage.ErrConflict) {
			continue
		} else if err != nil {
			return "", err
		}
		return id, nil
	}
	return "", storage.ErrConflict
}
func (l Ledger) Get(ctx context.Context, run, id string) (Record, storage.Record, error) {
	r, err := l.Store.Get(ctx, Bucket(run), id)
	if err != nil {
		return Record{}, r, err
	}
	v, err := storage.Decode[Record](r)
	return v, r, err
}
func (l Ledger) Begin(ref execution.Reference) (Record, []storage.Change, error) {
	now := time.Now().UTC()
	v := Record{Reference: ref, State: "prepared", EvidenceErrors: []string{}, CreatedAt: now, UpdatedAt: now, NextCheck: now}
	record, err := storage.Put(Bucket(ref.Run), ref.ID, ref.Project, 0, v)
	if err != nil {
		return v, nil, err
	}
	job, err := storage.Put("execution-work", ref.ID, ref.Project, 0, Job{Run: ref.Run, Execution: ref.ID})
	return v, []storage.Change{record, job}, err
}
func (l Ledger) Checkpoint(row storage.Record, v Record, problems []string, recovered bool) (storage.Change, error) {
	v.ResultSaved = true
	v.EvidenceComplete = len(problems) == 0
	v.Recovered = recovered
	v.State = "checkpointed"
	if !v.EvidenceComplete {
		v.State = "evidence-pending"
		v.NextCheck = time.Now().UTC().Add(30 * time.Second)
	}
	v.EvidenceErrors = problems
	v.UpdatedAt = time.Now().UTC()
	v.Error = ""
	return storage.Put(row.Bucket, row.ID, row.Project, row.Revision, v)
}
func (l Ledger) Finish(ctx context.Context, run, id, claim string, cleanupErr error) error {
	for range 8 {
		v, row, err := l.Get(ctx, run, id)
		if err != nil {
			return err
		}
		if v.Claim != claim {
			return storage.ErrConflict
		}
		now := time.Now().UTC()
		v.UpdatedAt = now
		v.CleanupAttempts++
		v.Claim = ""
		v.ClaimUntil = nil
		v.State = "cleaned"
		if cleanupErr == nil && !v.EvidenceComplete {
			v.EvidenceAbandoned = true
		}
		v.Error = ""
		if cleanupErr != nil {
			v.State = "cleanup-failed"
			v.Error = cleanupErr.Error()
			v.NextCheck = now.Add(30 * time.Second)
		}
		c, err := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, v)
		if err != nil {
			return err
		}
		changes := []storage.Change{c}
		if cleanupErr == nil {
			job, err := l.Store.Get(ctx, "execution-work", id)
			if err == nil {
				changes = append(changes, storage.Change{Record: job, Delete: true})
			} else if !errors.Is(err, storage.ErrNotFound) {
				return err
			}
		}
		err = l.Store.Commit(ctx, changes...)
		if errors.Is(err, storage.ErrConflict) {
			continue
		}
		return err
	}
	return storage.ErrConflict
}
