package azuresandbox

import (
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	l "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/logic/armlogic"
)

func wdlString(value string) string { return "'" + strings.ReplaceAll(value, "'", "''") + "'" }

func securedAction(kind string, inputs any, after string) map[string]any {
	action := map[string]any{"type": kind, "inputs": inputs, "runAfter": map[string]any{},
		"runtimeConfiguration": map[string]any{"secureData": map[string]any{"properties": []string{"inputs", "outputs"}}}}
	if after != "" {
		action["runAfter"] = map[string]any{after: []string{"Succeeded"}}
	}
	return action
}

func variable(name string, value any, after string) map[string]any {
	action := map[string]any{"type": "SetVariable", "inputs": map[string]any{"name": name, "value": value}, "runAfter": map[string]any{}}
	if after != "" {
		action["runAfter"] = map[string]any{after: []string{"Succeeded"}}
	}
	return action
}

func unsafePage(prefix string) map[string]any {
	return map[string]any{prefix + "Unsafe": variable("Unsafe", true, ""), prefix + "End": variable("Cursor", "", prefix+"Unsafe")}
}

func (c DeadlineConfig) httpAction(method, uri string) map[string]any {
	action := securedAction("Http", map[string]any{"method": method, "uri": uri,
		"authentication": map[string]any{"type": "ManagedServiceIdentity", "identity": c.StopIdentity, "audience": strings.TrimSuffix(TokenScope, "/.default")},
		"retryPolicy":    map[string]any{"type": "fixed", "count": 2, "interval": "PT5S"}}, "")
	action["operationOptions"] = "DisableAsyncPattern"
	action["limit"] = map[string]any{"timeout": "PT15S"}
	return action
}

func (c DeadlineConfig) definition(s DeadlineSpec) l.Workflow {
	due := s.Proof.Deadline.UTC().Truncate(time.Second)
	if due.Before(s.Proof.Deadline) {
		due = due.Add(time.Second)
	}
	base := c.inventoryURI(s)
	actions := initialVariables(base)
	pages := map[string]any{"type": "Until", "expression": "@empty(variables('Cursor'))",
		"limit": map[string]any{"count": 8, "timeout": "PT2M"}, "operationOptions": "FailWhenLimitsReached",
		"runAfter": map[string]any{"InitializeUnsafe": []string{"Succeeded"}},
		"actions": map[string]any{"CheckCursor": map[string]any{"type": "If", "expression": cursorExpression(base),
			"actions": c.pageActions(s), "else": map[string]any{"actions": unsafePage("Cursor")}, "runAfter": map[string]any{}}}}
	finish := map[string]any{"type": "If", "expression": "@and(empty(variables('Cursor')),equals(variables('Unsafe'),false),equals(actions('Pages')?['status'],'Succeeded'))",
		"actions": map[string]any{}, "else": map[string]any{"actions": map[string]any{"Fail": map[string]any{
			"type": "Terminate", "inputs": map[string]any{"runStatus": "Failed", "runError": map[string]any{"code": "IncompleteSandboxDeadline", "message": "Independent sandbox stop requires complete scoped inventory and acknowledged stop requests."}}, "runAfter": map[string]any{}}}},
		"runAfter": map[string]any{"Pages": []string{"Succeeded", "Failed", "TimedOut"}}}
	actions["Pages"], actions["Complete"] = pages, finish
	return l.Workflow{Location: to.Ptr(c.Worker.Region),
		Identity: &l.ManagedServiceIdentity{Type: to.Ptr(l.ManagedServiceIdentityTypeUserAssigned), UserAssignedIdentities: map[string]*l.UserAssignedIdentity{c.StopIdentity: {}}},
		Tags:     map[string]*string{"teststudio-owner": to.Ptr(hash(s.Proof.Reference)), "teststudio-spec": to.Ptr(s.Proof.Fingerprint), "teststudio-deadline-profile": to.Ptr(c.profile()), "teststudio-deadline": to.Ptr(s.Proof.Deadline.UTC().Format(time.RFC3339Nano))},
		Properties: &l.WorkflowProperties{State: to.Ptr(l.WorkflowStateEnabled), Definition: map[string]any{
			"$schema":        "https://schema.management.azure.com/providers/Microsoft.Logic/schemas/2016-06-01/workflowdefinition.json#",
			"contentVersion": "1.0.0.0", "parameters": map[string]any{}, "outputs": map[string]any{},
			"triggers": map[string]any{"Deadline": map[string]any{"type": "Recurrence", "recurrence": map[string]any{"frequency": "Minute", "interval": 1, "startTime": due.Format(time.RFC3339)},
				"runtimeConfiguration": map[string]any{"concurrency": map[string]any{"runs": 1, "maximumWaitingRuns": 1}}}},
			"actions": actions,
		}}}
}

func initialVariables(base string) map[string]any {
	type entry struct {
		name, kind string
		value      any
	}
	entries := []entry{{"Cursor", "String", base}, {"Next", "String", ""}, {"Items", "Array", []any{}}, {"Visited", "Array", []any{}}, {"Total", "Integer", 0}, {"Unsafe", "Boolean", false}}
	actions := map[string]any{}
	previous := ""
	for index, value := range entries {
		name := "Initialize" + value.name
		if index == 0 {
			name = "Initialize"
		}
		after := map[string]any{}
		if previous != "" {
			after[previous] = []string{"Succeeded"}
		}
		actions[name] = map[string]any{"type": "InitializeVariable", "inputs": map[string]any{"variables": []any{map[string]any{"name": value.name, "type": value.kind, "value": value.value}}}, "runAfter": after}
		previous = name
	}
	return actions
}

func cursorExpression(base string) string {
	token := "last(split(variables('Cursor'),'&'))"
	continuations := []string{}
	for _, key := range []string{"$skiptoken", "skipToken", "continuationToken", "continuation", "$skip", "$top"} {
		prefix := key + "="
		continuations = append(continuations, "and(startsWith("+token+","+wdlString(prefix)+"),greater(length("+token+"),"+strconv.Itoa(len(prefix))+"))")
	}
	return "@and(not(contains(variables('Visited'),variables('Cursor'))),not(contains(variables('Cursor'),'#')),lessOrEquals(length(variables('Cursor')),4096)," +
		"or(equals(variables('Cursor')," + wdlString(base) + "),and(equals(length(split(variables('Cursor'),'&'))," + strconv.Itoa(strings.Count(base, "&")+2) + "),startsWith(variables('Cursor')," + wdlString(base+"&") + "),or(" + strings.Join(continuations, ",") + "))))"
}

func (c DeadlineConfig) ownedExpression(s DeadlineSpec) string {
	plan, _ := Restore(c.Worker, s.Proof)
	labels := plan.definition()["labels"].(map[string]string)
	keys := make([]string, 0, len(labels))
	for key := range labels {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	checks := []string{}
	for _, key := range keys {
		checks = append(checks, "equals(item()?['labels']?["+wdlString(key)+"],"+wdlString(labels[key])+")")
	}
	return "@and(" + strings.Join(checks, ",") + ")"
}
