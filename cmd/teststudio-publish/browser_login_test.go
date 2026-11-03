package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/internal/cliauth"
	"github.com/wtiger001/teststudio/internal/sessions"
)

func TestCLIBrowserLoginSavesOnlyManagedCredentialsAndCleansFailedDelivery(t *testing.T) {
	for _, failedSave := range []bool{false, true} {
		t.Run(map[bool]string{false: "private-save", true: "failed-save"}[failedSave], func(t *testing.T) {
			cookie, csrf := sessions.Random(), sessions.Random()
			cancelled, exchanges := false, 0
			privateProof := ""
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/auth/cli/options":
					json.NewEncoder(w).Encode(cliauth.Options{ServerTime: time.Now().UTC(), LifetimeSeconds: 300, PollSeconds: 2})
				case "/auth/cli/exchange", "/auth/cli/cancel":
					var proof cliauth.Proof
					if json.NewDecoder(r.Body).Decode(&proof) != nil || !sessionToken.MatchString(proof.Verifier) || proof.Client != "Fresh publisher" || strings.Contains(r.URL.String(), proof.Verifier) {
						t.Error("CLI private proof transport is invalid")
						w.WriteHeader(400)
						return
					}
					privateProof = proof.Verifier
					if r.URL.Path == "/auth/cli/cancel" {
						cancelled = true
						w.Write([]byte(`{"state":"cancelled"}`))
						return
					}
					exchanges++
					json.NewEncoder(w).Encode(cliauth.Exchange{State: "authorized", Session: &cliauth.Grant{Cookie: cookie, CSRF: csrf, ExpiresAt: time.Now().Add(time.Hour)}})
				default:
					t.Error("unexpected browser login operation")
					w.WriteHeader(404)
				}
			}))
			defer server.Close()
			filename := filepath.Join(t.TempDir(), "session.json")
			if failedSave {
				if err := os.Mkdir(filename, 0700); err != nil {
					t.Fatal(err)
				}
			}
			var output bytes.Buffer
			err := run(t.Context(), []string{"login", "--browser", "--no-open", "--server", server.URL, "--client-name", "Fresh publisher", "--session-file", filename}, &output)
			if failedSave {
				if err == nil || !cancelled {
					t.Fatal("failed session delivery did not cancel issued session", err)
				}
			} else {
				if err != nil || cancelled {
					t.Fatal(err)
				}
				client, err := readClient(filename)
				if err != nil || client.session.Cookie != cookie || client.session.CSRF != csrf {
					t.Fatal("managed CLI session was not saved", err)
				}
				info, _ := os.Stat(filename)
				if info.Mode().Perm() != 0600 {
					t.Fatal("CLI session file is not private")
				}
			}
			if exchanges != 1 || privateProof == "" {
				t.Fatal("CLI did not exchange one private proof")
			}
			for _, secret := range []string{cookie, csrf, privateProof} {
				if strings.Contains(output.String(), secret) {
					t.Fatal("CLI printed a private credential")
				}
			}
			if !strings.Contains(output.String(), "Terminal code:") || !strings.Contains(output.String(), server.URL+"/cli-sign-in/") {
				t.Fatal("CLI did not provide browser confirmation instructions")
			}
		})
	}
}
