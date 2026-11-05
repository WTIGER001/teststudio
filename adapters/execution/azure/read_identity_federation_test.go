package azureworker

import (
	"context"
	"net/http"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func TestAzureReadIdentityFederationRejectsPresentAndUnknownCredentialRoutes(t *testing.T) {
	c, s, f, g := readIdentityTest(t)
	reader := g.reader.(*SDKReadIdentityReader)
	for mode, want := range map[string]error{
		"present":       hostedexecution.ErrOwnership,
		"missing-value": hostedexecution.ErrUnavailable, "null-value": hostedexecution.ErrUnavailable,
		"null-item": hostedexecution.ErrUnavailable, "incomplete": hostedexecution.ErrUnavailable,
		"foreign-page": hostedexecution.ErrUnavailable, "forbidden": hostedexecution.ErrUnavailable,
	} {
		before := f.reads
		f.federation = mode
		requireReadDenyError(t, reader.FederationAbsent(t.Context(), s.Identity), want)
		if f.reads != before+1 || f.protection.reads != 0 {
			t.Fatal("empty-only federation check followed a continuation or repeated parents", mode, f.reads)
		}
	}
	var clients []string
	g.credential = func(id string) (azcore.TokenCredential, error) {
		clients = append(clients, id)
		return fixtureCredential{}, nil
	}
	f.federation = "present"
	credential, err := g.Credential(t.Context(), c)
	requireReadDenyError(t, err, hostedexecution.ErrOwnership)
	if credential != nil || len(clients) != 0 {
		t.Fatal("federated identity route reached managed identity credential factory")
	}
	before := f.reads
	f.federation = ""
	credential, err = g.Credential(t.Context(), c)
	if err != nil || credential == nil || len(clients) != 1 || clients[0] != s.ClientID || f.reads != before+5 {
		t.Fatal("explicit empty federation did not permit one fresh credential inspection", err, f.reads)
	}
	f.federation = "missing-value"
	credential, err = g.Credential(t.Context(), c)
	requireReadDenyError(t, err, hostedexecution.ErrUnavailable)
	if credential != nil || len(clients) != 1 {
		t.Fatal("unknown federation state reused a previous empty observation")
	}
}

func TestAzureReadIdentityFederationSDKRejectsOtherIdentitiesAndContinuationRequests(t *testing.T) {
	c, s, f, g := readIdentityTest(t)
	reader := g.reader.(*SDKReadIdentityReader)
	requireReadDenyError(t, reader.FederationAbsent(t.Context(), c.PullIdentity), hostedexecution.ErrOwnership)
	for _, mutate := range []func(*http.Request){
		func(r *http.Request) { r.Method = "PUT" },
		func(r *http.Request) { r.Method = "POST" },
		func(r *http.Request) { r.Method = "DELETE" },
		func(r *http.Request) { r.URL.Path += "/private-name" },
		func(r *http.Request) { r.URL.Path = c.PullIdentity + "/federatedIdentityCredentials" },
		func(r *http.Request) { r.URL.RawQuery += "&%24top=1" },
		func(r *http.Request) { r.URL.RawQuery += "&%24skiptoken=private-token" },
		func(r *http.Request) { r.URL.RawQuery = "api-version=2023-01-31" },
		func(r *http.Request) { r.URL.Host = "foreign.invalid" },
	} {
		request, _ := http.NewRequestWithContext(t.Context(), "GET", "https://management.azure.com"+s.Identity+"/federatedIdentityCredentials?api-version="+readIdentityVersion, nil)
		mutate(request)
		_, err := reader.http.Do(request)
		requireReadDenyError(t, err, hostedexecution.ErrOwnership)
	}
	if f.reads != 0 {
		t.Fatal("unselected federation operation reached transport")
	}
	if err := reader.FederationAbsent(t.Context(), s.Identity); err != nil || f.reads != 1 {
		t.Fatal("explicit empty collection rejected", err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	requireReadDenyError(t, reader.FederationAbsent(ctx, s.Identity), hostedexecution.ErrUnavailable)
}

func TestAzureReadIdentityFederationGatesBoundHostAndRefreshesAbsence(t *testing.T) {
	c, s, f, g := observerHostTest(t)
	observer, err := g.BindHostObserver(t.Context(), c, s, f)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(observer.Close)
	bootstrap := f.parent.bootstrapReads
	identity := f.parent.identity
	before := identity.reads
	identity.federation = "present"
	requireReadDenyError(t, observer.Check(t.Context(), c), hostedexecution.ErrOwnership)
	if f.reads != 0 || identity.reads != before+5 || f.parent.bootstrapReads != bootstrap {
		t.Fatal("reported federation reached host reads or repeated bootstrap")
	}
	identity.federation = ""
	if err := observer.Check(t.Context(), c); err != nil || f.reads != 2 || identity.reads != before+10 {
		t.Fatal("fresh empty federation did not restore bound host inspection", err, f.reads)
	}
	identity.federation = "incomplete"
	requireReadDenyError(t, observer.Check(t.Context(), c), hostedexecution.ErrUnavailable)
	if f.reads != 2 || identity.reads != before+15 || len(f.parent.factoryClients) != 1 || f.parent.bootstrapReads != bootstrap {
		t.Fatal("incomplete federation reused a previous successful observation")
	}
}
