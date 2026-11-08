package statuspages

import (
	"context"
	"strings"
	"time"

	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/reliability"
	"github.com/wtiger001/teststudio/internal/runs"
	"github.com/wtiger001/teststudio/internal/storage"
)

func (s *Service) Public(ctx context.Context, token, window string) (Report, error) {
	token = strings.ToLower(token)
	bucket := "public-status-pages"
	vanity := !tokenPattern.MatchString(token)
	if vanity {
		if value, err := NormalizeSlug(token); err != nil || value == "" {
			return Report{}, storage.ErrNotFound
		}
		bucket = "status-page-slugs"
	}
	link, err := s.Store.Get(ctx, bucket, token)
	if err != nil {
		return Report{}, storage.ErrNotFound
	}
	key, err := storage.Decode[slugLink](link)
	if err != nil {
		return Report{}, storage.ErrNotFound
	}
	row, err := s.Store.Get(ctx, Bucket(link.Project), key.ID)
	if err != nil {
		return Report{}, storage.ErrNotFound
	}
	page, err := storage.Decode[Page](row)
	if err != nil || page.Published == nil || (!vanity && page.Token != token) || (vanity && (page.Published.Slug != token || key.Token != page.Token)) {
		return Report{}, storage.ErrNotFound
	}
	if err = s.authority(ctx, page.Publisher, link.Project, key.ID, "read"); err != nil {
		return Report{}, storage.ErrNotFound
	}
	if err = s.authority(ctx, page.Publisher, link.Project, key.ID, "publish"); err != nil {
		return Report{}, storage.ErrNotFound
	}
	return s.Report(ctx, page.Publisher, link.Project, *page.Published, window)
}
func (s *Service) Report(ctx context.Context, user, project string, d Definition, window string) (Report, error) {
	now := time.Now().UTC()
	if window == "" {
		window = d.Window
	}
	since, err := WindowStart(window, now)
	if err != nil {
		return Report{}, err
	}
	report := Report{LogoHeight: d.LogoHeight, Logo: d.Logo, LogoAlt: d.LogoAlt, Title: d.Title, Background: d.Background, Window: window, MaxAge: d.MaxAge, At: now, Probes: []Row{}}
	projectRow, err := s.Store.Get(ctx, "projects", project)
	if err != nil {
		return report, err
	}
	p, err := storage.Decode[projects.Project](projectRow)
	if err != nil {
		return report, err
	}
	if p.Archived {
		return report, storage.ErrNotFound
	}
	if err = s.Probes.Runs.ReliabilityReady(ctx); err != nil {
		return report, err
	}
	reader, err := s.Auth.NewRunReadBatch(ctx, user)
	if err != nil {
		return report, err
	}
	service := reliability.Service{Store: s.Store}
	for _, entry := range d.Probes {
		scope, test, environment, disabled, err := s.resolve(ctx, user, project, entry)
		if err != nil {
			return report, storage.ErrNotFound
		}
		maxAge := 0
		if entry.Probe != "" || entry.Monitor {
			maxAge = d.MaxAge
		}
		match := ""
		if maxAge > 0 {
			release, _ := s.Probes.Catalog.ActiveRelease(ctx, project, scope.Suite)
			match = runs.ReliabilityMatch(test, environment, release)
		}
		if disabled {
			match = "disabled"
		}
		value, err := service.Read(ctx, reader, reliability.Query{Scope: scope, Since: since, Until: now, MaxAge: maxAge, Match: match})
		if err != nil {
			return report, err
		}
		row := Row{Title: entry.Title, Description: entry.Description, State: value.State, History: []Point{}, Metric: value.Metric, Buckets: value.Buckets, Totals: value.Totals, PassRate: value.PassRate, Contexts: value.Contexts, Availability: Availability{Percent: value.Availability.Percent, Coverage: value.Availability.Coverage, Up: value.Availability.Up, Down: value.Availability.Down, Unknown: value.Availability.Unknown, Since: value.Availability.Since}}
		for _, point := range value.History {
			row.History = append(row.History, Point{At: point.At, State: point.State})
		}
		report.Probes = append(report.Probes, row)
	}
	return report, reader.Current(ctx)
}
