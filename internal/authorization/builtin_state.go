package authorization

import (
	"context"
	"slices"
	"strings"

	permissions "github.com/wtiger001/go-permissions"
)

type builtInStateStore struct {
	permissions.PermissionStore
	disabled []string
}

func (s *builtInStateStore) DisabledBuiltInRoles(context.Context) ([]string, error) {
	return slices.Clone(s.disabled), nil
}

func (s *Service) CurrentEngineWithIdentity(ctx context.Context, identity permissions.IdentityProvider) (*permissions.Service, error) {
	disabled, err := s.Engine.DisabledBuiltInRoles(ctx)
	if err != nil {
		return nil, err
	}
	return NewEngine(ctx, &builtInStateStore{PermissionStore: &applicationStore{PermissionStore: s.Engine.GetStore()}, disabled: slices.Clone(disabled)}, identity)
}

func (s *Service) CurrentEngine(ctx context.Context) (*permissions.Service, error) {
	disabled, err := s.Engine.DisabledBuiltInRoles(ctx)
	if err != nil {
		return nil, err
	}
	disabled = slices.Clone(disabled)
	slices.Sort(disabled)
	state := strings.Join(disabled, "\x00")
	s.evaluationMu.Lock()
	defer s.evaluationMu.Unlock()
	if s.evaluationEngine != nil && s.evaluationOrigin == s.Engine && s.evaluationState == state {
		return s.evaluationEngine, nil
	}
	engine, err := NewEngine(ctx, &builtInStateStore{PermissionStore: &applicationStore{PermissionStore: s.Engine.GetStore()}, disabled: disabled}, s.Directory)
	if err != nil {
		return nil, err
	}
	s.evaluationOrigin = s.Engine
	s.evaluationState = state
	s.evaluationEngine = engine
	return engine, nil
}
