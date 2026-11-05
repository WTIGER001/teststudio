# Azure sandbox trusted service admission

## Introduction

OpenRuntime now includes SDKServices in creation, source import, and live admission.
The stage verifies the selected exchange relay and trusted worker backend through reported Container Apps metadata.
It reuses existing private environment, identity, template, revision, endpoint approval, and certificate verifiers.
The trusted templates have independent builders. Native settings require no ordinary Jobs configuration.

This is service metadata admission. The [backend endpoint stage](azure-sandbox-backend-endpoint.md) now joins approval, NIC address, and endpoint network policy.
These stages do not establish deployed routing, fresh controller-observed readiness, or effective service permissions.
The remaining protection dependencies stay mandatory. Native commands and deployment templates still do not enable sandbox execution.

## Concepts

### Settings and profile binding

placement.services requires relay and backend settings.
The relay environment remains placement.private.exchangeEnvironment.

| Settings | Required fields |
| --- | --- |
| relay | Application, certificate, infrastructure subnet, infrastructure resource group, immutable image, and pull identity pin. |
| backend | Application, environment, infrastructure subnet, infrastructure resource group, origin, private address, immutable image, and replica count. |
| backend dependencies | Database Key Vault secret URL, Azure Blob service URL, evidence container, and pull, startup, and runtime identity pins. |
| identity pin | Exact resource, nonzero client identifier, and nonzero principal identifier. |

The two applications and environments differ. Resource comparisons reject case-only aliases.
Execution, relay, and backend infrastructure subnets belong to distinct VNets.
Service infrastructure groups differ from one another and the sandbox group resource group.
The backend address is canonical private IPv4 outside the execution VNet.
Its complete origin and the exchange origin follow the existing strict relay-origin policy.
The observed backend environment domain must produce exactly the configured backend application origin.

Trusted service images use immutable digests in the selected registry outside the uploaded-source repository namespace.
Tagged images and images from that source namespace reject.
Only Key Vault secret references are accepted for the backend database configuration.
Plaintext database connection strings reject before credential construction.
The Blob URL, container name, and replica count follow the existing backend policy.

Four service identities differ from one another and the import, timer, controller, registry inspection, and metadata identities.
Resources, client identifiers, and principal identifiers cannot reuse those bindings.
The sandbox-placement/v9 profile includes these settings, required backend endpoint settings, and the existing admission bindings.
Changing service images, identities, origins, or dependencies changes launch and source-cache ownership.
Older staged profiles require regeneration. No native command currently loads them.

ServiceDefinitions returns four reviewable bodies: two fixed applications and their disabled platform authentication configurations.
It makes no service request or resource mutation. Environment, endpoint, identity, certificate, dependency, and protection packaging remain separate work.

### Private environment and relay ownership

Both environments must match their identifiers, names, types, regions, infrastructure subnets, and successful provisioning states.
They use internal VNet configuration, disabled public network access, the expected infrastructure groups, and one Consumption workload profile.
Application log destinations are none. Log Analytics credentials, Dapr telemetry credentials, unexpected environment identity, and custom environment certificate secrets reject.
The reported default domains must match the selected region's Container Apps domain.

The relay environment reports exactly the selected approved worker-facing private endpoint connection.
Its embedded connection and complete connection inventory must agree.
The complete private-link resource inventory must report the managedEnvironment group and expected regional private zone.
Pending, rejected, mismatched, or missing observations block admission.
Both environments must have empty HTTP route configuration inventories.

The selected relay certificate must match resource identity, region, successful provisioning, and the exchange hostname.
Its issue date must be current. Expiration must exceed the current time by two hours and five minutes.
Invalid, expired, pending, mismatched, or secret-bearing certificate metadata rejects.
This compares certificate metadata. It does not perform a TLS handshake or validate the deployed certificate chain.

### Fixed templates and identity exposure

The relay runs only /app/teststudio-exchange with fixed public origin, backend origin, backend address, health probes, resources, and scale bounds.
Its sole image-pull identity has lifecycle None. The relay template receives no database secret or runtime identity.
The relay custom hostname binds to the selected certificate. Insecure ingress rejects.

