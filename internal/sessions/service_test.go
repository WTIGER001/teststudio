package sessions_test

import (
	"database/sql"
	"errors"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/internal/identity"
	"github.com/wtiger001/teststudio/internal/sessions"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/testkit"
)

func TestSessionCSRFRevocationAndUserDisabling(t *testing.T) {
	db, err := testkit.Open(t, filepath.Join(t.TempDir(), "session.db"))
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
	row, err := db.Get(t.Context(), "users", u.ID)
	if err != nil {
		t.Fatal(err)
	}
	u, err = dir.User(t.Context(), u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if u.LastSignIn == nil {
		t.Fatal("sign-in time was not saved")
	}
	u.Disabled = true
	if err := dir.SaveUser(t.Context(), u, row.Revision); err != nil {
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

func TestRotationRollsBackWithItsOwningTransaction(t *testing.T) {
	db, err := testkit.Open(t, filepath.Join(t.TempDir(), "rotation.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	directory := &identity.Directory{Store: db}
	if err := directory.SaveUser(t.Context(), identity.User{ID: "user", Name: "User"}, 0); err != nil {
		t.Fatal(err)
	}
	service := &sessions.Service{Store: db, Directory: directory}
	original, err := service.Create(t.Context(), httptest.NewRecorder(), "user", "local", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	rollback := errors.New("reject owning identity change")
	var nextID string
	err = db.Transaction(t.Context(), func(bound *storage.Store, _ *sql.Tx) error {
		local := *service
		local.Store, local.Directory = bound, &identity.Directory{Store: bound}
		next, _, err := local.Rotate(t.Context(), original.ID)
		if err != nil {
			return err
		}
		nextID = next.ID
		return rollback
	})
	if !errors.Is(err, rollback) {
		t.Fatal("transaction did not reach rotation", err)
	}
	current, err := service.Validate(t.Context(), original.ID)
	if err != nil || current.UserEpoch != original.UserEpoch {
		t.Fatal("rolled-back rotation invalidated session", err)
	}
	if _, err := service.Validate(t.Context(), nextID); !errors.Is(err, sessions.ErrUnauthenticated) {
		t.Fatal("rolled-back replacement survived", err)
	}
}

func TestSessionIdleAbsoluteAndAuthenticationDeadlinesFailNextRead(t *testing.T) {
	for _, mode := range []string{"idle", "absolute", "authentication"} {
		t.Run(mode, func(t *testing.T) {
			db, err := testkit.Open(t, filepath.Join(t.TempDir(), "deadline.db"))
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			directory := &identity.Directory{Store: db}
			if err := directory.SaveUser(t.Context(), identity.User{ID: "user", Name: "User"}, 0); err != nil {
				t.Fatal(err)
			}
			service := &sessions.Service{Store: db, Directory: directory, IdleTimeout: time.Minute}
			writer := httptest.NewRecorder()
			current, err := service.Create(t.Context(), writer, "user", "local", time.Now().Add(time.Hour))
			if err != nil {
				t.Fatal(err)
			}
			row, err := db.Get(t.Context(), "sessions", current.ID)
			if err != nil {
				t.Fatal(err)
			}
			switch mode {
			case "idle":
				current.LastSeen = time.Now().Add(-2 * time.Minute)
			case "absolute":
				current.ExpiresAt = time.Now().Add(-time.Second)
			case "authentication":
				current.AuthenticationUntil = time.Now().Add(-time.Second)
			}
			change, _ := storage.Put(row.Bucket, row.ID, "", row.Revision, current)
			if err := db.Commit(t.Context(), change); err != nil {
				t.Fatal(err)
			}
			request := httptest.NewRequest("GET", "http://studio.example/auth/session", nil)
			request.AddCookie(writer.Result().Cookies()[0])
			if _, err := service.Read(t.Context(), request); !errors.Is(err, sessions.ErrUnauthenticated) {
				t.Fatal("expired session read succeeded", mode, err)
			}
			if _, err := service.Validate(t.Context(), current.ID); !errors.Is(err, sessions.ErrUnauthenticated) {
				t.Fatal("expired private-evidence authorization succeeded", mode, err)
			}
		})
	}
}
