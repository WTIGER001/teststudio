# Azure sandbox placement admission

## Introduction

SDKPlacement reads the configured sandbox group, connection, subnet, import identity, and selected registry pull grant through the ARM SDK pipeline.
OpenRuntime now constructs this reader and includes it in create, live-instance, and import admission.
Callers cannot omit this stage from the runtime factory.
The existing additional authority and network protection dependencies remain mandatory.

This verifies selected reported metadata. It does not prove complete effective authority or network enforcement.
Native commands and deployment templates still do not enable sandbox execution.

## Concepts

### Settings and profile binding

Sandbox Settings now requires metadataClientId and placement.
The metadata client differs from the control, registry inspection, and import client identifiers.
The placement reader sends only selected GET requests. The separate resolver reader permits one empty POST read action.
These transport restrictions do not prove the identity has no other effective privileges.
The [authority stage](azure-sandbox-authority.md) now verifies selected identity grants. Effective additional access and host attachment remain separate checks.

| placement field | Required binding |
| --- | --- |
| subnet and subnetPrefix | Exact delegated subnet resource in the selected subscription and one canonical private IPv4 prefix. |
| securityGroup and routeTable | Exact subnet attachments in the selected subscription. The network stage checks their contents; effective routing remains separate. |
| registry | Exact ARM registry resource matching worker.registry. |
| pullAssignment | Exact registry-scoped role assignment resource. A subscription-scoped assignment does not qualify. |
| pullClientId and pullPrincipalId | Explicit nonzero client and principal identifiers for pullIdentity. They differ from each other and the runtime client identifiers. |
| policyProfile | Existing protected network-policy fingerprint. This is an expected binding, not evidence that policy is effective. |
| network | Required private firewall address and bounded private endpoint addresses. See the network guide. |
| network.topology | Required private VNet prefix, firewall subnet prefix, and selected firewall, policy, and public IP resources. |
| network.resolver | Required selected resolver policy and distinct approved-domain and root-block lists. |
| private | Required endpoint, NIC, service, subnet, address, and private zone bindings. See the private admission guide. |
| services | Required relay and backend resources, immutable images, dependency references, and four distinct service identity pins. |
| services.backendEndpoint | Required backend endpoint, NIC, subnet, source prefix, and security group bindings. |
| services.relayNetwork | Required relay VNet, egress policy, resolver, private registry endpoint, zone, and selected pull assignment. |

PlacementSettings.NetworkProfile computes the combined profile from placement and the worker configuration with its networkProfile field cleared.
Set worker.networkProfile to that result. Settings validation rejects a mismatched profile before constructing credentials.
The sandbox-placement/v8 profile includes trusted services, backend endpoint and relay network bindings, and the existing resolver, topology, firewall, subnet, and DNS bindings.
Changes to these addresses, subnet, import identity metadata, selected grant, or expected policy change launch and source-cache ownership.
The factory adds one explicit managed identity credential for metadata reads. Construction still acquires no token and calls no service.
Earlier staged settings without placement reject. No native command has installed those earlier sandbox settings.

### Selected observations

Initial creation and source import each read seven resources in parallel:

| Resource | Pinned API version |
| --- | --- |
| Sandbox group and selected VNet connection | 2026-02-01-preview |
| Selected subnet | 2025-03-01 |
| Import user-assigned identity | 2024-11-30 |
| Selected registry | 2025-11-01 |
| Selected pull assignment | 2022-04-01 |
| Built-in AcrPull role definition in the selected subscription | 2022-05-01-preview |

The group must match its complete identifier, name, type, region, and successful provisioning state.
It reports exactly the selected user-assigned pull identity with the configured client and principal identifiers.
System-assigned identity and additional attached identities reject.
Known preview resource defaults are permitted. Unknown group properties, including a different environment-linked shape, reject.

The VNet connection must report the exact subnet, region, and successful state.
Unknown writable connection fields reject.
The subnet must report the expected prefix, security group, route table, and one Microsoft.App/environments delegation.
Default outbound access, NAT attachment, private endpoints, and service endpoints reject this execution-subnet contract.
These attachment checks do not verify the attached security rules, route contents, VNet peerings, or DNS resolution.
The required [network stage](azure-sandbox-network.md) now verifies reported security rules and route contents.
The required [private stage](azure-sandbox-private.md) now checks selected endpoint ownership, NIC addresses, endpoint subnet contents, and complete selected DNS inventories.
These attachment and private binding checks do not establish VNet topology, effective routing, firewall enforcement, trusted service ownership, or effective DNS resolution.
The required [topology stage](azure-sandbox-topology.md) now verifies selected VNet contents and complete firewall policy metadata.
The required [resolver stage](azure-sandbox-resolver.md) now checks selected policy, actual association, and reported subscription private-zone links.
The required [service stage](azure-sandbox-services.md) now verifies selected relay and backend metadata, identity exposure, and revisions.
The required [backend endpoint stage](azure-sandbox-backend-endpoint.md) now joins selected environment approval, NIC address, and network policy.
Deployed routing and readiness, effective service authority, visibility across subscriptions, and actual network behavior remain unverified.

