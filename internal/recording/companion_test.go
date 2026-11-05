package recording

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/identity"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/testkit"
)

func companionFixture(t *testing.T) *Service {
	t.Helper()
	store, err := testkit.Open(t, filepath.Join(t.TempDir(), "companion.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	directory := &identity.Directory{Store: store}
	for _, id := range []string{"admin", "writer", "other"} {
		if err := directory.SaveUser(t.Context(), identity.User{ID: id, Name: id}, 0); err != nil {
			t.Fatal(err)
		}
	}
	auth, err := authorization.New(t.Context(), testkit.PermissionStore(t, store), directory)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"admin", "writer", "other"} {
		role, bindings := "builtin.teststudio.publisher", map[string]any{"team": "recording"}
		if id == "admin" {
			role, bindings = "builtin.teststudio.installation-admin", nil
		}
		if err := auth.Engine.AssignRole(t.Context(), permissions.PrincipalRef{Kind: permissions.PrincipalUser, ID: id}, role, bindings); err != nil {
			t.Fatal(err)
		}
	}
	p := &projects.Service{Store: store, Auth: auth}
	if err := p.SaveProject(t.Context(), "admin", projects.Project{ID: "recording", Name: "Recording"}, 0); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"browser", "another"} {
		if err := p.SaveSuite(t.Context(), "admin", "recording", projects.Suite{ID: id, Name: id}, 0); err != nil {
			t.Fatal(err)
		}
	}
	s := &Service{Store: store, Auth: auth, CompanionEnabled: true}
	row, err := s.Start(t.Context(), "writer", "recording", "browser", Input{ID: "session", TestID: "REC-001", Title: "Continue", URL: "https://example.test"})
	if err != nil {
		t.Fatal(err)
	}
	value, _ := storage.Decode[Session](row)
	if value.Mode != "companion" || value.State != "pairing" {
		t.Fatal("hosted reservation did not wait for the desktop")
	}
	return s
}

func pairCompanion(t *testing.T, s *Service) PairRequest {
	t.Helper()
	pairing, err := s.Pairing(t.Context(), "writer", "recording", "browser", "session")
	if err != nil {
		t.Fatal(err)
	}
	token, err := newCompanionToken()
	if err != nil {
		t.Fatal(err)
	}
	input := PairRequest{PairingToken: pairing.Token, Instance: "desktop-one", DeviceToken: token}
	packet, err := s.Pair(t.Context(), "session", input)
	if err != nil || packet.Input.ID != "session" || packet.Suite != "browser" || packet.Control.State != "starting" {
		t.Fatal("pairing did not return the bound recording", err)
	}
	return input
}

func TestCompanionPairingRotatesBindsOneDeviceAndSurvivesServiceRestart(t *testing.T) {
	s := companionFixture(t)
	for _, scope := range [][3]string{{"other", "recording", "browser"}, {"writer", "recording", "another"}, {"writer", "another", "browser"}} {
		if _, err := s.Pairing(t.Context(), scope[0], scope[1], scope[2], "session"); !errors.Is(err, authorization.ErrForbidden) {
			t.Fatal("pairing crossed owner or scope", err)
		}
	}
	old, err := s.Pairing(t.Context(), "writer", "recording", "browser", "session")
	if err != nil {
		t.Fatal(err)
	}
	input := pairCompanion(t, s)
	stale := input
	stale.PairingToken = old.Token
	if _, err := s.Pair(t.Context(), "session", stale); !errors.Is(err, authorization.ErrForbidden) {
		t.Fatal("rotated pairing remained usable", err)
	}
	restarted := &Service{Store: s.Store, Auth: s.Auth, CompanionEnabled: true}
	if _, err := restarted.Pair(t.Context(), "session", input); err != nil {
		t.Fatal("lost acknowledgement could not be retried", err)
	}
	for _, changed := range []PairRequest{{PairingToken: input.PairingToken, Instance: "desktop-two", DeviceToken: input.DeviceToken}, {PairingToken: input.PairingToken, Instance: input.Instance, DeviceToken: old.Token}} {
		if _, err := restarted.Pair(t.Context(), "session", changed); !errors.Is(err, authorization.ErrForbidden) {
			t.Fatal("paired session admitted another companion", err)
		}
	}
	if _, err := s.Pairing(t.Context(), "writer", "recording", "browser", "session"); !errors.Is(err, storage.ErrConflict) {
		t.Fatal("paired session issued another pairing", err)
	}
	if _, err := restarted.CompanionControl(t.Context(), "session", input.Instance, input.PairingToken); !errors.Is(err, authorization.ErrForbidden) {
		t.Fatal("pairing token became a device capability", err)
	}
	if _, err := restarted.CompanionControl(t.Context(), "session", input.Instance, input.DeviceToken); err != nil {
		t.Fatal(err)
	}
	private, err := s.Store.Get(t.Context(), companionBucket, "session")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(private.Data), input.DeviceToken) || strings.Contains(string(private.Data), input.PairingToken) {
		t.Fatal("server retained raw companion credentials")
	}
	if _, err := s.Read(t.Context(), "other", "recording", "browser", "session"); !errors.Is(err, authorization.ErrForbidden) {
		t.Fatal("other publisher read recording", err)
	}
}

