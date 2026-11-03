package localimage

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/wtiger001/teststudio/internal/sourcebundle"
	"github.com/wtiger001/teststudio/internal/suitespec"
)

func TestOfflineBrowserDependenciesRejectUnpreparedPackagesAndShadowing(t *testing.T) {
	fresh := func() sourcebundle.Snapshot {
		manifest := []byte(`{"devDependencies":{"@playwright/test":"1.64.1"}}`)
		lock := []byte(`{"lockfileVersion":3,"packages":{"":{"devDependencies":{"@playwright/test":"1.64.1"}},"node_modules/@playwright/test":{"version":"1.64.1"},"node_modules/playwright":{"version":"1.64.1"},"node_modules/playwright-core":{"version":"1.64.1"}}}`)
		return sourcebundle.Snapshot{Manifest: suitespec.Manifest{ModuleRoot: "browser"}, Files: map[string][]byte{"browser/package.json": manifest, "browser/package-lock.json": lock}}
	}
	if err := runtimeDependencies(fresh(), "1.64.1"); err != nil {
		t.Fatal(err)
	}
	cases := map[string]func(*sourcebundle.Snapshot){
		"unprepared-dependency": func(s *sourcebundle.Snapshot) {
			s.Files["browser/package.json"] = []byte(`{"devDependencies":{"@playwright/test":"1.64.1","additional-library":"1.0.0"}}`)
		},
		"floating-version": func(s *sourcebundle.Snapshot) {
			s.Files["browser/package.json"] = []byte(`{"devDependencies":{"@playwright/test":"^1.64.1"}}`)
		},
		"mismatched-lock": func(s *sourcebundle.Snapshot) {
			s.Files["browser/package-lock.json"] = []byte(strings.Replace(string(s.Files["browser/package-lock.json"]), `"node_modules/playwright":{"version":"1.64.1"}`, `"node_modules/playwright":{"version":"1.63.0"}`, 1))
		},
		"lock-root-conflict": func(s *sourcebundle.Snapshot) {
			var lock map[string]any
			json.Unmarshal(s.Files["browser/package-lock.json"], &lock)
			lock["packages"].(map[string]any)[""] = map[string]any{}
			s.Files["browser/package-lock.json"], _ = json.Marshal(lock)
		},
		"shadowed-runtime": func(s *sourcebundle.Snapshot) {
			s.Files["browser/node_modules/@playwright/test/index.js"] = []byte("unwanted replacement")
		},
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			snapshot := fresh()
			change(&snapshot)
			if err := runtimeDependencies(snapshot, "1.64.1"); err == nil {
				t.Fatal("unsupported offline dependency context accepted")
			}
		})
	}
}
