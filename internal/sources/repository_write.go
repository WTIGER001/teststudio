package sources

import (
	"context"
	"errors"
	"time"

	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/sourceprovider"
	"github.com/wtiger001/teststudio/internal/storage"
)

type RepositoryWrite struct {
	MetadataRevision int64                       `json:"metadataRevision"`
	State            string                      `json:"state"`
	Result           *sourceprovider.WriteResult `json:"result,omitempty"`
	Error            string                      `json:"error,omitempty"`
	StartedAt        time.Time                   `json:"startedAt"`
}

func (s *Service) CommitRepositoryMetadata(ctx context.Context, user, project, source, suite, key string, revision int64) (RepositoryWrite, error) {
	registered, binding, sourceRow, err := s.repositoryBinding(ctx, user, project, source, suite, "manage")
	if err != nil {
		return RepositoryWrite{}, err
	}
	if registered.Configuration.Mode != "read-write" {
		return RepositoryWrite{}, errors.New("read-only connections never write to Git")
	}
	if err = s.Auth.Check(ctx, user, project, "suite/"+suite, "catalog.import"); err != nil {
		return RepositoryWrite{}, err
	}
	if !domain.ValidID(key) {
		return RepositoryWrite{}, errors.New("metadata commit requires a stable operation identifier")
	}
	writer, ok := s.Providers[registered.Configuration.Provider].(sourceprovider.Writer)
	if !ok {
		return RepositoryWrite{}, errors.New("provider metadata writes are unavailable")
	}
	metadataRow, err := s.ReadRepositoryMetadata(ctx, user, project, source, suite)
	if err != nil {
		return RepositoryWrite{}, err
	}
	if metadataRow.Revision != revision {
		return RepositoryWrite{}, storage.ErrConflict
	}
	m, err := storage.Decode[RepositoryMetadata](metadataRow)
	if err != nil {
		return RepositoryWrite{}, err
	}
	if m.Base == nil {
		return RepositoryWrite{}, errors.New("synchronize once to establish the metadata base commit")
	}
	files, err := metadataFiles(binding, m)
	if err != nil {
		return RepositoryWrite{}, err
	}
	bucket := "repository-writes/" + project + "/" + source
	id := digest(suite + "/" + key)[7:]
	op := RepositoryWrite{MetadataRevision: revision, State: "writing", StartedAt: time.Now().UTC()}
	opRevision := int64(0)
	if row, readErr := s.Store.Get(ctx, bucket, id); readErr == nil {
		op, err = storage.Decode[RepositoryWrite](row)
		if err != nil {
			return op, err
		}
		if op.MetadataRevision != revision {
			return op, storage.ErrConflict
		}
		if op.State == "committed" {
			return op, nil
		}
		if op.State == "writing" && time.Since(op.StartedAt) < 2*time.Minute {
			return op, errors.New("metadata commit is in progress; inspect it before retrying")
		}
		opRevision = row.Revision
		op.State = "writing"
		op.StartedAt = time.Now().UTC()
	} else if !errors.Is(readErr, storage.ErrNotFound) {
		return op, readErr
	}
	change, err := storage.Put(bucket, id, project, opRevision, op)
	if err != nil {
		return op, err
	}
	err = s.Auth.WithAssignmentTransaction(ctx, func(auth *authorization.Service) error {
		bound := &Service{Store: auth.Directory.Store, Auth: auth}
		if _, _, _, err := bound.repositoryBinding(ctx, user, project, source, suite, "manage"); err != nil {
			return err
		}
		if err := auth.Check(ctx, user, project, "suite/"+suite, "catalog.import"); err != nil {
			return err
		}
		return bound.Store.Commit(ctx, change, storage.Change{Record: metadataRow, Assert: true}, storage.Change{Record: sourceRow, Assert: true})
	})
	if err != nil {
		return op, err
	}
	access := sourceprovider.Access{Project: project, Repository: registered.Configuration.Repository, RepositoryID: registered.RepositoryID, Credential: registered.Configuration.Credential, Write: true}
	result, writeErr := writer.WriteMetadata(ctx, access, sourceprovider.WriteRequest{Base: *m.Base, Key: id, Message: "Update TestStudio metadata for " + suite, Files: files})
	op.State = "committed"
	op.Result = &result
	op.Error = ""
	if writeErr != nil {
		op.State = "uncertain"
		op.Result = nil
		op.Error = "The remote write is not confirmed. Retry this operation to reconcile its review branch; pending edits remain saved."
	}
	checkpoint, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	row, err := s.Store.Get(checkpoint, bucket, id)
	if err != nil {
		return op, err
	}
	existing, err := storage.Decode[RepositoryWrite](row)
	if err != nil {
		return op, err
	}
	if existing.State == "committed" {
		return existing, nil
	}
	saved, err := storage.Put(bucket, id, project, row.Revision, op)
	if err != nil {
		return op, err
	}
	audit, err := domain.Audit(user, project, "sources.repository.metadata.commit", source+"/"+suite, nil, op)
	if err != nil {
		return op, err
	}
	if err = s.Store.Commit(checkpoint, saved, audit); err != nil {
		return op, err
	}
	return op, nil
}
