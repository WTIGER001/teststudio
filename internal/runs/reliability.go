package runs

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/reliability"
	"github.com/wtiger001/teststudio/internal/storage"
)

func ReliabilityMatch(test catalog.Test, environment *projects.Environment, release catalog.Release) string {
	var target any
	var credentials any
	if environment != nil {
		credentials = environment.Credentials
		if test.Check != nil {
			target = environment.Endpoints[test.Check.Endpoint]
		} else {
			target = environment.Endpoints
		}
	}
	return reliability.Hash([]any{target, credentials, test.Check, test.TimeoutSeconds, test.Timing, test.Source, test.Selector, test.Runner, test.RequiredBindings, release.Draft.Artifact, release.Draft.Timing})
}

func reliabilityObservations(run Run, results []execution.Result, missing bool) []reliability.Observation {
	apps := func(tests []string) []map[string]string {
		v := []map[string]string{}
		for _, app := range tests {
			v = append(v, map[string]string{"applicationId": app})
		}
		return v
	}
	selected, release := []string{}, []string{}
	for _, test := range run.Tests {
		selected = append(selected, test.Application)
	}
	for _, test := range run.Release.Draft.Tests {
		release = append(release, test.Application)
	}
	source := map[string]any{"applicationIds": run.Applications, "request": map[string]string{"applicationId": run.Request.Application}, "tests": apps(selected), "release": map[string]any{"catalog": map[string]any{"tests": apps(release)}}}
	if run.Environment != nil {
		source["environment"] = map[string]string{"applicationId": run.Environment.Application}
	}
	authority, _ := json.Marshal(source)
	authority, _ = authorization.CompactRunReadSource(authority)
	values := []reliability.Observation{}
	for _, test := range run.Tests {
		var result *execution.Result
		for i := range results {
			if results[i].TestID == test.ID {
				v := execution.NormalizeTimeout(results[i])
				result = &v
				break
			}
		}
		if result == nil && !missing {
			continue
		}
		at := run.UpdatedAt
		if run.FinishedAt != nil {
			at = *run.FinishedAt
		}
		if at.IsZero() {
			at = run.CreatedAt
		}
		context := timingContext(run, test)
		if prediction, ok := run.Timing.Tests[test.ID]; ok {
			context = prediction.Context
		}
		o := reliability.Observation{Scope: reliability.Scope{Project: run.Project, Suite: run.Request.Suite, Test: test.ID, Environment: run.Request.Environment, Location: run.Request.Location}, Run: run.ID, Title: test.Title, Type: test.Type, Runner: test.Runner, Context: reliability.Hash([]any{context, run.Release.Draft.Artifact, run.Release.Draft.Timing}), Match: ReliabilityMatch(test, run.Environment, run.Release), Created: run.CreatedAt, At: at, State: "unknown", MonitorState: "unknown", Source: authority}
		if result != nil {
			if !result.FinishedAt.IsZero() {
				o.At = result.FinishedAt
			}
			o.State = result.State
			o.Started = result.StartedAt
			o.Timing = result.Timing
			o.ObservedState = result.ObservedState
			o.FailureKind = result.FailureKind
			o.Censored = result.State == "timed-out" || result.State == "unknown" || result.Recovered || result.Timing != nil && result.Timing.Censored
			if !result.DurationUnavailable && result.State != "skipped" {
				duration := result.TotalDurationMs
				if duration == 0 {
					duration = result.DurationMs
				}
				o.DurationMs = &duration
			}
			if result.State == "passed" {
				o.MonitorState = "passed"
			} else if result.State == "flaky" || result.HTTP != nil || result.FailureKind == "assertion" || result.FailureKind == "timeout" || result.Timing != nil && result.Timing.Status == "slow" {
				o.MonitorState = result.State
			}
		}
		values = append(values, o)
	}
	return values
}

func (s *Service) commitReliability(ctx context.Context, run Run, results []execution.Result, missing bool, changes ...storage.Change) error {
	values := reliabilityObservations(run, results, missing)
	return s.Store.Transaction(ctx, func(bound *storage.Store, tx *sql.Tx) error {
		if err := bound.Commit(ctx, changes...); err != nil {
			return err
		}
		return reliability.Capture(ctx, tx, s.Store.Dialect, values)
	})
}

func (s *Service) reliabilityLoop(ctx context.Context) {
	service := reliability.Service{Store: s.Store}
	done := service.Start(ctx)
	defer func() { <-done }()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		if err := s.BackfillReliability(ctx, 100); err != nil && ctx.Err() != nil {
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *Service) BackfillReliability(ctx context.Context, limit int) error {
	row, err := s.Store.Get(ctx, "reliability-backfill", "v1")
	if err != nil && err != storage.ErrNotFound {
		return err
	}
	var checkpoint struct {
		Cursor   string `json:"cursor"`
		Complete bool   `json:"complete"`
	}
	if err == nil {
		if err = json.Unmarshal(row.Data, &checkpoint); err != nil {
			return err
		}
		if checkpoint.Complete {
			return nil
		}
	}
	page, err := s.Store.List(ctx, "runs", "", checkpoint.Cursor, limit)
	if err != nil {
		return err
	}
	for _, record := range page.Items {
		run, err := storage.Decode[Run](record)
		if err != nil {
			return err
		}
		if !Terminal(run.State) {
			continue
		}
		if run.Request.Location == "" {
			run.Request.Location = s.Location
		}
		if err = s.Store.Transaction(ctx, func(_ *storage.Store, tx *sql.Tx) error {
			return reliability.Capture(ctx, tx, s.Store.Dialect, reliabilityObservations(run, run.Results, true))
		}); err != nil {
			return err
		}
	}
	checkpoint.Cursor = page.Cursor
	checkpoint.Complete = page.Cursor == ""
	change, err := storage.Put("reliability-backfill", "v1", "", row.Revision, checkpoint)
	if err != nil {
		return err
	}
	err = s.Store.Commit(ctx, change)
	if err == storage.ErrConflict {
		return nil
	}
	return err
}

func (s *Service) ReliabilityReady(ctx context.Context) error {
	if err := s.BackfillReliability(ctx, 1000); err != nil {
		return err
	}
	row, err := s.Store.Get(ctx, "reliability-backfill", "v1")
	if err != nil {
		return err
	}
	var checkpoint struct {
		Complete bool `json:"complete"`
	}
	if err = json.Unmarshal(row.Data, &checkpoint); err != nil {
		return err
	}
	if !checkpoint.Complete {
		return reliability.ErrPending
	}
	return nil
}
