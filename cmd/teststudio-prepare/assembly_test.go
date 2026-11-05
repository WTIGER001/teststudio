package main

import (
	"flag"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAssemblyOnlyRejectsLocalPreparationBeforeCreatingState(t *testing.T) {
	originalFlags, originalArgs := flag.CommandLine, os.Args
	t.Cleanup(func() { flag.CommandLine, os.Args = originalFlags, originalArgs })
	flag.CommandLine = flag.NewFlagSet("owned-preparation-check", flag.ContinueOnError)
	flag.CommandLine.SetOutput(io.Discard)
	root := t.TempDir()
	database, directory := filepath.Join(root, "must-not-exist.db"), filepath.Join(root, "must-not-exist")
	os.Args = []string{"teststudio-prepare", "--assembly-only", "--database=" + database, "--build-directory=" + directory}
	err := run()
	if err == nil || !strings.Contains(err.Error(), "assembly-only preparation requires hosted registry settings") {
		t.Fatal("local destructive preparation entered assembly-only mode", err)
	}
	for _, path := range []string{database, directory} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatal("invalid assembly mode created state", path, err)
		}
	}
}
