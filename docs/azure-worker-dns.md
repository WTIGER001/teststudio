# Azure worker DNS policy inspection

## Current boundary

The Azure adapter now checks the worker VNet's actual DNS resolver policy association, policy links, security rules, and domain lists.
DNSGuard.Check first refreshes the [firewall and perimeter stages](azure-worker-firewall.md).
It then inspects the DNS resources and complete bounded inventories through the official SDK.
A configured policy name alone does not establish that the VNet uses that policy.

This stage does not implement the execution Network interface or release worker input.
The [selected private-zone stage](azure-worker-private-dns.md) now checks exact records, completed links, registration, and fallback settings.
Effective VNet-wide zone visibility, endpoint ownership, isolated exchange service, workload inventory, cross-job isolation, and execution placement remain required.
Azure coded execution stays disabled in native commands and infrastructure packaging.
Local fixtures make no DNS lookup and do not establish effective Azure resolution, propagation, CNAME behavior, or hosted deployment parity.

## Explicit installation settings

DNSSettings requires these values:

| Setting | Requirement |
| --- | --- |
| firewall | The exact immutable settings used by the previous firewall stage. |
| policy | The selected DNS resolver policy resource identifier. |
| allowList | The installation's approved-domain list resource identifier. |
| blockList | A separate installation domain list containing the DNS root. |
| registryDataRegion | The explicit Azure region of the selected registry data endpoint. |

Resource identifiers belong to the selected subscription and use the canonical resource naming rules.
Policy, lists, rules, and links use the worker VNet's region.
Registry data region can differ from worker region. The later endpoint inspector must verify the declared data endpoint and region.
The initial profile selects one registry data region. Additional replicas require explicit policy and endpoint support.

The constructor preserves all prior firewall and perimeter settings and copies nested endpoint addresses.
PolicyDefinition, DomainListDefinition, RuleDefinition, and LinkDefinition return fresh SDK values for later packaging.
They validate the complete supplied configuration. Mutating a returned definition does not change the saved guard.
These methods make no Azure write request.
The complete network adapter must bind these settings and every remaining control into networkProfile before enablement.

## Required rules and domains

The policy contains exactly two enabled security rules:

| Rule | Priority | Action and list |
| --- | --- | --- |
| worker-allow-approved | 100 | Allow the exact selected approved-domain list. |
| worker-default-block | 65000 | Block the exact selected root-domain list. |

The block list contains only the dot domain, `.`.
A threat-intelligence list or a list of known malicious names cannot substitute for this terminal block.
Alert actions, disabled rules, changed priorities, extra rules, duplicate rules, and additional list references reject inspection.

The approved list combines these declarations:

- The nine fixed platform destinations used by the firewall stage.
- The configured HTTPS worker exchange hostname.
- The exact registry login hostname and selected regional data hostname.
- The corresponding registry and data hostnames beneath privatelink.azurecr.io.

The list contains no generic Blob wildcard, arbitrary application target, package repository, unrelated registry, or unselected data region.
List order does not change policy. Duplicate, missing, unexpected, noncanonical, or null entries reject inspection.
Lists must expose their small inline content. Bulk list URLs are rejected and never followed.
An empty or unfinished observation does not prove that the policy is complete.

## Native hierarchy and private DNS requirements

Azure DNS policy uses DNS hierarchy. Allowing a parent name can also allow descendants.
This inspector does not claim exact-host-only DNS matching.
Its declaration checks include an allowed child beneath the exchange hostname to retain this distinction.
Azure also inspects CNAME chains. Actual platform alias handling remains part of hosted verification.

Private DNS and endpoint inspection must establish the exchange hostname's authoritative private zone, exact records, and linked worker VNet.
Do not depend on a customer-controlled public hostname and its descendant queries as the final isolation boundary.
Verify the exact private registry records and region-specific data records against the selected endpoint addresses.
Reject alternate records, unsafe aliases, registration, delegation, and public fallback that weaken the intended private path.
The selected-zone inspector now verifies the declared zone, exact records, and complete links.
It does not establish VNet-wide zone visibility or endpoint ownership. Those remain separate required controls.

The previous topology stage requires Azure-provided DNS and a dedicated worker VNet.
The previous NSG permits only the intended platform DNS path and denies other external DNS destinations.
A DNS proxy, empty custom-DNS list, or NSG rule alone cannot establish this resolver policy.
The DNS policy affects the whole linked VNet. It must not apply to the trusted controller network.

## Actual association and complete inventory

The selected policy must report the exact identity, type, region, and successful provisioning.
Its security-rule inventory must contain only the two selected rules with the expected complete supported properties.
Its link inventory must contain only worker-network, linked to the exact worker VNet.
An extra link, including a controller link, rejects inspection.

