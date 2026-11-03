package localcontainer

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/internal/capture"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/preparation"
)

func inspectionInput(t *testing.T, runner string) (*Executor, preparation.InspectionRequest) {
	t.Helper()
	image := os.Getenv("TESTSTUDIO_DISCOVERY_" + strings.ToUpper(runner) + "_IMAGE")
	if image == "" {
		t.Skip("Set discovery fixture images for isolated preparation checks")
	}
	executor := &Executor{Runner: runner, Images: []string{image}}
	id := domain.NewID()
	installation := os.Getenv("TESTSTUDIO_DISCOVERY_INSTALLATION")
	if installation == "" {
		installation = domain.NewID()
	}
	suite := "discovery-go"
	if runner == "playwright" {
		suite = "discovery-browser"
	}
	ref := executor.Reference(execution.Reference{ID: id, Run: id, Installation: "preparation-" + installation, Project: "discovery-tests", Test: "catalog", Attempt: 1, Worker: "inspection-worker"})
	request := preparation.InspectionRequest{Image: image, Suite: suite, Reference: ref}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := executor.CleanupInspection(ctx, request); err != nil {
			t.Error(err)
		}
	})
	return executor, request
}

func assertInspectionRemoved(t *testing.T, ref execution.Reference) {
	t.Helper()
	for _, kind := range []string{"container", "volume"} {
		name := ref.Container
		if kind == "volume" {
			name = ref.Volume
		}
		output, err := exec.Command("docker", kind, "inspect", name).CombinedOutput()
		if err == nil || !strings.Contains(strings.ToLower(string(output)), "no such") {
			t.Fatalf("owned preparation %s remains or cleanup is unverified", kind)
		}
	}
}

func TestIsolatedArtifactDiscoveryListsBothRunnersWithoutExecutingTests(t *testing.T) {
	for _, runner := range []string{"go", "playwright"} {
		t.Run(runner, func(t *testing.T) {
			executor, request := inspectionInput(t, runner)
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			inspection, err := executor.Inspect(ctx, request)
			if err != nil {
				t.Fatalf("inspection: %v log: %s", err, inspection.Log)
			}
			if len(inspection.Discovery.Tests) != 2 || inspection.ImageID == "" || inspection.Runtime == "" || inspection.FinishedAt.Before(inspection.StartedAt) {
				t.Fatalf("incomplete discovery: %+v", inspection)
			}
			if runner == "go" && !inspection.Discovery.Tests[1].ParentOnly {
				t.Fatal("dynamic subtest path was falsely verified")
			}
			if runner == "playwright" && inspection.Discovery.Manifest.Tests.Project != "chromium" {
				t.Fatal("Playwright project selection was lost")
			}
			assertInspectionRemoved(t, request.Reference)
		})
	}
}

func TestDiscoveryDestinationFailureAndCancellationKeepCleanupScoped(t *testing.T) {
	executor, request := inspectionInput(t, "playwright")
	request.Suite = "wrong-suite"
	result, err := executor.Inspect(t.Context(), request)
	if err == nil {
		t.Fatal("another suite was admitted")
	}
	if strings.Contains(result.Log, "must not execute") || strings.Contains(result.Log, "must not run") {
		t.Fatal("discovery executed a fixture hook, setup, or test")
	}
	assertInspectionRemoved(t, request.Reference)
	executor, request = inspectionInput(t, "go")
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := executor.Inspect(ctx, request); err == nil {
		t.Fatal("cancelled preparation succeeded")
	}
	assertInspectionRemoved(t, request.Reference)
}

func TestInspectionRefusesExecutionNamespacesAndMutableImages(t *testing.T) {
	executor, request := inspectionInput(t, "go")
	request.Reference.Installation = "ordinary-test-execution"
	if _, err := executor.Inspect(t.Context(), request); err == nil {
		t.Fatal("preparation reused an execution namespace")
	}
	request.Reference.Installation = "preparation-test"
	request.Image = "mutable:latest"
	if _, err := executor.Inspect(t.Context(), request); err == nil {
		t.Fatal("mutable preparation image admitted")
	}
}

