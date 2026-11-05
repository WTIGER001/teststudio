# Azure sandbox relay network and registry binding

## Introduction

OpenRuntime includes SDKRelayNetwork in creation, source import, and live admission.
It binds the trusted relay's VNet, security rules, default route, firewall, resolver, private registry endpoint, DNS, and selected pull grant.
The existing backend endpoint stage verifies the backend path in the same VNet.
The subscription DNS inventory now guards both the sandbox and relay VNets.

This stage verifies reported metadata. It does not establish deployed connectivity or effective permissions.
Native commands and deployment templates do not enable sandbox execution yet.
Remaining protection dependencies stay mandatory.

## Concepts

### Required settings

placement.services.relayNetwork requires these bindings:

| Field | Contract |
| --- | --- |
| topology.prefix | Canonical private IPv4 VNet prefix, separate from the execution VNet. |
| topology.firewallPrefix | Dedicated AzureFirewallSubnet prefix, with a size of /26 or larger. |
| topology.firewall, policy, and publicIp | Selected firewall, policy, and public IP resources in the configured subscription. |
| firewallIp | Usable private address in AzureFirewallSubnet. |
| securityGroup and routeTable | Selected exclusive attachments for the relay infrastructure subnet. |
| resolver | Selected policy and distinct approved-domain and root-block lists. |
| registry.endpoint and interface | Dedicated registry private endpoint and NIC. |
| registry.subnet and subnetPrefix | Dedicated endpoint subnet in the relay VNet, with a size of /29 or larger. |
| registry.securityGroup | Selected endpoint security group, separate from relay, backend, and execution groups. |
| registry.loginIp and dataIp | Distinct usable private addresses in the registry endpoint subnet. |
| registry.zone | Separate privatelink.azurecr.io zone resource linked only to the relay VNet. |
| registry.connection | Selected registry-side private endpoint connection. |
| registry.pullAssignment | Registry-scoped AcrPull assignment for the existing relay pull identity. |

The relay infrastructure, backend endpoint, registry endpoint, and firewall subnets must fit inside the configured VNet without overlap.
They cannot reuse the execution network or each other's protected resources.
Resolver resources and the registry zone differ from their execution-network equivalents.
The selected pull assignment differs from the source import assignment.
Invalid scopes, aliases, reserved addresses, overlaps, and stale profiles reject before credential construction.

The sandbox-placement/v8 profile binds these settings with all existing placement and service settings.
Earlier staged profiles require regeneration. No native command currently loads them.
RelayNetworkDefinitions returns 21 reviewable resource bodies without mutating resources.
They include rules, routes, subnets, firewall resources, resolver resources, private records, the zone link, and the pull assignment.
They do not provision the VNet, endpoint, NIC, approval, trusted applications, or identities.

### Trusted relay network policy

The VNet reports exactly four subnets. Embedded subnet contents must match their selected resource observations.
Peerings, custom DNS servers, extra subnets, and unrelated network attachments reject.
The relay subnet retains its configured prefix, successful environment delegation, security group, and route table.
Default outbound access, NAT, and unrelated service or private endpoints reject.

The relay security group permits the fixed Container Apps platform rules and HTTPS to three private addresses.
Those addresses are the backend endpoint, registry login endpoint, and regional registry data endpoint.
Other outbound traffic and metadata access are denied by the fixed rules.
The route table disables BGP propagation and has one default route through the selected private firewall address.
The firewall, public IP attachment, subnet configuration, and policy references must agree.
The complete firewall group inventory permits the fixed relay-platform group and empty known service defaults only.
The relay group permits approved platform HTTPS domains and denies other application traffic.

This trusted service policy stays separate from the sandbox's deny-all firewall policy and narrower connection rules.
No relay platform permission enters the sandbox definitions.
Platform compatibility and effective traffic enforcement remain separate deployment requirements.

### Private registry and DNS

The endpoint targets the selected registry and exactly its selected NIC and subnet.
The NIC reports exactly the configured login and regional data addresses, with matching registry member names.
The endpoint subnet contains exactly this endpoint and its two NIC IP configuration references.
Its security group permits only relay-subnet HTTPS to those two addresses, then denies remaining traffic.
Public addresses, forwarding, extra interfaces, unrelated attachments, and broader rules reject.

The registry must match the existing Premium, region, private access, data endpoint, and authentication settings.
The selected registry-side connection must approve the relay endpoint.
The registry's embedded connection with that identifier must also approve the same endpoint.
Missing or pending approval remains unavailable. Rejected or mismatched approval rejects admission.
This stage checks the selected connection. Complete registry consumer, replication, and private-link member inventory remains required.

The private zone has exact login and regional data A records, one SOA record, and one completed relay-network link.
Aliases, wildcard records, registration, extra records, extra links, and public fallback reject.
The resolver permits fixed platform domains and the four registry domain forms, followed by a root-domain block.
The backend hostname is absent because the relay pins the backend address and retains its hostname for TLS verification.
Complete rules, lists, links, and the actual VNet policy association must match.
Only an empty POST to the selected VNet's listDnsResolverPolicies read action is permitted.
Other selected metadata operations use GET.

The shared subscription inventory requires both sandbox-zone links and the relay registry-zone link.
Additional reported links to either guarded VNet reject. Unrelated VNets remain permitted.
Cross-subscription and deny-hidden links remain outside this reported inventory guarantee.

### Identity, cache, and shutdown

The relay pull identity must match its selected resource, tenant, client, and principal.
Its selected registry-scoped assignment must grant the fixed built-in AcrPull role without conditions or delegated authority.
Complete effective grants and identity host attachment remain separate requirements.

Creation and source import always read fresh evidence.
Concurrent live observations share successful evidence for at most 15 seconds from observation start.
Failed current observations invalidate prior success. Failures are not cached.
The stage has a five-second bound and joins all reads before returning.
Selected responses retain existing size, pagination, credential, redirect, and retry guards.
Metadata gaps remain unavailable and never establish sandbox absence.
Remaining protection, authorization, and sandbox-definition checks stay uncached.
Runtime.Close closes the relay reader after joining owned command work.

## Tutorials

### Verify the changed path locally

```sh
go test -race ./adapters/execution/azuresandbox -run '^TestSandboxRelayNetwork|^TestSandboxRuntimeRelayNetwork' -count=1 -timeout=30s
```

Six new checks cover trusted policy separation, required settings, the joined path, drift rejection, bounded shared evidence, and runtime admission.
Twelve affected native checks and two shared verifier checks also pass with race detection.
Evidence is output/azure-sandbox-relay-network/summary.json, native.log, and shared.log.
Checks use local SQLite, fake credentials, and in-memory ARM responses.
No cloud service, browser, container, UI build, demo restart, resource mutation, or remote push is involved.

### Continue native activation

Complete registry inventory, workload placement, identity host attachment, effective grants, deny effects, and protected write boundaries.
Then wire separate execution and discovery commands and package their dependencies.
Keep preview compatibility, private conversion, actual resolution, routing, platform readiness, token audience, and independent timer execution in deployment acceptance.

Primary references: [Container Apps firewall integration](https://learn.microsoft.com/en-us/azure/container-apps/firewall-integration)
and [private registry endpoints](https://learn.microsoft.com/en-us/azure/container-registry/container-registry-private-endpoints).
Related guides: [runtime](azure-sandbox-runtime.md), [backend endpoint](azure-sandbox-backend-endpoint.md), and [resolver](azure-sandbox-resolver.md).
Acceptance 51 advances. The total remains 48 verified of 57, with nine open.
