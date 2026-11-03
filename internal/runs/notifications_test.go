package runs_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/adapters/notifications/webhook"
	localsecrets "github.com/wtiger001/teststudio/adapters/secrets/local"
	"github.com/wtiger001/teststudio/internal/networkguard"
	"github.com/wtiger001/teststudio/internal/notifications"
	"github.com/wtiger001/teststudio/internal/runs"
	"github.com/wtiger001/teststudio/internal/schedules"
	"github.com/wtiger001/teststudio/internal/secrets"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/testkit"
)

type noticeFixture struct {
	service  *notifications.Service
	clock    time.Time
	calls    atomic.Int32
	messages chan notifications.Message
}

func notices(t *testing.T, r *runs.Service, status func(int, http.ResponseWriter)) *noticeFixture {
	t.Helper()
	f := &noticeFixture{clock: time.Now().UTC(), messages: make(chan notifications.Message, 100)}
	receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		n := f.calls.Add(1)
		var message notifications.Message
		if err := json.NewDecoder(req.Body).Decode(&message); err != nil {
			t.Error(err)
		}
		if req.Header.Get("X-TestStudio-Delivery-ID") != message.Delivery {
			t.Error("unstable notification identity")
		}
		f.messages <- message
		if status != nil {
			status(int(n), w)
		} else {
			w.WriteHeader(204)
		}
	}))
	t.Cleanup(receiver.Close)
	t.Setenv("TESTSTUDIO_TEST_NOTIFICATION_URL", receiver.URL+"/fresh-private-notification-token")
	provider, err := localsecrets.New([]localsecrets.Binding{{ID: "alerts", Project: "project", Purpose: secrets.NotificationWebhook, Environment: "TESTSTUDIO_TEST_NOTIFICATION_URL", Network: networkguard.Loopback}, {ID: "alternate", Project: "project", Purpose: secrets.NotificationWebhook, Environment: "TESTSTUDIO_TEST_NOTIFICATION_URL", Network: networkguard.Loopback}})
	if err != nil {
		t.Fatal(err)
	}
	f.service = &notifications.Service{Store: r.Store, Auth: r.Auth, Secrets: provider, Sender: &webhook.Sender{}, Origin: "https://studio.invalid", Now: func() time.Time { return f.clock }}
	r.Completion = f.service
	return f
}
func noticeDefinition() notifications.Definition {
	return notifications.Definition{ID: "availability", Name: "Availability alerts", Enabled: true, Kind: "webhook", Reference: "alerts", Outcomes: []string{"passed", "failed", "timed-out", "flaky", "unknown"}, MaxAttempts: 3}
}
func saveNotice(t *testing.T, f *noticeFixture) {
	t.Helper()
	if err := f.service.SaveChannel(t.Context(), "admin", "project", noticeDefinition(), 0); err != nil {
		t.Fatal(err)
	}
}
func noticeRows(t *testing.T, f *noticeFixture) []storage.Record {
	t.Helper()
	rows, err := f.service.Store.All(t.Context(), notifications.Deliveries("project"), "project")
	if err != nil {
		t.Fatal(err)
	}
	return rows
}
func noticeValue(t *testing.T, row storage.Record) notifications.Delivery {
	t.Helper()
	v, err := storage.Decode[notifications.Delivery](row)
	if err != nil {
		t.Fatal(err)
	}
	return v
}
func tickNotices(t *testing.T, f *noticeFixture) {
	t.Helper()
	if _, err := f.service.Tick(t.Context(), ""); err != nil {
		t.Fatal(err)
	}
}
func queueNoticeRun(t *testing.T, r *runs.Service, key string) runs.Run {
	t.Helper()
	v, err := r.Create(t.Context(), "admin", "project", runs.Request{Suite: "health", Environment: "local"}, key, "manual")
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func TestNotificationsShareManualAndScheduledCompletionAndClaimEachDeliveryOnce(t *testing.T) {
	r, _, executions := setup(t, nil)
	f := notices(t, r, nil)
	saveNotice(t, f)
	manual := queueNoticeRun(t, r, "manual-notification")
	scheduler := &schedules.Service{Store: r.Store, Auth: r.Auth, Runs: r, Now: func() time.Time { return f.clock }}
	if err := scheduler.Save(t.Context(), "admin", "project", scheduleDefinition("scheduled-notification"), 0); err != nil {
		t.Fatal(err)
	}
	schedule, _, err := scheduler.Read(t.Context(), "admin", "project", "scheduled-notification")
	if err != nil {
		t.Fatal(err)
	}
	f.clock = *schedule.NextAt
	if _, err := scheduler.Tick(t.Context(), ""); err != nil {
		t.Fatal(err)
	}
	schedule, _, err = scheduler.Read(t.Context(), "admin", "project", schedule.ID)
	if err != nil {
		t.Fatal(err)
	}
	start(t, r)
	finished(t, r, manual.ID)
	finished(t, r, schedule.LastRun)
	rows := noticeRows(t, f)
	if len(rows) != 2 {
		t.Fatalf("completion outbox: %d", len(rows))
	}
	var workers sync.WaitGroup
	problems := make(chan error, 8)
	for range 8 {
		workers.Go(func() { replica := *f.service; _, err := replica.Tick(t.Context(), ""); problems <- err })
	}
	workers.Wait()
	close(problems)
	for err := range problems {
		if err != nil {
			t.Fatal(err)
		}
	}
	if f.calls.Load() != 2 || executions.Load() != 2 {
		t.Fatal("notification retries duplicated runs or messages")
	}
	triggers := map[string]bool{}
	for range 2 {
		m := <-f.messages
		triggers[m.Trigger] = true
		if m.Counts.Passed != 1 || m.Counts.Total != 1 || m.Outcome != "passed" || m.URL != "https://studio.invalid/projects/project/runs/"+m.Run {
			t.Fatalf("notification summary: %+v", m)
		}
	}
	if !triggers["manual"] || !triggers["schedule"] {
		t.Fatal("manual and scheduled runs used different completion paths")
	}
	for _, row := range noticeRows(t, f) {
		v := noticeValue(t, row)
		if v.State != "delivered" || v.Attempts != 1 || v.DeliveredAt == nil {
			t.Fatalf("delivery checkpoint: %+v", v)
		}
		if strings.Contains(string(row.Data), "fresh-private-notification-token") {
			t.Fatal("resolved secret reached durable metadata")
		}
	}
	tickNotices(t, f)
	if f.calls.Load() != 2 {
		t.Fatal("completed notification was redelivered")
	}
}

func TestNotificationPlansFreezeAtQueueTimeWithoutRetroactiveOrRefilteredDelivery(t *testing.T) {
	r, _, _ := setup(t, nil)
	f := notices(t, r, nil)
	before := queueNoticeRun(t, r, "before-channel")
	d := noticeDefinition()
	d.Outcomes = []string{"passed"}
	if err := f.service.SaveChannel(t.Context(), "admin", "project", d, 0); err != nil {
		t.Fatal(err)
	}
	frozen := queueNoticeRun(t, r, "frozen-channel")
	d.Outcomes = []string{"failed"}
	if err := f.service.SaveChannel(t.Context(), "admin", "project", d, 1); err != nil {
		t.Fatal(err)
	}
	start(t, r)
	finished(t, r, before.ID)
	finished(t, r, frozen.ID)
	rows := noticeRows(t, f)
	if len(rows) != 1 || noticeValue(t, rows[0]).Run != frozen.ID {
		t.Fatal("notification changes rewrote queued run policy")
	}
	tickNotices(t, f)
	if f.calls.Load() != 1 {
		t.Fatal("frozen notification did not deliver")
	}
}

func TestNotificationRateLimitsDelayOtherRunsWithoutChangingTheirResults(t *testing.T) {
	r, _, _ := setup(t, nil)
	f := notices(t, r, func(n int, w http.ResponseWriter) {
		if n == 1 {
			w.Header().Set("Retry-After", "30")
			w.WriteHeader(429)
		} else {
			w.WriteHeader(204)
		}
	})
	saveNotice(t, f)
	one := queueNoticeRun(t, r, "limited-one")
	two := queueNoticeRun(t, r, "limited-two")
	start(t, r)
	finished(t, r, one.ID)
	finished(t, r, two.ID)
	tickNotices(t, f)
	if f.calls.Load() != 1 {
		t.Fatal("destination rate limit did not stop other queued sends")
	}
	for _, row := range noticeRows(t, f) {
		v := noticeValue(t, row)
		if v.State != "retrying" || v.NextAt.Before(f.clock.Add(30*time.Second)) {
			t.Fatalf("retry deadline: %+v", v)
		}
	}
	tickNotices(t, f)
	if f.calls.Load() != 1 {
		t.Fatal("notification retried before its deadline")
	}
	f.clock = f.clock.Add(31 * time.Second)
	tickNotices(t, f)
	if f.calls.Load() != 3 {
		t.Fatalf("expected one rate-limited send and two confirmations, got %d", f.calls.Load())
	}
	for _, id := range []string{one.ID, two.ID} {
		run, _, err := r.Get(t.Context(), "project", id)
		if err != nil || run.Outcome != "passed" || run.Counts.Passed != 1 {
			t.Fatal("notification failure changed a functional result")
		}
	}
	seen := map[string]int{}
	for range 3 {
		seen[(<-f.messages).Delivery]++
	}
	if len(seen) != 2 {
		t.Fatal("retry changed the delivery identifier")
	}
}

type interruptedNoticeSender struct {
	next   notifications.Sender
	cancel context.CancelFunc
}

func (s interruptedNoticeSender) Send(ctx context.Context, kind string, value secrets.Value, message notifications.Message) notifications.Outcome {
	result := s.next.Send(ctx, kind, value, message)
	s.cancel()
	return result
}
func TestNotificationRestartRetriesUncheckpointedAcceptanceWithTheSameIdentity(t *testing.T) {
	r, _, executions := setup(t, nil)
	f := notices(t, r, nil)
	saveNotice(t, f)
	run := queueNoticeRun(t, r, "accept-before-crash")
	start(t, r)
	finished(t, r, run.ID)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	f.service.Sender = interruptedNoticeSender{next: &webhook.Sender{}, cancel: cancel}
	if _, err := f.service.Tick(ctx, ""); !errors.Is(err, context.Canceled) {
		t.Fatalf("interruption not retained: %v", err)
	}
	rows := noticeRows(t, f)
	first := noticeValue(t, rows[0])
	if first.State != "sending" || first.Attempts != 1 || f.calls.Load() != 1 {
		t.Fatal("uncheckpointed acceptance was erased")
	}
	path := ""
	if f.service.Store.Dialect == "sqlite" {
		var sequence int
		var name string
		if err := f.service.Store.DB.QueryRowContext(t.Context(), "PRAGMA database_list").Scan(&sequence, &name, &path); err != nil {
			t.Fatal(err)
		}
	}
	db, err := storage.Open(t.Context(), testkit.ConnectionString(t, f.service.Store, path))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	replica := *f.service
	replica.Store = db
	replica.Sender = &webhook.Sender{}
	f.service = &replica
	f.clock = first.ClaimUntil.Add(time.Second)
	tickNotices(t, f)
	rows = noticeRows(t, f)
	last := noticeValue(t, rows[0])
	if len(rows) != 1 || last.State != "delivered" || last.Attempts != 2 || !last.AcceptanceUnknown || f.calls.Load() != 2 || executions.Load() != 1 {
		t.Fatalf("at-least-once recovery: %+v", last)
	}
	a := <-f.messages
	b := <-f.messages
	if a.Delivery != b.Delivery || a.Run != b.Run {
		t.Fatal("recovery created a new logical notification or run")
	}
	history, err := f.service.History(t.Context(), "admin", "project", last.ID, "", 30)
	if err != nil || len(history.Items) != 2 {
		t.Fatal("attempt history missing", err)
	}
	old, err := storage.Decode[notifications.Attempt](history.Items[1])
	if err != nil || old.State != "unknown" || !old.AcceptanceUnknown {
		t.Fatal("interrupted acceptance became a known failure")
	}
}

func TestNotificationsStopForPausedChangedOrRevokedChannelsAndRetryExplicitly(t *testing.T) {
	for _, condition := range []string{"paused", "changed", "revoked"} {
		t.Run(condition, func(t *testing.T) {
			r, _, _ := setup(t, nil)
			f := notices(t, r, nil)
			saveNotice(t, f)
			run := queueNoticeRun(t, r, "stopped-notification")
			start(t, r)
			finished(t, r, run.ID)
			d := noticeDefinition()
			switch condition {
			case "paused":
				d.Enabled = false
			case "changed":
				d.Reference = "alternate"
			case "revoked":
				user, err := r.Auth.Directory.User(t.Context(), "admin")
				if err != nil {
					t.Fatal(err)
				}
				user.Disabled = true
				if err = r.Auth.Directory.SaveUser(t.Context(), user, 1); err != nil {
					t.Fatal(err)
				}
			}
			if condition != "revoked" {
				if err := f.service.SaveChannel(t.Context(), "admin", "project", d, 1); err != nil {
					t.Fatal(err)
				}
			}
			tickNotices(t, f)
			row := noticeRows(t, f)[0]
			v := noticeValue(t, row)
			want := "cancelled"
			if condition == "revoked" {
				want = "blocked"
			}
			if v.State != want || f.calls.Load() != 0 {
				t.Fatalf("stopped channel sent a notification: %+v", v)
			}
			if condition == "revoked" {
				user, err := r.Auth.Directory.User(t.Context(), "admin")
				if err != nil {
					t.Fatal(err)
				}
				user.Disabled = false
				if err = r.Auth.Directory.SaveUser(t.Context(), user, 2); err != nil {
					t.Fatal(err)
				}
				if err := f.service.Retry(t.Context(), "admin", "project", v.ID, row.Revision); err != nil {
					t.Fatal(err)
				}
				tickNotices(t, f)
				v = noticeValue(t, noticeRows(t, f)[0])
				if v.State != "delivered" || f.calls.Load() != 1 {
					t.Fatal("authorized explicit retry did not resume delivery")
				}
			}
		})
	}
}

func TestNotificationsCaptureQueuedCancellationDeadlinesAndLostWorkers(t *testing.T) {
	for _, condition := range []string{"cancelled", "deadline", "lost-worker"} {
		t.Run(condition, func(t *testing.T) {
			r, _, _ := setup(t, nil)
			f := notices(t, r, nil)
			saveNotice(t, f)
			run := queueNoticeRun(t, r, "terminal-notification")
			if condition == "cancelled" {
				if _, err := r.Cancel(t.Context(), "admin", "project", run.ID); err != nil {
					t.Fatal(err)
				}
			} else {
				current, row, err := r.Get(t.Context(), "project", run.ID)
				if err != nil {
					t.Fatal(err)
				}
				changes := []storage.Change{}
				if condition == "deadline" {
					current.Deadline = time.Now().Add(-time.Second)
				} else {
					current.State = "running"
					current.Worker = "lost-worker"
					until := time.Now().Add(-time.Minute)
					current.LeaseUntil = &until
					lease, err := storage.Put("run-leases", run.ID, "project", 0, run.ID)
					if err != nil {
						t.Fatal(err)
					}
					queued, err := r.Store.Get(t.Context(), "run-queue", run.ID)
					if err != nil {
						t.Fatal(err)
					}
					changes = append(changes, lease, storage.Change{Record: queued, Delete: true})
				}
				change, err := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, current)
				if err != nil {
					t.Fatal(err)
				}
				if err = r.Store.Commit(t.Context(), append(changes, change)...); err != nil {
					t.Fatal(err)
				}
			}
			start(t, r)
			terminal := finished(t, r, run.ID)
			rows := noticeRows(t, f)
			if len(rows) != 1 {
				t.Fatalf("terminal path lost its outbox: %s rows %d", terminal.State, len(rows))
			}
			v := noticeValue(t, rows[0])
			if v.Message.State != terminal.State || v.Message.Outcome != terminal.Outcome || v.Message.Counts.Completed != 0 {
				t.Fatalf("terminal notification invented execution results: %+v", v.Message)
			}
			tickNotices(t, f)
			if f.calls.Load() != 1 {
				t.Fatal("terminal notification did not deliver")
			}
		})
	}
}