The backend runs only /app/teststudio with serve-role=worker, require-postgres=true, fixed worker port, and the Azure execution location.
Its database environment variable references the selected platform secret.
Its artifact configuration binds the selected Blob service, container, prefix, and explicit runtime identity client.
Pull and startup identities have lifecycle None. Only the selected runtime identity has lifecycle Main.
Actual identity client and principal identifiers must match their pins.

The complete application configuration and normalized template must match the fixed expected definition.
Extra containers, commands, arguments, identities, secrets, volumes, routing configuration, unknown application kind, and unexpected resource management ownership reject.
Platform authentication must explicitly report disabled because the worker exchange uses its own scoped ticket protocol.
This checks intended identity exposure. Effective grants, host reuse, federation, and deny effects remain separate authority work.

### Revisions and observations

Each complete revision inventory allows at most four entries across four pages.
Exactly one revision is active, healthy, provisioned, running, and equal to the application's latest ready revision.
It receives all traffic and uses the exact trusted template.
Inactive prior revisions must have zero replicas and zero traffic.
An accepted inactive flag alone does not establish quiescence.
Unknown, duplicate, extra active, mismatched, missing, or unfinished revision observations block admission.

The stage reads thirteen selected paths with API version 2025-07-01.
Seven resources and six bounded collections share a five-second observation bound.
The shared transport permits selected GET requests only, with no mutation, secret-list action, redirect, provider registration, or automatic retry.
Each response has a 1 MiB bound and closes its body.
Returned errors omit provider bodies, certificate material, and credential diagnostics.

Creation and image import obtain fresh observations.
Successful evidence can supply concurrent live callers for at most 15 seconds from observation start.
Current failed full observations invalidate earlier success. Failures are not cached.
Remaining protection, per-sandbox definitions, and current worker authorization remain uncached.
Metadata 404 responses remain unavailable and never establish sandbox absence or permit another launch.
Canceled or closed readers return unavailable.
Runtime.Close closes the service reader after joining owned command work.

## Tutorials

### Verify locally

```sh
go test -race ./adapters/execution/azuresandbox -run '^TestSandboxService|^TestSandboxRuntimeServices' -count=1 -timeout=40s
```

Six focused checks cover definition separation, service drift, rejected settings, complete revision paging, bounded evidence, and automatic runtime composition.
Nineteen selected native checks pass with race detection in 2.865 seconds.
Six affected shared relay and backend checks pass with race detection in 4.057 seconds.
Evidence is output/azure-sandbox-services/summary.json, native.log, and shared-services.log.
The checks use local SQLite, fake credentials, and in-memory ARM responses.
No cloud service, browser, container, UI build, demo restart, resource mutation, or remote push is involved.

### Finish activation

The backend endpoint stage now binds the selected environment and NIC to the address used by relay forwarding and readiness.
Readiness runs in the relay network. Native admission checks its exact probe and reported healthy active revision.
The [relay network stage](azure-sandbox-relay-network.md) now verifies the selected VNet, egress policy, registry endpoint, DNS, and pull grant.
The [registry service stage](azure-sandbox-registry-service.md) now verifies access policies and complete reported consumer, member, and replication inventories.
Complete trusted consumer network ownership, workload placement, identity host attachment, effective grants, deny effects, and protected write boundaries.
Then wire scoped execution and discovery commands and complete deployment packaging.
Verify actual private routing, TLS, service identity lifecycle behavior, dependency access, preview compatibility, and independent deadlines during separate deployment acceptance.

Primary contracts: [Microsoft Container Apps schema](https://learn.microsoft.com/en-us/azure/templates/microsoft.app/2025-07-01/containerapps)
and [Microsoft managed identities](https://learn.microsoft.com/en-us/azure/container-apps/managed-identity).
Related guides: [private endpoints](azure-sandbox-private.md), [selected authority](azure-sandbox-authority.md), and [runtime](azure-sandbox-runtime.md).
Acceptance 51 advances. The verified total remains 48 of 57, with nine open.
