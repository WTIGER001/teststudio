# Remaining deployment readiness

## Introduction

The current ledger has 50 verified items and seven open items.
This audit checks the remaining criteria against current code, deployment packages, and retained local evidence.
It adds no test runs, cloud connections, or new acceptance numbers.
The product remains unfinished. Some open work requires implementation, while other work requires deployment verification.
Azure implementation and verification remain deferred. A later Azure deployment script requires the user's request.

## Current profile coverage

| Profile | Authoritative current evidence | Remaining limits |
| --- | --- | --- |
| Local server | SQLite durability, managed identity, environment-backed browser and API execution, desktop recording, source publication, timing, evidence, and UI workflows. | This does not establish cloud deployment behavior. |
| Kubernetes | Actual local k3s installation, PostgreSQL, private evidence, two OIDC providers, source preparation, isolated offline Go and Playwright, native HTTPS, deadlines, and controller failover. | Environment-backed coded suites now have implementation and local protocol evidence. Actual network execution and hosted recording remain unverified or unfinished. |
| AWS ECS and CloudFront | Service and worker templates, private exchange, native SDK adapters, offline execution contracts, preparation packaging, and local protocol checks. | No live target verification. Environment-backed coded execution and hosted recording remain unfinished. |
| Azure | Core service and assembly packages, staged execution and sandbox adapters, and previously retained local protocol evidence. | Native coded execution and discovery wiring remain unfinished. The user defers this work and account verification. |

Local coverage is documented in [browser execution](browser-execution.md), [API tests](api-testing.md), and [browser recording](browser-recording.md).
Actual Kubernetes coverage is documented in [source execution](local-kubernetes-source-execution.md), [lifecycle](local-kubernetes-lifecycle.md), [failover](local-kubernetes-controllers.md), and [hosted identity](local-kubernetes-identity.md).
AWS and Azure packaging retain their separate [AWS](aws.md), [worker](aws-workers-package.md), and [Azure](azure.md) guides.
Packaging and SDK protocol checks do not prove target deployment acceptance.

## The seven open criteria

| Item | Current implementation and evidence | Missing implementation | Separate verification gate |
| --- | --- | --- | --- |
| 41 | Shared timing assessment, measured runner reports, immutable runtime profiles, and actual local and Kubernetes observations. | New hosted gateway profiles must retain the same measured-time and baseline contracts. Deferred Azure execution remains unwired. | Common timing behavior on actual ECS and Azure execution profiles. |
| 46 | Filesystem, S3, and Azure Blob adapters exist. Shared quotas, pins, retention, and local SDK protocol checks exist. | No additional shared evidence-engine gap is identified in this audit. | Actual provider identity, encryption, lifecycle, retention, quota, and pin behavior on S3 and Azure storage. |
| 51 | Kubernetes offline lifecycle has actual status, cancellation, deadline, artifact, and recovery evidence. ECS has native adapters and local contract evidence. | The ECS trusted gateway still needs implementation. Kubernetes network execution needs actual verification. Deferred Azure native execution and discovery wiring. | Actual ECS and Azure lifecycle scenarios, including independent termination and cleanup. |
| 53 | The local and Kubernetes installations work without AWS services or credentials. Azure has an independent Blob and identity configuration. | Deferred Azure installation and coded-execution wiring remain incomplete. | An actual independent Azure installation. |
| 54 | Two OIDC providers work through the shared directory locally and on Kubernetes. Project authorization and private evidence withdrawal pass. | No additional shared identity-engine gap is identified in this audit. | Equivalent configured-provider and project-scope scenarios on ECS and Azure. |
| 56 | Newly authored local suites and Kubernetes offline suites retain measured results, progress, history, and evidence. | Kubernetes environment-backed suites need actual verification. ECS trusted gateway delivery remains unfinished. Azure discovery and execution remain deferred. | Generic suites, reports, and trends on every required target. |
| 57 | All four deployment packages have implementation material. Local operation and substantial actual Kubernetes scenarios pass. | ECS gateway delivery, hosted recording, actual Kubernetes network verification, and deferred Azure wiring. | The complete common scenarios, installation behavior, and operational checks on each target. |

Keep each broad item unchecked until its complete requirement passes.
The completed generic failover and hosted-private-access contracts remain 52 and 55.
Their evidence does not close these separate cross-profile criteria.

## Confirmed implementation gaps

### Hosted browser and API networking

The [shared hosted admission](../internal/hostedexecution/admission.go) now permits environment-backed artifacts only through a configured gateway provider and inspected capable runtime.
Kubernetes implements that provider contract. ECS and Azure still reject these artifacts.
The [Kubernetes gateway component](kubernetes-gateway.md) now supports separate environment references and provider-owned gateway bundles.
Its normal executor registration, image approval, source discovery, and Helm path now pass targeted local checks.
The deployed offline worker policy still permits only the private exchange and DNS.
These are deliberate safety boundaries. Removing the checks does not implement hosted networking.

