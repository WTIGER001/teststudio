package workerexchange

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"time"

	"github.com/wtiger001/teststudio/internal/artifacts"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/workerstate"
)

func objectKey(record Record, file File) string {
	return "worker-exchange/" + record.Reference.Project + "/" + record.Reference.Run + "/" + record.Reference.ID + "/" + record.Nonce + "/" + file.SHA256 + "/" + file.Path
}

func selectedFile(record Record, name string) (File, bool) {
	if record.Submission != nil {
		for _, file := range record.Submission.Files {
			if file.Path == name {
				return file, true
			}
		}
	}
	return File{}, false
}

func (s *Service) Upload(ctx context.Context, id, token, instance, name string, reader io.Reader) error {
	if s.Blobs == nil || !plainName(name) {
		return ErrInvalid
	}
	record, _, _, err := s.authenticated(ctx, id, token, instance)
	if err != nil {
		return err
	}
	file, found := selectedFile(record, name)
	if !found || record.Instance != instance {
		return ErrInvalid
	}
	s.uploadOnce.Do(func() { s.uploads = make(chan struct{}, 4) })
	select {
	case s.uploads <- struct{}{}:
		defer func() { <-s.uploads }()
	case <-ctx.Done():
		return ErrUnavailable
	}
	staged, err := artifacts.Stage(ctx, reader, max(file.Size, 1))
	if err != nil {
		var fileError *os.PathError
		if errors.As(err, &fileError) || ctx.Err() != nil {
			return ErrUnavailable
		}
		return ErrInvalid
	}
	defer staged.Close()
	if staged.Object.Size != file.Size || staged.Object.Checksum != file.SHA256 {
		return ErrInvalid
	}
	var claim UploadLease
	reserved := false
	for range 8 {
		current, row, guards, err := s.authenticated(ctx, id, token, instance)
		if err != nil {
			return err
		}
		if current.Uploaded[name] {
			return nil
		}
		if lease, found := current.Uploads[name]; found && time.Now().Before(lease.Until) {
			return ErrConflict
		}
		if current.Uploads == nil {
			current.Uploads = map[string]UploadLease{}
		}
		claim = UploadLease{Claim: domain.NewID(), Until: time.Now().UTC().Add(35 * time.Second)}
		current.Uploads[name] = claim
		if err := s.save(ctx, row, current, guards); err == ErrConflict {
			continue
		} else if err != nil {
			return err
		}
		record, reserved = current, true
		break
	}
	if !reserved {
		return ErrConflict
	}
	writeCtx, cancel := context.WithDeadline(ctx, claim.Until.Add(-5*time.Second))
	stored, writeErr := s.Blobs.Put(writeCtx, objectKey(record, file), staged.File, max(file.Size, 1))
	cancel()
	valid := writeErr == nil && stored.Size == file.Size && stored.Checksum == file.SHA256
	finish, cancelFinish := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancelFinish()
	for range 8 {
		current, row, err := s.owned(finish, record.Reference)
		if err != nil {
			return err
		}
		if current.Uploads[name].Claim != claim.Claim {
			return ErrConflict
		}
		delete(current.Uploads, name)
		if valid {
			if current.Uploaded == nil {
				current.Uploaded = map[string]bool{}
			}
			current.Uploaded[name] = true
		}
		if err := s.save(finish, row, current, nil); err == ErrConflict {
			continue
		} else if err != nil {
			return err
		}
		if !valid {
			return ErrUnavailable
		}
		_, _, _, err = s.authenticated(ctx, id, token, instance)
		return err
	}
	return ErrConflict
}

func (s *Service) owned(ctx context.Context, ref execution.Reference) (Record, storage.Record, error) {
	row, err := s.Store.Get(ctx, Bucket, ref.ID)
	if err != nil {
		return Record{}, row, err
	}
	record, err := storage.Decode[Record](row)
	if err != nil {
		return record, row, ErrUnavailable
	}
	if record.Reference != ref {
		return record, row, ErrDenied
	}
	return record, row, nil
}

