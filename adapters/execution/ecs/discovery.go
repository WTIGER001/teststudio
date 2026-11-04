package ecsworker

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func (p *Provider) definition(ctx context.Context, plan *Plan, arn string, cleanup bool) (*ecs.DescribeTaskDefinitionOutput, error) {
	value, err := p.ecs.DescribeTaskDefinition(ctx, &ecs.DescribeTaskDefinitionInput{TaskDefinition: aws.String(arn), Include: []types.TaskDefinitionField{types.TaskDefinitionFieldTags}}, singleECS)
	if err != nil {
		return nil, err
	}
	if err := plan.verifyDefinition(value, cleanup); err != nil || aws.ToString(value.TaskDefinition.TaskDefinitionArn) != arn {
		return nil, hostedexecution.ErrOwnership
	}
	return value, nil
}

func (p *Provider) findDefinition(ctx context.Context, plan *Plan) (string, error) {
	found := ""
	for _, state := range []types.TaskDefinitionStatus{types.TaskDefinitionStatusActive, types.TaskDefinitionStatusInactive, types.TaskDefinitionStatusDeleteInProgress} {
		next := ""
		seen := map[string]bool{}
		for page := 0; page < 4; page++ {
			values, err := p.ecs.ListTaskDefinitions(ctx, &ecs.ListTaskDefinitionsInput{FamilyPrefix: aws.String(plan.Family()), Status: state, NextToken: optional(next), MaxResults: aws.Int32(100)}, singleECS)
			if err != nil || values == nil || len(values.TaskDefinitionArns) > 100 {
				return "", hostedexecution.ErrUnavailable
			}
			for _, arn := range values.TaskDefinitionArns {
				if !exactDefinitionFamily(arn, plan.Family()) {
					continue
				}
				if !plan.validDefinitionARN(arn) {
					return "", hostedexecution.ErrOwnership
				}
				if _, err := p.definition(ctx, plan, arn, true); err != nil {
					return "", err
				}
				if found != "" && found != arn {
					return "", hostedexecution.ErrOwnership
				}
				found = arn
			}
			next = aws.ToString(values.NextToken)
			if next == "" {
				break
			}
			if seen[next] || page == 3 {
				return "", hostedexecution.ErrUnavailable
			}
			seen[next] = true
		}
	}
	if found == "" {
		return "", hostedexecution.ErrUnavailable
	}
	return found, nil
}

func (p *Provider) task(ctx context.Context, plan *Plan, definitionARN, taskARN string, cleanup bool) (types.Task, error) {
	values, err := p.ecs.DescribeTasks(ctx, &ecs.DescribeTasksInput{Cluster: aws.String(p.config.Cluster), Tasks: []string{taskARN}, Include: []types.TaskField{types.TaskFieldTags}}, singleECS)
	if err != nil || values == nil {
		return types.Task{}, hostedexecution.ErrUnavailable
	}
	if len(values.Tasks) == 0 && len(values.Failures) == 1 && aws.ToString(values.Failures[0].Arn) == taskARN && aws.ToString(values.Failures[0].Reason) == "MISSING" {
		return types.Task{}, hostedexecution.ErrNotFound
	}
	if len(values.Tasks) != 1 || len(values.Failures) != 0 || aws.ToString(values.Tasks[0].TaskArn) != taskARN || plan.verifyTaskForCleanup(values.Tasks[0], definitionARN, cleanup) != nil {
		return types.Task{}, hostedexecution.ErrOwnership
	}
	return values.Tasks[0], nil
}

func (p *Provider) findTask(ctx context.Context, plan *Plan, definitionARN string) (string, error) {
	found := ""
	for _, desired := range []types.DesiredStatus{types.DesiredStatusRunning, types.DesiredStatusStopped} {
		next := ""
		seen := map[string]bool{}
		for page := 0; page < 4; page++ {
			values, err := p.ecs.ListTasks(ctx, &ecs.ListTasksInput{Cluster: aws.String(p.config.Cluster), Family: aws.String(plan.Family()), DesiredStatus: desired, NextToken: optional(next), MaxResults: aws.Int32(100)}, singleECS)
			if err != nil || values == nil || len(values.TaskArns) > 100 {
				return "", hostedexecution.ErrUnavailable
			}
			for _, arn := range values.TaskArns {
				if !plan.validTaskARN(arn) {
					return "", hostedexecution.ErrOwnership
				}
				if _, err := p.task(ctx, plan, definitionARN, arn, true); err != nil {
					if errors.Is(err, hostedexecution.ErrNotFound) {
						return "", hostedexecution.ErrUnavailable
					}
					return "", err
				}
				if found != "" && found != arn {
					return "", hostedexecution.ErrOwnership
				}
				found = arn
			}
			next = aws.ToString(values.NextToken)
			if next == "" {
				break
			}
			if seen[next] || page == 3 {
				return "", hostedexecution.ErrUnavailable
			}
			seen[next] = true
		}
	}
	if found == "" {
		return "", hostedexecution.ErrUnavailable
	}
	return found, nil
}

func optional(value string) *string {
	if value == "" {
		return nil
	}
	return aws.String(value)
}

func (p *Provider) reconcileReceipts(ctx context.Context, value dispatch, plan *Plan) (dispatch, error) {
	if value.RegisterAttempted && !value.RegisterRejected && value.DefinitionARN == "" {
		arn, err := p.findDefinition(ctx, plan)
		if err != nil {
			return value, err
		}
		value, err = p.update(ctx, plan.job.Reference, false, func(value *dispatch) error {
			if value.DefinitionARN != "" && value.DefinitionARN != arn {
				return hostedexecution.ErrOwnership
			}
			value.DefinitionARN = arn
			return nil
		})
		if err != nil {
			return value, err
		}
	}
	if value.LaunchAttempted && !value.LaunchRejected && value.TaskARN == "" {
		arn, err := p.findTask(ctx, plan, value.DefinitionARN)
		if err != nil {
			return value, err
		}
		return p.update(ctx, plan.job.Reference, false, func(value *dispatch) error {
			if value.TaskARN != "" && value.TaskARN != arn {
				return hostedexecution.ErrOwnership
			}
			value.TaskARN = arn
			return nil
		})
	}
	return value, nil
}
