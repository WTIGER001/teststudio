package gitlab_test

import (
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	localsecrets "github.com/wtiger001/teststudio/adapters/secrets/local"
	"github.com/wtiger001/teststudio/adapters/sources/gitlab"
	"github.com/wtiger001/teststudio/internal/networkguard"
	"github.com/wtiger001/teststudio/internal/sourceprovider"
)

func TestGitLabMetadataCommitRecoversWithoutDuplicateWrites(t *testing.T) {
	sha, created := strings.Repeat("a", 40), strings.Repeat("b", 40)
	branch := ""
	contents := map[string]string{}
	commits, reviews := 0, 0
	reviewed := false
	serverOrigin := ""
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v4/projects/group/subgroup/suite":
			json.NewEncoder(w).Encode(map[string]any{"id": 42, "path_with_namespace": "group/subgroup/suite", "visibility": "private"})
		case r.URL.Path == "/api/v4/projects/42/repository/branches/main":
			json.NewEncoder(w).Encode(map[string]any{"commit": map[string]string{"id": sha}})
		case r.URL.Path == "/api/v4/projects/42/repository/branches":
			if branch == "" {
				w.Write([]byte("[]"))
			} else {
				json.NewEncoder(w).Encode([]any{map[string]any{"name": branch, "commit": map[string]string{"id": created}}})
			}
		case strings.HasPrefix(r.URL.Path, "/api/v4/projects/42/repository/files/"):
			name := strings.TrimPrefix(r.URL.Path, "/api/v4/projects/42/repository/files/")
			value, ok := contents[name]
			if !ok {
				w.WriteHeader(404)
				return
			}
			json.NewEncoder(w).Encode(map[string]string{"file_path": name, "encoding": "base64", "content": base64.StdEncoding.EncodeToString([]byte(value))})
		case r.URL.Path == "/api/v4/projects/42/repository/commits":
			var input struct {
				Branch  string `json:"branch"`
				Start   string `json:"start_sha"`
				Actions []struct {
					Path    string `json:"file_path"`
					Content string `json:"content"`
				} `json:"actions"`
			}
			if json.NewDecoder(r.Body).Decode(&input) != nil || input.Start != sha || !strings.HasPrefix(input.Branch, "teststudio/") {
				t.Error("commit was not based on its reviewed revision")
			}
			branch = input.Branch
			for _, action := range input.Actions {
				contents[action.Path] = action.Content
			}
			commits++
			json.NewEncoder(w).Encode(map[string]string{"id": created})
		case r.URL.Path == "/api/v4/projects/42/merge_requests":
			if r.Method == "POST" {
				reviews++
				reviewed = true
				json.NewEncoder(w).Encode(map[string]string{"web_url": serverOrigin + "/group/subgroup/suite/-/merge_requests/1"})
			} else if reviewed {
				json.NewEncoder(w).Encode([]any{map[string]string{"web_url": serverOrigin + "/group/subgroup/suite/-/merge_requests/1"}})
			} else {
				w.Write([]byte("[]"))
			}
		default:
			w.WriteHeader(404)
		}
	}))
	defer server.Close()
	serverOrigin = server.URL
	roots := x509.NewCertPool()
	roots.AddCert(server.Certificate())
	client := &gitlab.Client{Secrets: &credentials{token: "fixture"}, BaseURL: server.URL, Network: networkguard.Loopback, RootCAs: roots}
	access := sourceprovider.Access{Project: "alpha", Repository: "group/subgroup/suite", Credential: "access"}
	base, err := client.Resolve(t.Context(), access, "heads/main")
	if err != nil {
		t.Fatal(err)
	}
	request := sourceprovider.WriteRequest{Base: base, Key: "operation-1", Message: "Update suite metadata", Files: map[string]string{"catalog.json": "{\"tests\":[]}", "suite.json": "{}"}}
	if result, err := client.WriteMetadata(t.Context(), access, request); err != nil || result.Commit != created || result.ReviewURL == "" {
		t.Fatal("metadata write failed", err)
	}
	if commits != 1 || reviews != 1 {
		t.Fatal("initial write did not commit both files once")
	}
	if _, err = client.WriteMetadata(t.Context(), access, request); err != nil {
		t.Fatal("metadata recovery failed", err)
	}
	if commits != 1 || reviews != 1 {
		t.Fatal("recovery duplicated a commit or merge request")
	}
	contents["catalog.json"] = "remote change"
	if _, err = client.WriteMetadata(t.Context(), access, request); err == nil {
		t.Fatal("remote review branch edit was overwritten")
	}
}
func TestGitLabRotationStoresReplacementWithoutChangingBindings(t *testing.T) {
	file := filepath.Join(t.TempDir(), "access.json")
	old := map[string]any{"token": "old-fixture", "tokenType": "project", "repositories": []string{"group/subgroup/suite"}}
	data, _ := json.Marshal(old)
	if err := os.WriteFile(file, data, 0600); err != nil {
		t.Fatal(err)
	}
	secrets, err := localsecrets.New([]localsecrets.Binding{{ID: "access", Project: "alpha", Purpose: sourceprovider.CredentialPurpose, File: file}})
	if err != nil {
		t.Fatal(err)
	}
	rotations := 0
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v4/projects/group/subgroup/suite":
			json.NewEncoder(w).Encode(map[string]any{"id": 42, "path_with_namespace": "group/subgroup/suite"})
		case "/api/v4/projects/42/access_tokens/self/rotate":
			rotations++
			if r.Header.Get("PRIVATE-TOKEN") != "old-fixture" {
				t.Error("rotation used the wrong current token")
			}
			json.NewEncoder(w).Encode(map[string]string{"token": "new-fixture", "expires_at": "2100-01-01"})
		default:
			w.WriteHeader(404)
		}
	}))
	defer server.Close()
	roots := x509.NewCertPool()
	roots.AddCert(server.Certificate())
	client := &gitlab.Client{Secrets: secrets, BaseURL: server.URL, Network: networkguard.Loopback, RootCAs: roots}
	access := sourceprovider.Access{Project: "alpha", Repository: "group/subgroup/suite", Credential: "access"}
	if err = client.RotateCredential(t.Context(), access); err != nil {
		t.Fatal(err)
	}
	current, err := secrets.Resolve(t.Context(), "alpha", "access", sourceprovider.CredentialPurpose)
	if err != nil {
		t.Fatal(err)
	}
	var value map[string]any
	json.Unmarshal(current.Bytes, &value)
	if value["token"] != "new-fixture" || rotations != 1 {
		t.Fatal("rotation did not persist its replacement exactly once")
	}
	info, err := client.CredentialInfo(t.Context(), access)
	if err != nil || info.ExpiresAt == nil || !info.CanRotate {
		t.Fatal("rotation metadata", err)
	}
}