func (s *Service) Observation(ctx context.Context, ref execution.Reference, input execution.CodedInput, exitError error) (execution.CodedReport, *time.Time, bool, error) {
	record, _, err := s.owned(ctx, ref)
	if err != nil {
		return execution.CodedReport{}, nil, false, err
	}
	payload, err := execution.MarshalWorkItem(input)
	if err != nil || fingerprint(payload) != record.InputHash {
		return execution.CodedReport{}, record.StartedAt, false, ErrConflict
	}
	if record.Submission == nil {
		return execution.CodedReport{}, record.StartedAt, false, nil
	}
	data, err := json.Marshal(record.Submission.Report)
	if err != nil {
		return execution.CodedReport{}, record.StartedAt, false, ErrUnavailable
	}
	var report execution.CodedReport
	if json.Unmarshal(data, &report) != nil {
		return report, record.StartedAt, false, ErrUnavailable
	}
	if exitError == nil && (record.Submission.ProcessExitCode != 0 || record.Submission.InfrastructureFailure != "") {
		exitError = errors.New("recorded runner process failed")
	}
	if err := execution.ValidateCodedReport(input, report, exitError); err != nil {
		return report, record.StartedAt, false, ErrInvalid
	}
	complete := true
	accepted := []execution.Attachment{}
	for _, attachment := range report.Attachments {
		file, found := selectedFile(record, attachment.Path)
		if !found || s.Blobs == nil {
			return report, record.StartedAt, false, ErrUnavailable
		}
		reader, err := s.Blobs.Open(ctx, objectKey(record, file))
		if err != nil {
			complete = false
			continue
		}
		body, readErr := io.ReadAll(io.LimitReader(reader, file.Size+1))
		reader.Close()
		if readErr != nil || int64(len(body)) != file.Size || fingerprint(body) != file.SHA256 {
			complete = false
			continue
		}
		attachment.Data = body
		accepted = append(accepted, attachment)
	}
	report.Attachments = accepted
	if !complete {
		report.EvidenceErrors = append(report.EvidenceErrors[:min(len(report.EvidenceErrors), 19)], "Hosted worker attachments are incomplete; collection can retry without executing the test again.")
	}
	return execution.RedactAPIReport(input, report), record.StartedAt, complete, nil
}

func (s *Service) Revoke(ctx context.Context, ref execution.Reference) error {
	for range 8 {
		record, row, err := s.owned(ctx, ref)
		if err != nil {
			return err
		}
		if record.Revoked {
			return nil
		}
		record.Revoked = true
		if err := s.save(ctx, row, record, nil); err == ErrConflict {
			continue
		} else {
			return err
		}
	}
	return ErrConflict
}

func (s *Service) Cleanup(ctx context.Context, ref execution.Reference) error {
	entry, _, err := (workerstate.Ledger{Store: s.Store}).Get(ctx, ref.Run, ref.ID)
	if err != nil || entry.Reference != ref || !entry.ResultSaved || !entry.EvidenceComplete && !entry.EvidenceAbandoned {
		return ErrConflict
	}
	if err := s.Revoke(ctx, ref); errors.Is(err, storage.ErrNotFound) {
		return nil
	} else if err != nil {
		return err
	}
	record, _, err := s.owned(ctx, ref)
	if err != nil {
		return err
	}
	for _, lease := range record.Uploads {
		if time.Now().Before(lease.Until) {
			return ErrConflict
		}
	}
	if record.Submission != nil {
		if len(record.Submission.Files) > 0 && s.Blobs == nil {
			return ErrUnavailable
		}
		for _, file := range record.Submission.Files {
			if err := s.Blobs.Delete(ctx, objectKey(record, file)); err != nil {
				return ErrUnavailable
			}
		}
	}
	for range 8 {
		current, currentRow, err := s.owned(ctx, ref)
		if errors.Is(err, storage.ErrNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		changes, err := s.release(ctx, current)
		if err != nil {
			return err
		}
		changes = append(changes, storage.Change{Record: currentRow, Delete: true})
		if err := s.Store.Commit(ctx, changes...); errors.Is(err, storage.ErrConflict) {
			continue
		} else if err != nil {
			return ErrUnavailable
		}
		return nil
	}
	return ErrConflict
}
