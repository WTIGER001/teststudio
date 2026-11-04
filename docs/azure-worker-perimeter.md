# Azure worker security rules and routing

## Current boundary

The Azure adapter now inspects the actual worker NSG and route table after successful [topology inspection](azure-worker-network.md).
PerimeterGuard.Check performs fresh topology checks before it reads either perimeter resource.
It checks complete custom rule content and routing against the fixed worker policy.
The same settings can produce SDK definitions for later infrastructure packaging.

This stage does not implement the execution Network interface or release worker input.
The [firewall stage](azure-worker-firewall.md) now verifies actual resource identity, attachment, policy, and complete group inventory.
The [DNS stage](azure-worker-dns.md) now checks the actual resolver policy and VNet association.
Private records, private endpoint ownership, isolated exchange service, and cross-job isolation remain required.
Azure coded execution stays disabled in native commands and infrastructure packaging.
Local fixtures do not establish live platform compatibility or effective packet isolation.

## Configuration and immutable definitions

PerimeterSettings contains the exact TopologySettings, one firewall private IPv4 address, and two through eight private endpoint IPv4 addresses.
All addresses must be canonical, unique, private, inside the worker VNet, and outside the delegated worker subnet.
The firewall address cannot double as an endpoint address.
Addresses outside the worker VNet, public addresses, IPv6, duplicates, and missing endpoint capacity reject configuration.
The constructor copies and sorts the endpoint list. Later caller mutations cannot change the saved policy.

Private endpoint addresses represent the intended exchange, registry, and registry data endpoints.
These settings alone do not verify their ownership or assignment. A later endpoint inspector must bind each address to its actual resource.
The firewall address alone does not establish a trusted firewall. Its resource, IP configuration, policy, and rule groups need separate inspection.

SecurityGroupDefinition and RouteTableDefinition return fresh SDK values.
They validate and normalize the same settings used by inspection. Mutating a returned value does not change another definition or the guard.
The definitions make no Azure write request. Infrastructure packaging must consume this policy without introducing a parallel permissive policy.
All topology, rule, route, firewall, DNS, and endpoint settings must enter the final networkProfile identity before execution enablement.

## Fixed custom NSG policy

The policy contains thirteen named custom rules. Lower priority numbers run first.

| Direction | Priority | Decision |
| --- | --- | --- |
| Inbound | 100 | Allow traffic within the delegated worker subnet for platform communication. |
| Inbound | 110 | Allow AzureLoadBalancer TCP probes on ports 30000–32767 into the worker subnet. |
| Inbound | 4096 | Deny all other inbound traffic. |
| Outbound | 100 | Deny AzurePlatformIMDS traffic. |
| Outbound | 110 | Allow traffic within the delegated worker subnet for platform communication. |
| Outbound | 120 | Allow worker UDP port 53 to AzurePlatformDNS. |
| Outbound | 121 | Allow worker TCP port 53 to AzurePlatformDNS. |
| Outbound | 130 | Deny other AzurePlatformDNS traffic. |
| Outbound | 200 | Allow worker TCP port 443 to MicrosoftContainerRegistry. |
| Outbound | 210 | Allow worker TCP port 443 to AzureFrontDoor.FirstParty. |
| Outbound | 220 | Allow worker TCP port 443 to AzureActiveDirectory. |
| Outbound | 300 | Allow worker TCP port 443 to the exact selected private endpoint addresses. |
| Outbound | 4096 | Deny all other outbound traffic. |

The explicit final denies precede Azure's default rules.
The policy adds no generic Internet, AzureCloud, Storage, AzureMonitor, alternate DNS server, or controller-VNet allow rule.
Public application targets must use the later trusted gateway contract. They do not become direct worker exceptions.
The initial worker topology disables external application telemetry. Telemetry destinations do not enter this policy.

The NSG must enable FlushConnection so Azure can reevaluate existing flows after rule changes.
The NSG must belong to the selected resource, region, and successful provisioning state.
It must attach only to the worker subnet and no individual network interfaces.

Inspection requires exactly the expected custom names, directions, priorities, protocols, actions, ports, sources, and destinations.
Extra, missing, duplicate, changed, foreign, or unfinished rules prevent success.
Rule order and endpoint address order do not change semantics. Duplicate endpoint entries remain invalid.
Descriptions and documented read-only metadata do not change traffic authority.
DefaultSecurityRules are provider metadata. The explicit custom deny rules override their lower-priority permissions.

## Routing policy

The worker route table contains one named route, worker-egress.
It routes 0.0.0.0/0 to the exact configured firewall address with next-hop type VirtualAppliance.
BGP route propagation must explicitly be disabled.
The route table attaches only to the delegated worker subnet.
Both table and route must report successful provisioning and the expected resource identity.

