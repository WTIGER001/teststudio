package sources_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	workerclient "github.com/wtiger001/teststudio/adapters/workerexchange/client"
	workerhttp "github.com/wtiger001/teststudio/adapters/workerexchange/http"
	"github.com/wtiger001/teststudio/internal/apipolicy"
	"github.com/wtiger001/teststudio/internal/browsergateway"
	"github.com/wtiger001/teststudio/internal/browserpolicy"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/preparation"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/sources"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/workerexchange"
	"github.com/wtiger001/teststudio/internal/workerprotocol"
)

type hostedBuilder struct{ bundleBuilder }

func (b *hostedBuilder) Build(ctx context.Context, request preparation.BuildRequest) (preparation.Build, error) {
	result, err := b.bundleBuilder.Build(ctx, request)
	result.Image = "registry.fixture.test/teststudio/source@sha256:" + strings.Repeat("a", 64)
	return result, err
}

type discoveryProvider struct {
	addressCapable bool
	t              *testing.T
	source         *sources.Service
	builder        *hostedBuilder
	exchange       *workerexchange.Service
	server         *httptest.Server
	job            hostedexecution.Job
	status         *hostedexecution.Status
	mode           string
	created        int
	discovered     int
	removed        int
	deliveryError  error
}

func (p *discoveryProvider) InspectImage(_ context.Context, image string) (hostedexecution.Image, error) {
	value := hostedexecution.Image{Reference: image, Project: "alpha", SourceDigest: p.builder.digest, Runner: "go", Runtime: "go-discovery-fixture-v1", OS: "linux", Architecture: "arm64", Supervisor: "worker-exchange/v1", Discovery: "worker-exchange/v1"}
	if p.mode == "environment" {
		value.GatewayTransport, value.SupervisorRevision, value.APIPolicy = browsergateway.RemoteProtocol, strings.Repeat("a", 64), "request-policy/v1"
		if p.addressCapable {
			value.GatewayAddress = browsergateway.RemoteAddressProtocol
		}
	}
	return value, nil
}

func (p *discoveryProvider) Ensure(_ context.Context, job hostedexecution.Job) error {
	p.created++
	p.job = job
	p.status = &hostedexecution.Status{Reference: job.Reference, Image: job.Image, Deadline: job.Ticket.Deadline, Phase: "running"}
	if job.Operation != "discover" || !strings.HasSuffix(job.Reference.Profile, "-discovery-v1") {
		return hostedexecution.ErrOwnership
	}
	if p.mode == "admission-rejected" {
		return hostedexecution.ErrOwnership
	}
	return nil
}

