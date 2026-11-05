package azuresandbox

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/internal/capture"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/workerexchange"
	"github.com/wtiger001/teststudio/internal/workerprotocol"
	"github.com/wtiger001/teststudio/internal/workerstate"
)

type instanceControl struct {
	mu            sync.Mutex
	items         map[string]Snapshot
	createCalls   int
	stopCalls     int
	deleteCalls   int
	lostCreate    bool
	emptyCreate   bool
	rejectCreate  bool
	keepDeleted   bool
	inventoryErr  error
	createEntered chan struct{}
	releaseCreate chan struct{}
}

func (m *instanceControl) Create(ctx context.Context, plan *Plan) (Snapshot, error) {
	m.mu.Lock()
	m.createCalls++
	entered, release := m.createEntered, m.releaseCreate
	m.mu.Unlock()
	if entered != nil {
		close(entered)
		select {
		case <-release:
		case <-ctx.Done():
			return Snapshot{}, hostedexecution.ErrUnavailable
		}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.rejectCreate {
		return Snapshot{}, hostedexecution.ErrRejected
	}
	if m.emptyCreate {
		return Snapshot{}, hostedexecution.ErrUnavailable
	}
	body, err := plan.launch(time.Now())
	if err != nil {
		return Snapshot{}, err
	}
	body["id"], body["state"] = "native-sandbox", "Running"
	data, _ := json.Marshal(body)
	actual, err := snapshot(data)
	if err != nil {
		return actual, err
	}
	m.items[actual.ID] = actual
	if m.lostCreate {
		return Snapshot{}, hostedexecution.ErrUnavailable
	}
	return actual, nil
}

func (m *instanceControl) Read(_ context.Context, id string) (Snapshot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	item, found := m.items[id]
	if !found {
		return Snapshot{}, hostedexecution.ErrNotFound
	}
	return item, nil
}

func (m *instanceControl) Inventory(context.Context) ([]Snapshot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.inventoryErr != nil {
		return nil, m.inventoryErr
	}
	items := []Snapshot{}
	for _, item := range m.items {
		items = append(items, item)
	}
	return items, nil
}

func (m *instanceControl) Stop(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stopCalls++
	item := m.items[id]
	item.State = "Stopping"
	m.items[id] = item
	return nil
}

func (m *instanceControl) Delete(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.deleteCalls++
	if !m.keepDeleted {
		delete(m.items, id)
	}
	return nil
}

type lifecycleControls struct {
	mu           sync.Mutex
	spec         string
	ready        bool
	ensureCalls  int
	removeCalls  int
	ensureErr    error
	verifyErr    error
	admissionErr error
	instanceErr  error
	removeLost   bool
}

func (m *lifecycleControls) Ensure(_ context.Context, spec DeadlineSpec) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ensureCalls++
	m.spec = hash(spec)
	if m.ensureErr != nil {
		return m.ensureErr
	}
	m.ready = true
	return nil
}

func (m *lifecycleControls) Verify(_ context.Context, spec DeadlineSpec) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.spec != hash(spec) {
		return hostedexecution.ErrOwnership
	}
	if m.verifyErr != nil {
		return m.verifyErr
	}
	if !m.ready {
		return hostedexecution.ErrUnavailable
	}
	return nil
}

func (m *lifecycleControls) Remove(_ context.Context, spec DeadlineSpec) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.spec != hash(spec) {
		return hostedexecution.ErrOwnership
	}
	m.removeCalls++
	m.ready = false
	if m.removeLost {
		m.removeLost = false
		return hostedexecution.ErrUnavailable
	}
	return nil
}

func (m *lifecycleControls) BeforeCreate(context.Context, Config) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.ready {
		return hostedexecution.ErrUnavailable
	}
	return m.admissionErr
}

func (m *lifecycleControls) VerifyInstance(context.Context, Config, Snapshot) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.instanceErr
}

type controllerFixture struct {
	t          *testing.T
	path       string
	store      *storage.Store
	exchange   *workerexchange.Service
	controller *Controller
	plan       *Plan
	api        *instanceControl
	controls   *lifecycleControls
	denied     atomic.Bool
	checkpoint atomic.Bool
}

