package main

import (
	"errors"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/internal/identity"
	"github.com/wtiger001/teststudio/internal/sessions"
	"github.com/wtiger001/teststudio/internal/testkit"
)

func TestLocalRecoveryInvalidatesEarlierSessionsAndRetainsAudit(t *testing.T) {
	db, err := testkit.Open(t, filepath.Join(t.TempDir(), "recovery.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	dir := &identity.Directory{Store: db}
	if err := dir.SaveUser(t.Context(), identity.User{ID: "local-administrator", Name: "Local administrator"}, 0); err != nil {
		t.Fatal(err)
	}
	s := &sessions.Service{Store: db, Directory: dir}
	w := httptest.NewRecorder()
	v, err := s.Create(t.Context(), w, "local-administrator", "local", time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if err := recoverLocalAdministrator(t.Context(), dir, "local-administrator"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Validate(t.Context(), v.ID); !errors.Is(err, sessions.ErrUnauthenticated) {
		t.Fatal("recovery revived an earlier session", err)
	}
	u, err := dir.User(t.Context(), "local-administrator")
	if err != nil {
		t.Fatal(err)
	}
	if u.Disabled || u.SessionEpoch != 1 || u.LastSignIn == nil {
		t.Fatal(u)
	}
	audit, err := db.All(t.Context(), "audit", "")
	if err != nil || len(audit) != 1 {
		t.Fatal("recovery audit missing", err)
	}
	if _, err := s.Create(t.Context(), httptest.NewRecorder(), u.ID, "local", time.Time{}); err != nil {
		t.Fatal("new session failed after recovery", err)
	}
}
