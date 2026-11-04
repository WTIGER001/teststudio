package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wtiger001/teststudio/internal/storage"
)

func TestTransferCommandCreatesPrivateArchiveAndRestoresWithoutExposingData(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source.db")
	archive := filepath.Join(root, "installation.ndjson")
	target := filepath.Join(root, "target.db")
	db, err := storage.Open(t.Context(), source)
	if err != nil {
		t.Fatal(err)
	}
	change, err := storage.Put("fresh", "row", "project", 0, map[string]string{"private": "never-print-this-value"})
	if err == nil {
		err = db.Commit(t.Context(), change)
	}
	db.Close()
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := run(t.Context(), []string{"export", "--database", source, "--archive", archive}, &output); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output.String(), "never-print-this-value") {
		t.Fatal("operator summary exposed private data")
	}
	info, err := os.Stat(archive)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatal("command did not create a private archive", err)
	}
	t.Setenv("TESTSTUDIO_TRANSFER_DESTINATION", target)
	if err := run(t.Context(), []string{"import", "--database-env", "TESTSTUDIO_TRANSFER_DESTINATION", "--archive", archive}, &output); err != nil {
		t.Fatal(err)
	}
	db, err = storage.Open(t.Context(), target)
	if err != nil {
		t.Fatal(err)
	}
	row, err := db.Get(t.Context(), "fresh", "row")
	db.Close()
	if err != nil || !strings.Contains(string(row.Data), "never-print-this-value") {
		t.Fatal("command did not restore the installation record", err)
	}
	if err := run(t.Context(), []string{"import", "--database", target, "--archive", archive}, &output); err == nil {
		t.Fatal("command accepted a nonempty target")
	}
	if strings.Contains(output.String(), "never-print-this-value") {
		t.Fatal("command output exposed archive data")
	}
}

func TestTransferCommandRejectsMissingSourcesAndAmbiguousFlags(t *testing.T) {
	root := t.TempDir()
	missing := filepath.Join(root, "missing.db")
	for _, args := range [][]string{{"export", "--database", missing, "--archive", filepath.Join(root, "missing.ndjson")}, {"import", "--database", "one", "--database-env", "TWO", "--archive", "archive"}, {"import", "--unknown", "must-not-echo"}} {
		var output bytes.Buffer
		err := run(t.Context(), args, &output)
		if err == nil {
			t.Fatal("invalid command was accepted")
		}
		if strings.Contains(err.Error(), "must-not-echo") {
			t.Fatal("flag error exposed an argument")
		}
	}
	if _, err := os.Stat(missing); !os.IsNotExist(err) {
		t.Fatal("export created a missing source database")
	}
	var output bytes.Buffer
	if err := run(t.Context(), []string{"--help"}, &output); err != nil || !strings.Contains(output.String(), "empty destination") {
		t.Fatal("operator help is unavailable", err)
	}
}
