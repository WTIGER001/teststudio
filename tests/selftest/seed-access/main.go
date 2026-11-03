package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
	"os"
	"time"

	permissions "github.com/wtiger001/go-permissions"
	permissionsqlite "github.com/wtiger001/teststudio/adapters/permissions/sqlite"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/identity"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/sessions"
	"github.com/wtiger001/teststudio/internal/storage"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "isolated access fixture preparation failed")
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) != 2 || !domain.ValidID(os.Args[1]) {
		return errors.New("one project identifier is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	db, err := storage.Open(ctx, "/tmp/teststudio-selftest.db")
	if err != nil {
		return err
	}
	defer db.Close()
	project := os.Args[1]
	r, err := db.Get(ctx, "projects", project)
	if err != nil {
		return err
	}
	p, err := storage.Decode[projects.Project](r)
	if err != nil {
		return err
	}
	if p.Archived {
		return errors.New("fixture project is archived")
	}
	directory := &identity.Directory{Store: db}
	if _, err := directory.User(ctx, "local-administrator"); err != nil {
		return err
	}
	auth, err := authorization.New(ctx, permissionsqlite.New(db), directory)
	if err != nil {
		return err
	}
	user, err := directory.ResolveExternal(ctx, "https://identity.selftest.invalid", "project-manager", "Project manager", "manager@example.test", true, true)
	if err != nil {
		return err
	}
	if err := auth.Engine.AssignRole(ctx, permissions.PrincipalRef{Kind: permissions.PrincipalUser, ID: user.ID}, "builtin.teststudio.project-admin", map[string]any{"team": project}); err != nil {
		return err
	}
	group := identity.Group{ID: "fixture-external", Name: "Provider reviewers", Description: "New isolated browser authority fixture", Project: project, ManagedBy: "oidc:browser-fixture"}
	if err := directory.SaveGroup(ctx, group, 0); err != nil {
		return err
	}
	if err := directory.SyncExternalGroups(ctx, user.ID, group.ManagedBy, []string{group.ID}, time.Now().Add(time.Hour)); err != nil {
		return err
	}
	service := &sessions.Service{Store: db, Directory: directory, Origin: "http://127.0.0.1:8040"}
	w := httptest.NewRecorder()
	if _, err := service.Create(ctx, w, user.ID, "isolated-browser-fixture", time.Now().Add(time.Hour)); err != nil {
		return err
	}
	cookies := w.Result().Cookies()
	if len(cookies) != 1 {
		return errors.New("fixture session cookie is unavailable")
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]any{"userId": user.ID, "groupId": group.ID, "cookie": cookies[0].Value})
}
