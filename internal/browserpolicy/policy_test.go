package browserpolicy

import "testing"

func TestBrowserOriginAndLocationValidation(t *testing.T) {
	for source, want := range map[string]string{"https://Example.test:443/": "https://example.test", "http://EXAMPLE.test.:80": "http://example.test", "http://[fd00::1]:8080": "http://[fd00::1]:8080"} {
		got, err := Origin(source)
		if err != nil || got != want {
			t.Fatalf("origin %s: %s %v", source, got, err)
		}
	}
	for _, source := range []string{"https://user:pass@example.test", "https://example.test/path", "https://example.test/?token=1", "https://example.test/#fragment", "file:///tmp/x", "http://example.test:0", "http://example.test:99999", "https://example.test/\\other"} {
		if _, err := Origin(source); err == nil {
			t.Fatalf("invalid origin accepted: %s", source)
		}
	}
	for _, source := range []string{"http://localhost:3000", "http://127.0.0.1:3000", "http://[::1]:3000", "http://0.0.0.0:3000", "http://app.localhost:3000"} {
		if ValidateTarget(Target{Origin: source, AllowPrivate: true}) == nil {
			t.Fatal("container localhost accepted", source)
		}
	}
	if err := ValidateTarget(Target{Origin: "http://host.docker.internal:3000", AllowPrivate: true}); err != nil {
		t.Fatal(err)
	}
}
