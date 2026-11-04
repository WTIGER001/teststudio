package hostedexecution

import (
	"context"
	"errors"
	"time"

	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/workerexchange"
)

func unknown(input execution.CodedInput, kind string, started time.Time) execution.CodedReport {
	state := "unknown"
	if kind == "timeout" {
		state = "timed-out"
	}
	return execution.CodedReport{TestID: input.Test.ID, Selector: input.Test.Selector, Result: execution.Result{TestID: input.Test.ID, State: state, FailureKind: kind, Error: "A complete hosted test observation is unavailable.", StartedAt: started, FinishedAt: time.Now().UTC(), DurationUnavailable: true}, EvidenceErrors: []string{"Hosted execution has an explicit observation gap; no test duration is inferred from provider status."}}
}

func (e *Executor) Execute(ctx context.Context, input execution.CodedInput) (report execution.CodedReport) {
	started := time.Now().UTC()
	if input.Reference == nil || !e.validReference(*input.Reference) || input.RunID != input.Reference.Run || input.Test.ID != input.Reference.Test || input.BeforeDispatch == nil || input.BeforeCleanup == nil || input.AfterCleanup == nil || !e.configured() {
		return unknown(input, "configuration", started)
	}
	defer func() { e.finalize(ctx, input, &report) }()
	if err := e.Admit(ctx, input.Test, input.Artifact); err != nil {
		return unknown(input, "configuration", started)
	}
	if input.Artifact.SourceProject != "" && input.Artifact.SourceProject != input.Reference.Project {
		return unknown(input, "configuration", started)
	}
	if _, err := execution.MarshalWorkItem(input); err != nil {
		return unknown(input, "configuration", started)
	}
	if err := input.BeforeDispatch(); err != nil {
		return unknown(input, "worker-lease", started)
	}
	deadline := started.Add(time.Duration(input.TimeoutMs)*time.Millisecond + 60*time.Second)
	if at, ok := ctx.Deadline(); ok && at.Before(deadline) {
		deadline = at
	}
	var ticket workerexchange.Ticket
	input.AwaitDispatch = true
	err := bounded(ctx, func(ctx context.Context) error {
		var err error
		ticket, err = e.Exchange.Prepare(ctx, *input.Reference, input, deadline)
		return err
	})
	if err != nil {
		report = unknown(input, "worker", started)
		report.RecoveryError = ErrUnavailable
		return report
	}
	if ticket.Execution != input.Reference.ID || len(ticket.Token) != 43 || !ticket.Deadline.After(time.Now()) || ticket.Deadline.After(deadline) {
		report = unknown(input, "configuration", started)
		report.RecoveryError = ErrOwnership
		return report
	}
	wall, cancel := context.WithDeadline(ctx, ticket.Deadline)
	defer cancel()
	if err := input.BeforeDispatch(); err != nil {
		return unknown(input, "worker-lease", started)
	}
	job := Job{Reference: *input.Reference, Image: input.Artifact.Image, Runner: e.Runner, ExchangeOrigin: e.ExchangeOrigin, Ticket: ticket}
	ticketErr := bounded(wall, func(ctx context.Context) error { return e.Provider.Ensure(ctx, job) })
	if ticketErr != nil {
		report = unknown(input, "worker", started)
		if !errors.Is(ticketErr, ErrRejected) {
			report.RecoveryError = ErrUnavailable
		}
		return report
	}
	status, waitErr := e.awaitAdmission(wall, *input.Reference, input.Artifact.Image, ticket.Deadline)
	if waitErr == nil && !terminal(status) {
		waitErr = bounded(wall, func(ctx context.Context) error { return e.Exchange.Activate(ctx, *input.Reference) })
		if waitErr == nil {
			status, waitErr = e.wait(wall, *input.Reference, input, ticket.Deadline)
		}
	}
	kind := "worker-lost"
	finish, release := context.WithTimeout(context.WithoutCancel(ctx), 20*time.Second)
	defer release()
	if err := e.revoke(finish, *input.Reference); err != nil {
		report = unknown(input, "worker", started)
		report.RecoveryError = err
		return report
	}
	if waitErr != nil && !errors.Is(waitErr, ErrNotFound) {
		if errors.Is(wall.Err(), context.DeadlineExceeded) {
			kind = "timeout"
		} else if errors.Is(wall.Err(), context.Canceled) {
			kind = "cancelled"
		}
		status, waitErr = e.quiesce(finish, *input.Reference, input.Artifact.Image, ticket.Deadline)
	}
	if waitErr != nil && !errors.Is(waitErr, ErrNotFound) {
		report = unknown(input, kind, started)
		report.RecoveryError = waitErr
		return report
	}
	return e.collect(finish, *input.Reference, input, status, waitErr, kind, started)
}

