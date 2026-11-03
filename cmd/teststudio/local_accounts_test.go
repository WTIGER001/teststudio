package main

import (
	"net/url"
	"testing"
)

func TestLocalAccountOriginRequiresProtectedCredentialTransport(t *testing.T) {
	for origin, allowed := range map[string]bool{"http://127.0.0.1:8040": true, "http://[::1]:8040": true, "http://localhost:8040": true, "https://studio.example.test": true, "http://studio.example.test": false, "http://192.168.1.20": false, "http://localhost.attacker.test": false} {
		parsed, err := url.Parse(origin)
		if err != nil {
			t.Fatal(err)
		}
		err = validateLocalAccountOrigin(parsed)
		if (err == nil) != allowed {
			t.Fatalf("%s: %v", origin, err)
		}
	}
}
