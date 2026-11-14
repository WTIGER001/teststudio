# Remaining deployment readiness

## Introduction

The current ledger has 50 verified items and six open items out of 56 active criteria.
The user retired item 54 on 2026-11-14. Multiple OIDC providers across deployment profiles are no longer a completion gate.
Existing configurable OIDC, identity isolation, and project permissions remain supported and required under their existing criteria.
This audit checks the remaining criteria against current code, deployment packages, and retained local evidence.
It adds no test runs, cloud connections, or new acceptance numbers.
The product remains unfinished. Some open work requires implementation, while other work requires deployment verification.
The user now plans a direct installation into their own Azure subscription. Follow [the Azure deployment plan](azure-deployment-plan.md).
The target and implementation milestones remain pending. Historical Azure inspector work and Marketplace delivery stay deferred.

## Current profile coverage

| Profile | Authoritative current evidence | Remaining limits |
| --- | --- | --- |
| Local server | SQLite durability, managed identity, environment-backed browser and API execution, desktop recording, source publication, timing, evidence, and UI workflows. | This does not establish cloud deployment behavior. |
| Kubernetes | Actual local k3s installation, PostgreSQL, private evidence, two OIDC providers, source preparation, offline suites, environment-backed browser and authenticated HTTPS API suites, gateway lifecycle, deadlines, controller failover, and the desktop recorder handoff now pass. | The owned local cluster does not establish another installation's routing, storage, runtime, or desktop trust configuration. |
| AWS ECS and CloudFront | Actual MMDev core deployment, PostgreSQL TLS, Cognito sign-in, one measured native HTTPS probe, private S3 evidence, and matching public UI assets now pass. Worker and gateway packages retain local contract evidence. | Actual coded workers, preparation, lifecycle, storage policies, and recorder handoff remain unverified. |
| Azure | Core service and assembly packages, staged execution and sandbox adapters, and previously retained local protocol evidence. | Direct installation planning is selected. The target remains unselected; native coded execution and discovery wiring remain unfinished. |

Local coverage is documented in [browser execution](browser-execution.md), [API tests](api-testing.md), and [browser recording](browser-recording.md).
Actual Kubernetes coverage is documented in [source execution](local-kubernetes-source-execution.md), [lifecycle](local-kubernetes-lifecycle.md), [failover](local-kubernetes-controllers.md), and [hosted identity](local-kubernetes-identity.md).
AWS and Azure packaging retain their separate [AWS](aws.md), [worker](aws-workers-package.md), and [Azure](azure.md) guides.
Packaging and SDK protocol checks do not prove target deployment acceptance.

## The six open criteria

| Item | Current implementation and evidence | Missing implementation | Separate verification gate |
| --- | --- | --- | --- |
| 41 | Shared timing assessment, measured reports, immutable profiles, local and Kubernetes observations, and one actual AWS native HTTPS measurement. | Deferred Azure execution remains unwired. | Common timing behavior on actual ECS and Azure execution profiles. |
| 46 | Filesystem, S3, and Azure Blob adapters exist. Shared quotas, pins, retention, and local SDK protocol checks exist. Actual AWS private evidence upload and checksum-verified download now pass. | No additional shared evidence-engine gap is identified in this audit. | Actual provider identity, encryption, lifecycle, retention, quota, and pin behavior on S3 and Azure storage. |
| 51 | Kubernetes gateway workflows have actual lifecycle evidence. ECS orchestration, native admission, registration, preparation, and environment packaging pass local checks. | Deferred Azure native execution and discovery wiring. | Actual ECS and Azure lifecycle scenarios. |
| 53 | The local and Kubernetes installations work without AWS services or credentials. Azure has an independent Blob and identity configuration. | Deferred Azure installation and coded-execution wiring remain incomplete. | An actual independent Azure installation. |
| 56 | Newly authored local suites, Kubernetes offline suites, and Kubernetes environment-backed suites retain measured results, progress, history, and evidence. ECS gateway delivery and packaging pass local contract checks. | Azure discovery and execution remain deferred. | Coded suites, reports, and trends on remaining required targets. The AWS core now retains one newly authored native HTTPS run. |
| 57 | All four deployment packages have implementation material. Actual AWS core operation now passes. Local operation and actual Kubernetes browser, API, lifecycle, identity, and desktop recording pass. The companion also has local UI, macOS package, and opt-in Kubernetes and ECS package evidence. | Deferred Azure wiring remains incomplete. | Complete common scenarios on ECS and Azure, including their deployed recorder handoff. |

