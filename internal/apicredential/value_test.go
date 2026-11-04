package apicredential

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/wtiger001/teststudio/internal/apipolicy"
	"github.com/wtiger001/teststudio/internal/browserpolicy"
	"github.com/wtiger001/teststudio/internal/networkguard"
	"github.com/wtiger001/teststudio/internal/secrets"
)

type provider struct {
	value                       secrets.Value
	err                         error
	project, reference, purpose string
}

func (p *provider) References(context.Context, string, string) ([]secrets.Reference, error) {
	return nil, nil
}
func (p *provider) Resolve(_ context.Context, project, ref, purpose string) (secrets.Value, error) {
	p.project, p.reference, p.purpose = project, ref, purpose
	return p.value, p.err
}

func credentialTarget() browserpolicy.Target {
	return browserpolicy.Target{Origin: "https://approved.test", Authenticated: true, Requests: &apipolicy.Spec{Authentication: &apipolicy.Authentication{Binding: "login", Scheme: "bearer"}, Rules: []apipolicy.Rule{{Method: "GET", PathPrefix: "/", ReadOnly: true}}, MaxRequests: 3, MaxResponseBytes: 1024}}
}

func TestAPICredentialsResolveOnlyExactOriginsWithFixedPrivateErrors(t *testing.T) {
	target := credentialTarget()
	p := &provider{value: secrets.Value{Bytes: []byte("bound-canary"), Origins: []string{target.Origin}, Network: networkguard.Public}}
	value, err := Resolve(t.Context(), p, "alpha", "reader", target)
	if err != nil || value.Header != "Authorization" || value.Value != "Bearer bound-canary" || p.project != "alpha" || p.reference != "reader" || p.purpose != secrets.APICredential {
		t.Fatal("credential scope did not resolve", err)
	}
	for _, mutate := range []func(*provider, *browserpolicy.Target){
		func(p *provider, _ *browserpolicy.Target) {
			p.err = errors.New("private provider diagnostic bound-canary")
		},
		func(p *provider, _ *browserpolicy.Target) { p.value.Origins = []string{"https://elsewhere.test"} },
		func(p *provider, _ *browserpolicy.Target) { p.value.Network = networkguard.Loopback },
		func(p *provider, _ *browserpolicy.Target) { p.value.Bytes = []byte("injected\r\nHeader: private") },
		func(_ *provider, v *browserpolicy.Target) { v.Origin = "http://approved.test" },
	} {
		candidate := target
		copy := *p
		mutate(&copy, &candidate)
		if _, err := Resolve(t.Context(), &copy, "alpha", "reader", candidate); !errors.Is(err, ErrUnavailable) || strings.Contains(err.Error(), "canary") {
			t.Fatal("credential boundary exposes provider text or permits an unapproved scope", err)
		}
	}
	for _, scenario := range []struct{ scheme, raw, header string }{{"basic", "reader:private-password", "Authorization"}, {"header", "private-api-key", "X-Api-Key"}} {
		target.Requests.Authentication.Scheme, target.Requests.Authentication.Header = scenario.scheme, ""
		if scenario.scheme == "header" {
			target.Requests.Authentication.Header = "X-API-Key"
		}
		p.value.Bytes = []byte(scenario.raw)
		value, err := Resolve(t.Context(), p, "alpha", "reader", target)
		if err != nil || value.Header != scenario.header || len(value.PrivateValues()) == 0 {
			t.Fatal("supported authentication does not resolve", err)
		}
	}
}
