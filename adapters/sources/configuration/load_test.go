package configuration_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wtiger001/teststudio/adapters/sources/configuration"
)

func TestGitLabServersRequireOperatorApprovedHTTPSAndNetworkScope(t *testing.T) {
	file := filepath.Join(t.TempDir(), "servers.json")
	for _, value := range []string{`[{"id":"company","url":"http://gitlab.example.test","network":"private"}]`, `[{"id":"company","url":"https://user:password@gitlab.example.test","network":"private"}]`, `[{"id":"company","url":"https://gitlab.example.test/api","network":"private"}]`, `[{"id":"github","url":"https://gitlab.example.test","network":"private"}]`, `[{"id":"company","url":"https://gitlab.example.test","network":"loopback"}]`, `[{"id":"company","url":"https://gitlab.example.test","network":"private","caFile":"/missing/teststudio-fixture-ca"}]`} {
		if err := os.WriteFile(file, []byte(value), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := configuration.Load(file, true, nil); err == nil {
			t.Fatal("invalid Git server approval was accepted")
		}
	}
	if err := os.WriteFile(file, []byte(`[{"id":"company","url":"https://gitlab.example.test","network":"private"}]`), 0600); err != nil {
		t.Fatal(err)
	}
	providers, err := configuration.Load(file, true, nil)
	if err != nil || providers["company"] == nil || providers["github"] == nil {
		t.Fatal("approved independent GitLab server did not load", err)
	}
}
