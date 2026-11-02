package evidence

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/wtiger001/teststudio/internal/artifacts"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/storage"
)

var ErrQuota = errors.New("run evidence exceeds its configured quota")

type Service struct {
	Store *storage.Store
	Blobs artifacts.Store
}

func Bucket(run string) string  { return "artifacts/" + run }
func blobKey(a Artifact) string { return a.Project + "/" + a.Run + "/" + a.ID + "/" + a.Checksum }

func (s *Service) SaveBytes(ctx context.Context, a Artifact, data []byte, limit int64) (Artifact, error) {
	if a.ID == "" {
		a.ID = domain.NewID()
	}
	if !domain.ValidID(a.ID) || !domain.ValidID(a.Project) || !domain.ValidID(a.Run) {
		return a, errors.New("artifact ownership identifiers are invalid")
	}
	if a.Filename == "" || len(a.Filename) > 255 || strings.ContainsAny(a.Filename, "/\\\x00\r\n") {
		return a, errors.New("artifact filename is invalid")
	}
	if len(data) > 64<<20 || limit < 1 || limit > 1<<30 {
		return a, ErrQuota
	}
	hash := sha256.Sum256(data)
	a.Checksum = hex.EncodeToString(hash[:])
	a.Size = int64(len(data))
	a.State = "pending"
	a.CreatedAt = time.Now().UTC()
	if a.ExpiresAt.IsZero() {
		a.ExpiresAt = a.CreatedAt.Add(30 * 24 * time.Hour)
	}
	row, existing, err := s.admit(ctx, a, limit)
	if err != nil {
		return a, err
	}
	a = existing
	if a.State == "available" {
		return a, nil
	}
	object, uploadErr := s.Blobs.Put(ctx, blobKey(a), bytes.NewReader(data), max(a.Size, 1))
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
		change, err := storage.Put(row.Bucket, row.ID, a.Project, row.Revision, a)
		if err != nil {
			return a, err
		}
		if err := s.Store.Commit(ctx, change); err == nil {
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
	}
	return a, storage.ErrConflict
}

func (s *Service) admit(ctx context.Context, a Artifact, limit int64) (storage.Record, Artifact, error) {
	for range 8 {
		row, err := s.Store.Get(ctx, Bucket(a.Run), a.ID)
		if err == nil {
			old, err := storage.Decode[Artifact](row)
			if err != nil {
				return row, a, err
			}
			if old.Project != a.Project || old.Run != a.Run || old.Checksum != a.Checksum || old.Size != a.Size || old.Kind != a.Kind || old.Test != a.Test || old.Attempt != a.Attempt || old.Worker != a.Worker || old.Sequence != a.Sequence || old.Stream != a.Stream || old.Filename != a.Filename || old.ContentType != a.ContentType {
				return row, a, storage.ErrConflict
			}
			if old.State == "expired" || old.State == "deleted" {
				return row, a, errors.New("artifact retention has ended")
			}
			return row, old, nil
		}
		if !errors.Is(err, storage.ErrNotFound) {
			return row, a, err
		}
		quotaRow, err := s.Store.Get(ctx, "artifact-quotas", a.Run)
		if err != nil && !errors.Is(err, storage.ErrNotFound) {
			return row, a, err
		}
		quota := Quota{Limit: limit}
		if quotaRow.Revision > 0 {
			quota, err = storage.Decode[Quota](quotaRow)
			if err != nil {
				return row, a, err
			}
			if quota.Limit != limit {
				return row, a, errors.New("capture quota differs from the run snapshot")
			}
		}
		if quota.Bytes+a.Size > quota.Limit {
			return row, a, ErrQuota
		}
		quota.Bytes += a.Size
		entry, err := storage.Put(Bucket(a.Run), a.ID, a.Project, 0, a)
		if err != nil {
			return row, a, err
		}
		reservation, err := storage.Put("artifact-quotas", a.Run, a.Project, quotaRow.Revision, quota)
		if err != nil {
			return row, a, err
		}
		if err := s.Store.Commit(ctx, entry, reservation); errors.Is(err, storage.ErrConflict) {
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
	if len(text) > 64<<10 || sequence < 1 {
		return Artifact{}, errors.New("invalid log segment")
	}
	hash := sha256.Sum256([]byte(fmt.Sprintf("%s:%s:%d", worker, stream, sequence)))
	a := Artifact{ID: "log-" + hex.EncodeToString(hash[:]), Project: project, Run: run, Test: test, Attempt: attempt, Worker: worker, Sequence: sequence, Stream: stream, Kind: "log", Filename: fmt.Sprintf("%s-%06d.log", stream, sequence), ContentType: "text/plain; charset=utf-8"}
	return s.SaveBytes(ctx, a, []byte(text), limit)
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
