package runner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/wtiger001/teststudio/internal/gosuite"
)

func Execute(ctx context.Context, input Input, suite, outputDirectory string) Report {
	started := time.Now()
	now := started.UTC()
	report := Report{Schema: "teststudio.go/v1", TestID: input.Test.ID, Selector: input.Test.Selector, Result: Result{TestID: input.Test.ID, State: "unknown", FailureKind: "configuration", DurationUnavailable: true, StartedAt: now, FinishedAt: now, RunnerStatus: "preparation"}, Attachments: []Attachment{}, EvidenceErrors: []string{}}
	fail := func(kind string, err error) Report {
		report.Result.FailureKind = kind
		report.Result.Error = err.Error()
		report.Result.FinishedAt = time.Now().UTC()
		report.Result.WorkerDurationMs = float64(time.Since(started)) / float64(time.Millisecond)
		return report
	}
	if input.TimeoutMs < 1 || input.TimeoutMs > 3600000 {
		return fail("configuration", errors.New("Go timeout must be between 1 and 3600000 milliseconds"))
	}
	suite, manifest, err := loadCatalog(suite)
	if err != nil {
		return fail("configuration", err)
	}
	var selected *gosuite.Entry
	for _, entry := range manifest.Tests {
		if entry.ID == input.Test.ID && entry.Selector() == input.Test.Selector {
			value := entry
			selected = &value
		}
	}
	if selected == nil {
		return fail("configuration", errors.New("Go selection does not match an immutable registered catalog entry"))
	}
	report.Result.Go.Package, report.Result.Go.Test = selected.Package, selected.Test
	temp, err := os.MkdirTemp("", "teststudio-go-")
	if err != nil {
		return fail("preparation", err)
	}
	defer os.RemoveAll(temp)
	environment := append(os.Environ(), "GOWORK=off", "GOTOOLCHAIN=local", "GOPROXY=off", "GOSUMDB=off", "GOFLAGS=", "CGO_ENABLED=0")
	toolCtx, cancelTool := context.WithTimeout(ctx, 5*time.Second)
	tool := command(toolCtx, "version")
	tool.Env = environment
	var toolOutput output
	tool.Stdout = &toolOutput
	tool.Stderr = &toolOutput
	toolErr := tool.Run()
	cancelTool()
	version := strings.Fields(toolOutput.buffer.String())
	if toolErr != nil || len(version) != 4 {
		return fail("preparation", errors.New("the registered Go toolchain is unavailable"))
	}
	report.Result.Go.Toolchain = version[2]
	platform := strings.Split(version[3], "/")
	if len(platform) == 2 {
		report.Result.Go.OS, report.Result.Go.Architecture = platform[0], platform[1]
	}
	buildCtx, cancelBuild := context.WithTimeout(ctx, 45*time.Second)
	build := command(buildCtx, "test", "-c", "-mod=readonly", "-o", filepath.Join(temp, "selected.test"), selected.Package)
	build.Dir = suite
	build.Env = environment
	var buildOutput output
	build.Stdout = &buildOutput
	build.Stderr = &buildOutput
	buildStarted := time.Now()
	buildErr := build.Run()
	cancelBuild()
	report.Result.Go.BuildDurationMs = float64(time.Since(buildStarted)) / float64(time.Millisecond)
	buildExit := exitCode(buildErr)
	report.Result.Go.BuildExitCode = &buildExit
	report.Result.Go.OutputTruncated = buildOutput.truncated
	appendLog(&report, input, outputDirectory, "go-build.log", "go.build", &buildOutput)
	if buildErr != nil {
		return fail("build", errors.New("Go test build failed: "+bounded(buildOutput.buffer.String(), 8192)))
	}
	if _, err := os.Stat(filepath.Join(temp, "selected.test")); err != nil {
		return fail("build", errors.New("Go package produced no executable test binary"))
	}
	events := NewEvents(*selected, func(at time.Time) {
		message, _ := json.Marshal(map[string]any{"event": "test.started", "testId": input.Test.ID, "selector": input.Test.Selector, "at": at})
		fmt.Println("TESTSTUDIO_EVENT " + string(message))
	})
	var stdout, stderr output
	stdout.onLine = events.Line
	executionCtx, cancelExecution := context.WithTimeout(ctx, time.Duration(input.TimeoutMs)*time.Millisecond+5*time.Second)
	process := command(executionCtx, "tool", "test2json", "-t", "-p", selected.Package, filepath.Join(temp, "selected.test"), "-test.v=test2json", "-test.count=1", "-test.run="+gosuite.Pattern(selected.Test), "-test.timeout="+(time.Duration(input.TimeoutMs)*time.Millisecond).String())
	process.Dir = filepath.Join(suite, selected.Package)
	process.Env = environment
	process.Stdout = &stdout
	process.Stderr = &stderr
	exit := process.Run()
	deadline := errors.Is(executionCtx.Err(), context.DeadlineExceeded)
	cancelExecution()
	if len(stdout.line) > 0 || stdout.oversized {
		events.Problem = "Go event stream ended with an incomplete or oversized event."
	}
	details := report.Result.Go
	testExit := exitCode(exit)
	details.ExitCode = &testExit
	details.OutputTruncated = details.OutputTruncated || stdout.truncated || stderr.truncated
	report.Result = events.Result(input.Test.ID, testExit, ctx.Err() != nil, deadline)
	report.Result.Go = details
	report.Result.WorkerDurationMs = float64(time.Since(started)) / float64(time.Millisecond)
	appendLog(&report, input, outputDirectory, "go-events.jsonl", "go.events", &stdout)
	appendLog(&report, input, outputDirectory, "go-stderr.log", "go.stderr", &stderr)
	return report
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return exit.ExitCode()
	}
	return -1
}
func bounded(value string, n int) string {
	if len(value) > n {
		return value[len(value)-n:]
	}
	return value
}
func appendLog(report *Report, input Input, directory, name, stream string, source *output) {
	if !input.Capture.Logs || source.buffer.Len() == 0 {
		return
	}
	if err := os.WriteFile(filepath.Join(directory, name), source.buffer.Bytes(), 0600); err != nil {
		report.EvidenceErrors = append(report.EvidenceErrors, "Go output could not be retained: "+err.Error())
		return
	}
	report.Attachments = append(report.Attachments, Attachment{Path: name, Filename: name, Kind: "log", Stream: stream, ContentType: "text/plain; charset=utf-8"})
}
