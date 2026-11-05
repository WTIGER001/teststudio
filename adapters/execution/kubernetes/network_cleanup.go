package kubernetes

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func (p *Provider) stopNetwork(ctx context.Context, ref execution.Reference) error {
	record, err := p.update(ctx, ref, func(current *dispatch) error { current.Stopping = true; return nil })
	if err != nil {
		return hostedexecution.ErrUnavailable
	}
	var workerErr, gatewayErr error
	var stopped sync.WaitGroup
	stopped.Add(2)
	go func() { defer stopped.Done(); workerErr = p.stopWorker(ctx, ref) }()
	go func() { defer stopped.Done(); gatewayErr = p.stopGateway(ctx, ref, record) }()
	stopped.Wait()
	if errors.Is(workerErr, hostedexecution.ErrNotFound) {
		workerErr = nil
	}
	return errors.Join(workerErr, gatewayErr)
}

func (p *Provider) stopGateway(ctx context.Context, ref execution.Reference, record dispatch) error {
	var gateway networkObject
	err := p.request(ctx, http.MethodGet, p.networkPath("Pod", ref.Gateway), nil, nil, &gateway)
	if errors.Is(err, errAbsent) {
		index := resourceIndex(record, "Pod", ref.Gateway)
		if index < 0 || record.Network.Resources[index].Attempted && record.Network.Resources[index].UID == "" && time.Now().Before(record.GuardUntil) {
			err = hostedexecution.ErrUnavailable
		} else {
			err = nil
		}
	} else if err == nil {
		err = p.verifyNetworkObject(gateway, record)
		if err == nil {
			var status struct {
				Phase string `json:"phase"`
			}
			if json.Unmarshal(gateway.Status, &status) != nil {
				err = hostedexecution.ErrUnavailable
			} else if status.Phase != "Succeeded" && status.Phase != "Failed" {
				patch := map[string]any{"metadata": map[string]string{"uid": gateway.Metadata.UID, "resourceVersion": gateway.Metadata.Version}, "spec": map[string]int{"activeDeadlineSeconds": 1}}
				err = p.request(ctx, http.MethodPatch, p.networkPath("Pod", ref.Gateway), nil, patch, &gateway)
				if err == nil {
					err = p.verifyNetworkObject(gateway, record)
				}
			}
		}
	}
	return err
}

func (p *Provider) removeNetwork(ctx context.Context, ref execution.Reference) error {
	if err := p.checkRemoval(ctx, ref); err != nil {
		return err
	}
	record, _, err := p.load(ctx, ref)
	if err != nil || record.Network == nil {
		return hostedexecution.ErrUnavailable
	}
	status, err := p.observeNetwork(ctx, ref)
	if err != nil && !errors.Is(err, hostedexecution.ErrNotFound) {
		return err
	}
	if err == nil && (status.Phase != "succeeded" && status.Phase != "failed" && status.Phase != "stopped" || status.AdmissionPending) {
		return hostedexecution.ErrUnavailable
	}
	var gateway networkObject
	err = p.request(ctx, http.MethodGet, p.networkPath("Pod", ref.Gateway), nil, nil, &gateway)
	if err == nil {
		if err := p.verifyNetworkObject(gateway, record); err != nil {
			return err
		}
		var state struct {
			Phase string `json:"phase"`
		}
		if json.Unmarshal(gateway.Status, &state) != nil || state.Phase != "Succeeded" && state.Phase != "Failed" {
			return hostedexecution.ErrUnavailable
		}
	} else if !errors.Is(err, errAbsent) {
		return hostedexecution.ErrUnavailable
	}
	if err := p.removeWorker(ctx, ref); err != nil {
		return err
	}
	for i := len(record.Network.Resources) - 1; i >= 0; i-- {
		if err := p.deleteNetworkObject(ctx, record, record.Network.Resources[i]); err != nil {
			return err
		}
	}
	_, err = p.update(ctx, ref, func(current *dispatch) error { current.Removed = true; return nil })
	return err
}

func (p *Provider) deleteNetworkObject(ctx context.Context, record dispatch, proof networkResource) error {
	var value networkObject
	path := p.networkPath(proof.Kind, proof.Name)
	err := p.request(ctx, http.MethodGet, path, nil, nil, &value)
	if errors.Is(err, errAbsent) {
		if proof.Attempted && proof.UID == "" && time.Now().Before(record.GuardUntil) {
			return hostedexecution.ErrUnavailable
		}
		return nil
	}
	if err != nil {
		return hostedexecution.ErrUnavailable
	}
	if err := p.verifyNetworkObject(value, record); err != nil {
		return err
	}
	options := map[string]any{"apiVersion": "v1", "kind": "DeleteOptions", "gracePeriodSeconds": 0, "propagationPolicy": "Foreground", "preconditions": map[string]string{"uid": value.Metadata.UID, "resourceVersion": value.Metadata.Version}}
	if err := p.request(ctx, http.MethodDelete, path, nil, options, nil); err != nil && !errors.Is(err, errAbsent) {
		return hostedexecution.ErrUnavailable
	}
	for {
		err := p.request(ctx, http.MethodGet, path, nil, nil, &value)
		if errors.Is(err, errAbsent) {
			return nil
		}
		if err != nil {
			return hostedexecution.ErrUnavailable
		}
		if err := p.verifyNetworkObject(value, record); err != nil {
			return err
		}
		timer := time.NewTimer(50 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return hostedexecution.ErrUnavailable
		case <-timer.C:
		}
	}
}
