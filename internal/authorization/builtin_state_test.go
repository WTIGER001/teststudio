package authorization_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/authorization"
)

type failingBuiltinState struct {
	permissions.PermissionStore
	fail atomic.Bool
}

var errBuiltinState = errors.New("built-in role state unavailable")

func (s *failingBuiltinState) DisabledBuiltInRoles(ctx context.Context) ([]string, error) {
	if s.fail.Load() {
		return nil, errBuiltinState
	}
	return s.PermissionStore.DisabledBuiltInRoles(ctx)
}

func TestBuiltinRoleStateErrorsCannotReuseAnAllowedEvaluation(t *testing.T) {
	auth := transactionFixture(t)
	if err := auth.Engine.AssignRole(t.Context(), permissions.PrincipalRef{Kind: permissions.PrincipalUser, ID: "operator"}, "builtin.teststudio.viewer", map[string]any{"team": "alpha"}); err != nil {
		t.Fatal(err)
	}
	store := &failingBuiltinState{PermissionStore: auth.Engine.GetStore()}
	engine, err := authorization.NewEngine(t.Context(), store, auth.Directory)
	if err != nil {
		t.Fatal(err)
	}
	service := &authorization.Service{Engine: engine, Directory: auth.Directory}
	if err := service.Check(t.Context(), "operator", "alpha", "", "catalog.read"); err != nil {
		t.Fatal(err)
	}
	store.fail.Store(true)
	if err := service.Check(t.Context(), "operator", "alpha", "", "catalog.read"); !errors.Is(err, errBuiltinState) {
		t.Fatal("reused cached permission after state error", err)
	}
	if _, err := service.CurrentEngineWithIdentity(t.Context(), auth.Directory); !errors.Is(err, errBuiltinState) {
		t.Fatal("guard evaluation ignored state error", err)
	}
	store.fail.Store(false)
	if err := service.Check(t.Context(), "operator", "alpha", "", "catalog.read"); err != nil {
		t.Fatal(err)
	}
	if err := auth.Engine.DisableBuiltInRole(t.Context(), "builtin.teststudio.viewer"); err != nil {
		t.Fatal(err)
	}
	if err := service.Check(t.Context(), "operator", "alpha", "", "catalog.read"); !errors.Is(err, authorization.ErrForbidden) {
		t.Fatal("fresh state did not revoke access", err)
	}
	if err := auth.Engine.EnableBuiltInRole(t.Context(), "builtin.teststudio.viewer"); err != nil {
		t.Fatal(err)
	}
	if err := service.Check(t.Context(), "operator", "alpha", "", "catalog.read"); err != nil {
		t.Fatal("fresh state did not restore access", err)
	}
}
