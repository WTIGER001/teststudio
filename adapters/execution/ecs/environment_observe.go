package ecsworker

import (
	"context"
	"errors"

	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func (p *EnvironmentProvider) Observe(ctx context.Context, ref execution.Reference) (result hostedexecution.Status, resultErr error) {
	defer func() { resultErr = controlError(resultErr) }()
	value, _, plan, err := p.worker.load(ctx, ref)
	if err != nil {
		return result, err
	}
	result = hostedexecution.Status{Reference: ref, Image: plan.job.Image, Deadline: plan.job.Ticket.Deadline, Phase: "pending", AdmissionPending: true}
	gateway, err := p.observeGateway(ctx, value)
	if err != nil {
		return result, err
	}
	ready := gateway.Phase == "running" && !gateway.AdmissionPending && !gateway.InfrastructureFailure
	transport, _, err := p.worker.exchange(ctx, value)
	if err != nil {
		return result, err
	}
	initialized := transport.Gateway != nil && transport.Gateway.Bootstrapped && transport.Gateway.Instance != "" && transport.Gateway.Settings.Address != ""
	failed := value.Environment.Failed || gateway.InfrastructureFailure || terminal(gateway) && !value.Stopping
	if !value.Stopping && transport.DispatchReady && !transport.Revoked && !ready {
		failed = true
	}
	if failed && !value.Environment.Failed {
		value, err = p.worker.update(ctx, ref, false, func(value *dispatch) error { value.Environment.Failed = true; return nil })
		if err != nil {
			return result, err
		}
	}
	if !value.Stopping && !failed && ready && initialized && !transport.Revoked && !value.RegisterAttempted && !value.RegisterRejected {
		if err := p.startWorker(ctx, plan); err != nil {
			if !errors.Is(err, hostedexecution.ErrUnavailable) {
				return result, err
			}
		}
		value, _, _, err = p.worker.load(ctx, ref)
		if err != nil {
			return result, err
		}
	}
	workerStopped := !value.LaunchAttempted || value.LaunchRejected
	if !workerStopped || value.Terminal != nil {
		result, err = p.worker.Observe(ctx, ref)
		if err != nil {
			return result, err
		}
		workerStopped = terminal(result)
	} else if value.RegisterRejected || value.LaunchRejected {
		failed = true
		if !value.Environment.Failed {
			if _, err := p.worker.update(ctx, ref, false, func(value *dispatch) error { value.Environment.Failed = true; return nil }); err != nil {
				return result, err
			}
		}
	}
	result.InfrastructureFailure = failed
	if value.Stopping || failed {
		result.AdmissionPending = false
		if !workerStopped || !terminal(gateway) {
			result.Phase, result.ExitCode, result.NativeCompletion = "running", nil, false
		} else if failed {
			result.Phase, result.ExitCode, result.NativeCompletion = "failed", nil, false
		} else if value.Terminal == nil {
			result.Phase = "stopped"
		}
		return result, nil
	}
	result.AdmissionPending = !ready || !initialized || result.AdmissionPending || !value.LaunchAttempted
	if terminal(result) && result.AdmissionPending {
		result.Phase, result.ExitCode, result.NativeCompletion = "running", nil, false
	}
	return result, nil
}

func (p *EnvironmentProvider) observeGateway(ctx context.Context, value dispatch) (hostedexecution.Status, error) {
	if !value.Environment.GatewayAttempted {
		return hostedexecution.Status{Phase: "stopped"}, nil
	}
	status, err := p.gateway.Observe(ctx, value.Proof.Reference)
	if errors.Is(err, hostedexecution.ErrNotFound) {
		gateway, _, _, lookupErr := p.gateway.control.load(ctx, value.Proof.Reference)
		if lookupErr != nil {
			return status, hostedexecution.ErrUnavailable
		}
		if !gateway.LaunchAttempted || gateway.LaunchRejected {
			status.Phase = "stopped"
			return status, nil
		}
		return status, hostedexecution.ErrUnavailable
	}
	return status, err
}

func (p *EnvironmentProvider) startWorker(ctx context.Context, proof *Plan) error {
	ticket, err := p.authority.ResumeDispatch(ctx, proof.job.Reference)
	if err != nil {
		return hostedexecution.ErrUnavailable
	}
	job := proof.job
	job.Ticket = ticket
	plan, err := newEnvironmentPlan(p.worker.config, *p.worker.environment, job, p.worker.now())
	if err != nil || plan.fingerprint != proof.fingerprint {
		return hostedexecution.ErrOwnership
	}
	if _, err := p.InspectImage(ctx, job.Image); err != nil {
		return err
	}
	return p.worker.startTask(ctx, plan)
}
