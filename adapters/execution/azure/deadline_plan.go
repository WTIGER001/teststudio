package azureworker

import (
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	l "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/logic/armlogic"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type DeadlineConfig struct {
	Worker        Config `json:"worker"`
	ResourceGroup string `json:"resourceGroup"`
	StopIdentity  string `json:"stopIdentity"`
}

func (c DeadlineConfig) validate() error {
	if Validate(c.Worker) != nil || !groupName.MatchString(c.ResourceGroup) || strings.HasSuffix(c.ResourceGroup, ".") || strings.EqualFold(c.ResourceGroup, c.Worker.ResourceGroup) || !c.Worker.ownsResource(c.StopIdentity, "Microsoft.ManagedIdentity", "userAssignedIdentities") || strings.EqualFold(c.StopIdentity, c.Worker.PullIdentity) {
		return hostedexecution.ErrConfiguration
	}
	return nil
}

func (c DeadlineConfig) profile() string { return fingerprint([]any{"azure-deadline/v1", c}) }
func (c DeadlineConfig) name(s DeadlineSpec) string {
	return c.Worker.NamePrefix + "-stop-" + fingerprint(s.Reference)[:24]
}
func (c DeadlineConfig) resourceID(s DeadlineSpec) string {
	worker := c.Worker
	worker.ResourceGroup = c.ResourceGroup
	return worker.resourceID("Microsoft.Logic", "workflows", c.name(s))
}

func (c DeadlineConfig) validateSpec(s DeadlineSpec) error {
	valid := false
	for _, runner := range []string{"go", "playwright"} {
		for _, operation := range []string{"", "discover"} {
			valid = valid || validReference(s.Reference, runner, operation)
		}
	}
	if !valid || !checksum.MatchString(s.Specification) || s.At.IsZero() || s.At.Year() < 2000 || s.At.Year() > 2200 || s.ResourceID != c.Worker.resourceID("Microsoft.App", "jobs", c.Worker.NamePrefix+"-"+fingerprint(s.Reference)[:24]) {
		return hostedexecution.ErrOwnership
	}
	return nil
}

func (c DeadlineConfig) definition(s DeadlineSpec) l.Workflow {
	due := s.At.UTC().Truncate(time.Second)
	if due.Before(s.At) {
		due = due.Add(time.Second)
	}
	return l.Workflow{
		Location: to.Ptr(c.Worker.Region),
		Identity: &l.ManagedServiceIdentity{Type: to.Ptr(l.ManagedServiceIdentityTypeUserAssigned), UserAssignedIdentities: map[string]*l.UserAssignedIdentity{c.StopIdentity: {}}},
		Tags: map[string]*string{
			"teststudio-owner": to.Ptr(fingerprint(s.Reference)), "teststudio-spec": to.Ptr(s.Specification), "teststudio-deadline-profile": to.Ptr(c.profile()),
			"teststudio-deadline": to.Ptr(s.At.UTC().Format(time.RFC3339Nano)),
		},
		Properties: &l.WorkflowProperties{State: to.Ptr(l.WorkflowStateEnabled), Definition: map[string]any{
			"$schema":        "https://schema.management.azure.com/providers/Microsoft.Logic/schemas/2016-06-01/workflowdefinition.json#",
			"contentVersion": "1.0.0.0", "parameters": map[string]any{}, "outputs": map[string]any{},
			"triggers": map[string]any{"Deadline": map[string]any{"type": "Recurrence", "recurrence": map[string]any{"frequency": "Minute", "interval": 1, "startTime": due.Format(time.RFC3339)}}},
			"actions": map[string]any{"Stop": map[string]any{"type": "Http", "runAfter": map[string]any{}, "operationOptions": "DisableAsyncPattern",
				"inputs": map[string]any{"method": "POST", "uri": "https://management.azure.com" + s.ResourceID + "/stop?api-version=2025-07-01",
					"authentication": map[string]any{"type": "ManagedServiceIdentity", "identity": c.StopIdentity, "audience": "https://management.azure.com/"},
					"retryPolicy":    map[string]any{"type": "fixed", "count": 2, "interval": "PT10S"},
				},
			}},
		}},
	}
}

func (c DeadlineConfig) verify(s DeadlineSpec, actual l.Workflow, ready bool) error {
	if !strings.EqualFold(value(actual.ID), c.resourceID(s)) || value(actual.Name) != c.name(s) || !strings.EqualFold(value(actual.Type), "Microsoft.Logic/workflows") || actual.Properties == nil {
		return hostedexecution.ErrOwnership
	}
	doc := document(actual)
	if doc == nil {
		return hostedexecution.ErrOwnership
	}
	for _, key := range []string{"id", "name", "type"} {
		delete(doc, key)
	}
	properties, _ := doc["properties"].(map[string]any)
	for _, key := range []string{"accessEndpoint", "changedTime", "createdTime", "provisioningState", "sku", "version"} {
		delete(properties, key)
	}
	if parameters, ok := properties["parameters"].(map[string]any); ok && len(parameters) == 0 {
		delete(properties, "parameters")
	}
	identity, _ := doc["identity"].(map[string]any)
	delete(identity, "principalId")
	delete(identity, "tenantId")
	assigned, _ := identity["userAssignedIdentities"].(map[string]any)
	for _, item := range assigned {
		fields, _ := item.(map[string]any)
		delete(fields, "clientId")
		delete(fields, "principalId")
	}
	definition, _ := properties["definition"].(map[string]any)
	triggers, _ := definition["triggers"].(map[string]any)
	trigger, _ := triggers["Deadline"].(map[string]any)
	if evaluated, ok := trigger["evaluatedRecurrence"]; ok {
		if fingerprint(evaluated) != fingerprint(trigger["recurrence"]) {
			return hostedexecution.ErrOwnership
		}
		delete(trigger, "evaluatedRecurrence")
	}
	if fingerprint(doc) != fingerprint(document(c.definition(s))) {
		return hostedexecution.ErrOwnership
	}
	if ready && value(actual.Properties.ProvisioningState) != l.WorkflowProvisioningStateSucceeded {
		return hostedexecution.ErrUnavailable
	}
	return nil
}
