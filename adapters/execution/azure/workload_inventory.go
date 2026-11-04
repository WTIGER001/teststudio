package azureworker

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type WorkloadPlacement struct {
	ResourceID    string `json:"resourceId"`
	EnvironmentID string `json:"environmentId"`
}

type WorkloadInventoryReader interface {
	Applications(context.Context) ([]WorkloadPlacement, error)
	Jobs(context.Context) ([]WorkloadPlacement, error)
}

type WorkloadInventoryGuard struct {
	worker   Config
	settings ExchangeSettings
	exchange *ExchangeGuard
	reader   WorkloadInventoryReader
}

func NewWorkloadInventoryGuard(c Config, s ExchangeSettings, exchange *ExchangeGuard, reader WorkloadInventoryReader) (*WorkloadInventoryGuard, error) {
	s, err := s.canonical(c)
	if err != nil || exchange == nil || reader == nil || fingerprint(c) != fingerprint(exchange.worker) || fingerprint(s) != fingerprint(exchange.settings) {
		return nil, hostedexecution.ErrConfiguration
	}
	return &WorkloadInventoryGuard{worker: c, settings: s, exchange: exchange, reader: reader}, nil
}

type workloadSnapshot struct{ applications, jobs []WorkloadPlacement }

func (g *WorkloadInventoryGuard) Check(ctx context.Context, c Config) error {
	_, err := g.inspect(ctx, c)
	return err
}

func (g *WorkloadInventoryGuard) inspect(ctx context.Context, c Config) (_ workloadSnapshot, resultErr error) {
	defer func() { resultErr = controlError(resultErr) }()
	if fingerprint(c) != fingerprint(g.worker) {
		return workloadSnapshot{}, hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 40*time.Second)
	defer cancel()
	if err := g.exchange.Check(ctx, c); err != nil {
		return workloadSnapshot{}, err
	}
	var applications, jobs []WorkloadPlacement
	var errs [2]error
	var group sync.WaitGroup
	group.Go(func() { applications, errs[0] = g.reader.Applications(ctx) })
	group.Go(func() { jobs, errs[1] = g.reader.Jobs(ctx) })
	group.Wait()
	for _, err := range errs {
		if errors.Is(err, hostedexecution.ErrOwnership) {
			return workloadSnapshot{}, hostedexecution.ErrOwnership
		}
	}
	for _, err := range errs {
		if err != nil {
			return workloadSnapshot{}, hostedexecution.ErrUnavailable
		}
	}
	if err := g.verifyPlacements(applications, "containerApps"); err != nil {
		return workloadSnapshot{}, err
	}
	if err := g.verifyPlacements(jobs, "jobs"); err != nil {
		return workloadSnapshot{}, err
	}
	return workloadSnapshot{applications: applications, jobs: jobs}, nil
}
