package catalog

import (
	"context"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/timing"
)

func (s *Service) timingSettings(ctx context.Context, project, suite string) (timing.Policy, error) {
	r, err := s.Store.Get(ctx, "projects", project)
	if err != nil {
		return timing.Policy{}, err
	}
	p, err := storage.Decode[projects.Project](r)
	if err != nil {
		return timing.Policy{}, err
	}
	v, _, err := s.suite(ctx, project, suite)
	if err != nil {
		return timing.Policy{}, err
	}
	return timing.Resolve(p.Timing, v.Timing), nil
}
