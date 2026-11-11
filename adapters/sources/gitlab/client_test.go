package gitlab_test

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/x509"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/wtiger001/teststudio/adapters/sources/gitlab"
	"github.com/wtiger001/teststudio/internal/networkguard"
	"github.com/wtiger001/teststudio/internal/secrets"
	"github.com/wtiger001/teststudio/internal/sourceprovider"
)

type credentials struct{ token string }

func (p *credentials) References(context.Context, string, string) ([]secrets.Reference, error) {
	return []secrets.Reference{{ID: "access", Name: "Access"}}, nil
}
func (p *credentials) Resolve(_ context.Context, project, id, purpose string) (secrets.Value, error) {
	if project != "alpha" || id != "access" {
		return secrets.Value{}, secrets.ErrUnavailable
	}
	if purpose == sourceprovider.WebhookPurpose {
		return secrets.Value{Bytes: []byte(strings.Repeat("s", 32))}, nil
	}
	data, _ := json.Marshal(map[string]any{"token": p.token, "write": true, "repositories": []string{"group/subgroup/suite"}})
	return secrets.Value{Bytes: data}, nil
}
func TestGitLabPinnedArchiveScopeAndCredentialReplacement(t *testing.T) {
	sha := strings.Repeat("a", 40)
	p := &credentials{token: "fixture-one"}
	downloads := 0
	var archive bytes.Buffer
	z := zip.NewWriter(&archive)
	file, _ := z.Create("suite-commit/check.spec.ts")
	file.Write([]byte("new independent fixture"))
	z.Close()
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("PRIVATE-TOKEN") != p.token {
			t.Error("current credential was not resolved")
		}
		switch {
		case r.URL.Path == "/api/v4/projects/group/subgroup/suite":
			json.NewEncoder(w).Encode(map[string]any{"id": 42, "path_with_namespace": "group/subgroup/suite", "visibility": "private"})
		case r.URL.Path == "/api/v4/projects/42/repository/branches/feature/a":
			json.NewEncoder(w).Encode(map[string]any{"commit": map[string]string{"id": sha}})
		case r.URL.Path == "/api/v4/projects/42/repository/archive.zip":
			if r.URL.Query().Get("sha") != sha {
				t.Error("archive was not pinned")
			}
			downloads++
			w.Write(archive.Bytes())
		default:
			w.WriteHeader(404)
		}
	}))
	defer server.Close()
	roots := x509.NewCertPool()
	roots.AddCert(server.Certificate())
	c := &gitlab.Client{Secrets: p, BaseURL: server.URL, Network: networkguard.Loopback, RootCAs: roots}
	access := sourceprovider.Access{Project: "alpha", Repository: "group/subgroup/suite", Credential: "access"}
	revision, err := c.Resolve(t.Context(), access, "heads/feature/a")
	if err != nil || revision.Commit != sha || revision.RepositoryID != "42" {
		t.Fatal("resolve", err)
	}
	p.token = "fixture-two"
	got, err := c.Archive(t.Context(), access, revision)
	if err != nil || got.Root != "suite-commit" || downloads != 1 {
		t.Fatal("archive", err)
	}
	access.RepositoryID = "999"
	if _, err = c.Resolve(t.Context(), access, "heads/feature/a"); err == nil {
		t.Fatal("repository replacement accepted")
	}
	access.RepositoryID = ""
	access.Repository = "other/suite"
	if _, err = c.Resolve(t.Context(), access, "heads/feature/a"); err == nil {
		t.Fatal("unapproved repository accepted")
	}
	untrusted := *c
	untrusted.RootCAs = nil
	access.Repository = "group/subgroup/suite"
	if _, err = untrusted.Resolve(t.Context(), access, "heads/feature/a"); err == nil {
		t.Fatal("private trust was bypassed")
	}
}
func TestGitLabWebhookAuthenticatesAndHandlesDeletion(t *testing.T) {
	c := &gitlab.Client{Secrets: &credentials{token: "fixture"}}
	body := []byte(`{"object_kind":"push","ref":"refs/heads/main","after":"` + strings.Repeat("0", 40) + `","project":{"id":42,"path_with_namespace":"group/subgroup/suite"}}`)
	request := sourceprovider.WebhookRequest{Delivery: "event-1", Event: "Push Hook", Signature: strings.Repeat("s", 32), Body: body}
	event, err := c.VerifyWebhook(t.Context(), sourceprovider.Access{Project: "alpha"}, "access", request)
	if err != nil || !event.Deleted || event.Kind != "push" {
		t.Fatal("deletion", err)
	}
	request.Signature = "bad"
	if _, err = c.VerifyWebhook(t.Context(), sourceprovider.Access{Project: "alpha"}, "access", request); err == nil {
		t.Fatal("unsigned payload accepted")
	}
}
