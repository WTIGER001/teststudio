package runs

import (
	"context"
	"errors"
	"fmt"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/timing"
	"slices"
	"time"
)

func timingContext(run Run, test catalog.Test) timing.Context {
	bindings := slices.Clone(test.RequiredBindings)
	slices.Sort(bindings)
	definition := struct {
		Application, Runner, Type, Source, Selector string
		Bindings                                    []string
		Check                                       *catalog.HTTPCheck
	}{test.Application, test.Runner, test.Type, test.Source, test.Selector, bindings, test.Check}
	target := struct{ Endpoints, Credentials map[string]string }{}
	if run.Environment != nil {
		target.Endpoints = run.Environment.Endpoints
		target.Credentials = run.Environment.Credentials
	}
	return timing.Context{Project: run.Project, Suite: run.Request.Suite, Test: test.ID, Environment: run.Request.Environment, Location: run.Request.Location, Runner: test.Runner, Profile: "native-http-v1", Definition: fingerprint(definition), Target: fingerprint(target)}
}
func (s *Service) predictTiming(ctx context.Context, run *Run) error {
	service := timing.Service{Store: s.Store}
	predictions := map[string]timing.Prediction{}
	for _, test := range run.Tests {
		policy := timing.Normalize(run.Release.Draft.Timing)
		if test.Timing != nil {
			policy = timing.Normalize(*test.Timing)
		}
		p, err := service.Prediction(ctx, timingContext(*run, test), policy, test.TimeoutSeconds*1000, run.CreatedAt)
		if err != nil {
			return err
		}
		predictions[test.ID] = p
	}
	run.Timing = timing.Summarize(predictions, run.Request.Retries, run.Request.TimeoutSeconds)
	return nil
}
func (s *Service) Estimate(ctx context.Context, user, project string, req Request) (timing.Estimate, error) {
	if err := s.Auth.Check(ctx, user, project, "", "runs.create"); err != nil {
		return timing.Estimate{}, err
	}
	if req.TimeoutSeconds == 0 {
		req.TimeoutSeconds = 1800
	}
	if req.Location == "" {
		req.Location = s.Location
	}
	if req.TimeoutSeconds < 1 || req.TimeoutSeconds > 3600 || req.Retries < 0 || req.Retries > 2 {
		return timing.Estimate{}, errors.New("run duration or retry count exceeds its limits")
	}
	if !domain.ValidID(project) || !domain.ValidID(req.Suite) {
		return timing.Estimate{}, errors.New("valid project and suite identifiers are required")
	}
	v, _, err := s.prepare(ctx, user, project, req)
	return v.Timing, err
}
func assessResult(run Run, result execution.Result) execution.Result {
	result.ObservedState = result.State
	if p, ok := run.Timing.Tests[result.TestID]; ok {
		a := timing.Assess(p, result.DurationMs, result.State, result.FailureKind)
		result.Timing = &a
		if result.State == "passed" && a.Status == "slow" && p.Policy.Mode == "fail" {
			result.State = "failed"
			result.FailureKind = "duration-regression"
			result.Error = fmt.Sprintf("Duration %.1f ms exceeds %.1f ms threshold (%.2fx baseline, %d samples).", result.DurationMs, *p.ThresholdMs, *a.Ratio, p.Samples)
		}
	}
	return result
}
func (s *Service) finalResult(ctx context.Context, run Run, worker string, result execution.Result) error {
	for range 20 {
		v, r, err := s.Get(ctx, run.Project, run.ID)
		if err != nil {
			return err
		}
		if err := owned(&v, worker); err != nil {
			return err
		}
		for _, old := range v.Results {
			if old.TestID == result.TestID {
				return errors.New("test result already exists")
			}
		}
		v.Results = append(v.Results, result)
		v.Counts = Count(len(v.Tests), v.Results)
		v.UpdatedAt = time.Now().UTC()
		c, err := storage.Put(r.Bucket, r.ID, v.Project, r.Revision, v)
		if err != nil {
			return err
		}
		changes := []storage.Change{c}
		if result.State == "passed" && result.Attempt == 1 && result.Timing != nil && !result.Timing.Censored && result.Timing.Status != "slow" && (result.Timing.Prediction.ThresholdMs == nil || result.DurationMs < *result.Timing.Prediction.ThresholdMs) {
			service := timing.Service{Store: s.Store}
			samples, err := service.RecordChanges(ctx, result.Timing.Prediction, timing.Sample{Run: v.ID, DurationMs: result.DurationMs, At: result.FinishedAt})
			if err != nil {
				return err
			}
			changes = append(changes, samples...)
		}
		err = s.Store.Commit(ctx, changes...)
		if !errors.Is(err, storage.ErrConflict) {
			return err
		}
	}
	return storage.ErrConflict
}
