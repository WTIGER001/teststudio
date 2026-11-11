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

type RepositoryAccessCheck struct {
	LastSuccessAt       *time.Time                    `json:"lastSuccessAt,omitempty"`
	State               string                        `json:"state"`
	Message             string                        `json:"message"`
	CredentialReference string                        `json:"credentialReference"`
	Generation          int64                         `json:"generation"`
	Current             bool                          `json:"current"`
	CheckedAt           time.Time                     `json:"checkedAt"`
	Revision            sourceprovider.Revision       `json:"revision"`
	Credential          sourceprovider.CredentialInfo `json:"credential"`
}

func (s *Service) CheckRepositoryAccess(ctx context.Context, user, project, source, credential string) (RepositoryAccessCheck, error) {
	row, err := s.ReadRepository(ctx, user, project, source)
	if err != nil {
		return RepositoryAccessCheck{}, err
	}
	registered, err := storage.Decode[RepositorySource](row)
	if err != nil {
		return RepositoryAccessCheck{}, err
	}
	if err = s.checkRepository(ctx, user, project, source, "manage"); err != nil {
		return RepositoryAccessCheck{}, err
	}
	config := registered.Configuration
	if credential == "" {
		credential = config.Credential
	}
	if credential != "" {
		if s.Secrets == nil {
			return RepositoryAccessCheck{}, errors.New("credential provider is unavailable")
		}
		refs, err := s.Secrets.References(ctx, project, sourceprovider.CredentialPurpose)
		if err != nil {
			return RepositoryAccessCheck{}, err
		}
		found := false
		for _, ref := range refs {
			found = found || ref.ID == credential
		}
		if !found {
			return RepositoryAccessCheck{}, errors.New("replacement credential is not approved for this project")
		}
	}
	provider := s.Providers[config.Provider]
	if provider == nil {
		return RepositoryAccessCheck{}, sourceprovider.ErrUnavailable
	}
	access := sourceprovider.Access{Project: project, Repository: config.Repository, RepositoryID: registered.RepositoryID, Credential: credential, Write: config.Mode == "read-write"}
	revision, err := provider.Resolve(ctx, access, config.Reference)
	if err != nil {
		failed := RepositoryAccessCheck{State: "failed", Message: "Repository access validation failed. Check the approved credential, permissions, expiry, and server trust.", CredentialReference: credential, Generation: registered.Generation, Current: credential == config.Credential, CheckedAt: time.Now().UTC()}
		if err := s.storeRepositoryAccess(ctx, user, project, source, row, failed); err != nil {
			return failed, err
		}
		return failed, errors.New(failed.Message)
	}
	result := RepositoryAccessCheck{State: "ready", Message: "Repository access validated.", CredentialReference: credential, Generation: registered.Generation, Current: credential == config.Credential, CheckedAt: time.Now().UTC(), Revision: revision, Credential: sourceprovider.CredentialInfo{Renewal: "operator replacement"}}
	result.LastSuccessAt = &result.CheckedAt
	if inspector, ok := provider.(sourceprovider.CredentialInspector); ok {
		result.Credential, err = inspector.CredentialInfo(ctx, access)
		if err != nil {
			return RepositoryAccessCheck{}, err
		}
	}
	if err = s.storeRepositoryAccess(ctx, user, project, source, row, result); err != nil {
		return result, err
	}
	return result, nil
}

type CredentialRotation struct {
	State     string    `json:"state"`
	Message   string    `json:"message"`
	UpdatedAt time.Time `json:"updatedAt"`
}
type credentialRotationLock struct {
	Digest string             `json:"digest"`
	Key    string             `json:"key"`
	Result CredentialRotation `json:"result"`
}

