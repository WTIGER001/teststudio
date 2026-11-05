# Azure sandbox backend endpoint and readiness binding

## Introduction

OpenRuntime now includes SDKBackendEndpoint in creation, image import, and live admission.
It joins the selected backend environment approval, endpoint, NIC, private address, endpoint subnet, security rules, and relay source subnet.
The observed private address must equal the address in the trusted relay command.
The existing relay readiness probe and worker forwarding use that same pinned backend address.

This verifies selected resource metadata and its readiness configuration binding.
It does not establish deployed routing, effective permissions, or a fresh controller-side HTTPS readiness observation.
The remaining protection dependencies stay mandatory. Native commands and templates still do not enable sandbox execution.

## Concepts

### Required endpoint settings

placement.services.backendEndpoint requires these fields:

| Field | Contract |
| --- | --- |
| endpoint and interface | Selected private endpoint and NIC in the configured subscription. They differ from worker-facing exchange and registry resources. |
| subnet and subnetPrefix | A dedicated private IPv4 endpoint subnet in the relay's VNet. It differs from the relay infrastructure subnet. |
| relaySubnetPrefix | Canonical private IPv4 source prefix for the relay infrastructure subnet. |
| securityGroup | Selected endpoint security group, distinct from execution and worker-facing endpoint groups. |

The endpoint and relay prefixes cannot overlap each other or the execution VNet.
The endpoint subnet has a size of /29 or larger. The relay subnet has a size of /27 or larger.
The existing backend.address must be usable in the endpoint subnet, outside Azure's reserved addresses.
Foreign resources, reused resources, mismatched addresses, overlaps, and stale profiles reject before credential construction.
The sandbox-placement/v9 profile binds these settings with the existing service and network configuration.
Older staged profiles require regeneration. No native command currently loads them.

BackendEndpointDefinitions returns two reviewable bodies: endpoint subnet and endpoint security group.
It makes no resource mutation and does not provision the private endpoint, NIC, or environment approval.

### Environment approval and endpoint ownership

The backend environment must still match its private service contract, infrastructure subnet, region, and disabled public access.
Its complete private connection inventory must contain exactly the selected approved endpoint.
The embedded environment connection and separate inventory must agree.
The complete private-link resource inventory must report the managedEnvironment group and expected regional private zone.
Its sole required member must match the observed NIC and any explicit endpoint IP configurations.
Missing and pending observations remain unavailable. Rejected approval, foreign resources, extra entries, or mismatched members reject.

The endpoint must target the selected backend environment and exactly its selected NIC and subnet.
The NIC must point back to the endpoint and expose exactly the configured private IPv4 address.
Public addresses, alternate services, extra interfaces, forwarding, unrelated workloads, and unrelated network attachments reject.
The existing independent endpoint and interface verifiers enforce those contracts.

The dedicated subnet reports exactly the selected endpoint and NIC IP configuration reference.
Default outbound access is disabled. Private endpoint and private-link network policies are enabled.
The expected security group attaches only to this subnet and has no NIC attachments.
Its complete custom rule list contains these rules:

| Direction | Priority | Decision | Scope |
| --- | --- | --- | --- |
| Inbound | 100 | Allow | Relay infrastructure subnet to the exact backend address on TCP port 443. |
| Inbound | 4096 | Deny | All remaining inbound traffic. |
| Outbound | 4096 | Deny | All outbound traffic. |

Broader sources, destinations, ports, protocols, or extra rules reject.
The relay source subnet must report the configured prefix and one successfully provisioned Microsoft.App/environments delegation.
The [relay network stage](azure-sandbox-relay-network.md) now verifies the selected VNet, egress policy, private registry path, and selected pull grant.
Workload ownership and complete effective authority remain separate protection work.

### Readiness runs from the relay network

The trusted service definition retains a platform readiness probe for GET /readyz with a four-second timeout.
The existing relay implementation opens a fresh HTTPS connection to the pinned backend address and requests /healthz.
It preserves the configured backend hostname for HTTP Host and TLS verification.
It sends no ticket, cookie, caller identity, or worker instance header.
The backend reply must report status up, service teststudio, and role worker.

