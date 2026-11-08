package recordingworker

import (
	"context"
	"errors"
	"net/url"
	"sync"
	"time"

	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/browserpolicy"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/recording"
	"github.com/wtiger001/teststudio/internal/storage"
)

type Worker struct {
	Store        *storage.Store
	Auth         *authorization.Service
	Runtime      Runtime
	ID           string
	Instance     string
	Capacity     int
	IdleLimit    time.Duration
	AllowPrivate bool
	Origins      OriginPolicy
	Now          func() time.Time
	mu           sync.Mutex
	drivers      map[string]recording.InteractiveDriver
}

func (w *Worker) now() time.Time {
	if w.Now != nil {
		return w.Now().UTC()
	}
	return time.Now().UTC()
}

func (w *Worker) Open(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.drivers != nil {
		return errors.New("recording worker is already open")
	}
	if !domain.ValidID(w.ID) || !domain.ValidID(w.Instance) || w.Capacity < 1 || w.Capacity > 16 || w.IdleLimit < 15*time.Second || w.IdleLimit > 5*time.Minute || w.Runtime == nil || w.Auth == nil {
		return errors.New("recording worker requires a stable identity, bounded capacity and idle limit, authorization, and an isolated runtime")
	}
	if runtime, ok := w.Runtime.(interface{ Preflight(context.Context) error }); ok {
		if err := runtime.Preflight(ctx); err != nil {
			return err
		}
	}
	if err := w.Origins.Validate(); err != nil {
		return err
	}
	row, err := w.Store.Get(ctx, PoolBucket, w.ID)
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return err
	}
	if err == nil {
		pool, err := storage.Decode[Pool](row)
		if err != nil {
			return err
		}
		if pool.Instance != w.Instance && w.now().Before(pool.Until) {
			return errors.New("this recording worker identity is already leased")
		}
	}
	change, _ := storage.Put(PoolBucket, w.ID, "", row.Revision, Pool{Instance: w.Instance, Until: w.now().Add(OwnerLifetime)})
	if err := w.Store.Commit(ctx, change); err != nil {
		return err
	}
	w.drivers = map[string]recording.InteractiveDriver{}
	rows, err := w.leases(ctx)
	if err != nil {
		return err
	}
	for _, row := range rows {
		lease, err := storage.Decode[Lease](row)
		if err != nil {
			return err
		}
		if !lease.CleanupPending {
			continue
		}
		lease.Snapshot.State, lease.Snapshot.Error = "failed", "The recording worker restarted. Previously captured steps remain available. Start a new recording to continue."
		if err := w.cleanup(ctx, &lease); err != nil {
			_, _ = w.save(context.WithoutCancel(ctx), row, lease)
			return err
		}
		if _, err := w.save(ctx, row, lease); err != nil {
			return err
		}
	}
	return nil
}

func (w *Worker) Ready(ctx context.Context) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.fence(ctx) == nil
}

func (w *Worker) authorized(ctx context.Context, b Binding) error {
	if !domain.ValidID(b.ID) || !domain.ValidID(b.Project) || !domain.ValidID(b.Suite) || b.Actor == "" || !recording.ValidURL(b.URL) {
		return authorization.ErrForbidden
	}
	row, err := w.Store.Get(ctx, recording.Bucket(), b.ID)
	if err != nil {
		return err
	}
	session, err := storage.Decode[recording.Session](row)
	if err != nil {
		return err
	}
	if Bind(session) != b || session.Mode != "embedded" {
		return authorization.ErrForbidden
	}
	for _, permission := range []string{"sources.read", "catalog.read", "sources.manage"} {
		if err := w.Auth.Check(ctx, b.Actor, b.Project, "suite/"+b.Suite, permission); err != nil {
			return err
		}
	}
	suiteRow, err := w.Store.Get(ctx, projects.Bucket("suites", b.Project), b.Suite)
	if err != nil {
		return err
	}
	suite, err := storage.Decode[projects.Suite](suiteRow)
	if err != nil {
		return err
	}
	if suite.Archived {
		return errors.New("the recording suite is archived")
	}
	return nil
}

func (w *Worker) Call(ctx context.Context, operation string, request Request) (Response, error) {
	ctx, cancel := context.WithTimeout(ctx, 35*time.Second)
	defer cancel()
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.fence(ctx); err != nil {
		return Response{}, err
	}
	b := request.Binding
	if operation == "start" {
		return w.start(ctx, b)
	}
	row, err := w.Store.Get(ctx, LeaseBucket, b.ID)
	if err != nil {
		return Response{}, err
	}
	lease, err := storage.Decode[Lease](row)
	if err != nil {
		return Response{}, err
	}
	if lease.Worker != w.ID || lease.Binding != b {
		return Response{}, authorization.ErrForbidden
	}
	if operation == "stop" {
		if lease.CleanupPending {
			lease.Closing = true
			row, err = w.save(ctx, row, lease)
			if err != nil {
				return Response{}, err
			}
			if driver := w.drivers[b.ID]; driver != nil {
				if snapshot, err := driver.Stop(ctx, b.ID); err == nil {
					lease.Snapshot = snapshot
				}
			}
			if lease.Snapshot.State == "recording" {
				lease.Snapshot.State = "ready"
			}
			cleanupErr := w.cleanup(ctx, &lease)
			_, saveErr := w.save(ctx, row, lease)
			if err := errors.Join(cleanupErr, saveErr); err != nil {
				return Response{}, err
			}
		}
		return Response{Snapshot: lease.Snapshot}, nil
	}
	if err := w.authorized(ctx, b); err != nil {
		return Response{}, err
	}
	if operation == "read" {
		return Response{Snapshot: lease.Snapshot}, nil
	}
	if !w.now().Before(b.Deadline) || lease.Closing || lease.Snapshot.State != "recording" || !lease.CleanupPending || w.drivers[b.ID] == nil {
		return Response{}, errors.New("the recording browser is closed; review its retained steps")
	}
	if operation != "view" && operation != "input" {
		return Response{}, errors.New("unsupported recording worker operation")
	}
	return w.interact(ctx, row, lease, operation, request.Action)
}

func (w *Worker) target(b Binding) (browserpolicy.Target, error) {
	u, _ := url.Parse(b.URL)
	origin, err := browserpolicy.Origin(u.Scheme + "://" + u.Host)
	if err != nil {
		return browserpolicy.Target{}, err
	}
	target := browserpolicy.Target{Origin: origin, AllowPrivate: w.AllowPrivate, AdditionalOrigins: w.Origins.additional(origin)}
	return target, browserpolicy.ValidateTarget(target)
}
