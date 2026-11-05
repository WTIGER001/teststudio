# Azure preparation registry path

## Current boundary

PreparationRegistryGuard inspects the selected trusted registry path after the existing [registry service guard](azure-worker-registry.md) succeeds.
It checks the preparation endpoint, NIC, endpoint subnet, source subnet, controller VNet, endpoint security group, and selected private DNS zone.
This is a read-only prerequisite for native preparation. It implements no execution Network and starts no preparation queue or registry publication.
The existing host-bound observer remains unchanged at 22 readers and its current request limits.
Acceptance 51 advances; the verified total remains 48 of 57, with nine open.

The source queue requires both image assembly and isolated discovery before source review and approval.
Registry publication alone cannot complete preparation. Native discovery and complete execution admission remain required.
The [writer publisher](azure-preparation-writer.md) now composes this path with a selected identity and registry-scoped push grant. Native placement remains unfinished.
See [registry source assembly](registry-preparation.md) and [trusted ACR authentication](azure-registry-authentication.md).

## Explicit settings

PreparationRegistrySettings requires these fields:

| Field | Requirement |
| --- | --- |
| registry | Complete preceding RegistrySettings, including its exact preparationEndpoint. |
| controllerPrefixes | Exact controller VNet address spaces, with one through eight canonical private IPv4 prefixes. |
| interface | The preparation endpoint's distinct network interface identifier. |
| subnet and subnetPrefix | Dedicated private endpoint subnet under the selected controller VNet and its canonical private IPv4 prefix. |
| securityGroup | Distinct endpoint security group identifier. |
| sourceSubnet and sourceSubnetPrefix | Separate trusted preparation source subnet under the controller VNet and its private IPv4 prefix. |
| registryIp and dataIp | Distinct usable private addresses in the endpoint subnet for the registry and selected regional data member. |
| registryZone | Separate privatelink.azurecr.io zone identifier, distinct from the worker registry zone. |

The endpoint identifier comes from RegistrySettings.preparationEndpoint. There is no independent override.
The controller VNet comes from the preceding topology. Neither selected subnet can reuse the core controller infrastructure subnet.
The endpoint and source prefixes cannot overlap each other or the worker VNet.
Both must fit within controllerPrefixes. The configured spaces cannot overlap each other or the worker VNet.
Canonical settings copy and sort the prefix list, so later caller edits cannot alter the bound policy. Endpoint prefixes support /29 or larger networks; source prefixes support /27 or larger.
The interface cannot reuse the worker registry or exchange interface. The group cannot reuse either worker security group.
Invalid settings or a parent configuration mismatch reject before any SDK observation.

## Network and endpoint checks

The registry stage refreshes its complete preceding chain once, including service-side approval for the selected preparation endpoint.
Only after that succeeds do six selected network GETs run concurrently.
They read endpoint, NIC, endpoint subnet, source subnet, endpoint security group, and controller VNet.

The shared endpoint verifier requires the selected registry service, registry group, one NIC, approved connection, successful state, and exact member addresses.
The NIC must backlink to the selected endpoint and expose only the global registry and configured regional data member.
Public IPs, unrelated attachments, alternate members, and substituted addresses reject the fixed shared endpoint contract.

The endpoint subnet contains only that endpoint and its selected NIC configurations.
Private endpoint security policies remain enabled. Default outbound access is explicitly false.
The shared subnet contract rejects NAT, route tables, delegations, service endpoints, and other unsupported bypasses.
The security group flushes connections and belongs only to that subnet.
Its fixed rules permit TCP 443 from exactly the configured controller VNet prefixes to the two member IPs, followed by inbound and outbound deny rules.
The same trusted path supports controller and backend platform image pulls as well as preparation.
A preparation-subnet-only rule would block those legitimate pulls. Registry writer authority remains a separate managed identity grant.
The rule uses explicit private CIDRs, with no VirtualNetwork service tag or worker VNet exception.

