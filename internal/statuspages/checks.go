package statuspages

import (
	"context"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/reliability"
	"github.com/wtiger001/teststudio/internal/runs"
	"github.com/wtiger001/teststudio/internal/storage"
)

func (s *Service) PublicChecks(ctx context.Context, token string, row int, q reliability.Query, cursor string, limit int) (reliability.CheckPage, error) {
	page, project, err := s.publishedPage(ctx, token)
	if err != nil {
		return reliability.CheckPage{}, err
	}
	if row < 0 || row >= len(page.Published.Probes) {
		return reliability.CheckPage{}, storage.ErrNotFound
	}
	record, err := s.Store.Get(ctx, "projects", project)
	if err != nil {
		return reliability.CheckPage{}, err
	}
	p, err := storage.Decode[projects.Project](record)
	if err != nil {
		return reliability.CheckPage{}, err
	}
	if p.Archived {
		return reliability.CheckPage{}, storage.ErrNotFound
	}
	entry := page.Published.Probes[row]
	scope, test, environment, disabled, err := s.resolve(ctx, page.Publisher, project, entry)
	if err != nil {
		return reliability.CheckPage{}, storage.ErrNotFound
	}
	if err = s.Probes.Runs.ReliabilityReady(ctx); err != nil {
		return reliability.CheckPage{}, err
	}
	reader, err := s.Auth.NewRunReadBatch(ctx, page.Publisher)
	if err != nil {
		return reliability.CheckPage{}, err
	}
	q.Scope = scope
	if entry.Probe != "" || entry.Monitor {
		release, _ := s.Probes.Catalog.ActiveRelease(ctx, project, scope.Suite)
		q.Match = runs.ReliabilityMatch(test, environment, release)
	}
	if disabled {
		q.Match = "disabled"
	}
	service := reliability.Service{Store: s.Store}
	result, err := service.Checks(ctx, reader, q, reliability.Hash([]any{page.Token, page.Published, page.PublishedAt, page.Publisher, row}), cursor, limit)
	for i := range result.Items {
		result.Items[i].Run = ""
	}
	return result, err
}
