package ecsworker

import (
	"reflect"
	"slices"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func (p *Plan) verifyTask(task types.Task, definitionARN string) error {
	return p.verifyTaskForCleanup(task, definitionARN, false)
}

func (p *Plan) verifyTaskForCleanup(task types.Task, definitionARN string, allowMissingTags bool) error {
	state := aws.ToString(task.LastStatus)
	if !p.validTaskARN(aws.ToString(task.TaskArn)) || aws.ToString(task.ClusterArn) != p.config.Cluster || aws.ToString(task.TaskDefinitionArn) != definitionARN || aws.ToString(task.StartedBy) != p.StartedBy() || aws.ToString(task.Group) != "teststudio:"+p.owner[:40] || task.LaunchType != types.LaunchTypeFargate || task.EnableExecuteCommand || len(task.InferenceAccelerators) != 0 || aws.ToString(task.ContainerInstanceArn) != "" || task.Overrides == nil {
		return hostedexecution.ErrOwnership
	}
	if state != "STOPPED" && !allowMissingTags || len(task.Tags) > 0 {
		if err := p.VerifyTags(task.Tags); err != nil {
			return err
		}
	}
	if task.CapacityProviderName != nil && *task.CapacityProviderName != "" && *task.CapacityProviderName != "FARGATE" {
		return hostedexecution.ErrOwnership
	}
	for _, field := range []struct{ actual, expected string }{{aws.ToString(task.Cpu), "2048"}, {aws.ToString(task.Memory), "4096"}, {aws.ToString(task.PlatformVersion), p.config.Platform}, {aws.ToString(task.PlatformFamily), "Linux"}} {
		if field.actual != "" && field.actual != field.expected || state == "RUNNING" && field.actual != field.expected {
			return hostedexecution.ErrOwnership
		}
	}
	override := *task.Overrides
	override.ContainerOverrides = slices.Clone(override.ContainerOverrides)
	if len(override.ContainerOverrides) != 1 || aws.ToString(override.ContainerOverrides[0].Name) != "runner" {
		return hostedexecution.ErrOwnership
	}
	env := override.ContainerOverrides[0].Environment
	if len(env) != 2 {
		return hostedexecution.ErrOwnership
	}
	values := map[string]string{}
	for _, field := range env {
		name := aws.ToString(field.Name)
		if _, duplicate := values[name]; duplicate {
			return hostedexecution.ErrOwnership
		}
		values[name] = aws.ToString(field.Value)
	}
	if values["TESTSTUDIO_JOB_EXECUTION"] != p.job.Reference.ID || fingerprint(values["TESTSTUDIO_JOB_TOKEN"]) != p.ticketHash {
		return hostedexecution.ErrOwnership
	}
	override.ContainerOverrides[0].Environment = nil
	expected := types.TaskOverride{ContainerOverrides: []types.ContainerOverride{{Name: aws.String("runner")}}}
	if !reflect.DeepEqual(document(override), document(expected)) {
		return hostedexecution.ErrOwnership
	}
	if len(task.Containers) > 1 || state == "RUNNING" && len(task.Containers) != 1 {
		return hostedexecution.ErrOwnership
	}
	for _, container := range task.Containers {
		if aws.ToString(container.Name) != "runner" || aws.ToString(container.Image) != "" && aws.ToString(container.Image) != p.job.Image || state == "RUNNING" && aws.ToString(container.Image) != p.job.Image || aws.ToString(container.TaskArn) != "" && aws.ToString(container.TaskArn) != aws.ToString(task.TaskArn) || len(container.ManagedAgents) != 0 || len(container.GpuIds) != 0 || len(container.NetworkBindings) != 0 {
			return hostedexecution.ErrOwnership
		}
	}
	return nil
}

func (p *Plan) taskStatus(task types.Task, stopping bool) (hostedexecution.Status, error) {
	status := hostedexecution.Status{Reference: p.job.Reference, Image: p.job.Image, Deadline: p.job.Ticket.Deadline, Phase: "pending"}
	switch aws.ToString(task.LastStatus) {
	case "PROVISIONING", "PENDING", "ACTIVATING", "DEACTIVATING", "STOPPING", "DEPROVISIONING":
	case "RUNNING":
		status.Phase = "running"
	case "STOPPED":
		status.Phase = "failed"
		if stopping {
			status.Phase = "stopped"
		}
		if len(task.Containers) == 1 && task.Containers[0].ExitCode != nil {
			code := int(*task.Containers[0].ExitCode)
			if code < -1 || code > 255 {
				return status, hostedexecution.ErrOwnership
			}
			status.ExitCode = &code
			if !stopping && code == 0 {
				status.Phase = "succeeded"
			}
		}
	default:
		return status, hostedexecution.ErrOwnership
	}
	return status, nil
}

func sameStatus(left, right hostedexecution.Status) bool {
	return fingerprint(left) == fingerprint(right)
}

func exactDefinitionFamily(arn, family string) bool {
	_, resource, found := strings.Cut(arn, ":task-definition/")
	name, _, revision := strings.Cut(resource, ":")
	return found && revision && name == family
}