func (e *Executor) finalize(ctx context.Context, input execution.CodedInput, report *execution.CodedReport) {
	finish, cancel := context.WithTimeout(context.WithoutCancel(ctx), 20*time.Second)
	defer cancel()
	if err := e.revoke(finish, *input.Reference); err != nil {
		report.RecoveryError = err
		report.EvidenceErrors = appendGap(report.EvidenceErrors, "Worker ticket revocation remains pending; resources remain for recovery.")
		return
	}
	*report = execution.RedactAPIReport(input, *report)
	if report.RecoveryError != nil {
		return
	}
	if err := input.BeforeCleanup(*report); err != nil {
		report.EvidenceErrors = appendGap(report.EvidenceErrors, "The result or evidence checkpoint remains pending; resources remain for recovery.")
		return
	}
	cleanupErr := e.Cleanup(finish, *input.Reference)
	if err := input.AfterCleanup(cleanupErr); err != nil {
		report.EvidenceErrors = appendGap(report.EvidenceErrors, "Worker cleanup metadata remains pending for recovery.")
	}
	if cleanupErr != nil {
		report.EvidenceErrors = appendGap(report.EvidenceErrors, "Hosted resource cleanup remains pending for recovery.")
	}
}

func (e *Executor) Recover(ctx context.Context, ref execution.Reference, input execution.CodedInput) execution.CodedReport {
	started := time.Now().UTC()
	fail := func(err error) execution.CodedReport {
		report := unknown(input, "worker-lost", started)
		report.RecoveryError = err
		return report
	}
	if !e.configured() || !e.validReference(ref) || input.RunID != ref.Run || input.Test.ID != ref.Test || !distribution.MatchString(input.Artifact.Image) || input.Artifact.Browser != nil {
		return fail(ErrOwnership)
	}
	if err := e.revoke(ctx, ref); err != nil {
		return fail(err)
	}
	state, err := e.state(ctx, ref)
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return fail(ErrUnavailable)
	}
	status, controlErr := e.quiesce(ctx, ref, input.Artifact.Image, state.Deadline)
	if controlErr != nil && !errors.Is(controlErr, ErrNotFound) {
		return fail(controlErr)
	}
	if errors.Is(err, storage.ErrNotFound) {
		return unknown(input, "worker-lost", started)
	}
	return e.collect(ctx, ref, input, status, controlErr, "worker-lost", started)
}

func (e *Executor) collect(ctx context.Context, ref execution.Reference, input execution.CodedInput, status Status, controlErr error, kind string, started time.Time) execution.CodedReport {
	report, at, _, err := e.Exchange.Observation(ctx, ref, input, nil)
	if err != nil {
		gap := unknown(input, kind, started)
		gap.RecoveryError = ErrUnavailable
		return gap
	}
	if at != nil {
		started = *at
	}
	if report.Schema == "" {
		return unknown(input, kind, started)
	}
	providerFailed := controlErr != nil || status.Phase != "succeeded" || status.ExitCode == nil || *status.ExitCode != 0
	if providerFailed {
		if err := execution.ValidateCodedReport(input, report, ErrUnavailable); err != nil {
			gap := unknown(input, kind, started)
			gap.Attachments = report.Attachments
			gap.EvidencePending = report.EvidencePending
			gap.EvidenceErrors = appendGap(report.EvidenceErrors, "Provider completion cannot confirm the reported passing outcome.")
			return gap
		}
	}
	return report
}

func appendGap(problems []string, message string) []string {
	return append(problems[:min(len(problems), 19)], message)
}
