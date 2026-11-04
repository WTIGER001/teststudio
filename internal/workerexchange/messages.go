package workerexchange

import (
	"context"
	"encoding/json"
	"errors"
	"mime"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/storage"
)

var checksum = regexp.MustCompile(`^[a-f0-9]{64}$`)

func plainName(name string) bool {
	return name != "" && name != "." && name != ".." && len(name) <= 255 && path.Base(name) == name && !strings.ContainsAny(name, "\\\x00\r\n")
}

func (s *Service) Input(ctx context.Context, id, token, instance string) (Packet, error) {
	for range 8 {
		record, row, guards, err := s.authenticated(ctx, id, token, instance)
		if err != nil {
			return Packet{}, err
		}
		if record.Submission != nil {
			return Packet{}, ErrConflict
		}
		if record.Instance == "" {
			record.Instance = instance
			if err := s.save(ctx, row, record, guards); err == ErrConflict {
				continue
			} else if err != nil {
				return Packet{}, err
			}
		}
		return Packet{Execution: id, Deadline: record.Deadline, Input: record.Input}, nil
	}
	return Packet{}, ErrConflict
}

func (s *Service) Started(ctx context.Context, id, token, instance string, at time.Time) error {
	for range 8 {
		record, row, guards, err := s.authenticated(ctx, id, token, instance)
		if err != nil {
			return err
		}
		if record.Instance != instance || at.IsZero() || at.Before(record.CreatedAt.Add(-time.Minute)) || at.After(time.Now().Add(time.Minute)) {
			return ErrInvalid
		}
		if record.StartedAt != nil {
			if record.StartedAt.Equal(at) {
				return nil
			}
			return ErrConflict
		}
		if record.Submission != nil {
			return ErrConflict
		}
		record.StartedAt = &at
		if err := s.save(ctx, row, record, guards); err == ErrConflict {
			continue
		} else {
			return err
		}
	}
	return ErrConflict
}

func validateSubmission(input execution.CodedInput, submission Submission) error {
	data, err := json.Marshal(submission.Report)
	if err != nil || len(data) > 2<<20 || len(submission.Files) != len(submission.Report.Attachments) || len(submission.Files) > 32 {
		return ErrInvalid
	}
	if submission.ProcessExitCode < -1 || submission.ProcessExitCode > 255 {
		return ErrInvalid
	}
	var processError error
	if submission.ProcessExitCode != 0 {
		processError = errors.New("runner process failed")
	}
	if submission.InfrastructureFailure != "" {
		if submission.Report.Schema != "teststudio.worker/v1" || submission.InfrastructureFailure != submission.Report.Result.FailureKind {
			return ErrInvalid
		}
		processError = errors.New("runner infrastructure failed")
	}
	if err := execution.ValidateCodedReport(input, submission.Report, processError); err != nil {
		return ErrInvalid
	}
	files := map[string]File{}
	var total int64
	for _, file := range submission.Files {
		if !plainName(file.Path) || !checksum.MatchString(file.SHA256) || file.Size < 0 || file.Size > min(input.Capture.MaxFileBytes, 64<<20) {
			return ErrInvalid
		}
		if _, exists := files[file.Path]; exists {
			return ErrInvalid
		}
		files[file.Path] = file
		total += file.Size
		if total > input.Capture.MaxBytes {
			return ErrInvalid
		}
	}
	seen := map[string]bool{}
	for _, attachment := range submission.Report.Attachments {
		if _, found := files[attachment.Path]; !found || seen[attachment.Path] || !plainName(attachment.Filename) || len(attachment.ContentType) > 255 || len(attachment.Stream) > 128 || strings.ContainsAny(attachment.ContentType+attachment.Stream, "\r\n\x00") {
			return ErrInvalid
		}
		if _, _, err := mime.ParseMediaType(attachment.ContentType); err != nil {
			return ErrInvalid
		}
		switch attachment.Kind {
		case "screenshot", "trace", "video", "attachment", "log":
		default:
			return ErrInvalid
		}
		if !execution.CaptureAttachment(input, submission.Report.Result, attachment) {
			return ErrInvalid
		}
		seen[attachment.Path] = true
	}
	return nil
}

func (s *Service) Submit(ctx context.Context, id, token, instance string, submission Submission) error {
	data, err := json.Marshal(submission)
	if err != nil || len(data) > MaxSubmissionBytes {
		return ErrInvalid
	}
	submission.Report.Result = execution.NormalizeTimeout(submission.Report.Result)
	for range 8 {
		record, row, _, err := s.authenticated(ctx, id, token, instance)
		if err != nil {
			return err
		}
		if record.Instance != instance {
			return ErrDenied
		}
		var input execution.CodedInput
		if json.Unmarshal(record.Input, &input) != nil {
			return ErrUnavailable
		}
		if err := validateSubmission(input, submission); err != nil {
			return err
		}
		canonical, err := json.Marshal(submission)
		if err != nil {
			return ErrInvalid
		}
		digest := fingerprint(canonical)
		if record.Submission != nil {
			if record.SubmissionHash == digest {
				return nil
			}
			return ErrConflict
		}
		access, err := s.Authorize(ctx, record.Reference)
		if err != nil {
			return ErrDenied
		}
		record.Submission, record.SubmissionHash = &submission, digest
		for _, file := range submission.Files {
			record.StagingBytes += file.Size
		}
		changes, err := s.reserve(ctx, record, access)
		if err != nil {
			return err
		}
		change, err := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, record)
		if err != nil {
			return ErrUnavailable
		}
		changes = append(changes, change)
		for _, guard := range access.Records {
			changes = append(changes, storage.Change{Record: guard, Assert: true})
		}
		if err := s.Store.Commit(ctx, changes...); err == storage.ErrConflict {
			continue
		} else if err != nil {
			return ErrUnavailable
		}
		return nil
	}
	return ErrConflict
}
