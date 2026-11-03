package sources

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"slices"

	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/sourcebundle"
	"github.com/wtiger001/teststudio/internal/suitespec"
)

const editorPlaywrightVersion = "1.64.1"

func StarterDraft() PlaywrightDraft {
	document := catalog.Document{Schema: catalog.SchemaVersion, Tests: []catalog.Test{{ID: "BROWSER-001", Title: "Show a welcome message", Category: "Browser", Type: "browser", Runner: "playwright", State: "enabled", Safety: "safe", Rationale: "Renders a local HTML fixture without external services or credentials", TimeoutSeconds: 20, Source: "checks/welcome.spec.ts", Selector: "@BROWSER-001", Capture: &catalog.CapturePolicy{Logs: true, Screenshots: "always", Trace: "failure", Video: "off", RetentionDays: 7}}}}
	catalogBytes, _ := json.MarshalIndent(document, "", "  ")
	return PlaywrightDraft{Name: "Welcome browser checks", Files: []EditorFile{
		{Path: "checks/welcome.spec.ts", Content: "import { test, expect } from '@playwright/test'\n\ntest('Show a welcome message', { tag: '@BROWSER-001' }, async ({ page }, testInfo) => {\n  await page.setContent('<main><h1>Welcome to TestStudio</h1><button>Continue</button></main>')\n  await expect(page.getByRole('heading', { name: 'Welcome to TestStudio' })).toBeVisible()\n  await expect(page.getByRole('button', { name: 'Continue' })).toBeEnabled()\n  await testInfo.attach('welcome', { body: await page.screenshot(), contentType: 'image/png' })\n})\n"},
		{Path: "catalog.json", Content: string(catalogBytes) + "\n"},
		{Path: "playwright.config.ts", Content: "import { defineConfig } from '@playwright/test'\n\nexport default defineConfig({\n  testDir: './checks',\n  outputDir: '/tmp/teststudio-editor-results',\n  projects: [{ name: 'chromium', use: { browserName: 'chromium' } }]\n})\n"},
	}}
}

func assembleDraft(suite string, value PlaywrightDraft) ([]byte, error) {
	if err := validateDraft(value); err != nil {
		return nil, err
	}
	files := slices.Clone(value.Files)
	if value.ManifestPath == "" {
		files = append(files, generatedEditorFiles(suite)...)
	}
	slices.SortFunc(files, func(a, b EditorFile) int {
		if a.Path < b.Path {
			return -1
		}
		if a.Path > b.Path {
			return 1
		}
		return 0
	})
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for _, file := range files {
		header := &zip.FileHeader{Name: file.Path, Method: zip.Deflate}
		header.SetMode(0644)
		if file.Executable {
			header.SetMode(0755)
		}
		entry, err := writer.CreateHeader(header)
		if err != nil {
			return nil, err
		}
		if _, err := entry.Write([]byte(file.Content)); err != nil {
			return nil, err
		}
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	snapshot, err := sourcebundle.Validate(buffer.Bytes(), draftManifestPath(value), suite)
	if err != nil {
		return nil, err
	}
	if snapshot.Manifest.Runner != "playwright" || snapshot.Manifest.BuildProfile != "playwright-node" {
		return nil, errors.New("the editor prepares only Playwright source with the offline playwright-node profile")
	}
	return buffer.Bytes(), nil
}

func draftManifestPath(value PlaywrightDraft) string {
	if value.ManifestPath != "" {
		return value.ManifestPath
	}
	return "teststudio-suite.json"
}

func generatedEditorFiles(suite string) []EditorFile {
	manifest := suitespec.Manifest{Schema: 1, Suite: suite, Type: "browser", Runner: "playwright", Applications: []string{}, ModuleRoot: ".", Tests: suitespec.Tests{Config: "playwright.config.ts", Project: "chromium"}, Catalog: "catalog.json", BuildProfile: "playwright-node"}
	manifestBytes, _ := json.Marshal(manifest)
	packageBytes, _ := json.Marshal(map[string]any{"name": "teststudio-editor-suite", "private": true, "type": "module", "devDependencies": map[string]string{"@playwright/test": editorPlaywrightVersion}})
	lockBytes, _ := json.Marshal(map[string]any{"lockfileVersion": 3, "packages": map[string]any{"": map[string]any{"devDependencies": map[string]string{"@playwright/test": editorPlaywrightVersion}}, "node_modules/@playwright/test": map[string]string{"version": editorPlaywrightVersion}, "node_modules/playwright": map[string]string{"version": editorPlaywrightVersion}, "node_modules/playwright-core": map[string]string{"version": editorPlaywrightVersion}}})
	return []EditorFile{{Path: "teststudio-suite.json", Content: string(manifestBytes)}, {Path: "package.json", Content: string(packageBytes)}, {Path: "package-lock.json", Content: string(lockBytes)}}
}
