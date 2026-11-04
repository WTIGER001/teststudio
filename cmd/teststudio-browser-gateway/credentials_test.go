package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/internal/browsergateway"
	"github.com/wtiger001/teststudio/internal/browserpolicy"
)

func TestAPICredentialInitializationUsesPrivateEphemeralSocket(t *testing.T) {
	root, err := os.MkdirTemp("/tmp", "studio-api-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	path := filepath.Join(root, "credentials.sock")
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	type outcome struct {
		value *browsergateway.Configuration
		err   error
	}
	done := make(chan outcome, 1)
	go func() { value, err := receiveCredential(ctx, path); done <- outcome{value, err} }()
	payload := `{"target":{"origin":"https://approved.test"},"credential":{"origin":"https://approved.test","header":"Authorization","value":"Bearer bound-canary","network":"public"}}`
	if err := sendCredential(ctx, path, strings.NewReader(payload)); err != nil {
		t.Fatal(err)
	}
	result := <-done
	if result.err != nil || result.value.Credential.Value != "Bearer bound-canary" {
		t.Fatal("private credential channel failed", result.err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("credential socket is retained")
	}
	data, _ := json.Marshal(result.value)
	if !strings.Contains(string(data), "bound-canary") {
		t.Fatal("private in-memory packet is empty")
	}
}

func TestPrivateGatewayConfigurationSupportsUnauthenticatedBrowserTargets(t *testing.T) {
	root, err := os.MkdirTemp("/tmp", "studio-config-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(root)
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	path := filepath.Join(root, "configuration.sock")
	finished := make(chan error, 1)
	go func() {
		config, err := receiveCredential(ctx, path)
		if err == nil {
			var gateway *browsergateway.Gateway
			gateway, err = browsergateway.NewAuthenticated(config.Target, config.Credential)
			if gateway != nil {
				gateway.Close()
			}
		}
		finished <- err
	}()
	data, _ := json.Marshal(browsergateway.Configuration{Target: browserpolicy.Target{Origin: "https://approved.test"}})
	if err := sendCredential(ctx, path, strings.NewReader(string(data))); err != nil {
		t.Fatal(err)
	}
	if err := <-finished; err != nil {
		t.Fatal("origin-only browser configuration rejected", err)
	}
}
