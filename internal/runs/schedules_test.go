package runs_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/runs"
	"github.com/wtiger001/teststudio/internal/schedules"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/testkit"
)

func scheduleDefinition(id string) schedules.Definition {
	return schedules.Definition{ID: id, Name: "Local availability", Enabled: true, Kind: "interval", IntervalSeconds: 5, Timezone: "UTC", Overlap: "skip", Request: runs.Request{Suite: "health", Environment: "local", Location: "local", TimeoutSeconds: 30}}
}
func readSchedule(t *testing.T, s *schedules.Service, id string) (schedules.Schedule, storage.Record) {
	t.Helper()
	row, err := s.Store.Get(t.Context(), schedules.Bucket("project"), id)
	if err != nil {
		t.Fatal(err)
	}
	v, err := storage.Decode[schedules.Schedule](row)
	if err != nil {
		t.Fatal(err)
	}
	row.Revision = v.Generation
	return v, row
}
func tickSchedule(t *testing.T, s *schedules.Service) {
	t.Helper()
	if _, err := s.Tick(t.Context(), ""); err != nil {
		t.Fatal(err)
	}
}

func TestScheduleReplicasCreateOneNormalRunWithAtomicFiringAndTiming(t *testing.T) {
	r, _, count := setup(t, nil)
	clock := time.Now().UTC()
	s := &schedules.Service{Store: r.Store, Auth: r.Auth, Runs: r, Now: func() time.Time { return clock }}
	if err := s.Save(t.Context(), "admin", "project", scheduleDefinition("probe"), 0); err != nil {
		t.Fatal(err)
	}
	v, _ := readSchedule(t, s, "probe")
	clock = *v.NextAt
	var wg sync.WaitGroup
	problems := make(chan error, 12)
	for range 12 {
		wg.Go(func() { replica := *s; _, err := replica.Tick(t.Context(), ""); problems <- err })
	}
	wg.Wait()
	close(problems)
	for err := range problems {
		if err != nil {
			t.Fatal(err)
		}
	}
	v, _ = readSchedule(t, s, "probe")
	rows, err := r.Store.All(t.Context(), "runs", "project")
	if err != nil || len(rows) != 1 {
		t.Fatalf("duplicate scheduled run: %d %v", len(rows), err)
	}
	run, _, err := r.Get(t.Context(), "project", v.LastRun)
	if err != nil || run.Trigger != "schedule" || run.Schedule == nil || run.Schedule.ID != "probe" || !run.Schedule.DueAt.Equal(clock) || len(run.Timing.Tests) != 1 || run.Environment == nil || len(run.Tests) != 1 {
		t.Fatalf("scheduled snapshot: %+v %v", run, err)
	}
	fires, err := s.History(t.Context(), "admin", "project", "probe", "", 30)
	if err != nil || len(fires.Items) != 1 {
		t.Fatal("firing checkpoint missing", err)
	}
	firing, err := storage.Decode[schedules.Firing](fires.Items[0])
	if err != nil || firing.Run != run.ID || firing.State != "triggered" {
		t.Fatalf("atomic firing: %+v %v", firing, err)
	}
	start(t, r)
	final := finished(t, r, run.ID)
	if final.State != "completed" || final.Counts.Passed != 1 || count.Load() != 1 || final.EvidenceState != "complete" {
		t.Fatalf("scheduled execution differs from manual: %+v", final)
	}
}

type scheduleGate struct {
	goExecutor
	gated            atomic.Bool
	entered, release chan struct{}
}

