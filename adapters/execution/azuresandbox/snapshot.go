package azuresandbox

import (
	"encoding/json"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func normalize(value any) any {
	data, _ := json.Marshal(value)
	var result any
	_ = json.Unmarshal(data, &result)
	return prune(result)
}

func prune(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		for key, item := range typed {
			item = prune(item)
			if item == nil {
				delete(typed, key)
			} else {
				typed[key] = item
			}
		}
		if len(typed) == 0 {
			return nil
		}
	case []any:
		if len(typed) == 0 {
			return nil
		}
		for index := range typed {
			typed[index] = prune(typed[index])
		}
	}
	return value
}

func snapshot(data []byte) (Snapshot, error) {
	var document map[string]any
	if json.Unmarshal(data, &document) != nil || document == nil {
		return Snapshot{}, hostedexecution.ErrUnavailable
	}
	id, _ := document["id"].(string)
	state, _ := document["state"].(string)
	if !identifier.MatchString(id) {
		return Snapshot{}, hostedexecution.ErrOwnership
	}
	if !knownState(state) {
		return Snapshot{}, hostedexecution.ErrUnavailable
	}
	env, ok := document["environment"].(map[string]any)
	if !ok {
		return Snapshot{}, hostedexecution.ErrUnavailable
	}
	document["environment"] = hash(env)
	for _, key := range []string{"id", "state", "stateDetails", "hostname", "createdAt", "region", "managementUrl", "sandboxGroupId"} {
		delete(document, key)
	}
	if document["skipEgressProxy"] == nil {
		document["skipEgressProxy"] = false
	}
	return Snapshot{ID: id, State: state, DefinitionHash: hash(normalize(document))}, nil
}

func knownState(state string) bool {
	switch state {
	case "Creating", "Running", "Stopped", "Suspended", "Resuming", "Stopping", "Deleting", "Failed":
		return true
	}
	return false
}

func quiescent(state string) bool { return state == "Stopped" || state == "Suspended" }
