package ecsworker

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/scheduler"
	"github.com/aws/smithy-go"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func rejected(err error) bool {
	var api smithy.APIError
	if !errors.As(err, &api) {
		return false
	}
	switch api.ErrorCode() {
	case "InvalidParameterException", "ClientException", "ClusterNotFoundException", "AccessDeniedException", "ValidationException":
		return true
	}
	return false
}

func missing(err error) bool {
	var api smithy.APIError
	return errors.Is(err, hostedexecution.ErrNotFound) || errors.As(err, &api) && api.ErrorCode() == "ResourceNotFoundException"
}

func (p *Provider) Ensure(ctx context.Context, job hostedexecution.Job) (resultErr error) {
	defer func() { resultErr = controlError(resultErr) }()
	if job.Reference.Gateway != "" {
		return hostedexecution.ErrConfiguration
	}
	plan, err := NewPlan(p.config, job, p.now())
	if err != nil {
		return err
	}
	return p.ensurePlan(ctx, plan)
}

func (p *Provider) ensurePlan(ctx context.Context, plan *Plan) error {
	job := plan.job
	value, winning, err := p.reserve(ctx, plan)
	if err != nil {
		return err
	}
	if !winning {
		if value.Removed || value.RegisterRejected || value.LaunchRejected {
			return hostedexecution.ErrRejected
		}
		_, err := p.Observe(ctx, job.Reference)
		return err
	}
	if err := p.admitTask(ctx); err != nil {
		_, saveErr := p.update(ctx, job.Reference, false, func(value *dispatch) error { value.RegisterRejected = true; return nil })
		if saveErr != nil {
			return hostedexecution.ErrUnavailable
		}
		return hostedexecution.ErrRejected
	}
	if _, err := p.update(ctx, job.Reference, true, func(value *dispatch) error { value.RegisterAttempted = true; return nil }); err != nil {
		return err
	}
	registered, err := p.ecs.RegisterTaskDefinition(ctx, plan.Definition(), singleECS)
	if err != nil {
		if rejected(err) {
			if _, saveErr := p.update(ctx, job.Reference, false, func(value *dispatch) error { value.RegisterRejected = true; return nil }); saveErr == nil {
				return hostedexecution.ErrRejected
			}
		}
		return hostedexecution.ErrUnavailable
	}
	if registered == nil || plan.VerifyDefinition(&ecs.DescribeTaskDefinitionOutput{TaskDefinition: registered.TaskDefinition, Tags: registered.Tags}) != nil {
		return hostedexecution.ErrOwnership
	}
	definitionARN := aws.ToString(registered.TaskDefinition.TaskDefinitionArn)
	value, err = p.update(ctx, job.Reference, true, func(value *dispatch) error {
		if value.DefinitionARN != "" && value.DefinitionARN != definitionARN {
			return hostedexecution.ErrOwnership
		}
		value.DefinitionARN, value.LaunchAttempted = definitionARN, true
		return nil
	})
	if err != nil {
		return err
	}
	request, err := plan.Launch(value.DefinitionARN)
	if err != nil {
		return err
	}
	launched, err := p.ecs.RunTask(ctx, request, singleECS)
	if err != nil || launched == nil || len(launched.Tasks) != 1 || len(launched.Failures) != 0 {
		if rejected(err) || err == nil && launched != nil && len(launched.Tasks) == 0 && len(launched.Failures) > 0 {
			if _, saveErr := p.update(ctx, job.Reference, false, func(value *dispatch) error { value.LaunchRejected = true; return nil }); saveErr == nil {
				return hostedexecution.ErrRejected
			}
		}
		return hostedexecution.ErrUnavailable
	}
	task := launched.Tasks[0]
	if err := plan.verifyTask(task, value.DefinitionARN); err != nil {
		return err
	}
	taskARN := aws.ToString(task.TaskArn)
	value, err = p.update(ctx, job.Reference, true, func(value *dispatch) error {
		if value.TaskARN != "" && value.TaskARN != taskARN {
			return hostedexecution.ErrOwnership
		}
		value.TaskARN, value.DeadlineAttempted = taskARN, true
		return nil
	})
	if err != nil {
		return err
	}
	deadline, err := plan.Deadline(value.TaskARN)
	if err != nil {
		return err
	}
	if _, err := p.scheduler.CreateSchedule(ctx, deadline, singleScheduler); err != nil {
		return hostedexecution.ErrUnavailable
	}
	actual, err := p.getDeadline(ctx, plan)
	if err != nil || plan.VerifyDeadline(value.TaskARN, actual) != nil {
		return hostedexecution.ErrUnavailable
	}
	_, err = p.update(ctx, job.Reference, true, func(value *dispatch) error { value.DeadlineVerified = true; return nil })
	return err
}

func controlError(err error) error {
	for _, public := range []error{hostedexecution.ErrNotFound, hostedexecution.ErrUnavailable, hostedexecution.ErrOwnership, hostedexecution.ErrConfiguration, hostedexecution.ErrRejected} {
		if errors.Is(err, public) {
			return public
		}
	}
	if err != nil {
		return hostedexecution.ErrUnavailable
	}
	return nil
}

func (p *Provider) getDeadline(ctx context.Context, plan *Plan) (*scheduler.GetScheduleOutput, error) {
	return p.scheduler.GetSchedule(ctx, &scheduler.GetScheduleInput{Name: aws.String(plan.resourceName()), GroupName: aws.String(p.config.ScheduleGroup)}, singleScheduler)
}
