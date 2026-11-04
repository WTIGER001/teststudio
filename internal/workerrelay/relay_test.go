package workerrelay

import (
	"context"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGoRequestRelayUsesOnlyOwnedUnixGatewayAndClosesConnections(t *testing.T) {
	root, err := os.MkdirTemp("/tmp", "studio-relay-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(root)
	path := filepath.Join(root, "gateway.sock")
	listener, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, r.Method+" "+r.URL.RequestURI())
	})}
	done := make(chan error, 1)
	go func() { done <- server.Serve(listener) }()
	t.Cleanup(func() { _ = server.Close(); <-done })
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	relay, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(relay.Close)
	client := &http.Client{Timeout: time.Second}
	defer client.CloseIdleConnections()
	response, err := client.Get(relay.URL() + "/health?part=1")
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if err != nil || string(body) != "GET /health?part=1" {
		t.Fatal("relay changes the guarded HTTP request", err)
	}
	relay.Close()
	if _, err := client.Get(relay.URL() + "/health"); err == nil {
		t.Fatal("closed relay remains reachable")
	}
	ctx, cancelMissing := context.WithCancel(t.Context())
	cancelMissing()
	if _, err := Open(ctx, filepath.Join(root, "absent.sock")); err == nil {
		t.Fatal("missing request gateway accepted")
	}
}
