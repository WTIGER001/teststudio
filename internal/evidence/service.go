package evidence

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/wtiger001/teststudio/internal/artifacts"
	"github.com/wtiger001/teststudio/internal/capture"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/storage"
)

var ErrQuota = errors.New("run evidence exceeds its configured quota")

type Service struct {
	Store           *storage.Store
	Blobs           artifacts.Store
	maintenanceMu   sync.Mutex
	retentionCursor string
	indexCursor     string
	cleanupCursor   string
	uploadCursor    string
}

func Bucket(run string) string  { return "artifacts/" + run }
func blobKey(a Artifact) string { return a.Project + "/" + a.Run + "/" + a.ID + "/" + a.Checksum }

func (s *Service) SaveBytes(ctx context.Context, a Artifact, data []byte, limit int64) (Artifact, error) {
	return s.Save(ctx, a, data, Budget{RunBytes: limit, TestBytes: limit, FileBytes: min(64<<20, limit), ProjectBytes: capture.DefaultLimits().ProjectBytes})
}

func (s *Service) Save(ctx context.Context, a Artifact, data []byte, budget Budget) (Artifact, error) {
	if a.ID == "" {
		a.ID = domain.NewID()
	}
	if !domain.ValidID(a.ID) || !domain.ValidID(a.Project) || !domain.ValidID(a.Run) {
		return a, errors.New("artifact ownership identifiers are invalid")
	}
	if a.Filename == "" || len(a.Filename) > 255 || strings.ContainsAny(a.Filename, "/\\\x00\r\n") {
		return a, errors.New("artifact filename is invalid")
	}
	if budget.RunBytes < 1 || budget.RunBytes > 1<<30 || budget.TestBytes < 1 || budget.TestBytes > 1<<30 || budget.FileBytes < 1 || budget.FileBytes > 64<<20 || budget.ProjectBytes < 1 || int64(len(data)) > budget.FileBytes {
		return a, ErrQuota
	}
	hash := sha256.Sum256(data)
	a.Checksum = hex.EncodeToString(hash[:])
	a.Size = int64(len(data))
	a.State = "pending"
	a.CreatedAt = time.Now().UTC()
	uploadToken := domain.NewID()
	until := time.Now().UTC().Add(30 * time.Second)
	a.UploadToken = uploadToken
	a.UploadUntil = &until
	if a.ExpiresAt.IsZero() {
		a.ExpiresAt = a.CreatedAt.Add(30 * 24 * time.Hour)
	}
	row, existing, err := s.admit(ctx, a, budget)
	if err != nil {
		return a, err
	}
	a = existing
	if a.State == "available" {
		return a, nil
	}
	if a.UploadToken != uploadToken {
		return a, nil
	}
	uploadContext, cancelUpload := context.WithTimeout(ctx, 25*time.Second)
	defer cancelUpload()
	object, uploadErr := s.Blobs.Put(uploadContext, blobKey(a), bytes.NewReader(data), max(a.Size, 1))
	if uploadErr == nil {
		uploadErr = uploadContext.Err()
	}
	if uploadErr == nil && (object.Size != a.Size || object.Checksum != a.Checksum) {
		uploadErr = errors.New("artifact size or checksum verification failed")
	}
	if uploadErr != nil {
		a.State = "upload-failed"
		a.Error = uploadErr.Error()
	} else {
		a.State = "available"
		a.Error = ""
	}
	for range 4 {
		a.UploadUntil = nil
		change, err := storage.Put(row.Bucket, row.ID, a.Project, row.Revision, a)
		if err != nil {
			return a, err
		}
		changes, err := s.uploadChanges(ctx, a)
		if err != nil {
			return a, err
		}
		if err := s.Store.Commit(ctx, append(changes, change)...); err == nil {
			return a, uploadErr
		} else if !errors.Is(err, storage.ErrConflict) {
			return a, err
		}
		row, err = s.Store.Get(ctx, row.Bucket, row.ID)
		if err != nil {
			return a, err
		}
		current, err := storage.Decode[Artifact](row)
		if err != nil {
			return a, err
		}
		if current.State == "available" && current.Checksum == a.Checksum {
			return current, nil
		}
		if current.UploadToken != uploadToken || current.State != "pending" {
			if current.State == "expired" || current.State == "deleted" || current.State == "deleting" {
				cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
				_ = s.Blobs.Delete(cleanup, blobKey(current))
				cancel()
			}
			return current, ErrUnavailable
		}
		a.ExpiresAt = current.ExpiresAt
		a.QuotaVersion = current.QuotaVersion
		a.Policy = current.Policy
	}
	return a, storage.ErrConflict
}

