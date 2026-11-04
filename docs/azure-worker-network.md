# Azure worker network admission

## Current boundary

The Azure adapter now includes a topology inspection stage with a production read-only SDK client.
It checks actual environment, VNet, and subnet responses against explicit installation settings.
Local fixtures exercise these SDK requests without Azure credentials or connections.

TopologyGuard.Check does not implement the execution Network interface.
A successful topology inspection does not authorize dispatch or release worker input.
The complete network adapter must also verify routing, security rules, DNS policy, private endpoints, exchange isolation, and execution placement.
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

1. Inspect the complete NSG rules and route table. Reject public, private-network, service-endpoint, and propagated-route bypasses.
2. Bind permitted platform traffic to the documented Container Apps requirements without allowing arbitrary application destinations.
3. Verify the Azure DNS resolver policy, VNet link, enabled traffic rules, domain lists, default block, and private DNS records.
4. Verify exact private registry and exchange endpoints, approval, resource ownership, routing, and address bindings.
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
