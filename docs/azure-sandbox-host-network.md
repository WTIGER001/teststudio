# Azure sandbox controller source perimeter

## Introduction

OpenRuntime now includes SDKHostNetwork before controller registry, host, and workload admission.
It checks the selected VNet, source subnet policy, default route, dedicated firewall, and resolver association.
The controller keeps its trusted service access separate from the sandbox's narrower network policy.
Native command selection and complete deployment packaging remain unfinished.

## Concepts

### Required settings

host.network contains these settings:

| Field | Required binding |
| --- | --- |
| topology.prefix | Private controller VNet address space, separate from sandbox and relay networks. |
| topology.firewallPrefix | Separate private AzureFirewallSubnet prefix with usable capacity. |
| topology.firewall, policy, publicIp | Selected dedicated firewall, policy, and public IP resources. |
| firewallIp | Usable private firewall address. |
| securityGroup, routeTable | Exclusive controller source-subnet rules and route table. |
| resolver.policy, allowList, blockList | Selected resolver policy and distinct domain lists. |

The source, registry endpoint, and firewall prefixes must fit inside the VNet without overlap.
Missing settings, reserved addresses, foreign resources, reused protected resources, and stale host profiles reject configuration.
Host settings remain outside the shared sandbox-placement/v9 worker profile.
Calculate host.profile through Settings.HostProfile() after assigning the network settings.

### VNet and routing

The VNet contains exactly the selected source, registry endpoint, and firewall subnets.
Embedded subnet observations must agree with their selected resource reads.
Peerings, custom DNS servers, public NAT, additional subnets, and changed address spaces reject admission.
The source subnet retains the Container Apps environment delegation and exact security-group and route-table attachments.
It has no NAT gateway, shared scope, service endpoint, or implicit outbound access.

The route table disables BGP propagation.
Its only route sends 0.0.0.0/0 to the selected private firewall address.
The firewall must retain its exact policy, public IP, subnet, and single egress IP configuration.
The firewall subnet permits only the expected firewall attachment.

### Trusted HTTPS access

The source security group retains the existing platform rules and two explicit private registry addresses.
Public HTTPS uses the dedicated firewall path. Remaining outbound traffic is denied by the fixed source rules.
The complete firewall policy permits the derived platform and dependency domains, then denies other HTTP and HTTPS application traffic.
It permits no arbitrary extra network, application, or inherited policy rules.
Only the owned collection and empty known platform default collections are accepted.

Dependency domains derive from these existing settings:

- The fixed Azure management host and selected regional sandbox management endpoint.
- The database and runtime-configuration Key Vault secret hosts.
- The selected Blob service host in execution mode only.
- Existing trusted platform registry and identity domains.

The UI origin supplies completion links and adds no outgoing access.
Discovery adds no Blob dependency. The sandbox worker receives none of this trusted-host platform policy.
Some platform rules contain their existing wildcard domains. This is a trusted controller policy.

### Resolver association

The resolver approved list contains the derived dependency domains and private registry login and regional data names.
Its second list contains the root-domain block.
Complete rules and links must match the fixed controller definitions.
The selected VNet's listDnsResolverPolicies observation must report exactly the configured policy.
The transport permits only an empty POST for this read action. Bodies and other actions reject before transmission.
DNS domain hierarchy includes descendants. The private registry stage independently requires exact records and no public fallback.

### Bounds and staged bodies

The stage observes 16 selected paths with a joined five-second bound.
Network resources use API 2025-03-01. Resolver resources use API 2025-05-01.
Firewall collection inventory permits four items and four pages; resolver inventories retain their existing exact bounds.
Construction acquires no token and calls no service.
Creation and import use fresh reads. Successful live evidence expires 15 seconds after observation start.
A current failure invalidates earlier success. Remaining protection and authorization remain uncached.
Runtime.Close closes the reader after owned work joins.

HostNetworkDefinitions returns 14 staged bodies for source rules, routing, firewall configuration, and resolver policy.
They do not create a complete deployment or establish service reachability.
Database transport, dependency permissions, and native HTTPS-probe target routing remain unfinished.
The staged source rules do not yet permit a PostgreSQL connection.
Do not enable the native controller before those remaining access contracts are complete.

## Tutorials

### Local verification

Three new checks and eleven affected checks pass with race detection in 4.217 seconds.
They cover both modes, shared worker profiles, dependency separation, invalid settings, topology, routing, firewall and resolver drift, and shutdown.
Fresh routing failure blocks creation, image import, and live admission after an earlier successful observation.
Affected checks retain explicit credentials, scoped runtime modes, controller and registry admission, relay networking, and resolver composition.
The central discovery composition observes 142 unique selected paths.
Evidence is output/azure-sandbox-host-network/summary.json and local.log.
Checks use local SQLite, fake credentials, and in-memory metadata. Owned readers close through cleanup.
No cloud service, browser, container, UI build, demo restart, resource mutation, or remote push runs.

### Continue activation

Complete database transport and the actual dependency permissions and network paths.
Bind native HTTPS-probe target routing to the registered-target and request-policy contracts.
Complete remaining identity attachment, effective authority, deny effects, protected writes, native command selection, and deployment packaging.
Keep real routing, DNS behavior, token issuance, preview compatibility, and independent timer execution in separate deployment acceptance.

Related guides: [controller host](azure-sandbox-host.md), [registry path](azure-sandbox-host-registry.md), and [runtime](azure-sandbox-runtime.md).
Acceptance 51 advances. The total remains 48 verified of 57, with nine open.