The import identity reports the configured resource, region, tenant, client, and principal.
The registry reports Premium, successful provisioning, the exact login host, disabled public access, and no network-rule bypass.
Admin access and anonymous pull are disabled. Dedicated data endpoints are enabled.
LegacyRegistryPermissions is required because this check uses the fixed AcrPull role rather than repository ABAC permissions.

The selected assignment binds that principal to the registry and built-in AcrPull role as a ServicePrincipal.
Conditions, delegated identity, a different scope, and another principal reject.
The reported built-in role permits only the expected registry pull action, with no additional action, data action, exclusion, or condition.
This checks one selected grant. It does not establish the absence of other assignments, deny effects, or identity reuse.

### Admission and observation cost

Initial and import checks always start a new seven-resource observation. They do not use the completed live cache.
Successful observations can supply shared live placement evidence for at most 15 seconds from the observation start.
After expiry, concurrent live callers share one successful refresh of group, connection, and subnet.
Live checks do not reread source-only registry grants for an already converted image.
Failed observations are not cached. Failure of the current full observation invalidates successful live evidence.
An older observation cannot replace a newer full observation's cached result.

The controller still reads and verifies each owned sandbox definition without this cache.
The additional protection VerifyInstance callback still runs for every live admission call, including cache hits.
Worker authorization, source authorization, exchange revocation, and command reservation checks remain uncached.
The 15-second window applies to shared ARM placement metadata. It does not extend account or project permission revocation.
Actual ARM latency, throttling, and effective policy behavior remain deployment verification work.

### Request and cleanup bounds

Every observation has a five-second total bound. Canceled or closed readers start no metadata request.
The ARM transport permits only seven configured GET paths and their exact API versions.
Mutations, lists, filters, foreign scopes, redirects, and alternate versions reject.
Provider registration and automatic retries remain disabled.
Responses have a 1 MiB bound. Original response bodies close after buffering, including redirect and oversized responses.
Returned errors omit provider bodies and credential diagnostics.
Missing metadata remains unavailable. A metadata 404 never establishes worker absence or permits another launch.
No raw metadata observation or usable token enters the launch journal or result record.

## Tutorials

### Verify the changed boundary locally

```sh
go test -race ./adapters/execution/azuresandbox -run '^TestSandboxPlacement|^TestSandboxRunningPlacement|^TestSandboxRuntime' -count=1 -timeout=35s
```

Six new focused checks cover native composition, selected resources, ownership drift, pull grants, metadata gaps, profile binding, request bounds, and cancellation.
They also verify cache expiry, twelve-reader refresh coalescing, failure invalidation, and uncached protection loss.
The affected runtime, shared execution, and existing authorization checks also pass.
Evidence is output/azure-sandbox-placement/summary.json and targeted.log.
These checks use local SQLite, private filesystem evidence, fake credentials, and in-memory ARM transports.
No cloud account, identity endpoint, registry service, browser, container, demo restart, or remote push is involved.

### Complete native admission

Implement the remaining authority and protected network checks behind the mandatory admission dependencies.
The [relay network stage](azure-sandbox-relay-network.md) now verifies the selected relay VNet and private registry path.
Complete registry inventory, metadata-reader attachment, effective grants, write protection, and deployed routing and readiness.
The selected authority stage now covers the intended role and identity bindings. Verify effective access and attachment in addition to those reports.
Bind those checks to placement.policyProfile before command activation.
Then wire execution and discovery roles and package their resources.
Verify service conversion access to the private registry, token audience, VNet routing, and proxy certificate trust in the separate deployment phase.
The image-import API's identity parameter and this metadata check do not prove that private conversion path works.

The resource contract follows the pinned preview SDK and [Microsoft sandbox quickstart](https://learn.microsoft.com/en-us/azure/container-apps/sandboxes-quickstart-bicep).
The egress model remains the fixed sandbox policy described in [Microsoft network controls](https://learn.microsoft.com/en-us/azure/container-apps/sandboxes-egress-policies).
The selected pull grant follows [Microsoft managed identity image pull](https://learn.microsoft.com/en-us/azure/container-apps/managed-identity-image-pull).
Related contracts: [runtime assembly](azure-sandbox-runtime.md), [provider](azure-sandbox-provider.md), and [source images](azure-sandbox-images.md).
Acceptance 51 advances. The verified count remains 48 of 57, with nine open.
