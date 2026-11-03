package access

import (
	"strconv"
	"strings"
	"testing"
)

func TestRolePolicyGraphChecksAllPathsAndDisabledDefinitions(t *testing.T) {
	graph := map[string]RolePolicy{}
	for i := 0; i <= 16; i++ {
		id := strconv.Itoa(i)
		value := RolePolicy{ID: id}
		if i > 0 {
			value.Inherits = []string{strconv.Itoa(i - 1)}
		}
		graph[id] = value
	}
	if err := validatePolicyGraph(graph); err != nil {
		t.Fatal("valid bounded chain rejected", err)
	}
	graph["extra"] = RolePolicy{ID: "extra", Inherits: []string{"0", "16"}}
	if err := validatePolicyGraph(graph); err == nil || !strings.Contains(err.Error(), "16") {
		t.Fatal("a shared shorter path concealed excessive depth", err)
	}
	delete(graph, "extra")
	value := graph["0"]
	value.Disabled = true
	value.Inherits = []string{"16"}
	graph["0"] = value
	if err := validatePolicyGraph(graph); err == nil || !strings.Contains(err.Error(), "cycle") {
		t.Fatal("disabled policy concealed a cycle", err)
	}
	graph = map[string]RolePolicy{"reader": {ID: "reader", Inherits: []string{"unmanaged"}}}
	if err := validatePolicyGraph(graph); err == nil {
		t.Fatal("unmanaged or foreign inheritance accepted")
	}
}
