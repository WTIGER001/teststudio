package localcontainer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/execution"
)

func TestRuntimeTimingProfilesFollowToolchainsRatherThanSourceImageRevisions(t *testing.T) {
	directory := t.TempDir()
	metadata := filepath.Join(directory, "labels.json")
	binary := filepath.Join(directory, "docker")
	if err := os.WriteFile(binary, []byte("#!/bin/sh\ncat \"$TESTSTUDIO_RUNTIME_LABELS\"\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TESTSTUDIO_RUNTIME_LABELS", metadata)
	driver := &Executor{Runner: "go", Docker: binary}
	label := `{"teststudio.runtime":"go1.26-linux-arm64-cgo0-source-v1"}`
	if err := os.WriteFile(metadata, []byte(label), 0600); err != nil {
		t.Fatal(err)
	}
	first, err := driver.TimingProfile(t.Context(), catalog.Artifact{Image: "sha256:" + strings.Repeat("a", 64)})
	if err != nil {
		t.Fatal(err)
	}
	second, err := driver.TimingProfile(t.Context(), catalog.Artifact{Image: "sha256:" + strings.Repeat("b", 64)})
	if err != nil || second != first {
		t.Fatal("source image revision reset the timing profile")
	}
	if err := os.WriteFile(metadata, []byte(`{"teststudio.runtime":"go1.27-linux-arm64-cgo0-source-v1"}`), 0600); err != nil {
		t.Fatal(err)
	}
	third, err := driver.TimingProfile(t.Context(), catalog.Artifact{Image: "sha256:" + strings.Repeat("b", 64)})
	if err != nil || third == first {
		t.Fatal("toolchain versions shared a timing profile")
	}
	for _, labels := range []string{`{}`, `{"teststudio.runtime":"invalid/value"}`, `broken`} {
		if err := os.WriteFile(metadata, []byte(labels), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := driver.TimingProfile(t.Context(), catalog.Artifact{Image: "sha256:" + strings.Repeat("b", 64)}); err == nil {
			t.Fatal("unknown Go runtime entered timing history")
		}
	}
}

func TestWorkerProfilesCannotCleanOrRecoverAnotherRunner(t *testing.T) {
	goDriver := &Executor{Runner: "go", Docker: "not-a-real-docker-binary"}
	ref := (&Executor{}).Reference(execution.Reference{ID: "foreign"})
	if err := goDriver.Cleanup(t.Context(), ref); err == nil || !strings.Contains(err.Error(), "profile") {
		t.Fatalf("foreign cleanup: %v", err)
	}
	report := goDriver.Recover(t.Context(), ref, execution.CodedInput{})
	if report.RecoveryError == nil || !strings.Contains(report.RecoveryError.Error(), "profile") {
		t.Fatal("foreign recovery tried another worker profile")
	}
}