The inspector also calls the worker VNet's listDnsResolverPolicies operation.
That result must identify only the selected policy. A declared link alone cannot substitute for the actual VNet association.
No association, missing links, incomplete rules, or unready resources leave admission unproven.
Multiple policies, foreign policy identity, substituted resources, and changed rule authority reject inspection.

Rules allow at most two items and two pages. Links allow at most one item and two pages.
The VNet association operation allows one policy and one complete response.
A continuation response from that operation remains unavailable. The inspector does not follow it or assume the first page is complete.
Bounded rule and link paging rejects foreign origins, other parents, repeated tokens, extra queries, and excess inventory.
The canonical bounded inventory helper now serves DNS and firewall inspection.

## SDK scope and operator authority

SDKDNSReader pins armdnsresolver 1.3.0 and management API 2025-05-01.
It uses an explicit Azure TokenCredential and the existing fixed public management endpoint.
The transport allows GET on the selected policy, two domain lists, security-rule collection, and link collection.
Only the exact worker VNet association operation uses POST. This documented operation reads association inventory and has no request body.
Other POST actions, writes, deletes, bulk endpoints, foreign VNets, child actions, credentials in URLs, and substituted resource kinds are rejected.
The action exception does not widen the existing topology, perimeter, or firewall read scope.
Automatic retries and resource-provider registration remain disabled.

The controller needs read authority on the selected DNS policy, rules, links, and domain lists.
It also needs Microsoft.Network/virtualNetworks/listDnsResolverPolicies/action on the exact worker VNet.
The inspector requires no DNS mutation, domain-list bulk, secret, registry-write, or worker identity authority.
Close the reader to close idle transport connections.

A normal complete DNS inspection makes nineteen reads, including thirteen prior-stage reads and six DNS observations.
The six DNS observations run concurrently after the firewall succeeds.
The full bound is twenty-four reads when earlier inventories and DNS rule or link inventories paginate.
The guard has a fifteen-second total context. Each SDK operation retains a five-second context and four-megabyte response limit.
No successful-inspection cache can hide later changes.
Provider diagnostics remain fixed control errors.
Missing, denied, unavailable, redirected, oversized, cancelled, unfinished, and incomplete observations do not establish admission.

## Targeted local verification

Seven new focused race checks cover actual association, exact lists and rules, exclusive links, concurrency, immutable definitions, uncertainty, and SDK boundaries.
They include a nineteen-case declaration matrix for approved, unapproved, descendant, private-link, and region-specific names.
This matrix verifies declared hierarchy and default-block intent. It does not perform recursive DNS resolution or inspect live CNAME chains.
Tests reject extra rules, disabled blocks, foreign list references, bulk URLs, controller links, and hidden or conflicting association state.
They also cover bounded rule paging, escaped resource groups, read-only POST scope, payload rejection, and resource-kind substitution.
Three selected existing firewall, perimeter, and topology checks preserve shared paging and transport behavior.
Only affected region and settings checks repeat after making registry data region explicit.

To reproduce the DNS checks locally:

```sh
go test -race ./adapters/execution/azure -run '^TestAzureDNS' -count=1
```

Evidence is output/azure-worker-dns/summary.json and the focused command results.
No Azure credential, cloud API, DNS probe, packet probe, registry push, deployment, UI build, browser, container, or image build runs.
Acceptance 51 advances. The verified total remains 48 of 57, with nine open items.

## Primary references

[Resolver policy semantics](https://learn.microsoft.com/en-us/azure/dns/dns-security-policy) defines rule priority, VNet association, hierarchy, and CNAME handling.
[Domain list read](https://learn.microsoft.com/en-us/rest/api/dns/dnsresolver/dns-resolver-domain-lists/get?view=rest-dns-dnsresolver-2025-05-01) defines inline domains and bulk-content references.
[Policy links](https://learn.microsoft.com/en-us/rest/api/dns/dnsresolver/dns-resolver-policy-virtual-network-links/list?view=rest-dns-dnsresolver-2025-05-01) defines the complete link inventory.
[Actual VNet association](https://learn.microsoft.com/en-us/rest/api/dns/dnsresolver/dns-resolver-policies/list-by-virtual-network?view=rest-dns-dnsresolver-2025-05-01) defines the read-only POST operation.
[Networking permissions](https://learn.microsoft.com/en-us/azure/role-based-access-control/permissions/networking) defines its scoped authorization action.
[Registry private DNS](https://learn.microsoft.com/en-us/azure/container-registry/container-registry-private-link) defines private registry and regional data records.