func newControllerFixture(t *testing.T, runner, operation string) *controllerFixture {
	t.Helper()
	base := fixturePlan(t, runner, operation)
	f := &controllerFixture{t: t, path: filepath.Join(t.TempDir(), "sandbox-dispatch.sqlite"), api: &instanceControl{items: map[string]Snapshot{}}, controls: &lifecycleControls{}}
	var err error
	f.store, err = storage.Open(t.Context(), f.path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { f.store.Close() })
	f.exchange = &workerexchange.Service{Store: f.store}
	f.exchange.Authorize = func(context.Context, execution.Reference) (workerstate.Access, error) {
		if f.denied.Load() {
			return workerstate.Access{}, workerexchange.ErrDenied
		}
		return workerstate.Access{Deadline: base.proof.Deadline}, nil
	}
	f.exchange.AuthorizeDiscoveryCleanup = func(context.Context, execution.Reference) error {
		if f.checkpoint.Load() {
			return nil
		}
		return workerexchange.ErrDenied
	}
	ref := base.proof.Reference
	var ticket workerexchange.Ticket
	if operation == "discover" {
		ticket, err = f.exchange.PrepareDiscovery(t.Context(), ref, workerprotocol.DiscoveryInput{Project: ref.Project, Suite: "parcel-inspection", Runner: runner, Image: base.proof.Image, SourceDigest: "sha256:" + strings.Repeat("c", 64)}, base.proof.Deadline)
	} else {
		selector := "./parcel::TestPackage"
		if runner == "playwright" {
			selector = "@PARCEL-1"
		}
		input := execution.CodedInput{RunID: ref.Run, Test: catalog.Test{ID: ref.Test, Runner: runner, Type: "integration", Selector: selector}, Artifact: catalog.Artifact{Image: base.proof.Image, SourceProject: ref.Project, Discovered: map[string]string{ref.Test: selector}}, Capture: capture.DefaultPolicy(), TimeoutMs: 1000, AwaitDispatch: true}
		ticket, err = f.exchange.Prepare(t.Context(), ref, input, base.proof.Deadline)
		if err == nil {
			_, changes, beginErr := (workerstate.Ledger{Store: f.store}).Begin(ref)
			if beginErr != nil {
				t.Fatal(beginErr)
			}
			err = f.store.Commit(t.Context(), changes...)
		}
	}
	if err != nil {
		t.Fatal(err)
	}
	proof := base.Proof()
	proof.TicketHash, proof.EnvironmentHash = hash(ticket.Token), hash(environment(ref.ID, ticket.Token))
	f.plan, err = newPlan(base.config, proof)
	if err != nil {
		t.Fatal(err)
	}
	f.plan.token = ticket.Token
	f.openController()
	return f
}

func (f *controllerFixture) openController() {
	f.t.Helper()
	var err error
	f.controller, err = NewController(ControllerOptions{Config: f.plan.config, Instances: f.api, Deadlines: f.controls, Admission: f.controls, Exchange: f.exchange})
	if err != nil {
		f.t.Fatal(err)
	}
}

func (f *controllerFixture) reopen() {
	f.t.Helper()
	if err := f.store.Close(); err != nil {
		f.t.Fatal(err)
	}
	var err error
	f.store, err = storage.Open(f.t.Context(), f.path)
	if err != nil {
		f.t.Fatal(err)
	}
	f.exchange.Store = f.store
	f.openController()
}

func (f *controllerFixture) retainCheckpoint() {
	f.t.Helper()
	ref := f.plan.proof.Reference
	if err := f.exchange.Revoke(f.t.Context(), ref); err != nil {
		f.t.Fatal(err)
	}
	f.checkpoint.Store(true)
	if f.plan.proof.Operation == "discover" {
		return
	}
	ledger := workerstate.Ledger{Store: f.store}
	value, row, err := ledger.Get(f.t.Context(), ref.Run, ref.ID)
	if err != nil {
		f.t.Fatal(err)
	}
	change, err := ledger.Checkpoint(row, value, nil, false)
	if err != nil || f.store.Commit(f.t.Context(), change) != nil {
		f.t.Fatal("result checkpoint is unavailable", err)
	}
}

func (f *controllerFixture) instanceState(state string) {
	f.api.mu.Lock()
	defer f.api.mu.Unlock()
	item := f.api.items["native-sandbox"]
	item.State = state
	f.api.items[item.ID] = item
}