Extra specific routes, Internet next hops, substituted appliances, changed prefixes, unknown BGP configuration, and BGP override observations reject inspection.
Private VNet system routes still exist. The endpoint allow rule and final NSG deny remain necessary for private destinations.
An inspected default route does not prove the firewall's policy or the endpoint's identity.
The previous topology stage also rejects peerings, service endpoints, NAT attachments, and overlapping controller address spaces.

## Platform exceptions and remaining boundaries

Container Apps requires platform connectivity. These configured exceptions remain subject to separate deployment verification.
Verify registry pulls, managed identity behavior, DNS, probes, and platform startup before enabling the profile.
In particular, verify the interaction between the metadata deny rule and platform-only image-pull identity.
Do not remove a boundary silently to make platform startup pass. Resolve the isolation arrangement explicitly.

Service tags represent shared Azure service ranges. They do not provide the required application-level destination or request restriction by themselves.
The firewall inspector now checks the fixed platform allowlist, final deny, complete group inventory, and absence of classic or inherited-policy bypasses.
It does not establish live platform compatibility or effective packet isolation.
Azure platform DNS requires its specific service tag for NSG overrides. A generic deny alone does not establish DNS filtering.
The resolver-policy inspector now verifies the actual VNet association, exclusive link, enabled rule order, exact domain lists, and default block.
Private DNS and endpoint verification remain separate required stages.

The required within-subnet platform rules allow communication inside that subnet.
They do not establish cross-job isolation. Workload inventory and execution isolation need separate proof before cross-project coded execution.
No stage currently claims that ordinary jobs satisfy the complete execution isolation contract.
See [remaining network admission stages](azure-worker-network.md#remaining-admission-stages).

## SDK and read uncertainty

SDKPerimeterReader permits GET on exactly the selected NSG and route table.
It uses armnetwork/v8 8.0.0 and management API 2025-03-01 with an explicit Azure TokenCredential.
It shares the bounded read transport with the topology reader. Each client retains its own resource allowlist.
Mutation verbs, list operations, child actions, substituted resource kinds, foreign resources, redirects, and alternate queries are rejected.
Automatic retries and resource-provider registration remain disabled.

A complete perimeter inspection makes eight reads: six topology reads, one NSG read, and one route-table read.
The guard has a five-second total context and no successful-inspection cache.
Reader methods also bound their operation contexts. Each response has a four-megabyte read limit.
Provider diagnostics map to fixed control errors.
Missing, denied, unavailable, oversized, redirected, cancelled, and unfinished observations leave admission unproven.
The controller needs read authority on these exact resources. This reader needs no mutation, secret, or registry authority.

## Targeted local verification

Eight new focused race checks cover exact inspection, immutable settings, concurrent refresh, traffic policy, rule drift, route bypasses, and uncertain reads.
They also cover typed SDK scope, escaped resource groups, rejected mutations, and resource-kind substitution.
An independent fixture evaluates 24 declared traffic cases against the generated custom rules.
It requires explicit decisions for private endpoints, platform dependencies, metadata, public destinations, controller access, DNS, and probe ports.
These synthetic service-tag labels verify declaration intent. They do not model actual Azure service-tag address membership or live packet routing.
Two selected existing topology SDK boundary checks pass after sharing the read transport.
Only the changed address normalization and security-rule checks repeat alongside the new traffic matrix.

To reproduce the perimeter checks locally:

```sh
go test -race ./adapters/execution/azure -run '^TestAzurePerimeter' -count=1
```

Evidence is output/azure-worker-perimeter/summary.json and the focused command results.
No Azure credential, cloud API, packet probe, registry push, deployment, UI build, browser, container, or image build runs.
Acceptance 51 advances. The verified total remains 48 of 57, with nine open items.

## Primary references

[NSG semantics](https://learn.microsoft.com/en-us/azure/virtual-network/network-security-groups-overview) defines priorities, stateful flow behavior, and default rules.
[Container Apps NSG requirements](https://learn.microsoft.com/en-us/azure/container-apps/firewall-integration) defines platform dependencies and environment differences.
[Azure routing](https://learn.microsoft.com/en-us/azure/virtual-network/virtual-networks-udr-overview) defines custom routes, system routes, and BGP interaction.
[Azure service tags](https://learn.microsoft.com/en-us/azure/virtual-network/service-tags-overview) describes shared service ranges and platform service tags.
The pinned SDK SecurityGroupPropertiesFormat model defines FlushConnection behavior.
