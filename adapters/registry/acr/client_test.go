package acr

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/adapters/registry/oci"
)

func TestACRInspectionUsesScopedTokensAndRefreshesOnIdentityChange(t *testing.T) {
	for _, runner := range []string{"go", "playwright"} {
		t.Run(runner, func(t *testing.T) {
			f := fresh(t, runner, false, false)
			image, err := f.client.InspectImage(t.Context(), f.reference)
			if err != nil || image.Runner != runner || image.Project != "parcel" || image.Architecture != "amd64" {
				t.Fatal("ACR did not retain normal supervised image inspection", image, err)
			}
			var group sync.WaitGroup
			for range 8 {
				group.Go(func() {
					if _, err := f.client.InspectImage(t.Context(), f.reference); err != nil {
						t.Error(err)
					}
				})
			}
			group.Wait()
			if f.exchanges != 1 || f.grants != 1 {
				t.Fatal("concurrent inspection repeated token exchange", f.exchanges, f.grants)
			}
			f.credential.mu.Lock()
			f.credential.current.Token = "entra-rotated-private-canary"
			f.credential.mu.Unlock()
			if _, err := f.client.InspectImage(t.Context(), f.reference); err != nil {
				t.Fatal(err)
			}
			if f.exchanges != 2 || f.grants != 2 {
				t.Fatal("changed source credentials reused an older registry token")
			}
			f.clock.advance(6 * time.Minute)
			if _, err := f.client.InspectImage(t.Context(), f.reference); err != nil || f.exchanges != 3 || f.grants != 3 {
				t.Fatal("cache did not respect its maximum age", err)
			}
			before := f.exchanges
			f.credential.mu.Lock()
			f.credential.fail = true
			f.credential.mu.Unlock()
			if _, err := f.client.InspectImage(t.Context(), f.reference); err == nil || strings.Contains(err.Error(), "canary") || f.exchanges != before {
				t.Fatal("failed source identity reused a cached token or leaked diagnostics", err)
			}
			data, _ := json.Marshal(f.client)
			if strings.Contains(string(data)+fmt.Sprintf("%+v %#v", f.client, f.client), "private-canary") {
				t.Fatal("credential appeared in client serialization")
			}
		})
	}
}

func TestACRInspectionRejectsForeignChallengesAndInvalidTokenResponses(t *testing.T) {
	for _, mode := range []string{"foreign-challenge", "exchange-redirect", "token-redirect", "duplicate-json", "oversize", "denied", "expired", "missing-exp", "future-nbf", "invalid-token"} {
		t.Run(mode, func(t *testing.T) {
			f := fresh(t, "go", false, false)
			f.mode = mode
			_, err := f.client.InspectImage(t.Context(), f.reference)
			if err == nil || strings.Contains(err.Error(), "canary") || f.unexpected != 0 {
				t.Fatal("invalid authentication response accepted or escaped to another host", err)
			}
			if mode == "foreign-challenge" && (f.exchanges != 0 || f.credential.calls != 0) {
				t.Fatal("foreign challenge obtained credentials")
			}
		})
	}
}

func TestACRReadAccessAndBlobRedirectsNeverForwardRegistryCredentials(t *testing.T) {
	f := fresh(t, "playwright", false, false)
	f.mode = "blob-redirect"
	if _, err := f.client.InspectImage(t.Context(), f.reference); err != nil || f.blobReads != 1 {
		t.Fatal("approved credential-free blob fetch failed", err)
	}
	for _, scope := range []struct{ repo, actions string }{{"teststudio-other/private", "pull"}, {"teststudio/playwright", "pull,push"}, {"teststudio/playwright", "delete"}, {"teststudio/playwright", "*"}} {
		if _, err := f.client.auth.Token(t.Context(), scope.repo, scope.actions); !errors.Is(err, oci.ErrDenied) {
			t.Fatal("inspection authority widened its repository or actions", err)
		}
	}
	before := f.metadata
	if _, err := f.client.InspectImage(t.Context(), strings.Replace(f.reference, "/teststudio/", "/teststudio-other/", 1)); err == nil || f.metadata != before {
		t.Fatal("foreign repository reached the registry")
	}
	f.client.Close()
	if _, err := f.client.InspectImage(t.Context(), f.reference); err == nil || f.metadata != before {
		t.Fatal("closed authentication client performed another request")
	}
}

func TestACRCacheHonorsEntraExpiryAndRemainsBounded(t *testing.T) {
	f := fresh(t, "go", false, false)
	for i := 0; i < 40; i++ {
		if _, err := f.client.auth.Token(t.Context(), fmt.Sprintf("teststudio/suite-%d", i), "pull"); err != nil {
			t.Fatal(err)
		}
	}
	if len(f.client.auth.entries) != 32 || f.exchanges != 1 {
		t.Fatal("repository token cache was unbounded or repeated root exchange")
	}
	f.credential.mu.Lock()
	f.credential.current.ExpiresOn = f.clock.now().Add(20 * time.Second)
	f.credential.mu.Unlock()
	if _, err := f.client.auth.Token(t.Context(), "teststudio/suite-39", "pull"); !errors.Is(err, oci.ErrUnavailable) || len(f.client.auth.entries) != 0 {
		t.Fatal("expiring source identity retained an older token", err)
	}
}