func TestCompanionSnapshotsAreOrderedAndStopCancelRemainDurable(t *testing.T) {
	for _, cancel := range []bool{false, true} {
		t.Run(map[bool]string{false: "stop", true: "cancel"}[cancel], func(t *testing.T) {
			s := companionFixture(t)
			input := pairCompanion(t, s)
			snapshot := CompanionSnapshot{Sequence: 1, State: "recording", Code: "captured steps"}
			submit := func(value CompanionSnapshot) (CompanionControl, error) {
				return s.SubmitCompanion(t.Context(), "session", input.Instance, input.DeviceToken, value)
			}
			if _, err := submit(snapshot); err != nil {
				t.Fatal(err)
			}
			if _, err := submit(snapshot); err != nil {
				t.Fatal("exact retry failed", err)
			}
			for _, changed := range []CompanionSnapshot{{Sequence: 1, State: "recording", Code: "changed"}, {Sequence: 3, State: "ready", Code: "future"}} {
				if _, err := submit(changed); !errors.Is(err, storage.ErrConflict) {
					t.Fatal("snapshot order changed", err)
				}
			}
			if _, err := s.Stop(t.Context(), "writer", "recording", "browser", "session", false); err != nil {
				t.Fatal(err)
			}
			if cancel {
				if _, err := s.Stop(t.Context(), "writer", "recording", "browser", "session", true); err != nil {
					t.Fatal(err)
				}
			}
			s = &Service{Store: s.Store, Auth: s.Auth, CompanionEnabled: true}
			control, err := s.CompanionControl(t.Context(), "session", input.Instance, input.DeviceToken)
			wantAction := map[bool]string{false: "stop", true: "cancel"}[cancel]
			if err != nil || control.State != "stopping" || control.Action != wantAction {
				t.Fatal("stop intent was lost on restart", control, err)
			}
			if control, err = submit(CompanionSnapshot{Sequence: 2, State: "recording", Code: "last steps"}); err != nil || control.State != "stopping" {
				t.Fatal("late recording cleared stop", err)
			}
			terminal := CompanionSnapshot{Sequence: 3, State: "ready", Code: "final steps"}
			control, err = submit(terminal)
			wantState := map[bool]string{false: "ready", true: "cancelled"}[cancel]
			if err != nil || control.State != wantState {
				t.Fatal("final acknowledgement lost intent", control, err)
			}
			if _, err := submit(terminal); err != nil {
				t.Fatal("terminal retry failed", err)
			}
			if _, err := submit(CompanionSnapshot{Sequence: 4, State: "recording", Code: "overwrite"}); !errors.Is(err, authorization.ErrForbidden) {
				t.Fatal("terminal recording reopened", err)
			}
		})
	}
}

