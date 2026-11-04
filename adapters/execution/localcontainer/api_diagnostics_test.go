package localcontainer

import (
	"archive/tar"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wtiger001/teststudio/internal/diagnostics"
	"github.com/wtiger001/teststudio/internal/execution"
)

func TestAPIDiagnosticCollectionHelper(t *testing.T) {
	if os.Getenv("TESTSTUDIO_API_DIAGNOSTIC_HELPER") != "1" {
		return
	}
	args := os.Args
	for len(args) > 0 && args[0] != "--" {
		args = args[1:]
	}
	args = args[1:]
	if args[0] == "logs" {
		fmt.Fprintln(os.Stdout, "actual response-canary")
		os.Exit(0)
	}
	if args[0] != "cp" {
		os.Exit(2)
	}
	file := strings.TrimPrefix(args[1], "owned:/output/")
	if file == "private.zip" {
		os.Exit(3)
	}
	data, err := os.ReadFile(filepath.Join(os.Getenv("TESTSTUDIO_API_DIAGNOSTIC_FILES"), file))
	if err != nil {
		os.Exit(4)
	}
	writer := tar.NewWriter(os.Stdout)
	if writer.WriteHeader(&tar.Header{Name: file, Mode: 0600, Size: int64(len(data))}) != nil {
		os.Exit(5)
	}
	if _, err := writer.Write(data); err != nil {
		os.Exit(6)
	}
	if writer.Close() != nil {
		os.Exit(7)
	}
	os.Exit(0)
}

func TestAPIDiagnosticRecoveryCollectsAndRedactsBeforeRetention(t *testing.T) {
	directory, input, report := contractFixture(t)
	input.Test.Type = "api"
	report.Result.Error = `response contains {"token":"response-canary"}`
	report.Result.State = "failed"
	report.Result.FailureKind = "assertion"
	report.Attachments = []execution.Attachment{{Path: "response.json", Kind: "attachment", Filename: "response.json", ContentType: "application/json"}, {Path: "private.zip", Kind: "trace", Filename: "private.zip", ContentType: "application/zip"}}
	writeReport(t, directory, report)
	if err := os.WriteFile(filepath.Join(directory, "response.json"), []byte(`{"status":"up","token":"response-canary"}`), 0600); err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(t.TempDir(), "docker")
	if err := os.WriteFile(binary, []byte("#!/bin/sh\nexec \"$TESTSTUDIO_API_DIAGNOSTIC_BINARY\" -test.run='^TestAPIDiagnosticCollectionHelper$' -- \"$@\"\n"), 0700); err != nil {
		t.Fatal(err)
	}
	helper, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("TESTSTUDIO_API_DIAGNOSTIC_HELPER", "1")
	t.Setenv("TESTSTUDIO_API_DIAGNOSTIC_BINARY", helper)
	t.Setenv("TESTSTUDIO_API_DIAGNOSTIC_FILES", directory)
	parsed := (&Executor{Docker: binary}).collectReport(t.Context(), "owned", input, fmt.Errorf("assertion failed"), true)
	if parsed.Result.State != "failed" || parsed.Result.DurationMs != 1000 || parsed.Result.DiagnosticsPolicy != diagnostics.Version || len(parsed.Attachments) != 2 || len(parsed.EvidenceErrors) != 1 {
		t.Fatalf("recovery observation changed: %+v", parsed)
	}
	all, _ := json.Marshal(parsed)
	for _, file := range parsed.Attachments {
		all = append(all, file.Data...)
	}
	if strings.Contains(string(all), "canary") {
		t.Fatal("recovery exposes private evidence")
	}
	if !strings.Contains(string(parsed.Attachments[0].Data), `"status":"up"`) {
		t.Fatal("recovery loses useful request context")
	}
}
