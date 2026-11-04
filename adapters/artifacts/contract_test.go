package artifacts_test

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blockblob"
	"github.com/aws/aws-sdk-go-v2/aws"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/wtiger001/teststudio/adapters/artifacts/azureblob"
	"github.com/wtiger001/teststudio/adapters/artifacts/filesystem"
	artifactS3 "github.com/wtiger001/teststudio/adapters/artifacts/s3"
	"github.com/wtiger001/teststudio/internal/artifacts"
	"github.com/wtiger001/teststudio/internal/evidence"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/testkit"
)

type localCredential struct{}

func (localCredential) GetToken(context.Context, policy.TokenRequestOptions) (azcore.AccessToken, error) {
	return azcore.AccessToken{Token: "local-protocol-token", ExpiresOn: time.Now().Add(time.Hour)}, nil
}

type protocolFixture struct {
	mu       sync.Mutex
	objects  map[string][]byte
	requests atomic.Int32
	fail     atomic.Bool
	provider string
}

func (f *protocolFixture) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.requests.Add(1)
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.provider == "s3" && !strings.HasPrefix(r.Header.Get("Authorization"), "AWS4-HMAC-SHA256 ") {
		w.WriteHeader(401)
		return
	}
	if f.provider == "azure" && r.Header.Get("Authorization") != "Bearer local-protocol-token" {
		w.WriteHeader(401)
		return
	}
	failure := func(code string) {
		w.Header().Set("Content-Type", "application/xml")
		w.Header().Set("x-ms-error-code", code)
		w.WriteHeader(404)
		fmt.Fprintf(w, "<Error><Code>%s</Code><Message>Local protocol fixture</Message></Error>", code)
	}
	if f.fail.Load() {
		if f.provider == "s3" {
			failure("NoSuchBucket")
		} else {
			failure("ContainerNotFound")
		}
		return
	}
	w.Header().Set("ETag", `"local-object"`)
	w.Header().Set("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
	w.Header().Set("x-ms-request-id", "local-request")
	switch r.Method {
	case http.MethodPut:
		data, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(500)
			return
		}
		if f.provider == "s3" {
			sum := sha256.Sum256(data)
			expected := base64.StdEncoding.EncodeToString(sum[:])
			if r.Header.Get("x-amz-checksum-sha256") != expected || r.Header.Get("x-amz-expected-bucket-owner") != "123456789012" {
				w.WriteHeader(400)
				return
			}
			w.Header().Set("x-amz-checksum-sha256", expected)
		} else {
			if checksum := r.Header.Get("x-ms-content-crc64"); checksum != "" {
				w.Header().Set("x-ms-content-crc64", checksum)
			}
		}
		f.objects[r.URL.Path] = data
		if f.provider == "azure" {
			w.WriteHeader(201)
		} else {
			w.WriteHeader(200)
		}
	case http.MethodGet:
		data, ok := f.objects[r.URL.Path]
		if !ok {
			if f.provider == "s3" {
				failure("NoSuchKey")
			} else {
				failure("BlobNotFound")
			}
			return
		}
		w.Header().Set("Content-Length", fmt.Sprint(len(data)))
		w.WriteHeader(200)
		_, _ = w.Write(data)
	case http.MethodDelete:
		delete(f.objects, r.URL.Path)
		if f.provider == "azure" {
			w.WriteHeader(202)
		} else {
			w.WriteHeader(204)
		}
	default:
		w.WriteHeader(405)
	}
}

