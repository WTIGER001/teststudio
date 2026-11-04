package hostedexecution

import (
	"context"
	"errors"
	"time"

	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/workerexchange"
)

func bounded(ctx context.Context, call func(context.Context) error) error {
	step, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return call(step)
}

func (e *Executor) state(ctx context.Context, ref execution.Reference) (state workerexchange.State, err error) {
	err = bounded(ctx, func(ctx context.Context) error { state, err = e.Exchange.State(ctx, ref); return err })
	return
}

func terminal(status Status) bool {
	return status.Phase == "succeeded" || status.Phase == "failed" || status.Phase == "stopped"
}

func (e *Executor) observe(ctx context.Context, ref execution.Reference, image string, deadline time.Time) (status Status, err error) {
	err = bounded(ctx, func(ctx context.Context) error { status, err = e.Provider.Observe(ctx, ref); return err })
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return status, ErrNotFound
		}
		return status, ErrUnavailable
	}
	if status.Reference != ref || status.Image != image || status.Deadline.IsZero() || !deadline.IsZero() && !status.Deadline.Equal(deadline) {
		return status, ErrOwnership
	}
	if status.ExitCode != nil && (*status.ExitCode < -1 || *status.ExitCode > 255) {
		return status, ErrOwnership
	}
	switch status.Phase {
	case "pending", "running":
		if status.ExitCode != nil {
			return status, ErrOwnership
		}
	case "succeeded":
		if status.ExitCode == nil || *status.ExitCode != 0 {
			return status, ErrOwnership
		}
	case "failed":
		if status.ExitCode != nil && *status.ExitCode == 0 {
			return status, ErrOwnership
		}
	case "stopped":
	default:
		return status, ErrOwnership
	}
	return status, nil
}

func (e *Executor) pause(ctx context.Context) error {
	delay := e.PollInterval
	if delay == 0 {
		delay = 250 * time.Millisecond
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (e *Executor) wait(ctx context.Context, ref execution.Reference, input execution.CodedInput, deadline time.Time) (Status, error) {
	forwarded := false
	for {
		status, err := e.observe(ctx, ref, input.Artifact.Image, deadline)
		if err != nil {
			return status, err
		}
		state, err := e.state(ctx, ref)
		if err != nil {
			return Status{}, ErrUnavailable
		}
		if state.Revoked {
			return Status{}, ErrUnavailable
		}
		if state.StartedAt != nil && !forwarded && input.OnStart != nil {
			if err := input.OnStart(*state.StartedAt); err != nil {
				return Status{}, ErrUnavailable
			}
			forwarded = true
		}
		if terminal(status) {
			return status, nil
		}
		if err := e.pause(ctx); err != nil {
			return status, err
		}
	}
}

func (e *Executor) quiesce(ctx context.Context, ref execution.Reference, image string, deadline time.Time) (Status, error) {
	status, err := e.observe(ctx, ref, image, deadline)
	if err != nil || terminal(status) {
		return status, err
	}
	if err := bounded(ctx, func(ctx context.Context) error { return e.Provider.Stop(ctx, ref) }); err != nil && !errors.Is(err, ErrNotFound) {
		return status, ErrUnavailable
	}
	for {
		status, err = e.observe(ctx, ref, image, deadline)
		if err != nil || terminal(status) {
			return status, err
		}
		if err := e.pause(ctx); err != nil {
			return status, ErrUnavailable
		}
	}
}

func (e *Executor) revoke(ctx context.Context, ref execution.Reference) error {
	if err := bounded(ctx, func(ctx context.Context) error { return e.Exchange.Revoke(ctx, ref) }); err != nil && !errors.Is(err, storage.ErrNotFound) {
		return ErrUnavailable
	}
	return nil
}

func (e *Executor) Cleanup(ctx context.Context, ref execution.Reference) error {
	if !e.configured() || !e.validReference(ref) {
		return ErrOwnership
	}
	if err := bounded(ctx, func(ctx context.Context) error { return e.Exchange.CheckCleanup(ctx, ref) }); err != nil {
		return ErrUnavailable
	}
	if err := e.revoke(ctx, ref); err != nil {
		return err
	}
	if err := bounded(ctx, func(ctx context.Context) error { return e.Provider.Stop(ctx, ref) }); err != nil && !errors.Is(err, ErrNotFound) {
		return ErrUnavailable
	}
	if err := bounded(ctx, func(ctx context.Context) error { return e.Provider.Remove(ctx, ref) }); err != nil && !errors.Is(err, ErrNotFound) {
		return ErrUnavailable
	}
	if err := bounded(ctx, func(ctx context.Context) error { return e.Exchange.Cleanup(ctx, ref) }); err != nil {
		return ErrUnavailable
	}
	return nil
}

func (e *Executor) Inventory(ctx context.Context, installation string) (refs []execution.Reference, err error) {
	if !e.configured() || !domain.ValidID(installation) {
		return nil, ErrConfiguration
	}
	err = bounded(ctx, func(ctx context.Context) error {
		refs, err = e.Provider.Inventory(ctx, installation, e.Profile())
		return err
	})
	if err != nil {
		return nil, ErrUnavailable
	}
	seen := map[string]bool{}
	for _, ref := range refs {
		if !e.validReference(ref) || ref.Installation != installation || seen[ref.ID] {
			return nil, ErrOwnership
		}
		seen[ref.ID] = true
	}
	return refs, nil
}