func TestCompanionRevocationLeaseAndExpiryRetainCapturedCode(t *testing.T) {
	for _, reason := range []string{"revoked", "suite-archived", "expired", "disconnected", "disabled"} {
		t.Run(reason, func(t *testing.T) {
			s := companionFixture(t)
			input := pairCompanion(t, s)
			if _, err := s.SubmitCompanion(t.Context(), "session", input.Instance, input.DeviceToken, CompanionSnapshot{Sequence: 1, State: "recording", Code: "last safe capture"}); err != nil {
				t.Fatal(err)
			}
			switch reason {
			case "revoked":
				row, _ := s.Store.Get(t.Context(), "users", "writer")
				user, _ := storage.Decode[identity.User](row)
				user.Disabled = true
				if err := s.Auth.Directory.SaveUser(t.Context(), user, row.Revision); err != nil {
					t.Fatal(err)
				}
			case "suite-archived":
				row, _ := s.Store.Get(t.Context(), projects.Bucket("suites", "recording"), "browser")
				value, _ := storage.Decode[projects.Suite](row)
				value.Archived = true
				change, _ := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, value)
				if err := s.Store.Commit(t.Context(), change); err != nil {
					t.Fatal(err)
				}
			case "disabled":
				s.CompanionEnabled = false
			default:
				row, _ := s.Store.Get(t.Context(), Bucket(), "session")
				value, _ := storage.Decode[Session](row)
				private, _ := s.Store.Get(t.Context(), companionBucket, "session")
				connection, _ := storage.Decode[companionRecord](private)
				if reason == "expired" {
					value.DeadlineAt = time.Now().Add(-time.Minute)
					connection.Binding = companionBinding(value)
				} else {
					connection.LastSeen = time.Now().Add(-CompanionLease - time.Second)
				}
				a, _ := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, value)
				b, _ := storage.Put(private.Bucket, private.ID, private.Project, private.Revision, connection)
				if err := s.Store.Commit(t.Context(), a, b); err != nil {
					t.Fatal(err)
				}
			}
			s.maintain(t.Context())
			row, _ := s.Store.Get(t.Context(), Bucket(), "session")
			value, _ := storage.Decode[Session](row)
			if value.State != "failed" || value.Code != "last safe capture" {
				t.Fatal("lost authority did not close and retain capture", value.State)
			}
			if _, err := s.SubmitCompanion(t.Context(), "session", input.Instance, input.DeviceToken, CompanionSnapshot{Sequence: 2, State: "ready", Code: "late replacement"}); !errors.Is(err, authorization.ErrForbidden) {
				t.Fatal("revoked companion replaced captured code", err)
			}
		})
	}
}

func TestCompanionRejectsUnsafeSnapshotsAndExpiredPairing(t *testing.T) {
	s := companionFixture(t)
	input := pairCompanion(t, s)
	for _, snapshot := range []CompanionSnapshot{{State: "ready"}, {Sequence: 1, State: "cancelled"}, {Sequence: 1, State: "recording", Code: string([]byte{255})}, {Sequence: 1, State: "recording", Code: strings.Repeat("a", ContentLimit+1)}, {Sequence: 1, State: "failed", Failure: "/private/browser/log"}} {
		if _, err := s.SubmitCompanion(t.Context(), "session", input.Instance, input.DeviceToken, snapshot); err == nil {
			t.Fatal("unsafe snapshot admitted")
		}
	}
	row, _ := s.Store.Get(t.Context(), companionBucket, "session")
	connection, _ := storage.Decode[companionRecord](row)
	connection.PairUntil = time.Now().Add(-time.Second)
	change, _ := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, connection)
	if err := s.Store.Commit(t.Context(), change); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Pair(t.Context(), "session", input); !errors.Is(err, authorization.ErrForbidden) {
		t.Fatal("expired pairing accepted", err)
	}
	if _, err := s.SubmitCompanion(t.Context(), "session", input.Instance, input.DeviceToken, CompanionSnapshot{Sequence: 1, State: "failed", Failure: "capture"}); err != nil {
		t.Fatal("pairing expiry revoked an already paired device", err)
	}
}

func TestCompanionTransientEmptyCaptureDoesNotDiscardRetainedSource(t *testing.T) {
	s := companionFixture(t)
	input := pairCompanion(t, s)
	for i, snapshot := range []CompanionSnapshot{{State: "recording", Code: "previously captured source"}, {State: "recording"}, {State: "failed", Failure: "capture"}} {
		snapshot.Sequence = uint64(i + 1)
		if _, err := s.SubmitCompanion(t.Context(), "session", input.Instance, input.DeviceToken, snapshot); err != nil {
			t.Fatal(err)
		}
	}
	row, err := s.Read(t.Context(), "writer", "recording", "browser", "session")
	if err != nil {
		t.Fatal(err)
	}
	value, _ := storage.Decode[Session](row)
	if value.Code != "previously captured source" || value.State != "failed" {
		t.Fatal("partial file write discarded the last captured source")
	}
}
