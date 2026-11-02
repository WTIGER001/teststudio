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
	Execute(context.Context, catalog.Test, projects.Environment) execution.Result
}

type Service struct {
	Store             *storage.Store
	Auth              *authorization.Service
	Catalog           *catalog.Service
	Evidence          *evidence.Service
	Playwright        execution.CodedExecutor
	Go                execution.CodedExecutor
	RecoveryDriver    execution.ManagedExecutor
	RecoveryDrivers   []execution.ManagedExecutor
	RecoveryInterval  time.Duration
	HTTP              HTTPExecutor
	Location          string
	Concurrency       int
	PollInterval      time.Duration
	HeartbeatInterval time.Duration
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
	v, projectRow, err := s.prepare(ctx, user, project, req)
	if err != nil {
		return Run{}, err
	}
	v.Trigger = trigger
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
	audit, err := domain.Audit(user, project, "run.create", v.ID, nil, req)
	if err != nil {
		return v, err
	}
	err = s.Store.Commit(ctx, run, queued, idempotency, audit, storage.Change{Record: projectRow, Assert: true})
	if errors.Is(err, storage.ErrConflict) {
		return s.Create(ctx, user, project, req, key, trigger)
	}
	return v, err
}

func (s *Service) prepare(ctx context.Context, user, project string, req Request) (Run, storage.Record, error) {
	projectRow, err := s.Store.Get(ctx, "projects", project)
	if err != nil {
		return Run{}, storage.Record{}, err
	}
	p, err := storage.Decode[projects.Project](projectRow)
	if err != nil {
		return Run{}, storage.Record{}, err
	}
	if p.Archived {
		return Run{}, storage.Record{}, errors.New("project is archived")
	}
	suiteRow, err := s.Store.Get(ctx, projects.Bucket("suites", project), req.Suite)
	if err != nil {
		return Run{}, storage.Record{}, err
	}
	suite, err := storage.Decode[projects.Suite](suiteRow)
	if err != nil {
		return Run{}, storage.Record{}, err
	}
	if suite.Archived {
		return Run{}, storage.Record{}, errors.New("suite is archived")
	}
	releaseID := req.Release
	if releaseID == "" {
		releaseID = suite.ActiveRelease
	}
	if releaseID == "" {
		return Run{}, storage.Record{}, errors.New("suite has no published active release")
	}
	releaseRow, err := s.Store.Get(ctx, catalog.Bucket("releases", project, req.Suite), releaseID)
	if err != nil {
		return Run{}, storage.Record{}, err
	}
	release, err := storage.Decode[catalog.Release](releaseRow)
	if err != nil {
		return Run{}, storage.Record{}, err
	}
	if req.Location != s.Location {
		return Run{}, storage.Record{}, errors.New("execution location is not configured on this installation")
	}
	var environment *projects.Environment
	protected := false
	if req.Environment != "" {
		r, err := s.Store.Get(ctx, projects.Bucket("environments", project), req.Environment)
		if err != nil {
			return Run{}, storage.Record{}, err
		}
		e, err := storage.Decode[projects.Environment](r)
		if err != nil {
			return Run{}, storage.Record{}, err
		}
		if e.Disabled {
			return Run{}, storage.Record{}, errors.New("environment is disabled")
		}
		if len(suite.Applications) > 0 && !slices.Contains(suite.Applications, e.Application) {
			return Run{}, storage.Record{}, errors.New("environment application is outside this suite")
		}
		if len(e.Locations) > 0 && !slices.Contains(e.Locations, req.Location) {
			return Run{}, storage.Record{}, errors.New("execution location cannot reach this environment")
		}
		environment = &e
		protected = e.Protected
	} else if suite.RequiresTarget {
		return Run{}, storage.Record{}, errors.New("suite requires a target environment")
	}
	if protected {
		if err := s.Auth.Check(ctx, user, project, "environment/"+req.Environment, "runs.execute-protected"); err != nil {
			return Run{}, storage.Record{}, err
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
		return Run{}, storage.Record{}, err
	}
	if err := s.Catalog.CheckCurrentState(ctx, project, req.Suite, tests, protected); err != nil {
		return Run{}, storage.Record{}, err
	}
	policy := capture.Normalize(release.Draft.Capture)
	limits := capture.ResolveLimits(p.CaptureLimits)
	if req.Capture != nil {
		if err := s.Auth.Check(ctx, user, project, "", "evidence.override-capture"); err != nil {
			return Run{}, storage.Record{}, err
		}
		policy = capture.Normalize(*req.Capture)
	}
	if err := capture.Validate(policy, limits); err != nil {
		return Run{}, storage.Record{}, err
	}
	testPolicies := map[string]capture.Policy{}
	for _, test := range tests {
		effective := capture.Normalize(policy)
		if test.Capture != nil && req.Capture == nil {
			effective = capture.Normalize(*test.Capture)
		}
		if err := capture.Validate(effective, limits); err != nil {
			return Run{}, storage.Record{}, err
		}
		testPolicies[test.ID] = effective
	}
	for _, test := range tests {
		if driver := s.codedDriver(test.Runner); driver != nil {
			if req.Location != "local" {
				return Run{}, storage.Record{}, errors.New("the configured coded profile runs only at the local execution location")
			}
			if environment != nil || len(test.RequiredBindings) > 0 {
				return Run{}, storage.Record{}, errors.New("the isolated local coded profile requires a targetless suite without external bindings")
			}
			if err := driver.Admit(ctx, test, release.Draft.Artifact); err != nil {
				return Run{}, storage.Record{}, err
			}
			continue
		}
		if test.Runner != "http" || s.HTTP == nil {
			return Run{}, storage.Record{}, fmt.Errorf("no compatible worker is configured for runner %s", test.Runner)
		}
		if environment == nil || test.Check == nil {
			return Run{}, storage.Record{}, errors.New("HTTP checks require a registered environment")
		}
		address := environment.Endpoints[test.Check.Endpoint]
		u, err := url.Parse(address)
		if err != nil || u.Hostname() == "" {
			return Run{}, storage.Record{}, fmt.Errorf("required endpoint %s is missing", test.Check.Endpoint)
		}
		ip := net.ParseIP(u.Hostname())
		if req.Location != "local" && (u.Hostname() == "localhost" || (ip != nil && ip.IsLoopback())) {
			return Run{}, storage.Record{}, errors.New("remote execution cannot reach a developer localhost target")
		}
	}
	now := time.Now().UTC()
	v := Run{ID: domain.NewID(), Project: project, Request: req, Release: release, Environment: environment, Tests: tests, Capture: release.Draft.Capture, Actor: user, State: "requested", Outcome: "unknown", Phase: "queued", Counts: Counts{Total: len(tests)}, Results: []execution.Result{}, CreatedAt: now, UpdatedAt: now, Deadline: now.Add(time.Duration(req.TimeoutSeconds) * time.Second), EvidenceState: "pending", EvidenceErrors: []string{}}
	v.Capture = policy
	v.CaptureLimits = limits
	v.TestCapture = testPolicies
	if err := s.predictTiming(ctx, &v); err != nil {
		return Run{}, storage.Record{}, err
	}
	return v, projectRow, nil
}
