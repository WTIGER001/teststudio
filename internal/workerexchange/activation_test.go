package workerexchange_test

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/internal/workerexchange"
)

func TestPendingDispatchCannotFetchOrStartBeforeControllerActivation(t *testing.T) {
	f := fresh(t)
	f.input.AwaitDispatch = true
	ticket := f.prepare(t)
	if _, err := f.service.Input(t.Context(), f.ref.ID, ticket.Token, "worker"); !errors.Is(err, workerexchange.ErrUnavailable) {
		t.Fatal("unverified provider received executable work", err)
	}
	if err := f.service.Started(t.Context(), f.ref.ID, ticket.Token, "worker", time.Now().UTC()); !errors.Is(err, workerexchange.ErrUnavailable) {
		t.Fatal("unverified provider reported a test start", err)
	}
	state, err := f.service.State(t.Context(), f.ref)
	if err != nil || !state.DispatchPending {
		t.Fatal(state, err)
	}
	if err := f.service.Activate(t.Context(), f.ref); err != nil {
		t.Fatal(err)
	}
	packet, err := f.service.Input(t.Context(), f.ref.ID, ticket.Token, "worker")
	if err != nil || bytes.Contains(packet.Input, []byte("awaitDispatch")) {
		t.Fatal("activation did not preserve the worker contract", err)
	}
	if err := f.service.Activate(t.Context(), f.ref); err != nil {
		t.Fatal("activation retry was not idempotent", err)
	}
	if err := f.service.Revoke(t.Context(), f.ref); err != nil {
		t.Fatal(err)
	}
	if err := f.service.Activate(t.Context(), f.ref); !errors.Is(err, workerexchange.ErrDenied) {
		t.Fatal("revoked work was reactivated", err)
	}
}
