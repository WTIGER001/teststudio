# Azure worker firewall inspection

## Current boundary

The Azure adapter now verifies the firewall resource reached by the inspected worker default route.
FirewallGuard.Check first refreshes [topology and perimeter inspection](azure-worker-perimeter.md).
It then checks the firewall, its subnet, public IP association, firewall policy, and complete bounded rule-group inventory.
A selected private address alone no longer substitutes for firewall resource ownership.
The [relay firewall observer](azure-relay-firewall.md) now reuses this implementation with separate relay resources and source-prefix settings.

This stage does not implement the execution Network interface or release worker input.
The [DNS stage](azure-worker-dns.md) now checks resolver policy, complete inventories, and actual VNet association.
Private DNS records, private endpoint ownership, isolated exchange service, workload inventory, cross-job isolation, and execution placement remain required.
Azure coded execution stays disabled in native commands and infrastructure packaging.
Local SDK fixtures do not establish Azure platform compatibility, firewall delivery, effective packet isolation, or hosted deployment parity.

## Settings and network attachment

FirewallSettings contains these values:

| Setting | Requirement |
| --- | --- |
| perimeter | The exact immutable perimeter settings used by the previous stage. |
| firewall | The dedicated Azure Firewall resource identifier. |
| policy | Its dedicated firewall policy identifier. |
| publicIp | The dedicated public IP resource identifier used by the firewall. |
| subnetPrefix | The private IPv4 prefix for AzureFirewallSubnet in the worker VNet. |

All resource identifiers select the worker subscription and canonical resource names.
The firewall subnet must contain at least 64 addresses, remain inside the worker VNet, and avoid the delegated worker subnet.
It must contain the exact routed firewall private address and exclude the selected private endpoint addresses.
The constructor preserves the previous perimeter identity and copies nested endpoint settings.

The firewall requires the Standard AZFW_VNet profile, the selected policy, and one egress IP configuration.
Its reported private IP must equal the worker route's configured next hop.
The same configuration must reference the exact AzureFirewallSubnet and selected public IP resource.
Extra IP configurations, classic application rules, classic network rules, classic NAT rules, management configurations, virtual hubs, and unexpected properties reject inspection.
Capacity settings and read-only IP-group metadata do not define traffic authority and do not enter the policy comparison.
Firewall capacity and hosted availability still need installation sizing and deployment verification.

The firewall subnet must report the selected prefix, successful provisioning, and explicitly disabled implicit outbound access.
It must have no subnet NSG, custom route table, NAT gateway, service endpoint, delegation, private endpoint, or sharing configuration.
The public IP must report Standard Regional, static IPv4 allocation, a public address, and successful provisioning.
Its IP configuration must belong to the exact selected firewall configuration.
Linked, service, prefix-based, NAT-associated, or tagged alternate public IP arrangements are outside this profile.

## Dedicated policy

The policy belongs to the selected firewall and region and reports successful provisioning.
It has no parent policy, child policies, policy identity, explicit proxy, TLS interception, or unrelated policy settings.
DNS proxy, required network-rule DNS proxy, and custom DNS servers remain disabled.
Threat intelligence uses Deny. SNAT uses IANAPrivateRanges with automatic private-range learning disabled.
The policy cannot attach to another firewall.

PolicyDefinition, FirewallDefinition, and RuleGroupDefinition return fresh validated SDK values for later infrastructure packaging.
These methods make no Azure write request. Caller mutations cannot change the saved guard configuration.
The full network adapter must bind these settings and all remaining controls into networkProfile before execution enablement.

## Complete rule inventory

The reader lists every rule collection group beneath the selected policy within a four-group and four-page bound.
The returned groups must agree exactly with the policy's group references.
Only worker-platform and the three documented default group names are permitted.
Default groups can be absent or empty. Their priorities must retain the expected 100, 200, and 300 values.
A populated default group cannot supply an additional allow path.
Missing visibility and incomplete inventory leave admission unproven.

The worker-platform group has priority 400 and exactly two application filter collections:

| Collection | Priority | Decision |
| --- | --- | --- |
| worker-platform | 100 | Allow HTTPS port 443 from the delegated worker subnet to the fixed platform domain list. |
| worker-deny-other-applications | 65000 | Deny HTTP port 80 and HTTPS port 443 from every source to all other application names. |

The allow list contains Microsoft Artifact Registry, its data hosts, AKS package hosts, and the documented platform identity hosts.
It adds no public source registry, Blob wildcard, application target, monitoring destination, or arbitrary domain.
Private registry and exchange traffic retain their separate exact private endpoint path.
The NSG and firewall restrictions compose. Neither layer alone establishes the complete outbound policy.