func (p *discoveryProvider) Observe(ctx context.Context, ref execution.Reference) (hostedexecution.Status, error) {
	if p.status == nil {
		return hostedexecution.Status{}, hostedexecution.ErrNotFound
	}
	state, err := p.exchange.State(ctx, ref)
	if err == nil && !state.DispatchPending && !state.Revoked && p.status.Phase == "running" {
		p.discovered++
		client, err := workerclient.New(p.server.URL, ref.ID, "discovery-pod", p.job.Ticket.Token, p.server.Client().Transport)
		if err != nil {
			return hostedexecution.Status{}, err
		}
		defer client.Close()
		packet, err := client.Input(ctx)
		if err != nil || packet.Operation != "discover" {
			return hostedexecution.Status{}, hostedexecution.ErrOwnership
		}
		if p.mode == "revoke-access" {
			row, _ := p.source.Store.Get(ctx, sources.JobBucket("alpha", "health"), ref.Run)
			job, _ := storage.Decode[sources.Job](row)
			job.CancelRequested = true
			change, _ := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, job)
			p.source.Store.CommitAll(ctx, change)
		}
		result, err := (&inspector{count: 1, version: 1}).Inspect(ctx, preparation.InspectionRequest{Image: p.job.Image, Suite: "health", Reference: ref})
		if err != nil {
			return hostedexecution.Status{}, err
		}
		result.Discovery.Origin = p.builder.origin
		if p.mode == "environment" {
			result.Discovery.Manifest.Type, result.Discovery.Manifest.BuildProfile = "api", "go-environment"
			result.Discovery.Manifest.Browser = &browserpolicy.Spec{Endpoint: "api", Requests: &apipolicy.Spec{Rules: []apipolicy.Rule{{Method: "GET", PathPrefix: "/status", ReadOnly: true}}, MaxRequests: 1, MaxResponseBytes: 4096}}
			for index := range result.Discovery.Catalog.Tests {
				result.Discovery.Catalog.Tests[index].Type = "api"
			}
		}
		if p.mode == "wrong-source" {
			result.Discovery.Origin.SnapshotDigest = "sha256:" + strings.Repeat("f", 64)
		}
		raw, _ := json.Marshal(result.Discovery)
		submission := workerprotocol.Submission{ProcessExitCode: 0, Discovery: &workerprotocol.DiscoveryReport{Raw: raw, Log: "Owned remote discovery fixture. No test dispatch.\n"}}
		if err := client.Started(ctx, time.Now().UTC()); err == nil {
			err = client.Submit(ctx, submission)
			if err == nil {
				err = client.Submit(ctx, submission)
			}
			p.deliveryError = err
		} else {
			p.deliveryError = err
		}
		code := 0
		p.status.Phase = "succeeded"
		if p.deliveryError != nil || p.mode == "provider-failed" {
			code, p.status.Phase = 2, "failed"
		}
		p.status.ExitCode = &code
	}
	return *p.status, nil
}

func (p *discoveryProvider) Stop(_ context.Context, _ execution.Reference) error {
	if p.status != nil && p.status.ExitCode == nil {
		code := 137
		p.status.Phase, p.status.ExitCode = "stopped", &code
	}
	return nil
}

func (p *discoveryProvider) Remove(ctx context.Context, ref execution.Reference) error {
	if err := p.exchange.CheckCleanup(ctx, ref); err != nil {
		p.t.Error("discovery resource deletion preceded its source checkpoint", err)
		return err
	}
	p.removed++
	return nil
}

func (p *discoveryProvider) Inventory(context.Context, string, string) ([]execution.Reference, error) {
	return nil, nil
}

func hostedPreparation(t *testing.T, mode string) (*sources.Service, *discoveryProvider) {
	t.Helper()
	s, _, _ := preparedService(t)
	s.Location, s.BundleRunners = "kubernetes", []string{"go"}
	builder := &hostedBuilder{}
	exchange := &workerexchange.Service{Store: s.Store, Authorize: s.AuthorizeDiscovery, AuthorizeDiscoveryCleanup: s.AuthorizeDiscoveryCleanup}
	server := httptest.NewTLSServer(workerhttp.Handler(exchange))
	t.Cleanup(server.Close)
	provider := &discoveryProvider{t: t, source: s, builder: builder, exchange: exchange, server: server, mode: mode}
	inspector := &hostedexecution.DiscoveryInspector{Provider: provider, Exchange: exchange, Name: "kubernetes", Runner: "go", ExchangeOrigin: server.URL, PollInterval: time.Millisecond}
	s.Inspectors, s.RecoveryInspectors = map[string]preparation.ArtifactInspector{"go": inspector}, map[string]preparation.ArtifactInspector{"go": inspector}
	s.Builders, s.RecoveryBuilders = map[string]preparation.ArtifactBuilder{"go": builder}, map[string]preparation.ArtifactBuilder{"go": builder}
	return s, provider
}

