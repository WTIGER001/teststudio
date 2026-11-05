package kubernetes

import (
	"context"
	"errors"
	"net/http"
	"net/url"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/storage"
)

func (p *Provider) ensureNetwork(ctx context.Context, job hostedexecution.Job) error {
	if p.config.Gateway == nil || p.gatewayAuthority == nil || job.Operation != "" {
		return hostedexecution.ErrConfiguration
	}
	ticket, err := p.gatewayAuthority.PrepareGateway(ctx, job.Reference, p.gatewaySettings(job.Reference))
	if err != nil || ticket.Execution != job.Reference.ID || !ticket.Deadline.Equal(job.Ticket.Deadline) || len(ticket.Token) != 43 {
		return hostedexecution.ErrUnavailable
	}
	record, worker, err := p.jobRecord(job)
	if err != nil {
		return err
	}
	if previous, _, lookupErr := p.load(ctx, job.Reference); lookupErr == nil {
		record.ActiveSeconds = previous.ActiveSeconds
		worker = p.build(job, previous.ActiveSeconds)
		spec, err := normalizedSpec(worker.Spec, previous.ActiveSeconds, false)
		if err != nil {
			return err
		}
		record.SpecHash = fingerprint(spec)
		worker.Metadata.Annotations[specAnnotation] = record.SpecHash
	} else if !errors.Is(lookupErr, storage.ErrNotFound) {
		return lookupErr
	}
	objects, err := p.networkObjects(job, ticket, record.ActiveSeconds)
	if err != nil {
		return err
	}
	record.Network = &networkDispatch{Image: p.config.Gateway.Image}
	for _, object := range objects {
		record.Network.Resources = append(record.Network.Resources, networkResource{Kind: object.Kind, Name: object.Metadata.Name, SpecHash: object.Metadata.Annotations[specAnnotation]})
	}
	record.Fingerprint = fingerprint([]any{record.Fingerprint, record.Network})
	record, _, err = p.reserve(ctx, record)
	if err != nil {
		return err
	}
	if record.Removed || record.Stopping || record.Rejected {
		return hostedexecution.ErrRejected
	}
	if err := p.checkIsolation(ctx); err != nil {
		return hostedexecution.ErrUnavailable
	}
	for _, object := range objects {
		actual, err := p.ensureNetworkObject(ctx, record, object)
		if err != nil {
			return err
		}
		if object.Kind == "Pod" {
			if err := p.gatewayAuthority.BindGateway(ctx, job.Reference, actual.Metadata.UID); err != nil {
				return hostedexecution.ErrUnavailable
			}
		}
	}
	claimed := false
	record, err = p.update(ctx, job.Reference, func(current *dispatch) error {
		claimed = false
		if current.Removed || current.Stopping || current.Network == nil {
			return hostedexecution.ErrOwnership
		}
		if !current.Network.WorkerAttempted {
			current.Network.WorkerAttempted = true
			claimed = true
		}
		return nil
	})
	if err != nil {
		return err
	}
	var actual pod
	if claimed {
		err = p.request(ctx, http.MethodPost, p.podsPath(), url.Values{"fieldValidation": {"Strict"}}, worker, &actual)
	} else {
		err = p.request(ctx, http.MethodGet, p.podPath(job.Reference.Container), nil, nil, &actual)
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

func (p *Provider) ensureNetworkObject(ctx context.Context, record dispatch, object networkObject) (networkObject, error) {
	claimed := false
	record, err := p.update(ctx, record.Reference, func(current *dispatch) error {
		claimed = false
		index := resourceIndex(*current, object.Kind, object.Metadata.Name)
		if index < 0 || current.Removed || current.Stopping {
			return hostedexecution.ErrOwnership
		}
		if !current.Network.Resources[index].Attempted {
			current.Network.Resources[index].Attempted = true
			claimed = true
		}
		return nil
	})
	if err != nil {
		return networkObject{}, err
	}
	var actual networkObject
	if claimed {
		err = p.request(ctx, http.MethodPost, p.networkPath(object.Kind, ""), url.Values{"fieldValidation": {"Strict"}}, object, &actual)
		if errors.Is(err, errConflict) {
			err = p.request(ctx, http.MethodGet, p.networkPath(object.Kind, object.Metadata.Name), nil, nil, &actual)
		}
	} else {
		err = p.request(ctx, http.MethodGet, p.networkPath(object.Kind, object.Metadata.Name), nil, nil, &actual)
	}
	if err != nil {
		return actual, hostedexecution.ErrUnavailable
	}
	if err := p.verifyNetworkObject(actual, record); err != nil {
		return actual, err
	}
	_, err = p.rememberNetworkObject(ctx, record.Reference, actual)
	return actual, err
}
