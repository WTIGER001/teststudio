package azureworker

import (
	"encoding/json"
	"regexp"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v4"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

var executionName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9-]{0,79}$`)

type Observation struct {
	ExecutionID string
	Phase       string
	StartedAt   *time.Time
	FinishedAt  *time.Time
}

func value[T any](p *T) (result T) {
	if p != nil {
		return *p
	}
	return
}

func normalize(v any) any {
	switch object := v.(type) {
	case map[string]any:
		for key, child := range object {
			if list, ok := child.([]any); ok && len(list) == 0 {
				delete(object, key)
				continue
			}
			object[key] = normalize(child)
		}
	case []any:
		for i, child := range object {
			object[i] = normalize(child)
		}
	}
	return v
}

func document(v any) map[string]any {
	data, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	var result map[string]any
	if json.Unmarshal(data, &result) != nil {
		return nil
	}
	normalize(result)
	return result
}

func clearEphemeral(containers any) {
	list, _ := containers.([]any)
	for _, item := range list {
		container, _ := item.(map[string]any)
		resources, _ := container["resources"].(map[string]any)
		delete(resources, "ephemeralStorage")
	}
}

func (p *Plan) VerifyDefinition(actual a.Job) error {
	if err := p.VerifyRetainedDefinition(actual); err != nil {
		return err
	}
	if value(actual.Properties.ProvisioningState) != a.JobProvisioningStateSucceeded {
		return hostedexecution.ErrUnavailable
	}
	return nil
}

func (p *Plan) VerifyRetainedDefinition(actual a.Job) error {
	if !strings.EqualFold(value(actual.ID), p.ResourceID()) || value(actual.Name) != p.name || !strings.EqualFold(value(actual.Type), "Microsoft.App/jobs") || actual.Properties == nil {
		return hostedexecution.ErrOwnership
	}
	doc := document(actual)
	if doc == nil {
		return hostedexecution.ErrOwnership
	}
	for _, key := range []string{"id", "name", "type", "systemData"} {
		delete(doc, key)
	}
	properties, _ := doc["properties"].(map[string]any)
	for _, key := range []string{"provisioningState", "eventStreamEndpoint", "outboundIpAddresses"} {
		delete(properties, key)
	}
	identity, _ := doc["identity"].(map[string]any)
	delete(identity, "principalId")
	delete(identity, "tenantId")
	assigned, _ := identity["userAssignedIdentities"].(map[string]any)
	for _, entry := range assigned {
		fields, _ := entry.(map[string]any)
		delete(fields, "clientId")
		delete(fields, "principalId")
	}
	template, _ := properties["template"].(map[string]any)
	clearEphemeral(template["containers"])
	if fingerprint(doc) != fingerprint(document(p.Definition())) {
		return hostedexecution.ErrOwnership
	}
	return nil
}

func (p *Plan) VerifyReceipt(receipt a.JobExecutionBase) error {
	name := value(receipt.Name)
	if !executionName.MatchString(name) || !strings.EqualFold(value(receipt.ID), p.ResourceID()+"/executions/"+name) {
		return hostedexecution.ErrOwnership
	}
	return nil
}

func (p *Plan) VerifyExecutionOwnership(receipt a.JobExecutionBase, actual a.JobExecution) error {
	if p.VerifyReceipt(receipt) != nil || value(actual.Name) != value(receipt.Name) || value(actual.ID) != "" && !strings.EqualFold(value(actual.ID), value(receipt.ID)) || value(actual.Type) != "" && !strings.EqualFold(value(actual.Type), "Microsoft.App/jobs/executions") || actual.Properties == nil || actual.Properties.Template == nil {
		return hostedexecution.ErrOwnership
	}
	template := actual.Properties.Template
	if len(template.Containers) != 1 || template.Containers[0] == nil || len(template.InitContainers) != 0 {
		return hostedexecution.ErrOwnership
	}
	container := *template.Containers[0]
	if len(container.Env) != 2 {
		return hostedexecution.ErrOwnership
	}
	env := map[string]string{}
	for _, field := range container.Env {
		if field == nil || field.Name == nil || field.Value == nil || field.SecretRef != nil {
			return hostedexecution.ErrOwnership
		}
		key := value(field.Name)
		if _, duplicate := env[key]; duplicate {
			return hostedexecution.ErrOwnership
		}
		env[key] = value(field.Value)
	}
	if env["TESTSTUDIO_JOB_EXECUTION"] != p.job.Reference.ID || fingerprint(env["TESTSTUDIO_JOB_TOKEN"]) != p.ticketHash {
		return hostedexecution.ErrOwnership
	}
	container.Env = nil
	if container.Resources != nil {
		resources := *container.Resources
		resources.EphemeralStorage = nil
		container.Resources = &resources
	}
	wanted := p.template().Containers[0]
	expected := a.JobExecutionContainer{Name: wanted.Name, Image: wanted.Image, Command: wanted.Command, Args: wanted.Args, Resources: wanted.Resources}
	if fingerprint(document(container)) != fingerprint(document(expected)) {
		return hostedexecution.ErrOwnership
	}
	return nil
}

func (p *Plan) VerifyExecution(receipt a.JobExecutionBase, actual a.JobExecution) (Observation, error) {
	result := Observation{ExecutionID: value(receipt.ID)}
	if err := p.VerifyExecutionOwnership(receipt, actual); err != nil {
		return result, err
	}
	properties := actual.Properties
	if properties.EndTime != nil && properties.StartTime != nil && properties.EndTime.Before(*properties.StartTime) {
		return result, hostedexecution.ErrOwnership
	}
	switch value(properties.Status) {
	case a.JobExecutionRunningStateProcessing:
		result.Phase = "pending"
	case a.JobExecutionRunningStateRunning:
		result.Phase = "running"
	case a.JobExecutionRunningStateSucceeded, a.JobExecutionRunningStateFailed, a.JobExecutionRunningStateStopped:
		result.Phase = map[a.JobExecutionRunningState]string{a.JobExecutionRunningStateSucceeded: "succeeded", a.JobExecutionRunningStateFailed: "failed", a.JobExecutionRunningStateStopped: "stopped"}[value(properties.Status)]
	default:
		return result, hostedexecution.ErrUnavailable
	}
	if result.Phase == "pending" || result.Phase == "running" {
		if properties.EndTime != nil {
			return result, hostedexecution.ErrOwnership
		}
	}
	if properties.StartTime != nil {
		result.StartedAt = to.Ptr(properties.StartTime.UTC())
	}
	if properties.EndTime != nil {
		result.FinishedAt = to.Ptr(properties.EndTime.UTC())
	}
	return result, nil
}