func (s *Service) admit(ctx context.Context, a Artifact, budget Budget) (storage.Record, Artifact, error) {
	for range 8 {
		row, err := s.Store.Get(ctx, Bucket(a.Run), a.ID)
		if err == nil {
			old, err := storage.Decode[Artifact](row)
			if err != nil {
				return row, a, err
			}
			if old.Project != a.Project || old.Run != a.Run || old.Checksum != a.Checksum || old.Size != a.Size || old.Kind != a.Kind || old.Test != a.Test || old.Attempt != a.Attempt || old.Worker != a.Worker || old.Sequence != a.Sequence || old.Stream != a.Stream || old.Filename != a.Filename || old.ContentType != a.ContentType || old.Step != a.Step || old.RedactionPolicy != a.RedactionPolicy || !reflect.DeepEqual(old.Policy, a.Policy) {
				return row, a, storage.ErrConflict
			}
			if old.State == "expired" || old.State == "deleted" || old.State == "deleting" || old.State == "delete-pending" || old.State == "cleanup-failed" {
				return row, a, errors.New("artifact retention has ended")
			}
			if old.State == "available" || (old.State == "pending" && old.UploadUntil != nil && time.Now().Before(*old.UploadUntil)) {
				return row, old, nil
			}
			old.State = "pending"
			old.UploadToken = a.UploadToken
			old.UploadUntil = a.UploadUntil
			old.Error = ""
			change, err := storage.Put(row.Bucket, row.ID, old.Project, row.Revision, old)
			if err != nil {
				return row, old, err
			}
			changes, err := s.uploadChanges(ctx, old)
			if err != nil {
				return row, old, err
			}
			if err := s.Store.Commit(ctx, append(changes, change)...); errors.Is(err, storage.ErrConflict) {
				continue
			} else if err != nil {
				return row, old, err
			}
			row, err = s.Store.Get(ctx, row.Bucket, row.ID)
			return row, old, err
		}
		if !errors.Is(err, storage.ErrNotFound) {
			return row, a, err
		}
		reservations, err := s.reservations(ctx, a, budget)
		if err != nil {
			return row, a, err
		}
		a.QuotaVersion = 1
		entry, err := storage.Put(Bucket(a.Run), a.ID, a.Project, 0, a)
		if err != nil {
			return row, a, err
		}
		index, err := s.expirationChange(ctx, a)
		if err != nil {
			return row, a, err
		}
		uploads, err := s.uploadChanges(ctx, a)
		if err != nil {
			return row, a, err
		}
		changes := append(reservations, entry, index)
		if err := s.Store.Commit(ctx, append(changes, uploads...)...); errors.Is(err, storage.ErrConflict) {
			continue
		} else if err != nil {
			return row, a, err
		}
		row, err = s.Store.Get(ctx, Bucket(a.Run), a.ID)
		return row, a, err
	}
	return storage.Record{}, a, storage.ErrConflict
}

func (s *Service) Log(ctx context.Context, project, run, worker, stream, test string, attempt int, sequence int64, text string, limit int64) (Artifact, error) {
	policy := capture.DefaultPolicy()
	policy.MaxBytes = limit
	policy.MaxFileBytes = min(limit, 64<<20)
	return s.LogSegment(ctx, LogInput{Project: project, Run: run, Worker: worker, Stream: stream, Test: test, Attempt: attempt, Sequence: sequence, Text: text, Policy: policy, Budget: Budget{RunBytes: limit, TestBytes: limit, FileBytes: min(limit, 64<<20), ProjectBytes: capture.DefaultLimits().ProjectBytes}})
}

func (s *Service) LogSegment(ctx context.Context, input LogInput) (Artifact, error) {
	if len(input.Text) > 64<<10 || input.Sequence < 1 || !domain.ValidID(input.Stream) || !domain.ValidID(input.Worker) {
		return Artifact{}, errors.New("invalid log segment")
	}
	policy := capture.Normalize(input.Policy)
	if err := capture.Validate(policy, capture.DefaultLimits()); err != nil {
		return Artifact{}, err
	}
	hash := sha256.Sum256([]byte(fmt.Sprintf("%s:%s:%d", input.Worker, input.Stream, input.Sequence)))
	a := Artifact{ID: "log-" + hex.EncodeToString(hash[:]), Project: input.Project, Run: input.Run, Test: input.Test, Attempt: input.Attempt, Worker: input.Worker, Sequence: input.Sequence, Stream: input.Stream, Kind: "log", Filename: fmt.Sprintf("%s-%06d.log", input.Stream, input.Sequence), ContentType: "text/plain; charset=utf-8", Policy: &policy}
	a.ExpiresAt = capture.Expiry(time.Now().UTC(), policy, "log", "")
	return s.Save(ctx, a, []byte(input.Text), input.Budget)
}

func (s *Service) Event(ctx context.Context, e Event) error {
	if !domain.ValidID(e.Run) || !domain.ValidID(e.Project) || len(e.Message) > 2048 {
		return errors.New("invalid execution event")
	}
	sequence, err := s.Store.Next(ctx, "events/"+e.Run)
	if err != nil {
		return err
	}
	e.Sequence = sequence
	e.ID = fmt.Sprintf("%020d", sequence)
	e.At = time.Now().UTC()
	c, err := storage.Put("events/"+e.Run, e.ID, e.Project, 0, e)
	if err != nil {
		return err
	}
	return s.Store.Commit(ctx, c)
}

func (s *Service) Open(ctx context.Context, project, run, id string) (Artifact, io.ReadCloser, error) {
	r, err := s.Store.Get(ctx, Bucket(run), id)
	if err != nil {
		return Artifact{}, nil, err
	}
	a, err := storage.Decode[Artifact](r)
	if err != nil {
		return a, nil, err
	}
	if a.Project != project || a.Run != run {
		return a, nil, storage.ErrNotFound
	}
	if a.State != "available" {
		return a, nil, errors.New("artifact is not available")
	}
	available, err := s.Available(ctx, a, time.Now())
	if err != nil {
		return a, nil, err
	}
	if !available {
		a.State = "expired"
		return a, nil, ErrUnavailable
	}
	reader, err := s.Blobs.Open(ctx, blobKey(a))
	if errors.Is(err, artifacts.ErrNotFound) {
		a.State = "missing"
		a.Error = "Artifact content is missing from storage."
		change, encodeErr := storage.Put(r.Bucket, r.ID, a.Project, r.Revision, a)
		if encodeErr != nil {
			return a, nil, encodeErr
		}
		if updateErr := s.Store.Commit(ctx, change); updateErr != nil {
			return a, nil, updateErr
		}
	}
	return a, reader, err
}