func localStore(t *testing.T, driver string) (artifacts.Store, *protocolFixture) {
	t.Helper()
	if driver == "filesystem" {
		s, err := filesystem.New(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { s.Close() })
		return s, nil
	}
	fixture := &protocolFixture{objects: map[string][]byte{}, provider: driver}
	server := httptest.NewTLSServer(fixture)
	t.Cleanup(server.Close)
	if driver == "s3" {
		client := awss3.New(awss3.Options{BaseEndpoint: aws.String(server.URL), Region: "us-test-1", UsePathStyle: true, HTTPClient: server.Client(), RetryMaxAttempts: 1, Credentials: aws.CredentialsProviderFunc(func(context.Context) (aws.Credentials, error) {
			return aws.Credentials{AccessKeyID: "local-protocol-id", SecretAccessKey: "local-protocol-secret"}, nil
		}), ResponseChecksumValidation: aws.ResponseChecksumValidationWhenRequired})
		s, err := artifactS3.New(client, "evidence", "installation", "123456789012", "")
		if err != nil {
			t.Fatal(err)
		}
		return s, fixture
	}
	s, err := azureblob.New(server.URL, localCredential{}, "evidence", "installation", &blockblob.ClientOptions{ClientOptions: azcore.ClientOptions{Transport: server.Client(), Retry: policy.RetryOptions{MaxRetries: -1}}})
	if err != nil {
		t.Fatal(err)
	}
	return s, fixture
}

func TestArtifactStoresUseBoundedPrivateObjectsAndConsistentMissingErrors(t *testing.T) {
	for _, driver := range []string{"filesystem", "s3", "azure"} {
		t.Run(driver, func(t *testing.T) {
			s, fixture := localStore(t, driver)
			key := "project/run/screenshot %25.png"
			data := "new independent evidence"
			object, err := s.Put(t.Context(), key, strings.NewReader(data), 100)
			sum := sha256.Sum256([]byte(data))
			if err != nil || object.Size != int64(len(data)) || object.Checksum != hex.EncodeToString(sum[:]) {
				t.Fatal("upload did not preserve bytes", object, err)
			}
			reader, err := s.Open(t.Context(), key)
			if err != nil {
				t.Fatal(err)
			}
			read, err := io.ReadAll(reader)
			reader.Close()
			if err != nil || string(read) != data {
				t.Fatal("download changed evidence", string(read), err)
			}
			if _, err := s.Put(t.Context(), "project/run/large", strings.NewReader(data), 2); err == nil {
				t.Fatal("oversized upload accepted")
			}
			if _, err := s.Open(t.Context(), "project/run/large"); !errors.Is(err, artifacts.ErrNotFound) {
				t.Fatal("failed upload became visible", err)
			}
			for _, invalid := range []string{"../escape", "/absolute", "a//b", "a/../b", "bad\\path", "bad\x00key", "."} {
				if _, err := s.Put(t.Context(), invalid, strings.NewReader("x"), 1); err == nil {
					t.Fatal("unsafe key accepted", invalid)
				}
				if _, err := s.Open(t.Context(), invalid); err == nil {
					t.Fatal("unsafe read accepted")
				}
				if err := s.Delete(t.Context(), invalid); err == nil {
					t.Fatal("unsafe delete accepted")
				}
			}
			if fixture != nil {
				fixture.mu.Lock()
				if len(fixture.objects) != 1 {
					t.Error("unbounded or invalid writes reached object storage")
				}
				for path := range fixture.objects {
					if !strings.HasPrefix(path, "/evidence/installation/project/run/") {
						t.Error("installation prefix missing", path)
					}
				}
				fixture.mu.Unlock()
				fixture.fail.Store(true)
				if _, err := s.Open(t.Context(), key); err == nil || errors.Is(err, artifacts.ErrNotFound) {
					t.Fatal("missing storage infrastructure looked like missing evidence", err)
				}
				fixture.fail.Store(false)
			}
			if err := s.Delete(t.Context(), key); err != nil {
				t.Fatal(err)
			}
			if err := s.Delete(t.Context(), key); err != nil {
				t.Fatal("duplicate deletion failed", err)
			}
			if _, err := s.Open(t.Context(), key); !errors.Is(err, artifacts.ErrNotFound) {
				t.Fatal("deleted evidence remained readable", err)
			}
			empty, err := s.Put(t.Context(), "project/run/empty", strings.NewReader(""), 1)
			if err != nil || empty.Size != 0 {
				t.Fatal("empty evidence upload failed", empty, err)
			}
			ctx, cancel := context.WithCancel(t.Context())
			cancel()
			if _, err := s.Put(ctx, "project/run/cancelled", strings.NewReader("x"), 1); err == nil {
				t.Fatal("cancelled upload accepted")
			}
		})
	}
}

