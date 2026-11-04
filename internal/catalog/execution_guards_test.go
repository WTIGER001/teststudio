package catalog_test

import (
	"strings"
	"testing"

	"github.com/wtiger001/teststudio/internal/catalog"
)

func TestExecutionGuardsRejectInvalidFrozenDependencySelections(t *testing.T) {
	s, _ := setup(t)
	a, b := check("A"), check("B")
	a.Dependencies = []string{b.ID}
	for _, scenario := range []struct {
		name  string
		tests []catalog.Test
		issue string
	}{
		{"missing", []catalog.Test{a}, "unknown selected test B"},
		{"cycle", []catalog.Test{a, func() catalog.Test { v := b; v.Dependencies = []string{a.ID}; return v }()}, "dependency cycle"},
		{"disabled", []catalog.Test{a, func() catalog.Test { v := b; v.State = "disabled"; return v }()}, "B is disabled"},
		{"duplicate", []catalog.Test{b, b}, "duplicate selected test B"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			guards, err := s.ExecutionGuards(t.Context(), "alpha", "health", scenario.tests, false)
			if err == nil || !strings.Contains(err.Error(), scenario.issue) || len(guards) != 0 {
				t.Fatal("invalid frozen dependency selection receives admission guards", err)
			}
		})
	}
}
