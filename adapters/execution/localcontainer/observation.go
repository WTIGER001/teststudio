package localcontainer

import (
	"context"
	"github.com/wtiger001/teststudio/internal/execution"
	"os"
	"os/exec"
	"slices"
	"time"
)

func appendLogs(report *execution.CodedReport, input execution.CodedInput, stdout, stderr *boundedOutput) {
	if !input.Capture.Logs {
		return
	}
	runner := "playwright"
	if input.Test.Runner == "go" {
		runner = "go"
	}
	for _, stream := range []struct {
		name   string
		output *boundedOutput
	}{{"stdout", stdout}, {"stderr", stderr}} {
		present := false
		for _, file := range report.Attachments {
			if file.Kind == "log" && file.Stream == runner+"."+stream.name {
				present = true
			}
		}
		if present {
			continue
		}
		if stream.output.Len() == 0 {
			continue
		}
		body := slices.Clone(stream.output.Bytes())
		if stream.output.Truncated {
			body = append(body, []byte("\n[output truncated at 2 MiB]\n")...)
		}
		report.Attachments = append(report.Attachments, execution.Attachment{Kind: "log", Stream: runner + "." + stream.name, Filename: runner + "-" + stream.name + ".log", ContentType: "text/plain; charset=utf-8", Data: body})
	}
}
func (e *Executor) collectReport(ctx context.Context, container string, input execution.CodedInput, exitErr error, logs bool) (report execution.CodedReport) {
	defer func() {
		if logs {
			report = execution.RedactAPIReport(input, report)
		}
	}()
	now := time.Now().UTC()
	report = execution.CodedReport{Result: execution.Result{TestID: input.Test.ID, DurationUnavailable: true, State: "unknown", FailureKind: "worker-report", StartedAt: now, FinishedAt: now}}
	directory, err := os.MkdirTemp("", "teststudio-worker-")
	if err == nil {
		defer os.RemoveAll(directory)
		copyErr := e.collect(ctx, container, directory, input)
		parsed, readErr := readReport(directory, input, exitErr)
		if readErr == nil {
			report = parsed
			if copyErr != nil {
				report.EvidenceErrors = append(report.EvidenceErrors, copyErr.Error())
			}
		} else {
			report.Result.Error = readErr.Error()
			report.EvidenceErrors = append(report.EvidenceErrors, "Structured worker observation is unavailable: "+readErr.Error())
			if input.Reference != nil {
				if _, _, probeErr := e.container(ctx, *input.Reference); probeErr != nil {
					report.RecoveryError = probeErr
				}
			}
		}
	} else {
		report.Result.Error = err.Error()
		report.RecoveryError = err
	}
	if logs {
		var stdout, stderr boundedOutput
		command := exec.CommandContext(ctx, e.binary(), "logs", container)
		command.Stdout = &stdout
		command.Stderr = &stderr
		if err := command.Run(); err != nil {
			report.EvidenceErrors = append(report.EvidenceErrors, "Worker logs could not be recovered: "+err.Error())
		}
		appendLogs(&report, input, &stdout, &stderr)
	}
	return report
}
