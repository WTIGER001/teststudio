package azureworker

import (
	"context"
	"errors"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v4"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/storage"
)

func (p *Provider) saveReceipt(ctx context.Context, plan *Plan, receipt a.JobExecutionBase) error {
	if plan.VerifyReceipt(receipt) != nil {
		return hostedexecution.ErrOwnership
	}
	_, err := p.update(ctx, plan.job.Reference, false, func(d *dispatch) error {
		if !d.StartAttempted || d.StartRejected || d.Receipt != nil && (value(d.Receipt.Name) != value(receipt.Name) || !strings.EqualFold(value(d.Receipt.ID), value(receipt.ID))) {
			return hostedexecution.ErrOwnership
		}
		d.Receipt = &receipt
		return nil
	})
	return err
}

func (p *Provider) readExecution(ctx context.Context, d dispatch, plan *Plan) (a.JobExecution, a.JobExecutionBase, error) {
	items, err := p.jobs.Executions(ctx, plan.Name())
	if err != nil || len(items) == 0 {
		return a.JobExecution{}, a.JobExecutionBase{}, hostedexecution.ErrUnavailable
	}
	if len(items) != 1 {
		return a.JobExecution{}, a.JobExecutionBase{}, hostedexecution.ErrOwnership
	}
	actual := items[0]
	receipt := d.Receipt
	if receipt == nil {
		receipt = &a.JobExecutionBase{Name: actual.Name, ID: actual.ID}
		if value(receipt.ID) == "" {
			receipt.ID = to.Ptr(plan.ResourceID() + "/executions/" + value(actual.Name))
		}
	}
	if err := plan.VerifyExecutionOwnership(*receipt, actual); err != nil {
		return actual, a.JobExecutionBase{}, err
	}
	if d.Receipt == nil {
		if err := p.saveReceipt(ctx, plan, *receipt); err != nil {
			return actual, a.JobExecutionBase{}, err
		}
	}
	return actual, *receipt, nil
}

func (p *Provider) findExecution(ctx context.Context, d dispatch, plan *Plan) (a.JobExecution, Observation, error) {
	actual, receipt, err := p.readExecution(ctx, d, plan)
	if err != nil {
		return actual, Observation{}, err
	}
	observed, err := plan.VerifyExecution(receipt, actual)
	return actual, observed, err
}

func (p *Provider) remember(ctx context.Context, ref execution.Reference, status hostedexecution.Status, beforeStart bool) error {
	_, err := p.update(ctx, ref, false, func(d *dispatch) error {
		if beforeStart && d.StartAttempted && !d.StartRejected || !beforeStart && d.Receipt == nil || d.Terminal != nil && fingerprint(*d.Terminal) != fingerprint(status) {
			return hostedexecution.ErrUnavailable
		}
		d.Terminal = &status
		return nil
	})
	return err
}

func statusFor(d dispatch, phase string) hostedexecution.Status {
	return hostedexecution.Status{Reference: d.Proof.Reference, Image: d.Proof.Image, Deadline: d.Proof.Deadline, Phase: phase, NativeCompletion: phase == "succeeded"}
}

func (p *Provider) Observe(ctx context.Context, ref execution.Reference) (status hostedexecution.Status, resultErr error) {
	defer func() { resultErr = controlError(resultErr) }()
	d, _, plan, err := p.load(ctx, ref)
	if errors.Is(err, storage.ErrNotFound) {
		return status, hostedexecution.ErrNotFound
	}
	if err != nil {
		return status, err
	}
	if d.Terminal != nil {
		return *d.Terminal, nil
	}
	if d.Removed || d.CreateRejected || d.CreationBlocked {
		return status, hostedexecution.ErrNotFound
	}
	status = statusFor(d, "pending")
	status.AdmissionPending = true
	transport, _, err := p.exchange(ctx, d)
	if err != nil {
		return status, err
	}
	if !d.CreateAttempted {
		if !transport.Revoked && !d.Stopping {
			return status, nil
		}
		status = statusFor(d, "stopped")
		return status, p.remember(ctx, ref, status, true)
	}
	definition, err := p.jobs.Get(ctx, plan.Name())
	if err != nil {
		if !transport.DispatchReady && !transport.Revoked && !d.Stopping {
			return status, nil
		}
		return status, hostedexecution.ErrUnavailable
	}
	if err := plan.VerifyRetainedDefinition(definition); err != nil {
		return status, err
	}
	if !d.DefinitionSeen {
		d, err = p.update(ctx, ref, false, func(current *dispatch) error { current.DefinitionSeen = true; return nil })
		if err != nil {
			return status, err
		}
	}
	if d.StartRejected || !d.StartAttempted && (transport.Revoked || d.Stopping) {
		status = statusFor(d, "stopped")
		if d.StartRejected {
			status.Phase = "failed"
		}
		return status, p.remember(ctx, ref, status, true)
	}
	if !d.StartAttempted {
		switch value(definition.Properties.ProvisioningState) {
		case a.JobProvisioningStateFailed, a.JobProvisioningStateCanceled:
			status = statusFor(d, "failed")
			return status, p.remember(ctx, ref, status, true)
		case a.JobProvisioningStateSucceeded:
			if err := p.prepareStart(ctx, d, plan); errors.Is(err, hostedexecution.ErrOwnership) {
				return status, err
			}
		}
		return status, nil
	}
	actual, observed, err := p.findExecution(ctx, d, plan)
	if err != nil {
		if !errors.Is(err, hostedexecution.ErrOwnership) && !transport.DispatchReady && !transport.Revoked && !d.Stopping {
			return status, nil
		}
		return status, err
	}
	status = statusFor(d, observed.Phase)
	if terminal(status) {
		return status, p.remember(ctx, ref, status, false)
	}
	if transport.Revoked || d.Stopping {
		return status, nil
	}
	if status.Phase != "running" || p.deadlines.Verify(ctx, plan.deadlineSpec()) != nil {
		status.AdmissionPending = true
		return status, nil
	}
	if err := p.network.VerifyExecution(ctx, p.config, plan.ResourceID(), actual); err != nil {
		if errors.Is(err, hostedexecution.ErrOwnership) {
			return status, err
		}
		status.AdmissionPending = true
	}
	return status, nil
}
