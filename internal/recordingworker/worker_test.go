package recordingworker

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/browserpolicy"
	"github.com/wtiger001/teststudio/internal/identity"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/recording"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/testkit"
)

type fixtureRuntime struct {
	starts, cleanups int
	failCleanup      bool
	drivers          map[string]*fixtureDriver
	store            *storage.Store
}
type fixtureDriver struct {
	sequence, inputs, stops int
	code                    string
	interrupt               bool
}

func (d *fixtureDriver) Start(context.Context, string, string) error { return nil }
func (d *fixtureDriver) Read(context.Context, string) (recording.Snapshot, error) {
	return recording.Snapshot{State: "recording", Code: d.code}, nil
}
func (d *fixtureDriver) Stop(context.Context, string) (recording.Snapshot, error) {
	d.stops++
	return recording.Snapshot{State: "ready", Code: d.code}, nil
}
func (d *fixtureDriver) View(context.Context, string) (recording.View, error) {
	return recording.View{Sequence: d.sequence, Image: "private-frame", Width: 1280, Height: 720, Steps: []recording.Step{}}, nil
}
func (d *fixtureDriver) Interact(ctx context.Context, id string, action recording.Action) (recording.View, error) {
	d.inputs++
	d.sequence = action.Sequence
	d.code = "captured News click and story assertion"
	if d.interrupt {
		d.interrupt = false
		return recording.View{}, errors.New("uncertain input")
	}
	return d.View(ctx, id)
}
func (r *fixtureRuntime) Plan(b Binding, target browserpolicy.Target) (Reference, error) {
	return Reference{Provider: "fixture", Container: b.ID, Fingerprint: b.Fingerprint()}, nil
}
func (r *fixtureRuntime) Start(ctx context.Context, b Binding, ref Reference, target browserpolicy.Target) (recording.InteractiveDriver, error) {
	if _, err := r.store.Get(ctx, LeaseBucket, b.ID); err != nil {
		return nil, errors.New("runtime started before durable reservation")
	}
	r.starts++
	d := &fixtureDriver{code: "captured navigation"}
	r.drivers[b.ID] = d
	return d, nil
}
func (r *fixtureRuntime) Cleanup(context.Context, Binding, Reference) error {
	r.cleanups++
	if r.failCleanup {
		return errors.New("cleanup pending")
	}
	return nil
}

