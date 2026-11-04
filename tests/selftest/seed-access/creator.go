package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"os"
	"time"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/sessions"
	"github.com/wtiger001/teststudio/internal/storage"
)

func prepareCreator(ctx context.Context, db *storage.Store, auth *authorization.Service) error {
	user, err := auth.Directory.ResolveExternal(ctx, "https://identity.selftest.invalid", "project-creator", "Project creator", "creator@example.test", true, true)
	if err != nil {
		return err
	}
	if err := auth.Engine.CreateGrant(ctx, permissions.Grant{OwnerKind: permissions.PrincipalUser, OwnerID: user.ID, PermissionName: "teststudio.projects.create", Effect: permissions.EffectAllow, TeamScope: "*"}); err != nil {
		return err
	}
	service := &sessions.Service{Store: db, Directory: auth.Directory, Origin: "http://127.0.0.1:8040"}
	w := httptest.NewRecorder()
	if _, err := service.Create(ctx, w, user.ID, "isolated-browser-fixture", time.Now().Add(time.Hour)); err != nil {
		return err
	}
	cookies := w.Result().Cookies()
	if len(cookies) != 1 {
		return errors.New("creator session is unavailable")
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]string{"userId": user.ID, "cookie": cookies[0].Value})
}
