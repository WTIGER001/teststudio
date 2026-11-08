package reliability_test

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"testing"
	"time"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/identity"
	"github.com/wtiger001/teststudio/internal/reliability"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/testkit"
)

func fixture(t *testing.T) (*reliability.Service, *authorization.Service) {
	t.Helper()
	db, err := testkit.Open(t, filepath.Join(t.TempDir(), "history.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	dir := &identity.Directory{Store: db}
	if err = dir.SaveUser(t.Context(), identity.User{ID: "reader", Name: "Reader"}, 0); err != nil {
		t.Fatal(err)
	}
	auth, err := authorization.New(t.Context(), testkit.PermissionStore(t, db), dir)
	if err != nil {
		t.Fatal(err)
	}
	if err = auth.Engine.AssignRole(t.Context(), permissions.PrincipalRef{Kind: permissions.PrincipalUser, ID: "reader"}, "builtin.teststudio.installation-admin", nil); err != nil {
		t.Fatal(err)
	}
	return &reliability.Service{Store: db}, auth
}

func TestReliabilityCaptureRollsBackWithResultTransaction(t *testing.T) {
	s, _ := fixture(t)
	now := time.Now().UTC()
	abort := errors.New("abort result acceptance")
	o := reliability.Observation{Scope: reliability.Scope{Project: "project", Suite: "suite", Test: "test", Location: "local"}, Run: "run", At: now, Created: now, State: "passed", Source: json.RawMessage(`{}`)}
	err := s.Store.Transaction(t.Context(), func(bound *storage.Store, tx *sql.Tx) error {
		change, err := storage.Put("results", "result", "project", 0, map[string]string{"state": "passed"})
		if err != nil {
			return err
		}
		if err = bound.Commit(t.Context(), change); err != nil {
			return err
		}
		if err = reliability.Capture(t.Context(), tx, s.Store.Dialect, []reliability.Observation{o}); err != nil {
			return err
		}
		return abort
	})
	if !errors.Is(err, abort) {
		t.Fatal(err)
	}
	if _, err = s.Store.Get(t.Context(), "results", "result"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatal("result escaped rollback", err)
	}
	for _, table := range []string{"reliability_observations", "reliability_jobs", "reliability_series"} {
		var count int
		if err = s.Store.DB.QueryRowContext(t.Context(), "SELECT count(*) FROM "+table).Scan(&count); err != nil || count != 0 {
			t.Fatal("capture escaped rollback", table, count, err)
		}
	}
}

func TestReliabilityMultiYearReportsReuseDurableSummaries(t *testing.T) {
	verifyVolume(t, 1100, 1)
}

func TestReliabilityDenseYearUsesCompactSummaries(t *testing.T) {
	verifyVolume(t, 365, 288)
}

func verifyVolume(t *testing.T, days, perDay int) {
	t.Helper()
	s, auth := fixture(t)
	start := time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)
	scope := reliability.Scope{Project: "project", Suite: "suite", Test: "multi-year", Location: "local"}
	values := make([]reliability.Observation, days*perDay)
	for i := range values {
		at := start.Add(time.Duration(i) * (24 * time.Hour / time.Duration(perDay)))
		state := "passed"
		if i%(100*perDay) == 0 {
			state = "failed"
		}
		values[i] = reliability.Observation{Scope: scope, Run: fmt.Sprintf("run-%04d", i), At: at, Created: at.Add(-time.Second), State: state, MonitorState: state, Context: "stable", Source: json.RawMessage(`{}`)}
	}
	if err := s.Store.Transaction(t.Context(), func(_ *storage.Store, tx *sql.Tx) error {
		return reliability.Capture(t.Context(), tx, s.Store.Dialect, values)
	}); err != nil {
		t.Fatal(err)
	}
	q := reliability.Query{Scope: scope, Since: start, Until: start.Add(time.Duration(days) * 24 * time.Hour), MaxAge: 900}
	read := func() reliability.Row {
		t.Helper()
		batch, err := auth.NewRunReadBatch(t.Context(), "reader")
		if err != nil {
			t.Fatal(err)
		}
		row, err := s.Read(t.Context(), batch, q)
		if err != nil {
			t.Fatal(err)
		}
		return row
	}
	started := time.Now()
	first := read()
	t.Log("initial aggregation", time.Since(started))
	expectedDown := float64((days+99)/100) * min(900.0, 86400/float64(perDay))
	if first.Totals.Counts.Total() != days*perDay || first.Availability.Down != expectedDown || len(first.Buckets) > 60 {
		t.Fatal("multi-year aggregation mismatch", first.Totals, len(first.Buckets))
	}
	if _, err := s.Store.DB.ExecContext(t.Context(), `DELETE FROM reliability_observations WHERE series=$1`, scope.ID()); err != nil {
		t.Fatal(err)
	}
	started = time.Now()
	cached := read()
	t.Log("summary-only read after observation removal", time.Since(started))
	if cached.Totals.Counts != first.Totals.Counts || cached.Availability.Up != first.Availability.Up || cached.Availability.Down != first.Availability.Down {
		t.Fatal("report rescanned original observations instead of stored summaries", cached.Totals, first.Totals)
	}
	q.Since = time.Time{}
	all := read()
	if !all.Availability.Since.Equal(start) || all.Totals.Counts != first.Totals.Counts {
		t.Fatal("all-time discarded retained summary history", all.Availability, all.Totals)
	}
}

func TestReliabilityRollupsReplayEdgesFreshnessAndRevocation(t *testing.T) {
	s, auth := fixture(t)
	start := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	scope := reliability.Scope{Project: "project", Suite: "suite", Test: "check", Location: "local"}
	makeObservation := func(run, state string, at time.Time) reliability.Observation {
		duration := 100.0
		return reliability.Observation{Scope: scope, Run: run, Title: "Check", Context: "definition", Match: "target", Created: at.Add(-time.Minute), At: at, State: state, MonitorState: state, DurationMs: &duration, Source: json.RawMessage(`{}`)}
	}
	values := []reliability.Observation{makeObservation("before", "passed", start.Add(-5*time.Minute)), makeObservation("timeout", "timed-out", start.Add(10*time.Minute)), makeObservation("lost", "unknown", start.Add(30*time.Minute)), makeObservation("restored", "passed", start.Add(45*time.Minute))}
	values[1].Censored = true
	values[2].DurationMs = nil
	capture := func() {
		t.Helper()
		if err := s.Store.Transaction(t.Context(), func(_ *storage.Store, tx *sql.Tx) error {
			return reliability.Capture(t.Context(), tx, s.Store.Dialect, values)
		}); err != nil {
			t.Fatal(err)
		}
	}
	capture()
	capture()
	read := func(q reliability.Query) reliability.Row {
		t.Helper()
		reader, err := auth.NewRunReadBatch(t.Context(), "reader")
		if err != nil {
			t.Fatal(err)
		}
		row, err := s.Read(t.Context(), reader, q)
		if err != nil {
			t.Fatal(err)
		}
		return row
	}
	q := reliability.Query{Scope: scope, Since: start, Until: start.Add(time.Hour), MaxAge: 900, Resolution: "hour"}
	row := read(q)
	if len(row.Buckets) != 1 || row.Totals.Counts.Total() != 3 || row.Totals.Counts.TimedOut != 1 || row.Availability.Up != 1500 || row.Availability.Down != 900 || row.Availability.Unknown != 1200 || row.Totals.Censored != 1 || row.Totals.Unavailable != 1 {
		t.Fatalf("rollup: %+v", row)
	}
	if row.Availability.Percent == nil || math.Abs(*row.Availability.Percent-62.5) > .0001 {
		t.Fatal(row.Availability)
	}
	if row.State != "unknown" {
		t.Fatal("freshness expiry retained a healthy current state", row.State)
	}
	q.Since = start.Add(5 * time.Minute)
	q.Until = start.Add(55 * time.Minute)
	clipped := read(q)
	if clipped.Availability.Up != 900 || clipped.Availability.Down != 900 || clipped.Availability.Unknown != 1200 {
		t.Fatalf("edges: %+v", clipped.Availability)
	}
	object := "project/run/timeout"
	if err := auth.Engine.CreateGrant(t.Context(), permissions.Grant{OwnerKind: permissions.PrincipalUser, OwnerID: "reader", PermissionName: "teststudio.runs.read", TeamScope: "project", ObjectScope: &object, Effect: permissions.EffectDeny}); err != nil {
		t.Fatal(err)
	}
	denied := read(q)
	if denied.Availability.Down != 0 || denied.Totals.Counts.TimedOut != 0 || denied.Availability.Unknown != 2100 {
		t.Fatalf("denied observation leaked: %+v", denied)
	}
	q.MaxAge = 0
	outcomes := read(q)
	if outcomes.Metric != "outcomes" || outcomes.Availability.Percent != nil || outcomes.Totals.Counts.Passed != 1 {
		t.Fatal(outcomes)
	}
}

func TestReliabilityLateLogicalRunsAndDefinitionChanges(t *testing.T) {
	s, auth := fixture(t)
	start := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	scope := reliability.Scope{Project: "project", Suite: "suite", Test: "test", Location: "local"}
	values := []reliability.Observation{
		{Scope: scope, Run: "new", Context: "new", Match: "new", At: start.Add(10 * time.Minute), Created: start.Add(9 * time.Minute), State: "passed", MonitorState: "passed", Source: json.RawMessage(`{}`)},
		{Scope: scope, Run: "old-late", Context: "old", Match: "old", At: start.Add(12 * time.Minute), Created: start.Add(time.Minute), State: "failed", MonitorState: "failed", Source: json.RawMessage(`{}`)},
		{Scope: scope, Run: "changed", Context: "other", Match: "other", At: start.Add(20 * time.Minute), Created: start.Add(19 * time.Minute), State: "passed", MonitorState: "passed", Source: json.RawMessage(`{}`)},
	}
	if err := s.Store.Transaction(t.Context(), func(_ *storage.Store, tx *sql.Tx) error {
		return reliability.Capture(t.Context(), tx, s.Store.Dialect, values)
	}); err != nil {
		t.Fatal(err)
	}
	reader, err := auth.NewRunReadBatch(t.Context(), "reader")
	if err != nil {
		t.Fatal(err)
	}
	row, err := s.Read(t.Context(), reader, reliability.Query{Scope: scope, Since: start, Until: start.Add(time.Hour), MaxAge: 900, Match: "new"})
	if err != nil {
		t.Fatal(err)
	}
	if row.Availability.Up != 600 || row.Availability.Down != 0 || row.Totals.Counts.Passed != 1 || row.Contexts != 1 {
		t.Fatalf("late result or changed definition bridged coverage: %+v", row)
	}
}
