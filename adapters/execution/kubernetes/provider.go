package kubernetes

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"time"

	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/storage"
)

func (p *Provider) Ensure(ctx context.Context, job hostedexecution.Job) error {
	if job.Reference.Gateway != "" {
		return p.ensureNetwork(ctx, job)
	}
	record, value, err := p.jobRecord(job)
	if err != nil {
		return err
	}
	record, create, err := p.reserve(ctx, record)
	if err != nil {
		return err
	}
	if !create {
		if record.Rejected || record.Removed {
			return hostedexecution.ErrRejected
		}
		_, err := p.Observe(ctx, job.Reference)
		return err
	}
	if err := p.checkIsolation(ctx); err != nil {
		if _, saveErr := p.update(ctx, job.Reference, func(record *dispatch) error { record.Rejected = true; return nil }); saveErr != nil {
			return hostedexecution.ErrUnavailable
		}
		return hostedexecution.ErrRejected
	}
	var actual pod
	err = p.request(ctx, http.MethodPost, p.podsPath(), url.Values{"fieldValidation": {"Strict"}}, value, &actual)
	if errors.Is(err, errRejected) {
		_, saveErr := p.update(ctx, job.Reference, func(record *dispatch) error { record.Rejected = true; return nil })
		if saveErr != nil {
			return hostedexecution.ErrUnavailable
		}
		return hostedexecution.ErrRejected
	}
	if err != nil {
		return hostedexecution.ErrUnavailable
	}
	if err := p.verifyPod(actual, record); err != nil {
		return err
	}
	status, err := statusOf(actual, record)
	if err != nil {
		return err
	}
	_, err = p.remember(ctx, record, actual, status)
	return err
}

func (p *Provider) Observe(ctx context.Context, ref execution.Reference) (hostedexecution.Status, error) {
	if ref.Gateway != "" {
		return p.observeNetwork(ctx, ref)
	}
	return p.observeWorker(ctx, ref)
}

func (p *Provider) observeWorker(ctx context.Context, ref execution.Reference) (hostedexecution.Status, error) {
	record, _, err := p.load(ctx, ref)
	if errors.Is(err, storage.ErrNotFound) {
		record, err = p.adopt(ctx, ref)
	}
	if err != nil {
		return hostedexecution.Status{}, err
	}
	var value pod
	err = p.request(ctx, http.MethodGet, p.podPath(ref.Container), nil, nil, &value)
	if errors.Is(err, errAbsent) {
		if record.Terminal != nil {
			return *record.Terminal, nil
		}
		if record.Network != nil && (record.UID != "" || !record.Network.WorkerAttempted || time.Now().After(record.GuardUntil)) {
			return hostedexecution.Status{}, hostedexecution.ErrNotFound
		}
		if record.Rejected || record.Removed || record.UID == "" && time.Now().After(record.GuardUntil) {
			return hostedexecution.Status{}, hostedexecution.ErrNotFound
		}
		return hostedexecution.Status{}, hostedexecution.ErrUnavailable
	}
	if err != nil {
		return hostedexecution.Status{}, hostedexecution.ErrUnavailable
	}
	if err := p.verifyPod(value, record); err != nil {
		return hostedexecution.Status{}, err
	}
	status, err := statusOf(value, record)
	if err != nil {
		return status, err
	}
	if _, err := p.remember(ctx, record, value, status); err != nil {
		return status, hostedexecution.ErrUnavailable
	}
	return status, nil
}
