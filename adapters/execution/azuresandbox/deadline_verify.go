package azuresandbox

import (
	"encoding/json"
	"strings"

	l "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/logic/armlogic"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func deadlineDocument(value l.Workflow) map[string]any {
	data, err := json.Marshal(value)
	var doc map[string]any
	if err != nil || json.Unmarshal(data, &doc) != nil {
		return nil
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
	return doc
}

func textValue[T ~string](value *T) string {
	if value == nil {
		return ""
	}
	return string(*value)
}

func (c DeadlineConfig) verify(s DeadlineSpec, actual l.Workflow, ready bool) error {
	if c.validateSpec(s) != nil || !strings.EqualFold(textValue(actual.ID), c.resourceID(s)) || textValue(actual.Name) != c.name(s) || !strings.EqualFold(textValue(actual.Type), "Microsoft.Logic/workflows") || actual.Properties == nil {
		return hostedexecution.ErrOwnership
	}
	doc := deadlineDocument(actual)
	if doc == nil {
		return hostedexecution.ErrOwnership
	}
	properties, _ := doc["properties"].(map[string]any)
	definition, _ := properties["definition"].(map[string]any)
	triggers, _ := definition["triggers"].(map[string]any)
	trigger, _ := triggers["Deadline"].(map[string]any)
	if evaluated, exists := trigger["evaluatedRecurrence"]; exists {
		if hash(evaluated) != hash(trigger["recurrence"]) {
			return hostedexecution.ErrOwnership
		}
		delete(trigger, "evaluatedRecurrence")
	}
	if hash(doc) != hash(deadlineDocument(c.definition(s))) {
		return hostedexecution.ErrOwnership
	}
	if ready && textValue(actual.Properties.ProvisioningState) != string(l.WorkflowProvisioningStateSucceeded) {
		return hostedexecution.ErrUnavailable
	}
	return nil
}
