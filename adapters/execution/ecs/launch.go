package ecsworker

import (
	"encoding/json"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/aws/aws-sdk-go-v2/service/scheduler"
	scheduletypes "github.com/aws/aws-sdk-go-v2/service/scheduler/types"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

var revision = regexp.MustCompile(`^[1-9][0-9]{0,9}$`)
var taskID = regexp.MustCompile(`^[a-f0-9]{32}$`)

func (p *Plan) validDefinitionARN(value string) bool {
	base := p.config.ecsPrefix() + "task-definition/" + p.family + ":"
	return strings.HasPrefix(value, base) && revision.MatchString(strings.TrimPrefix(value, base))
}

func (p *Plan) validTaskARN(value string) bool {
	base := p.config.ecsPrefix() + "task/" + p.config.clusterName() + "/"
	return strings.HasPrefix(value, base) && taskID.MatchString(strings.TrimPrefix(value, base))
}

func (p *Plan) Launch(definitionARN string) (*ecs.RunTaskInput, error) {
	if !p.validDefinitionARN(definitionARN) || !ticket.MatchString(p.job.Ticket.Token) {
		return nil, hostedexecution.ErrOwnership
	}
	return &ecs.RunTaskInput{
		Cluster: aws.String(p.config.Cluster), TaskDefinition: aws.String(definitionARN),
		LaunchType: ecstypes.LaunchTypeFargate, PlatformVersion: aws.String(p.config.Platform), Count: aws.Int32(1),
		ClientToken: aws.String(p.fingerprint), StartedBy: aws.String(p.StartedBy()), Group: aws.String("teststudio:" + p.owner[:40]),
		EnableExecuteCommand: false, EnableECSManagedTags: false, Tags: p.tags(),
		NetworkConfiguration: &ecstypes.NetworkConfiguration{AwsvpcConfiguration: &ecstypes.AwsVpcConfiguration{
			AssignPublicIp: ecstypes.AssignPublicIpDisabled, Subnets: slices.Clone(p.config.Subnets), SecurityGroups: slices.Clone(p.config.SecurityGroups),
		}},
		Overrides: &ecstypes.TaskOverride{ContainerOverrides: []ecstypes.ContainerOverride{{Name: aws.String("runner"), Environment: []ecstypes.KeyValuePair{
			{Name: aws.String("TESTSTUDIO_JOB_EXECUTION"), Value: aws.String(p.job.Reference.ID)},
			{Name: aws.String("TESTSTUDIO_JOB_TOKEN"), Value: aws.String(p.job.Ticket.Token)},
		}}}},
	}, nil
}

func (p *Plan) Deadline(taskARN string) (*scheduler.CreateScheduleInput, error) {
	if !p.validTaskARN(taskARN) {
		return nil, hostedexecution.ErrOwnership
	}
	input, _ := json.Marshal(map[string]string{"cluster": p.config.Cluster, "task": taskARN, "reason": "TestStudio worker deadline"})
	at := p.job.Ticket.Deadline.UTC().Add(time.Second - time.Nanosecond).Truncate(time.Second)
	return &scheduler.CreateScheduleInput{
		Name: aws.String(p.job.Reference.Container), GroupName: aws.String(p.config.ScheduleGroup),
		ClientToken:        aws.String(fingerprint([]string{p.fingerprint, taskARN, "deadline"})),
		Description:        aws.String("TestStudio owner " + p.owner),
		ScheduleExpression: aws.String("at(" + at.Format("2006-01-02T15:04:05") + ")"), ScheduleExpressionTimezone: aws.String("UTC"),
		State: scheduletypes.ScheduleStateEnabled, ActionAfterCompletion: scheduletypes.ActionAfterCompletionNone,
		FlexibleTimeWindow: &scheduletypes.FlexibleTimeWindow{Mode: scheduletypes.FlexibleTimeWindowModeOff},
		Target: &scheduletypes.Target{
			Arn: aws.String("arn:aws:scheduler:::aws-sdk:ecs:stopTask"), RoleArn: aws.String(p.config.DeadlineRole), Input: aws.String(string(input)),
			RetryPolicy: &scheduletypes.RetryPolicy{MaximumEventAgeInSeconds: aws.Int32(60), MaximumRetryAttempts: aws.Int32(2)},
		},
	}, nil
}
