package azureworker

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v4"
	"github.com/wtiger001/teststudio/internal/capture"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/workerexchange"
	"github.com/wtiger001/teststudio/internal/workerprotocol"
	"github.com/wtiger001/teststudio/internal/workerstate"
)

func clone[T any](v T) T {
	data, _ := json.Marshal(v)
	var result T
	json.Unmarshal(data, &result)
	return result
}

type jobFixture struct {
	mu                                     sync.Mutex
	definition                             *a.Job
	items                                  []a.JobExecution
	creates, starts, stops, deletes, reads int
	mode                                   string
	hidden                                 bool
	async                                  bool
}

func (f *jobFixture) Get(context.Context, string) (a.Job, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reads++
	if f.definition == nil {
		return a.Job{}, hostedexecution.ErrNotFound
	}
	return clone(*f.definition), nil
}
func (f *jobFixture) Create(_ context.Context, name string, d a.Job) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.creates++
	d = clone(d)
	d.Name, d.Type = to.Ptr(name), to.Ptr("Microsoft.App/jobs")
	d.Properties.ProvisioningState = to.Ptr(a.JobProvisioningStateSucceeded)
	if f.async {
		d.Properties.ProvisioningState = to.Ptr(a.JobProvisioningStateInProgress)
	}
	if f.mode == "create-rejected" {
		return hostedexecution.ErrRejected
	}
	f.definition = &d
	if f.mode == "lost-create" {
		return errors.New("private-provider-canary")
	}
	return nil
}
func (f *jobFixture) Start(_ context.Context, name string, options *a.JobsClientBeginStartOptions) (*a.JobExecutionBase, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.starts++
	if f.mode == "start-rejected" {
		return nil, hostedexecution.ErrRejected
	}
	executionName := name + "-xyz123"
	receipt := a.JobExecutionBase{ID: to.Ptr(value(f.definition.ID) + "/executions/" + executionName), Name: to.Ptr(executionName)}
	f.items = append(f.items, a.JobExecution{ID: receipt.ID, Name: receipt.Name, Properties: &a.JobExecutionProperties{Template: clone(options.Template), Status: to.Ptr(a.JobExecutionRunningStateRunning), StartTime: to.Ptr(time.Now().UTC())}})
	if f.mode == "lost-start" || f.mode == "hidden-start" {
		return nil, errors.New("private-provider-canary")
	}
	return &receipt, nil
}
func (f *jobFixture) Executions(context.Context, string) ([]a.JobExecution, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.hidden {
		return nil, nil
	}
	return clone(f.items), nil
}
func (f *jobFixture) Stop(_ context.Context, name, execution string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.stops++
	if len(f.items) != 1 || value(f.items[0].Name) != execution || !strings.HasPrefix(execution, name+"-") {
		return hostedexecution.ErrOwnership
	}
	f.items[0].Properties.Status, f.items[0].Properties.EndTime = to.Ptr(a.JobExecutionRunningStateStopped), to.Ptr(time.Now().UTC())
	return nil
}
func (f *jobFixture) Delete(context.Context, string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.deletes++
	if f.mode != "delete-pending" {
		f.definition = nil
		f.items = nil
	}
	return nil
}

type controlFixture struct {
	mu               sync.Mutex
	job              hostedexecution.Job
	deadline         *DeadlineSpec
	ensures, removes int
	pending, blocked bool
}

func (f *controlFixture) InspectImage(_ context.Context, image string) (hostedexecution.Image, error) {
	return hostedexecution.Image{Reference: image, Runner: f.job.Runner, Project: f.job.Reference.Project, OS: "linux", Architecture: "amd64", Runtime: f.job.Runner + "-fixture-v1", Supervisor: "worker-exchange/v1", Discovery: "worker-exchange/v1"}, nil
}
func (f *controlFixture) Admit(context.Context, Config) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.blocked {
		return hostedexecution.ErrOwnership
	}
	return nil
}
func (f *controlFixture) VerifyExecution(context.Context, Config, string, a.JobExecution) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.pending {
		return hostedexecution.ErrUnavailable
	}
	if f.blocked {
		return hostedexecution.ErrOwnership
	}
	return nil
}
func (f *controlFixture) Ensure(_ context.Context, s DeadlineSpec) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.ensures++
	copy := s
	f.deadline = &copy
	return nil
}
func (f *controlFixture) Verify(_ context.Context, s DeadlineSpec) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.deadline == nil || fingerprint(*f.deadline) != fingerprint(s) {
		return hostedexecution.ErrUnavailable
	}
	return nil
}
func (f *controlFixture) Remove(_ context.Context, s DeadlineSpec) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.deadline != nil && fingerprint(*f.deadline) != fingerprint(s) {
		return hostedexecution.ErrOwnership
	}
	f.removes++
	f.deadline = nil
	return nil
}

type providerFixture struct {
	t          *testing.T
	provider   *Provider
	store      *storage.Store
	path       string
	job        hostedexecution.Job
	config     Config
	now        time.Time
	exchange   *workerexchange.Service
	jobs       *jobFixture
	control    *controlFixture
	checkpoint bool
}

