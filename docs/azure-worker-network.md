# Azure worker network admission

## Current boundary

The Azure adapter now includes a topology inspection stage with a production read-only SDK client.
It checks actual environment, VNet, and subnet responses against explicit installation settings.
Local fixtures exercise these SDK requests without Azure credentials or connections.
The [perimeter stage](azure-worker-perimeter.md) now inspects complete custom NSG rules and the worker route table after topology succeeds.

TopologyGuard.Check does not implement the execution Network interface.
A successful topology inspection does not authorize dispatch or release worker input.
The [firewall stage](azure-worker-firewall.md) now verifies firewall attachment, dedicated policy, and complete group inventory.
The [DNS stage](azure-worker-dns.md) now checks actual resolver association, exclusive links, exact lists, and enabled allow/block rules.
The [selected private-zone stage](azure-worker-private-dns.md) now checks exact records, complete links, registration, and fallback.
The [endpoint stage](azure-worker-endpoints.md) now checks consumer-side approval, NIC ownership, subnet policy, and regional address roles.
The [registry service stage](azure-worker-registry.md) now checks private configuration, regional data members, and service-side approval.
The [selected exchange stage](azure-worker-exchange.md) now checks app specification, service approval, routes, revisions, and HTTPS metadata.
The [workload inventory stage](azure-worker-workloads.md) now checks reported application and job occupancy across the configured subscription.
The [staged backend package](azure-worker-backend.md) now defines the separate internal backend environment, worker-only app, and scoped installation identities.
It creates no worker job or relay path and proves no deployed isolation.
The [backend observer](azure-worker-backend-admission.md) now checks selected service configuration, identity metadata, current revisions, and reported backend occupancy.
The [backend endpoint observer](azure-worker-backend-endpoint.md) now binds selected approval, NIC and IP ownership, relay source prefix, and dedicated endpoint subnet policy.
The [relay VNet observer](azure-relay-topology.md) now checks the selected address space, DNS settings, peering, and subnet consistency.
The [relay perimeter observer](azure-relay-perimeter.md) now binds the source NSG and default firewall route through shared policy checks.
The [relay firewall observer](azure-relay-firewall.md) now binds the actual firewall resource, dedicated policy, subnet, and public IP association.
The [relay DNS observer](azure-relay-dns.md) now checks the selected resolver policy, exact lists, enabled rules, exclusive link, and actual VNet association.
The [relay registry endpoint observer](azure-relay-registry-endpoint.md) now binds explicit service approval, NIC members, addresses, subnet policy, and dedicated security rules.
The [relay private DNS observer](azure-relay-private-dns.md) now checks derived registry records, a completed exclusive relay link, disabled registration, and no public fallback.
The [reported private DNS inventory](azure-private-dns-inventory.md) now detects unexpected subscription-visible links into worker and relay VNets.
It establishes neither cross-subscription or cross-tenant coverage nor authority to prevent later attachments.
The [selected deny observer](azure-network-deny.md) now checks explicit enforcement, all-principal coverage, child scopes, and fixed exceptions on both VNets and three selected zones.
Deployed enforcement, existing cross-scope links, clean provenance, read coverage, and authority to change protection remain separate requirements.
The [observer identity stage](azure-read-identity.md) now binds the reported client, principal, tenant, and selected subscription read grant.
Bound observer sessions now use that verified client credential throughout the rebuilt SDK inspection chain.
Effective access, host attachment, and native command wiring remain unverified.
The [reported read-deny scan](azure-read-deny.md) now checks subscription conflicts before the bound resource chain and requires all five selected protection assignments.
Local SDK fixtures verify this gate. Hosted visibility and actual enforcement remain unproven.
The [observer host stage](azure-observer-host.md) now checks selected controller placement, identity lifecycles, image, and healthy current revision.
Optional core packaging declares its separate API-only identity and fixed subscription read grant. Native startup and execution admission remain incomplete.
Host-bound workload inventory now rejects reported observer sharing on other apps and jobs without another collection request.
Other identity consumers, complete read visibility, and future attachment authority remain unproven.
The identity reader now rejects reported federated credentials and unknown collection state before credential construction or completed bound checks.
Future credential writes, other consumer types, actual visibility, and identity-management authority remain separate requirements.
The complete adapter must also prove read coverage, attachment authority, effective zone visibility, full backend authority, exchange networking, and execution placement.
Azure coded execution remains disabled in native commands and infrastructure packaging.
See [durable Azure execution](azure-worker-lifecycle.md), [job contracts](azure-worker-contract.md), and [independent deadlines](azure-worker-deadlines.md).