func TestDiscoveryRespectsActualContainerIsolation(t *testing.T) {
	executor, request := inspectionInput(t, "go")
	done := make(chan error, 1)
	go func() { _, err := executor.Inspect(t.Context(), request); done <- err }()
	deadline := time.NewTimer(20 * time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case err := <-done:
			if err != nil {
				t.Fatal(err)
			}
			t.Fatal("discovery finished before its isolation could be observed")
		case <-deadline.C:
			t.Fatal("preparation did not create its bounded container")
		case <-ticker.C:
			data, err := exec.Command("docker", "container", "inspect", request.Reference.Container).Output()
			if err != nil {
				continue
			}
			var values []struct {
				State  struct{ Running bool }
				Config struct {
					User string
					Env  []string
				}
				HostConfig struct {
					ReadonlyRootfs bool
					NetworkMode    string
					Memory         int64
					NanoCpus       int64
					Privileged     bool
					CapDrop        []string
					SecurityOpt    []string
					Tmpfs          map[string]string
				}
			}
			if json.Unmarshal(data, &values) != nil || len(values) != 1 || !values[0].State.Running {
				continue
			}
			actual := values[0]
			if actual.Config.User != "1000:1000" || !actual.HostConfig.ReadonlyRootfs || actual.HostConfig.NetworkMode != "none" || actual.HostConfig.Memory != 2<<30 || actual.HostConfig.NanoCpus != 2e9 || actual.HostConfig.Privileged {
				t.Error("preparation isolation differs from the registered profile")
			}
			if !strings.Contains(strings.Join(actual.HostConfig.CapDrop, ","), "ALL") || !strings.Contains(strings.Join(actual.HostConfig.SecurityOpt, ","), "no-new-privileges") {
				t.Error("preparation retains unexpected privileges")
			}
			for _, entry := range actual.Config.Env {
				if strings.HasPrefix(entry, "TESTSTUDIO_") {
					t.Error("preparation received application execution configuration")
				}
			}
			if err := <-done; err != nil {
				t.Fatal(err)
			}
			assertInspectionRemoved(t, request.Reference)
			return
		}
	}
}

func TestInspectedArtifactsExecuteThroughTheExistingManagedRunner(t *testing.T) {
	for _, runner := range []string{"go", "playwright"} {
		t.Run(runner, func(t *testing.T) {
			executor, request := inspectionInput(t, runner)
			inspection, err := executor.Inspect(t.Context(), request)
			if err != nil {
				t.Fatalf("inspection: %v %s", err, inspection.Log)
			}
			test := inspection.Discovery.Catalog.Tests[0]
			id := domain.NewID()
			ref := executor.Reference(execution.Reference{ID: id, Run: id, Installation: "runtime-check-" + os.Getenv("TESTSTUDIO_DISCOVERY_INSTALLATION"), Project: "discovery-tests", Test: test.ID, Attempt: 1, Worker: "runtime-checker"})
			t.Cleanup(func() {
				ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
				defer cancel()
				if err := executor.Cleanup(ctx, ref); err != nil {
					t.Error(err)
				}
			})
			result := executor.Execute(t.Context(), execution.CodedInput{Reference: &ref, RunID: id, Test: test, Artifact: catalog.Artifact{Image: request.Image, Discovered: map[string]string{test.ID: test.Selector}}, Capture: capture.DefaultPolicy(), TimeoutMs: 10000})
			if result.Result.State != "passed" || len(result.EvidenceErrors) != 0 {
				t.Fatalf("prepared execution failed: %+v evidence=%v", result.Result, result.EvidenceErrors)
			}
			assertInspectionRemoved(t, ref)
		})
	}
}
