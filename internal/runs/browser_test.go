package runs

import (
	"maps"
	"testing"

	"github.com/wtiger001/teststudio/internal/apipolicy"
	"github.com/wtiger001/teststudio/internal/browserpolicy"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/projects"
)

func TestProtectedAPITargetRequiresAnExplicitReadOnlyRequestPolicy(t *testing.T) {
	environment := projects.Environment{Application: "app", Stage: "local", Protected: true, Endpoints: map[string]string{"api": "http://host.docker.internal:3000"}}
	artifact := catalog.Artifact{Browser: &browserpolicy.Spec{Endpoint: "api"}}
	test := catalog.Test{Runner: "playwright", Type: "api"}
	if _, err := browserTarget(&environment, artifact, test); err == nil {
		t.Fatal("origin-only proxy claims protected API request enforcement")
	}
	artifact.Browser.Requests = &apipolicy.Spec{Rules: []apipolicy.Rule{{Method: "GET", PathPrefix: "/", ReadOnly: true}}, MaxRequests: 10, MaxResponseBytes: 1024}
	if target, err := browserTarget(&environment, artifact, test); err != nil || target.Requests == nil {
		t.Fatal("protected read-only API policy is rejected", err)
	}
	artifact.Browser.Requests.Rules[0] = apipolicy.Rule{Method: "DELETE", PathPrefix: "/items"}
	if _, err := browserTarget(&environment, artifact, test); err == nil {
		t.Fatal("mutating API policy enters a protected target")
	}
	environment.Protected = false
	if _, err := browserTarget(&environment, artifact, test); err != nil {
		t.Fatal("explicit unprotected API mutation policy is rejected", err)
	}
	test.Type = "browser"
	if _, err := browserTarget(&environment, artifact, test); err == nil {
		t.Fatal("browser type can silently select request-restricted execution")
	}
}

func TestBrowserAdmissionRequiresReviewedReachableTargetAndNoBindings(t *testing.T) {
	artifact := catalog.Artifact{Browser: &browserpolicy.Spec{Endpoint: "api"}}
	test := catalog.Test{Runner: "playwright"}
	environment := projects.Environment{Application: "app", Stage: "local", Endpoints: map[string]string{"api": "http://host.docker.internal:3000"}, Locations: []string{"local"}}
	target, err := browserTarget(&environment, artifact, test)
	if err != nil || !target.AllowPrivate || target.Origin != environment.Endpoints["api"] {
		t.Fatal(target, err)
	}
	if _, err := browserTarget(nil, artifact, test); err == nil {
		t.Fatal("targetless recorded browser was admitted")
	}
	test.RequiredBindings = []string{"private-login"}
	if _, err := browserTarget(&environment, artifact, test); err == nil {
		t.Fatal("unsupported credential injection was admitted")
	}
	test.RequiredBindings = nil
	for _, address := range []string{"http://localhost:3000", "http://127.0.0.1:3000", "http://host.docker.internal:3000/path"} {
		candidate := environment
		candidate.Endpoints = map[string]string{"api": address}
		if _, err := browserTarget(&candidate, artifact, test); err == nil {
			t.Fatal("incompatible target admitted", address)
		}
	}
	environment.Stage = "production"
	if _, err := browserTarget(&environment, artifact, test); err == nil {
		t.Fatal("HTTP bypassed the environment policy")
	}
}

func TestBrowserAttemptsRejectChangedTargetPolicy(t *testing.T) {
	original := projects.Environment{Application: "app", Stage: "local", Endpoints: map[string]string{"api": "http://host.docker.internal:3000"}, Locations: []string{"local"}}
	run := Run{Environment: &original, Release: catalog.Release{Draft: catalog.Draft{Artifact: catalog.Artifact{Browser: &browserpolicy.Spec{Endpoint: "api"}}}}, Request: Request{Location: "local"}}
	suite := projects.Suite{Applications: []string{"app"}}
	if err := currentEnvironmentPolicy(run, original, suite); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*projects.Environment){
		func(v *projects.Environment) {
			v.Endpoints = maps.Clone(v.Endpoints)
			v.Endpoints["api"] = "https://other.test"
		},
		func(v *projects.Environment) { v.AllowPrivate = true },
		func(v *projects.Environment) { v.AllowHTTP = true },
		func(v *projects.Environment) { v.Stage = "production" },
		func(v *projects.Environment) { v.Application = "foreign" },
		func(v *projects.Environment) { v.Locations = []string{"remote"} },
		func(v *projects.Environment) { v.Credentials = map[string]string{"login": "new-secret"} },
	} {
		current := original
		mutate(&current)
		if currentEnvironmentPolicy(run, current, suite) == nil {
			t.Fatal("changed policy remained executable")
		}
	}
	if currentEnvironmentPolicy(run, original, projects.Suite{Applications: []string{"foreign"}}) == nil {
		t.Fatal("current suite application boundary was ignored")
	}
}

func TestAPICredentialTargetsRequireDeclaredBindingsAndSecureTransport(t *testing.T) {
	spec := &apipolicy.Spec{Authentication: &apipolicy.Authentication{Binding: "api-login", Scheme: "bearer"}, Rules: []apipolicy.Rule{{Method: "GET", PathPrefix: "/", ReadOnly: true}}, MaxRequests: 3, MaxResponseBytes: 1024}
	artifact := catalog.Artifact{Browser: &browserpolicy.Spec{Endpoint: "api", Requests: spec}}
	test := catalog.Test{Type: "api", Runner: "playwright", RequiredBindings: []string{"api-login"}}
	environment := projects.Environment{Stage: "local", Endpoints: map[string]string{"api": "http://host.docker.internal:3000"}}
	if _, err := browserTarget(&environment, artifact, test); err == nil {
		t.Fatal("missing environment credential admitted")
	}
	environment.Credentials = map[string]string{"api-login": "approved-reader"}
	target, err := browserTarget(&environment, artifact, test)
	if err != nil || !target.Authenticated || !target.AllowCredentialHTTP {
		t.Fatal("explicit local credential target rejected", err)
	}
	environment.Stage, environment.AllowHTTP = "development", true
	if _, err := browserTarget(&environment, artifact, test); err == nil {
		t.Fatal("remote HTTP credential target admitted")
	}
	environment.Endpoints["api"] = "https://approved.test"
	if target, err := browserTarget(&environment, artifact, test); err != nil || !target.Authenticated || target.AllowCredentialHTTP {
		t.Fatal("HTTPS credential target rejected", err)
	}
	test.RequiredBindings = nil
	if target, err := browserTarget(&environment, artifact, test); err != nil || target.Authenticated {
		t.Fatal("no-credential test acquires authentication", err)
	}
}
