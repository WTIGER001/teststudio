package runs

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"slices"
	"time"

	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/capture"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/evidence"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/storage"
)

var ErrLeaseLost = errors.New("execution lease is no longer owned by this worker")

type HTTPExecutor interface {
	Execute(context.Context, string, catalog.Test, projects.Environment) execution.Result
}

type Service struct {
	Completion        CompletionHook
	Store             *storage.Store
	Auth              *authorization.Service
	Catalog           *catalog.Service
	Evidence          *evidence.Service
	Browser           execution.CodedExecutor
	Playwright        execution.CodedExecutor
	Go                execution.CodedExecutor
	GoAPI             execution.CodedExecutor
	RecoveryDriver    execution.ManagedExecutor
	RecoveryDrivers   []execution.ManagedExecutor
	RecoveryInterval  time.Duration
	HTTP              HTTPExecutor
	Location          string
	Concurrency       int
	PollInterval      time.Duration
	HeartbeatInterval time.Duration
}

type CompletionHook interface {
	Prepare(context.Context, Run) ([]storage.Change, error)
	Complete(context.Context, Run) ([]storage.Change, error)
}

type receipt struct {
	RunID       string `json:"runId"`
	Fingerprint string `json:"fingerprint"`
}

func fingerprint(value any) string {
	data, _ := json.Marshal(value)
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func (s *Service) Get(ctx context.Context, project, id string) (Run, storage.Record, error) {
	r, err := s.Store.Get(ctx, "runs", id)
	if err != nil {
		return Run{}, r, err
	}
	v, err := storage.Decode[Run](r)
	if err == nil && v.Project != project {
		err = storage.ErrNotFound
	}
	return NormalizeTimeouts(v), r, err
}

func (s *Service) Create(ctx context.Context, user, project string, req Request, key, trigger string) (Run, error) {
	return s.create(ctx, user, project, req, key, trigger, nil, nil, 0)
}

func (s *Service) CreateScheduled(ctx context.Context, user, project string, req Request, key string, schedule ScheduleTrigger, commit func(Run) ([]storage.Change, error)) (Run, error) {
	if commit == nil {
		return Run{}, errors.New("scheduled runs require an atomic firing checkpoint")
	}
	return s.create(ctx, user, project, req, key, "schedule", &schedule, commit, 0)
}

func (s *Service) create(ctx context.Context, user, project string, req Request, key, trigger string, schedule *ScheduleTrigger, checkpoint func(Run) ([]storage.Change, error), retry int) (Run, error) {
	if err := s.Auth.Check(ctx, user, project, "", "runs.create"); err != nil {
		return Run{}, err
	}
	if !domain.ValidID(project) || !domain.ValidID(req.Suite) || len(key) == 0 || len(key) > 128 {
		return Run{}, errors.New("project, suite, and an idempotency key are required")
	}
	if req.TimeoutSeconds == 0 {
		req.TimeoutSeconds = 1800
	}
	if req.Location == "" {
		req.Location = s.Location
	}
	if req.TimeoutSeconds < 1 || req.TimeoutSeconds > 3600 || req.Retries < 0 || req.Retries > 2 {
		return Run{}, errors.New("run duration or retry count exceeds its limits")
	}
	requestID := fingerprint([]string{user, project, key})
	requestHash := fingerprint(req)
	if r, err := s.Store.Get(ctx, "run-requests", requestID); err == nil {
		old, err := storage.Decode[receipt](r)
		if err != nil {
			return Run{}, err
		}
		if old.Fingerprint != requestHash {
			return Run{}, errors.New("idempotency key was reused for a different run request")
		}
		v, _, err := s.Get(ctx, project, old.RunID)
		return v, err
	} else if !errors.Is(err, storage.ErrNotFound) {
		return Run{}, err
	}
	v, admission, err := s.prepare(ctx, user, project, req)
	if err != nil {
		return Run{}, err
	}
	v.Trigger = trigger
	v.Schedule = schedule
	var completionChanges []storage.Change
	if s.Completion != nil {
		completionChanges, err = s.Completion.Prepare(ctx, v)
		if err != nil {
			return v, err
		}
	}
	run, err := storage.Put("runs", v.ID, project, 0, v)
	if err != nil {
		return v, err
	}
	queued, err := storage.Put("run-queue", v.ID, project, 0, v.ID)
	if err != nil {
		return v, err
	}
	idempotency, err := storage.Put("run-requests", requestID, project, 0, receipt{RunID: v.ID, Fingerprint: requestHash})
	if err != nil {
		return v, err
	}
	audit, err := domain.Audit(user, project, "run.create", v.ID, nil, v.Request)
	if err != nil {
		return v, err
	}
	changes := []storage.Change{run, queued, idempotency, audit}
	for _, row := range admission {
		changes = append(changes, storage.Change{Record: row, Assert: true})
	}
	changes = append(changes, completionChanges...)
	if checkpoint != nil {
		extra, err := checkpoint(v)
		if err != nil {
			return v, err
		}
		changes = append(changes, extra...)
	}
	err = s.Store.CommitAll(ctx, changes...)
	if errors.Is(err, storage.ErrConflict) {
		if checkpoint != nil || retry >= 7 {
			return v, err
		}
		return s.create(ctx, user, project, req, key, trigger, nil, nil, retry+1)
	}
	return v, err
}

func (s *Service) prepare(ctx context.Context, user, project string, req Request) (Run, []storage.Record, error) {
	projectRow, err := s.Store.Get(ctx, "projects", project)
	if err != nil {
		return Run{}, nil, err
	}
	p, err := storage.Decode[projects.Project](projectRow)
	if err != nil {
		return Run{}, nil, err
	}
	if p.Archived {
		return Run{}, nil, errors.New("project is archived")
	}
	suiteRow, err := s.Store.Get(ctx, projects.Bucket("suites", project), req.Suite)
	if err != nil {
		return Run{}, nil, err
	}
	suite, err := storage.Decode[projects.Suite](suiteRow)
	if err != nil {
		return Run{}, nil, err
	}
	if suite.Archived {
		return Run{}, nil, errors.New("suite is archived")
	}
	admission := []storage.Record{projectRow, suiteRow}
	var saved *catalog.SavedSelectionSnapshot
	var resolved []string
	if req.SelectionPreview != "" {
		if len(req.IDs) != 0 {
			return Run{}, nil, errors.New("saved selection previews cannot combine with explicit test identifiers")
		}
		preview, guards, err := s.Catalog.SelectionFromPreview(ctx, user, project, req.Suite, req.SelectionPreview)
		if err != nil {
			return Run{}, nil, err
		}
		if req.Release != "" && req.Release != preview.Release {
			return Run{}, nil, errors.New("run release differs from the reviewed saved selection")
		}
		req.Release, req.IDs = preview.Release, slices.Clone(preview.Roots)
		saved, resolved = &preview.Snapshot, preview.Resolved
		admission = append(admission, guards...)
	}
	releaseID := req.Release
	if releaseID == "" {
		releaseID = suite.ActiveRelease
	}
	if releaseID == "" {
		return Run{}, nil, errors.New("suite has no published active release")
	}
	releaseRow, err := s.Store.Get(ctx, catalog.Bucket("releases", project, req.Suite), releaseID)
	if err != nil {
		return Run{}, nil, err
	}
	release, err := storage.Decode[catalog.Release](releaseRow)
	if err != nil {
		return Run{}, nil, err
	}
	admission = append(admission, releaseRow)
	if req.Location != s.Location {
		return Run{}, nil, errors.New("execution location is not configured on this installation")
	}
	var environment *projects.Environment
	protected := false
	if req.Environment != "" {
		r, err := s.Store.Get(ctx, projects.Bucket("environments", project), req.Environment)
		if err != nil {
			return Run{}, nil, err
		}
		e, err := storage.Decode[projects.Environment](r)
		if err != nil {
			return Run{}, nil, err
		}
		if e.Disabled {
			return Run{}, nil, errors.New("environment is disabled")
		}
		if len(suite.Applications) > 0 && !slices.Contains(suite.Applications, e.Application) {
			return Run{}, nil, errors.New("environment application is outside this suite")
		}
		if len(e.Locations) > 0 && !slices.Contains(e.Locations, req.Location) {
			return Run{}, nil, errors.New("execution location cannot reach this environment")
		}
		environment = &e
		protected = e.Protected
		admission = append(admission, r)
	} else if suite.RequiresTarget {
		return Run{}, nil, errors.New("suite requires a target environment")
	}
	if protected {
		if err := s.Auth.Check(ctx, user, project, "environment/"+req.Environment, "runs.execute-protected"); err != nil {
			return Run{}, nil, err
		}
	}
	eligible := []catalog.Test{}
	for _, test := range release.Draft.Tests {
		if environment != nil && test.Application != "" && test.Application != environment.Application {
			continue
		}
		eligible = append(eligible, test)
	}
	tests, err := catalog.ResolveSelection(eligible, req.IDs, protected)
	if err != nil {
		return Run{}, nil, err
	}
	if saved != nil {
		actual := make([]string, 0, len(tests))
		for _, test := range tests {
			actual = append(actual, test.ID)
		}
		slices.Sort(actual)
		if !slices.Equal(actual, resolved) {
			return Run{}, nil, errors.New("run tests differ from the reviewed saved selection")
		}
	}
	guards, err := s.Catalog.ExecutionGuards(ctx, project, req.Suite, tests, protected)
	if err != nil {
		return Run{}, nil, err
	}
	admission = append(admission, guards...)
	policy := capture.Normalize(release.Draft.Capture)
	limits := capture.ResolveLimits(p.CaptureLimits)
	if req.Capture != nil {
		if err := s.Auth.Check(ctx, user, project, "", "evidence.override-capture"); err != nil {
			return Run{}, nil, err
		}
		policy = capture.Normalize(*req.Capture)
	}
	if err := capture.Validate(policy, limits); err != nil {
		return Run{}, nil, err
	}
	testPolicies := map[string]capture.Policy{}
	for _, test := range tests {
		if test.Runner == "go" && release.Draft.Artifact.Browser != nil && s.GoAPI == nil {
			return Run{}, nil, errors.New("environment-backed Go API execution is disabled; enable the trusted gateway profile")
		}
		effective := capture.Normalize(policy)
		if test.Capture != nil && req.Capture == nil {
			effective = capture.Normalize(*test.Capture)
		}
		if err := capture.Validate(effective, limits); err != nil {
			return Run{}, nil, err
		}
		testPolicies[test.ID] = effective
	}
	for _, test := range tests {
		if test.Runner == "playwright" && release.Draft.Artifact.Browser != nil && s.Browser == nil {
			return Run{}, nil, errors.New("environment-backed browser execution is disabled; the installation needs its trusted browser gateway profile")
		}
		if driver := s.codedDriver(test.Runner, release.Draft.Artifact); driver != nil {
			if req.Location != "local" {
				return Run{}, nil, errors.New("the configured coded profile runs only at the local execution location")
			}
			if release.Draft.Artifact.Browser != nil {
				if _, err := browserTarget(environment, release.Draft.Artifact, test); err != nil {
					return Run{}, nil, err
				}
			} else if environment != nil || len(test.RequiredBindings) > 0 {
				return Run{}, nil, errors.New("the isolated local coded profile requires a targetless suite without external bindings")
			}
			if err := driver.Admit(ctx, test, release.Draft.Artifact); err != nil {
				return Run{}, nil, err
			}
			continue
		}
		if test.Runner != "http" || s.HTTP == nil {
			return Run{}, nil, fmt.Errorf("no compatible worker is configured for runner %s", test.Runner)
		}
		if environment == nil || test.Check == nil {
			return Run{}, nil, errors.New("HTTP checks require a registered environment")
		}
		address := environment.Endpoints[test.Check.Endpoint]
		u, err := url.Parse(address)
		if err != nil || u.Hostname() == "" {
			return Run{}, nil, fmt.Errorf("required endpoint %s is missing", test.Check.Endpoint)
		}
		ip := net.ParseIP(u.Hostname())
		if req.Location != "local" && (u.Hostname() == "localhost" || (ip != nil && ip.IsLoopback())) {
			return Run{}, nil, errors.New("remote execution cannot reach a developer localhost target")
		}
	}
	now := time.Now().UTC()
	v := Run{ID: domain.NewID(), Project: project, Request: req, Release: release, Environment: environment, Tests: tests, Capture: release.Draft.Capture, Actor: user, State: "requested", Outcome: "unknown", Phase: "queued", Counts: Counts{Total: len(tests)}, Results: []execution.Result{}, CreatedAt: now, UpdatedAt: now, Deadline: now.Add(time.Duration(req.TimeoutSeconds) * time.Second), EvidenceState: "pending", EvidenceErrors: []string{}}
	v.SavedSelection = saved
	v.Capture = policy
	v.CaptureLimits = limits
	v.TestCapture = testPolicies
	if err := s.predictTiming(ctx, &v); err != nil {
		return Run{}, nil, err
	}
	return v, admission, nil
}
