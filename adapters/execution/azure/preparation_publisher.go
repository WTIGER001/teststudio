package azureworker

import (
	"context"
	"sync"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/wtiger001/teststudio/adapters/registry/acr"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type PreparationPublisher struct {
	mu      sync.RWMutex
	guard   *PreparationWriterGuard
	host    *PreparationHostGuard
	network *PreparationNetworkGuard
	readers []observerReader
}

func (s PreparationSettings) OpenPublisher(credential azcore.TokenCredential, transport policy.Transporter) (*PreparationPublisher, error) {
	c := s.Worker
	var err error
	s.Writer, err = s.Writer.canonical(c)
	if err != nil || credential == nil {
		return nil, hostedexecution.ErrConfiguration
	}
	if s.Network != nil && s.ValidateNative() != nil {
		return nil, hostedexecution.ErrConfiguration
	}
	b := &observerBuilder{worker: c, credential: credential, transport: transport}
	registry := b.registry(s.Writer.Registry.Registry)
	network := observerStage(b, func() (*SDKPreparationRegistryReader, error) {
		return NewSDKPreparationRegistryReader(c, s.Writer.Registry, credential, transport)
	}, func(reader *SDKPreparationRegistryReader) (*PreparationRegistryGuard, error) {
		dns, err := NewSDKPreparationRegistryDNSReader(c, s.Writer.Registry, credential, transport)
		if err != nil {
			return nil, err
		}
		b.readers = append(b.readers, dns)
		return NewPreparationRegistryGuard(c, s.Writer.Registry, registry, reader, dns)
	})
	guard := observerStage(b, func() (*SDKPreparationWriterReader, error) {
		return NewSDKPreparationWriterReader(c, s.Writer, credential, transport)
	}, func(reader *SDKPreparationWriterReader) (*PreparationWriterGuard, error) {
		return NewPreparationWriterGuard(c, s.Writer, network, reader)
	})
	var host *PreparationHostGuard
	if s.Host != nil {
		host = observerStage(b, func() (*SDKPreparationHostReader, error) {
			return NewSDKPreparationHostReader(s, credential, transport)
		}, func(reader *SDKPreparationHostReader) (*PreparationHostGuard, error) {
			return NewPreparationHostGuard(s, guard, reader)
		})
	}
	var admission *PreparationNetworkGuard
	if s.Network != nil {
		admission = observerStage(b, func() (*SDKPreparationNetworkReader, error) {
			return NewSDKPreparationNetworkReader(s, credential, transport)
		}, func(reader *SDKPreparationNetworkReader) (*PreparationNetworkGuard, error) {
			dns, err := NewSDKPreparationFileDNSReader(s, credential, transport)
			if err != nil {
				return nil, err
			}
			b.readers = append(b.readers, dns)
			return NewPreparationNetworkGuard(s, host, reader, dns)
		})
	}
	if b.err != nil {
		closeObserverReaders(b.readers)
		return nil, controlError(b.err)
	}
	return &PreparationPublisher{guard: guard, host: host, network: admission, readers: b.readers}, nil
}

func (p *PreparationPublisher) OpenRegistry(ctx context.Context, c Config) (*acr.Client, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.guard == nil {
		return nil, hostedexecution.ErrUnavailable
	}
	if err := p.check(ctx, c); err != nil {
		return nil, err
	}
	return p.guard.openRegistry(ctx, c)
}

func (p *PreparationPublisher) Check(ctx context.Context, c Config) error {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.guard == nil {
		return hostedexecution.ErrUnavailable
	}
	return p.check(ctx, c)
}

func (p *PreparationPublisher) check(ctx context.Context, c Config) error {
	if p.network != nil {
		return p.network.Check(ctx, c)
	}
	if p.host != nil {
		return p.host.Check(ctx, c)
	}
	return p.guard.Check(ctx, c)
}

func (p *PreparationPublisher) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	closeObserverReaders(p.readers)
	p.guard, p.host, p.network, p.readers = nil, nil, nil, nil
}
