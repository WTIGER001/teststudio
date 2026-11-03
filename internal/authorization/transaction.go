package authorization

import (
	"context"
	"errors"
	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/identity"
	"github.com/wtiger001/teststudio/internal/storage"
)

type AssignmentTransactor interface {
	WithAssignmentTransaction(context.Context, *storage.Store, func(*storage.Store, permissions.PermissionStore) error) error
}

func (s *Service) WithAssignmentTransaction(ctx context.Context, fn func(*Service) error) error {
	transactor, ok := s.Engine.GetStore().(AssignmentTransactor)
	if !ok {
		return errors.New("permission store does not support atomic assignment changes")
	}
	version, err := s.Directory.Revision(ctx)
	if err != nil {
		return err
	}
	return transactor.WithAssignmentTransaction(ctx, s.Directory.Store, func(metadata *storage.Store, store permissions.PermissionStore) error {
		directory := &identity.Directory{Store: metadata}
		if err := directory.CommitAt(ctx, version); err != nil {
			return err
		}
		bound, err := New(ctx, store, directory)
		if err != nil {
			return err
		}
		return fn(bound)
	})
}
