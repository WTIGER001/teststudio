package kubernetes

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/workerexchange"
)

func TestNetworkBundleBindsGatewayBeforeActivationAndCleansBothWorkloads(t *testing.T) {
	n := newNetworkFixture(t)
	p, ref := n.base.provider, n.job.Reference
	if err := p.Ensure(t.Context(), n.job); err != nil {
		t.Fatal(err)
	}
	status, err := p.Observe(t.Context(), ref)
	if err != nil || !status.AdmissionPending || status.InfrastructureFailure {
		t.Fatal("unready gateway released work or failed prematurely", status, err)
	}
	if err := n.base.exchange.Activate(t.Context(), ref); !errors.Is(err, workerexchange.ErrUnavailable) {
		t.Fatal("uninitialized gateway released work", err)
	}
	n.initialize(t)
	status, err = p.Observe(t.Context(), ref)
	if err != nil || status.AdmissionPending || status.InfrastructureFailure {
		t.Fatal("verified gateway bundle remains blocked", status, err)
	}
	if err := n.base.exchange.Activate(t.Context(), ref); err != nil {
		t.Fatal(err)
	}
	refs, err := p.Inventory(t.Context(), ref.Installation, ref.Profile)
	if err != nil || len(refs) != 1 || refs[0] != ref {
		t.Fatal("network bundle lost inventory ownership", refs, err)
	}
	if err := p.Remove(t.Context(), ref); err == nil {
		t.Fatal("active bundle removed before result checkpoint")
	}
	row, err := n.base.store.Get(t.Context(), Bucket, ref.ID)
	if err != nil {
		t.Fatal(err)
	}
	n.mu.Lock()
	gateway := n.objects[p.podPath(ref.Gateway)]
	data, _ := json.Marshal(gateway.Spec)
	n.mu.Unlock()
	var containers []struct {
		Env []struct{ Name, Value string }
	}
	encoded, _ := json.Marshal(gateway.Spec["containers"])
	json.Unmarshal(encoded, &containers)
	for _, env := range containers[0].Env {
		if env.Name == "TESTSTUDIO_GATEWAY_TOKEN" && strings.Contains(string(row.Data), env.Value) {
			t.Fatal("raw initialization capability entered provider journal")
		}
	}
	if strings.Contains(string(data), n.job.Ticket.Token) {
		t.Fatal("trusted gateway received runner ticket")
	}
	if err := p.Stop(t.Context(), ref); err != nil {
		t.Fatal(err)
	}
	status, err = p.Observe(t.Context(), ref)
	if err != nil || status.Phase != "stopped" || status.InfrastructureFailure {
		t.Fatal("both workloads did not stop", status, err)
	}
	n.checkpoint(t)
	if err := p.Remove(t.Context(), ref); err != nil {
		t.Fatal(err)
	}
	n.mu.Lock()
	remaining := len(n.objects)
	n.mu.Unlock()
	if remaining != 0 || n.base.value != nil {
		t.Fatal("bundle resources remain after cleanup")
	}
	n.assertNoRepeatedCreation(t)
}

func TestNetworkBundleRecoversLostGatewayAcknowledgementWithoutAnotherCreate(t *testing.T) {
	n := newNetworkFixture(t)
	n.lostGatewayAck = true
	if err := n.base.provider.Ensure(t.Context(), n.job); !errors.Is(err, hostedexecution.ErrUnavailable) {
		t.Fatal("lost acknowledgement lost uncertainty", err)
	}
	restarted, err := New(Options{Config: n.base.provider.config, Store: n.base.store, Inspector: imageInspector{"playwright"}, Transport: n.base.server.Client().Transport, Token: n.base.provider.token, GatewayAuthority: n.base.exchange})
	if err != nil {
		t.Fatal(err)
	}
	defer restarted.Close()
	if err := restarted.Ensure(t.Context(), n.job); err != nil {
		t.Fatal("restart could not recover same bundle", err)
	}
	n.assertNoRepeatedCreation(t)
	if err := restarted.Stop(t.Context(), n.job.Reference); err != nil {
		t.Fatal(err)
	}
	n.checkpoint(t)
	if err := restarted.Remove(t.Context(), n.job.Reference); err != nil {
		t.Fatal(err)
	}
}