func TestNotificationEvidenceGapPolicyDoesNotChangeTheObservedPass(t *testing.T) {
	r, _, _ := setup(t, nil)
	f := notices(t, r, nil)
	d := noticeDefinition()
	d.Outcomes = []string{"failed"}
	d.EvidenceGaps = true
	if err := f.service.SaveChannel(t.Context(), "admin", "project", d, 0); err != nil {
		t.Fatal(err)
	}
	r.Evidence = nil
	run := queueNoticeRun(t, r, "capture-gap-notification")
	start(t, r)
	terminal := finished(t, r, run.ID)
	rows := noticeRows(t, f)
	if terminal.Outcome != "passed" || terminal.EvidenceState != "incomplete" || len(rows) != 1 {
		t.Fatalf("capture gap policy: %+v", terminal)
	}
	tickNotices(t, f)
	message := <-f.messages
	if message.Outcome != "passed" || message.Evidence != "incomplete" || message.Counts.Passed != 1 {
		t.Fatal("capture gap notification erased the functional pass")
	}
}

func TestNotificationAttemptBudgetStopsAutomaticRetriesAndCanBeCancelled(t *testing.T) {
	r, _, _ := setup(t, nil)
	f := notices(t, r, func(_ int, w http.ResponseWriter) { w.WriteHeader(503) })
	d := noticeDefinition()
	d.MaxAttempts = 2
	if err := f.service.SaveChannel(t.Context(), "admin", "project", d, 0); err != nil {
		t.Fatal(err)
	}
	run := queueNoticeRun(t, r, "bounded-retry")
	start(t, r)
	finished(t, r, run.ID)
	tickNotices(t, f)
	first := noticeValue(t, noticeRows(t, f)[0])
	f.clock = first.NextAt
	tickNotices(t, f)
	row := noticeRows(t, f)[0]
	v := noticeValue(t, row)
	if v.State != "failed" || v.Attempts != 2 || f.calls.Load() != 2 {
		t.Fatalf("unbounded retry: %+v", v)
	}
	f.clock = f.clock.Add(time.Hour)
	tickNotices(t, f)
	if f.calls.Load() != 2 {
		t.Fatal("exhausted notification sent again")
	}
	if err := f.service.Retry(t.Context(), "admin", "project", v.ID, row.Revision); err != nil {
		t.Fatal(err)
	}
	row = noticeRows(t, f)[0]
	if err := f.service.Cancel(t.Context(), "admin", "project", v.ID, row.Revision); err != nil {
		t.Fatal(err)
	}
	tickNotices(t, f)
	v = noticeValue(t, noticeRows(t, f)[0])
	if v.State != "cancelled" || f.calls.Load() != 2 {
		t.Fatal("cancelled retry still sent")
	}
}

