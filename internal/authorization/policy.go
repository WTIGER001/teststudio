package authorization

import (
	"context"
	"errors"
)

type PolicyRemovalStore interface {
	RemoveGrant(context.Context, int64) error
	RemoveRoleInheritance(context.Context, string, string) error
}

func (s *Service) policyRemovalStore() (PolicyRemovalStore, error) {
	if !s.policyTransaction {
		return nil, errors.New("policy removal requires an atomic policy transaction")
	}
	store, ok := s.Engine.GetStore().(PolicyRemovalStore)
	if !ok {
		return nil, errors.New("permission store does not support atomic policy removal")
	}
	return store, nil
}

func (s *Service) RemovePolicyGrant(ctx context.Context, id int64) error {
	store, err := s.policyRemovalStore()
	if err != nil {
		return err
	}
	if id < 1 {
		return errors.New("a positive grant identifier is required")
	}
	return store.RemoveGrant(ctx, id)
}

func (s *Service) RemoveRoleInheritance(ctx context.Context, parent, child string) error {
	store, err := s.policyRemovalStore()
	if err != nil {
		return err
	}
	return store.RemoveRoleInheritance(ctx, parent, child)
}
