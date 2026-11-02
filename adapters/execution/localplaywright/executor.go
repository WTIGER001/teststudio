package localplaywright

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/execution"
	"os"
	"os/exec"
	"regexp"
	"slices"
	"strings"
	"time"
)

var digest = regexp.MustCompile(`^(sha256:[a-f0-9]{64}|[A-Za-z0-9][A-Za-z0-9._:/-]*@sha256:[a-f0-9]{64})$`)
var selector = regexp.MustCompile(`^@[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)

type Executor struct {
	Images []string
	Docker string
}

func (e *Executor) Profile() string { return "playwright-local-isolated-v1" }
func (e *Executor) binary() string {
	if e.Docker != "" {
		return e.Docker
	}
	return "docker"
}
func (e *Executor) Admit(ctx context.Context, test catalog.Test, artifact catalog.Artifact) error {
	if !digest.MatchString(artifact.Image) || !slices.Contains(e.Images, artifact.Image) {
		return errors.New("Playwright image is not enabled in this installation")
	}
	if !selector.MatchString(test.Selector) || artifact.Discovered[test.ID] != test.Selector {
		return errors.New("Playwright requires a discovered permanent test tag selector")
	}
	checkCtx, cancelCheck := context.WithTimeout(ctx, 5*time.Second)
	defer cancelCheck()
	command := exec.CommandContext(checkCtx, e.binary(), "image", "inspect", artifact.Image, "--format", "{{.Id}}")
	if err := command.Run(); err != nil {
		return errors.New("the enabled Playwright image is unavailable locally")
	}
	return nil
}

type boundedOutput struct {
	buffer      bytes.Buffer
	Truncated   bool
	OnLine      func(string)
	line        []byte
	discardLine bool
}

func (b *boundedOutput) Len() int      { return b.buffer.Len() }
func (b *boundedOutput) Bytes() []byte { return b.buffer.Bytes() }

func (b *boundedOutput) Write(p []byte) (int, error) {
	n := len(p)
	if b.OnLine != nil {
		for _, value := range p {
			if value == '\n' {
				if !b.discardLine {
					b.OnLine(string(b.line))
				}
				b.line = b.line[:0]
				b.discardLine = false
				continue
			}
			if len(b.line) < 65536 && !b.discardLine {
				b.line = append(b.line, value)
			} else {
				b.discardLine = true
				b.line = b.line[:0]
			}
		}
	}
	remaining := (2 << 20) - b.Len()
	if len(p) > remaining {
		p = p[:max(remaining, 0)]
		b.Truncated = true
	}
	_, _ = b.buffer.Write(p)
	return n, nil
}

func (e *Executor) Execute(ctx context.Context, input execution.CodedInput) (report execution.CodedReport) {
	started := time.Now()
	fail := func(kind string, err error) execution.CodedReport {
		report.Result = execution.Result{TestID: input.Test.ID, State: "unknown", FailureKind: kind, Error: err.Error(), StartedAt: started.UTC(), FinishedAt: time.Now().UTC(), DurationMs: float64(time.Since(started)) / float64(time.Millisecond)}
		return report
	}
	if err := e.Admit(ctx, input.Test, input.Artifact); err != nil {
		return fail("configuration", err)
	}
	directory, err := os.MkdirTemp("", "teststudio-playwright-")
	if err != nil {
		return fail("worker", err)
	}
	defer os.RemoveAll(directory)
	name := "teststudio-pw-" + domain.NewID()
	defer func() {
		cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()
		if output, err := exec.CommandContext(cleanup, e.binary(), "rm", "--force", "--volumes", name).CombinedOutput(); err != nil {
			if !strings.Contains(strings.ToLower(string(output)), "no such container") {
				report = fail("worker-cleanup", fmt.Errorf("Playwright container cleanup failed: %s", string(output)))
			}
		}
	}()
	payload, _ := json.Marshal(input)
	wall, cancel := context.WithTimeout(ctx, time.Duration(input.TimeoutMs)*time.Millisecond+60*time.Second)
	defer cancel()
	command := exec.CommandContext(wall, e.binary(), "run", "--pull=never", "--name", name, "--label", "teststudio.worker=playwright-local-isolated-v1", "--label", "teststudio.run="+input.RunID, "--init", "--network=none", "--read-only", "--cap-drop=ALL", "--security-opt=no-new-privileges", "--pids-limit=256", "--memory=2g", "--cpus=2", "--shm-size=256m", "--tmpfs", "/tmp:rw,nosuid,size=536870912", "--user", "1000:1000", "--env", "HOME=/tmp", "--mount", "type=volume,dst=/output", "-i", input.Artifact.Image)
	command.Stdin = bytes.NewReader(payload)
	var stdout, stderr boundedOutput
	var progressError error
	startedTest := false
	stdout.OnLine = func(line string) {
		if startedTest || input.OnStart == nil {
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
		if json.Unmarshal([]byte(data), &event) != nil || event.Event != "test.started" || event.TestID != input.Test.ID || event.Selector != input.Test.Selector || event.At.IsZero() || event.At.After(time.Now().Add(time.Minute)) || event.At.Before(started.Add(-time.Minute)) {
			return
		}
		startedTest = true
		progressError = input.OnStart(event.At)
	}
	command.Stdout = &stdout
	command.Stderr = &stderr
	exitErr := command.Run()
	if wall.Err() != nil {
		kind := "cancelled"
		if errors.Is(wall.Err(), context.DeadlineExceeded) {
			kind = "timeout"
		}
		report = fail(kind, wall.Err())
	} else {
		copyCtx, releaseCopy := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
		copyErr := e.collect(copyCtx, name, directory, input)
		releaseCopy()
		parsed, err := readReport(directory, input, exitErr)
		if copyErr != nil && err == nil {
			parsed.EvidenceErrors = append(parsed.EvidenceErrors, copyErr.Error())
		}
		if err != nil {
			report = fail("worker-report", err)
		} else {
			report = parsed
		}
	}
	if progressError != nil {
		report.EvidenceErrors = append(report.EvidenceErrors, "Playwright progress update: "+progressError.Error())
	}
	if input.Capture.Logs {
		for _, stream := range []struct {
			name   string
			output *boundedOutput
		}{{"stdout", &stdout}, {"stderr", &stderr}} {
			if stream.output.Len() > 0 {
				body := slices.Clone(stream.output.Bytes())
				if stream.output.Truncated {
					body = append(body, []byte("\n[output truncated at 2 MiB]\n")...)
				}
				report.Attachments = append(report.Attachments, execution.Attachment{Kind: "log", Stream: "playwright." + stream.name, Filename: "playwright-" + stream.name + ".log", ContentType: "text/plain; charset=utf-8", Data: body})
			}
		}
	}
	return report
}
