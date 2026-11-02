package localplaywright

import (
	"context"
	"fmt"
	"github.com/wtiger001/teststudio/internal/capture"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/execution"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func containerInput(t *testing.T, id string, timeout int) (*Executor, execution.CodedInput) {
	t.Helper()
	image := os.Getenv("TESTSTUDIO_PLAYWRIGHT_IMAGE")
	if image == "" {
		t.Skip("Set TESTSTUDIO_PLAYWRIGHT_IMAGE for local container contract checks")
	}
	input := execution.CodedInput{RunID: domain.NewID(), Test: catalog.Test{ID: id, Selector: "@" + id}, Artifact: catalog.Artifact{Image: image, Discovered: map[string]string{id: "@" + id}}, Capture: capture.DefaultPolicy(), TimeoutMs: timeout}
	return &Executor{Images: []string{image}}, input
}
func assertRemoved(t *testing.T, run string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	data, err := exec.CommandContext(ctx, "docker", "ps", "-a", "--filter", "label=teststudio.run="+run, "--format", "{{.Names}}").Output()
	if err != nil || strings.TrimSpace(string(data)) != "" {
		t.Fatalf("worker containers remain: %s %v", data, err)
	}
}

func TestContainerFailureAndTimeoutPreserveMediaAndSteps(t *testing.T) {
	for _, tc := range []struct {
		id, kind string
		timeout  int
	}{{"ADAPTER-FAIL", "assertion", 60000}, {"ADAPTER-TIMEOUT", "timeout", 3000}} {
		t.Run(tc.id, func(t *testing.T) {
			executor, input := containerInput(t, tc.id, tc.timeout)
			started := false
			input.OnStart = func(at time.Time) error { started = true; return nil }
			report := executor.Execute(t.Context(), input)
			if !started {
				t.Error("worker did not report its real test start")
			}
			assertRemoved(t, input.RunID)
			if report.Result.State != "failed" || report.Result.FailureKind != tc.kind || len(report.Result.Steps) == 0 || report.Result.DurationMs <= 0 || len(report.EvidenceErrors) != 0 {
				t.Fatalf("failure contract: %+v errors=%v", report.Result, report.EvidenceErrors)
			}
			kinds := map[string]int{}
			for _, file := range report.Attachments {
				kinds[file.Kind]++
				if len(file.Data) == 0 {
					t.Fatal("empty failure evidence")
				}
			}
			if kinds["trace"] == 0 || kinds["screenshot"] == 0 {
				t.Fatalf("missing failure media: %v", kinds)
			}
		})
	}
}

func TestContainerCancellationCannotPassAndRemovesItsBrowser(t *testing.T) {
	executor, input := containerInput(t, "ADAPTER-TIMEOUT", 60000)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan execution.CodedReport, 1)
	go func() { done <- executor.Execute(ctx, input) }()
	deadline := time.Now().Add(15 * time.Second)
	running := false
	for time.Now().Before(deadline) {
		data, err := exec.CommandContext(t.Context(), "docker", "ps", "--filter", "label=teststudio.run="+input.RunID, "--format", "{{.Names}}").Output()
		if err != nil {
			t.Fatal(err)
		}
		if len(strings.TrimSpace(string(data))) > 0 {
			running = true
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !running {
		cancel()
		select {
		case report := <-done:
			t.Fatalf("worker did not start: %s", report.Result.Error)
		case <-time.After(15 * time.Second):
			t.Fatal("worker did not stop")
		}
	}
	cancel()
	select {
	case report := <-done:
		if report.Result.State != "unknown" || report.Result.FailureKind != "cancelled" {
			t.Fatal(fmt.Sprintf("cancelled worker: %+v", report.Result))
		}
	case <-time.After(20 * time.Second):
		t.Fatal("cancelled worker did not stop")
	}
	assertRemoved(t, input.RunID)
}
