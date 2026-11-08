package runs

import (
	"context"
	"errors"

	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/reliability"
)

func (s *Service) ReliabilityChecks(ctx context.Context, user string, q reliability.Query, cursor string, limit int) (reliability.CheckPage, error) {
	if !domain.ValidID(q.Project) || !domain.ValidID(q.Suite) || !domain.ValidID(q.Test) || q.Environment != "" && !domain.ValidID(q.Environment) || len(q.Location) > 128 {
		return reliability.CheckPage{}, errors.New("valid reliability scope is required")
	}
	if _, err := s.Catalog.RetainedTest(ctx, user, q.Project, q.Suite, q.Test); err != nil {
		return reliability.CheckPage{}, err
	}
	if err := s.Auth.CheckCollection(ctx, user, q.Project, "runs.read"); err != nil {
		return reliability.CheckPage{}, err
	}
	if q.Environment != "" {
		if _, err := s.Store.Get(ctx, projects.Bucket("environments", q.Project), q.Environment); err != nil {
			return reliability.CheckPage{}, err
		}
	}
	if err := s.ReliabilityReady(ctx); err != nil {
		return reliability.CheckPage{}, err
	}
	if q.Location == "" {
		q.Location = s.Location
	}
	reader, err := s.Auth.NewRunReadBatch(ctx, user)
	if err != nil {
		return reliability.CheckPage{}, err
	}
	service := reliability.Service{Store: s.Store}
	return service.Checks(ctx, reader, q, user, cursor, limit)
}
