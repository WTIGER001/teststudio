package suitespec_test

import (
	"testing"

	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/suitespec"
)

func TestGitDiscoveryAssignsPermanentIdentitiesAndRequiresRenameReview(t *testing.T) {
	manifest := suitespec.Manifest{Suite: "website", Type: "browser", Applications: []string{"app"}, Tests: suitespec.Tests{Project: "chromium"}}
	actual := []suitespec.ListedTest{{File: "browser/check.spec.ts", Line: 8, Project: "chromium", Title: "Find a story", TitlePath: []string{"Search", "Find a story"}}}
	document, err := suitespec.DiscoverCatalog(manifest, catalog.Document{}, actual)
	if err != nil {
		t.Fatal(err)
	}
	first := document.Tests[0]
	if first.State != "disabled" || first.Safety != "needs-review" || first.Framework == nil || first.Application != "app" {
		t.Fatal("unreviewed test became usable")
	}
	document.Tests[0].Title = "Reviewed catalog title"
	document.Tests[0].State = "enabled"
	actual[0].Line = 25
	again, err := suitespec.DiscoverCatalog(manifest, document, actual)
	if err != nil || again.Tests[0].ID != first.ID || again.Tests[0].Title != "Reviewed catalog title" || again.Tests[0].Framework.Line != 25 {
		t.Fatal("line move changed permanent identity", err)
	}
	actual[0].TitlePath = []string{"Search", "Renamed story"}
	if _, err = suitespec.DiscoverCatalog(manifest, document, actual); err == nil {
		t.Fatal("rename silently reassigned history")
	}
	actual[0].TitlePath = []string{"Search", "Find a story"}
	if _, err = suitespec.DiscoverCatalog(manifest, document, append(actual, actual[0])); err == nil {
		t.Fatal("duplicate title mapping accepted")
	}
}
