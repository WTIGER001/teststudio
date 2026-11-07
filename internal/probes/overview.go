package probes

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/wtiger001/teststudio/internal/catalog"
	"net/url"
	"reflect"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/paging"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/runs"
	"github.com/wtiger001/teststudio/internal/schedules"
	"github.com/wtiger001/teststudio/internal/storage"
)

func (s *Service) List(ctx context.Context, user, project, cursor string, q Query) (Page, error) {
	now := time.Now().UTC()
	page := Page{Items: []Summary{}, Now: now}
	if !domain.ValidID(project) || q.Limit < 1 || q.Limit > 100 || q.Hours < 1 || q.Hours > 720 {
		return page, errors.New("valid project, page limit 1 through 100, and window 1 through 720 hours are required")
	}
	if err := s.Auth.CheckCollection(ctx, user, project, "catalog.read"); err != nil {
		return page, err
	}
	page.Since = now.Add(-time.Duration(q.Hours) * time.Hour)
	bindings, err := s.Store.List(ctx, Bucket(project), project, "", 1000)
	if err != nil {
		return page, err
	}
	if bindings.Cursor != "" {
		return page, errors.New("probe configuration exceeds the 1000-row overview limit")
	}
	runPage := storage.Page{Items: []storage.Record{}}
	canReadRuns := s.Auth.CheckCollection(ctx, user, project, "runs.read") == nil
	if canReadRuns {
		runPage, err = s.Store.ListNewest(ctx, "runs", project, "", 1000)
		if err != nil {
			return page, err
		}
	}
	page.Scanned = len(runPage.Items)
	history := []runs.Run{}
	for _, row := range runPage.Items {
		v, err := storage.Decode[runs.Run](row)
		if err != nil {
			return page, err
		}
		if v.Project != project {
			continue
		}
		err = s.Auth.Check(ctx, user, project, "run/"+v.ID, "runs.read")
		if errors.Is(err, authorization.ErrForbidden) {
			continue
		}
		if err != nil {
			return page, err
		}
		history = append(history, runs.NormalizeTimeouts(v))
	}
	page.Partial = runPage.Cursor != ""
	scheduleRows := []storage.Record{}
	if s.Schedules != nil && s.Auth.CheckCollection(ctx, user, project, "schedules.read") == nil {
		sp, e := s.Store.List(ctx, schedules.Bucket(project), project, "", 1000)
		if e != nil {
			return page, e
		}
		if sp.Cursor != "" {
			return page, errors.New("schedule configuration exceeds the 1000-row overview limit")
		}
		for _, row := range sp.Items {
			if s.Auth.Check(ctx, user, project, "schedule/"+row.ID, "schedules.read") == nil {
				scheduleRows = append(scheduleRows, row)
			}
		}
	}
	all := []Summary{}
	for _, row := range bindings.Items {
		v, err := storage.Decode[Binding](row)
		if err != nil {
			return page, err
		}
		if v.Archived != q.Archived {
			continue
		}
		err = s.authority(ctx, user, project, v, false)
		if errors.Is(err, authorization.ErrForbidden) {
			continue
		}
		if err != nil {
			return page, err
		}
		result := s.summarize(ctx, user, project, v, row.Revision, history, scheduleRows, page.Partial, canReadRuns, now, page.Since)
		if q.Application != "" && result.Application != q.Application || q.Stage != "" && result.Stage != q.Stage || q.State != "" && result.State != q.State || q.Freshness != "" && result.Freshness != q.Freshness || q.Schedule != "" && result.ScheduleState != q.Schedule {
			continue
		}
		if q.Search != "" && !strings.Contains(strings.ToLower(v.Name+" "+result.URL), strings.ToLower(q.Search)) {
			continue
		}
		all = append(all, result)
	}
	sort.SliceStable(all, func(i, j int) bool {
		rank := func(v Summary) int {
			if v.State == "failed" || v.State == "timed-out" || v.Freshness == "stale" {
				return 0
			}
			if v.State != "passed" {
				return 1
			}
			return 2
		}
		if rank(all[i]) != rank(all[j]) {
			return rank(all[i]) < rank(all[j])
		}
		if strings.ToLower(all[i].Name) != strings.ToLower(all[j].Name) {
			return strings.ToLower(all[i].Name) < strings.ToLower(all[j].Name)
		}
		return all[i].ID < all[j].ID
	})
	page.Matched = len(all)
	raw, _ := json.Marshal([]any{user, project, q, all})
	digest := sha256.Sum256(raw)
	scope := hex.EncodeToString(digest[:])
	position, err := paging.Open(ctx, s.Store, "probe-pages", scope, cursor)
	if err != nil {
		return page, errors.New("probe results changed; refresh the overview before paging")
	}
	offset := 0
	if position != "" {
		offset, err = strconv.Atoi(position)
		if err != nil || offset < 0 || offset > len(all) {
			return page, errors.New("invalid probe cursor")
		}
	}
	end := min(offset+q.Limit, len(all))
	page.Items = all[offset:end]
	if end < len(all) {
		page.Cursor, err = paging.Seal(ctx, s.Store, "probe-pages", scope, strconv.Itoa(end))
	}
	return page, err
}
func (s *Service) summarize(ctx context.Context, user, project string, v Binding, revision int64, history []runs.Run, scheduleRows []storage.Record, partial, canReadRuns bool, now, since time.Time) Summary {
	result := Summary{Binding: v, Revision: revision, State: "never-checked", Freshness: "manual-only", Availability: "available", History: []Observation{}, Schedules: []Schedule{}, ScheduleState: "none", Partial: partial}
	t, env, _, err := s.references(ctx, user, project, v)
	if err != nil {
		result.Availability = "references-unavailable"
		result.Freshness = "unavailable"
		return result
	}
	result.Application = env.Application
	result.Stage = env.Stage
	desired := t.Test
	result.DraftChanged = true
	if active, e := s.Catalog.ActiveRelease(ctx, project, v.Suite); e == nil {
		result.ActiveRelease = active.ID
		for _, test := range active.Draft.Tests {
			if test.ID == v.Test {
				desired = test
				result.DraftChanged = !reflect.DeepEqual(test, t.Test)
				break
			}
		}
	}
	if desired.Check != nil {
		base, e := url.Parse(env.Endpoints[v.Endpoint])
		if e == nil && base.Host != "" {
			rel, _ := url.Parse(desired.Check.Path)
			base.Path = strings.TrimRight(base.Path, "/") + rel.Path
			base.RawQuery = ""
			base.Fragment = ""
			result.URL = base.String()
		}
	}
	if !t.Current || t.SuiteArchived || t.Test.State != "enabled" || env.Disabled {
		result.Availability = "disabled-or-removed"
	}
	suiteRow, e := s.Store.Get(ctx, projects.Bucket("suites", project), v.Suite)
	if e == nil {
		suite, _ := storage.Decode[projects.Suite](suiteRow)
		result.ActiveRelease = suite.ActiveRelease
	}
	if !canReadRuns {
		result.State = "unavailable"
		result.Freshness = "unavailable"
	}
	for _, run := range history {
		location := run.Request.Location
		if location == "" {
			location = s.Runs.Location
		}
		if run.Request.Suite != v.Suite || run.Request.Environment != v.Environment || location != v.Location {
			continue
		}
		testFound := false
		var snapshot catalog.Test
		for _, test := range run.Tests {
			if test.ID == v.Test && test.Check != nil && test.Check.Endpoint == v.Endpoint {
				testFound = true
				snapshot = test
				break
			}
		}
		if !testFound {
			continue
		}
		definitionChanged := !reflect.DeepEqual(snapshot.Check, desired.Check) || snapshot.TimeoutSeconds != desired.TimeoutSeconds || !reflect.DeepEqual(snapshot.Timing, desired.Timing)
		changed := run.Environment == nil || run.Environment.Endpoints[v.Endpoint] != env.Endpoints[v.Endpoint]
		observation := Observation{Run: run.ID, Release: run.Release.ID, CreatedAt: run.CreatedAt, At: run.FinishedAt, State: run.State, Error: run.Error, Evidence: run.EvidenceState, TargetChanged: changed, DefinitionChanged: definitionChanged}
		if !runs.Terminal(run.State) {
			if result.Active == nil {
				copy := observation
				result.Active = &copy
			}
			continue
		}
		for _, r := range run.Results {
			if r.TestID == v.Test {
				observation.State = r.State
				observation.Error = r.Error
				observation.HTTP = r.HTTP
				at := r.FinishedAt
				observation.At = &at
				break
			}
		}
		if !run.CreatedAt.Before(since) && len(result.History) < 30 {
			result.History = append(result.History, observation)
		}
		if !changed && !definitionChanged && result.Latest == nil {
			copy := observation
			result.Latest = &copy
			result.State = observation.State
		}
		if !changed && !definitionChanged && !run.CreatedAt.Before(since) {
			switch observation.State {
			case "passed", "failed", "timed-out", "flaky":
				result.Samples++
				if observation.State == "passed" {
					result.Passed++
				}
			default:
				result.Other++
			}
		}
	}
	if result.Samples > 0 && !partial {
		rate := 100 * float64(result.Passed) / float64(result.Samples)
		result.Rate = &rate
	}
	enabled := 0
	for _, row := range scheduleRows {
		schedule, e := storage.Decode[schedules.Schedule](row)
		if e != nil || schedule.Archived {
			continue
		}
		request := schedule.Request
		if request.Suite != v.Suite || request.Environment != v.Environment || request.Location != v.Location || len(request.IDs) > 0 && !slices.Contains(request.IDs, v.Test) {
			continue
		}
		result.Schedules = append(result.Schedules, Schedule{ID: schedule.ID, Name: schedule.Name, Enabled: schedule.Enabled, Kind: schedule.Kind, Interval: schedule.IntervalSeconds, Cron: schedule.Cron, Next: schedule.NextAt, LastState: schedule.LastState, Shared: len(request.IDs) != 1})
		result.ScheduleState = "paused"
		if schedule.Enabled {
			enabled++
			result.ScheduleState = "enabled"
			if result.Freshness != "unavailable" && result.Freshness != "stale" {
				result.Freshness = "current"
			}
			grace := time.Duration(max(request.TimeoutSeconds, 1))*time.Second + 60*time.Second
			due := schedule.NextAt
			if schedule.LastAt != nil && schedule.LastState != "skipped" && (due == nil || !now.After(due.Add(grace))) {
				due = schedule.LastAt
			}
			missing := result.Latest == nil || result.Latest.CreatedAt.Before(func() time.Time {
				if due != nil {
					return *due
				}
				return now
			}())
			if schedule.LastState == "blocked" || due != nil && now.After(due.Add(grace)) && missing {
				result.Freshness = "stale"
			}
		}
	}
	if len(result.Schedules) > 1 {
		result.ScheduleState = "multiple"
	} else if enabled > 0 {
		result.ScheduleState = "enabled"
	}
	if len(result.Schedules) > 0 && enabled == 0 && result.Freshness != "unavailable" {
		result.Freshness = "paused"
	}
	if result.Latest == nil && result.State == "never-checked" {
		if partial {
			result.State = "not-in-window"
		}
		if enabled > 0 && result.Freshness != "stale" {
			result.Freshness = "never-checked"
		}
	}
	return result
}
