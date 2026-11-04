package ecsworker

import (
	"context"
	"errors"
	"strings"

	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/workerexchange"
)

func (p *Provider) load(ctx context.Context, ref execution.Reference) (dispatch, storage.Record, *Plan, error) {
	row, err := p.store.Get(ctx, Bucket, ref.ID)
	if err != nil {
		return dispatch{}, row, nil, err
	}
	value, err := storage.Decode[dispatch](row)
	if err != nil || value.Proof.Reference != ref || row.Project != ref.Project {
		return value, row, nil, hostedexecution.ErrOwnership
	}
	plan, err := Restore(p.config, value.Proof)
	if err != nil || value.DefinitionARN != "" && !plan.validDefinitionARN(value.DefinitionARN) || value.TaskARN != "" && !plan.validTaskARN(value.TaskARN) {
		return value, row, nil, hostedexecution.ErrOwnership
	}
	if value.DefinitionARN != "" && !value.RegisterAttempted || value.TaskARN != "" && (!value.LaunchAttempted || value.DefinitionARN == "") {
		return value, row, nil, hostedexecution.ErrOwnership
	}
	if value.Terminal != nil {
		status := value.Terminal
		if value.TaskARN == "" || status.AdmissionPending || status.Reference != ref || status.Image != value.Proof.Image || !status.Deadline.Equal(value.Proof.Deadline) || !terminal(*status) || status.ExitCode != nil && (*status.ExitCode < -1 || *status.ExitCode > 255) || status.Phase == "succeeded" && (status.ExitCode == nil || *status.ExitCode != 0) || status.Phase == "failed" && status.ExitCode != nil && *status.ExitCode == 0 {
			return value, row, nil, hostedexecution.ErrOwnership
		}
	}
	return value, row, plan, nil
}

func (p *Provider) exchange(ctx context.Context, value dispatch) (workerexchange.Record, storage.Record, error) {
	row, err := p.store.Get(ctx, workerexchange.Bucket, value.Proof.Reference.ID)
	if err != nil {
		return workerexchange.Record{}, row, hostedexecution.ErrUnavailable
	}
	record, err := storage.Decode[workerexchange.Record](row)
	if err != nil || record.Reference != value.Proof.Reference || record.Operation != value.Proof.Operation || !record.DispatchGated || !record.Deadline.Equal(value.Proof.Deadline) {
		return record, row, hostedexecution.ErrOwnership
	}
	return record, row, nil
}

func (p *Provider) reserve(ctx context.Context, plan *Plan) (dispatch, bool, error) {
	value := dispatch{Proof: plan.Proof(), CreatedAt: p.now().UTC()}
	current, _, _, lookupErr := p.load(ctx, value.Proof.Reference)
	if lookupErr == nil {
		if current.Proof.Fingerprint != value.Proof.Fingerprint {
			return current, false, hostedexecution.ErrOwnership
		}
		return current, false, nil
	}
	if !errors.Is(lookupErr, storage.ErrNotFound) {
		return value, false, lookupErr
	}
	transport, guard, err := p.exchange(ctx, value)
	if err != nil || transport.Revoked {
		return value, false, hostedexecution.ErrUnavailable
	}
	change, err := storage.Put(Bucket, value.Proof.Reference.ID, value.Proof.Reference.Project, 0, value)
	if err != nil {
		return value, false, hostedexecution.ErrUnavailable
	}
	err = p.store.Commit(ctx, change, storage.Change{Record: guard, Assert: true})
	if err == nil {
		return value, true, nil
	}
	if !errors.Is(err, storage.ErrConflict) {
		return value, false, hostedexecution.ErrUnavailable
	}
	current, _, _, err = p.load(ctx, value.Proof.Reference)
	if err != nil || current.Proof.Fingerprint != value.Proof.Fingerprint {
		return current, false, hostedexecution.ErrOwnership
	}
	return current, false, nil
}

func (p *Provider) update(ctx context.Context, ref execution.Reference, dispatching bool, change func(*dispatch) error) (dispatch, error) {
	for range 8 {
		value, row, _, err := p.load(ctx, ref)
		if err != nil {
			return value, err
		}
		guards := []storage.Change{}
		if dispatching {
			record, guard, err := p.exchange(ctx, value)
			if err != nil || record.Revoked || !p.now().Before(record.Deadline) || value.Removed || value.Stopping {
				return value, hostedexecution.ErrUnavailable
			}
			guards = append(guards, storage.Change{Record: guard, Assert: true})
		}
		if err := change(&value); err != nil {
			return value, err
		}
		saved, err := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, value)
		if err != nil {
			return value, hostedexecution.ErrUnavailable
		}
		err = p.store.Commit(ctx, append(guards, saved)...)
		if errors.Is(err, storage.ErrConflict) {
			continue
		}
		if err != nil {
			return value, hostedexecution.ErrUnavailable
		}
		return value, nil
	}
	return dispatch{}, hostedexecution.ErrUnavailable
}

func (p *Provider) Inventory(ctx context.Context, installation, profile string) (result []execution.Reference, resultErr error) {
	defer func() { resultErr = controlError(resultErr) }()
	if !domain.ValidID(installation) || profile != "go-ecs-isolated-v1" && profile != "playwright-ecs-isolated-v1" && profile != "go-ecs-discovery-v1" && profile != "playwright-ecs-discovery-v1" {
		return nil, hostedexecution.ErrConfiguration
	}
	refs := []execution.Reference{}
	cursor := ""
	for {
		page, err := p.store.List(ctx, Bucket, "", cursor, 100)
		if err != nil {
			return nil, hostedexecution.ErrUnavailable
		}
		for _, row := range page.Items {
			value, err := storage.Decode[dispatch](row)
			if err != nil {
				return nil, hostedexecution.ErrOwnership
			}
			ref := value.Proof.Reference
			if ref.Installation != installation || ref.Profile != profile || value.Removed {
				continue
			}
			if _, err := Restore(p.config, value.Proof); err != nil {
				continue
			}
			if row.ID != ref.ID || row.Project != ref.Project {
				return nil, hostedexecution.ErrOwnership
			}
			refs = append(refs, ref)
		}
		cursor = page.Cursor
		if cursor == "" {
			return refs, nil
		}
	}
}

func terminal(status hostedexecution.Status) bool {
	return status.Phase == "succeeded" || status.Phase == "failed" || status.Phase == "stopped"
}

func runnerFor(ref execution.Reference) (string, string) {
	operation := ""
	suffix := "-ecs-isolated-v1"
	if strings.HasSuffix(ref.Profile, "-ecs-discovery-v1") {
		operation, suffix = "discover", "-ecs-discovery-v1"
	}
	return strings.TrimSuffix(ref.Profile, suffix), operation
}
