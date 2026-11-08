package runs

import (
	"context"
	"errors"
	"time"

	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/reliability"
	"github.com/wtiger001/teststudio/internal/storage"
)

func (s *Service) ReliabilityReport(ctx context.Context, user string, q reliability.Query, window string) (reliability.Report, error) {
	if !domain.ValidID(q.Project) || !domain.ValidID(q.Suite) || !domain.ValidID(q.Test) || q.Environment != "" && !domain.ValidID(q.Environment) || len(q.Location) > 128 {
		return reliability.Report{}, errors.New("valid reliability scope is required")
	}
	if _, err := s.Store.Get(ctx, "projects", q.Project); err != nil {
		return reliability.Report{}, err
	}
	retained, err := s.Catalog.RetainedTest(ctx, user, q.Project, q.Suite, q.Test)
	if err != nil {
		return reliability.Report{}, err
	}
	if err = s.Auth.CheckCollection(ctx, user, q.Project, "runs.read"); err != nil {
		return reliability.Report{}, err
	}
	if err = s.ReliabilityReady(ctx); err != nil {
		return reliability.Report{}, err
	}
	var environment *projects.Environment
	if q.Environment != "" {
		env, err := s.Store.Get(ctx, projects.Bucket("environments", q.Project), q.Environment)
		if err != nil {
			return reliability.Report{}, err
		}
		if env.Project != q.Project {
			return reliability.Report{}, storage.ErrNotFound
		}
		value, err := storage.Decode[projects.Environment](env)
		if err != nil {
			return reliability.Report{}, err
		}
		environment = &value
	}
	if q.Location == "" {
		q.Location = s.Location
	}
	if window == "" {
		window = "24h"
	}
	if q.Until.IsZero() {
		q.Until = time.Now().UTC()
	}
	if q.Since.IsZero() {
		q.Since, err = reliability.WindowStart(window, q.Until)
		if err != nil {
			return reliability.Report{}, err
		}
	}
	if q.MaxAge != 0 && (q.MaxAge < 60 || q.MaxAge > 86400) {
		return reliability.Report{}, errors.New("monitor freshness must be zero or 60–86400 seconds")
	}
	if q.MaxAge > 0 {
		release, err := s.Catalog.ActiveRelease(ctx, q.Project, q.Suite)
		if err != nil {
			return reliability.Report{}, err
		}
		desired := catalog.Test{}
		for _, test := range release.Draft.Tests {
			if test.ID == q.Test {
				desired = test
				break
			}
		}
		q.Match = ReliabilityMatch(desired, environment, release)
		if !retained.Current || retained.SuiteArchived || retained.Test.State != "enabled" || desired.ID == "" || environment != nil && environment.Disabled {
			q.Match = "disabled"
		}
	}
	reader, err := s.Auth.NewRunReadBatch(ctx, user)
	if err != nil {
		return reliability.Report{}, err
	}
	service := reliability.Service{Store: s.Store}
	row, err := service.Read(ctx, reader, q)
	if err != nil {
		return reliability.Report{}, err
	}
	row.Title = retained.Test.Title
	row.Description = retained.Test.Description
	return reliability.Report{Title: retained.Test.Title, Window: window, MaxAge: q.MaxAge, At: q.Until, Probes: []reliability.Row{row}}, nil
}
