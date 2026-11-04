package evidence

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"time"

	"github.com/wtiger001/teststudio/internal/artifacts"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/storage"
)

type uploadWork struct {
	Project  string `json:"projectId"`
	Run      string `json:"runId"`
	Artifact string `json:"artifactId"`
}

func (s *Service) uploadChanges(ctx context.Context, a Artifact) ([]storage.Change, error) {
	row, err := s.Store.Get(ctx, "artifact-upload-work", expirationID(a))
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return nil, err
	}
	if row.Revision > 0 {
		work, err := storage.Decode[uploadWork](row)
		if err != nil {
			return nil, err
		}
		if work.Project != a.Project || work.Run != a.Run || work.Artifact != a.ID {
			return nil, storage.ErrConflict
		}
		return []storage.Change{{Record: row, Assert: a.State == "pending", Delete: a.State != "pending"}}, nil
	}
	if a.State != "pending" {
		return nil, nil
	}
	change, err := storage.Put("artifact-upload-work", expirationID(a), a.Project, 0, uploadWork{a.Project, a.Run, a.ID})
	return []storage.Change{change}, err
}

func (s *Service) RecoverUploads(ctx context.Context, limit int) error {
	s.maintenanceMu.Lock()
	defer s.maintenanceMu.Unlock()
	page, err := s.Store.List(ctx, "artifact-upload-work", "", s.uploadCursor, limit)
	if err != nil {
		return err
	}
	s.uploadCursor = page.Cursor
	var failures []error
	for _, row := range page.Items {
		if err := s.recoverUpload(ctx, row); err != nil && !errors.Is(err, storage.ErrConflict) {
			failures = append(failures, err)
		}
	}
	return errors.Join(failures...)
}

func (s *Service) recoverUpload(parent context.Context, workRow storage.Record) error {
	work, err := storage.Decode[uploadWork](workRow)
	if err != nil {
		return err
	}
	row, err := s.Store.Get(parent, Bucket(work.Run), work.Artifact)
	if errors.Is(err, storage.ErrNotFound) {
		return s.Store.Commit(parent, storage.Change{Record: workRow, Delete: true})
	}
	if err != nil {
		return err
	}
	a, err := storage.Decode[Artifact](row)
	if err != nil {
		return err
	}
	if work.Project != workRow.Project || a.Project != work.Project || a.Run != work.Run || a.ID != work.Artifact || expirationID(a) != workRow.ID {
		return storage.ErrConflict
	}
	if a.State != "pending" {
		return s.Store.Commit(parent, storage.Change{Record: row, Assert: true}, storage.Change{Record: workRow, Delete: true})
	}
	if a.UploadUntil != nil && time.Now().Before(*a.UploadUntil) {
		return nil
	}
	a.UploadToken = domain.NewID()
	until := time.Now().UTC().Add(30 * time.Second)
	a.UploadUntil = &until
	change, err := storage.Put(row.Bucket, row.ID, a.Project, row.Revision, a)
	if err != nil {
		return err
	}
	if err := s.Store.Commit(parent, change, storage.Change{Record: workRow, Assert: true}); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(parent, 25*time.Second)
	defer cancel()
	state, message, verifyErr := s.verifyUpload(ctx, a)
	for range 8 {
		row, err := s.Store.Get(parent, Bucket(a.Run), a.ID)
		if err != nil {
			return err
		}
		current, err := storage.Decode[Artifact](row)
		if err != nil {
			return err
		}
		if current.State != "pending" || current.UploadToken != a.UploadToken {
			return storage.ErrConflict
		}
		current.State, current.Error = state, message
		if verifyErr == nil {
			current.UploadUntil = nil
		}
		change, err := storage.Put(row.Bucket, row.ID, a.Project, row.Revision, current)
		if err != nil {
			return err
		}
		changes, err := s.uploadChanges(parent, current)
		if err != nil {
			return err
		}
		if err := s.Store.Commit(parent, append(changes, change)...); errors.Is(err, storage.ErrConflict) {
			continue
		} else if err != nil {
			return err
		}
		return verifyErr
	}
	return storage.ErrConflict
}

func (s *Service) verifyUpload(ctx context.Context, a Artifact) (string, string, error) {
	if a.Size < 0 || a.Size > 64<<20 || len(a.Checksum) != 64 {
		return "upload-failed", "Interrupted upload has invalid size or checksum metadata.", nil
	}
	reader, err := s.Blobs.Open(ctx, blobKey(a))
	if errors.Is(err, artifacts.ErrNotFound) {
		return "upload-failed", "Upload was interrupted before content became available. The source worker can retry this attachment.", nil
	}
	if err != nil {
		return "pending", "Upload recovery cannot read evidence storage. Recovery will retry.", err
	}
	hash := sha256.New()
	size, readErr := io.Copy(hash, io.LimitReader(reader, a.Size+1))
	closeErr := reader.Close()
	if err := errors.Join(readErr, closeErr, ctx.Err()); err != nil {
		return "pending", "Upload recovery could not verify content. Recovery will retry.", err
	}
	if size != a.Size || hex.EncodeToString(hash.Sum(nil)) != a.Checksum {
		return "upload-failed", "Interrupted upload content does not match its recorded size or checksum. The source worker can retry this attachment.", nil
	}
	return "available", "", nil
}
