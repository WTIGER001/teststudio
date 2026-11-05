package ecsworker

import (
	"context"
	"errors"
	"sort"
	"sync"

	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func bothRoles(worker, gateway func() error) error {
	var workerErr, gatewayErr error
	var group sync.WaitGroup
	group.Add(2)
	go func() { defer group.Done(); workerErr = worker() }()
	go func() { defer group.Done(); gatewayErr = gateway() }()
	group.Wait()
	if errors.Is(workerErr, hostedexecution.ErrOwnership) || errors.Is(gatewayErr, hostedexecution.ErrOwnership) {
		return hostedexecution.ErrOwnership
	}
	if workerErr != nil || gatewayErr != nil {
		return hostedexecution.ErrUnavailable
	}
	return nil
}

func (p *EnvironmentProvider) Stop(ctx context.Context, ref execution.Reference) (resultErr error) {
	defer func() { resultErr = controlError(resultErr) }()
	value, err := p.worker.update(ctx, ref, false, func(value *dispatch) error { value.Stopping = true; return nil })
	if err != nil {
		return err
	}
	return bothRoles(func() error { return p.worker.Stop(ctx, ref) }, func() error {
		if !value.Environment.GatewayAttempted {
			return nil
		}
		return p.gateway.Stop(ctx, ref)
	})
}

func (p *EnvironmentProvider) Remove(ctx context.Context, ref execution.Reference) (resultErr error) {
	defer func() { resultErr = controlError(resultErr) }()
	value, _, _, err := p.worker.load(ctx, ref)
	if err != nil {
		return err
	}
	if err := p.worker.authorizeCleanup(ctx, value); err != nil {
		return err
	}
	return bothRoles(func() error { return p.worker.Remove(ctx, ref) }, func() error {
		if !value.Environment.GatewayAttempted {
			return nil
		}
		return p.gateway.Remove(ctx, ref)
	})
}

func (p *EnvironmentProvider) Inventory(ctx context.Context, installation, profile string) ([]execution.Reference, error) {
	worker, err := p.worker.Inventory(ctx, installation, profile)
	if err != nil {
		return nil, err
	}
	gateway, err := p.gateway.Inventory(ctx, installation, profile)
	if err != nil {
		return nil, err
	}
	byID := map[string]execution.Reference{}
	for _, ref := range append(worker, gateway...) {
		if previous, exists := byID[ref.ID]; exists && previous != ref {
			return nil, hostedexecution.ErrOwnership
		}
		byID[ref.ID] = ref
	}
	result := make([]execution.Reference, 0, len(byID))
	for _, ref := range byID {
		result = append(result, ref)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}
