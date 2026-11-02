package evidence

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/wtiger001/teststudio/internal/capture"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/storage"
)

type RetentionView struct {
	Finished     bool            `json:"finished"`
	Pin          Pin             `json:"pin"`
	Revision     int64           `json:"revision"`
	Status       RetentionStatus `json:"status"`
	RunQuota     Quota           `json:"runQuota"`
	ProjectQuota Quota           `json:"projectQuota"`
	Capture      capture.Policy  `json:"capture"`
	Limits       capture.Limits  `json:"limits"`
}

func (s *Service) Retention(ctx context.Context, project, run string) (RetentionView, error) {
	v := RetentionView{Status: RetentionStatus{State: "pending"}}
	runRow, err := s.Store.Get(ctx, "runs", run)
	if err != nil {
		return v, err
	}
	snapshot, err := storage.Decode[struct {
		Project    string         `json:"projectId"`
		Capture    capture.Policy `json:"capture"`
		Limits     capture.Limits `json:"captureLimits"`
		FinishedAt *string        `json:"finishedAt"`
	}](runRow)
	if err != nil {
		return v, err
	}
	if snapshot.Project != project {
		return v, storage.ErrNotFound
	}
	v.Finished = snapshot.FinishedAt != nil
	v.Capture = capture.Normalize(snapshot.Capture)
	v.Limits = snapshot.Limits
	if v.Limits.ProjectBytes == 0 {
		v.Limits = capture.DefaultLimits()
	}
	v.RunQuota.Limit = v.Capture.MaxBytes
	if snapshot.FinishedAt == nil {
		v.Status.State = "waiting"
	}
	p, r, err := s.ReadPin(ctx, project, run)
	if err != nil {
		return v, err
	}
	v.Pin = p
	v.Revision = r.Revision
	for _, entry := range []struct {
		bucket, id string
		value      any
	}{
		{"evidence-retention-status", run, &v.Status},
		{"artifact-quotas", run, &v.RunQuota},
		{"artifact-project-quotas", project, &v.ProjectQuota},
	} {
		r, err := s.Store.Get(ctx, entry.bucket, entry.id)
		if errors.Is(err, storage.ErrNotFound) {
			continue
		}
		if err != nil {
			return v, err
		}
		if r.Project != project {
			return v, storage.ErrNotFound
		}
		if err := json.Unmarshal(r.Data, entry.value); err != nil {
			return v, err
		}
	}
	projectRow, err := s.Store.Get(ctx, "projects", project)
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return v, err
	}
	if err == nil {
		p, err := storage.Decode[projects.Project](projectRow)
		if err != nil {
			return v, err
		}
		v.ProjectQuota.Limit = capture.ResolveLimits(p.CaptureLimits).ProjectBytes
	}
	return v, nil
}
