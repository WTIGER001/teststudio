package apipolicy

import (
	"net/url"
	"testing"
)

func policy() *Spec {
	return &Spec{Rules: []Rule{{Method: "GET", PathPrefix: "/v1/items", ReadOnly: true}, {Method: "POST", PathPrefix: "/v1/search", ReadOnly: true, MaxBodyBytes: 1024}}, MaxRequests: 20, MaxResponseBytes: 65536}
}

func TestAPIRequestRulesRequireCanonicalPathsAndExplicitReadOnlyReview(t *testing.T) {
	spec := policy()
	if Validate(spec) != nil || !ReadOnly(spec) {
		t.Fatal("reviewed read-only search policy is rejected")
	}
	for _, scenario := range []struct {
		method, address string
		allowed         bool
	}{
		{"GET", "/v1/items/7?limit=5", true}, {"POST", "/v1/search", true},
		{"GET", "/v1/items/", true}, {"GET", "/v1/items//", false},
		{"DELETE", "/v1/items/7", false}, {"GET", "/v1/items-other", false},
		{"GET", "/v1/items/../admin", false}, {"GET", "/v1/items//admin", false},
		{"GET", "/v1/items/%2e%2e/admin", false}, {"GET", "/v1/items/%252e%252e/admin", false},
		{"GET", "/v1/items%2Fadmin", false}, {"GET", "/v1/items#private", false},
	} {
		u, err := url.Parse(scenario.address)
		if err != nil {
			t.Fatal(err)
		}
		_, err = Match(spec, scenario.method, u)
		if (err == nil) != scenario.allowed {
			t.Fatal("request-policy decision differs", scenario.method, scenario.address, err)
		}
	}
	for _, mutate := range []func(*Spec){
		func(v *Spec) { v.Rules = nil }, func(v *Spec) { v.MaxRequests = 0 },
		func(v *Spec) { v.MaxResponseBytes = 4<<20 + 1 },
		func(v *Spec) { v.Rules[0].Method = "CONNECT" },
		func(v *Spec) { v.Rules[0].Method = "DELETE" },
		func(v *Spec) { v.Rules[0].PathPrefix = "/v1/../admin" },
		func(v *Spec) { v.Rules[0].MaxBodyBytes = 1 },
		func(v *Spec) { v.Rules = append(v.Rules, v.Rules[0]) },
	} {
		value := policy()
		mutate(value)
		if Validate(value) == nil || ReadOnly(value) {
			t.Fatal("invalid request policy can become protected read-only execution")
		}
	}
	spec.Rules[1].ReadOnly = false
	if ReadOnly(spec) || Validate(spec) != nil {
		t.Fatal("explicit non-production POST policy is not distinguished")
	}
}
