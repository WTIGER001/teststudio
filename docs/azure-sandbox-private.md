# Azure sandbox private endpoint and DNS admission

## Introduction

OpenRuntime now constructs SDKPrivate with the existing metadata identity.
Creation, live admission, and source import cannot omit this stage.
It binds the allowed private addresses to selected endpoint NICs, services, subnet contents, security rules, and private DNS records.
The implementation reuses existing endpoint, NIC, subnet, and DNS verifiers through independent binding types.
It does not require ordinary Jobs settings.

These are reported resource checks. They do not prove effective DNS resolution, service ownership, or deployed preview-service compatibility.
Native commands and deployment templates still do not enable sandbox execution.

## Concepts

### Required settings

placement.private supplies these bindings:

| Field | Contract |
| --- | --- |
| subnet and subnetPrefix | A distinct canonical private IPv4 subnet in the sandbox connection's VNet. It cannot overlap the execution subnet. |
| securityGroup | A distinct selected endpoint security group in the subscription. |
| registryEndpoint and registryInterface | Exact distinct ARM resources for the registry private endpoint and NIC. |
| exchangeEndpoint and exchangeInterface | Exact distinct ARM resources for the exchange private endpoint and NIC. |
| exchangeEnvironment | Selected managed environment serving the trusted exchange relay. Application and backend ownership require additional checks. |
| registryZone | Exact selected privatelink.azurecr.io zone. |
| exchangeZone | Exact selected zone matching the configured private exchange hostname. |
| registryIp, registryDataIp, and exchangeIp | Three distinct usable addresses in the endpoint subnet. Their sorted set must equal placement.network.privateEndpointIps. |

The endpoint subnet reserves Azure's first four addresses and its final address.
Public, noncanonical, overlapping, undersized, foreign, reused, and inconsistent bindings reject before credential construction.
The sandbox-placement/v5 profile includes these bindings, network settings, and required topology, firewall, and resolver bindings.
Changes require a new derived worker.networkProfile. Older staged profiles reject.
No native command currently loads those profiles.

PrivateDNSDefinitions returns seven reviewable resource bodies for two zones, two links, and three A records.
It makes no service request or resource mutation.
It does not create endpoint, relay, registry, firewall, or protection resources.

### Endpoint and NIC checks

Each endpoint and NIC must match its resource identifier, name, type, region, and successful provisioning state.
The endpoint attaches to the selected private subnet and exactly its selected NIC.
One approved automatic private-link connection targets the expected service and subresource.
Pending approval remains unavailable. Rejected approval, manual connections, extra interfaces, and extended locations reject.

The registry endpoint targets the selected registry and registry subresource.
Its NIC exposes exactly the registry member and registry_data_REGION member for the configured region.
The exchange endpoint targets exchangeEnvironment and the managedEnvironment subresource.
Its NIC exposes exactly the configured exchange address.
NIC IP configurations must report the selected subnet, IPv4, expected address, and private-link member.
Public addresses, forwarding, virtual machines, extra workloads, load-balancer attachments, alternate DNS settings, and unrelated NIC features reject.

The endpoint subnet must report its prefix, security group, disabled default outbound access, and enabled private endpoint network policies.
It contains exactly the two selected endpoints and the three selected NIC IP configurations.
NAT, routes, delegation, service endpoints, shared allocations, and unrelated associations reject.

The endpoint security group attaches only to that subnet, has no NIC attachments, and enables flushConnection.
Its complete custom rule list permits inbound TCP port 443 only from the execution subnet to the three endpoint addresses.
Separate rules deny all remaining inbound and outbound traffic.
The shared network verifier checks priorities, address forms, ports, directions, child ownership, and successful provisioning.
Broader or additional rules reject.

### Private DNS checks

The exchange zone contains an apex A record for exchangeIp.
The registry zone contains the registry login A record and its regional data A record.
Each zone also contains one valid apex SOA record.
The complete record inventory must contain exactly these records.
Aliases, wildcard records, duplicate records, extra addresses, automatic registration, and unrelated records reject.
A-record TTL values must be between one and 300 seconds.
SOA timing and the reported zone record count must agree with the expected bounded contract.

Each zone has exactly one completed sandbox-network link to the sandbox connection's VNet.
Registration is disabled. Public fallback resolution rejects.
Reported zone link counts must match the complete link inventory.
The stage does not resolve these names or inspect every other zone linked to the VNet.
The required [topology stage](azure-sandbox-topology.md) now verifies the selected VNet contents and firewall policy without enabling native execution.
The required [resolver stage](azure-sandbox-resolver.md) now checks selected policy, actual association, and other reported subscription zone links.
Visibility across subscriptions, deny-hidden resources, write protection, and effective resolution remain unverified.

### Observations and pagination

The initial observation reads twelve selected paths:
two endpoints, two NICs, the endpoint subnet, its security group, two zones, and four DNS collections.
Network resources use API version 2025-03-01. Private DNS uses 2024-06-01.
Creation and import always obtain fresh observations.
Live callers share successful evidence for at most 15 seconds from observation start.
Current failed full observations invalidate earlier success. Failures are not cached.
Remaining protection callbacks, sandbox definitions, and domain authorization remain uncached.

Each observation has a five-second bound and joins all selected reads.
Each collection allows at most three pages and its exact expected item count.
Continuation URLs must retain the selected management.azure.com host, collection path, and API version.
Only one bounded nonempty $skipToken is permitted beside api-version.
Changed scopes, filters, repeated tokens, duplicate identifiers, missing value arrays, and unfinished pagination reject.
Unsafe continuations reject before credential acquisition.
Partial inventories are never returned as complete evidence.

This stage's reader permits GET only, bounds each response to 1 MiB, and closes response bodies.
It disables redirects, provider registration, and automatic retries.
Metadata gaps remain unavailable. A metadata 404 never establishes sandbox absence or permits another launch.
Errors omit provider bodies and credential diagnostics.
Runtime.Close closes the private reader after joining owned command work.

## Tutorials

### Verify locally

```sh
go test -race ./adapters/execution/azuresandbox -run '^TestSandboxPrivate|^TestSandboxRuntimePrivate' -count=1 -timeout=40s
```

Five focused checks cover endpoint and DNS drift, bounded pagination, invalid settings, shared observation evidence, and automatic runtime integration.
The affected network, placement, authority, runtime, shared execution, and command authorization checks also pass.
The combined targeted run passes 28 test functions with race detection in 2.559 seconds.
One initial existing runtime admission check also passes before the targeted regression run.
Evidence is output/azure-sandbox-private/summary.json and targeted.log.
These checks use local SQLite fixtures, fake credentials, and in-memory ARM responses.
No cloud service, browser, container, UI build, demo restart, resource mutation, or remote push is involved.

### Finish activation

Verify effective DNS resolution, visibility across subscriptions, write protection, and trusted exchange application and backend ownership.
Observe additional grants, deny effects, credential host attachment, and protection bindings.
Then wire the scoped execution and discovery commands and package their dependencies.
Actual preview compatibility, private source conversion, role acceptance, timer execution, token audience, and proxy trust remain separate deployment work.

Primary contracts: [Microsoft registry private endpoints](https://learn.microsoft.com/en-us/azure/container-registry/container-registry-private-endpoints)
and [Microsoft Container Apps private endpoints](https://learn.microsoft.com/en-us/azure/container-apps/how-to-use-private-endpoint).
Related guides: [connection rules and routes](azure-sandbox-network.md), [placement](azure-sandbox-placement.md), and [runtime](azure-sandbox-runtime.md).
Acceptance 51 advances. The verified count remains 48 of 57, with nine open.