## Installation topology

Use separate worker and controller environments, VNets, and infrastructure subnets in the selected subscription and region.
The worker environment uses the default workload-profile environment type with one Consumption profile.
The legacy Consumption-only environment type cannot satisfy the intended user-defined routing profile.
Consumption here names the workload profile, not the legacy environment type.

TopologySettings requires these explicit resource identifiers and values:

| Setting | Requirement |
| --- | --- |
| workerVnet | Dedicated worker VNet resource identifier. |
| workerSubnet | Its Container Apps infrastructure subnet identifier. |
| workerVnetPrefix | One canonical private IPv4 address-space prefix. |
| workerSubnetPrefix | One canonical private IPv4 prefix inside that VNet. The subnet contains at least 32 addresses. |
| controllerVnet | Separate controller VNet resource identifier. |
| controllerSubnet | The controller environment's actual infrastructure subnet identifier. |
| securityGroup | Expected worker subnet NSG resource identifier. |
| routeTable | Expected worker subnet route-table resource identifier. |
| infrastructureGroup | Expected platform-managed worker resource group name. Keep it separate from controller infrastructure and the application resource group. |

Resource groups can differ within the selected subscription. Resource names use the worker configuration's canonical naming rules.
IPv4 prefixes must lie entirely within an RFC 1918 private range. A private starting address alone does not establish this condition.
IPv6, mapped IPv4, noncanonical prefixes, overlapping networks, foreign subscriptions, and substituted parent resources are rejected.

The full network adapter must bind these settings and all later perimeter settings into Config.networkProfile.
This stage compares the complete supplied Config with its constructor configuration. It does not derive or certify the final network profile digest.
Do not substitute a topology digest for the complete execution-network identity.

## Environment checks

Both environments must report the selected identity, name, type, region, successful provisioning, and exact infrastructure subnet.
The worker environment requires an internal load balancer and disabled public network access.
Its reported static IPv4 address must belong to the configured worker subnet.
Missing addresses and unfinished provisioning remain unavailable.
Private ingress endpoint connections, unexpected workload profiles, and shared infrastructure groups are rejected.

The initial worker profile has no environment identity, Dapr telemetry credentials, or external application-log destination.
AppLogsConfiguration explicitly selects none. Runner reports and attachments remain on the authenticated worker exchange.
This profile avoids adding shared telemetry destinations to the worker egress policy.
It does not provide Azure startup logs for a worker that cannot reach its supervisor. Preserve that evidence gap explicitly.
Controller logging and identity remain separate installation concerns.

## VNet and subnet checks

The worker VNet must report the exact configured private address space and successful provisioning.
It must have no peering, custom DNS servers, default public NAT gateway, BGP community configuration, or external IP allocation.
The controller VNet must report bounded, canonical private address spaces that do not overlap the worker VNet.
The controller subnet must belong to one of those address spaces.

The worker subnet must report the exact prefix, expected NSG and route-table bindings, and successful provisioning.
DefaultOutboundAccess must explicitly be false. This subnet setting alone does not establish Container Apps outbound isolation.
The subnet must have one successful Microsoft.App/environments delegation.
NAT gateways, service endpoints, endpoint policies, private endpoints, application gateways, extra IP allocations, and subnet sharing are rejected.
Private endpoints belong in a separate verified endpoint subnet, not the delegated worker subnet.
The VNet's embedded subnet and the direct subnet response must agree. Duplicate, absent, or conflicting entries reject inspection.

These checks verify attachment and topology only.
An attached NSG can still contain unsafe rules. An attached route table can still permit a bypass.
An empty custom-DNS list does not prove that Azure DNS blocks other names.
No claim of complete isolation follows from these bindings.

## SDK read boundary

