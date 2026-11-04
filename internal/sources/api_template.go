package sources

import (
	"encoding/json"
	"errors"

	"github.com/wtiger001/teststudio/internal/apipolicy"
	"github.com/wtiger001/teststudio/internal/browserpolicy"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/suitespec"
)

func APIStarterDraft(suite string) (PlaywrightDraft, error) {
	if !domain.ValidID(suite) {
		return PlaywrightDraft{}, errors.New("API starter requires a valid suite identifier")
	}
	document := catalog.Document{Schema: catalog.SchemaVersion, Tests: []catalog.Test{{ID: "API-001", Title: "Check service health", Category: "API", Type: "api", Runner: "playwright", State: "enabled", Safety: "needs-review", Rationale: "Review endpoint behavior and request policy before labeling application requests safe", TimeoutSeconds: 20, Source: "checks/api.spec.ts", Selector: "@API-001", Capture: &catalog.CapturePolicy{Logs: true, Screenshots: "off", Trace: "off", Video: "off", RetentionDays: 7}}}}
	catalogBytes, _ := json.MarshalIndent(document, "", "  ")
	generated := generatedEditorFiles(suite)
	var manifest suitespec.Manifest
	if err := json.Unmarshal([]byte(generated[0].Content), &manifest); err != nil {
		return PlaywrightDraft{}, err
	}
	manifest.Type, manifest.BuildProfile, manifest.Tests.Project = "api", "playwright-environment", "api"
	manifest.Browser = &browserpolicy.Spec{Endpoint: "api", Requests: &apipolicy.Spec{Rules: []apipolicy.Rule{{Method: "GET", PathPrefix: "/health", ReadOnly: true}}, MaxRequests: 10, MaxResponseBytes: 65536}}
	manifestBytes, _ := json.MarshalIndent(manifest, "", "  ")
	generated[0].Content = string(manifestBytes) + "\n"
	value := PlaywrightDraft{Name: "Service API checks", ManifestPath: "teststudio-suite.json", Files: []EditorFile{
		{Path: "checks/api.spec.ts", Content: "import { test, expect } from '@playwright/test'\n\ntest('Check service health', { tag: '@API-001' }, async ({ request }) => {\n  await test.step('Read the registered service health endpoint', async () => {\n    const response = await request.get('/health', { maxRedirects: 0 })\n    expect(response.status()).toBe(200)\n    expect(await response.json()).toHaveProperty('status', 'up')\n  })\n})\n"},
		{Path: "catalog.json", Content: string(catalogBytes) + "\n"},
		{Path: "playwright.config.ts", Content: "import { defineConfig } from '@playwright/test'\n\nexport default defineConfig({ testDir: './checks', projects: [{ name: 'api' }] })\n"},
	}}
	value.Files = append(value.Files, generated...)
	return value, nil
}