The refreshed controller VNet must report exactly controllerPrefixes, allowing order changes.
Those spaces remain private, nonduplicate, and disjoint from the worker VNet.
Both selected subnet prefixes must fit within those address spaces and remain separate from the reported core infrastructure prefix.
The source subnet reports its exact prefix and no private endpoint.
The VNet's embedded endpoint and source subnet values must agree with the direct reads.
The verifier uses the canonical subnet-prefix helper, accepting either supported single-prefix representation.
The selected VNet permits at most eight address spaces and 256 embedded subnets.

## Selected private DNS checks

DNS observation starts only after network checks succeed.
The existing private DNS reader inspects the selected zone, its complete record collection, and its complete link collection concurrently.
The zone contains the fixed SOA and exact A records for the registry and regional data authority, with the shared TTL and count limits.
Those records must name the preparation endpoint IPs.
The single completed preparation-network link selects the controller VNet, disables automatic registration, and permits no public fallback.
Worker links, substituted addresses, aliases, extra records, incomplete counts, and extra links remain blocking under the shared contract.

ZoneDefinition, LinkDefinition, RecordDefinitions, SubnetDefinition, and SecurityGroupDefinition return SDK definitions for installation review.
They perform no provider operation and create no network resource.
Microsoft's [registry private endpoint guide](https://learn.microsoft.com/en-us/azure/container-registry/container-registry-private-endpoints) describes separate global and regional data addresses and their DNS records.

## SDK bounds and ownership

The network reader permits GET only on its six selected resources through the existing network SDK and API 2025-03-01.
The DNS reader permits only its selected zone and bounded record and link collections through the existing DNS SDK and API 2024-06-01.
Both require an explicit trusted credential. Neither discovers a credential or uses a fallback.
Mutations, credential operations, alternate resources, unexpected queries, and redirects reject the shared restricted transport.
Each SDK operation retains its five-second context and four-megabyte response limit. Retries and provider registration remain disabled.
The caller owns both readers and closes them after their final inspection.

Check has a 40-second total context, including the existing registry stage's 30-second bound.
The added path normally costs nine requests: six network GETs and three DNS operations.
Its additional paging ceiling is twelve requests: six network GETs plus one zone, at most three record pages, and at most two link pages.
The preceding parent limits remain unchanged. Successful observations are never cached or persisted.
A changed path blocks the next check. Restored valid state can pass later.
A failed parent prevents all preparation reads; failed endpoint or network checks prevent DNS reads.
Errors expose fixed public categories and omit provider diagnostics.

## Remaining installation work

This observer proves only consistency of selected reported resources. It does not prove effective DNS visibility or actual packet routing.
Source-subnet prefix observation does not inspect its complete egress policy, environment delegation, or preparation-process placement.
Native setup must bind the actual preparation workload, its private registry route, and its separate writer identity and effective grants.
The complete profile must also preserve independent relay DNS and existing worker isolation.
Protection-management authority, clean provenance, cross-scope coverage, and prevention of later attachments remain separate installation requirements.

The core package declares no preparation workload or these endpoint and zone resources.
The separate [native assembly profile](azure-native-assembly.md) now supplies a trusted worker template and command configuration.
Native commands still expose no Azure coded-worker option. Complete source preparation requires isolated discovery admission.
Do not substitute a fake Network, successful registry write, or configured private hostname for that admission.
Four targeted local race checks pass after the final trusted-controller source-policy correction.
They cover settings ownership, caller mutation, live selected metadata, CIDR containment, subnet representations, fixed policy, DNS, SDK restrictions, and uncertainty.
Earlier review repeats only changed branches; the final semantic policy change receives the four-check selection. No existing or broad suite repeats.
Local checks use fake credentials and in-memory SDK transports, with no provider request, listener, registry publication, or deployment.

## Assembly queue foundation

The source service now supports [durable assembly-only handoff](source-assembly-handoff.md), with discovery deferred to a compatible worker.
This separates confirmed source publication from execution-control authority without skipping discovery or approval.
The native mode supports existing ECS and Kubernetes registry settings and separate Azure assembly settings.
Azure command configuration and publisher composition now exist. Actual placement and isolated discovery remain unverified or unfinished.
