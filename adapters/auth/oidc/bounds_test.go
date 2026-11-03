package oidc

import (
	"net/http"
	"testing"
	"time"
)

func TestProviderNetworkBudgetCapsCallerClientsWithoutMutatingThem(t *testing.T) {
	for _, duration := range []time.Duration{0, time.Hour, time.Millisecond} {
		supplied := &http.Client{Timeout: duration}
		client := (&Service{Client: supplied}).httpClient()
		want := duration
		if want == 0 || want > 10*time.Second {
			want = 10 * time.Second
		}
		if client.Timeout != want || supplied.Timeout != duration || client == supplied {
			t.Fatal("network budget is unbounded or mutated the caller client")
		}
	}
}

func TestSignInReturnRoutesRejectEncodedAuthoritiesAndServerActions(t *testing.T) {
	for _, path := range []string{"https://other.example", "//other.example", "/%2fother.example", "/%5cother.example", "/auth/logout/opaque", "/api/v1/projects"} {
		if returnPath(path) != "/" {
			t.Fatal("unsafe sign-in return path accepted", path)
		}
	}
	if got := returnPath("/projects/alpha/runs/one?test=check#evidence"); got != "/projects/alpha/runs/one?test=check#evidence" {
		t.Fatal("safe deep link changed", got)
	}
}

func TestProviderRedirectsRejectDowngradesAndRespectCallerPolicy(t *testing.T) {
	called := false
	supplied := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { called = true; return http.ErrUseLastResponse }}
	client := (&Service{Client: supplied}).httpClient()
	secure, _ := http.NewRequest("GET", "https://identity.example/discovery", nil)
	insecure, _ := http.NewRequest("GET", "http://identity.example/discovery", nil)
	if err := client.CheckRedirect(insecure, []*http.Request{secure}); err == nil || called {
		t.Fatal("downgrade reached the caller redirect policy")
	}
	if err := client.CheckRedirect(secure, []*http.Request{secure}); err != http.ErrUseLastResponse || !called {
		t.Fatal("caller redirect policy was lost")
	}
}
