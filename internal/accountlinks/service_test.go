package accountlinks

import (
	"encoding/json"
	"errors"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/internal/identity"
	"github.com/wtiger001/teststudio/internal/localauth"
	"github.com/wtiger001/teststudio/internal/passwords"
	"github.com/wtiger001/teststudio/internal/sessions"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/testkit"
)

func fixture(t *testing.T) (*Service, sessions.Session) {
	t.Helper()
	store, err := testkit.Open(t, filepath.Join(t.TempDir(), "link.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	directory := &identity.Directory{Store: store}
	if err := directory.SaveUser(t.Context(), identity.User{ID: "owner", Name: "Owner"}, 0); err != nil {
		t.Fatal(err)
	}
	sessionService := &sessions.Service{Store: store, Directory: directory, Origin: "https://studio.example", Secure: true}
	current, err := sessionService.Create(t.Context(), httptest.NewRecorder(), "owner", "local", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	provider, err := storage.Put("providers", "target", "", 0, map[string]any{"id": "target", "name": "Target", "issuer": "https://identity.example", "clientId": "private-client", "secretRef": "env:PRIVATE_SECRET"})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Commit(t.Context(), provider); err != nil {
		t.Fatal(err)
	}
	return &Service{Store: store, Sessions: sessionService, LocalUser: "owner"}, current
}

func start(t *testing.T, s *Service, current sessions.Session) View {
	t.Helper()
	op, err := s.Start(t.Context(), current.ID, StartInput{ProviderID: "target", Reason: "Add a second sign-in", ConfirmLocal: true}, "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	return op
}

func prove(t *testing.T, s *Service, current sessions.Session, op View, subject string) View {
	t.Helper()
	req, err := s.PrepareProof(t.Context(), current.ID, op.ID, sessions.Digest(sessions.Random()))
	if err != nil {
		t.Fatal(err)
	}
	login, err := storage.Put("login-transactions", req.TransactionID, "", 0, req)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.BindProof(t.Context(), current.ID, req, login); err != nil {
		t.Fatal(err)
	}
	result, err := s.AcceptProof(t.Context(), current.ID, req, Proof{Issuer: op.TargetProvider.Issuer, Subject: subject}, time.Now(), time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.AcceptProof(t.Context(), current.ID, req, Proof{Issuer: op.TargetProvider.Issuer, Subject: subject}, time.Now(), time.Now().Add(time.Hour)); err == nil {
		t.Fatal("accepted proof replay")
	}
	return result
}

func TestLinkRequiresExplicitConfirmationAndRotatesAllSessions(t *testing.T) {
	s, current := fixture(t)
	other, err := s.Sessions.Create(t.Context(), httptest.NewRecorder(), "owner", "local", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	op := prove(t, s, current, start(t, s, current), "exact-subject")
	if op.State != "review" || op.Identity.Subject != "exact-subject" {
		t.Fatal("missing exact review", op)
	}
	if _, err := s.Sessions.Directory.ExternalLink(t.Context(), op.Identity.Issuer, op.Identity.Subject); !errors.Is(err, storage.ErrNotFound) {
		t.Fatal("proof linked identity without confirmation", err)
	}
	if _, _, err := s.Confirm(t.Context(), other.ID, op.ID, op.Revision); !errors.Is(err, storage.ErrNotFound) {
		t.Fatal("different session confirmed", err)
	}
	if _, _, err := s.Confirm(t.Context(), current.ID, op.ID, 0); !errors.Is(err, storage.ErrConflict) {
		t.Fatal("missing revision accepted", err)
	}
	completed, cookie, err := s.Confirm(t.Context(), current.ID, op.ID, op.Revision)
	if err != nil {
		t.Fatal(err)
	}
	if completed.State != "complete" || cookie == nil || !cookie.Secure || !cookie.HttpOnly {
		t.Fatal("incomplete link confirmation")
	}
	next, err := s.Sessions.Validate(t.Context(), sessions.Digest(cookie.Value))
	if err != nil {
		t.Fatal(err)
	}
	if next.ID == current.ID || next.CSRF == current.CSRF || next.UserEpoch != current.UserEpoch+1 || !next.ExpiresAt.Equal(current.ExpiresAt) || !next.AuthenticationUntil.Equal(current.AuthenticationUntil) || next.AuthenticationMethod != current.AuthenticationMethod {
		t.Fatal("session rotation changed deadlines or retained credentials")
	}
	for _, old := range []string{current.ID, other.ID} {
		if _, err := s.Sessions.Validate(t.Context(), old); !errors.Is(err, sessions.ErrUnauthenticated) {
			t.Fatal("older session survived", err)
		}
	}
	linked, err := s.Sessions.Directory.ExternalLink(t.Context(), op.Identity.Issuer, op.Identity.Subject)
	if err != nil || linked.UserID != current.UserID {
		t.Fatal("wrong identity owner", linked, err)
	}
	if recovered, err := s.Read(t.Context(), next.ID, op.ID); err != nil || recovered.State != "complete" {
		t.Fatal("rotated session cannot recover confirmation", err)
	}
	if _, _, err := s.Confirm(t.Context(), next.ID, op.ID, completed.Revision); err == nil {
		t.Fatal("confirmation replay accepted")
	}
	data, _ := json.Marshal(completed)
	for _, private := range []string{"sessionId", "activeTransaction", "sourceVerifiedAt", "proofStartedAt", "password", "clientId", "secretRef"} {
		if strings.Contains(string(data), private) {
			t.Fatal("private operation state leaked", private)
		}
	}
}

func TestLinkRejectsClaimedIdentitiesAndPreservesOwner(t *testing.T) {
	s, current := fixture(t)
	op := prove(t, s, current, start(t, s, current), "claimed-subject")
	other, err := s.Sessions.Directory.ResolveExternal(t.Context(), op.Identity.Issuer, op.Identity.Subject, "Other", "same@example.test", true, true)
	if err != nil {
		t.Fatal(err)
	}
	if _, cookie, err := s.Confirm(t.Context(), current.ID, op.ID, op.Revision); !errors.Is(err, ErrClaimed) || cookie != nil {
		t.Fatal("claimed identity accepted", err)
	}
	link, err := s.Sessions.Directory.ExternalLink(t.Context(), op.Identity.Issuer, op.Identity.Subject)
	if err != nil || link.UserID != other.ID {
		t.Fatal("identity transferred", link, err)
	}
	if _, err := s.Sessions.Validate(t.Context(), current.ID); err != nil {
		t.Fatal("failed confirmation revoked session", err)
	}
}

func TestLinkRejectsProviderChangesExpiryCancellationAndSessionChanges(t *testing.T) {
	for _, change := range []string{"provider", "expiry", "cancel", "session"} {
		t.Run(change, func(t *testing.T) {
			s, current := fixture(t)
			op := prove(t, s, current, start(t, s, current), "subject")
			switch change {
			case "provider":
				row, _ := s.Store.Get(t.Context(), "providers", "target")
				c, _ := storage.Put(row.Bucket, row.ID, "", row.Revision, map[string]any{"id": "target", "name": "Changed", "issuer": op.TargetProvider.Issuer})
				if err := s.Store.Commit(t.Context(), c); err != nil {
					t.Fatal(err)
				}
			case "expiry":
				row, _ := s.Store.Get(t.Context(), bucket, op.ID)
				value, _ := storage.Decode[operation](row)
				value.ExpiresAt = time.Now().Add(-time.Second)
				c, _ := storage.Put(bucket, op.ID, "", row.Revision, value)
				if err := s.Store.Commit(t.Context(), c); err != nil {
					t.Fatal(err)
				}
				op.Revision++
			case "cancel":
				cancelled, err := s.Cancel(t.Context(), current.ID, op.ID, op.Revision)
				if err != nil || cancelled.State != "cancelled" || cancelled.Identity != nil {
					t.Fatal("cancel failed", err)
				}
				op.Revision = cancelled.Revision
			case "session":
				if err := s.Sessions.Revoke(t.Context(), current.ID); err != nil {
					t.Fatal(err)
				}
			}
			if _, cookie, err := s.Confirm(t.Context(), current.ID, op.ID, op.Revision); err == nil || cookie != nil {
				t.Fatal("invalidated link confirmed")
			}
			if _, err := s.Sessions.Directory.ExternalLink(t.Context(), op.TargetProvider.Issuer, "subject"); !errors.Is(err, storage.ErrNotFound) {
				t.Fatal("invalid link persisted", err)
			}
		})
	}
}

func TestStartRejectsUnverifiedLocalAndInternalSessions(t *testing.T) {
	s, current := fixture(t)
	input := StartInput{ProviderID: "target", Reason: "Link identity"}
	if _, err := s.Start(t.Context(), current.ID, input, "127.0.0.1"); err == nil {
		t.Fatal("local confirmation not required")
	}
	input.ConfirmLocal = true
	if _, err := s.Start(t.Context(), current.ID, input, "192.0.2.1"); err == nil {
		t.Fatal("remote local proof accepted")
	}
	trusted, err := s.Sessions.Create(t.Context(), httptest.NewRecorder(), "owner", "fixture", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Start(t.Context(), trusted.ID, input, "127.0.0.1"); err == nil {
		t.Fatal("internal fixture used as fresh authentication")
	}
}

func TestPasswordSourceRequiresFreshCurrentCredential(t *testing.T) {
	s, _ := fixture(t)
	s.Sessions.LocalAccounts = true
	local, err := localauth.New(t.Context(), s.Sessions)
	if err != nil {
		t.Fatal(err)
	}
	s.LocalAccounts = local
	hash, err := passwords.Hash(t.Context(), "First fresh password! 47")
	if err != nil {
		t.Fatal(err)
	}
	c, _ := storage.Put("local-credentials", "owner", "", 0, identity.LocalCredential{UserID: "owner", Hash: hash})
	if err := s.Store.Commit(t.Context(), c); err != nil {
		t.Fatal(err)
	}
	active, _, err := local.Login(t.Context(), "owner", "First fresh password! 47", "127.0.0.1", "")
	if err != nil {
		t.Fatal(err)
	}
	input := StartInput{ProviderID: "target", Reason: "Attach provider", Password: "wrong"}
	if _, err := s.Start(t.Context(), active.ID, input, "127.0.0.1"); !errors.Is(err, localauth.ErrInvalid) {
		t.Fatal("password not verified", err)
	}
	input.Password = "First fresh password! 47"
	op, err := s.Start(t.Context(), active.ID, input, "127.0.0.1")
	if err != nil || op.State != "prove-target" {
		t.Fatal("fresh password rejected", err)
	}
	row, _ := s.Store.Get(t.Context(), "local-credentials", "owner")
	credential, _ := storage.Decode[identity.LocalCredential](row)
	credential.Disabled = true
	c, _ = storage.Put(row.Bucket, row.ID, "", row.Revision, credential)
	if err := s.Store.Commit(t.Context(), c); err != nil {
		t.Fatal(err)
	}
	if _, err := s.PrepareProof(t.Context(), active.ID, op.ID, sessions.Digest(sessions.Random())); err == nil {
		t.Fatal("disabled credential retained linking proof")
	}
}

func TestNewProofSupersedesOlderBrowserTransaction(t *testing.T) {
	s, current := fixture(t)
	op := start(t, s, current)
	bind := func() ProofRequest {
		req, err := s.PrepareProof(t.Context(), current.ID, op.ID, sessions.Digest(sessions.Random()))
		if err != nil {
			t.Fatal(err)
		}
		login, _ := storage.Put("login-transactions", req.TransactionID, "", 0, req)
		if err := s.BindProof(t.Context(), current.ID, req, login); err != nil {
			t.Fatal(err)
		}
		return req
	}
	first, second := bind(), bind()
	proof := Proof{Issuer: op.TargetProvider.Issuer, Subject: "new-subject"}
	if _, err := s.AcceptProof(t.Context(), current.ID, first, proof, time.Now(), time.Now().Add(time.Hour)); err == nil {
		t.Fatal("superseded proof accepted")
	}
	if err := s.RejectProof(t.Context(), current.ID, first); err == nil {
		t.Fatal("superseded callback poisoned the new proof")
	}
	if _, err := s.AcceptProof(t.Context(), current.ID, second, proof, time.Now(), time.Now().Add(time.Hour)); err != nil {
		t.Fatal("current proof did not survive stale callback", err)
	}
}
