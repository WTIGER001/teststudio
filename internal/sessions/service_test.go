package sessions_test

import (
	"errors"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/internal/identity"
	"github.com/wtiger001/teststudio/internal/sessions"
	"github.com/wtiger001/teststudio/internal/storage"
)

func TestSessionCSRFRevocationAndUserDisabling(t *testing.T) {
	db, err := storage.Open(t.Context(), filepath.Join(t.TempDir(), "session.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	dir := &identity.Directory{Store: db}
	u := identity.User{ID: "user", Name: "User"}
	if err := dir.SaveUser(t.Context(), u, 0); err != nil {
		t.Fatal(err)
	}
	s := &sessions.Service{Store: db, Directory: dir, Origin: "https://teststudio.example", Secure: true}
	w := httptest.NewRecorder()
	v, err := s.Create(t.Context(), w, u.ID, "oidc", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	cookie := w.Result().Cookies()[0]
	if !cookie.HttpOnly || !cookie.Secure {
		t.Fatal("session cookie is missing required flags")
	}
	r := httptest.NewRequest("POST", s.Origin+"/api/v1/projects", nil)
	r.AddCookie(cookie)
	r.Header.Set("Origin", s.Origin)
	r.Header.Set("X-CSRF-Token", v.CSRF)
	if err := s.CheckMutation(r, v); err != nil {
		t.Fatal(err)
	}
	r.Header.Set("Origin", "https://other.example")
	if err := s.CheckMutation(r, v); err == nil {
		t.Fatal("accepted cross-origin mutation")
	}
	if _, err := s.Read(t.Context(), r); err != nil {
		t.Fatal(err)
	}
	u.Disabled = true
	if err := dir.SaveUser(t.Context(), u, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Read(t.Context(), r); !errors.Is(err, sessions.ErrUnauthenticated) {
		t.Fatalf("disabled user kept session: %v", err)
	}
	if err := s.Revoke(t.Context(), v.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Read(t.Context(), r); !errors.Is(err, sessions.ErrUnauthenticated) {
		t.Fatalf("revoked session survived: %v", err)
	}
}