func TestNotificationPauseAndResumeDoesNotReviveOldQueuedDeliveries(t *testing.T) {
	r, _, _ := setup(t, nil)
	f := notices(t, r, nil)
	saveNotice(t, f)
	run := queueNoticeRun(t, r, "pause-before-worker")
	start(t, r)
	finished(t, r, run.ID)
	d := noticeDefinition()
	d.Enabled = false
	if err := f.service.SaveChannel(t.Context(), "admin", "project", d, 1); err != nil {
		t.Fatal(err)
	}
	d.Enabled = true
	if err := f.service.SaveChannel(t.Context(), "admin", "project", d, 2); err != nil {
		t.Fatal(err)
	}
	tickNotices(t, f)
	row := noticeRows(t, f)[0]
	v := noticeValue(t, row)
	if v.State != "cancelled" || f.calls.Load() != 0 {
		t.Fatal("reenabling a channel silently revived an old queued message")
	}
	if err := f.service.Retry(t.Context(), "admin", "project", v.ID, row.Revision); err != nil {
		t.Fatal(err)
	}
	tickNotices(t, f)
	v = noticeValue(t, noticeRows(t, f)[0])
	if v.State != "delivered" || f.calls.Load() != 1 || v.DispatchEpoch == v.Destination.Channel.Epoch {
		t.Fatal("explicit retry did not retain its new authorization and old snapshot")
	}
}