func recoverHostedPreparation(t *testing.T, s *sources.Service, id string) sources.Job {
	t.Helper()
	row, err := s.Store.Get(t.Context(), sources.JobBucket("alpha", "health"), id)
	if err != nil {
		t.Fatal(err)
	}
	job, _ := storage.Decode[sources.Job](row)
	expired := time.Now().UTC().Add(-time.Second)
	job.LeaseUntil = &expired
	change, _ := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, job)
	if err := s.Store.CommitAll(t.Context(), change); err != nil {
		t.Fatal(err)
	}
	if err := s.RecoverPreparations(t.Context(), "recovery-worker"); err != nil {
		t.Fatal(err)
	}
	row, _ = s.Store.Get(t.Context(), row.Bucket, row.ID)
	job, _ = storage.Decode[sources.Job](row)
	return job
}

func TestHostedBundleDiscoveryReachesReviewApprovalAndCheckpointedCleanup(t *testing.T) {
	s, provider := hostedPreparation(t, "")
	queued, original := enqueueZIP(t, s, "hosted-source")
	if processed, err := s.ProcessNext(t.Context(), "api"); processed || err != nil || provider.created != 0 {
		t.Fatal("API consumed a source build", processed, err)
	}
	_, ready := processBundle(t, s, queued.ID)
	if ready.State != "ready" || !ready.CleanupPending || provider.created != 1 || provider.discovered != 1 || provider.removed != 0 || provider.deliveryError != nil || provider.builder.calls != 1 {
		t.Fatal("hosted preparation failed", ready.State, ready.Error, provider.deliveryError)
	}
	preview, err := s.Read(t.Context(), "admin", "alpha", "health", ready.Upload)
	if err != nil {
		t.Fatal(err)
	}
	upload, _ := storage.Decode[sources.Upload](preview.Upload)
	if upload.Artifact == nil || s.ApproveArtifact(t.Context(), *upload.Artifact) == nil {
		t.Fatal("unpublished hosted artifact became executable")
	}
	retained, err := s.ReadBundle(t.Context(), "admin", "alpha", "health", queued.ID)
	if err != nil || string(retained.Content) != string(original) || upload.Artifact.Provenance.SnapshotDigest != retained.Digest {
		t.Fatal("hosted source ownership or bytes changed", err)
	}
	if err := s.AuthorizeDiscoveryCleanup(t.Context(), *ready.Reference); err != nil {
		t.Fatal(err)
	}
	s.Inspectors, s.Builders = nil, nil
	finished := recoverHostedPreparation(t, s, queued.ID)
	if finished.State != "ready" || finished.CleanupPending || provider.removed != 1 || provider.builder.cleanups != 1 || provider.created != 1 {
		t.Fatal("checkpointed cleanup reran or discarded discovery", finished, provider.created)
	}
	release, err := s.Publish(t.Context(), "admin", "alpha", "health", ready.Upload, preview.Upload.Revision, false)
	if err != nil || s.ApproveArtifact(t.Context(), release.Draft.Artifact) != nil || release.Draft.Artifact.Image != ready.Image {
		t.Fatal("normal source approval did not bind the registry artifact", err)
	}
	if rows, _ := s.Store.All(t.Context(), "runs", "alpha"); len(rows) != 0 {
		t.Fatal("preparation or publication created a test run")
	}
}

