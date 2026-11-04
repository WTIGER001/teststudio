package azureworker

import (
	"context"
	"time"

	n "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type PerimeterReader interface {
	SecurityGroup(context.Context, string) (n.SecurityGroup, error)
	RouteTable(context.Context, string) (n.RouteTable, error)
}

type PerimeterGuard struct {
	worker   Config
	settings PerimeterSettings
	topology *TopologyGuard
	reader   PerimeterReader
}

func NewPerimeterGuard(c Config, s PerimeterSettings, topology *TopologyGuard, reader PerimeterReader) (*PerimeterGuard, error) {
	s, err := s.canonical(c)
	if err != nil || topology == nil || reader == nil || fingerprint(topology.worker) != fingerprint(c) || fingerprint(topology.settings) != fingerprint(s.Topology) {
		return nil, hostedexecution.ErrConfiguration
	}
	return &PerimeterGuard{worker: c, settings: s, topology: topology, reader: reader}, nil
}

func (g *PerimeterGuard) Check(ctx context.Context, c Config) (resultErr error) {
	defer func() { resultErr = controlError(resultErr) }()
	if fingerprint(g.worker) != fingerprint(c) {
		return hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := g.topology.Check(ctx, c); err != nil {
		return err
	}
	group, err := g.reader.SecurityGroup(ctx, g.settings.Topology.SecurityGroup)
	if err != nil {
		return hostedexecution.ErrUnavailable
	}
	if err := g.verifyGroup(group); err != nil {
		return err
	}
	routes, err := g.reader.RouteTable(ctx, g.settings.Topology.RouteTable)
	if err != nil {
		return hostedexecution.ErrUnavailable
	}
	return g.verifyRoutes(routes)
}
