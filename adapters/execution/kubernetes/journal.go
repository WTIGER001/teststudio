package kubernetes

import (
	"context"
	"errors"

	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/storage"
)

func (p *Provider) load(ctx context.Context, ref execution.Reference) (dispatch, storage.Record, error) {
	if !validReference(ref) {
		return dispatch{}, storage.Record{}, hostedexecution.ErrOwnership
	}
	row, err := p.store.Get(ctx, Bucket, ref.ID)
	if err != nil {
		return dispatch{}, row, err
	}
	record, err := storage.Decode[dispatch](row)
	if err != nil || record.Reference != ref || row.Project != ref.Project || record.Scope != p.scope() {
		return dispatch{}, row, hostedexecution.ErrOwnership
	}
	return record, row, nil
}

func (p *Provider) reserve(ctx context.Context, record dispatch) (dispatch, bool, error) {
	change, err := storage.Put(Bucket, record.Reference.ID, record.Reference.Project, 0, record)
	if err != nil {
		return record, false, hostedexecution.ErrUnavailable
	}
	if err := p.store.Commit(ctx, change); err == nil {
		return record, true, nil
	} else if !errors.Is(err, storage.ErrConflict) {
		return record, false, hostedexecution.ErrUnavailable
	}
	current, _, err := p.load(ctx, record.Reference)
	if err != nil {
		return record, false, hostedexecution.ErrUnavailable
	}
	if current.Fingerprint != record.Fingerprint {
		return current, false, hostedexecution.ErrOwnership
	}
	return current, false, nil
}

func (p *Provider) update(ctx context.Context, ref execution.Reference, apply func(*dispatch) error) (dispatch, error) {
	for range 8 {
		record, row, err := p.load(ctx, ref)
		if err != nil {
			return record, err
		}
		if err := apply(&record); err != nil {
			return record, err
		}
		change, err := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, record)
		if err != nil {
			return record, hostedexecution.ErrUnavailable
		}
		if err := p.store.Commit(ctx, change); errors.Is(err, storage.ErrConflict) {
			continue
		} else if err != nil {
			return record, hostedexecution.ErrUnavailable
		}
		return record, nil
	}
	return dispatch{}, hostedexecution.ErrUnavailable
}

func (p *Provider) remember(ctx context.Context, record dispatch, value pod, status hostedexecution.Status) (dispatch, error) {
	return p.update(ctx, record.Reference, func(current *dispatch) error {
		if current.SpecHash != record.SpecHash || current.UID != "" && current.UID != value.Metadata.UID {
			return hostedexecution.ErrOwnership
		}
		if current.Terminal != nil && fingerprint(*current.Terminal) != fingerprint(status) {
			return hostedexecution.ErrOwnership
		}
		current.UID = value.Metadata.UID
		if status.Phase == "succeeded" || status.Phase == "failed" || status.Phase == "stopped" {
			current.Terminal = &status
		}
		return nil
	})
}
