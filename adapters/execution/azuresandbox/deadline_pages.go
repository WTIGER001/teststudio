package azuresandbox

import (
	"net/url"
	"sort"
	"strings"
)

func (c DeadlineConfig) inventoryURI(s DeadlineSpec) string {
	plan, _ := Restore(c.Worker, s.Proof)
	labels := plan.definition()["labels"].(map[string]string)
	parts := make([]string, 0, len(labels))
	for key, value := range labels {
		parts = append(parts, key+"="+value)
	}
	sort.Strings(parts)
	return c.Worker.endpoint() + c.Worker.path() + "/sandboxes?api-version=" + APIVersion + "&labels=" + url.QueryEscape(strings.Join(parts, ","))
}

func (c DeadlineConfig) pageActions(s DeadlineSpec) map[string]any {
	selectArray := securedAction("Select", map[string]any{"from": "@body('List')", "select": map[string]any{"id": "@item()?['id']", "owned": c.ownedExpression(s)}}, "")
	selectEnvelope := securedAction("Select", map[string]any{"from": "@body('List')?['value']", "select": map[string]any{"id": "@item()?['id']", "owned": c.ownedExpression(s)}}, "")
	arrayPage := map[string]any{"type": "If", "expression": "@startsWith(trim(string(body('List'))),'[')", "runAfter": map[string]any{},
		"actions": map[string]any{"SelectArray": selectArray, "ArrayItems": variable("Items", "@body('SelectArray')", "SelectArray"), "ArrayNext": variable("Next", "", "ArrayItems")},
		"else":    map[string]any{"actions": map[string]any{"SelectEnvelope": selectEnvelope, "EnvelopeItems": variable("Items", "@body('SelectEnvelope')", "SelectEnvelope"), "EnvelopeNext": variable("Next", "@coalesce(body('List')?['nextLink'],'')", "EnvelopeItems")}}}
	stopURI := "@concat(" + wdlString(c.Worker.endpoint()+c.Worker.path()+"/sandboxes/") + ",uriComponent(string(items('StopInstances')?['id']))," + wdlString("/stop?api-version="+APIVersion) + ")"
	stop := c.httpAction("POST", stopURI)
	safeID := "@and(equals(items('StopInstances')?['id'],string(items('StopInstances')?['id'])),equals(uriComponent(string(items('StopInstances')?['id'])),string(items('StopInstances')?['id'])),not(empty(string(items('StopInstances')?['id']))),lessOrEquals(length(string(items('StopInstances')?['id'])),128)"
	for _, forbidden := range []string{"/", "\\", ".", "%", "?", "#", ":", "@", "~", "!", "'", "(", ")", "*"} {
		safeID += ",not(contains(string(items('StopInstances')?['id'])," + wdlString(forbidden) + "))"
	}
	safeID += ")"
	instances := map[string]any{"type": "Foreach", "foreach": "@variables('Items')", "operationOptions": "Sequential", "runAfter": map[string]any{},
		"actions": map[string]any{"Owned": map[string]any{"type": "If", "expression": "@equals(items('StopInstances')?['owned'],true)", "runAfter": map[string]any{},
			"actions": map[string]any{"SafeID": map[string]any{"type": "If", "expression": safeID, "runAfter": map[string]any{},
				"actions": map[string]any{"Stop": stop}, "else": map[string]any{"actions": map[string]any{"UnsafeID": variable("Unsafe", true, "")}}}},
			"else": map[string]any{"actions": map[string]any{}}}}}
	bounds := map[string]any{"type": "If", "expression": "@and(lessOrEquals(length(variables('Items')),500),lessOrEquals(variables('Total'),1000))",
		"actions": map[string]any{"StopInstances": instances, "NextCursor": variable("Cursor", "@variables('Next')", "StopInstances")},
		"else":    map[string]any{"actions": unsafePage("Size")}, "runAfter": map[string]any{"Count": []string{"Succeeded"}}}
	count := map[string]any{"type": "IncrementVariable", "inputs": map[string]any{"name": "Total", "value": "@length(variables('Items'))"}, "runAfter": map[string]any{"ArrayPage": []string{"Succeeded"}}}
	list := c.httpAction("GET", "@variables('Cursor')")
	list["runAfter"] = map[string]any{"Visit": []string{"Succeeded"}}
	return map[string]any{
		"Visit":       map[string]any{"type": "AppendToArrayVariable", "inputs": map[string]any{"name": "Visited", "value": "@variables('Cursor')"}, "runAfter": map[string]any{}},
		"List":        list,
		"ReadFailure": map[string]any{"type": "Scope", "actions": unsafePage("Read"), "runAfter": map[string]any{"List": []string{"Failed", "TimedOut"}}},
		"CheckResponse": map[string]any{"type": "If", "expression": "@and(equals(outputs('List')?['statusCode'],200),lessOrEquals(length(string(body('List'))),4194304))",
			"actions": map[string]any{"ArrayPage": arrayPage, "Count": count, "CheckSize": bounds},
			"else":    map[string]any{"actions": unsafePage("Response")}, "runAfter": map[string]any{"List": []string{"Succeeded"}}},
	}
}
