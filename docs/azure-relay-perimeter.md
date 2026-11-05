# Azure relay security rules and routing

## Scope

RelayPerimeterGuard checks the selected relay source NSG and route table after [relay VNet observation](azure-relay-topology.md).
It reuses the preceding call's validated source subnet. It adds two concurrent resource reads without repeating subnet or VNet reads.
The worker and relay use the same platform rule generator, bounded SDK reader, security rule comparison, and route comparison.

This stage does not implement Network, enable coded jobs, or authorize dispatch.
It inspects reported configuration and sends no packet to Azure or the backend.
The [relay firewall observer](azure-relay-firewall.md) now binds actual firewall ownership, policy, subnet, and public IP association.
Effective DNS, registry endpoint ownership, effective authority, and complete execution isolation remain required.
Acceptance 51 advances. The verified total remains 48 of 57, with nine open.

## Explicit settings

RelayPerimeterSettings requires these values:

| Field | Requirement |
| --- | --- |
| topology | Exact RelayTopologySettings used by the preceding guard. |
| securityGroup | Dedicated relay source NSG identifier in the installation subscription. |
| routeTable | Dedicated relay route-table identifier in the installation subscription. |
| firewallIp | Canonical private IPv4 address inside the relay VNet and outside both selected subnets. |
| registryEndpointIps | Two through eight unique private IPv4 addresses for the intended registry and regional data endpoints. |

Registry addresses must belong to the relay VNet and lie outside both the relay source and backend endpoint subnet prefixes.
They must differ from the firewall address. Public addresses, mapped IPv6, duplicate addresses, and foreign network addresses reject configuration.
These intended addresses do not prove assignment or endpoint ownership. A later endpoint stage must bind them to actual NICs and service approval.
The firewall address does not prove its resource, subnet, or policy ownership.

The source NSG cannot reuse the worker source, worker endpoint, or backend endpoint NSG.
The route table cannot reuse the worker route table.
The constructor copies and sorts registry addresses. Caller mutation cannot change the saved policy.
SecurityGroupDefinition and RouteTableDefinition return fresh SDK values without issuing resource writes.

## Source subnet and NSG

The current source subnet must attach the selected NSG and route table and explicitly disable default outbound access.
NAT gateways, service endpoints, endpoint policies, private endpoints, application gateway bindings, extra IP allocations, and subnet sharing reject observation.
The preceding endpoint and VNet stages still require the exact source prefix, successful delegation, and matching embedded subnet observation.
Platform subnet attachments and within-subnet communication do not establish workload isolation or prevent later attachments.

The NSG must attach only to the selected source subnet, attach to no individual NIC, and enable FlushConnection.
It must report its selected identity, region, successful provisioning, and exactly thirteen custom rules.
Names use relay- as their prefix. Directions, priorities, protocols, sources, ports, and destinations must match the generated policy.
The [worker perimeter guide](azure-worker-perimeter.md#fixed-custom-nsg-policy) describes the shared platform rules and priorities.

The relay private-endpoint rule allows outbound TCP 443 to only the pinned backend address and selected registry addresses.
The policy denies metadata access, permits platform DNS only on TCP and UDP 53, and ends with explicit inbound and outbound denies.
Platform registry, first-party CDN, and identity service exceptions retain their fixed TCP 443 rules.
It adds no general Internet, AzureCloud, Storage, monitoring, worker VNet, or controller VNet exception.
The within-source-subnet and load-balancer rules retain the shared platform requirements.
Service tags alone do not restrict application requests. Firewall and effective DNS checks remain mandatory.

## Route table

The route table must attach only to the selected source subnet and explicitly disable BGP route propagation.
It contains exactly one route named relay-egress.
That route sends 0.0.0.0/0 to the selected firewall address with next-hop type VirtualAppliance.
Both table and route must report successful provisioning and their expected resource identities.
Extra routes, Internet next hops, substituted appliances, shared attachments, and BGP override observations reject inspection.

Private VNet system routes still exist. The default route does not establish firewall ownership or effective packet isolation.
The complete installation must verify platform startup, image pulls, metadata restrictions, probes, and private transport in the separate hosted phase.
Do not relax these controls silently to make startup pass.

## Read boundary and verification

NewSDKRelayPerimeterReader creates the shared SDKPerimeterReader with only the selected NSG and route table in its allowlist.
It requires an explicit credential and uses armnetwork/v8 with management API 2025-03-01.
Typed methods reject resource-kind substitution before transport. The transport rejects mutations, child actions, collections, expansions, foreign hosts, and alternate queries.
Each read has a five-second context and a four-megabyte response limit. Automatic retries and provider registration remain disabled.
The composed guard has a 60-second context, 60 normal reads, and at most 113 reads with bounded parent paging.
Unfinished or unavailable observations cannot establish approval. Successful checks are not cached or persisted.

Four new focused race checks and five selected regression checks pass locally in 3.718 seconds.
They cover parent reuse, immutable settings, source bindings, policy drift, route bypasses, unavailable resources, provisioning state, and exact SDK scope.
Existing checks cover the shared traffic matrix, worker route comparison, SDK construction, immutable definitions, and relay topology observation.
Fixtures use explicit fake credentials and in-memory responses. They open no network connection or listener.
An initial test fixture used the wrong SDK type for its NAT reference. The corrected targeted command passes.
No broad suite, UI build, image build, browser, container, cloud request, registry push, deployment, or demo restart runs.

```sh
go test -race ./adapters/execution/azure -run '^(TestAzureRelayPerimeter.*|TestAzureRelayTopologyReusesFreshSubnetObservations|TestAzurePerimeterPolicyTrafficMatrixUsesExplicitDecisions|TestAzurePerimeterRejectsRoutingBypasses|TestAzurePerimeterSDKPinsTwoResourcesAndSupportsEscapedGroups|TestAzurePerimeterSettingsAndDefinitionsFreezePrivateAddresses)$' -count=1
```

Evidence is output/azure-relay-perimeter/summary.json and the targeted command result.
Continue effective DNS, registry endpoint ownership, identity and attachment authority, and complete native wiring.
