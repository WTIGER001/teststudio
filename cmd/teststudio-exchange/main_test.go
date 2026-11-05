package main

import (
	"context"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/adapters/workerexchange/relay"
)

func TestExchangeCommandRejectsUnconfiguredOrCredentialBearingOptions(t *testing.T) {
	for _, args := range [][]string{nil, {"--database=private-canary"}, {"--public-origin=https://workers.example.test", "--backend-origin=https://backend.example.test", "--backend-address=private-canary"}, {"--public-origin=https://workers.example.test", "--backend-origin=https://user:private-canary@backend.example.test"}, {"--public-origin=https://workers.example.test", "--backend-origin=http://backend.example.test"}, {"--public-origin=https://workers.example.test", "--backend-origin=https://backend.example.test", "extra"}} {
		if err := run(t.Context(), args); err == nil || strings.Contains(err.Error(), "canary") {
			t.Fatal("relay command accepted invalid options or exposed their values", err)
		}
	}
}

func TestExchangeCommandServesProcessHealthAndClosesItsListener(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	ctx, cancel := context.WithCancel(t.Context())
	handler, err := relay.New("https://workers.example.test", "https://backend.example.test", nil)
	if err != nil {
		listener.Close()
		cancel()
		t.Fatal(err)
	}
	finished := make(chan struct{})
	var serveErr error
	go func() { serveErr = serve(ctx, listener, handler); close(finished) }()
	t.Cleanup(func() {
		cancel()
		listener.Close()
		handler.Close()
		select {
		case <-finished:
		case <-time.After(2 * time.Second):
			t.Error("owned relay listener did not stop")
		}
	})
	transport := &http.Transport{}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: time.Second}
	response, err := client.Get("http://" + address + "/livez")
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil || response.StatusCode != 200 || !strings.Contains(string(body), `"service":"teststudio"`) {
		t.Fatal("relay process health failed", err, string(body))
	}
	cancel()
	select {
	case <-finished:
	case <-time.After(2 * time.Second):
		t.Fatal("relay did not stop after cancellation")
	}
	if serveErr != nil {
		t.Fatal(serveErr)
	}
	connection, err := net.DialTimeout("tcp", address, 100*time.Millisecond)
	if err == nil {
		connection.Close()
		t.Fatal("owned relay listener remains open")
	}
}