type globallyLimitedNoticeSender struct{ calls atomic.Int32 }

func (s *globallyLimitedNoticeSender) Send(context.Context, string, secrets.Value, notifications.Message) notifications.Outcome {
	if s.calls.Add(1) == 1 {
		return notifications.Outcome{HTTPStatus: 429, Retryable: true, GlobalRateLimit: true, RetryAfter: 30 * time.Second, Error: "Provider rate limit."}
	}
	return notifications.Outcome{HTTPStatus: 204}
}
func TestNotificationProviderRateLimitsApplyAcrossDiscordDestinations(t *testing.T) {
	r, _, _ := setup(t, nil)
	f := notices(t, r, nil)
	sender := &globallyLimitedNoticeSender{}
	f.service.Sender = sender
	d := noticeDefinition()
	d.Kind = "discord"
	if err := f.service.SaveChannel(t.Context(), "admin", "project", d, 0); err != nil {
		t.Fatal(err)
	}
	d.ID = "second"
	d.Reference = "alternate"
	if err := f.service.SaveChannel(t.Context(), "admin", "project", d, 0); err != nil {
		t.Fatal(err)
	}
	run := queueNoticeRun(t, r, "provider-limited")
	start(t, r)
	finished(t, r, run.ID)
	tickNotices(t, f)
	if sender.calls.Load() != 1 {
		t.Fatal("global provider limit did not stop a different destination")
	}
	for _, row := range noticeRows(t, f) {
		v := noticeValue(t, row)
		if v.State != "retrying" || v.NextAt.Before(f.clock.Add(30*time.Second)) {
			t.Fatal("global provider cooldown was not reflected in pending delivery")
		}
	}
	f.clock = f.clock.Add(31 * time.Second)
	tickNotices(t, f)
	if sender.calls.Load() != 3 {
		t.Fatal("global provider cooldown did not permit later delivery")
	}
}

