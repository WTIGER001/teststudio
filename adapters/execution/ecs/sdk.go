package ecsworker

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/route53profiles"
	"github.com/aws/aws-sdk-go-v2/service/route53resolver"
	"github.com/aws/aws-sdk-go-v2/service/scheduler"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/wtiger001/teststudio/adapters/registry/oci"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/storage"
)

type Runtime struct {
	Provider *Provider
	Registry *oci.Client
	sdk      *sdkHTTP
	once     sync.Once
}

func (r *Runtime) Close() {
	r.once.Do(func() { r.Registry.Close(); r.sdk.CloseIdleConnections() })
}

func (s Settings) Open(ctx context.Context, store *storage.Store, cleanup func(context.Context, execution.Reference) error) (*Runtime, error) {
	if store == nil {
		return nil, hostedexecution.ErrConfiguration
	}
	worker, err := s.Network.Bind(s.Worker)
	if err != nil || s.Worker.NetworkProfile != worker.NetworkProfile || strings.HasPrefix(worker.Region, "cn-") || strings.HasPrefix(worker.Region, "us-gov-") {
		return nil, hostedexecution.ErrConfiguration
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	client := privateHTTP()
	loaded, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(worker.Region), awsconfig.WithHTTPClient(client), awsconfig.WithRetryMaxAttempts(1), awsconfig.WithDefaultsMode(aws.DefaultsModeStandard))
	if err != nil {
		client.CloseIdleConnections()
		return nil, hostedexecution.ErrConfiguration
	}
	result, err := s.open(ctx, store, cleanup, loaded.Credentials, client, nil)
	if err != nil {
		client.CloseIdleConnections()
	}
	return result, err
}

func (s Settings) open(ctx context.Context, store *storage.Store, cleanup func(context.Context, execution.Reference) error, credentials aws.CredentialsProvider, client aws.HTTPClient, registryTransport http.RoundTripper) (*Runtime, error) {
	worker, err := s.Network.Bind(s.Worker)
	if err != nil || s.Worker.NetworkProfile != worker.NetworkProfile || store == nil || credentials == nil || client == nil || strings.HasPrefix(worker.Region, "cn-") || strings.HasPrefix(worker.Region, "us-gov-") {
		return nil, hostedexecution.ErrConfiguration
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	transport := &sdkHTTP{base: client, region: worker.Region}
	cfg := aws.Config{Region: worker.Region, Credentials: credentials, HTTPClient: transport, RetryMaxAttempts: 1, Retryer: func() aws.Retryer { return aws.NopRetryer{} }, DefaultsMode: aws.DefaultsModeStandard}
	caller, err := sts.NewFromConfig(cfg).GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil || caller == nil {
		return nil, hostedexecution.ErrUnavailable
	}
	principal, err := arn.Parse(aws.ToString(caller.Arn))
	if err != nil || aws.ToString(caller.Account) != worker.Account || principal.AccountID != worker.Account || principal.Partition != "aws" || principal.Region != "" || principal.Service != "sts" && principal.Service != "iam" || principal.Resource == "" || strings.ContainsAny(principal.Resource, "\r\n\x00") {
		return nil, hostedexecution.ErrOwnership
	}
	auth := &registryAuth{client: ecr.NewFromConfig(cfg), credentials: credentials, account: worker.Account, origin: registryOrigin(worker), now: time.Now}
	registry, err := oci.New(oci.Options{Origin: auth.origin, RepositoryPrefix: s.Network.RepositoryPrefix, Architecture: worker.Architecture, Credentials: auth.Credential, Transport: registryTransport, BlobRedirectOrigins: []string{"https://prod-" + worker.Region + "-starport-layer-bucket.s3." + worker.Region + ".amazonaws.com", "https://prod-" + worker.Region + "-starport-layer-bucket.s3.amazonaws.com"}})
	if err != nil {
		return nil, hostedexecution.ErrConfiguration
	}
	guard, err := NewNetworkGuard(worker, s.Network, ec2.NewFromConfig(cfg), elasticloadbalancingv2.NewFromConfig(cfg), route53resolver.NewFromConfig(cfg), route53profiles.NewFromConfig(cfg))
	if err != nil {
		registry.Close()
		return nil, err
	}
	provider, err := New(Options{Config: worker, Store: store, ECS: ecs.NewFromConfig(cfg), Scheduler: scheduler.NewFromConfig(cfg), Inspector: registry, Network: guard, DiscoveryCleanup: cleanup})
	if err != nil {
		registry.Close()
		return nil, err
	}
	return &Runtime{Provider: provider, Registry: registry, sdk: transport}, nil
}
