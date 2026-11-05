# Azure relay firewall observation

## Scope

RelayFirewallGuard observes the selected relay firewall after [relay security rules and routing](azure-relay-perimeter.md).
It binds the default route's private next hop to the firewall resource, dedicated policy, subnet, and public IP association.
It checks the complete bounded rule-group inventory and compares the firewall subnet with the preceding validated VNet observation.
It adds five normal reads without repeating VNet or source-subnet reads.

The worker and relay use one firewall implementation for SDK reads, resource comparison, policy definitions, and bounded inventory.
Their public settings and selected resources remain separate. Existing worker rule names and public APIs retain their behavior.
This stage implements no Network, authorizes no dispatch, and enables no Azure coded execution.
The [relay DNS observer](azure-relay-dns.md) now checks its resolver policy, lists, rules, link, and actual VNet association.
Effective private-zone visibility, registry endpoint ownership, identity and attachment authority, execution isolation, and complete provider wiring remain required.
Acceptance 51 advances. The verified total remains 48 of 57, with nine open.

## Explicit settings and attachment

RelayFirewallSettings requires these values:

| Field | Requirement |
| --- | --- |
| perimeter | Exact RelayPerimeterSettings used by the preceding guard. |
| firewall | Dedicated relay Azure Firewall identifier. |
| policy | Its dedicated firewall policy identifier. |
| publicIp | Dedicated public IP identifier for the firewall. |
| subnetPrefix | Canonical private IPv4 prefix for AzureFirewallSubnet in the relay VNet. |

Identifiers must belong to the installation subscription. The relay firewall, policy, and public IP cannot reuse their worker counterparts.
The firewall subnet contains at least 64 addresses and lies entirely within the relay VNet.
It must avoid the relay source and backend endpoint prefixes and exclude every selected registry endpoint address.
The route's firewall address must lie within this subnet. Its first four addresses and final address cannot identify the firewall.
Public ranges, mapped IPv6, host bits, overlapping prefixes, foreign resources, and parent-setting substitution reject configuration.

The reserved AzureFirewallSubnet name is permitted explicitly in the relay VNet inventory.
Other subnet names retain the existing canonical naming rules. This exception does not permit arbitrary names or foreign parents.
The subnet's direct observation must agree with its embedded VNet observation. Missing or inconsistent observations remain unavailable.
The validated VNet exists only during the composed call. It is not cached or persisted.

## Shared resource and policy requirements

The selected firewall must report Standard AZFW_VNet, successful provisioning, the selected policy, and one egress IP configuration.
Its private IP must equal the relay route's configured next hop.
That configuration must identify the selected firewall subnet and public IP resource.
Extra IP configurations, classic network or NAT rules, classic application rules, management configurations, virtual hubs, and unexpected properties reject observation.

The firewall subnet must explicitly disable default outbound access and contain no NSG, route table, NAT gateway, delegation, service endpoint, or private endpoint.
The public IP must report Standard Regional, static public IPv4 allocation, successful provisioning, and the exact firewall configuration backlink.
Alternate linked, service, prefix, NAT, and tagged public IP arrangements remain outside this profile.

The policy must belong only to the selected firewall. It permits no parent policy, child policy, policy identity, explicit proxy, or TLS interception.
Threat intelligence uses Deny. SNAT uses IANAPrivateRanges without automatic learning.
DNS proxy and custom DNS servers remain disabled. Their absence does not establish effective DNS isolation.
FirewallDefinition, PolicyDefinition, and RuleGroupDefinition return fresh validated SDK values without resource writes.

The relay-platform group has priority 400 and exactly two application filter collections:

| Collection | Priority | Requirement |
| --- | --- | --- |
| relay-platform | 100 | Allow HTTPS 443 from the exact relay source prefix to the shared fixed platform domain list. |
| relay-deny-other-applications | 65000 | Deny HTTP 80 and HTTPS 443 from every source to all other names. |

The [worker firewall guide](azure-worker-firewall.md#complete-rule-inventory) documents the shared platform domains and comparison rules.
The relay adds no registry wildcard, Blob wildcard, application target, arbitrary domain, FQDN tag, URL, or extra protocol exception.
Private backend and registry traffic retain the separate exact private endpoint path.
Only the selected relay group and up to three empty platform default groups can appear.
Reported policy references must agree with the complete group inventory. Populated defaults, inherited policy, and extra collections reject observation.

These checks establish reported configuration. They do not establish effective packet isolation or live platform startup compatibility.
The separate hosted phase must verify image pulls, platform identity behavior, probes, routes, private transport, and certificate trust.

## SDK scope, bounds, and targeted local checks

NewSDKRelayFirewallReader creates the shared SDKFirewallReader with five selected paths and an explicit credential.
It reads only the selected firewall, policy, firewall subnet, public IP, and policy rule-group collection.
Network resources use armnetwork/v8 and management API 2025-03-01.
Only the exact group collection accepts bounded paging: at most four groups and four pages.
Foreign resources, child actions, mutations, alternate APIs, foreign pages, repeated tokens, redirects, and oversized responses remain rejected.
Individual reads have five-second contexts and four-megabyte response limits. Retries and provider registration remain disabled.
The composed guard has a 70-second context, 65 normal reads, and at most 121 reads with bounded parent paging.

Four new focused race checks and seven selected existing checks pass across targeted local commands.
They cover fresh ownership, parent reuse, firewall and public IP backlinks, policy bypasses, subnet consistency, settings, SDK scope, and paging uncertainty.
The first batch exposes rejection of the reserved firewall subnet name.
After that correction, only the three failed checks and the affected topology conflict check run again. The final command passes in 4.431 seconds.
Existing worker checks cover resource substitution, effective policy bypasses, SDK construction, and platform declarations.
Existing relay checks cover source-perimeter reuse and fresh topology observations.
Fixtures use explicit fake credentials and in-memory responses. They open no network connection or listener.
No broad suite, cloud request, browser, UI build, image build, container, registry push, deployment, or demo restart runs.

Evidence is output/azure-relay-firewall/summary.json and the targeted command results.
Continue effective private-zone visibility, registry endpoint ownership, effective identity and attachment authority, and complete native admission.
