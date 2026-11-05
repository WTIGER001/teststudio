package azureworker

import (
	"context"
	"sync"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type Observer struct {
	mu    sync.RWMutex
	guard interface {
		Check(context.Context, Config) error
	}
	readers []observerReader
}

type observerReader interface{ Close() }

type observerBuilder struct {
	worker     Config
	credential azcore.TokenCredential
	transport  policy.Transporter
	readers    []observerReader
	err        error
}

func (g *ReadIdentityGuard) BindObserver(ctx context.Context, c Config, transport policy.Transporter) (*Observer, error) {
	return g.bindObserver(ctx, c, nil, transport)
}

func (g *ReadIdentityGuard) BindHostObserver(ctx context.Context, c Config, s ObserverHostSettings, transport policy.Transporter) (*Observer, error) {
	s, err := s.canonical(c)
	if err != nil || fingerprint(s.Identity) != fingerprint(g.settings) {
		return nil, hostedexecution.ErrConfiguration
	}
	return g.bindObserver(ctx, c, &s, transport)
}

func (g *ReadIdentityGuard) bindObserver(ctx context.Context, c Config, host *ObserverHostSettings, transport policy.Transporter) (*Observer, error) {
	credential, err := g.Credential(ctx, c)
	if err != nil {
		return nil, err
	}
	if ctx.Err() != nil {
		return nil, hostedexecution.ErrUnavailable
	}
	return newSDKObserver(c, g.settings, credential, host, transport)
}

func newSDKObserver(c Config, s ReadIdentitySettings, credential azcore.TokenCredential, host *ObserverHostSettings, transport policy.Transporter) (*Observer, error) {
	s, err := s.canonical(c)
	if err != nil || credential == nil {
		return nil, hostedexecution.ErrConfiguration
	}
	b := &observerBuilder{worker: c, credential: credential, transport: transport}
	endpoint := b.backend(s.Protection.PrivateDNS.Endpoint.DNS.Firewall.Perimeter.Topology.BackendEndpoint)
	identity := b.relay(s, endpoint)
	guard := observerStage(b, func() (*SDKReadDenyReader, error) {
		return NewSDKReadDenyReader(c, credential, transport)
	}, func(reader *SDKReadDenyReader) (*ReadDenyGuard, error) {
		return NewReadDenyGuard(c, s, identity, reader)
	})
	var check interface {
		Check(context.Context, Config) error
	} = guard
	if host != nil {
		check = observerStage(b, func() (*SDKObserverHostReader, error) {
			return NewSDKObserverHostReader(c, *host, credential, transport)
		}, func(reader *SDKObserverHostReader) (*ObserverHostGuard, error) {
			return NewObserverHostGuard(c, *host, guard, reader)
		})
	}
	if b.err != nil {
		closeObserverReaders(b.readers)
		return nil, controlError(b.err)
	}
	return &Observer{guard: check, readers: b.readers}, nil
}

func (o *Observer) Check(ctx context.Context, c Config) error {
	o.mu.RLock()
	defer o.mu.RUnlock()
	if o.guard == nil {
		return hostedexecution.ErrUnavailable
	}
	return o.guard.Check(ctx, c)
}

func (o *Observer) Close() {
	o.mu.Lock()
	defer o.mu.Unlock()
	closeObserverReaders(o.readers)
	o.guard, o.readers = nil, nil
}

func closeObserverReaders(readers []observerReader) {
	for i := len(readers) - 1; i >= 0; i-- {
		readers[i].Close()
	}
}

func observerStage[R observerReader, G any](b *observerBuilder, reader func() (R, error), guard func(R) (G, error)) (result G) {
	if b.err != nil {
		return result
	}
	r, err := reader()
	if err != nil {
		b.err = err
		return result
	}
	b.readers = append(b.readers, r)
	result, b.err = guard(r)
	return result
}
