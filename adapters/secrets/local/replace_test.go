package local_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wtiger001/teststudio/adapters/secrets/local"
)

func TestGitCredentialReplacementIsPrivateAndRejectsStaleValues(t *testing.T) {
	file := filepath.Join(t.TempDir(), "credential.json")
	if err := os.WriteFile(file, []byte("old-fixture-value"), 0600); err != nil {
		t.Fatal(err)
	}
	p, err := local.New([]local.Binding{{ID: "repo", Project: "alpha", Purpose: "source-repository", File: file}})
	if err != nil {
		t.Fatal(err)
	}
	if err = p.Replace(t.Context(), "alpha", "repo", "source-repository", []byte("old-fixture-value"), []byte("new-fixture-value")); err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat(file)
	if info.Mode().Perm() != 0600 {
		t.Fatal("replacement is not private")
	}
	if err = p.Replace(t.Context(), "alpha", "repo", "source-repository", []byte("old-fixture-value"), []byte("stale")); err == nil {
		t.Fatal("stale replacement overwrote current credential")
	}
	data, _ := os.ReadFile(file)
	if string(data) != "new-fixture-value" {
		t.Fatal("replacement changed after conflict")
	}
}
