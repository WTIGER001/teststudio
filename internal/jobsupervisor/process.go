package jobsupervisor

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/wtiger001/teststudio/internal/execution"
)

type Process struct {
	PrivateValues []string
	Runner        string
	Directory     string
	Output        string
	Temporary     string
	StopGrace     time.Duration
	Command       func(context.Context) *exec.Cmd
}

func (p *Process) command(ctx context.Context, discovery bool) (*exec.Cmd, error) {
	if p.Command != nil {
		return p.Command(ctx), nil
	}
	switch p.Runner {
	case "go":
		args := []string{}
		if discovery {
			args = append(args, "--discover")
		}
		return exec.CommandContext(ctx, "/runner/teststudio-go", args...), nil
	case "playwright":
		for _, name := range []string{"/usr/local/bin/node", "/usr/bin/node", "/bin/node"} {
			if info, err := os.Stat(name); err == nil && info.Mode().IsRegular() {
				args := []string{"/runner/run.mjs"}
				if discovery {
					args = append(args, "--discover")
				}
				return exec.CommandContext(ctx, name, args...), nil
			}
		}
	}
	return nil, ErrRunner
}

func childEnvironment() []string {
	return []string{"PATH=/usr/local/go/bin:/usr/local/bin:/usr/bin:/bin", "HOME=/tmp", "TMPDIR=/tmp", "LANG=C.UTF-8", "TZ=UTC", "GOCACHE=/tmp/go-cache", "GOPATH=/tmp/go-path", "GOWORK=off", "GOTOOLCHAIN=local", "GOFLAGS=", "CGO_ENABLED=0", "PLAYWRIGHT_BROWSERS_PATH=/ms-playwright", "NODE_PATH=/runner/node_modules"}
}

func (p *Process) Run(ctx context.Context, input execution.CodedInput, wire []byte, onStart func(time.Time)) (Prepared, error) {
	if input.Test.Runner != p.Runner || p.Directory == "" || p.Output == "" {
		return Prepared{}, ErrRunner
	}
	root, err := os.OpenRoot(p.Output)
	if err != nil {
		return Prepared{}, ErrRunner
	}
	defer root.Close()
	for _, name := range []string{"report.json", "report.tmp"} {
		if _, err := root.Lstat(name); err == nil {
			return Prepared{}, ErrRunner
		} else if !errors.Is(err, os.ErrNotExist) {
			return Prepared{}, ErrRunner
		}
	}
	started := time.Now().UTC()
	wall, stop := context.WithTimeout(ctx, time.Duration(input.TimeoutMs)*time.Millisecond+60*time.Second)
	defer stop()
	seen := false
	onLine := func(line string) {
		if seen || onStart == nil {
			return
		}
		data, ok := strings.CutPrefix(line, "TESTSTUDIO_EVENT ")
		if !ok {
			return
		}
		var event struct {
			Event, TestID, Selector string
			At                      time.Time
		}
		if json.Unmarshal([]byte(data), &event) != nil || event.Event != "test.started" || event.TestID != input.Test.ID || event.Selector != input.Test.Selector || event.At.Before(started.Add(-time.Minute)) || event.At.After(time.Now().Add(time.Minute)) {
			return
		}
		seen = true
		onStart(event.At)
	}
	stdout, stderr, exit, runErr := p.runCommand(wall, wire, false, onLine)
	if errors.Is(runErr, ErrRunner) {
		return Prepared{}, runErr
	}
	report, readErr := readReport(root, input, runErr)
	failure := ""
	if readErr != nil {
		failure = "worker"
		if errors.Is(wall.Err(), context.DeadlineExceeded) {
			failure = "timeout"
		} else if errors.Is(wall.Err(), context.Canceled) {
			failure = "cancelled"
		}
		state := "unknown"
		if failure == "timeout" {
			state = "timed-out"
		}
		report = execution.CodedReport{Schema: "teststudio.worker/v1", TestID: input.Test.ID, Selector: input.Test.Selector, Result: execution.Result{TestID: input.Test.ID, State: state, FailureKind: failure, Error: "The coded runner produced no valid matching report.", StartedAt: started, FinishedAt: time.Now().UTC(), DurationUnavailable: true, WorkerDurationMs: float64(time.Since(started)) / float64(time.Millisecond), RunnerStatus: "worker"}, EvidenceErrors: []string{"Structured runner observation was unavailable; available process logs remain private evidence."}}
	}
	prepared, err := p.snapshot(root, input, report, &stdout, &stderr)
	if err != nil {
		return Prepared{}, err
	}
	prepared.Submission.ProcessExitCode = exit
	prepared.Submission.InfrastructureFailure = failure
	return prepared, nil
}

func readReport(root *os.Root, input execution.CodedInput, exitError error) (execution.CodedReport, error) {
	data, err := readRegular(root, "report.json", 2<<20)
	if err != nil {
		return execution.CodedReport{}, err
	}
	var report execution.CodedReport
	if json.Unmarshal(data, &report) != nil {
		return report, ErrRunner
	}
	report.Result = execution.NormalizeTimeout(report.Result)
	if err := execution.ValidateCodedReport(input, report, exitError); err != nil {
		return report, err
	}
	return report, nil
}

func plainFile(name string) bool {
	return name != "" && name != "." && name != ".." && len(name) <= 255 && filepath.Base(name) == name && !strings.ContainsAny(name, "\\\x00\r\n")
}

func (p *Process) String() string   { return "hosted runner process: " + p.Runner }
func (p *Process) GoString() string { return p.String() }
