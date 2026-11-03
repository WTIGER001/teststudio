package access_test

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/access"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/identity"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/testkit"
)

func fixture(t *testing.T) *access.Service {
	t.Helper()
	db, err := testkit.Open(t, filepath.Join(t.TempDir(), "access.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	dir := &identity.Directory{Store: db}
	if err := dir.SaveUser(t.Context(), identity.User{ID: "admin", Name: "Administrator"}, 0); err != nil {
		t.Fatal(err)
	}
	auth, err := authorization.New(t.Context(), testkit.PermissionStore(t, db), dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := auth.Engine.AssignRole(t.Context(), permissions.PrincipalRef{Kind: permissions.PrincipalUser, ID: "admin"}, "builtin.teststudio.installation-admin", nil); err != nil {
		t.Fatal(err)
	}
	return &access.Service{Store: db, Auth: auth, LocalUser: "admin"}
}

func providerUser(t *testing.T, s *access.Service, subject string) identity.User {
	t.Helper()
	c, err := storage.Put("providers", "example", "", 0, map[string]any{"id": "example", "name": "Example provider", "clientId": "teststudio", "issuer": "https://identity.example.test", "disabled": false})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Store.Get(t.Context(), "providers", "example"); errors.Is(err, storage.ErrNotFound) {
		if err := s.Store.Commit(t.Context(), c); err != nil {
			t.Fatal(err)
		}
	}
	u, err := s.Auth.Directory.ResolveExternal(t.Context(), "https://identity.example.test", subject, subject, "", false, true)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func TestAccessGuardUsesLibraryEvaluationAndKeepsAlternateAdministratorPaths(t *testing.T) {
	s := fixture(t)
	u := providerUser(t, s, "backup")
	for _, id := range []string{"rescue-one", "rescue-two"} {
		if err := s.SaveGroup(t.Context(), "admin", access.GroupDefinition{ID: id, Name: id}, 0); err != nil {
			t.Fatal(err)
		}
		if err := s.Auth.Engine.AssignRole(t.Context(), permissions.PrincipalRef{Kind: permissions.PrincipalGroup, ID: id}, "builtin.teststudio.installation-admin", nil); err != nil {
			t.Fatal(err)
		}
		if err := s.SaveMember(t.Context(), "admin", id, u.ID, nil, 0); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.UserState(t.Context(), "admin", "admin", "Use configured OIDC administrator", 1, true); err != nil {
		t.Fatal(err)
	}
	if err := s.RemoveMember(t.Context(), u.ID, "rescue-one", u.ID, "Keep alternate administrator group", 1); err != nil {
		t.Fatal(err)
	}
	if err := s.GroupState(t.Context(), u.ID, "rescue-two", "Cannot remove final login", 1, true); err == nil || !strings.Contains(err.Error(), "last usable") {
		t.Fatalf("last admin group: %v", err)
	}
	if err := s.RemoveMember(t.Context(), u.ID, "rescue-two", u.ID, "Cannot remove final membership", 1); err == nil || !strings.Contains(err.Error(), "last usable") {
		t.Fatalf("last admin membership: %v", err)
	}
	if err := s.UserState(t.Context(), u.ID, u.ID, "Cannot disable final administrator", 1, true); err == nil || !strings.Contains(err.Error(), "last usable") {
		t.Fatalf("last admin user: %v", err)
	}
	if err := s.UserState(t.Context(), u.ID, "admin", "Restore local recovery access", 2, false); err != nil {
		t.Fatal(err)
	}
	if err := s.GroupState(t.Context(), u.ID, "rescue-two", "Local administrator remains available", 1, true); err != nil {
		t.Fatal(err)
	}
}

func TestAccessGuardDoesNotCountDisabledProvidersOrDeniedAdministrators(t *testing.T) {
	for _, mode := range []string{"disabled-provider", "explicit-deny", "unlinked-user"} {
		t.Run(mode, func(t *testing.T) {
			s := fixture(t)
			u := providerUser(t, s, "backup")
			if mode == "unlinked-user" {
				u = identity.User{ID: "unlinked", Name: "Unlinked"}
				if err := s.Auth.Directory.SaveUser(t.Context(), u, 0); err != nil {
					t.Fatal(err)
				}
			}
			if err := s.Auth.Engine.AssignRole(t.Context(), permissions.PrincipalRef{Kind: permissions.PrincipalUser, ID: u.ID}, "builtin.teststudio.installation-admin", nil); err != nil {
				t.Fatal(err)
			}
			if mode == "disabled-provider" {
				c, err := storage.Put("providers", "example", "", 1, map[string]any{"issuer": "https://identity.example.test", "disabled": true})
				if err != nil {
					t.Fatal(err)
				}
				if err := s.Store.Commit(t.Context(), c); err != nil {
					t.Fatal(err)
				}
			}
			if mode == "explicit-deny" {
				if err := s.Auth.Engine.CreateGrant(t.Context(), permissions.Grant{OwnerKind: permissions.PrincipalUser, OwnerID: u.ID, PermissionName: "teststudio.system.manage", Effect: permissions.EffectDeny, TeamScope: "*"}); err != nil {
					t.Fatal(err)
				}
			}
			if err := s.UserState(t.Context(), "admin", "admin", "No usable backup", 1, true); err == nil || !strings.Contains(err.Error(), "last usable") {
				t.Fatalf("counted unusable backup: %v", err)
			}
		})
	}
}

type barrierStore struct {
	permissions.PermissionStore
	arrivals *atomic.Int32
	ready    chan struct{}
}

func (s *barrierStore) GrantsForOwners(ctx context.Context, owners []permissions.PrincipalRef, request permissions.Request) ([]permissions.Grant, error) {
	if request.Perm == "teststudio.system.manage" {
		n := s.arrivals.Add(1)
		if n == 2 {
			close(s.ready)
		}
		if n <= 2 {
			select {
			case <-s.ready:
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	}
	return s.PermissionStore.GrantsForOwners(ctx, owners, request)
}

func TestAccessGuardSerializesCompetingAdministratorRemovalAcrossInstances(t *testing.T) {
	s := fixture(t)
	u := providerUser(t, s, "backup")
	if err := s.Auth.Engine.AssignRole(t.Context(), permissions.PrincipalRef{Kind: permissions.PrincipalUser, ID: u.ID}, "builtin.teststudio.installation-admin", nil); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	gate := &barrierStore{PermissionStore: s.Auth.Engine.GetStore(), arrivals: &atomic.Int32{}, ready: make(chan struct{})}
	a, err := authorization.New(ctx, gate, s.Auth.Directory)
	if err != nil {
		t.Fatal(err)
	}
	b, err := authorization.New(ctx, gate, s.Auth.Directory)
	if err != nil {
		t.Fatal(err)
	}
	first := &access.Service{Store: s.Store, Auth: a, LocalUser: "admin"}
	second := &access.Service{Store: s.Store, Auth: b, LocalUser: "admin"}
	results := make(chan error, 2)
	go func() { results <- first.UserState(ctx, "admin", u.ID, "First concurrent removal", 1, true) }()
	go func() { results <- second.UserState(ctx, u.ID, "admin", "Second concurrent removal", 1, true) }()
	success := 0
	for range 2 {
		err := <-results
		if err == nil {
			success++
		} else if !errors.Is(err, storage.ErrConflict) {
			t.Fatal(err)
		}
	}
	if success != 1 {
		t.Fatalf("concurrent removals succeeded %d times", success)
	}
	admin, err := s.Auth.Directory.User(t.Context(), "admin")
	if err != nil {
		t.Fatal(err)
	}
	backup, err := s.Auth.Directory.User(t.Context(), u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if admin.Disabled == backup.Disabled {
		t.Fatal("expected exactly one usable administrator to remain")
	}
}

func TestAccessDirectoryVersionRejectsStaleMutationAndPersistsAcrossRestart(t *testing.T) {
	s := fixture(t)
	version, err := s.Auth.Directory.Revision(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SaveGroup(t.Context(), "admin", access.GroupDefinition{ID: "local-group", Name: "Local group"}, 0); err != nil {
		t.Fatal(err)
	}
	c, err := storage.Put("users", "stale-user", "", 0, identity.User{ID: "stale-user", Name: "Stale"})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Auth.Directory.CommitAt(t.Context(), version, c); !errors.Is(err, storage.ErrConflict) {
		t.Fatal(err)
	}
	if _, err := s.Store.Get(t.Context(), "users", "stale-user"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatal("stale transaction partially committed", err)
	}
	if err := s.SaveMember(t.Context(), "admin", "local-group", "admin", nil, 0); err != nil {
		t.Fatal(err)
	}
	version, err = s.Auth.Directory.Revision(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	var local string
	if s.Store.Dialect == "sqlite" {
		if err := s.Store.DB.QueryRowContext(t.Context(), "SELECT file FROM pragma_database_list WHERE name='main'").Scan(&local); err != nil {
			t.Fatal(err)
		}
	}
	dsn := testkit.ConnectionString(t, s.Store, local)
	db, err := storage.Open(t.Context(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	dir := &identity.Directory{Store: db}
	after, err := dir.Revision(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if after.Revision != version.Revision {
		t.Fatal("identity version did not persist")
	}
	ok, err := dir.IsUserInGroup(t.Context(), "admin", "local-group")
	if err != nil || !ok {
		t.Fatal("membership did not persist", err)
	}
}

func TestAccessCurrentProfileRequiresNoDirectoryGrantAndKeepsIdentityOwnership(t *testing.T) {
	s := fixture(t)
	u := providerUser(t, s, "profile-user")
	other := providerUser(t, s, "other-user")
	profile, err := s.CurrentProfile(t.Context(), u.ID, "", 30)
	if err != nil {
		t.Fatal(err)
	}
	if profile.User.ID != u.ID || profile.LocalAccount || len(profile.Identities.Items) != 1 {
		t.Fatal(profile)
	}
	link, err := storage.Decode[identity.ExternalIdentity](profile.Identities.Items[0])
	if err != nil {
		t.Fatal(err)
	}
	if link.UserID != u.ID || link.Subject != "profile-user" {
		t.Fatal("profile crossed identity ownership", link)
	}
	if _, err := s.UserIdentities(t.Context(), u.ID, other.ID, "", 30); !errors.Is(err, authorization.ErrForbidden) {
		t.Fatal("unprivileged profile read another user", err)
	}
	if err := s.UserState(t.Context(), "admin", u.ID, "Disable account", 1, true); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CurrentProfile(t.Context(), u.ID, "", 30); !errors.Is(err, authorization.ErrForbidden) {
		t.Fatal("disabled user read profile", err)
	}
}
