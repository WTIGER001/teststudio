package recording

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/identity"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/testkit"
)

type maintenanceDriver struct{ stopped int }

func (d *maintenanceDriver) Start(context.Context, string, string) error { return nil }
func (d *maintenanceDriver) Read(context.Context, string) (Snapshot, error) {
	return Snapshot{State: "recording", Code: "captured browser steps"}, nil
}
func (d *maintenanceDriver) Stop(context.Context, string) (Snapshot, error) {
	d.stopped++
	return Snapshot{State: "ready", Code: "captured browser steps"}, nil
}

func TestRecordingMaintenanceStopsExpiredAndRevokedSessionsWithoutLosingCapturedCode(t *testing.T) {
	store, err := testkit.Open(t, filepath.Join(t.TempDir(), "recording.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	directory := &identity.Directory{Store: store}
	for _, id := range []string{"admin", "writer"} {
		if err := directory.SaveUser(t.Context(), identity.User{ID: id, Name: id}, 0); err != nil {
			t.Fatal(err)
		}
	}
	auth, err := authorization.New(t.Context(), testkit.PermissionStore(t, store), directory)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range []struct {
		id, role string
		bindings map[string]any
	}{{"admin", "builtin.teststudio.installation-admin", nil}, {"writer", "builtin.teststudio.publisher", map[string]any{"team": "recording"}}} {
		if err := auth.Engine.AssignRole(t.Context(), permissions.PrincipalRef{Kind: permissions.PrincipalUser, ID: item.id}, item.role, item.bindings); err != nil {
			t.Fatal(err)
		}
	}
	p := &projects.Service{Store: store, Auth: auth}
	if err := p.SaveProject(t.Context(), "admin", projects.Project{ID: "recording", Name: "Recording"}, 0); err != nil {
		t.Fatal(err)
	}
	if err := p.SaveSuite(t.Context(), "admin", "recording", projects.Suite{ID: "browser", Name: "Browser"}, 0); err != nil {
		t.Fatal(err)
	}
	driver := &maintenanceDriver{}
	s := &Service{Store: store, Auth: auth, Driver: driver}
	row, err := s.Start(t.Context(), "writer", "recording", "browser", Input{ID: "expired", TestID: "REC-001", Title: "Expired", URL: "http://localhost:8070"})
	if err != nil {
		t.Fatal(err)
	}
	value, _ := storage.Decode[Session](row)
	value.DeadlineAt = time.Now().Add(-time.Minute)
	change, _ := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, value)
	if err := store.Commit(t.Context(), change); err != nil {
		t.Fatal(err)
	}
	s.maintain(t.Context())
	row, err = s.Read(t.Context(), "writer", "recording", "browser", "expired")
	if err != nil {
		t.Fatal(err)
	}
	value, _ = storage.Decode[Session](row)
	if value.State != "failed" || value.Code != "captured browser steps" || driver.stopped != 1 {
		t.Fatal("expiry did not stop and retain recording")
	}
	if _, err := s.Start(t.Context(), "writer", "recording", "browser", Input{ID: "revoked", TestID: "REC-002", Title: "Revoked", URL: "http://localhost:8070"}); err != nil {
		t.Fatal(err)
	}
	userRow, err := store.Get(t.Context(), "users", "writer")
	if err != nil {
		t.Fatal(err)
	}
	user, _ := storage.Decode[identity.User](userRow)
	user.Disabled = true
	if err := directory.SaveUser(t.Context(), user, userRow.Revision); err != nil {
		t.Fatal(err)
	}
	s.maintain(t.Context())
	row, err = store.Get(t.Context(), Bucket(), "revoked")
	if err != nil {
		t.Fatal(err)
	}
	value, _ = storage.Decode[Session](row)
	if value.State != "failed" || driver.stopped != 2 {
		t.Fatal("revocation left a browser running")
	}
}
