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

type PolicyTransactor interface {
	WithPolicyTransaction(context.Context, *storage.Store, func(*storage.Store, permissions.PermissionStore) error) error
}

func (s *Service) WithPolicyTransaction(ctx context.Context, fn func(*Service) error) error {
	transactor, ok := s.Engine.GetStore().(PolicyTransactor)
	if !ok {
		return errors.New("permission store does not support atomic policy changes")
	}
	return s.withTransaction(ctx, transactor.WithPolicyTransaction, func(bound *Service) error { bound.policyTransaction = true; return fn(bound) })
}

func (s *Service) WithAssignmentTransaction(ctx context.Context, fn func(*Service) error) error {
	transactor, ok := s.Engine.GetStore().(AssignmentTransactor)
	if !ok {
		return errors.New("permission store does not support atomic assignment changes")
	}
	return s.withTransaction(ctx, transactor.WithAssignmentTransaction, fn)
}

func (s *Service) withTransaction(ctx context.Context, transaction func(context.Context, *storage.Store, func(*storage.Store, permissions.PermissionStore) error) error, fn func(*Service) error) error {
	version, err := s.Directory.Revision(ctx)
	if err != nil {
		return err
	}
	return transaction(ctx, s.Directory.Store, func(metadata *storage.Store, store permissions.PermissionStore) error {
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