Keep each broad item unchecked until its complete requirement passes.
The completed generic failover and hosted-private-access contracts remain 52 and 55.
Their evidence does not close these separate cross-profile criteria.

## Implementation status and remaining gaps

### Hosted browser and API networking

The [shared hosted admission](../internal/hostedexecution/admission.go) now permits environment-backed artifacts only through a configured gateway provider and inspected capable runtime.
Kubernetes and optional ECS environment profiles implement that provider contract. Azure native execution wiring remains deferred.
ECS defaults remain offline-only. Environment execution requires complete separate settings and current exchange authority.
The [ECS gateway provider](ecs-gateway.md) now has durable role journals, capability admission, verified address binding, and guarded cleanup.
Its composed parent waits for private initialization and healthy gateway status before creating one worker under current run authority.
Targeted local checks use the real private exchange and authored AWS protocol fixtures, including concurrent observation and partial cancellation.
The [native gateway network guard](ecs-gateway-network.md) now verifies private task ownership, narrow peering, trusted NAT, and separate private exchange delivery.
Production SDK selection now preserves the offline pool and enables environment providers only with complete separate settings and current authority.
ECS source assembly and discovery enforce the verified-address capability for environment metadata.
The [optional environment package](aws-environment-package.md) now provides private networks, gateway permissions, certificate references, repository, and combined settings.
Its rendered resources pass local schema and native admission checks. Actual AWS coded execution remains unverified.
The [Kubernetes gateway component](kubernetes-gateway.md) now supports separate environment references and provider-owned gateway bundles.
Its normal executor registration, image approval, source discovery, and Helm path now pass targeted local checks.
The deployed offline worker policy still permits only the private exchange and DNS.
These are deliberate safety boundaries. Removing the checks does not implement hosted networking.
Actual local gateway verification now passes through both normal source and execution workflows.
See [local application networking](local-kubernetes-gateway.md) for exact run identifiers, artifacts, integration fixes, and remaining lifecycle scope.

The working local [gateway adapter](../adapters/execution/localcontainer/gateway.go) starts a separate trusted container.
The [gateway engine](../internal/browsergateway/gateway.go) already checks the reviewed target and guarded network destination.
Its [API handler](../internal/browsergateway/api.go) enforces methods, paths, request counts, body limits, credential injection, and reflected-secret redaction.
The existing Playwright bridge reaches that gateway through an owned Unix socket.
That transport and Docker lifecycle do not provide a remote Kubernetes or ECS gateway.

Retain provider-owned gateways and authenticated transport as requirements for admitting hosted artifacts.
Reuse the current target, request-policy, credential, redaction, timing, and execution-ledger contracts.
Keep uploaded test code outside the trusted gateway's credentials and network identity.

### Hosted recording

The [native recording driver](../adapters/recording/localplaywright/driver.go) starts Playwright on the server's desktop.
The [service command](../cmd/teststudio/main.go) requires explicit loopback-only local mode when that recorder is configured.
The hosted packages cannot turn that process into the tester's desktop browser.
The current browser-recording guide correctly reports this limit.

The local companion now has a scoped pairing API and desktop command that reuse the trusted recording helper and source conversion.
Targeted local lifecycle and authored native helper checks pass. The guided connection workflow now also passes with actual native Chromium against a local installation.
Standalone macOS ARM64 distribution now passes offline setup. Opt-in Kubernetes and ECS configuration pass focused local checks.
The recorder handoff now also passes on the actual dedicated local Kubernetes installation through HTTPS, OIDC, PostgreSQL, and real native Chromium.
See [local Kubernetes recording](local-kubernetes-recording.md). Actual AWS and Azure profile verification remains separate.
Bind pairing, recording, stopping, code submission, permissions, and expiry to the requesting user, project, suite, and recording session.
Keep that authorization narrower than the user's application session.
Preserve the existing separation between recording, draft saving, publication, and execution.
A streamed browser remains an alternative delivery mechanism, not an additional requirement for the first companion workflow.

