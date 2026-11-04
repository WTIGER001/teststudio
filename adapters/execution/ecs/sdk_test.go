package ecsworker

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/storage"
)

type sdkFixture struct {
	t                 *testing.T
	account           string
	arn               string
	status            int
	calls             map[string]int
	registry          bool
	closed            int
	config, manifest  []byte
	configID, imageID string
}

func contentID(value []byte) string {
	sum := sha256.Sum256(value)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func sdkProtocol(t *testing.T) (*sdkFixture, Settings, *storage.Store, aws.CredentialsProvider) {
	t.Helper()
	guard, _, worker, _ := network(t)
	f := &sdkFixture{t: t, account: worker.Account, arn: "arn:aws:sts::123456789012:assumed-role/studio-controller/fixture", calls: map[string]int{}}
	configuration := map[string]any{"architecture": "arm64", "os": "linux", "config": map[string]any{"User": "1000:1000", "WorkingDir": "/suite", "Entrypoint": []string{"/runner/teststudio-go"}, "Volumes": map[string]any{"/tmp": map[string]any{}, "/output": map[string]any{}}, "Labels": map[string]string{"teststudio.runtime": "go-fixture-v1", "teststudio.runtime-kind": "source-base/v1", "teststudio.job-supervisor": "worker-exchange/v1", "teststudio.discovery-supervisor": "worker-exchange/v1", "teststudio.task-storage": "task-volumes/v1", "teststudio.work-item-bytes": "4194304", "teststudio.job-runtime-base": "sha256:" + strings.Repeat("b", 64), "teststudio.project": "project", "teststudio.installation": "installation", "teststudio.preparation": "preparation", "teststudio.source": "sha256:" + strings.Repeat("c", 64)}}, "rootfs": map[string]any{"type": "layers", "diff_ids": []string{"sha256:" + strings.Repeat("d", 64)}}}
	f.config, _ = json.Marshal(configuration)
	f.configID = contentID(f.config)
	f.manifest, _ = json.Marshal(map[string]any{"schemaVersion": 2, "mediaType": "application/vnd.oci.image.manifest.v1+json", "config": map[string]any{"mediaType": "application/vnd.oci.image.config.v1+json", "digest": f.configID, "size": len(f.config)}, "layers": []any{map[string]any{"mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "sha256:" + strings.Repeat("e", 64), "size": 100}}})
	f.imageID = contentID(f.manifest)
	db, err := storage.Open(t.Context(), filepath.Join(t.TempDir(), "sdk.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	credentials := aws.CredentialsProviderFunc(func(context.Context) (aws.Credentials, error) {
		return aws.Credentials{AccessKeyID: "fixture-access", SecretAccessKey: "fixture-secret", SessionToken: "fixture-session"}, nil
	})
	return f, Settings{Worker: worker, Network: guard.settings}, db, credentials
}

func response(request *http.Request, status int, contentType string, body []byte) *http.Response {
	return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": []string{contentType}}, Body: io.NopCloser(strings.NewReader(string(body))), ContentLength: int64(len(body)), Request: request}
}

func (f *sdkFixture) Do(request *http.Request) (*http.Response, error) {
	if request.Context().Err() != nil {
		return nil, request.Context().Err()
	}
	host := request.URL.Host
	f.calls[host]++
	switch host {
	case "sts.us-east-1.amazonaws.com", "api.ecr.us-east-1.amazonaws.com":
		authorization := request.Header.Get("Authorization")
		service := "sts"
		if strings.HasPrefix(host, "api.ecr") {
			service = "ecr"
		}
		if !strings.Contains(authorization, "Credential=fixture-access/") || !strings.Contains(authorization, "/us-east-1/"+service+"/aws4_request") || request.Header.Get("X-Amz-Security-Token") != "fixture-session" {
			f.t.Fatal("SDK did not sign with the explicit identity and region")
		}
		if f.status != 0 {
			return response(request, f.status, "application/json", []byte(`{"message":"private-sdk-canary"}`)), nil
		}
		if service == "sts" {
			request.ParseForm()
			if request.Form.Get("Action") != "GetCallerIdentity" {
				f.t.Fatal("unexpected STS operation")
			}
			return response(request, 200, "text/xml", []byte(`<GetCallerIdentityResponse xmlns="https://sts.amazonaws.com/doc/2011-06-15/"><GetCallerIdentityResult><Arn>`+f.arn+`</Arn><Account>`+f.account+`</Account><UserId>fixture</UserId></GetCallerIdentityResult></GetCallerIdentityResponse>`)), nil
		}
		body, _ := io.ReadAll(request.Body)
		var input struct {
			RegistryIDs []string `json:"registryIds"`
		}
		json.Unmarshal(body, &input)
		if len(input.RegistryIDs) != 1 || input.RegistryIDs[0] != "123456789012" {
			f.t.Fatal("ECR request omitted its registry scope")
		}
		encoded, _ := json.Marshal(map[string]any{"authorizationData": []any{map[string]any{"authorizationToken": base64.StdEncoding.EncodeToString([]byte("AWS:registry-password-canary")), "expiresAt": time.Now().Add(12 * time.Hour).Unix(), "proxyEndpoint": "https://123456789012.dkr.ecr.us-east-1.amazonaws.com"}}})
		return response(request, 200, "application/x-amz-json-1.1", encoded), nil
	case "123456789012.dkr.ecr.us-east-1.amazonaws.com":
		user, password, ok := request.BasicAuth()
		if !ok || user != "AWS" || password != "registry-password-canary" {
			f.t.Fatal("registry received incorrect credential kind")
		}
		if request.Header.Get("X-Amz-Security-Token") != "" {
			f.t.Fatal("AWS credentials reached the registry")
		}
		if request.URL.Path == "/v2/teststudio/worker/manifests/"+f.imageID {
			return response(request, 200, "application/vnd.oci.image.manifest.v1+json", f.manifest), nil
		}
		if request.URL.Path == "/v2/teststudio/worker/blobs/"+f.configID {
			value := response(request, 307, "application/octet-stream", nil)
			value.Header.Set("Location", "https://prod-us-east-1-starport-layer-bucket.s3.us-east-1.amazonaws.com/config?signature=private-url-canary")
			return value, nil
		}
		f.t.Fatal("unexpected registry path")
	case "prod-us-east-1-starport-layer-bucket.s3.us-east-1.amazonaws.com":
		if request.Header.Get("Authorization") != "" || request.Header.Get("X-Amz-Security-Token") != "" {
			f.t.Fatal("registry or AWS credentials reached a signed blob URL")
		}
		return response(request, 200, "application/octet-stream", f.config), nil
	default:
		f.t.Fatal("unexpected external endpoint", host)
	}
	return nil, errors.New("unexpected request")
}

func (f *sdkFixture) RoundTrip(request *http.Request) (*http.Response, error) { return f.Do(request) }
func (f *sdkFixture) CloseIdleConnections()                                   { f.closed++ }

func TestECSSDKFactoryBindsAccountRegionAndECRInspectionWithoutAmbientEndpoints(t *testing.T) {
	t.Setenv("AWS_REGION", "us-west-2")
	t.Setenv("AWS_ENDPOINT_URL", "https://foreign.invalid")
	t.Setenv("AWS_ENDPOINT_URL_STS", "https://foreign.invalid")
	t.Setenv("AWS_ENDPOINT_URL_ECR", "https://foreign.invalid")
	f, settings, db, credentials := sdkProtocol(t)
	runtime, err := settings.open(t.Context(), db, nil, credentials, f, f)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(runtime.Close)
	if len(f.calls) != 1 || f.calls["sts.us-east-1.amazonaws.com"] != 1 {
		t.Fatal("construction performed unexpected calls")
	}
	image := strings.TrimPrefix(registryOrigin(settings.Worker), "https://") + "/teststudio/worker@" + f.imageID
	inspected, err := runtime.Provider.InspectImage(t.Context(), image)
	if err != nil || inspected.Reference != image || inspected.Project != "project" || inspected.TaskStorage != hostedexecution.NonRootTaskStorage || !strings.HasPrefix(inspected.ExecutionProfile, "fargate-v2-") {
		t.Fatal("scoped ECR image inspection failed", err)
	}
	if _, err := runtime.Provider.InspectImage(t.Context(), image); err != nil || f.calls["api.ecr.us-east-1.amazonaws.com"] != 1 {
		t.Fatal("repeated image inspection repeated ECR authorization", err)
	}
	if _, err := runtime.Provider.InspectImage(t.Context(), strings.Replace(image, "/teststudio/", "/teststudio-neighbor/", 1)); err == nil {
		t.Fatal("neighboring registry namespace admitted")
	}
	runtime.Close()
	runtime.Close()
	if f.closed != 2 {
		t.Fatal("runtime did not close each owned HTTP pool exactly once", f.closed)
	}
}

func TestECSSDKFactoryRejectsWrongAccountAndDoesNotRetryPrivateProviderErrors(t *testing.T) {
	for _, mode := range []string{"wrong-account", "wrong-arn", "private-error"} {
		t.Run(mode, func(t *testing.T) {
			f, settings, db, credentials := sdkProtocol(t)
			if mode == "wrong-account" {
				f.account = "999999999999"
			}
			if mode == "wrong-arn" {
				f.arn = "arn:aws:sts::999999999999:assumed-role/foreign/fixture"
			}
			if mode == "private-error" {
				f.status = 503
			}
			runtime, err := settings.open(t.Context(), db, nil, credentials, f, f)
			if runtime != nil || err == nil || strings.Contains(err.Error(), "canary") || f.calls["sts.us-east-1.amazonaws.com"] != 1 || len(f.calls) != 1 {
				t.Fatal("invalid caller passed or SDK retried", err)
			}
		})
	}
}

func TestECSSDKHTTPRejectsEndpointDrift(t *testing.T) {
	f, _, _, _ := sdkProtocol(t)
	client := &sdkHTTP{base: f, region: "us-east-1"}
	for _, origin := range []string{"http://sts.us-east-1.amazonaws.com", "https://sts.us-west-2.amazonaws.com", "https://sts.us-east-1.amazonaws.com:443", "https://foreign.invalid"} {
		parsed, _ := url.Parse(origin)
		_, err := client.Do(&http.Request{URL: parsed, Header: http.Header{}})
		if !errors.Is(err, hostedexecution.ErrConfiguration) {
			t.Fatal("unapproved SDK endpoint admitted", err)
		}
	}
	if len(f.calls) != 0 {
		t.Fatal("rejected endpoints performed I/O")
	}
}

type sdkRequest func(*http.Request) (*http.Response, error)

func (f sdkRequest) Do(request *http.Request) (*http.Response, error) { return f(request) }

type closedSDKBody struct {
	io.Reader
	closed bool
}

func (b *closedSDKBody) Close() error { b.closed = true; return nil }

func TestECSSDKHTTPClosesRedirectAndOversizedResponses(t *testing.T) {
	for _, status := range []int{302, 200} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			body := &closedSDKBody{Reader: strings.NewReader("private-canary")}
			base := sdkRequest(func(request *http.Request) (*http.Response, error) {
				value := response(request, status, "text/plain", nil)
				value.Body = body
				if status == 200 {
					value.ContentLength = 9 << 20
				}
				return value, nil
			})
			request, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://sts.us-east-1.amazonaws.com", nil)
			value, err := (&sdkHTTP{base: base, region: "us-east-1"}).Do(request)
			if value != nil || !errors.Is(err, hostedexecution.ErrUnavailable) || !body.closed {
				t.Fatal("SDK response escaped its transport bound", err)
			}
		})
	}
}
