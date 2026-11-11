package github_test

import (
	"crypto/rand"
	"crypto/rsa"
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
	"github.com/wtiger001/teststudio/internal/sourceprovider"
)

func TestGitHubMetadataReviewCommitIsAtomicAndRecoverable(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	private := string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}))
	data, _ := json.Marshal(map[string]any{"clientId": "fixture-client", "installationId": 7, "privateKeyPem": private, "repositories": []string{"owner/suite"}, "write": true})
	sha, treeSHA, created := strings.Repeat("a", 40), strings.Repeat("c", 40), strings.Repeat("b", 40)
	branch := ""
	files := map[string]string{}
	commits, reviews, tokens := 0, 0, 0
	reviewed := false
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/app/installations/7/access_tokens":
			tokens++
			var input struct {
				Permissions map[string]string `json:"permissions"`
			}
			json.NewDecoder(r.Body).Decode(&input)
			permissions := input.Permissions
			permissions["metadata"] = "read"
			w.WriteHeader(201)
			json.NewEncoder(w).Encode(map[string]any{"token": "fixture-token", "expires_at": time.Now().Add(time.Hour), "permissions": permissions})
		case r.URL.Path == "/repos/owner/suite":
			json.NewEncoder(w).Encode(map[string]any{"id": 42, "full_name": "owner/suite", "private": true})
		case r.URL.Path == "/repos/owner/suite/commits/heads/main":
			w.Write([]byte(sha))
		case strings.HasPrefix(r.URL.Path, "/repos/owner/suite/git/ref/heads/teststudio/"):
			if branch == "" {
				w.WriteHeader(404)
			} else {
				json.NewEncoder(w).Encode(map[string]any{"object": map[string]string{"sha": created}})
			}
		case r.URL.Path == "/repos/owner/suite/git/commits/"+sha:
			json.NewEncoder(w).Encode(map[string]any{"tree": map[string]string{"sha": treeSHA}})
		case r.URL.Path == "/repos/owner/suite/git/trees":
			var input struct {
				Base string `json:"base_tree"`
				Tree []struct {
					Path    string `json:"path"`
					Content string `json:"content"`
				} `json:"tree"`
			}
			json.NewDecoder(r.Body).Decode(&input)
			if input.Base != treeSHA || len(input.Tree) != 2 {
				t.Error("metadata files were not committed together against the reviewed tree")
			}
			for _, file := range input.Tree {
				files[file.Path] = file.Content
			}
			w.WriteHeader(201)
			json.NewEncoder(w).Encode(map[string]string{"sha": treeSHA})
		case r.URL.Path == "/repos/owner/suite/git/commits":
			commits++
			var input struct {
				Parents []string `json:"parents"`
			}
			json.NewDecoder(r.Body).Decode(&input)
			if len(input.Parents) != 1 || input.Parents[0] != sha {
				t.Error("metadata parent is not pinned")
			}
			w.WriteHeader(201)
			json.NewEncoder(w).Encode(map[string]string{"sha": created})
		case r.URL.Path == "/repos/owner/suite/git/refs":
			var input struct {
				Ref string `json:"ref"`
			}
			json.NewDecoder(r.Body).Decode(&input)
			branch = input.Ref
			w.WriteHeader(201)
			w.Write([]byte("{}"))
		case strings.HasPrefix(r.URL.Path, "/repos/owner/suite/contents/"):
			name := strings.TrimPrefix(r.URL.Path, "/repos/owner/suite/contents/")
			json.NewEncoder(w).Encode(map[string]string{"content": base64.StdEncoding.EncodeToString([]byte(files[name])), "encoding": "base64"})
		case r.URL.Path == "/repos/owner/suite/pulls":
			if r.Method == "POST" {
				reviews++
				reviewed = true
				w.WriteHeader(201)
				json.NewEncoder(w).Encode(map[string]string{"html_url": "https://github.com/owner/suite/pull/1"})
			} else if reviewed {
				json.NewEncoder(w).Encode([]any{map[string]string{"html_url": "https://github.com/owner/suite/pull/1"}})
			} else {
				w.Write([]byte("[]"))
			}
		default:
			w.WriteHeader(404)
		}
	}))
	defer server.Close()
	roots := x509.NewCertPool()
	roots.AddCert(server.Certificate())
	client := &github.Client{Secrets: credentials{data: data}, APIBase: server.URL, DownloadOrigin: server.URL, Network: networkguard.Loopback, RootCAs: roots}
	access := sourceprovider.Access{Project: "alpha", Repository: "owner/suite", Credential: "app"}
	base, err := client.Resolve(t.Context(), access, "heads/main")
	if err != nil {
		t.Fatal(err)
	}
	request := sourceprovider.WriteRequest{Base: base, Key: "operation-1", Message: "Update metadata", Files: map[string]string{"suite.json": "{}", "catalog.json": "{\"tests\":[]}"}}
	result, err := client.WriteMetadata(t.Context(), access, request)
	if err != nil || result.Commit != created || result.ReviewURL == "" {
		t.Fatal("write", err)
	}
	if _, err = client.WriteMetadata(t.Context(), access, request); err != nil {
		t.Fatal("recover", err)
	}
	if commits != 1 || reviews != 1 || tokens < 3 {
		t.Fatal("commit recovery duplicated writes or reused an expired token")
	}
	files["catalog.json"] = "remote change"
	if _, err = client.WriteMetadata(t.Context(), access, request); err == nil {
		t.Fatal("remote review branch changes were overwritten")
	}
	info, err := client.CredentialInfo(t.Context(), access)
	if err != nil || info.Fingerprint == "" || info.Renewal != "automatic installation token renewal" {
		t.Fatal("key rollover metadata", err)
	}
}
