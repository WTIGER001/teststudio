package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestInspectionResultsArePrivateAndCannotReplaceExistingFiles(t *testing.T) {
	root := t.TempDir()
	filename := filepath.Join(root, "result.json")
	if err := writeNew(filename, map[string]string{"status": "ready"}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filename)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatal("inspection output is not private")
	}
	before, _ := os.ReadFile(filename)
	if err := writeNew(filename, map[string]string{"status": "failed"}); err == nil {
		t.Fatal("existing immutable inspection replaced")
	}
	after, _ := os.ReadFile(filename)
	if !bytes.Equal(before, after) {
		t.Fatal("existing inspection changed")
	}
	link := filepath.Join(root, "alias.json")
	if err := os.Symlink(filename, link); err != nil {
		t.Fatal(err)
	}
	if err := writeNew(link, map[string]string{"status": "failed"}); err == nil {
		t.Fatal("inspection replaced a symbolic link target")
	}
	files, err := filepath.Glob(filepath.Join(root, ".teststudio-*.tmp"))
	if err != nil || len(files) != 0 {
		t.Fatal("temporary inspection files remain")
	}
}