The readiness implementation retains its three-second bound, strict 1 KiB response contract, normal TLS verification, and certificate lifetime requirement.
Backend unavailability returns 503 from /readyz while relay liveness stays independent.
The worker backend health handler also reports 503 when its database is unavailable.
The detailed wire contract remains in [relay readiness](worker-relay-readiness.md).

Native service admission checks the configured probe, fixed relay arguments, and Azure's reported healthy active revision.
The new endpoint stage binds the probe's configured address to the selected backend environment and NIC.
The controller does not acquire another route into the worker-facing private endpoint or perform a new direct readiness request.
Reported service health remains bounded metadata evidence. It is not a fresh controller-observed TLS probe.
Actual deployed readiness, route enforcement, trust, and dependency behavior remain separate deployment verification.

### Observation bounds and lifecycle

The stage reads six selected resources and two complete selected collections.
Network resources use API version 2025-03-01. Environment and connection resources use 2025-07-01.
Connections and private-link resources allow one item across at most two pages each.
The shared transport retains exact GET scope, complete pagination, a 1 MiB response bound, no redirects, no retries, and no mutations.
Each observation has a five-second bound and joins its selected reads.

Creation and source import always obtain fresh observations.
Successful evidence can supply concurrent live callers for at most 15 seconds from observation start.
Failed current full observations invalidate prior success. Failures are not cached.
Remaining protection, per-sandbox definitions, and worker authorization remain uncached.
Metadata 404 responses remain unavailable. They do not establish sandbox absence or permit another launch.
Canceled or closed readers return unavailable, and errors omit provider and credential diagnostics.
Runtime.Close closes the endpoint reader after joining owned command work.

## Tutorials

### Verify locally

```sh
go test -race ./adapters/execution/azuresandbox -run '^TestSandboxBackendEndpoint|^TestSandboxRuntimeBackendEndpoint' -count=1 -timeout=40s
```

Five new focused checks cover settings and definitions, metadata drift, bounded evidence, automatic runtime composition, and the existing TLS readiness contract.
Nineteen selected native checks pass with race detection in 3.120 seconds.
Two selected relay TLS checks pass in 1.393 seconds. The worker backend database-health check passes in 1.480 seconds.
Evidence is output/azure-sandbox-backend-endpoint/summary.json, native.log, relay-readiness.log, and backend-health.log.

The new protocol check joins native settings and observed endpoint metadata to the real relay readiness handler.
An owned loopback TLS transport supplies its configured backend host with normal certificate verification.
It verifies fresh connections, credential-free requests, outage and recovery responses, and independent liveness.
The existing relay checks separately cover production private-address pinning, certificate trust, and lifetime.
The database-health check uses local SQLite and the actual worker health handler.
Owned TLS listeners close through test cleanup. No cloud service, browser, container, UI build, demo restart, resource mutation, or remote push is involved.

### Complete native activation

The [relay network stage](azure-sandbox-relay-network.md) now binds the selected VNet and private registry path.
The [registry service stage](azure-sandbox-registry-service.md) now verifies access policies and complete reported consumer, member, and replication inventories.
Complete trusted consumer network ownership, workload placement, identity host attachment, effective grants, deny effects, and protected writes.
Then wire scoped execution and discovery commands and complete deployment packaging.
Verify actual private routing, platform readiness, TLS trust, dependency access, preview compatibility, and independent deadlines during separate deployment acceptance.

Primary contracts: [Microsoft Container Apps private endpoints](https://learn.microsoft.com/en-us/azure/container-apps/how-to-use-private-endpoint)
and [Microsoft private endpoint network policies](https://learn.microsoft.com/en-us/azure/private-link/disable-private-endpoint-network-policy).
Related guides: [trusted services](azure-sandbox-services.md), [runtime](azure-sandbox-runtime.md), and [relay readiness](worker-relay-readiness.md).
Acceptance 51 advances. The verified total remains 48 of 57, with nine open.
