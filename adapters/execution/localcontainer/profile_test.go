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
	if err := os.WriteFile(metadata, timingMetadata(label, "c", "arm64"), 0600); err != nil {
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
	if err := os.WriteFile(metadata, timingMetadata(`{"teststudio.runtime":"go1.27-linux-arm64-cgo0-source-v1"}`, "c", "arm64"), 0600); err != nil {
		t.Fatal(err)
	}
	third, err := driver.TimingProfile(t.Context(), catalog.Artifact{Image: "sha256:" + strings.Repeat("b", 64)})
	if err != nil || third == first {
		t.Fatal("toolchain versions shared a timing profile")
	}
	for _, labels := range []string{`{}`, `{"teststudio.runtime":"invalid/value"}`, `broken`} {
		if err := os.WriteFile(metadata, timingMetadata(labels, "c", "arm64"), 0600); err != nil {
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

func TestPlaywrightRuntimeMetadataSeparatesUnlabelledImagesAndSharesKnownToolchains(t *testing.T) {
	directory := t.TempDir()
	metadata := filepath.Join(directory, "labels.json")
	binary := filepath.Join(directory, "docker")
	if err := os.WriteFile(binary, []byte("#!/bin/sh\ncat \"$TESTSTUDIO_RUNTIME_LABELS\"\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TESTSTUDIO_RUNTIME_LABELS", metadata)
	driver := &Executor{Runner: "playwright", Docker: binary}
	first := catalog.Artifact{Image: "sha256:" + strings.Repeat("a", 64)}
	second := catalog.Artifact{Image: "sha256:" + strings.Repeat("b", 64)}
	if err := os.WriteFile(metadata, timingMetadata(`{}`, "a", "arm64"), 0600); err != nil {
		t.Fatal(err)
	}
	a, err := driver.TimingProfile(t.Context(), first)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(metadata, timingMetadata(`{}`, "b", "arm64"), 0600); err != nil {
		t.Fatal(err)
	}
	b, err := driver.TimingProfile(t.Context(), second)
	if err != nil || a == b {
		t.Fatal("unproven runtimes shared history", a, b, err)
	}
	if err := os.WriteFile(metadata, timingMetadata(`{"teststudio.runtime":"playwright1.64.1-linux-arm64"}`, "c", "arm64"), 0600); err != nil {
		t.Fatal(err)
	}
	a, err = driver.TimingProfile(t.Context(), first)
	if err != nil {
		t.Fatal(err)
	}
	b, err = driver.TimingProfile(t.Context(), second)
	if err != nil || a != b {
		t.Fatal("source-only revisions split a known runtime", a, b, err)
	}
	if err := os.WriteFile(metadata, timingMetadata(`{"teststudio.runtime":"playwright1.64.1-linux-arm64"}`, "c", "amd64"), 0600); err != nil {
		t.Fatal(err)
	}
	platform, err := driver.TimingProfile(t.Context(), first)
	if err != nil || platform == a {
		t.Fatal("actual platform change reused timing history", platform, err)
	}
	if err := os.WriteFile(metadata, timingMetadata(`{"teststudio.runtime":"invalid/value"}`, "c", "arm64"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := driver.TimingProfile(t.Context(), first); err == nil {
		t.Fatal("malformed runtime metadata was silently accepted")
	}
}

func timingMetadata(labels, id, architecture string) []byte {
	return []byte(`{"labels":` + labels + `,"id":"sha256:` + strings.Repeat(id, 64) + `","os":"linux","architecture":"` + architecture + `"}`)
}

func TestRuntimeTimingMetadataReadsActualLocalImagePlatform(t *testing.T) {
	driver, input := containerInput(t, "ADAPTER-NOMEDIA", 1000)
	profile, err := driver.TimingProfile(t.Context(), input.Artifact)
	if err != nil || (!strings.HasSuffix(profile, ":linux-arm64") && !strings.HasSuffix(profile, ":linux-amd64")) {
		t.Fatal("actual local image omitted its platform", profile, err)
	}
	t.Log("local runtime timing profile includes its inspected platform")
}
