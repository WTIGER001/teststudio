package github_test

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/adapters/sources/github"
	"github.com/wtiger001/teststudio/internal/networkguard"
	"github.com/wtiger001/teststudio/internal/secrets"
	"github.com/wtiger001/teststudio/internal/sourcebundle"
	"github.com/wtiger001/teststudio/internal/sourceprovider"
)

type credentials struct{ data []byte }

func (p credentials) References(context.Context, string, string) ([]secrets.Reference, error) {
	return []secrets.Reference{{ID: "app", Name: "Local fixture app"}}, nil
}
func (p credentials) Resolve(_ context.Context, project, id, purpose string) (secrets.Value, error) {
	if project != "alpha" || id != "app" || purpose != sourceprovider.CredentialPurpose {
		return secrets.Value{}, secrets.ErrUnavailable
	}
	return secrets.Value{Bytes: p.data}, nil
}

type githubFixture struct {
	client                                *github.Client
	key                                   *rsa.PrivateKey
	archive                               []byte
	commit                                string
	permission                            string
	apiToken                              string
	redirected                            string
	repositoryID                          int64
	tokenRequests, downloads, apiRequests int
}

func fixture(t *testing.T) *githubFixture {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	f := &githubFixture{key: key, commit: strings.Repeat("a", 40), permission: "read", repositoryID: 42, apiToken: "fixture-installation-token"}
	var zipBytes bytes.Buffer
	writer := zip.NewWriter(&zipBytes)
	for name, data := range map[string]string{"owner-suite-fixed/teststudio-suite.json": `{"schemaVersion":1,"suiteId":"unit","type":"unit","runner":"go","applications":[],"moduleRoot":".","tests":{"packages":["./checks"]},"catalog":"catalog.json","buildProfile":"go-module"}`, "owner-suite-fixed/catalog.json": `{"schemaVersion":"teststudio.catalog/v1","tests":[{"id":"GITHUB-001","title":"Compute a value","category":"Unit","runner":"go","type":"unit","state":"enabled","safety":"safe","rationale":"Computes in memory","timeoutSeconds":5,"source":"checks/value_test.go","selector":"./checks::TestValue"}]}`, "owner-suite-fixed/go.mod": "module github-fixture.local/source\n\ngo 1.24\n", "owner-suite-fixed/checks/value_test.go": "package checks\nimport \"testing\"\nfunc TestValue(t *testing.T) {}\n"} {
		file, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := file.Write([]byte(data)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	f.archive = zipBytes.Bytes()
	download := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.downloads++
		if r.Header.Get("Authorization") != "" {
			t.Error("installation token reached archive redirect")
		}
		w.Header().Set("Content-Type", "application/zip")
		w.Write(f.archive)
	}))
	t.Cleanup(download.Close)
	api := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.apiRequests++
		if r.Header.Get("X-GitHub-Api-Version") != "2026-03-10" {
			t.Error("missing pinned API version")
		}
		switch r.URL.Path {
		case "/app/installations/7/access_tokens":
			f.tokenRequests++
			token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			parts := strings.Split(token, ".")
			if len(parts) != 3 {
				t.Error("missing app assertion")
				w.WriteHeader(401)
				return
			}
			header, _ := base64.RawURLEncoding.DecodeString(parts[0])
			claimsJSON, _ := base64.RawURLEncoding.DecodeString(parts[1])
			signature, _ := base64.RawURLEncoding.DecodeString(parts[2])
			hash := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
			var algorithm struct {
				Algorithm string `json:"alg"`
			}
			var claims struct {
				Issuer  string `json:"iss"`
				Issued  int64  `json:"iat"`
				Expires int64  `json:"exp"`
			}
			json.Unmarshal(header, &algorithm)
			json.Unmarshal(claimsJSON, &claims)
			if algorithm.Algorithm != "RS256" || claims.Issuer != "fixture-client" || claims.Issued > time.Now().Unix() || claims.Expires > time.Now().Add(10*time.Minute).Unix() || rsa.VerifyPKCS1v15(&key.PublicKey, crypto.SHA256, hash[:], signature) != nil {
				t.Error("invalid app assertion")
			}
			var request struct {
				Repositories []string          `json:"repositories"`
				Permissions  map[string]string `json:"permissions"`
			}
			json.NewDecoder(r.Body).Decode(&request)
			if len(request.Repositories) != 1 || request.Repositories[0] != "suite" || len(request.Permissions) != 1 || request.Permissions["contents"] != "read" {
				t.Error("installation token was not repository-scoped and read-only")
			}
			w.WriteHeader(201)
			json.NewEncoder(w).Encode(map[string]any{"token": f.apiToken, "expires_at": time.Now().Add(time.Hour), "permissions": map[string]string{"contents": f.permission, "metadata": "read"}})
		case "/repos/owner/suite":
			if r.Header.Get("Authorization") != "" && r.Header.Get("Authorization") != "Bearer "+f.apiToken {
				t.Error("repository received app assertion")
			}
			json.NewEncoder(w).Encode(map[string]any{"id": f.repositoryID, "full_name": "owner/suite", "private": r.Header.Get("Authorization") != ""})
		default:
			if strings.HasPrefix(r.URL.Path, "/repos/owner/suite/commits/") {
				if r.Header.Get("Accept") != "application/vnd.github.sha" {
					t.Error("commit lookup requested unbounded diff metadata")
				}
				w.Write([]byte(f.commit))
				return
			}
			if strings.HasPrefix(r.URL.Path, "/repos/owner/suite/zipball/") {
				target := download.URL + "/owner/suite/legacy.zip/" + f.commit + "?token=fixture-signed-link"
				if f.redirected != "" {
					target = f.redirected
				}
				w.Header().Set("Location", target)
				w.WriteHeader(302)
				return
			}
			w.WriteHeader(500)
			w.Write([]byte("private diagnostic must never be returned"))
		}
	}))
	t.Cleanup(api.Close)
	roots := x509.NewCertPool()
	roots.AddCert(api.Certificate())
	roots.AddCert(download.Certificate())
	private := string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}))
	data, _ := json.Marshal(map[string]any{"repositories": []string{"owner/suite"}, "clientId": "fixture-client", "installationId": 7, "privateKeyPem": private})
	f.client = &github.Client{Secrets: credentials{data: data}, APIBase: api.URL, DownloadOrigin: download.URL, Network: networkguard.Loopback, RootCAs: roots}
	return f
}
func TestGitHubAppResolvesPrivateSourceAndDownloadsWithoutForwardingToken(t *testing.T) {
	f := fixture(t)
	access := sourceprovider.Access{Project: "alpha", Repository: "owner/suite", Credential: "app"}
	revision, err := f.client.Resolve(t.Context(), access, "heads/main")
	if err != nil {
		t.Fatal(err)
	}
	if revision.Commit != f.commit || revision.RepositoryID != "42" || !revision.Private {
		t.Fatal("resolved identity or commit was lost")
	}
	archive, err := f.client.Archive(t.Context(), access, revision)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := sourcebundle.ValidateRoot(archive.Content, "teststudio-suite.json", "unit", archive.Root)
	if err != nil {
		t.Fatal(err)
	}
	if archive.Root != "owner-suite-fixed" || len(snapshot.Catalog.Tests) != 1 || f.downloads != 1 || f.tokenRequests != 2 {
		t.Fatal("archive isolation or app authentication contract failed")
	}
}
func TestGitHubRejectsScopeEscalationIdentityChangesAndUnexpectedRedirects(t *testing.T) {
	for _, kind := range []string{"permission", "repository", "redirect", "pinned-commit"} {
		t.Run(kind, func(t *testing.T) {
			f := fixture(t)
			access := sourceprovider.Access{Project: "alpha", Repository: "owner/suite", Credential: "app"}
			if kind == "permission" {
				f.permission = "write"
				if _, err := f.client.Resolve(t.Context(), access, "heads/main"); err == nil {
					t.Fatal("overprivileged token accepted")
				}
				return
			}
			reference := "heads/main"
			if kind == "pinned-commit" {
				reference = strings.Repeat("b", 40)
				if _, err := f.client.Resolve(t.Context(), access, reference); err == nil {
					t.Fatal("pinned commit changed")
				}
				return
			}
			revision, err := f.client.Resolve(t.Context(), access, reference)
			if err != nil {
				t.Fatal(err)
			}
			if kind == "repository" {
				f.repositoryID = 99
			} else {
				f.redirected = "https://unapproved.invalid/owner/suite/legacy.zip/" + f.commit
			}
			if _, err := f.client.Archive(t.Context(), access, revision); err == nil {
				t.Fatal("changed source authority was accepted")
			}
			if f.downloads != 0 {
				t.Fatal("rejected source reached an archive endpoint")
			}
		})
	}
}
func TestGitHubPublicSourcesAndInvalidCredentialsRemainIsolated(t *testing.T) {
	f := fixture(t)
	access := sourceprovider.Access{Project: "alpha", Repository: "owner/suite"}
	revision, err := f.client.Resolve(t.Context(), access, "tags/v1")
	if err != nil || revision.Private || f.tokenRequests != 0 {
		t.Fatal("public source required app credentials", err)
	}
	access.Project, access.Credential = "other", "app"
	before := f.apiRequests
	if _, err := f.client.Resolve(t.Context(), access, "heads/main"); err == nil || strings.Contains(err.Error(), "fixture-installation-token") {
		t.Fatal("scoped credentials leaked")
	}
	if f.apiRequests != before {
		t.Fatal("foreign project credentials reached GitHub")
	}
	for _, name := range []string{"https://github.com/owner/suite", "owner/../suite", "owner/suite.git", "owner/suite?token=hidden"} {
		if err := f.client.Validate(sourceprovider.Access{Project: "alpha", Repository: name}, "heads/main"); err == nil {
			t.Fatal("credential-bearing or ambiguous repository accepted")
		}
	}
}
