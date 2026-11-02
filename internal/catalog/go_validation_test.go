package catalog_test

import (
	"strings"
	"testing"

	"github.com/wtiger001/teststudio/internal/catalog"
)

func TestGoPublicationRejectsRecursivePackagesAndOverlappingSubtestCounts(t *testing.T) {
	definition := check("GO-1")
	definition.Runner, definition.Type, definition.Source, definition.Selector, definition.Check = "go", "unit", "fresh-go-suite", "./unit::TestRows", nil
	artifact := catalog.Artifact{Image: "sha256:" + strings.Repeat("a", 64), Discovered: map[string]string{definition.ID: definition.Selector}}
	draft := catalog.Draft{Tests: []catalog.Test{definition}, Artifact: artifact}
	if issues := catalog.Validate(draft); len(issues) != 0 {
		t.Fatalf("valid Go publication: %+v", issues)
	}
	child := definition
	child.ID = "GO-2"
	child.Selector += "/child"
	draft.Tests = append(draft.Tests, child)
	draft.Artifact.Discovered[child.ID] = child.Selector
	if issues := catalog.Validate(draft); len(issues) == 0 {
		t.Fatal("parent and child double counting was admitted")
	}
	draft.Tests = []catalog.Test{definition}
	draft.Tests[0].Selector = "./...::TestRows"
	draft.Artifact.Discovered[definition.ID] = draft.Tests[0].Selector
	if issues := catalog.Validate(draft); len(issues) == 0 {
		t.Fatal("recursive Go package was admitted")
	}
}
