package jobsupervisor

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"time"

	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/workerprotocol"
)

func (s *Supervisor) Run(ctx context.Context) (Outcome, error) {
	if s.Transport == nil || s.Runner == nil {
		return Outcome{}, ErrRunner
	}
	var packet workerprotocol.Packet
	if err := s.retry(ctx, func() error { var err error; packet, err = s.Transport.Input(ctx); return err }); err != nil {
		return Outcome{}, ErrWork
	}
	if len(packet.Input) > execution.MaxWorkItemBytes || packet.Deadline.IsZero() || !time.Now().Before(packet.Deadline) || packet.Deadline.After(time.Now().Add(2*time.Hour)) {
		return Outcome{}, ErrWork
	}
	var input execution.CodedInput
	if json.Unmarshal(packet.Input, &input) != nil {
		return Outcome{}, ErrWork
	}
	wire, err := execution.MarshalWorkItem(input)
	if err != nil {
		return Outcome{}, ErrWork
	}
	ctx, cancel := context.WithDeadline(ctx, packet.Deadline)
	defer cancel()
	progress := make(chan time.Time, 1)
	stopProgress := make(chan struct{})
	progressDone := make(chan error, 1)
	go func() {
		var at time.Time
		select {
		case at = <-progress:
		case <-stopProgress:
			select {
			case at = <-progress:
			default:
				progressDone <- nil
				return
			}
		case <-ctx.Done():
			progressDone <- ctx.Err()
			return
		}
		progressCtx, stop := context.WithTimeout(ctx, 5*time.Second)
		defer stop()
		err := s.retry(progressCtx, func() error { return s.Transport.Started(progressCtx, at) })
		if errors.Is(err, workerprotocol.ErrDenied) {
			cancel()
		}
		progressDone <- err
	}()
	prepared, err := s.Runner.Run(ctx, input, wire, func(at time.Time) {
		select {
		case progress <- at:
		default:
		}
	})
	close(stopProgress)
	progressError := <-progressDone
	if prepared.Cleanup != nil {
		defer prepared.Cleanup()
	}
	if err != nil {
		return Outcome{}, ErrRunner
	}
	if progressError != nil {
		prepared.Submission.Report.EvidenceErrors = appendProblem(prepared.Submission.Report.EvidenceErrors, "Worker start progress was not acknowledged.")
	}
	if ctx.Err() != nil {
		return Outcome{}, ErrDelivery
	}
	submission := prepared.Submission
	err = s.retry(ctx, func() error { return s.Transport.Submit(ctx, submission) })
	if errors.Is(err, workerprotocol.ErrQuota) {
		submission.Report.Attachments = nil
		submission.Files = nil
		submission.Report.EvidenceErrors = appendProblem(submission.Report.EvidenceErrors, "Hosted worker evidence omitted because private staging quota is exhausted.")
		prepared.Files = nil
		err = s.retry(ctx, func() error { return s.Transport.Submit(ctx, submission) })
	}
	if err != nil {
		return Outcome{}, ErrDelivery
	}
	outcome := Outcome{ReportAccepted: true, ProcessExitCode: submission.ProcessExitCode, EvidenceIncomplete: len(submission.Report.EvidenceErrors) > 0}
	for _, file := range prepared.Files {
		err = s.retry(ctx, func() error {
			reader, err := file.Open()
			if err != nil {
				return workerprotocol.ErrInvalid
			}
			defer reader.Close()
			return s.Transport.Upload(ctx, file.Descriptor.Path, io.LimitReader(reader, file.Descriptor.Size+1))
		})
		if err != nil {
			outcome.PendingAttachments++
		}
	}
	outcome.EvidenceIncomplete = outcome.EvidenceIncomplete || outcome.PendingAttachments > 0
	return outcome, nil
}

func appendProblem(problems []string, message string) []string {
	return append(problems[:min(len(problems), 19)], message)
}
