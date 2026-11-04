package main

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestListenerFailureDoesNotStartWorkersAndReleasesEarlierPorts(t *testing.T) {
	occupied, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer occupied.Close()
	available, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := available.Addr().String()
	available.Close()
	started := false
	err = serveServers(t.Context(), []*http.Server{{Addr: address}, {Addr: occupied.Addr().String()}}, func() func() { started = true; return nil })
	if err == nil || started {
		t.Fatal("listener failure started workers", err)
	}
	listener, err := net.Listen("tcp", address)
	if err != nil {
		t.Fatal("partially opened listener was not released", err)
	}
	listener.Close()
}

func TestServiceCancellationStopsItsOwnedListenersAndWorkers(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	started := make(chan struct{})
	cleaned := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- serveServers(ctx, []*http.Server{{Addr: "127.0.0.1:0"}, {Addr: "127.0.0.1:0"}}, func() func() {
			close(started)
			return func() { close(cleaned) }
		})
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("listeners did not start")
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("service did not stop")
	}
	select {
	case <-cleaned:
	default:
		t.Fatal("workers were not stopped")
	}
}