func setupWorker(t *testing.T) (*Worker, *fixtureRuntime, *time.Time) {
	t.Helper()
	db, err := testkit.Open(t, filepath.Join(t.TempDir(), "worker.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	directory := &identity.Directory{Store: db}
	if err := directory.SaveUser(t.Context(), identity.User{ID: "admin", Name: "Administrator"}, 0); err != nil {
		t.Fatal(err)
	}
	auth, err := authorization.New(t.Context(), testkit.PermissionStore(t, db), directory)
	if err != nil {
		t.Fatal(err)
	}
	if err := auth.Engine.AssignRole(t.Context(), permissions.PrincipalRef{Kind: permissions.PrincipalUser, ID: "admin"}, "builtin.teststudio.installation-admin", nil); err != nil {
		t.Fatal(err)
	}
	p := &projects.Service{Store: db, Auth: auth}
	if err := p.SaveProject(t.Context(), "admin", projects.Project{ID: "recording", Name: "Recording"}, 0); err != nil {
		t.Fatal(err)
	}
	if err := p.SaveSuite(t.Context(), "admin", "recording", projects.Suite{ID: "browser", Name: "Browser"}, 0); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	runtime := &fixtureRuntime{store: db, drivers: map[string]*fixtureDriver{}}
	w := &Worker{Store: db, Auth: auth, Runtime: runtime, ID: "worker", Instance: "instance-one", Capacity: 1, IdleLimit: 30 * time.Second, Now: func() time.Time { return now }}
	if err := w.Open(t.Context()); err != nil {
		t.Fatal(err)
	}
	return w, runtime, &now
}

func newBinding(t *testing.T, w *Worker, id string) Binding {
	t.Helper()
	s := recording.Session{Input: recording.Input{ID: id, URL: "https://example.test/", TestID: "NEWS-" + id, Title: "News stories"}, Actor: "admin", Project: "recording", Suite: "browser", State: "recording", Mode: "embedded", CreatedAt: w.now(), DeadlineAt: w.now().Add(recording.Lifetime)}
	change, _ := storage.Put(recording.Bucket(), id, "recording", 0, s)
	if err := w.Store.Commit(t.Context(), change); err != nil {
		t.Fatal(err)
	}
	return Bind(s)
}

func TestWorkerReservationCapacitySequencingAndCleanupRecovery(t *testing.T) {
	w, runtime, now := setupWorker(t)
	b := newBinding(t, w, "first")
	if _, err := w.Call(t.Context(), "start", Request{Binding: b}); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Call(t.Context(), "start", Request{Binding: b}); err != nil || runtime.starts != 1 {
		t.Fatal("start retry opened another browser", err)
	}
	second := newBinding(t, w, "second")
	if _, err := w.Call(t.Context(), "start", Request{Binding: second}); err == nil {
		t.Fatal("worker capacity was exceeded")
	}
	input := recording.Action{Action: "click", Sequence: 1, X: 20, Y: 20}
	for i := 0; i < 2; i++ {
		if _, err := w.Call(t.Context(), "input", Request{Binding: b, Action: &input}); err != nil {
			t.Fatal(err)
		}
	}
	if runtime.drivers[b.ID].inputs != 1 {
		t.Fatal("exact input retry repeated a click")
	}
	input.X = 21
	if _, err := w.Call(t.Context(), "input", Request{Binding: b, Action: &input}); err == nil {
		t.Fatal("changed duplicate was accepted")
	}
	input.Sequence = 2
	runtime.drivers[b.ID].interrupt = true
	if _, err := w.Call(t.Context(), "input", Request{Binding: b, Action: &input}); err == nil {
		t.Fatal("uncertain input was hidden")
	}
	if _, err := w.Call(t.Context(), "view", Request{Binding: b}); err != nil {
		t.Fatal(err)
	}
	if runtime.drivers[b.ID].inputs != 2 {
		t.Fatal("view recovery replayed an input")
	}
	runtime.failCleanup = true
	if _, err := w.Call(t.Context(), "stop", Request{Binding: b}); err == nil {
		t.Fatal("failed cleanup was hidden")
	}
	if _, err := w.Call(t.Context(), "start", Request{Binding: second}); err == nil {
		t.Fatal("uncleaned browser released its capacity")
	}
	runtime.failCleanup = false
	*now = now.Add(time.Second)
	if err := w.Maintain(t.Context()); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Call(t.Context(), "start", Request{Binding: second}); err != nil {
		t.Fatal("cleaned capacity was not reusable", err)
	}
	if err := w.Close(t.Context()); err != nil {
		t.Fatal(err)
	}
}

func TestWorkerRestartFencesPreviousOwnerAndRetainsSourceWithoutRelaunch(t *testing.T) {
	w, runtime, now := setupWorker(t)
	b := newBinding(t, w, "restart")
	if _, err := w.Call(t.Context(), "start", Request{Binding: b}); err != nil {
		t.Fatal(err)
	}
	next := &Worker{Store: w.Store, Auth: w.Auth, Runtime: runtime, ID: w.ID, Instance: "instance-two", Capacity: 1, IdleLimit: w.IdleLimit, Now: w.Now}
	if err := next.Open(t.Context()); err == nil {
		t.Fatal("another process took a live worker identity")
	}
	*now = now.Add(OwnerLifetime + time.Second)
	if err := next.Open(t.Context()); err != nil {
		t.Fatal(err)
	}
	value, err := next.Call(t.Context(), "read", Request{Binding: b})
	if err != nil || value.Snapshot.State != "failed" || value.Snapshot.Code == "" || runtime.starts != 1 {
		t.Fatal("restart lost source or relaunched the browser", err)
	}
	if _, err := w.Call(t.Context(), "view", Request{Binding: b}); err == nil {
		t.Fatal("stale worker retained control")
	}
	if err := next.Close(t.Context()); err != nil {
		t.Fatal(err)
	}
}

func TestWorkerRechecksBindingArchiveAuthorityIdleAndDeadline(t *testing.T) {
	for _, reason := range []string{"binding", "archive", "authority", "idle", "deadline"} {
		t.Run(reason, func(t *testing.T) {
			w, runtime, now := setupWorker(t)
			b := newBinding(t, w, reason)
			if _, err := w.Call(t.Context(), "start", Request{Binding: b}); err != nil {
				t.Fatal(err)
			}
			switch reason {
			case "binding":
				other := b
				other.Actor = "another-user"
				if _, err := w.Call(t.Context(), "view", Request{Binding: other}); !errors.Is(err, authorization.ErrForbidden) {
					t.Fatal("binding escaped ownership", err)
				}
				if err := w.Close(t.Context()); err != nil {
					t.Fatal(err)
				}
				return
			case "archive":
				p := &projects.Service{Store: w.Store, Auth: w.Auth}
				if _, err := p.ChangeSuiteState(t.Context(), "admin", b.Project, b.Suite, "archive", 1); err != nil {
					t.Fatal(err)
				}
			case "authority":
				object := "recording/suite/browser"
				if err := w.Auth.Engine.CreateGrant(t.Context(), permissions.Grant{OwnerKind: permissions.PrincipalUser, OwnerID: "admin", TeamScope: b.Project, ObjectScope: &object, PermissionName: "teststudio.sources.manage", Effect: permissions.EffectDeny}); err != nil {
					t.Fatal(err)
				}
			case "idle":
				*now = now.Add(w.IdleLimit + time.Second)
				if _, err := w.Call(t.Context(), "read", Request{Binding: b}); err != nil {
					t.Fatal(err)
				}
			case "deadline":
				*now = b.Deadline
				poolRow, _ := w.Store.Get(t.Context(), PoolBucket, w.ID)
				change, _ := storage.Put(PoolBucket, w.ID, "", poolRow.Revision, Pool{Instance: w.Instance, Until: now.Add(OwnerLifetime)})
				if err := w.Store.Commit(t.Context(), change); err != nil {
					t.Fatal(err)
				}
			}
			if err := w.Maintain(t.Context()); err != nil {
				t.Fatal(err)
			}
			value, err := w.Store.Get(t.Context(), LeaseBucket, b.ID)
			if err != nil {
				t.Fatal(err)
			}
			lease, _ := storage.Decode[Lease](value)
			if lease.CleanupPending || lease.Snapshot.State != "failed" || lease.Snapshot.Code == "" || runtime.cleanups != 1 {
				t.Fatal("recording did not stop and retain its source")
			}
			if err := w.Close(t.Context()); err != nil {
				t.Fatal(err)
			}
		})
	}
}