func (e *scheduleGate) Admit(ctx context.Context, _ catalog.Test, _ catalog.Artifact) error {
	if e.gated.Load() {
		select {
		case e.entered <- struct{}{}:
		default:
		}
		select {
		case <-e.release:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}
func TestScheduleDisableFencesAnInFlightRunCreation(t *testing.T) {
	r, p, _ := setup(t, nil)
	configureGo(t, r, p)
	driver := &scheduleGate{goExecutor: goExecutor{state: "passed", inputs: make(chan execution.CodedInput, 1)}, entered: make(chan struct{}, 1), release: make(chan struct{})}
	r.Go = driver
	clock := time.Now().UTC()
	s := &schedules.Service{Store: r.Store, Auth: r.Auth, Runs: r, Now: func() time.Time { return clock }}
	d := scheduleDefinition("probe")
	d.Request.Environment = ""
	if err := s.Save(t.Context(), "admin", "project", d, 0); err != nil {
		t.Fatal(err)
	}
	v, _ := readSchedule(t, s, "probe")
	clock = *v.NextAt
	driver.gated.Store(true)
	done := make(chan error, 1)
	go func() { _, err := s.Tick(t.Context(), ""); done <- err }()
	select {
	case <-driver.entered:
	case <-time.After(3 * time.Second):
		t.Fatal("scheduler did not reach run admission")
	}
	_, row := readSchedule(t, s, "probe")
	d.Enabled = false
	if err := s.Save(t.Context(), "admin", "project", d, row.Revision); err != nil {
		close(driver.release)
		t.Fatal(err)
	}
	close(driver.release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	rows, err := r.Store.All(t.Context(), "runs", "project")
	if err != nil || len(rows) != 0 {
		t.Fatalf("disabled schedule left a ghost run: %d %v", len(rows), err)
	}
	fires, err := s.History(t.Context(), "admin", "project", "probe", "", 30)
	if err != nil || len(fires.Items) != 0 {
		t.Fatal("failed transaction committed a firing")
	}
	v, _ = readSchedule(t, s, "probe")
	if v.Enabled || v.NextAt != nil || v.Claim != "" {
		t.Fatal("disabled schedule retained work")
	}
}

func TestScheduleReclaimsExpiredClaimsAfterReopenAndCoalescesDowntime(t *testing.T) {
	r, _, _ := setup(t, nil)
	clock := time.Now().UTC()
	s := &schedules.Service{Store: r.Store, Auth: r.Auth, Runs: r, Now: func() time.Time { return clock }}
	if err := s.Save(t.Context(), "admin", "project", scheduleDefinition("probe"), 0); err != nil {
		t.Fatal(err)
	}
	v, row := readSchedule(t, s, "probe")
	due := *v.NextAt
	clock = due
	v.Claim = "lost-scheduler"
	until := clock.Add(30 * time.Second)
	v.ClaimUntil = &until
	c, _ := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, v)
	if err := s.Store.Commit(t.Context(), c); err != nil {
		t.Fatal(err)
	}
	tickSchedule(t, s)
	rows, _ := s.Store.All(t.Context(), "runs", "project")
	if len(rows) != 0 {
		t.Fatal("live scheduler claim was stolen")
	}
	path := ""
	if s.Store.Dialect == "sqlite" {
		var sequence int
		var name string
		if err := s.Store.DB.QueryRowContext(t.Context(), "PRAGMA database_list").Scan(&sequence, &name, &path); err != nil {
			t.Fatal(err)
		}
	}
	db, err := storage.Open(t.Context(), testkit.ConnectionString(t, s.Store, path))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	s.Store = db
	r.Store = db
	clock = until.Add(time.Second)
	tickSchedule(t, s)
	v, _ = readSchedule(t, s, "probe")
	rows, _ = s.Store.All(t.Context(), "runs", "project")
	if len(rows) != 1 || v.Claim != "" || v.NextAt == nil || !v.NextAt.Equal(due.Add(35*time.Second)) {
		t.Fatalf("reclaimed schedule: %+v, runs %d", v, len(rows))
	}
	fires, err := s.History(t.Context(), "admin", "project", "probe", "", 30)
	if err != nil {
		t.Fatal(err)
	}
	firing, err := storage.Decode[schedules.Firing](fires.Items[0])
	if err != nil || !firing.Coalesced {
		t.Fatal("downtime was not visible in the firing")
	}
}

func TestScheduleRecordsBlockedPermissionAndCatalogChanges(t *testing.T) {
	r, _, _ := setup(t, nil)
	clock := time.Now().UTC()
	s := &schedules.Service{Store: r.Store, Auth: r.Auth, Runs: r, Now: func() time.Time { return clock }}
	if err := s.Save(t.Context(), "admin", "project", scheduleDefinition("probe"), 0); err != nil {
		t.Fatal(err)
	}
	u, err := r.Auth.Directory.User(t.Context(), "admin")
	if err != nil {
		t.Fatal(err)
	}
	u.Disabled = true
	if err := r.Auth.Directory.SaveUser(t.Context(), u, 1); err != nil {
		t.Fatal(err)
	}
	v, _ := readSchedule(t, s, "probe")
	clock = *v.NextAt
	tickSchedule(t, s)
	v, _ = readSchedule(t, s, "probe")
	if v.LastState != "blocked" || v.LastError == "" || v.LastRun != "" {
		t.Fatalf("revoked owner silently executed: %+v", v)
	}
	u.Disabled = false
	if err := r.Auth.Directory.SaveUser(t.Context(), u, 2); err != nil {
		t.Fatal(err)
	}
	if err := r.Catalog.ChangeState(t.Context(), "admin", "project", "health", "HEALTH-1", "disabled", 1); err != nil {
		t.Fatal(err)
	}
	clock = *v.NextAt
	tickSchedule(t, s)
	v, _ = readSchedule(t, s, "probe")
	if v.LastState != "blocked" || v.LastRun != "" {
		t.Fatalf("disabled test silently executed: %+v", v)
	}
	d := scheduleDefinition("probe")
	d.Enabled = false
	if err := s.Save(t.Context(), "admin", "project", d, v.Generation); err != nil {
		t.Fatal(err)
	}
	if err := s.Save(t.Context(), "admin", "project", d, 1); !errors.Is(err, storage.ErrConflict) {
		t.Fatal("stale configuration overwrote scheduler state")
	}
}

func TestScheduleOverlapIsLocalToItsScheduleAndCanBeAllowed(t *testing.T) {
	r, _, _ := setup(t, nil)
	clock := time.Now().UTC()
	s := &schedules.Service{Store: r.Store, Auth: r.Auth, Runs: r, Now: func() time.Time { return clock }}
	d := scheduleDefinition("probe")
	if err := s.Save(t.Context(), "admin", "project", d, 0); err != nil {
		t.Fatal(err)
	}
	v, _ := readSchedule(t, s, "probe")
	clock = *v.NextAt
	tickSchedule(t, s)
	v, _ = readSchedule(t, s, "probe")
	first := v.LastRun
	clock = *v.NextAt
	tickSchedule(t, s)
	v, row := readSchedule(t, s, "probe")
	if v.LastState != "skipped" || v.LastRun != first {
		t.Fatal("overlap did not skip the still-active previous run")
	}
	d.Overlap = "allow"
	if err := s.Save(t.Context(), "admin", "project", d, row.Revision); err != nil {
		t.Fatal(err)
	}
	v, _ = readSchedule(t, s, "probe")
	clock = *v.NextAt
	tickSchedule(t, s)
	v, _ = readSchedule(t, s, "probe")
	if v.LastState != "triggered" || v.LastRun == first {
		t.Fatal("explicit overlapping execution was not admitted")
	}
	d.ID = "independent"
	d.Overlap = "skip"
	if err := s.Save(t.Context(), "admin", "project", d, 0); err != nil {
		t.Fatal(err)
	}
	v, _ = readSchedule(t, s, "independent")
	clock = *v.NextAt
	tickSchedule(t, s)
	v, _ = readSchedule(t, s, "independent")
	if v.LastState != "triggered" || v.LastRun == "" {
		t.Fatal("another schedule inherited an unrelated overlap lock")
	}
}

func TestSchedulesFollowActiveReleasesOrRetainTheirExplicitPin(t *testing.T) {
	r, _, _ := setup(t, nil)
	clock := time.Now().UTC()
	s := &schedules.Service{Store: r.Store, Auth: r.Auth, Runs: r, Now: func() time.Time { return clock }}
	probe, err := r.Create(t.Context(), "admin", "project", runs.Request{Suite: "health", Environment: "local"}, "find-release", "manual")
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"active", "pinned"} {
		d := scheduleDefinition(id)
		if id == "pinned" {
			d.Request.Release = probe.Release.ID
		}
		if err := s.Save(t.Context(), "admin", "project", d, 0); err != nil {
			t.Fatal(err)
		}
	}
	row, err := r.Store.Get(t.Context(), catalog.Bucket("tests", "project", "health"), "HEALTH-1")
	if err != nil {
		t.Fatal(err)
	}
	definition, err := storage.Decode[catalog.Test](row)
	if err != nil {
		t.Fatal(err)
	}
	definition.Check.StatusCodes = []int{201}
	if err := r.Catalog.SaveTest(t.Context(), "admin", "project", "health", definition, row.Revision); err != nil {
		t.Fatal(err)
	}
	draft, err := r.Catalog.CreateDraft(t.Context(), "admin", "project", "health", catalog.Artifact{})
	if err != nil {
		t.Fatal(err)
	}
	latest, err := r.Catalog.Publish(t.Context(), "admin", "project", "health", draft.ID, "next-release", 1, true)
	if err != nil {
		t.Fatal(err)
	}
	v, _ := readSchedule(t, s, "active")
	clock = *v.NextAt
	tickSchedule(t, s)
	for _, id := range []string{"active", "pinned"} {
		v, _ := readSchedule(t, s, id)
		run, _, err := r.Get(t.Context(), "project", v.LastRun)
		if err != nil {
			t.Fatal(err)
		}
		want := latest.ID
		if id == "pinned" {
			want = probe.Release.ID
		}
		if run.Release.ID != want {
			t.Fatalf("%s release changed: %s expected %s", id, run.Release.ID, want)
		}
	}
}

func TestScheduleConfigurationRevisionsStayStableAcrossFirings(t *testing.T) {
	r, _, _ := setup(t, nil)
	clock := time.Now().UTC()
	s := &schedules.Service{Store: r.Store, Auth: r.Auth, Runs: r, Now: func() time.Time { return clock }}
	d := scheduleDefinition("probe")
	if err := s.Save(t.Context(), "admin", "project", d, 0); err != nil {
		t.Fatal(err)
	}
	v, row, err := s.Read(t.Context(), "admin", "project", "probe")
	if err != nil {
		t.Fatal(err)
	}
	clock = *v.NextAt
	tickSchedule(t, s)
	latest, current, err := s.Read(t.Context(), "admin", "project", "probe")
	if err != nil {
		t.Fatal(err)
	}
	if row.Revision != current.Revision || latest.LastRun == "" {
		t.Fatal("runtime updates invalidated the editor configuration revision")
	}
	d.Name = "Renamed during execution"
	if err := s.Save(t.Context(), "admin", "project", d, row.Revision); err != nil {
		t.Fatal(err)
	}
	after, _, err := s.Read(t.Context(), "admin", "project", "probe")
	if err != nil || after.LastRun != latest.LastRun || after.Generation != row.Revision+1 {
		t.Fatal("configuration edit lost runtime state")
	}
	if err := s.Save(t.Context(), "admin", "project", d, row.Revision); !errors.Is(err, storage.ErrConflict) {
		t.Fatal("another editor's configuration change did not conflict")
	}
}
