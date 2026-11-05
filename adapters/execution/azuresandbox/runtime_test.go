package azuresandbox

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func runtimeSettings() Settings {
	c := fixtureConfig()
	identity := "/subscriptions/" + c.Subscription + "/resourceGroups/studio-identities/providers/Microsoft.ManagedIdentity/userAssignedIdentities/"
	return Settings{Worker: c, Mode: "execution", TenantID: "33333333-3333-3333-3333-333333333333", ControlClientID: "11111111-1111-1111-1111-111111111111", RegistryClientID: "22222222-2222-2222-2222-222222222222", PullIdentity: identity + "studio-pull", Timer: TimerSettings{ResourceGroup: "studio-timers", NamePrefix: "studio", StopIdentity: identity + "studio-stop"}}
}

type runtimeCredential struct {
	token  string
	scopes []string
}

func (c *runtimeCredential) GetToken(_ context.Context, r policy.TokenRequestOptions) (azcore.AccessToken, error) {
	c.scopes = append(c.scopes, r.Scopes...)
	return azcore.AccessToken{Token: c.token, ExpiresOn: time.Now().Add(time.Hour)}, nil
}

type runtimeTransport func(*http.Request) (*http.Response, error)

func (f runtimeTransport) Do(r *http.Request) (*http.Response, error)        { return f(r) }
func (f runtimeTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestSandboxRuntimeLoadsIndependentBoundedSettings(t *testing.T) {
	s := runtimeSettings()
	data, _ := json.Marshal(s)
	lookup := func(name string) (string, bool) {
		if name != "TESTSTUDIO_SANDBOX" {
			t.Fatal("unexpected configuration reference")
		}
		return string(data), true
	}
	loaded, err := LoadSettings("", "TESTSTUDIO_SANDBOX", lookup)
	if err != nil || loaded == nil || hash(*loaded) != hash(s) {
		t.Fatal("independent runtime settings failed", err)
	}
	file := filepath.Join(t.TempDir(), "sandbox.json")
	if err := os.WriteFile(file, data, 0600); err != nil {
		t.Fatal(err)
	}
	loaded, err = LoadSettings(file, "", nil)
	if err != nil || hash(*loaded) != hash(s) {
		t.Fatal("file and environment differ", err)
	}
	if loaded, err := LoadSettings("", "", nil); loaded != nil || err != nil {
		t.Fatal("absent sandbox config changed startup", err)
	}
	data = []byte(strings.Replace(string(data), `"worker":{`, `"worker":{"workerEnvironment":"legacy-jobs",`, 1))
	if _, err := LoadSettings("", "TESTSTUDIO_SANDBOX", lookup); err == nil {
		t.Fatal("Jobs environment fields entered sandbox settings")
	}
	if _, err := LoadSettings(file, "TESTSTUDIO_SANDBOX", lookup); err == nil {
		t.Fatal("multiple configuration sources accepted")
	}
	if _, err := LoadSettings("", "TESTSTUDIO_SANDBOX", nil); err == nil {
		t.Fatal("missing lookup accepted")
	}
}

func TestSandboxRuntimeRejectsInvalidBindingsBeforeCredentials(t *testing.T) {
	for _, mode := range []string{"mode", "same-client", "zero-client", "tenant", "pull-stop", "timer-group", "redirect", "admission", "import-admission", "discovery-cleanup"} {
		t.Run(mode, func(t *testing.T) {
			f := newControllerFixture(t, "go", "")
			s := runtimeSettings()
			options := RuntimeOptions{Settings: s, Exchange: f.exchange, Admission: f.controls}
			switch mode {
			case "mode":
				options.Settings.Mode = "both"
			case "same-client":
				options.Settings.RegistryClientID = s.ControlClientID
			case "zero-client":
				options.Settings.ControlClientID = "00000000-0000-0000-0000-000000000000"
			case "tenant":
				options.Settings.TenantID = "not-an-installation"
			case "pull-stop":
				options.Settings.Timer.StopIdentity = s.PullIdentity
			case "timer-group":
				options.Settings.Timer.ResourceGroup = s.Worker.ResourceGroup
			case "redirect":
				options.Settings.BlobRedirectOrigins = []string{"https://foreign.example"}
			case "admission":
				options.Admission = nil
			case "import-admission":
				options.Settings.Mode = "discovery"
			case "discovery-cleanup":
				options.Settings.Mode = "discovery"
				options.ImageAdmission = imageAdmission{}
				f.exchange.AuthorizeDiscoveryCleanup = nil
			}
			options.Credentials = func(string) (azcore.TokenCredential, error) {
				t.Error("invalid configuration constructed credentials")
				return nil, errors.New("credential-canary")
			}
			if runtime, err := OpenRuntime(options); err != hostedexecution.ErrConfiguration || runtime != nil {
				t.Fatal("invalid native binding was accepted", err)
			}
		})
	}
}

func TestSandboxRuntimeAssemblesConcreteClientsWithExplicitCredentials(t *testing.T) {
	f := newControllerFixture(t, "go", "")
	s := runtimeSettings()
	control := &runtimeCredential{token: "native-control-canary"}
	reader := &runtimeCredential{token: "registry-reader-canary"}
	selected := []string{}
	dataCalls, armCalls := 0, 0
	data := runtimeTransport(func(r *http.Request) (*http.Response, error) {
		dataCalls++
		if r.Method != "GET" || r.URL.Host != "management.westus2.azuredevcompute.io" || r.URL.Path != s.Worker.path()+"/sandboxes/selected-sandbox" || r.Header.Get("Authorization") != "Bearer "+control.token {
			t.Fatal("data-plane identity or path changed")
		}
		return jsonResponse(instanceBody(t, f.plan, "selected-sandbox")), nil
	})
	spec := DeadlineSpec{Config: s.Worker, Proof: f.plan.proof}
	arm := runtimeTransport(func(r *http.Request) (*http.Response, error) {
		armCalls++
		if r.Method != "GET" || r.URL.Host != "management.azure.com" || r.URL.Path != s.deadlineConfig().resourceID(spec) || r.URL.RawQuery != "api-version=2019-05-01" || r.Header.Get("Authorization") != "Bearer "+control.token {
			t.Fatal("timer identity or path changed")
		}
		return response(404, `{"error":{"code":"ResourceNotFound"}}`), nil
	})
	runtime, err := OpenRuntime(RuntimeOptions{Settings: s, Exchange: f.exchange, Admission: f.controls, Transport: data, ARMTransport: arm, Credentials: func(id string) (azcore.TokenCredential, error) {
		selected = append(selected, id)
		if id == s.ControlClientID {
			return control, nil
		}
		if id == s.RegistryClientID {
			return reader, nil
		}
		return nil, errors.New("unexpected credential selection")
	}})
	if err != nil {
		t.Fatal(err)
	}
	defer runtime.Close()
	if len(selected) != 2 || selected[0] != s.ControlClientID || selected[1] != s.RegistryClientID || len(control.scopes) != 0 || len(reader.scopes) != 0 || dataCalls+armCalls != 0 {
		t.Fatal("construction obtained tokens or discovered another credential")
	}
	if runtime.Provider.controller.instances != runtime.client || runtime.Provider.controller.commands != runtime.client || runtime.Provider.controller.deadlines != runtime.deadlines || runtime.Provider.inspector != runtime.registry || runtime.Provider.images.writer || runtime.Provider.mode != "execution" {
		t.Fatal("native factory did not select concrete supervised clients")
	}
	actual, err := runtime.client.Read(t.Context(), "selected-sandbox")
	if err != nil || f.plan.Verify(actual) != nil {
		t.Fatal("concrete data-plane client failed", err)
	}
	if _, err := runtime.deadlines.get(t.Context(), spec); err != hostedexecution.ErrNotFound {
		t.Fatal("concrete timer SDK lost authoritative absence", err)
	}
	if dataCalls != 1 || armCalls != 1 || len(control.scopes) != 2 || control.scopes[0] != TokenScope || control.scopes[1] != "https://management.core.windows.net//.default" || len(reader.scopes) != 0 {
		t.Fatal("native clients mixed credential audiences or retried")
	}
	runtime.Close()
	runtime.Close()
	if _, err := runtime.Provider.InspectImage(t.Context(), f.plan.proof.Image); err != hostedexecution.ErrUnavailable || dataCalls+armCalls != 2 {
		t.Fatal("closed runtime can acquire credentials", err)
	}
	if _, err := runtime.Provider.Observe(t.Context(), f.plan.proof.Reference); err != hostedexecution.ErrUnavailable {
		t.Fatal("closed runtime observed work", err)
	}
	if _, err := runtime.Provider.Inventory(t.Context(), f.plan.proof.Reference.Installation, f.plan.proof.Reference.Profile); err != hostedexecution.ErrUnavailable {
		t.Fatal("closed runtime enumerated work", err)
	}
	for _, err := range []error{runtime.Provider.Ensure(t.Context(), hostedexecution.Job{}), runtime.Provider.Stop(t.Context(), f.plan.proof.Reference), runtime.Provider.Remove(t.Context(), f.plan.proof.Reference)} {
		if err != hostedexecution.ErrUnavailable {
			t.Fatal("closed runtime accepted mutation", err)
		}
	}
}

func TestSandboxRuntimeCredentialFailureRemainsRedacted(t *testing.T) {
	f := newControllerFixture(t, "go", "")
	runtime, err := OpenRuntime(RuntimeOptions{Settings: runtimeSettings(), Exchange: f.exchange, Admission: f.controls, Credentials: func(string) (azcore.TokenCredential, error) {
		return nil, errors.New("private-identity-failure-canary")
	}})
	if runtime != nil || err != hostedexecution.ErrUnavailable || strings.Contains(err.Error(), "canary") {
		t.Fatal("credential construction exposed diagnostics", err)
	}
}
