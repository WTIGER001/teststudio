package statuspages

import (
	"context"
	"errors"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/probes"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/runs"
	"github.com/wtiger001/teststudio/internal/storage"
)

type source struct {
	binding     probes.Binding
	environment projects.Environment
	test        catalog.Test
	points      []Point
}

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
	report := Report{Title: d.Title, Background: d.Background, Window: window, MaxAge: d.MaxAge, At: now, Probes: []Row{}}
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
	sources := make([]source, len(d.Probes))
	for i, entry := range d.Probes {
		binding, _, err := s.Probes.Read(ctx, user, project, entry.Probe)
		if err != nil {
			return report, storage.ErrNotFound
		}
		sources[i].binding = binding
		sources[i].points = []Point{}
		if binding.Archived {
			continue
		}
		retained, err := s.Probes.Catalog.RetainedTest(ctx, user, project, binding.Suite, binding.Test)
		if err != nil {
			return report, storage.ErrNotFound
		}
		envRow, err := s.Store.Get(ctx, projects.Bucket("environments", project), binding.Environment)
		if err != nil {
			return report, storage.ErrNotFound
		}
		env, err := storage.Decode[projects.Environment](envRow)
		if err != nil {
			return report, err
		}
		if env.Disabled || retained.SuiteArchived || !retained.Current || retained.Test.State != "enabled" {
			continue
		}
		desired := retained.Test
		if release, e := s.Probes.Catalog.ActiveRelease(ctx, project, binding.Suite); e == nil {
			for _, test := range release.Draft.Tests {
				if test.ID == binding.Test {
					desired = test
					break
				}
			}
		}
		sources[i].environment = env
		sources[i].test = desired
	}
	reader, err := s.Auth.NewRunReadBatch(ctx, user)
	if err != nil {
		return report, err
	}
	cursor := ""
	for {
		batch, err := s.Store.List(ctx, "runs", project, cursor, 1000)
		if err != nil {
			return report, err
		}
		for _, row := range batch.Items {
			if err = ctx.Err(); err != nil {
				return report, err
			}
			run, err := storage.Decode[runs.Run](row)
			if err != nil {
				return report, err
			}
			if run.Project != project || !runs.Terminal(run.State) {
				continue
			}
			run = runs.NormalizeTimeouts(run)
			matching := []int{}
			for i, v := range sources {
				if v.test.Check != nil && v.binding.Suite == run.Request.Suite && v.binding.Environment == run.Request.Environment && (v.binding.Location == run.Request.Location || run.Request.Location == "" && v.binding.Location == s.Probes.Runs.Location) {
					matching = append(matching, i)
				}
			}
			if len(matching) == 0 {
				continue
			}
			permission := reader.Check(ctx, project, row)
			if permission != nil && !errors.Is(permission, authorization.ErrForbidden) {
				return report, permission
			}
			for _, i := range matching {
				v := &sources[i]
				state := "unknown"
				at := run.CreatedAt
				if run.FinishedAt != nil {
					at = *run.FinishedAt
				}
				same := run.Environment != nil && run.Environment.Endpoints[v.binding.Endpoint] == v.environment.Endpoints[v.binding.Endpoint]
				found := false
				for _, snapshot := range run.Tests {
					if snapshot.ID == v.binding.Test {
						found = true
						same = same && reflect.DeepEqual(snapshot.Check, v.test.Check) && snapshot.TimeoutSeconds == v.test.TimeoutSeconds && reflect.DeepEqual(snapshot.Timing, v.test.Timing)
						break
					}
				}
				if !found {
					continue
				}
				for _, result := range run.Results {
					if result.TestID == v.binding.Test {
						if !result.FinishedAt.IsZero() {
							at = result.FinishedAt
						}
						if permission == nil && same {
							switch result.State {
							case "passed":
								state = "passed"
							case "failed", "flaky":
								if result.HTTP != nil || result.FailureKind == "assertion" {
									state = result.State
								}
							case "timed-out":
								if result.FailureKind == "timeout" || result.HTTP != nil {
									state = "timed-out"
								}
							}
						}
						break
					}
				}
				if at.After(now) {
					continue
				}
				v.points = append(v.points, Point{At: at, State: state, Created: run.CreatedAt, Hidden: permission != nil})
			}
		}
		if batch.Cursor == "" {
			break
		}
		cursor = batch.Cursor
	}
	for i, entry := range d.Probes {
		points := sources[i].points
		sort.SliceStable(points, func(i, j int) bool {
			if points[i].At.Equal(points[j].At) {
				return points[i].Created.Before(points[j].Created)
			}
			return points[i].At.Before(points[j].At)
		})
		filtered := []Point{}
		var latest time.Time
		for _, point := range points {
			if !point.Created.Before(latest) {
				filtered = append(filtered, point)
				latest = point.Created
			}
		}
		points = filtered
		start := since
		if start.IsZero() {
			start = now
			for _, point := range points {
				if !point.Hidden {
					start = point.At
					break
				}
			}
		}
		row := Row{Title: entry.Title, Description: entry.Description, State: "unknown", History: []Point{}, Availability: Calculate(points, start, now, time.Duration(d.MaxAge)*time.Second)}
		for _, point := range points {
			if !point.Hidden && !point.At.Before(row.Availability.Since) {
				row.History = append(row.History, point)
			}
		}
		if len(row.History) > 30 {
			row.History = row.History[len(row.History)-30:]
		}
		if len(points) > 0 {
			last := points[len(points)-1]
			if now.Sub(last.At) <= time.Duration(d.MaxAge)*time.Second {
				row.State = last.State
			}
		}
		report.Probes = append(report.Probes, row)
	}
	if err := reader.Current(ctx); err != nil {
		return report, err
	}
	return report, nil
}
