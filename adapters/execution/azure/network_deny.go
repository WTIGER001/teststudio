package azureworker

import (
	"context"
	"errors"
	"sync"
	"time"

	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v3"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type NetworkDenyReader interface {
	Assignment(context.Context, string) (a.DenyAssignment, error)
}

type NetworkDenyGuard struct {
	worker    Config
	settings  NetworkDenySettings
	inventory *PrivateDNSInventoryGuard
	reader    NetworkDenyReader
}

func NewNetworkDenyGuard(c Config, s NetworkDenySettings, inventory *PrivateDNSInventoryGuard, reader NetworkDenyReader) (*NetworkDenyGuard, error) {
	s, err := s.canonical(c)
	if err != nil || inventory == nil || reader == nil || fingerprint(c) != fingerprint(inventory.worker) || fingerprint(s.PrivateDNS) != fingerprint(inventory.settings) {
		return nil, hostedexecution.ErrConfiguration
	}
	return &NetworkDenyGuard{worker: c, settings: s, inventory: inventory, reader: reader}, nil
}

func (g *NetworkDenyGuard) Check(ctx context.Context, c Config) (resultErr error) {
	defer func() { resultErr = controlError(resultErr) }()
	if fingerprint(c) != fingerprint(g.worker) {
		return hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()
	if err := g.inventory.Check(ctx, c); err != nil {
		return err
	}
	bindings := g.settings.bindings()
	assignments := make([]a.DenyAssignment, len(bindings))
	errs := make([]error, len(bindings))
	var group sync.WaitGroup
	for i, binding := range bindings {
		group.Go(func() { assignments[i], errs[i] = g.reader.Assignment(ctx, binding.assignment) })
	}
	group.Wait()
	for _, err := range errs {
		if errors.Is(err, hostedexecution.ErrOwnership) {
			return hostedexecution.ErrOwnership
		}
	}
	for i, err := range errs {
		if err != nil {
			return hostedexecution.ErrUnavailable
		}
		if err := verifyNetworkDeny(bindings[i], assignments[i]); err != nil {
			return err
		}
	}
	return nil
}
