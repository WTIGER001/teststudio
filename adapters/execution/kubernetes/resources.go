package kubernetes

import (
	"context"
	"errors"
	"net/http"

	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/workerstate"
)

func (p *Provider) Stop(ctx context.Context, ref execution.Reference) error {
	status, err := p.Observe(ctx, ref)
	if err != nil {
		return err
	}
	if status.Phase == "succeeded" || status.Phase == "failed" || status.Phase == "stopped" {
		return nil
	}
	record, err := p.update(ctx, ref, func(value *dispatch) error { value.Stopping = true; return nil })
	if err != nil {
		return hostedexecution.ErrUnavailable
	}
	var value pod
	if err := p.request(ctx, http.MethodGet, p.podPath(ref.Container), nil, nil, &value); err != nil {
		return hostedexecution.ErrUnavailable
	}
	if err := p.verifyPod(value, record); err != nil {
		return err
	}
	patch := map[string]any{"metadata": map[string]string{"uid": value.Metadata.UID, "resourceVersion": value.Metadata.Version}, "spec": map[string]int{"activeDeadlineSeconds": 1}}
	if err := p.request(ctx, http.MethodPatch, p.podPath(ref.Container), nil, patch, &value); err != nil {
		return hostedexecution.ErrUnavailable
	}
	return p.verifyPod(value, record)
}

func (p *Provider) Remove(ctx context.Context, ref execution.Reference) error {
	entry, _, err := (workerstate.Ledger{Store: p.store}).Get(ctx, ref.Run, ref.ID)
	if err != nil || entry.Reference != ref || !entry.ResultSaved || !entry.EvidenceComplete && !entry.EvidenceAbandoned {
		return hostedexecution.ErrUnavailable
	}
	status, err := p.Observe(ctx, ref)
	if err != nil && !errors.Is(err, hostedexecution.ErrNotFound) {
		return err
	}
	if err == nil && status.Phase != "succeeded" && status.Phase != "failed" && status.Phase != "stopped" {
		return hostedexecution.ErrUnavailable
	}
	var value pod
	err = p.request(ctx, http.MethodGet, p.podPath(ref.Container), nil, nil, &value)
	if err == nil {
		record, _, loadErr := p.load(ctx, ref)
		if loadErr != nil {
			return hostedexecution.ErrUnavailable
		}
		if err := p.verifyPod(value, record); err != nil {
			return err
		}
		terminal, err := statusOf(value, record)
		if err != nil || terminal.Phase != "succeeded" && terminal.Phase != "failed" && terminal.Phase != "stopped" {
			return hostedexecution.ErrUnavailable
		}
		options := map[string]any{"apiVersion": "v1", "kind": "DeleteOptions", "gracePeriodSeconds": 0, "propagationPolicy": "Foreground", "preconditions": map[string]string{"uid": value.Metadata.UID, "resourceVersion": value.Metadata.Version}}
		if err := p.request(ctx, http.MethodDelete, p.podPath(ref.Container), nil, options, nil); err != nil && !errors.Is(err, errAbsent) {
			return hostedexecution.ErrUnavailable
		}
		if err := p.request(ctx, http.MethodGet, p.podPath(ref.Container), nil, nil, &value); !errors.Is(err, errAbsent) {
			return hostedexecution.ErrUnavailable
		}
	} else if !errors.Is(err, errAbsent) {
		return hostedexecution.ErrUnavailable
	}
	_, err = p.update(ctx, ref, func(record *dispatch) error { record.Removed = true; return nil })
	if errors.Is(err, storage.ErrNotFound) {
		return nil
	}
	if err != nil {
		return hostedexecution.ErrUnavailable
	}
	return nil
}
