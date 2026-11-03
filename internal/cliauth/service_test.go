package cliauth_test

import (
	"context"
	"errors"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/internal/cliauth"
	"github.com/wtiger001/teststudio/internal/identity"
	"github.com/wtiger001/teststudio/internal/sessions"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/testkit"
)

func setup(t *testing.T) (*cliauth.Service, *sessions.Service, sessions.Session) {
	t.Helper()
	store, err := testkit.Open(t, filepath.Join(t.TempDir(), "cli.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	directory := &identity.Directory{Store: store}
	if err := directory.SaveUser(t.Context(), identity.User{ID: "user", Name: "User"}, 0); err != nil {
		t.Fatal(err)
	}
	sessionService := &sessions.Service{Store: store, Directory: directory, Origin: "https://teststudio.example", Secure: true}
	parent, err := sessionService.Create(t.Context(), httptest.NewRecorder(), "user", "local", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	return &cliauth.Service{Sessions: sessionService}, sessionService, parent
}
func request(t *testing.T) (cliauth.Proof, string) {
	t.Helper()
	id, secret, code, err := cliauth.NewRequest(time.Now(), "Publisher on local workstation")
	if err != nil {
		t.Fatal(err)
	}
	return cliauth.Proof{Request: id, Client: "Publisher on local workstation", Verifier: secret}, code
}
func TestCLIConfirmationKeepsProofPrivateAndExchangesOnceAcrossCompetingClients(t *testing.T) {
	service, sessionService, parent := setup(t)
	proof, code := request(t)
	if result, err := service.Exchange(t.Context(), proof); err != nil || result.State != "pending" || result.Session != nil {
		t.Fatal(result, err)
	}
	if rows, _ := sessionService.Store.All(t.Context(), "cli-authorizations", ""); len(rows) != 0 {
		t.Fatal("anonymous polling created durable authorization state")
	}
	if _, err := service.Confirm(t.Context(), parent.ID, proof.Request, cliauth.Confirmation{Client: proof.Client, Code: "incorrect", Approve: true}); err == nil {
		t.Fatal("incorrect terminal code accepted")
	}
	if _, err := service.Confirm(t.Context(), parent.ID, proof.Request, cliauth.Confirmation{Client: proof.Client, Code: code, Approve: true}); err != nil {
		t.Fatal(err)
	}
	forged := proof
	forged.Verifier = sessions.Random()
	if _, err := service.Exchange(t.Context(), forged); !errors.Is(err, cliauth.ErrInvalid) {
		t.Fatal("wrong private proof accepted", err)
	}
	forged = proof
	forged.Client = "Another client"
	if _, err := service.Exchange(t.Context(), forged); !errors.Is(err, cliauth.ErrInvalid) {
		t.Fatal("client label was not bound to proof", err)
	}
	results := make(chan cliauth.Exchange, 2)
	failures := make(chan error, 2)
	var workers sync.WaitGroup
	for range 2 {
		workers.Go(func() { result, err := service.Exchange(t.Context(), proof); results <- result; failures <- err })
	}
	workers.Wait()
	close(results)
	close(failures)
	for err := range failures {
		if err != nil && !errors.Is(err, storage.ErrConflict) {
			t.Fatal(err)
		}
	}
	grants := []cliauth.Grant{}
	for result := range results {
		if result.Session != nil {
			grants = append(grants, *result.Session)
		}
	}
	if len(grants) != 1 {
		t.Fatal("CLI proof issued multiple managed sessions", len(grants))
	}
	grant := grants[0]
	child, err := sessionService.Validate(t.Context(), sessions.Digest(grant.Cookie))
	if err != nil {
		t.Fatal(err)
	}
	if child.ID == parent.ID || child.CSRF == parent.CSRF || child.UserID != parent.UserID || child.UserEpoch != parent.UserEpoch || child.AuthenticationMethod != parent.AuthenticationMethod || child.Provider != parent.Provider || child.Client != "publisher-cli" || child.ClientName != proof.Client || !child.AuthenticationUntil.Equal(parent.AuthenticationUntil) || !child.ExpiresAt.Equal(parent.AuthenticationUntil) {
		t.Fatalf("delegation altered authentication bounds: %+v", child)
	}
	if _, err := sessionService.Validate(t.Context(), parent.ID); err != nil {
		t.Fatal("delegation replaced browser session", err)
	}
	if result, err := service.Exchange(t.Context(), proof); err != nil || result.State != "consumed" || result.Session != nil {
		t.Fatal("CLI proof replay", result, err)
	}
	if err := service.Cancel(t.Context(), proof); err != nil {
		t.Fatal(err)
	}
	if _, err := sessionService.Validate(t.Context(), child.ID); !errors.Is(err, sessions.ErrUnauthenticated) {
		t.Fatal("private cancellation did not revoke issued CLI session", err)
	}
	if _, err := sessionService.Validate(t.Context(), parent.ID); err != nil {
		t.Fatal("CLI cancellation revoked browser", err)
	}
}
func TestCLIConfirmationDenialExpiryAndParentRevocationIssueNoSessions(t *testing.T) {
	for _, scenario := range []string{"declined", "parent-revoked", "expired", "user-disabled"} {
		t.Run(scenario, func(t *testing.T) {
			service, sessionService, parent := setup(t)
			proof, code := request(t)
			approve := scenario != "declined"
			if _, err := service.Confirm(t.Context(), parent.ID, proof.Request, cliauth.Confirmation{Client: proof.Client, Code: code, Approve: approve}); err != nil {
				t.Fatal(err)
			}
			switch scenario {
			case "parent-revoked":
				if err := sessionService.Revoke(t.Context(), parent.ID); err != nil {
					t.Fatal(err)
				}
			case "expired":
				service.Now = func() time.Time { return time.Now().Add(cliauth.Lifetime + time.Minute) }
			case "user-disabled":
				user, err := sessionService.Directory.User(t.Context(), "user")
				if err != nil {
					t.Fatal(err)
				}
				row, _ := sessionService.Store.Get(t.Context(), "users", "user")
				user.Disabled = true
				if err := sessionService.Directory.SaveUser(t.Context(), user, row.Revision); err != nil {
					t.Fatal(err)
				}
			}
			result, err := service.Exchange(t.Context(), proof)
			if result.Session != nil || scenario == "declined" && (err != nil || result.State != "denied") || scenario != "declined" && err == nil {
				t.Fatal("invalid authorization issued a CLI session", result, err)
			}
			rows, _ := sessionService.Store.All(t.Context(), "sessions", "")
			for _, row := range rows {
				value, _ := storage.Decode[sessions.Session](row)
				if value.Client == "publisher-cli" {
					t.Fatal("unapproved CLI session persisted")
				}
			}
			if scenario == "expired" {
				if err := service.Sweep(t.Context()); err != nil {
					t.Fatal(err)
				}
				rows, _ := sessionService.Store.All(t.Context(), "cli-authorizations", "")
				if len(rows) != 0 {
					t.Fatal("expired approval was not removed")
				}
			}
		})
	}
}
func TestCLILabelAndRequestValidation(t *testing.T) {
	for _, label := range []string{"", " leading", "line\nbreak", strings.Repeat("x", 101)} {
		if _, _, _, err := cliauth.NewRequest(time.Now(), label); err == nil {
			t.Fatal("invalid client label accepted")
		}
	}
	service, _, parent := setup(t)
	if _, err := service.Read(context.Background(), parent.ID, "invalid", "Client"); !errors.Is(err, cliauth.ErrInvalid) {
		t.Fatal("invalid request accepted", err)
	}
}
