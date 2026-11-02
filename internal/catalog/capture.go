package catalog

import (
	"context"

	"github.com/wtiger001/teststudio/internal/capture"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/storage"
)

func (s *Service) captureSettings(ctx context.Context, project, suite string) (capture.Policy, capture.Limits, error) {
	r, err := s.Store.Get(ctx, "projects", project)
	if err != nil {
		return capture.Policy{}, capture.Limits{}, err
	}
	p, err := storage.Decode[projects.Project](r)
	if err != nil {
		return capture.Policy{}, capture.Limits{}, err
	}
	v, _, err := s.suite(ctx, project, suite)
	if err != nil {
		return capture.Policy{}, capture.Limits{}, err
	}
	return capture.Resolve(p.Capture, v.Capture), capture.ResolveLimits(p.CaptureLimits), nil
}