func providerTest(t *testing.T, runner, operation string) *providerFixture {
	t.Helper()
	c, job, _, _ := fixture(t, runner, operation)
	f := &providerFixture{t: t, config: c, job: job, now: time.Now().UTC(), path: filepath.Join(t.TempDir(), "dispatch.db"), jobs: &jobFixture{}}
	var err error
	f.store, err = storage.Open(t.Context(), f.path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { f.store.Close() })
	deadline := f.now.Add(90 * time.Second)
	f.exchange = &workerexchange.Service{Store: f.store, Authorize: func(context.Context, execution.Reference) (workerstate.Access, error) {
		return workerstate.Access{Deadline: deadline}, nil
	}}
	if operation == "discover" {
		f.job.Ticket, err = f.exchange.PrepareDiscovery(t.Context(), job.Reference, workerprotocol.DiscoveryInput{Project: job.Reference.Project, Suite: "orders", Runner: runner, Image: job.Image, SourceDigest: "sha256:" + strings.Repeat("c", 64)}, deadline)
	} else {
		selector := "@ORDER-3"
		if runner == "go" {
			selector = "./checks::TestOrder"
		}
		input := execution.CodedInput{RunID: job.Reference.Run, Test: catalog.Test{ID: job.Reference.Test, Runner: runner, Type: "integration", Selector: selector}, Artifact: catalog.Artifact{Image: job.Image, SourceProject: job.Reference.Project, Discovered: map[string]string{job.Reference.Test: selector}}, Capture: capture.DefaultPolicy(), TimeoutMs: 1000, AwaitDispatch: true}
		f.job.Ticket, err = f.exchange.Prepare(t.Context(), job.Reference, input, deadline)
		_, changes, beginErr := (workerstate.Ledger{Store: f.store}).Begin(job.Reference)
		if beginErr != nil || f.store.Commit(t.Context(), changes...) != nil {
			t.Fatal("cannot prepare worker ledger", beginErr)
		}
	}
	if err != nil {
		t.Fatal(err)
	}
	f.control = &controlFixture{job: f.job}
	f.makeProvider()
	return f
}
func (f *providerFixture) makeProvider() {
	f.t.Helper()
	var err error
	f.provider, err = New(Options{Config: f.config, Store: f.store, Jobs: f.jobs, Network: f.control, Inspector: f.control, Deadlines: f.control, ResumeDispatch: f.exchange.ResumeDispatch, Now: func() time.Time { return f.now }, DiscoveryCleanup: func(context.Context, execution.Reference) error {
		if !f.checkpoint {
			return hostedexecution.ErrUnavailable
		}
		return nil
	}})
	if err != nil {
		f.t.Fatal(err)
	}
}
func (f *providerFixture) ready() {
	f.jobs.mu.Lock()
	defer f.jobs.mu.Unlock()
	if f.jobs.definition != nil {
		f.jobs.definition.Properties.ProvisioningState = to.Ptr(a.JobProvisioningStateSucceeded)
	}
}
func (f *providerFixture) bindDefinition() {
	f.jobs.mu.Lock()
	defer f.jobs.mu.Unlock()
	plan, err := NewPlan(f.config, f.job, f.now)
	if err != nil {
		f.t.Fatal(err)
	}
	if f.jobs.definition != nil {
		f.jobs.definition.ID = to.Ptr(plan.ResourceID())
	}
}
func (f *providerFixture) start() {
	f.t.Helper()
	if err := f.provider.Ensure(f.t.Context(), f.job); err != nil {
		f.t.Fatal(err)
	}
	f.bindDefinition()
	f.ready()
	for range 3 {
		s, err := f.provider.Observe(f.t.Context(), f.job.Reference)
		if err != nil {
			f.t.Fatal(err)
		}
		if !s.AdmissionPending {
			return
		}
	}
	f.t.Fatal("owned worker never reaches admission")
}
func (f *providerFixture) reopen() {
	f.t.Helper()
	f.store.Close()
	var err error
	f.store, err = storage.Open(f.t.Context(), f.path)
	if err != nil {
		f.t.Fatal(err)
	}
	f.exchange.Store = f.store
	f.makeProvider()
}
func (f *providerFixture) revoke(checkpoint bool) {
	f.t.Helper()
	if err := f.exchange.Revoke(f.t.Context(), f.job.Reference); err != nil {
		f.t.Fatal(err)
	}
	f.checkpoint = checkpoint
	if f.job.Operation == "" && checkpoint {
		ledger := workerstate.Ledger{Store: f.store}
		r, row, err := ledger.Get(f.t.Context(), f.job.Reference.Run, f.job.Reference.ID)
		if err != nil {
			f.t.Fatal(err)
		}
		r.ResultSaved, r.EvidenceComplete = true, true
		c, _ := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, r)
		if err := f.store.Commit(f.t.Context(), c); err != nil {
			f.t.Fatal(err)
		}
	}
}