The working local [gateway adapter](../adapters/execution/localcontainer/gateway.go) starts a separate trusted container.
The [gateway engine](../internal/browsergateway/gateway.go) already checks the reviewed target and guarded network destination.
Its [API handler](../internal/browsergateway/api.go) enforces methods, paths, request counts, body limits, credential injection, and reflected-secret redaction.
The existing Playwright bridge reaches that gateway through an owned Unix socket.
That transport and Docker lifecycle do not provide a remote Kubernetes or ECS gateway.

Implement a provider-owned gateway and authenticated transport before admitting the corresponding hosted artifact.
Reuse the current target, request-policy, credential, redaction, timing, and execution-ledger contracts.
Keep uploaded test code outside the trusted gateway's credentials and network identity.

### Hosted recording

The [native recording driver](../adapters/recording/localplaywright/driver.go) starts Playwright on the server's desktop.
The [service command](../cmd/teststudio/main.go) requires explicit loopback-only local mode when that recorder is configured.
The hosted packages cannot turn that process into the tester's desktop browser.
The current browser-recording guide correctly reports this limit.

Implement a local companion for hosted TestStudio, reusing the existing trusted recording helper and source conversion.
Bind pairing, recording, stopping, code submission, permissions, and expiry to the requesting user, project, suite, and recording session.
Keep that authorization narrower than the user's application session.
Preserve the existing separation between recording, draft saving, publication, and execution.
A streamed browser remains an alternative delivery mechanism, not an additional requirement for the first companion workflow.

### Azure native wiring

The main service and headless execution controller currently select the implemented local, ECS, or Kubernetes execution registrations.
The preparation command has a separate Azure assembly-only path.
Staged sandbox and provider packages do not establish enabled native coded execution or complete discovery in an installation.
Preserve that code and its existing evidence. Add no Azure inspectors, command wiring, or deployment script during this deferred phase.

## Active implementation sequence

### 1. Kubernetes trusted gateway

Deliver one complete environment-backed browser and coded API path on the current dedicated local cluster.
Use a separate trusted gateway workload with its own network identity and an immutable installation-approved image.
The untrusted runner can reach only its owned gateway, private exchange, and declared DNS services.
The gateway applies the existing reviewed target and request policy before outbound delivery.
Do not grant direct target egress to the runner as a shortcut.

Bind gateway initialization and transport to the frozen execution, current dispatch authority, and deadline.
The [remote transport component](remote-gateway.md) now provides an authenticated TLS gateway and a supervisor-owned Unix socket bridge.
Shared private initialization, instance binding, frozen credential references, and supervisor delivery now pass targeted local checks.
The Kubernetes bundle provider now verifies resource identity and implements scoped policies, inventory, shutdown, and guarded cleanup.
Normal admission, source discovery, image capability approval, and installation wiring are implemented. Actual network execution remains unverified.
Resolve application credential references only inside the trusted delivery boundary.
Credential values must not enter uploaded source, runner input, diagnostics, or durable execution metadata.
Keep gateway configuration distinct from the ordinary worker packet and activation authority.

Extend the provider's durable reference, admission, inventory, stop, deadline, recovery, and cleanup to include the gateway.
Confirm both workload identities before releasing uploaded code through activation.
Preserve unknown observations and pending cleanup when termination is uncertain.
Use distinct runtime profile identities when measurement or resource boundaries change.
Keep offline discovery and the existing offline execution profile available.

Verification uses newly authored local website and API targets with bounded cases.
One reviewed successful flow and targeted policy, credential, cancellation, and recovery checks must cover the changed boundary.
Reuse the existing registry, identity, database, evidence, and runtime builds wherever their inputs stay unchanged.
Do not repeat passing offline suites or run calibration loops to create timing history.
Port the completed shared gateway contract to ECS after the Kubernetes path works.

### 2. Hosted recorder companion

Add the local companion and its scoped pairing and recording workflow to the current authoring wizard.
Keep ordinary configuration in the guided flow and uncommon connection settings under Advanced.
Update contextual help, API documentation, and the operator guide with the actual workflow.
Use targeted local verification and close every owned recording browser and helper process.

### 3. Target verification when separately authorized

Keep live AWS and Azure verification outside the current local implementation session.
AWS verification uses the previously selected MMDev account when the user authorizes that deployment work.
Azure needs a later account and deployment-script request. Its deferral stays in force.
Retain the full original target scope and seven open criteria while these gates remain unsatisfied.

Related guides: [hosted lifecycle](hosted-execution.md), [controller package](kubernetes-controllers.md), and [acceptance ledger](implementation.md).