func TestArtifactStoresShareQuotaPinsRetentionAndDurableManifests(t *testing.T) {
	for _, driver := range []string{"filesystem", "s3", "azure"} {
		t.Run(driver, func(t *testing.T) {
			blobs, _ := localStore(t, driver)
			db, err := testkit.Open(t, filepath.Join(t.TempDir(), "evidence.db"))
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { db.Close() })
			s := &evidence.Service{Store: db, Blobs: blobs}
			a := evidence.Artifact{ID: "retained", Project: "project", Run: "run", Test: "CHECK-1", Attempt: 1, Kind: "attachment", Filename: "result.txt", ContentType: "text/plain", ExpiresAt: time.Now().Add(-time.Hour)}
			for range 2 {
				if _, err := s.SaveBytes(t.Context(), a, []byte("retained evidence"), 32); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := s.SaveBytes(t.Context(), evidence.Artifact{ID: "over-quota", Project: "project", Run: "run", Kind: "attachment", Filename: "other.txt"}, []byte("another large attachment"), 32); !errors.Is(err, evidence.ErrQuota) {
				t.Fatal("duplicate admission changed quota", err)
			}
			run, _ := storage.Put("runs", "run", "project", 0, map[string]any{"projectId": "project", "finishedAt": time.Now().UTC(), "outcome": "failed"})
			if err := db.Commit(t.Context(), run); err != nil {
				t.Fatal(err)
			}
			if err := s.SavePin(t.Context(), "tester", "project", "run", true, nil, "Retain adapter investigation", 0); err != nil {
				t.Fatal(err)
			}
			if err := s.EnsureRetentionIndex(t.Context(), 10); err != nil {
				t.Fatal(err)
			}
			if err := s.ProcessRetention(t.Context(), 10); err != nil {
				t.Fatal(err)
			}
			restarted := &evidence.Service{Store: db, Blobs: blobs}
			if err := restarted.Sweep(t.Context(), 10); err != nil {
				t.Fatal(err)
			}
			_, reader, err := restarted.Open(t.Context(), "project", "run", a.ID)
			if err != nil {
				t.Fatal("pin did not preserve content", err)
			}
			read, err := io.ReadAll(reader)
			reader.Close()
			if err != nil || string(read) != "retained evidence" {
				t.Fatal("pinned bytes changed")
			}
			_, pin, err := restarted.ReadPin(t.Context(), "project", "run")
			if err != nil {
				t.Fatal(err)
			}
			if err := restarted.SavePin(t.Context(), "tester", "project", "run", false, nil, "Finish adapter investigation", pin.Revision); err != nil {
				t.Fatal(err)
			}
			if err := restarted.Sweep(t.Context(), 10); err != nil {
				t.Fatal(err)
			}
			row, err := db.Get(t.Context(), evidence.Bucket("run"), a.ID)
			if err != nil {
				t.Fatal(err)
			}
			expired, err := storage.Decode[evidence.Artifact](row)
			if err != nil || expired.State != "expired" {
				t.Fatal("expiry lost the durable manifest", expired, err)
			}
			for _, scope := range [][2]string{{"artifact-quotas", "run"}, {"artifact-project-quotas", "project"}, {"artifact-test-quotas", "run/CHECK-1"}} {
				row, err := db.Get(t.Context(), scope[0], scope[1])
				if err != nil {
					t.Fatal(err)
				}
				quota, err := storage.Decode[evidence.Quota](row)
				if err != nil || quota.Bytes != 0 {
					t.Fatal("cleanup did not release quota", scope, quota, err)
				}
			}
			rows, _ := db.All(t.Context(), evidence.Bucket("run"), "project")
			if len(rows) != 1 {
				t.Fatal("replay duplicated attachments")
			}
		})
	}
}
