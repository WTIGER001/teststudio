package ecsworker

import (
	"context"
	"errors"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/aws/aws-sdk-go-v2/service/scheduler"
	"github.com/aws/smithy-go"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/workerstate"
)

func (p *Provider) Stop(ctx context.Context, ref execution.Reference) (resultErr error) {
	defer func() { resultErr = controlError(resultErr) }()
	value, _, plan, err := p.load(ctx, ref)
	if errors.Is(err, storage.ErrNotFound) {
		return hostedexecution.ErrNotFound
	}
	if err != nil {
		return err
	}
	if value.Terminal != nil || !value.LaunchAttempted || value.LaunchRejected {
		return nil
	}
	value, err = p.reconcileReceipts(ctx, value, plan)
	if err != nil {
		return err
	}
	task, err := p.task(ctx, plan, value.DefinitionARN, value.TaskARN, true)
	if err != nil {
		return hostedexecution.ErrUnavailable
	}
	status, err := plan.taskStatus(task, value.Stopping)
	if err != nil {
		return err
	}
	if terminal(status) {
		return p.remember(ctx, ref, status)
	}
	if _, err := p.update(ctx, ref, false, func(value *dispatch) error { value.Stopping = true; return nil }); err != nil {
		return err
	}
	if _, err := p.ecs.StopTask(ctx, &ecs.StopTaskInput{Cluster: aws.String(p.config.Cluster), Task: aws.String(value.TaskARN), Reason: aws.String("TestStudio controller stop")}, singleECS); err != nil {
		return hostedexecution.ErrUnavailable
	}
	return nil
}

func (p *Provider) authorizeCleanup(ctx context.Context, value dispatch) error {
	transport, _, err := p.exchange(ctx, value)
	if err != nil || !transport.Revoked {
		return hostedexecution.ErrUnavailable
	}
	ref := value.Proof.Reference
	if value.Proof.Operation == "discover" {
		if p.discoveryCleanup == nil {
			return hostedexecution.ErrUnavailable
		}
		return p.discoveryCleanup(ctx, ref)
	}
	entry, _, err := (workerstate.Ledger{Store: p.store}).Get(ctx, ref.Run, ref.ID)
	if err != nil || entry.Reference != ref || !entry.ResultSaved || !entry.EvidenceComplete && !entry.EvidenceAbandoned {
		return hostedexecution.ErrUnavailable
	}
	return nil
}

func definitionMissing(err error) bool {
	var api smithy.APIError
	return missing(err) || errors.As(err, &api) && api.ErrorCode() == "ClientException" && api.ErrorMessage() == "Unable to describe task definition."
}

func (p *Provider) removeDeadline(ctx context.Context, plan *Plan, value dispatch) error {
	if !value.DeadlineAttempted || value.DeadlineDeleted {
		return nil
	}
	actual, err := p.getDeadline(ctx, plan)
	if missing(err) && value.DeadlineVerified {
		_, err = p.update(ctx, plan.job.Reference, false, func(value *dispatch) error { value.DeadlineDeleted = true; return nil })
		return err
	}
	if err != nil || plan.VerifyDeadline(value.TaskARN, actual) != nil {
		return hostedexecution.ErrUnavailable
	}
	if _, err := p.update(ctx, plan.job.Reference, false, func(value *dispatch) error { value.DeadlineVerified = true; return nil }); err != nil {
		return err
	}
	if _, err := p.scheduler.DeleteSchedule(ctx, &scheduler.DeleteScheduleInput{Name: aws.String(plan.job.Reference.Container), GroupName: aws.String(p.config.ScheduleGroup), ClientToken: aws.String(fingerprint([]string{value.Proof.Fingerprint, "delete-deadline"}))}, singleScheduler); err != nil && !missing(err) {
		return hostedexecution.ErrUnavailable
	}
	if _, err := p.getDeadline(ctx, plan); !missing(err) {
		return hostedexecution.ErrUnavailable
	}
	_, err = p.update(ctx, plan.job.Reference, false, func(value *dispatch) error { value.DeadlineDeleted = true; return nil })
	return err
}

func (p *Provider) Remove(ctx context.Context, ref execution.Reference) (resultErr error) {
	defer func() { resultErr = controlError(resultErr) }()
	value, _, plan, err := p.load(ctx, ref)
	if errors.Is(err, storage.ErrNotFound) {
		return hostedexecution.ErrNotFound
	}
	if err != nil {
		return err
	}
	if value.Removed {
		return nil
	}
	if err := p.authorizeCleanup(ctx, value); err != nil {
		return err
	}
	value, err = p.reconcileReceipts(ctx, value, plan)
	if err != nil {
		return err
	}
	if value.LaunchAttempted && !value.LaunchRejected && value.Terminal == nil {
		status, err := p.Observe(ctx, ref)
		if err != nil || !terminal(status) {
			return hostedexecution.ErrUnavailable
		}
	}
	if err := p.removeDeadline(ctx, plan, value); err != nil {
		return err
	}
	if value.DefinitionARN != "" {
		if p.now().Before(value.DeleteCheckAfter) {
			return hostedexecution.ErrUnavailable
		}
		actual, err := p.definition(ctx, plan, value.DefinitionARN, true)
		if !definitionMissing(err) {
			if err != nil {
				return hostedexecution.ErrUnavailable
			}
			if actual.TaskDefinition.Status == types.TaskDefinitionStatusActive {
				if _, err := p.ecs.DeregisterTaskDefinition(ctx, &ecs.DeregisterTaskDefinitionInput{TaskDefinition: aws.String(value.DefinitionARN)}, singleECS); err != nil {
					return hostedexecution.ErrUnavailable
				}
				actual, err = p.definition(ctx, plan, value.DefinitionARN, true)
				if err != nil || actual.TaskDefinition.Status == types.TaskDefinitionStatusActive {
					return hostedexecution.ErrUnavailable
				}
			}
			if actual.TaskDefinition.Status != types.TaskDefinitionStatusDeleteInProgress {
				if _, err := p.update(ctx, ref, false, func(value *dispatch) error { value.DefinitionDeleting = true; return nil }); err != nil {
					return err
				}
				deleted, err := p.ecs.DeleteTaskDefinitions(ctx, &ecs.DeleteTaskDefinitionsInput{TaskDefinitions: []string{value.DefinitionARN}}, singleECS)
				if err != nil || deleted == nil || len(deleted.Failures) != 0 {
					return hostedexecution.ErrUnavailable
				}
			}
			if _, err := p.definition(ctx, plan, value.DefinitionARN, true); !definitionMissing(err) {
				if err != nil {
					return hostedexecution.ErrUnavailable
				}
				_, err = p.update(ctx, ref, false, func(value *dispatch) error { value.DeleteCheckAfter = p.now().UTC().Add(time.Minute); return nil })
				if err != nil {
					return err
				}
				return hostedexecution.ErrUnavailable
			}
		}
	}
	_, err = p.update(ctx, ref, false, func(value *dispatch) error { value.Removed = true; return nil })
	return err
}
