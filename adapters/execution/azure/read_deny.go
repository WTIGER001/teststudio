package azureworker

import (
	"context"
	"time"

	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v3"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type ReadDenyReader interface {
	Assignments(context.Context) ([]a.DenyAssignment, error)
}

type ReadDenyGuard struct {
	worker   Config
	settings ReadIdentitySettings
	identity *ReadIdentityGuard
	reader   ReadDenyReader
}

func NewReadDenyGuard(c Config, s ReadIdentitySettings, identity *ReadIdentityGuard, reader ReadDenyReader) (*ReadDenyGuard, error) {
	s, err := s.canonical(c)
	if err != nil || identity == nil || reader == nil || fingerprint(c) != fingerprint(identity.worker) || fingerprint(s) != fingerprint(identity.settings) {
		return nil, hostedexecution.ErrConfiguration
	}
	return &ReadDenyGuard{worker: c, settings: s, identity: identity, reader: reader}, nil
}

func (g *ReadDenyGuard) Check(ctx context.Context, c Config) (resultErr error) {
	defer func() { resultErr = controlError(resultErr) }()
	if fingerprint(c) != fingerprint(g.worker) {
		return hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 130*time.Second)
	defer cancel()
	assignments, err := g.reader.Assignments(ctx)
	if err != nil {
		return err
	}
	if err := verifyReadDenies(c, g.settings, assignments); err != nil {
		return err
	}
	return g.identity.Check(ctx, c)
}