NewSDKTopologyReader requires an explicit TokenCredential and validated worker and topology settings.
The reader permits GET on exactly six resources: two environments, two VNets, and two subnets.
Environment reads use armappcontainers/v4 4.0.0 and management API 2025-07-01.
Network reads use armnetwork/v8 8.0.0 and management API 2025-03-01.

The reader fixes the public Azure management origin and selected resource subscription.
The shared ARM configuration disables automatic retries and resource-provider registration.
Mutation verbs, list operations, child actions, foreign resources, redirects, extra query parameters, and alternate versions are rejected.
The guard has a five-second total inspection context. Individual reader methods also bound their operation contexts.
Each response has a four-megabyte read limit. Raw provider diagnostics become fixed control errors.

A complete inspection performs six reads. It retains no successful-admission cache.
Every later inspection observes fresh state. Concurrent calls share no mutable admission result.
Unavailable resources, denied reads, cancelled operations, missing resources, and unfinished provisioning leave admission unproven.
No read uncertainty substitutes for resource absence or successful isolation.

The controller needs read authority on the exact configured environments, VNets, and subnets.
The topology reader needs no resource mutation, secret, registry, or worker identity authority.
Close the reader to close idle transport connections.

## Remaining admission stages

Before enabling coded Azure jobs, implement and verify these controls:

1. Compose the existing topology and perimeter inspectors with the remaining admission stages. Neither stage alone authorizes execution.
2. Compose the existing firewall inspector. Preserve its fixed platform allowlist, terminal deny, resource ownership, and complete group inventory.
3. Compose the DNS policy and selected-zone inspectors. Resolve effective zone visibility and attachment controls. Preserve hierarchy and CNAME semantics.
4. Compose endpoint, registry, exchange, and workload inspection. Verify read coverage, attachment authority, backend authority, and the trusted preparation registry path.
5. Isolate the exchange from API, database, evidence, registry-write, and identity credentials.
6. Verify that unrelated workloads cannot share the worker trust boundary. Establish cross-job isolation and execution placement explicitly.
7. Compose the complete Network implementation and bind its whole policy into the execution profile.
8. Add native configuration, role bindings, infrastructure packaging, and separate deployment verification.

Container Apps environments share networking across workloads.
A subnet rule required by the platform does not prove isolation between test executions.
Do not enable cross-project execution until that boundary is established.
If ordinary jobs cannot meet the execution isolation contract, select a suitable Azure execution service rather than weaken the contract.

Azure DNS resolver policy can filter at the VNet level. Rules follow DNS hierarchy and inspect CNAME chains.
A threat-intelligence blocklist alone does not provide the required allowlist-and-default-block policy.
A DNS proxy alone does not prove query filtering. Check the actual policy and its association.
The worker VNet must remain dedicated because these DNS restrictions affect all resources in that VNet.

## Targeted local verification

Seven new focused race checks cover valid topology, concurrent fresh reads, environment drift, network bypasses, and uncertain state.
They also cover private-prefix validation, exact read scope, escaped resource groups, rejected mutations, redirects, and response limits.
Two selected existing SDK checks preserve the job mutation and independent stop request contracts after sharing ARM configuration.
Only the affected prefix and new boundary checks repeat after their focused updates.

To reproduce the topology checks locally:

```sh
go test -race ./adapters/execution/azure -run '^TestAzureTopology' -count=1
```

Evidence is output/azure-worker-topology/summary.json and the focused command results.
No cloud credential, Azure connection, registry push, deployment, UI build, browser, container, or image build runs.
Acceptance 51 advances. The verified total remains 48 of 57, with nine open items.

## Primary references

[Virtual-network configuration](https://learn.microsoft.com/en-us/azure/container-apps/custom-virtual-networks) defines subnet sizing, delegation, and environment networking.
[User-defined routes](https://learn.microsoft.com/en-us/azure/container-apps/user-defined-routes) defines the workload-profile environment requirement.
[Container Apps firewall rules](https://learn.microsoft.com/en-us/azure/container-apps/firewall-integration) defines platform dependencies and DNS considerations.
[DNS resolver policies](https://learn.microsoft.com/en-us/azure/dns/dns-security-policy) defines VNet association, rule priority, hierarchy, and CNAME behavior.
[Environment security guidance](https://learn.microsoft.com/en-us/azure/container-apps/secure-deployment) defines environment-level security boundaries.
