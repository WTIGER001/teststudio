package sources_test

import (
	"archive/zip"
	"bytes"
	"errors"
	"testing"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/access"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/sources"
)

func TestRepositoryWizardPreviewFindsConfigurationsWithoutSavingOrExecuting(t *testing.T) {
	s, _, provider, _ := repositorySetup(t, false, false)
	var content bytes.Buffer
	writer := zip.NewWriter(&content)
	for name, body := range map[string]string{
		"browser/playwright.config.ts": "throw new Error('Inspection must not evaluate configuration')",
		"other/playwright.config.js":   "export default {}",
		"testing/suite.json":           `{"suiteId":"health","buildProfile":"playwright-node"}`,
	} {
		entry, err := writer.Create("repository-root/" + name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = entry.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	provider.archive = content.Bytes()
	input := sources.RepositoryInspectionRequest{Provider: "github", Repository: "owner/suite", Reference: "heads/main", Mode: "read-only"}
	preview, err := s.PreviewRepository(t.Context(), "admin", "alpha", "wizard", input)
	if err != nil || len(preview.Candidates) != 2 || len(preview.Manifests) != 1 || preview.Candidates[0].ModuleRoot != "browser" || preview.Revision.Commit != provider.commit {
		t.Fatal("unsaved repository inspection", preview, err)
	}
	rows, err := s.ListRepositories(t.Context(), "admin", "alpha", "", 100)
	if err != nil || len(rows.Items) != 1 {
		t.Fatal("preview saved a connection", err)
	}
	for _, bucket := range []string{"repository-syncs/alpha/wizard", "runs", sources.JobBucket("alpha", "health")} {
		rows, err := s.Store.All(t.Context(), bucket, "alpha")
		if err != nil || len(rows) != 0 {
			t.Fatal("inspection queued preparation or execution", bucket, err)
		}
	}
	s.Secrets = gitCredentials{}
	input.Credential = "unapproved"
	before := provider.resolves
	if _, err = s.PreviewRepository(t.Context(), "admin", "alpha", "wizard", input); err == nil || provider.resolves != before {
		t.Fatal("unapproved credential reached the provider", err)
	}
	input.Credential = ""
	input.Mode = "read-write"
	if _, err = s.PreviewRepository(t.Context(), "admin", "alpha", "wizard", input); err == nil || provider.resolves != before {
		t.Fatal("write mode accepted missing credentials", err)
	}
	input.Mode = "read-only"
	control := &access.Service{Store: s.Store, Auth: s.Auth}
	if _, err = control.CreateGrant(t.Context(), "admin", "alpha", access.GrantDefinition{PrincipalKind: permissions.PrincipalUser, PrincipalID: "admin", Permission: "teststudio.sources.manage", Effect: permissions.EffectDeny, Object: "source/monorepo", Reason: "Verify wizard scope"}); err != nil {
		t.Fatal(err)
	}
	if _, err = s.PreviewRepository(t.Context(), "admin", "alpha", "monorepo", input); !errors.Is(err, authorization.ErrForbidden) || provider.resolves != before {
		t.Fatal("preview bypassed an exact source deny", err)
	}
}
