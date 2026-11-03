package sources_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/wtiger001/teststudio/internal/sources"
)

func TestStrictSourceDocumentDecoding(t *testing.T) {
	input := upload(t, "HEALTH-1")
	document, err := sources.Decode("json", input.Content)
	if err != nil || len(document.Tests) != 1 {
		t.Fatal(err)
	}
	yaml := "schemaVersion: teststudio.catalog/v1\ntests:\n  - id: HEALTH-1\n    title: Health\n    category: Availability\n    type: https\n    runner: http\n    state: enabled\n    safety: safe\n    rationale: Reads health only\n    timeoutSeconds: 5\n    check:\n      endpoint: api\n      path: /healthz\n      method: GET\n      statusCodes: [200]\n"
	fromYAML, err := sources.Decode("yaml", yaml)
	if err != nil {
		t.Fatal(err)
	}
	a, _ := json.Marshal(document)
	b, _ := json.Marshal(fromYAML)
	if string(a) != string(b) {
		t.Fatal("JSON and YAML disagree")
	}
	for _, item := range []struct{ name, format, content string }{
		{"duplicate JSON", "json", `{"schemaVersion":"a","schemaVersion":"b","tests":[]}`},
		{"duplicate YAML", "yaml", "tests: []\ntests: []"},
		{"alias", "yaml", "tests: &tests []\nschemaVersion: *tests"},
		{"merge", "yaml", "<<: {}\ntests: []"},
		{"tag", "yaml", "tests: !arbitrary []"},
		{"key", "yaml", "1: tests"},
		{"unknown field", "yaml", "tests: []\nunexpected: value"},
		{"trailing", "yaml", "tests: []\n---\ntests: []"},
		{"JSON trailing", "json", "{}{}"},
		{"depth", "yaml", "tests: " + strings.Repeat("[", 40) + strings.Repeat("]", 40)},
		{"size", "json", strings.Repeat(" ", sources.ContentLimit+1)},
		{"invalid UTF8", "yaml", string([]byte{255})},
	} {
		t.Run(item.name, func(t *testing.T) {
			if _, err := sources.Decode(item.format, item.content); err == nil {
				t.Fatal("invalid source accepted")
			}
		})
	}
}

func TestInvalidUploadKeepsDiagnosticsWithoutChangingCatalog(t *testing.T) {
	s, _ := setup(t)
	preview, err := s.Stage(t.Context(), "admin", "alpha", "health", sources.UploadInput{Name: "bad.yaml", Format: "yaml", Content: "tests: !arbitrary []"})
	if err != nil {
		t.Fatal(err)
	}
	if preview.Plan != nil || !strings.Contains(string(preview.Upload.Data), `"state":"failed"`) {
		t.Fatal("missing failed staging diagnostics")
	}
	if _, err := s.Publish(t.Context(), "admin", "alpha", "health", preview.Upload.ID, 1, false); err == nil {
		t.Fatal("malformed upload published")
	}
}
