package main

import (
	"context"
	"testing"
	"time"
)

func TestSandboxJobModesKeepIdleSeparateFromExecutableWork(t *testing.T) {
	now := time.Now().UTC()
	at := now.Add(time.Minute).Format(time.RFC3339Nano)
	options, err := parseOptions([]string{"--sandbox-idle-until=" + at}, now)
	if err != nil || options.idleUntil.IsZero() || options.runner != "" || options.origin != "" {
		t.Fatal("idle mode accepted executable work", err)
	}
	for _, args := range [][]string{
		{"--sandbox-idle-until=" + at, "--runner=go"},
		{"--sandbox-idle-until=" + at, "--sandbox-command"},
		{"--sandbox-idle-until=" + now.Add(-time.Second).Format(time.RFC3339Nano)},
		{"--sandbox-idle-until=" + now.Add(3*time.Hour).Format(time.RFC3339Nano)},
		{"--sandbox-command", "--runner=unknown"},
	} {
		if _, err := parseOptions(args, now); err == nil {
			t.Fatal("invalid or unbounded sandbox mode accepted")
		}
	}
	options, err = parseOptions([]string{"--sandbox-command", "--runner=playwright", "--exchange-origin=https://exchange.example", "--operation=discover"}, now)
	if err != nil || !options.sandbox || options.operation != "discover" {
		t.Fatal("fixed command mode unavailable", err)
	}
}

func TestSandboxIdleObservesCancellationWithoutWorkOrCredentials(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	idleSandbox(ctx, time.Now().Add(time.Hour))
	idleSandbox(t.Context(), time.Now().Add(-time.Second))
}