Azure Firewall includes built-in infrastructure allowances after custom application rules.
The explicit final application deny overrides that fallback for the protocols allowed by this profile.
Network rules can take precedence over application rules, and parent policy can take precedence over child policy.
This profile therefore rejects inherited policy, network collections, NAT collections, extra groups, and changed filter actions.

Inspection compares the complete supported traffic-bearing group and rule content.
Changed sources, protocols, ports, destinations, FQDN tags, URLs, web categories, header insertion, IP groups, TLS behavior, names, and priorities reject inspection.
Descriptions, documented read-only metadata, default false TLS termination, and semantic array ordering do not create false drift.
Unknown or changed rule types cannot substitute for the fixed application rules.
The declared rule set still requires live compatibility and effective-isolation verification before hosted enablement.

## Read-only SDK boundary and uncertainty

SDKFirewallReader uses the existing armnetwork/v8 8.0.0 SDK and management API 2025-03-01.
It requires an explicit Azure TokenCredential and permits GET on only the selected firewall, policy, public IP, firewall subnet, and policy group collection.
It grants no read scope to other topology resources, secret endpoints, child actions, or unrelated firewalls.
Automatic retries and resource-provider registration remain disabled.

Only the exact group collection accepts a bounded opaque skip token.
Paging rejects foreign origins, other policies, extra queries, repeated tokens, duplicate parameters, oversized tokens, and excess inventory.
Other readers retain their original query and resource allowlists. Paging does not widen topology or perimeter authority.
Redirects, mutation verbs, alternate API versions, credential-bearing URLs, and oversized responses remain rejected.

A normal complete inspection performs thirteen reads: eight previous-stage reads, four firewall attachment reads, and one group-list read.
The four independent attachment reads run concurrently after the perimeter succeeds.
Additional group pages remain within four pages and sixteen total reads.
The guard has a ten-second total context. Each SDK operation retains its five-second context and four-megabyte response limit.
The guard caches no successful inspection. Concurrent calls share no mutable admission result.

Unavailable, denied, missing, unfinished, cancelled, or incomplete observations do not establish admission.
Raw provider diagnostics remain fixed control errors.
The controller needs read authority on the exact resources and list authority for that policy's rule groups.
The reader needs no mutation, secret, registry-write, or worker identity authority.
Close the reader to close idle transport connections.

## Remaining work

The DNS inspector now checks the actual policy, VNet association, exclusive link, default block, and approved domain lists.
The [selected private-zone inspector](azure-worker-private-dns.md) now verifies exact records and links.
Resolve effective zone visibility and endpoint ownership before complete admission.
Bind private registry and exchange endpoints to approved resources and exact addresses.
Establish isolated exchange authority, unrelated workload inventory, cross-job isolation, and actual execution placement.
Then compose complete Network admission, bind the full execution profile, and add native configuration and infrastructure packaging.
Preserve separate hosted verification for platform startup, identity behavior, compatibility, service-tag membership, and effective isolation.
Do not enable ordinary jobs unless they meet the complete execution isolation contract.

## Targeted local verification

Eight new focused race checks cover the actual resource chain, immutable definitions, declared application policy, effective policy drift, uncertainty, and SDK boundaries.
They reject substituted resources, inherited policy, populated default groups, missing final deny, wildcard allows, classic rules, NAT, and extra groups.
Paging checks cover complete two-page inventory, hidden or missing groups, foreign pages, repeated tokens, and bounded unfinished paging.
Two selected existing topology and perimeter transport checks preserve their read-only boundaries after the paging extension.
Only the affected uncertainty check repeats after adding missing-group and endless-paging handling.

To reproduce the firewall checks locally:

```sh
go test -race ./adapters/execution/azure -run '^TestAzureFirewall' -count=1
```

Evidence is output/azure-worker-firewall/summary.json and the focused command results.
No Azure credential, cloud API, packet probe, registry push, deployment, UI build, browser, container, or image build runs.
Acceptance 51 advances. The verified total remains 48 of 57, with nine open items.

## Primary references

[Container Apps firewall integration](https://learn.microsoft.com/en-us/azure/container-apps/use-azure-firewall) defines the platform connectivity requirements.
[Firewall rule processing](https://learn.microsoft.com/en-us/azure/firewall/rule-processing) defines precedence and the infrastructure fallback override.
[Policy rule sets](https://learn.microsoft.com/en-us/azure/firewall/policy-rule-sets) defines policy, collection group, collection, and rule behavior.
The pinned network SDK defines the resource and policy fields used by this inspector.
