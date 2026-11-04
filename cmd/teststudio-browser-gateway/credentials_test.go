package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/internal/apicredential"
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
		value *apicredential.Value
		err   error
	}
	done := make(chan outcome, 1)
	go func() { value, err := receiveCredential(ctx, path); done <- outcome{value, err} }()
	payload := `{"origin":"https://approved.test","header":"Authorization","value":"Bearer bound-canary","network":"public"}`
	if err := sendCredential(ctx, path, strings.NewReader(payload)); err != nil {
		t.Fatal(err)
	}
	result := <-done
	if result.err != nil || result.value.Value != "Bearer bound-canary" {
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