func TestNotificationHistoryCursorsStayBoundToProjectAndRun(t *testing.T) {
	r, _, _ := setup(t, nil)
	f := notices(t, r, nil)
	saveNotice(t, f)
	one := queueNoticeRun(t, r, "history-one")
	two := queueNoticeRun(t, r, "history-two")
	start(t, r)
	finished(t, r, one.ID)
	finished(t, r, two.ID)
	page, err := f.service.ListDeliveries(t.Context(), "admin", "project", "", "", 1)
	if err != nil || len(page.Items) != 1 || page.Cursor == "" {
		t.Fatal("bounded delivery history missing", err)
	}
	if _, err := f.service.ListDeliveries(t.Context(), "admin", "project", one.ID, page.Cursor, 1); err == nil {
		t.Fatal("unfiltered cursor silently changed its run scope")
	}
	if _, err := f.service.ListDeliveries(t.Context(), "admin", "other", "", page.Cursor, 1); err == nil {
		t.Fatal("cursor crossed a project boundary")
	}
	filtered, err := f.service.ListDeliveries(t.Context(), "admin", "project", one.ID, "", 1)
	if err != nil || len(filtered.Items) != 1 || noticeValue(t, filtered.Items[0]).Run != one.ID {
		t.Fatal("run history stopped before matching rows", err)
	}
}
