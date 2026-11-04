package sources_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/wtiger001/teststudio/internal/apipolicy"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/sourcebundle"
	"github.com/wtiger001/teststudio/internal/sources"
)

func TestAPIStarterSavesAndPreparesAnExactRequestRestrictedWorkspace(t *testing.T) {
	s, _, _ := preparedService(t)
	s.BundleRunners = []string{"playwright"}
	value, err := sources.APIStarterDraft("health")
	if err != nil {
		t.Fatal(err)
	}
	value.ID = "api-source"
	row, err := s.SaveDraft(t.Context(), "admin", "alpha", "health", value, 0)
	if err != nil {
		t.Fatal(err)
	}
	job, err := s.PrepareDraft(t.Context(), "admin", "alpha", "health", value.ID, "api-once", row.Revision)
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := s.ReadBundle(t.Context(), "admin", "alpha", "health", job.ID)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := sourcebundle.Validate(bundle.Content, bundle.Manifest, "health")
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Manifest.Type != "api" || snapshot.Manifest.BuildProfile != "playwright-environment" || snapshot.Manifest.Tests.Project != "api" || snapshot.Manifest.Browser == nil || !apipolicy.ReadOnly(snapshot.Manifest.Browser.Requests) {
		t.Fatal("prepared API starter loses its reviewed request policy", snapshot.Manifest)
	}
	var document catalog.Document
	if err := json.Unmarshal(snapshot.Files["catalog.json"], &document); err != nil {
		t.Fatal(err)
	}
	if len(document.Tests) != 1 || document.Tests[0].Type != "api" || document.Tests[0].Safety != "needs-review" || document.Tests[0].Capture.Trace != "off" || strings.Contains(string(snapshot.Files["checks/api.spec.ts"]), "{ page }") {
		t.Fatal("API starter launches a browser or silently grants safe status")
	}
	for _, bucket := range []string{"runs", "run-queue"} {
		rows, err := s.Store.All(t.Context(), bucket, "alpha")
		if err != nil || len(rows) != 0 {
			t.Fatal("API source preparation starts execution", bucket, err)
		}
	}
}
