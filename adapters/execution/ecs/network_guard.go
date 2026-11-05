package ecsworker

import (
	"context"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/route53profiles"
	"github.com/aws/aws-sdk-go-v2/service/route53resolver"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type NetworkGuard struct {
	worker     Config
	settings   NetworkSettings
	ec2        EC2
	elb        ELB
	resolver   Resolver
	profiles   DNSProfiles
	mu         sync.Mutex
	validUntil time.Time
	routeIDs   []string
}

var _ Network = (*NetworkGuard)(nil)

func NewNetworkGuard(c Config, settings NetworkSettings, ec2Client EC2, elbClient ELB, resolver Resolver, profiles DNSProfiles) (*NetworkGuard, error) {
	bound, err := settings.Bind(c)
	if err != nil || c.NetworkProfile != bound.NetworkProfile || ec2Client == nil || elbClient == nil || resolver == nil || profiles == nil {
		return nil, hostedexecution.ErrConfiguration
	}
	if settings.Gateway != nil {
		if _, ok := ec2Client.(GatewayEC2); !ok {
			return nil, hostedexecution.ErrConfiguration
		}
		selected := settings.Gateway.canonical()
		settings.Gateway = &selected
	}
	return &NetworkGuard{worker: bound, settings: settings, ec2: ec2Client, elb: elbClient, resolver: resolver, profiles: profiles}, nil
}

func oneEC2(options *ec2.Options)                    { options.RetryMaxAttempts = 1 }
func oneELB(options *elasticloadbalancingv2.Options) { options.RetryMaxAttempts = 1 }
func oneProfiles(options *route53profiles.Options)   { options.RetryMaxAttempts = 1 }
func oneResolver(options *route53resolver.Options)   { options.RetryMaxAttempts = 1 }

func (g *NetworkGuard) Admit(ctx context.Context, c Config) error {
	if fingerprint(canonical(c)) != fingerprint(g.worker) {
		return hostedexecution.ErrOwnership
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if ctx.Err() != nil {
		return hostedexecution.ErrUnavailable
	}
	if time.Now().Before(g.validUntil) {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := g.checkVPC(ctx); err != nil {
		return err
	}
	if g.settings.Gateway != nil {
		if err := g.checkGatewayNetwork(ctx); err != nil {
			return err
		}
	}
	if err := g.checkGroups(ctx); err != nil {
		return err
	}
	if err := g.checkEndpoints(ctx); err != nil {
		return err
	}
	if err := g.checkExchange(ctx); err != nil {
		return err
	}
	if err := g.checkDNS(ctx); err != nil {
		return err
	}
	g.validUntil = time.Now().Add(2 * time.Second)
	return nil
}