func TestHostedReadyPreparationCanCancelBeforeDeferredCleanup(t *testing.T) {
	s, provider := hostedPreparation(t, "")
	queued, _ := enqueueZIP(t, s, "cancel-hosted-ready")
	_, ready := processBundle(t, s, queued.ID)
	row, err := s.Store.Get(t.Context(), sources.JobBucket("alpha", "health"), ready.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.CancelJob(t.Context(), "admin", "alpha", "health", ready.ID, row.Revision); err != nil {
		t.Fatal("ready discovery with an existing cleanup lease could not cancel", err)
	}
	finished := recoverHostedPreparation(t, s, ready.ID)
	if finished.State != "cancelled" || finished.CleanupPending || provider.created != 1 || provider.removed != 1 {
		t.Fatal("cancelled discovery was redispatched or left without cleanup", finished)
	}
}

func TestHostedTargetSuiteDiscoversOfflineBeforeReviewedPublication(t *testing.T) {
	s, provider := hostedPreparation(t, "environment")
	row, err := s.Store.Get(t.Context(), projects.Bucket("suites", "alpha"), "health")
	if err != nil {
		t.Fatal(err)
	}
	suite, _ := storage.Decode[projects.Suite](row)
	suite.RequiresTarget = true
	change, _ := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, suite)
	if err := s.Store.CommitAll(t.Context(), change); err != nil {
		t.Fatal(err)
	}
	queued, _ := enqueueZIP(t, s, "target-suite-discovery")
	_, ready := processBundle(t, s, queued.ID)
	if ready.State != "ready" || provider.created != 1 || provider.discovered != 1 || provider.deliveryError != nil || provider.job.Reference.Gateway != "" || provider.job.Operation != "discover" {
		t.Fatal("target suite did not reach offline discovery and review", ready.State, ready.Error, provider.deliveryError)
	}
	if rows, _ := s.Store.All(t.Context(), "runs", "alpha"); len(rows) != 0 {
		t.Fatal("catalog discovery dispatched test execution")
	}
}

func TestECSTargetDiscoveryFreezesAddressCapabilityAndKeepsLegacyOfflineSources(t *testing.T) {
	for _, mode := range []string{"environment approved", "environment missing address", "offline legacy"} {
		t.Run(mode, func(t *testing.T) {
			s, provider := hostedPreparation(t, "environment")
			s.Location = "ecs"
			inspector := s.Inspectors["go"].(*hostedexecution.DiscoveryInspector)
			inspector.Name, inspector.RequireGatewayAddress = "ecs", true
			provider.addressCapable = mode == "environment approved"
			if mode == "offline legacy" {
				provider.mode = ""
			}
			queued, _ := enqueueZIP(t, s, "ecs-address-capability")
			_, result := processBundle(t, s, queued.ID)
			if mode == "environment missing address" {
				if result.State != "failed" || result.Upload != "" || !errors.Is(provider.deliveryError, workerprotocol.ErrInvalid) {
					t.Fatal("unsupported ECS target source reached catalog review", result.State, result.Error, provider.deliveryError)
				}
			} else if result.State != "ready" || provider.deliveryError != nil {
				t.Fatal("compatible ECS discovery did not reach review", result.State, result.Error, provider.deliveryError)
			}
			if provider.created != 1 || provider.job.Reference.Gateway != "" || provider.job.Operation != "discover" {
				t.Fatal("source discovery started an application gateway or another workload")
			}
			if rows, _ := s.Store.All(t.Context(), "runs", "alpha"); len(rows) != 0 {
				t.Fatal("source preparation created a test run")
			}
		})
	}
}

func TestHostedDiscoveryRejectsWrongSourceRevocationAndAdmissionMutation(t *testing.T) {
	for _, mode := range []string{"wrong-source", "revoke-access", "admission-rejected", "provider-failed"} {
		t.Run(mode, func(t *testing.T) {
			s, provider := hostedPreparation(t, mode)
			queued, _ := enqueueZIP(t, s, mode)
			_, result := processBundle(t, s, queued.ID)
			if result.State != "failed" && result.State != "cancelled" || result.Upload != "" || !result.CleanupPending {
				t.Fatal(result)
			}
			if mode == "admission-rejected" && provider.discovered != 0 {
				t.Fatal("unverified provider creation received executable work")
			}
			if mode == "wrong-source" && !errors.Is(provider.deliveryError, workerprotocol.ErrInvalid) {
				t.Fatal("foreign source discovery was accepted", provider.deliveryError)
			}
			finished := recoverHostedPreparation(t, s, queued.ID)
			if finished.CleanupPending || provider.created != 1 || provider.builder.calls != 1 || provider.removed != 1 {
				t.Fatal("failed preparation cleanup repeated dispatch", finished)
			}
		})
	}
}
