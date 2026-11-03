package authorization_test

import (
	"context"
	"errors"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/identity"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/testkit"
)

type policyBarrier struct {
	permissions.PermissionStore
	authorization.PolicyTransactor
	arrivals *atomic.Int32
	ready    chan struct{}
}

func (s *policyBarrier) WithPolicyTransaction(ctx context.Context, metadata *storage.Store, fn func(*storage.Store, permissions.PermissionStore) error) error {
	if s.arrivals.Add(1) == 2 {
		close(s.ready)
	}
	select {
	case <-s.ready:
	case <-ctx.Done():
		return ctx.Err()
	}
	return s.PolicyTransactor.WithPolicyTransaction(ctx, metadata, fn)
}

func TestPolicyTransactionSerializesCompetingRolesAcrossIndependentConnections(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	path := filepath.Join(t.TempDir(), "policy.db")
	first, err := testkit.Open(t, path)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	directory := &identity.Directory{Store: first}
	if err := directory.SaveUser(ctx, identity.User{ID: "operator", Name: "Operator"}, 0); err != nil {
		t.Fatal(err)
	}
	second, err := storage.Open(ctx, testkit.ConnectionString(t, first, path))
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	arrivals, ready := &atomic.Int32{}, make(chan struct{})
	instances := []*authorization.Service{}
	for _, db := range []*storage.Store{first, second} {
		store := testkit.PermissionStore(t, db)
		barrier := &policyBarrier{PermissionStore: store, PolicyTransactor: store.(authorization.PolicyTransactor), arrivals: arrivals, ready: ready}
		auth, err := authorization.New(ctx, barrier, &identity.Directory{Store: db})
		if err != nil {
			t.Fatal(err)
		}
		instances = append(instances, auth)
	}
	results := make(chan error, 2)
	for i, id := range []string{"one", "two"} {
		auth := instances[i]
		go func() {
			results <- auth.WithPolicyTransaction(ctx, func(bound *authorization.Service) error {
				if err := bound.Engine.CreateRole(ctx, permissions.Role{ID: id, Name: id}); err != nil {
					return err
				}
				if err := bound.Engine.CreateGrant(ctx, permissions.Grant{OwnerKind: permissions.PrincipalRole, OwnerID: id, TeamScope: "alpha", PermissionName: "teststudio.catalog.read", Effect: permissions.EffectAllow}); err != nil {
					return err
				}
				if err := bound.Engine.AssignRole(ctx, permissions.PrincipalRef{Kind: permissions.PrincipalUser, ID: "operator"}, id, nil); err != nil {
					return err
				}
				c, _ := storage.Put("audit", id, "alpha", 0, id)
				return bound.Directory.Store.Commit(ctx, c)
			})
		}()
	}
	wins := 0
	for range 2 {
		err := <-results
		if err == nil {
			wins++
		} else if !errors.Is(err, storage.ErrConflict) {
			t.Fatal(err)
		}
	}
	if wins != 1 {
		t.Fatal("expected one committed policy", wins)
	}
	store := testkit.PermissionStore(t, first)
	roles, err := store.RoleDefinitions(ctx)
	if err != nil || len(roles) != 1 {
		t.Fatal("competing roles leaked", roles, err)
	}
	grants, err := store.ListGrants(ctx, permissions.GrantQuery{})
	if err != nil || len(grants.Grants) != 1 {
		t.Fatal("competing grants leaked", grants, err)
	}
	assignments, err := store.RoleAssignmentsForPrincipal(ctx, permissions.PrincipalRef{Kind: permissions.PrincipalUser, ID: "operator"})
	if err != nil || len(assignments) != 1 || assignments[0].RoleID != roles[0].ID {
		t.Fatal("competing assignments leaked", assignments, err)
	}
	audit, err := first.All(ctx, "audit", "")
	if err != nil || len(audit) != 1 || audit[0].ID != roles[0].ID {
		t.Fatal("competing audit leaked", audit, err)
	}
}
