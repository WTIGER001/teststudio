package runs

import (
	"context"
	"github.com/wtiger001/teststudio/internal/capture"
	"github.com/wtiger001/teststudio/internal/evidence"
	"github.com/wtiger001/teststudio/internal/execution"
	"time"
)

func (s *Service) captureCodedReport(parent context.Context, run Run, worker string, result execution.Result, failedBefore bool, report execution.CodedReport, problems *[]string) {
	add := func(message string) {
		if len(*problems) < 20 {
			*problems = append(*problems, message)
		}
	}
	for _, message := range report.EvidenceErrors {
		add(message)
	}
	policy := capture.Normalize(run.TestCapture[result.TestID])
	base := capture.Normalize(run.Capture)
	failed := result.State == "failed" || result.State == "unknown" || failedBefore
	ctx, cancel := context.WithTimeout(context.WithoutCancel(parent), 10*time.Second)
	defer cancel()
	for _, file := range report.Attachments {
		if ctx.Err() != nil {
			add("Remaining coded evidence could not be saved before the capture deadline.")
			break
		}
		mode := "always"
		switch file.Kind {
		case "screenshot":
			mode = policy.Screenshots
		case "trace":
			mode = policy.Trace
		case "video":
			mode = policy.Video
		case "log":
			if !policy.Logs {
				continue
			}
		}
		if mode == "off" || (mode == "failure" && !failed) {
			continue
		}
		if s.Evidence == nil {
			add("Coded test evidence storage is not configured.")
			continue
		}
		artifact := evidence.Artifact{Project: run.Project, Run: run.ID, Test: result.TestID, Attempt: result.Attempt, Worker: worker, Kind: file.Kind, Filename: file.Filename, ContentType: file.ContentType, Policy: &policy}
		if file.Kind == "log" {
			artifact.Stream = file.Stream
			if artifact.Stream == "" {
				artifact.Stream = "playwright"
			}
			sequence, err := s.Store.Next(ctx, "coded-logs/"+run.ID+"/"+worker+"/"+artifact.Stream)
			if err != nil {
				add(err.Error())
				continue
			}
			artifact.Sequence = sequence
		}
		artifact.ExpiresAt = capture.Expiry(time.Now().UTC(), policy, file.Kind, result.State)
		_, err := s.Evidence.Save(ctx, artifact, file.Data, evidence.Budget{RunBytes: base.MaxBytes, TestBytes: policy.MaxBytes, FileBytes: min(base.MaxFileBytes, policy.MaxFileBytes, run.CaptureLimits.FileBytes), ProjectBytes: run.CaptureLimits.ProjectBytes})
		if err != nil {
			add(file.Filename + ": " + err.Error())
		}
	}
}
