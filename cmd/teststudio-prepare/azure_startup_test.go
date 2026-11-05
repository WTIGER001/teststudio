package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/adapters/registry/acr"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func TestAzureAssemblyStartupWaitsForReadinessAndStopsOnMismatchOrCancellation(t *testing.T) {
	attempts := 0
	ready := &acr.Client{}
	client, err := awaitAzureRegistry(t.Context(), func(context.Context) (*acr.Client, error) {
		attempts++
		if attempts < 3 {
			return nil, hostedexecution.ErrUnavailable
		}
		return ready, nil
	}, time.Millisecond)
	if err != nil || client != ready || attempts != 3 {
		t.Fatal("startup claimed readiness before observation succeeded", err, attempts)
	}
	attempts = 0
	_, err = awaitAzureRegistry(t.Context(), func(context.Context) (*acr.Client, error) { attempts++; return nil, hostedexecution.ErrOwnership }, time.Millisecond)
	if !errors.Is(err, hostedexecution.ErrOwnership) || attempts != 1 {
		t.Fatal("startup retried a definite configuration mismatch", err, attempts)
	}
	ctx, cancel := context.WithCancel(t.Context())
	attempts = 0
	_, err = awaitAzureRegistry(ctx, func(context.Context) (*acr.Client, error) {
		attempts++
		cancel()
		return nil, hostedexecution.ErrUnavailable
	}, time.Minute)
	if !errors.Is(err, hostedexecution.ErrUnavailable) || attempts != 1 {
		t.Fatal("startup did not stop during its readiness wait", err, attempts)
	}
}
