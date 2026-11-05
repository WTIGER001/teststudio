package ecsworker

import (
	"context"
	"errors"

	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/storage"
)

func (p *Provider) remember(ctx context.Context, ref execution.Reference, status hostedexecution.Status) error {
	if !terminal(status) {
		return nil
	}
	_, err := p.update(ctx, ref, false, func(value *dispatch) error {
		if value.Terminal != nil && !sameStatus(*value.Terminal, status) {
			return hostedexecution.ErrOwnership
		}
		value.Terminal = &status
		return nil
	})
	return err
}

func (p *Provider) Observe(ctx context.Context, ref execution.Reference) (result hostedexecution.Status, resultErr error) {
	defer func() { resultErr = controlError(resultErr) }()
	value, _, plan, err := p.load(ctx, ref)
	if errors.Is(err, storage.ErrNotFound) {
		return hostedexecution.Status{}, hostedexecution.ErrNotFound
	}
	if err != nil {
		return hostedexecution.Status{}, err
	}
	if value.Terminal != nil {
		return *value.Terminal, nil
	}
	if !value.LaunchAttempted || value.LaunchRejected {
		return hostedexecution.Status{}, hostedexecution.ErrNotFound
	}
	value, err = p.reconcileReceipts(ctx, value, plan)
	if err != nil {
		return hostedexecution.Status{}, err
	}
	transport, _, err := p.exchange(ctx, value)
	if err != nil {
		return hostedexecution.Status{}, err
	}
	if _, err := p.definition(ctx, plan, value.DefinitionARN, transport.Revoked); err != nil {
		return hostedexecution.Status{}, hostedexecution.ErrUnavailable
	}
	task, err := p.task(ctx, plan, value.DefinitionARN, value.TaskARN, transport.Revoked || value.Stopping)
	if err != nil {
		if errors.Is(err, hostedexecution.ErrNotFound) {
			return hostedexecution.Status{}, hostedexecution.ErrUnavailable
		}
		return hostedexecution.Status{}, err
	}
	status, err := plan.taskStatus(task, value.Stopping)
	if err != nil {
		return status, err
	}
	if terminal(status) {
		if err := p.remember(ctx, ref, status); err != nil {
			return status, err
		}
		return status, nil
	}
	if transport.Revoked {
		return status, nil
	}
	if !value.DeadlineAttempted {
		status.AdmissionPending = true
		return status, nil
	}
	deadline, err := p.getDeadline(ctx, plan)
	if err != nil {
		status.AdmissionPending = true
		return status, nil
	}
	if err := plan.VerifyDeadline(value.TaskARN, deadline); err != nil {
		return status, err
	}
	if !value.DeadlineVerified {
		if _, err := p.update(ctx, ref, false, func(value *dispatch) error { value.DeadlineVerified = true; return nil }); err != nil {
			return status, err
		}
	}
	if status.Phase != "running" {
		status.AdmissionPending = true
		return status, nil
	}
	if err := p.verifyTaskNetwork(ctx, plan, task); err != nil {
		if errors.Is(err, hostedexecution.ErrOwnership) {
			return status, err
		}
		status.AdmissionPending = true
	}
	return status, nil
}
