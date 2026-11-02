package runs

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/wtiger001/teststudio/internal/capture"
	"github.com/wtiger001/teststudio/internal/evidence"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/storage"
	"time"
)

func (s *Service) captureCodedReport(parent context.Context, run Run, worker string, result execution.Result, failedBefore bool, report execution.CodedReport, problems *[]string) bool {
	retryable := false
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
	failed := result.State == "failed" || result.State == "timed-out" || result.State == "unknown" || failedBefore
	ctx, cancel := context.WithTimeout(context.WithoutCancel(parent), 10*time.Second)
	defer cancel()
	for index, file := range report.Attachments {
		if ctx.Err() != nil {
			retryable = true
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
			retryable = true
			add("Coded test evidence storage is not configured.")
			continue
		}
		hash := sha256.Sum256(file.Data)
		slot := ""
		if file.Path == "" && file.Stream == "" {
			slot = fmt.Sprint(index)
		}
		artifactID := "execution-artifact-" + fingerprint([]any{run.ID, worker, result.TestID, result.Attempt, result.ExecutionID, slot, file.Kind, file.Path, file.Filename, file.Stream, hex.EncodeToString(hash[:])})
		artifact := evidence.Artifact{ID: artifactID, Project: run.Project, Run: run.ID, Test: result.TestID, Attempt: result.Attempt, Worker: worker, Kind: file.Kind, Filename: file.Filename, ContentType: file.ContentType, Policy: &policy}
		if file.Kind == "log" {
			artifact.Stream = file.Stream
			if artifact.Stream == "" {
				artifact.Stream = "playwright"
			}
			sequence, err := s.codedLogSequence(ctx, run.Project, run.ID, worker, artifact.Stream, artifact.ID)
			if err != nil {
				retryable = true
				add(err.Error())
				continue
			}
			artifact.Sequence = sequence
		}
		artifact.ExpiresAt = capture.Expiry(time.Now().UTC(), policy, file.Kind, result.State)
		_, err := s.Evidence.Save(ctx, artifact, file.Data, evidence.Budget{RunBytes: base.MaxBytes, TestBytes: policy.MaxBytes, FileBytes: min(base.MaxFileBytes, policy.MaxFileBytes, run.CaptureLimits.FileBytes), ProjectBytes: run.CaptureLimits.ProjectBytes})
		if err != nil {
			retryable = true
			add(file.Filename + ": " + err.Error())
		}
	}
	return retryable
}

func (s *Service) codedLogSequence(ctx context.Context, project, run, worker, stream, id string) (int64, error) {
	for range 8 {
		row, err := s.Store.Get(ctx, "coded-log-sequences/"+run, id)
		if err == nil {
			return storage.Decode[int64](row)
		}
		if !errors.Is(err, storage.ErrNotFound) {
			return 0, err
		}
		sequence, err := s.Store.Next(ctx, "coded-logs/"+run+"/"+worker+"/"+stream)
		if err != nil {
			return 0, err
		}
		change, err := storage.Put("coded-log-sequences/"+run, id, project, 0, sequence)
		if err != nil {
			return 0, err
		}
		if err := s.Store.Commit(ctx, change); errors.Is(err, storage.ErrConflict) {
			continue
		} else if err != nil {
			return 0, err
		}
		return sequence, nil
	}
	return 0, storage.ErrConflict
}