func TestNetworkBundleRecoversPartialCreationWithoutStartingUploadedCode(t *testing.T) {
	n := newNetworkFixture(t)
	n.lostGatewayAck = true
	p, ref := n.base.provider, n.job.Reference
	if err := p.Ensure(t.Context(), n.job); !errors.Is(err, hostedexecution.ErrUnavailable) {
		t.Fatal(err)
	}
	if err := n.base.exchange.Revoke(t.Context(), ref); err != nil {
		t.Fatal(err)
	}
	refs, err := p.Inventory(t.Context(), ref.Installation, ref.Profile)
	if err != nil || len(refs) != 1 || refs[0] != ref {
		t.Fatal("partial bundle disappeared from inventory", refs, err)
	}
	status, err := p.Observe(t.Context(), ref)
	if err != nil || status.Phase != "running" || !status.AdmissionPending {
		t.Fatal("partial creation claimed quiescence or lost recoverability", status, err)
	}
	if err := p.Stop(t.Context(), ref); err != nil {
		t.Fatal("partial bundle could not stop", err)
	}
	n.checkpoint(t)
	if err := p.Remove(t.Context(), ref); err != nil {
		t.Fatal("partial bundle could not clean", err)
	}
	if n.base.posts != 0 {
		t.Fatal("partial creation recovery launched uploaded code")
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	if len(n.objects) != 0 {
		t.Fatal("partial bundle resources remain")
	}
}

func TestNetworkBundleStopsGatewayWhenRunnerObservationStalls(t *testing.T) {
	n := newNetworkFixture(t)
	p, ref := n.base.provider, n.job.Reference
	if err := p.Ensure(t.Context(), n.job); err != nil {
		t.Fatal(err)
	}
	n.blockWorkerObservation.Store(true)
	ctx, cancel := context.WithTimeout(t.Context(), 500*time.Millisecond)
	defer cancel()
	if err := p.Stop(ctx, ref); err == nil {
		t.Fatal("stalled runner observation falsely confirmed shutdown")
	}
	n.mu.Lock()
	gateway := n.objects[p.podPath(ref.Gateway)]
	n.mu.Unlock()
	if !bytes.Contains(gateway.Status, []byte(`"phase":"Failed"`)) {
		t.Fatal("stalled runner prevented independent gateway shutdown")
	}
	n.blockWorkerObservation.Store(false)
	status, err := p.Observe(t.Context(), ref)
	if err != nil || status.Phase != "running" {
		t.Fatal("unconfirmed runner shutdown became terminal", status, err)
	}
}

func TestNetworkBundleRejectsForeignGatewayAndStopsAfterPolicyLoss(t *testing.T) {
	t.Run("foreign gateway", func(t *testing.T) {
		n := newNetworkFixture(t)
		n.badGateway = true
		if err := n.base.provider.Ensure(t.Context(), n.job); !errors.Is(err, hostedexecution.ErrOwnership) {
			t.Fatal("foreign gateway admitted", err)
		}
		if n.base.posts != 0 {
			t.Fatal("uploaded runner launched before gateway ownership confirmation")
		}
		if err := n.base.exchange.Activate(t.Context(), n.job.Reference); !errors.Is(err, workerexchange.ErrUnavailable) {
			t.Fatal("foreign gateway activated code", err)
		}
	})
	t.Run("policy loss", func(t *testing.T) {
		n := newNetworkFixture(t)
		p, ref := n.base.provider, n.job.Reference
		if err := p.Ensure(t.Context(), n.job); err != nil {
			t.Fatal(err)
		}
		n.initialize(t)
		if err := n.base.exchange.Activate(t.Context(), ref); err != nil {
			t.Fatal(err)
		}
		n.mu.Lock()
		path := p.networkPath("NetworkPolicy", ref.Container)
		value := n.objects[path]
		value.Spec["egress"] = []any{map[string]any{}}
		n.objects[path] = value
		n.mu.Unlock()
		status, err := p.Observe(t.Context(), ref)
		if err != nil || !status.InfrastructureFailure {
			t.Fatal("policy loss did not invalidate admission", status, err)
		}
		if err := p.Stop(t.Context(), ref); err != nil {
			t.Fatal("broken policy prevented owned workload shutdown", err)
		}
		status, err = p.Observe(t.Context(), ref)
		if err != nil || status.Phase != "failed" || !status.InfrastructureFailure {
			t.Fatal("policy loss became a verified execution", status, err)
		}
		n.checkpoint(t)
		if err := p.Remove(t.Context(), ref); !errors.Is(err, hostedexecution.ErrOwnership) {
			t.Fatal("foreign policy deleted or cleanup falsely completed", err)
		}
		n.mu.Lock()
		_, exists := n.objects[path]
		n.mu.Unlock()
		if !exists {
			t.Fatal("changed policy was deleted without ownership proof")
		}
	})
}
