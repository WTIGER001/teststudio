package localcontainer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/gosuite"
	"os/exec"
	"regexp"
	"slices"
	"strings"
	"time"
)

var digest = regexp.MustCompile(`^(sha256:[a-f0-9]{64}|[A-Za-z0-9][A-Za-z0-9._:/-]*@sha256:[a-f0-9]{64})$`)
var selector = regexp.MustCompile(`^@[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)

type Executor struct {
	Environment        bool
	GatewayImage       string
	AllowProjectImages bool
	RequireProject     bool
	ApproveArtifact    func(context.Context, catalog.Artifact) error
	Images             []string
	Docker             string
	Runner             string
}

func (e *Executor) runner() string {
	if e.Runner == "" {
		return "playwright"
	}
	return e.Runner
}
func (e *Executor) Profile() string {
	if e.Environment {
		return "playwright-local-environment-v1"
	}
	return e.runner() + "-local-isolated-v1"
}
func (e *Executor) binary() string {
	if e.Docker != "" {
		return e.Docker
	}
	return "docker"
}
func (e *Executor) Admit(ctx context.Context, test catalog.Test, artifact catalog.Artifact) error {
	if (artifact.Browser != nil) != e.Environment || e.Environment && (e.runner() != "playwright" || len(test.RequiredBindings) > 0) {
		return errors.New("browser target policy does not match the enabled execution profile")
	}
	if e.Environment {
		if !digest.MatchString(e.GatewayImage) {
			return errors.New("environment browser execution requires an immutable trusted gateway image")
		}
		if _, err := e.output(ctx, "image", "inspect", e.GatewayImage, "--format", "{{.Id}}"); err != nil {
			return errors.New("the trusted browser gateway image is unavailable locally")
		}
	}
	if (e.runner() != "playwright" && e.runner() != "go") || (test.Runner != "" && test.Runner != e.runner()) {
		return errors.New("coded runner does not match the enabled execution profile")
	}
	if artifact.Browser != nil && artifact.Browser.Requests != nil {
		if err := e.admitAPIPolicy(ctx, artifact); err != nil {
			return err
		}
	}
	if !digest.MatchString(artifact.Image) {
		return errors.New("coded runner image requires an immutable digest")
	}
	if !slices.Contains(e.Images, artifact.Image) {
		if e.ApproveArtifact == nil {
			return errors.New("coded runner image is not enabled in this installation")
		}
		if err := e.ApproveArtifact(ctx, artifact); err != nil {
			return err
		}
	}
	if artifact.Discovered[test.ID] != test.Selector {
		return errors.New("coded selector is absent from the immutable discovered catalog")
	}
	if e.runner() == "go" {
		if _, err := gosuite.Parse(test.Selector); err != nil {
			return err
		}
	} else if !selector.MatchString(test.Selector) {
		return errors.New("Playwright requires a discovered permanent test tag selector")
	}
	checkCtx, cancelCheck := context.WithTimeout(ctx, 5*time.Second)
	defer cancelCheck()
	command := exec.CommandContext(checkCtx, e.binary(), "image", "inspect", artifact.Image, "--format", "{{.Id}}")
	if err := command.Run(); err != nil {
		return errors.New("the enabled coded runner image is unavailable locally")
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
		report.Result = execution.Result{TestID: input.Test.ID, State: "unknown", FailureKind: kind, Error: err.Error(), StartedAt: started.UTC(), FinishedAt: time.Now().UTC(), DurationUnavailable: true, WorkerDurationMs: float64(time.Since(started)) / float64(time.Millisecond)}
		return report
	}
	if err := e.Admit(ctx, input.Test, input.Artifact); err != nil {
		return fail("configuration", err)
	}

	ref := input.Reference
	if ref == nil {
		value := e.Reference(execution.Reference{ID: domain.NewID(), Installation: "standalone-" + domain.NewID(), Run: input.RunID, Test: input.Test.ID, Attempt: 1, Worker: "standalone"})
		ref = &value
	}
	input.Reference = ref
	defer func() {
		if input.BeforeCleanup != nil {
			if err := input.BeforeCleanup(report); err != nil {
				report.EvidenceErrors = append(report.EvidenceErrors, "Result checkpoint failed; resources remain for recovery: "+err.Error())
				return
			}
		}
		cleanup, cancelCleanup := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancelCleanup()
		cleanupErr := e.Cleanup(cleanup, *ref)
		if input.AfterCleanup != nil {
			if err := input.AfterCleanup(cleanupErr); err != nil {
				report.EvidenceErrors = append(report.EvidenceErrors, "Worker cleanup metadata: "+err.Error())
			}
		}
		if cleanupErr != nil {
			report = fail("worker-cleanup", cleanupErr)
		}
	}()
	if input.BeforeDispatch != nil {
		if err := input.BeforeDispatch(); err != nil {
			return fail("worker-lease", err)
		}
	}
	volumeCtx, cancelVolume := context.WithTimeout(ctx, 5*time.Second)
	volumeErr := e.prepareVolume(volumeCtx, *ref)
	cancelVolume()
	if volumeErr != nil {
		return fail("worker", volumeErr)
	}
	if input.BeforeDispatch != nil {
		if err := input.BeforeDispatch(); err != nil {
			return fail("worker-lease", err)
		}
	}
	if e.Environment {
		if err := e.startGateway(ctx, *ref, input.Browser); err != nil {
			return fail("configuration", err)
		}
		if input.BeforeDispatch != nil {
			if err := input.BeforeDispatch(); err != nil {
				return fail("worker-lease", err)
			}
		}
	} else if input.Browser != nil {
		return fail("configuration", errors.New("offline browser workers cannot receive a target"))
	}
	wire := input
	wire.Artifact.Discovered = map[string]string{input.Test.ID: input.Test.Selector}
	payload, _ := json.Marshal(wire)
	wall, cancel := context.WithTimeout(ctx, time.Duration(input.TimeoutMs)*time.Millisecond+60*time.Second)
	defer cancel()
	temporary := "/tmp:rw,nosuid,size=536870912"
	if e.runner() == "go" {
		temporary += ",exec"
	}
	command := exec.CommandContext(wall, e.binary(), "run", "--pull=never", "--name", ref.Container, "--init", "--network=none", "--read-only", "--cap-drop=ALL", "--security-opt=no-new-privileges", "--pids-limit=256", "--memory=2g", "--cpus=2", "--shm-size=256m", "--tmpfs", temporary, "--user", "1000:1000", "--env", "HOME=/tmp", "--mount", "type=volume,src="+ref.Volume+",dst=/output", "-i", input.Artifact.Image)
	for key, value := range labels(*ref) {
		args := []string{"--label", key + "=" + value}
		command.Args = append(command.Args[:2], append(args, command.Args[2:]...)...)
	}
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
		cleanupObservation, cancelObservation := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		report = e.Recover(cleanupObservation, *ref, input)
		cancelObservation()
		report.Result.State = "unknown"
		if kind == "timeout" {
			report.Result.State = "timed-out"
		}
		report.Result.TestID = input.Test.ID
		report.Result.FailureKind = kind
		report.Result.Error = wall.Err().Error()
		if report.Result.StartedAt.IsZero() {
			report.Result.StartedAt = started.UTC()
			report.Result.FinishedAt = time.Now().UTC()
			report.Result.DurationUnavailable = true
		}
		if report.Result.DurationUnavailable {
			report.Result.WorkerDurationMs = float64(time.Since(started)) / float64(time.Millisecond)
		}
	} else {

		copyCtx, releaseCopy := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
		report = e.collectReport(copyCtx, ref.Container, input, exitErr, false)
		releaseCopy()

	}
	if progressError != nil {
		report.EvidenceErrors = append(report.EvidenceErrors, "Coded worker progress update: "+progressError.Error())
	}
	appendLogs(&report, input, &stdout, &stderr)
	return report
}