### Azure native wiring

The main service and headless execution controller currently select the implemented local, ECS, or Kubernetes execution registrations.
The preparation command has a separate Azure assembly-only path.
Staged sandbox and provider packages do not establish enabled native coded execution or complete discovery in an installation.
Preserve that code and its existing evidence. Add no Azure inspectors, command wiring, or deployment script during this deferred phase.

## Retained milestones and next gates

### 1. Kubernetes trusted gateway

Both environment-backed browser and authenticated HTTPS API paths now pass on the dedicated local cluster.
Gateway cancellation and interrupted-run recovery also pass with the same execution identities and complete resource cleanup.
Retain these results and the completed ECS integration checks. Actual ECS deployment verification remains separate work.
Use a separate trusted gateway workload with its own network identity and an immutable installation-approved image.
The untrusted runner can reach only its owned gateway, private exchange, and declared DNS services.
The gateway applies the existing reviewed target and request policy before outbound delivery.
Do not grant direct target egress to the runner as a shortcut.

Bind gateway initialization and transport to the frozen execution, current dispatch authority, and deadline.
The [remote transport component](remote-gateway.md) now provides an authenticated TLS gateway and a supervisor-owned Unix socket bridge.
Shared private initialization, instance binding, frozen credential references, and supervisor delivery now pass targeted local checks.
The Kubernetes bundle provider now verifies resource identity and implements scoped policies, inventory, shutdown, and guarded cleanup.
Normal admission, source discovery, image capability approval, installation wiring, and actual network execution now pass local verification.
Resolve application credential references only inside the trusted delivery boundary.
Credential values must not enter uploaded source, runner input, diagnostics, or durable execution metadata.
Keep gateway configuration distinct from the ordinary worker packet and activation authority.

The provider's durable reference, admission, inventory, stop, deadline, recovery, and cleanup now include the gateway.
Confirm both workload identities before releasing uploaded code through activation.
Preserve unknown observations and pending cleanup when termination is uncertain.
Use distinct runtime profile identities when measurement or resource boundaries change.
Keep offline discovery and the existing offline execution profile available.

Verification uses newly authored local website and API targets with bounded cases.
One reviewed successful flow and targeted policy, credential, cancellation, and recovery checks must cover the changed boundary.
Reuse the existing registry, identity, database, evidence, and runtime builds wherever their inputs stay unchanged.
Do not repeat passing offline suites or run calibration loops to create timing history.
The ECS adapter now implements the shared gateway contract and has local protocol evidence. Do not infer actual AWS execution from those checks.

### 2. Hosted recorder companion

The companion, guided wizard, contextual help, API contract, and operator package now have focused local evidence.
The native browser workflow closes all sixteen owned processes and retains one reviewed source draft without running its target test.
Kubernetes and ECS now have default-disabled opt-in settings. Their local package checks preserve API role boundaries and existing routes.
The native recorder handoff passes on Helm revision 15. Current revision 16 updates bundled UI styles and reuses all three unchanged service binaries.
The gateway and runtime choices remain unchanged.
Actual native capture, owner stop, retained source, and draft saving pass. All 23 owned verification processes exit.
The original twelve offline and seven application-networking runs remain unchanged. No target test runs are added.
Keep other passing browser and execution scenarios as retained evidence. Do not rerun the complete profile matrix.

### 3. Target verification when separately authorized

The user authorizes MMDev deployment on 2026-11-05, using the existing ECS cluster and service name teststudioapi.
The [MMDev deployment guide](aws-mmdev.md) records actual changes and evidence. Complete provider acceptance remains open.
Azure needs a later account and deployment-script request. Its deferral stays in force.
Retain the required target scope and six open criteria while these gates remain unsatisfied. Retired item 54 adds no gate.

Related guides: [hosted lifecycle](hosted-execution.md), [controller package](kubernetes-controllers.md), and [acceptance ledger](implementation.md).