func (s *Service) RotateRepositoryCredential(ctx context.Context, user, project, source, key string) (CredentialRotation, error) {
	if !domain.ValidID(key) {
		return CredentialRotation{}, errors.New("rotation requires a stable operation identifier")
	}
	row, err := s.ReadRepository(ctx, user, project, source)
	if err != nil {
		return CredentialRotation{}, err
	}
	registered, err := storage.Decode[RepositorySource](row)
	if err != nil {
		return CredentialRotation{}, err
	}
	if err = s.checkRepository(ctx, user, project, source, "manage"); err != nil {
		return CredentialRotation{}, err
	}
	config := registered.Configuration
	provider, ok := s.Providers[config.Provider].(sourceprovider.CredentialRotator)
	if !ok {
		return CredentialRotation{}, errors.New("this connection renews tokens automatically or requires operator key replacement")
	}
	if s.Secrets == nil {
		return CredentialRotation{}, errors.New("credential provider is unavailable")
	}
	secret, err := s.Secrets.Resolve(ctx, project, config.Credential, sourceprovider.CredentialPurpose)
	if err != nil {
		return CredentialRotation{}, err
	}
	bucket := "repository-credential-rotation/" + project
	stateRow, readErr := s.Store.Get(ctx, bucket, config.Credential)
	revision := int64(0)
	lock := credentialRotationLock{Digest: digest(string(secret.Bytes)), Key: key, Result: CredentialRotation{State: "rotating", Message: "Credential rotation started.", UpdatedAt: time.Now().UTC()}}
	if readErr == nil {
		prior, err := storage.Decode[credentialRotationLock](stateRow)
		if err != nil {
			return lock.Result, err
		}
		if prior.Key == key {
			return prior.Result, nil
		}
		if prior.Result.State == "rotating" && time.Since(prior.Result.UpdatedAt) < 3*time.Minute {
			return prior.Result, errors.New("credential rotation is in progress; inspect its outcome before another operation")
		}
		if prior.Digest == lock.Digest && prior.Result.State != "rotated" {
			return prior.Result, errors.New("rotation outcome needs operator recovery; this token must not be rotated again")
		}
		revision = stateRow.Revision
	} else if !errors.Is(readErr, storage.ErrNotFound) {
		return lock.Result, readErr
	}
	change, err := storage.Put(bucket, config.Credential, project, revision, lock)
	if err != nil {
		return lock.Result, err
	}
	err = s.Auth.WithAssignmentTransaction(ctx, func(auth *authorization.Service) error {
		if err := auth.Check(ctx, user, project, "source/"+source, "sources.manage"); err != nil {
			return err
		}
		return auth.Directory.Store.Commit(ctx, change, storage.Change{Record: row, Assert: true})
	})
	if err != nil {
		return lock.Result, err
	}
	err = provider.RotateCredential(ctx, sourceprovider.Access{Project: project, Repository: config.Repository, RepositoryID: registered.RepositoryID, Credential: config.Credential})
	lock.Result = CredentialRotation{State: "rotated", Message: "Credential rotated and stored. Existing suite bindings remain unchanged.", UpdatedAt: time.Now().UTC()}
	if err != nil {
		lock.Result.State = "recovery-required"
		lock.Result.Message = "Rotation was not confirmed. Replace the approved credential and validate access. Do not retry the old token."
	}
	checkpoint, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	current, err := s.Store.Get(checkpoint, bucket, config.Credential)
	if err != nil {
		return lock.Result, err
	}
	previous, err := storage.Decode[credentialRotationLock](current)
	if err != nil {
		return lock.Result, err
	}
	if previous.Key != key {
		return lock.Result, storage.ErrConflict
	}
	saved, err := storage.Put(bucket, config.Credential, project, current.Revision, lock)
	if err != nil {
		return lock.Result, err
	}
	audit, err := domain.Audit(user, project, "sources.repository.credential.rotate", source, nil, lock.Result)
	if err != nil {
		return lock.Result, err
	}
	return lock.Result, s.Store.Commit(checkpoint, saved, audit)
}

func (s *Service) storeRepositoryAccess(ctx context.Context, user, project, source string, sourceRow storage.Record, value RepositoryAccessCheck) error {
	return s.Auth.WithAssignmentTransaction(ctx, func(auth *authorization.Service) error {
		if err := auth.Check(ctx, user, project, "source/"+source, "sources.manage"); err != nil {
			return err
		}
		store := auth.Directory.Store
		previous, err := store.Get(ctx, "repository-access/"+project, source)
		revision := int64(0)
		if err == nil {
			revision = previous.Revision
			before, err := storage.Decode[RepositoryAccessCheck](previous)
			if err != nil {
				return err
			}
			value.LastSuccessAt = before.LastSuccessAt
		} else if !errors.Is(err, storage.ErrNotFound) {
			return err
		}
		if value.State == "ready" {
			value.LastSuccessAt = &value.CheckedAt
		}
		change, err := storage.Put("repository-access/"+project, source, project, revision, value)
		if err != nil {
			return err
		}
		audit, err := domain.Audit(user, project, "sources.repository.access.validate", source, nil, value)
		if err != nil {
			return err
		}
		return store.Commit(ctx, change, audit, storage.Change{Record: sourceRow, Assert: true})
	})
}
func (s *Service) RepositoryAccessStatus(ctx context.Context, user, project, source string) (RepositoryAccessCheck, error) {
	row, err := s.ReadRepository(ctx, user, project, source)
	if err != nil {
		return RepositoryAccessCheck{}, err
	}
	if err = s.checkRepository(ctx, user, project, source, "manage"); err != nil {
		return RepositoryAccessCheck{}, err
	}
	registered, err := storage.Decode[RepositorySource](row)
	if err != nil {
		return RepositoryAccessCheck{}, err
	}
	previous, err := s.Store.Get(ctx, "repository-access/"+project, source)
	if errors.Is(err, storage.ErrNotFound) {
		return RepositoryAccessCheck{State: "unchecked", Message: "Validate repository access to inspect its current credential and trust.", CredentialReference: registered.Configuration.Credential}, nil
	}
	if err != nil {
		return RepositoryAccessCheck{}, err
	}
	value, err := storage.Decode[RepositoryAccessCheck](previous)
	if err != nil {
		return value, err
	}
	value.Current = value.Generation == registered.Generation && value.CredentialReference == registered.Configuration.Credential
	return value, nil
}
