package kubernetes

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func (p *Provider) observeNetwork(ctx context.Context, ref execution.Reference) (hostedexecution.Status, error) {
	record, _, err := p.load(ctx, ref)
	if err != nil || record.Network == nil {
		return hostedexecution.Status{}, hostedexecution.ErrUnavailable
	}
	status, workerErr := p.observeWorker(ctx, ref)
	workerAbsent := errors.Is(workerErr, hostedexecution.ErrNotFound)
	if workerErr != nil && !workerAbsent {
		if errors.Is(workerErr, hostedexecution.ErrOwnership) {
			return status, workerErr
		}
		var value pod
		if err := p.request(ctx, http.MethodGet, p.podPath(ref.Container), nil, nil, &value); errors.Is(err, errAbsent) && (!record.Network.WorkerAttempted || time.Now().After(record.GuardUntil)) {
			workerAbsent = true
		}
		status = hostedexecution.Status{Reference: ref, Image: record.Image, Deadline: record.Deadline, Phase: "pending"}
	}
	if workerAbsent {
		status = hostedexecution.Status{Reference: ref, Image: record.Image, Deadline: record.Deadline, Phase: "stopped"}
	}
	gatewayReady, gatewayStopped := false, false
	failed := record.Network.Failed
	if workerAbsent && record.UID != "" && record.Terminal == nil && !record.Stopping {
		failed = true
	}
	for _, proof := range record.Network.Resources {
		var actual networkObject
		err := p.request(ctx, http.MethodGet, p.networkPath(proof.Kind, proof.Name), nil, nil, &actual)
		if errors.Is(err, errAbsent) {
			knownAbsent := !proof.Attempted || proof.UID != "" || time.Now().After(record.GuardUntil)
			if proof.Kind == "Pod" {
				gatewayStopped = knownAbsent
			}
			if !record.Stopping && (proof.UID != "" || time.Now().After(record.GuardUntil)) {
				failed = true
			}
			continue
		}
		if err != nil {
			return status, hostedexecution.ErrUnavailable
		}
		if err := p.verifyNetworkObject(actual, record); err != nil {
			if proof.Kind == "Pod" {
				return status, err
			}
			failed = true
			continue
		}
		if _, err := p.rememberNetworkObject(ctx, ref, actual); err != nil {
			return status, err
		}
		if proof.Kind != "Pod" {
			continue
		}
		var value pod
		value.APIVersion, value.Kind, value.Metadata, value.Spec = actual.APIVersion, actual.Kind, actual.Metadata, actual.Spec
		if len(actual.Status) > 0 && json.Unmarshal(actual.Status, &value.Status) != nil {
			return status, hostedexecution.ErrUnavailable
		}
		if len(value.Status.Containers) > 1 {
			return status, hostedexecution.ErrOwnership
		}
		if len(value.Status.Containers) == 1 {
			container := value.Status.Containers[0]
			if container.Name != "gateway" || container.Restarts != 0 {
				return status, hostedexecution.ErrOwnership
			}
			gatewayReady = value.Status.Phase == "Running" && container.Ready && container.State.Terminated == nil && value.Metadata.DeletionTimestamp == nil
		}
		gatewayStopped = value.Status.Phase == "Succeeded" || value.Status.Phase == "Failed"
		if gatewayStopped && !record.Stopping {
			failed = true
		}
	}
	if !record.Stopping {
		if p.checkIsolation(ctx) != nil {
			failed = true
		}
		if p.gatewayAuthority == nil {
			failed = true
		} else {
			state, err := p.gatewayAuthority.State(ctx, ref)
			if err != nil {
				return status, hostedexecution.ErrUnavailable
			}
			if !state.DispatchPending && !state.Revoked && !gatewayReady {
				failed = true
			}
		}
	}
	if failed && !record.Network.Failed {
		record, err = p.update(ctx, ref, func(current *dispatch) error { current.Network.Failed = true; return nil })
		if err != nil {
			return status, err
		}
	}
	workerStopped := workerAbsent || status.Phase == "succeeded" || status.Phase == "failed" || status.Phase == "stopped"
	status.InfrastructureFailure = failed
	if (record.Stopping || failed) && (!workerStopped || !gatewayStopped) {
		status.Phase, status.ExitCode, status.NativeCompletion = "running", nil, false
		status.AdmissionPending = !record.Stopping
		return status, nil
	}
	if failed {
		status.Phase, status.ExitCode, status.NativeCompletion = "failed", nil, false
		return status, nil
	}
	if workerAbsent && !gatewayStopped || workerStopped && !gatewayReady && !gatewayStopped {
		status.Phase, status.ExitCode, status.NativeCompletion = "running", nil, false
	}
	status.AdmissionPending = !record.Stopping && (!gatewayReady || status.Phase == "pending")
	if workerAbsent && gatewayStopped && record.Removed {
		return status, hostedexecution.ErrNotFound
	}
	return status, nil
}
