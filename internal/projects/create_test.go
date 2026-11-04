package projects_test

import (
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/access"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/identity"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/testkit"
)

func creatorFixture(t *testing.T) *projects.Service {
	t.Helper()
	db, err := testkit.Open(t, filepath.Join(t.TempDir(), "projects.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	directory := &identity.Directory{Store: db}
	for _, name := range []string{"creator", "other"} {
		if err := directory.SaveUser(t.Context(), identity.User{ID: name, Name: name}, 0); err != nil {
			t.Fatal(err)
		}
	}
	auth, err := authorization.New(t.Context(), testkit.PermissionStore(t, db), directory)
	if err != nil {
		t.Fatal(err)
	}
	if err := auth.Engine.CreateGrant(t.Context(), permissions.Grant{OwnerKind: permissions.PrincipalUser, OwnerID: "creator", PermissionName: "teststudio.projects.create", Effect: permissions.EffectAllow, TeamScope: "*"}); err != nil {
		t.Fatal(err)
	}
	return &projects.Service{Store: db, Auth: auth}
}

func TestProjectCreatorGetsManagedScopedRolesAndRevocationStaysEffective(t *testing.T) {
	s := creatorFixture(t)
	if err := s.SaveProject(t.Context(), "creator", projects.Project{ID: "alpha", Name: "Alpha"}, 0); err != nil {
		t.Fatal(err)
	}
	for _, action := range []string{"projects.manage", "members.manage", "environments.manage", "suites.manage", "catalog.create", "catalog.publish", "sources.manage", "runs.create", "schedules.create"} {
		if err := s.Auth.Check(t.Context(), "creator", "alpha", "", action); err != nil {
			t.Fatal(action, err)
		}
		if err := s.Auth.Check(t.Context(), "creator", "beta", "", action); !errors.Is(err, authorization.ErrForbidden) {
			t.Fatal("access crossed project", action, err)
		}
	}
	for _, action := range []string{"users.manage", "providers.manage", "permissions.grant", "system.manage"} {
		if err := s.Auth.Check(t.Context(), "creator", "", "", action); !errors.Is(err, authorization.ErrForbidden) {
			t.Fatal("installation access granted", action, err)
		}
	}
	control := &access.Service{Store: s.Store, Auth: s.Auth}
	rows, err := control.ListAssignments(t.Context(), "creator", "alpha", "", 100)
	if err != nil || len(rows.Items) != 4 {
		t.Fatal(rows, err)
	}
	for _, row := range rows.Items {
		if !row.Data.Managed || !row.Data.CanRevoke || row.Revision != 2 || row.Updated == nil || row.Data.Bindings["team"] != "alpha" {
			t.Fatal("initial assignment metadata is incomplete", row)
		}
	}
	definition := access.AssignmentDefinition{PrincipalKind: permissions.PrincipalUser, PrincipalID: "creator", RoleID: "builtin.teststudio.operator", Reason: "Withdraw run access"}
	if err := control.RemoveAssignment(t.Context(), "creator", "alpha", definition, 2); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveProject(t.Context(), "creator", projects.Project{ID: "alpha", Name: "Renamed"}, 1); err != nil {
		t.Fatal(err)
	}
	if err := s.Auth.Check(t.Context(), "creator", "alpha", "", "runs.create"); !errors.Is(err, authorization.ErrForbidden) {
		t.Fatal("project edit restored a revoked role", err)
	}
	fresh, err := authorization.New(t.Context(), s.Auth.Engine.GetStore(), s.Auth.Directory)
	if err != nil {
		t.Fatal(err)
	}
	if err := fresh.Check(t.Context(), "creator", "alpha", "", "catalog.create"); err != nil {
		t.Fatal("initial roles were not durable", err)
	}
	if err := fresh.Check(t.Context(), "creator", "alpha", "", "runs.create"); !errors.Is(err, authorization.ErrForbidden) {
		t.Fatal(err)
	}
}

func TestProjectCreatorDisabledRoleRollsBackProjectAssignmentsAuditAndClock(t *testing.T) {
	s := creatorFixture(t)
	if err := s.Auth.Engine.DisableBuiltInRole(t.Context(), "builtin.teststudio.operator"); err != nil {
		t.Fatal(err)
	}
	before, err := s.Auth.Directory.Revision(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SaveProject(t.Context(), "creator", projects.Project{ID: "alpha", Name: "Alpha"}, 0); err == nil || !strings.Contains(err.Error(), "enabled Operator") {
		t.Fatal(err)
	}
	if _, err := s.Store.Get(t.Context(), "projects", "alpha"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatal("partial project survived", err)
	}
	assignments, err := s.Auth.Engine.RoleAssignmentsForPrincipal(t.Context(), permissions.PrincipalRef{Kind: permissions.PrincipalUser, ID: "creator"})
	if err != nil || len(assignments) != 0 {
		t.Fatal("partial role assignment survived", assignments, err)
	}
	for _, bucket := range []string{"access-assignments", "audit"} {
		rows, err := s.Store.All(t.Context(), bucket, "alpha")
		if err != nil || len(rows) != 0 {
			t.Fatal("partial metadata survived", bucket, rows, err)
		}
	}
	after, err := s.Auth.Directory.Revision(t.Context())
	if err != nil || after.Revision != before.Revision {
		t.Fatal("failed creation changed access clock", before, after, err)
	}
}

func TestProjectCreatorRejectsUnauthorizedDuplicateAndDisabledActorsAndPreservesDenies(t *testing.T) {
	s := creatorFixture(t)
	if err := s.SaveProject(t.Context(), "other", projects.Project{ID: "unauthorized", Name: "Unauthorized"}, 0); !errors.Is(err, authorization.ErrForbidden) {
		t.Fatal(err)
	}
	if _, err := s.Store.Get(t.Context(), "projects", "unauthorized"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatal(err)
	}
	if err := s.Auth.Engine.CreateGrant(t.Context(), permissions.Grant{OwnerKind: permissions.PrincipalUser, OwnerID: "creator", PermissionName: "teststudio.runs.create", TeamScope: "alpha", Effect: permissions.EffectDeny}); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveProject(t.Context(), "creator", projects.Project{ID: "alpha", Name: "Original"}, 0); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveProject(t.Context(), "creator", projects.Project{ID: "alpha", Name: "Overwrite"}, 0); !errors.Is(err, storage.ErrConflict) {
		t.Fatal(err)
	}
	if err := s.Auth.Check(t.Context(), "creator", "alpha", "", "runs.create"); !errors.Is(err, authorization.ErrForbidden) {
		t.Fatal("initial role bypassed explicit deny", err)
	}
	rows, err := s.Store.All(t.Context(), "audit", "alpha")
	if err != nil || len(rows) != 5 {
		t.Fatal("duplicate create wrote audit or access", len(rows), err)
	}
	user, err := s.Auth.Directory.User(t.Context(), "creator")
	if err != nil {
		t.Fatal(err)
	}
	user.Disabled = true
	if err := s.Auth.Directory.SaveUser(t.Context(), user, 1); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveProject(t.Context(), "creator", projects.Project{ID: "disabled", Name: "Disabled"}, 0); !errors.Is(err, authorization.ErrForbidden) {
		t.Fatal(err)
	}
}

func TestProjectCreatorConcurrentCreatesGiveAccessOnlyToTheWinner(t *testing.T) {
	s := creatorFixture(t)
	if err := s.Auth.Engine.CreateGrant(t.Context(), permissions.Grant{OwnerKind: permissions.PrincipalUser, OwnerID: "other", PermissionName: "teststudio.projects.create", Effect: permissions.EffectAllow, TeamScope: "*"}); err != nil {
		t.Fatal(err)
	}
	type result struct {
		actor string
		err   error
	}
	results := make(chan result, 2)
	var start sync.WaitGroup
	start.Add(1)
	for _, actor := range []string{"creator", "other"} {
		go func() {
			start.Wait()
			results <- result{actor, s.SaveProject(t.Context(), actor, projects.Project{ID: "contested", Name: actor}, 0)}
		}()
	}
	start.Done()
	wins := 0
	for range 2 {
		outcome := <-results
		if outcome.err == nil {
			wins++
		} else if !errors.Is(outcome.err, storage.ErrConflict) {
			t.Fatal(outcome.err)
		}
		err := s.Auth.Check(t.Context(), outcome.actor, "contested", "", "projects.manage")
		if outcome.err == nil && err != nil || outcome.err != nil && !errors.Is(err, authorization.ErrForbidden) {
			t.Fatal("losing creator gained access", outcome, err)
		}
	}
	if wins != 1 {
		t.Fatal("expected one creator", wins)
	}
	rows, err := s.Store.All(t.Context(), "access-assignments", "contested")
	if err != nil || len(rows) != 4 {
		t.Fatal("unexpected initial assignment count", len(rows), err)
	}
}
