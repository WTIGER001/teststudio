package azureworker

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v4"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func TestAzureObserverHostAuthenticationRequiresExplicitDisabledSelectedPlatform(t *testing.T) {
	c, s, f, _ := observerHostTest(t)
	if err := verifyObserverHostAuthentication(s, f.auth); err != nil {
		t.Fatal(err)
	}
	actual := clone(f.auth)
	actual.ID, actual.Type = to.Ptr(strings.ToLower(value(actual.ID))), to.Ptr(strings.ToLower(value(actual.Type)))
	if err := verifyObserverHostAuthentication(s, actual); err != nil {
		t.Fatal("resource identifier case changed the selected authentication", err)
	}
	for name, check := range map[string]struct {
		change func(*a.AuthConfig)
		want   error
	}{
		"enabled":            {change: func(auth *a.AuthConfig) { auth.Properties.Platform.Enabled = to.Ptr(true) }, want: hostedexecution.ErrOwnership},
		"unknown":            {change: func(auth *a.AuthConfig) { auth.Properties.Platform.Enabled = nil }, want: hostedexecution.ErrUnavailable},
		"missing-platform":   {change: func(auth *a.AuthConfig) { auth.Properties.Platform = nil }, want: hostedexecution.ErrUnavailable},
		"missing-properties": {change: func(auth *a.AuthConfig) { auth.Properties = nil }, want: hostedexecution.ErrUnavailable},
		"missing-id":         {change: func(auth *a.AuthConfig) { auth.ID = nil }, want: hostedexecution.ErrUnavailable},
		"missing-name":       {change: func(auth *a.AuthConfig) { auth.Name = nil }, want: hostedexecution.ErrUnavailable},
		"missing-type":       {change: func(auth *a.AuthConfig) { auth.Type = nil }, want: hostedexecution.ErrUnavailable},
		"foreign-app": {change: func(auth *a.AuthConfig) {
			auth.ID = to.Ptr(c.resourceID("Microsoft.App", "containerApps", "foreign") + "/authConfigs/current")
		}, want: hostedexecution.ErrOwnership},
		"foreign-config": {change: func(auth *a.AuthConfig) { auth.Name = to.Ptr("foreign") }, want: hostedexecution.ErrOwnership},
		"wrong-type":     {change: func(auth *a.AuthConfig) { auth.Type = to.Ptr("Microsoft.App/containerApps") }, want: hostedexecution.ErrOwnership},
	} {
		t.Run(name, func(t *testing.T) {
			actual := clone(f.auth)
			check.change(&actual)
			requireReadDenyError(t, verifyObserverHostAuthentication(s, actual), check.want)
		})
	}
	definition, err := s.AuthDefinition(c)
	if err != nil || value(definition.Properties.Platform.Enabled) || fingerprint(document(definition)) != fingerprint(map[string]any{"properties": map[string]any{"platform": map[string]any{"enabled": false}}}) {
		t.Fatal("authentication definition widened the fixed disabled contract", err)
	}
	changed := s
	changed.Runtime.DatabaseSecretURL = "private-invalid-configuration-canary"
	_, err = changed.AuthDefinition(c)
	requireReadDenyError(t, err, hostedexecution.ErrConfiguration)
	if f.reads != 0 || f.parent.bootstrapReads != 0 {
		t.Fatal("authentication definition performed observation")
	}
}

func TestAzureObserverHostAuthenticationSDKRestrictsSelectionAndDropsProviderDetails(t *testing.T) {
	c, s, f, _ := observerHostTest(t)
	f.auth.Properties.EncryptionSettings = &a.EncryptionSettings{ContainerAppAuthEncryptionSecretName: to.Ptr("private-encryption-canary"), ContainerAppAuthSigningSecretName: to.Ptr("private-signing-canary")}
	f.auth.Properties.IdentityProviders = &a.IdentityProviders{Facebook: &a.Facebook{Registration: &a.AppRegistration{AppID: to.Ptr("private-provider-canary"), AppSecretSettingName: to.Ptr("private-setting-canary")}}}
	reader, err := NewSDKObserverHostReader(c, s, f.parent.credential, f)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(reader.Close)
	for _, id := range []string{s.Application + "/authConfigs/current", c.resourceID("Microsoft.App", "containerApps", "foreign")} {
		_, err := reader.Auth(t.Context(), id)
		requireReadDenyError(t, err, hostedexecution.ErrOwnership)
	}
	for _, change := range []func(*http.Request){
		func(r *http.Request) { r.Method = "PUT" },
		func(r *http.Request) { r.Method = "POST"; r.URL.Path += "/listSecrets" },
		func(r *http.Request) { r.URL.Path = s.Application + "/authConfigs" },
		func(r *http.Request) { r.URL.Path = s.Application + "/authConfigs/foreign" },
		func(r *http.Request) { r.URL.RawQuery += "&%24expand=private" },
	} {
		request, _ := http.NewRequestWithContext(t.Context(), "GET", "https://management.azure.com"+s.Application+"/authConfigs/current?api-version="+exchangeVersion, nil)
		change(request)
		_, err := reader.service.http.Do(request)
		requireReadDenyError(t, err, hostedexecution.ErrOwnership)
	}
	if f.reads != 0 {
		t.Fatal("unselected authentication operation reached transport")
	}
	actual, err := reader.Auth(t.Context(), s.Application)
	if err != nil || verifyObserverHostAuthentication(s, actual) != nil || f.reads != 1 {
		t.Fatal("selected authentication observation rejected", err, f.reads)
	}
	body, _ := json.Marshal(actual)
	if strings.Contains(string(body), "private-") || strings.Contains(string(body), "identityProviders") || strings.Contains(string(body), "encryptionSettings") {
		t.Fatal("authentication observer returned private provider configuration")
	}
	for _, mode := range []string{"unavailable", "missing", "redirect"} {
		f.authMode = mode
		_, err := reader.Auth(t.Context(), s.Application)
		requireReadDenyError(t, err, hostedexecution.ErrUnavailable)
	}
	if f.reads != 4 {
		t.Fatal("authentication reader retried or followed a redirect", f.reads)
	}
}

func TestAzureObserverHostAuthenticationRefreshesBeforeSuccessfulBoundInspection(t *testing.T) {
	c, s, f, g := observerHostTest(t)
	observer, err := g.BindHostObserver(t.Context(), c, s, f)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(observer.Close)
	bootstrap := f.parent.bootstrapReads
	if err := observer.Check(t.Context(), c); err != nil {
		t.Fatal(err)
	}
	f.auth.Properties.Platform.Enabled = to.Ptr(true)
	requireReadDenyError(t, observer.Check(t.Context(), c), hostedexecution.ErrOwnership)
	f.auth.Properties.Platform.Enabled = nil
	requireReadDenyError(t, observer.Check(t.Context(), c), hostedexecution.ErrUnavailable)
	f.auth.Properties.Platform.Enabled = to.Ptr(false)
	if err := observer.Check(t.Context(), c); err != nil {
		t.Fatal("restored disabled platform authentication rejected", err)
	}
	f.authMode = "missing"
	requireReadDenyError(t, observer.Check(t.Context(), c), hostedexecution.ErrUnavailable)
	if f.reads != 20 || f.parent.bootstrapReads != bootstrap || len(f.parent.factoryClients) != 1 {
		t.Fatal("authentication checks reused stale state, repeated bootstrap, or widened host cost", f.reads)
	}
	f.parent.unavailable = true
	requireReadDenyError(t, observer.Check(t.Context(), c), hostedexecution.ErrUnavailable)
	if f.reads != 20 {
		t.Fatal("unverified parent chain reached host authentication")
	}
}
