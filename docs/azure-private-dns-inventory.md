# Azure reported private DNS inventory

## Scope and evidence limit

PrivateDNSInventoryGuard checks reported private-zone links across the configured subscription after [selected relay zone observation](azure-relay-private-dns.md).
It uses direct management API inventories rather than Resource Graph or a resource-group-only list.
It detects reported links from unapproved zones into the selected worker or relay VNet.
It also requires the three selected zone links to appear in that reported inventory.

This is reported-subscription inspection. A matching result does not establish complete effective VNet-wide visibility or attachment authority.
Azure permits private-zone links across subscriptions and tenants. The [Azure Private DNS FAQ](https://learn.microsoft.com/en-us/azure/dns/dns-faq-private) documents those arrangements.
The [zone list API](https://learn.microsoft.com/rest/api/dns/privatedns/private-zones?view=rest-dns-privatedns-2018-09-01) lists one subscription.
The [link list API](https://learn.microsoft.com/en-us/rest/api/dns/privatedns/virtual-network-links/list?view=rest-dns-privatedns-2018-09-01) lists children of one zone.
Those scopes imply that this scan alone cannot establish the absence of linked zones in other subscriptions or tenants.
A successful list response also does not independently establish effective authorization coverage or prevent later attachments.

The guard implements no Network, authorizes no dispatch, and enables no Azure coded execution.
Cross-scope coverage, installation authority, attachment controls, trusted preparation networking, execution isolation, and full native wiring remain required.
Acceptance 51 advances. The verified total remains 48 of 57, with nine open.

## Reported-link contract

The guard uses the exact RelayPrivateDNSSettings and Config from the preceding selected-zone guard.
It adds no operator-selected domain exceptions, alternate subscription, or independent resource list.
SDKPrivateDNSInventoryReader receives an explicit credential and the installation subscription.
Observe returns only zone identifiers and reduced link references. It retains no record values, tags, or unrelated service configuration.

Every reported zone must identify a unique private-zone resource in the selected subscription.
Its identity, global location, provisioning state, total link count, and registration-enabled link count must be available and consistent.
Resource groups can differ. Identity comparison preserves case-insensitive ARM semantics.
The reader normalizes request paths without changing resource-group or zone names.

For each reported zone, the reader lists its complete link inventory, including zones whose reported count is zero.
It does not assume a zero counter proves absence of links without that list.
Link identities must belong to their reported parent zone and identify valid VNet references.
Provisioning must succeed and link propagation must complete. Registration state must be explicit.
Observed link and registration counts must agree with the zone's reported counters.
Missing arrays, missing metadata, pending state, and inconsistent counters remain unavailable.

The reported inventory must include these exact links:

| Zone | Link | Target |
| --- | --- | --- |
| Selected worker exchange zone | worker-network | Selected worker VNet. |
| Selected worker registry zone | worker-network | Selected worker VNet. |
| Selected relay registry zone | relay-network | Selected relay VNet. |

Registration must remain disabled on those links.
Any other reported zone or link targeting either selected VNet rejects ownership.
Missing selected zones or links remain unavailable. Duplicate, foreign-subscription, malformed, or conflicting references reject ownership.
Unrelated valid zones and links remain permitted. Their targets can belong to another subscription; the reader issues no request to those targets.
Records and no-fallback behavior remain the responsibility of the preceding selected-zone checks.

## SDK request boundary and bounds

The reader uses armprivatedns and management API 2024-06-01.
It first permits GET only on the selected subscription's privateDnsZones collection.
After that complete list validates, it adds only virtualNetworkLinks paths beneath the zones returned by that list.
The allowed child paths exist only during the current Observe call. Caller-supplied zone identifiers cannot widen them.
No selected-zone records, arbitrary resources, mutations, credentials, filters, expansions, foreign origins, or foreign pages become readable.
Automatic retries and provider registration remain disabled. Each response retains the shared four-megabyte bound.

| Bound | Limit |
| --- | --- |
| Reported zones | 1024. |
| Zone collection pages | 16. |
| Links per zone | 1024. |
| Link collection pages per zone | 16. |
| Total reported links | 4096. |
| ARM requests per Observe call | 2048. |
| Concurrent zone-link reads | Eight. |
| Individual collection operation | Five seconds. |
| Complete Observe call | Thirty seconds. |

Excess, unfinished, denied, or unavailable inventory cannot return partial approval.
The per-call request budget uses atomic accounting across concurrent reads.
The composed guard has a 115-second context and at most 2188 reads including its existing 140-read parent bound.
Normal scan cost is one subscription list plus one link collection per reported zone, before paging.
It does not repeat record inventories. Its broader link lists independently establish the reported subscription inventory.
No successful observation or dynamically discovered allowlist persists between calls.

## Targeted local verification and next work

Four new focused race checks and two selected existing checks pass locally in 3.379 seconds.
They cover fresh reported links, parent gating, mixed-case identities, unexpected worker or relay links, unrelated and zero-link zones, and malformed references.
They also cover missing arrays, count mismatches, incomplete propagation, safe paging, foreign pages, request exhaustion, and configuration binding.
Existing checks preserve the selected relay zone and scoped private DNS SDK behavior.
Fixtures use explicit fake credentials and in-memory responses. They open no network connection or listener.
No broad suite, cloud request, browser, UI build, image build, container, registry push, deployment, or demo restart runs.
The only external requests in this batch read public Microsoft documentation.

```sh
go test -race ./adapters/execution/azure -run '^(TestAzurePrivateDNSInventory.*|TestAzureRelayPrivateDNSBindsFreshEndpointAddressesWithoutParentRereads|TestAzurePrivateDNSSDKScopesSelectedZoneInventoriesAndEscapedGroups)$' -count=1
```

Evidence is output/azure-private-dns-inventory/summary.json and the targeted command result.
Continue independent effective-read coverage and controls on authority to link or change zones, VNets, and environments.
The [selected deny observer](azure-network-deny.md) now reads actual protection metadata for the worker and relay VNets and their three selected zones.
It does not establish deployed enforcement, remove pre-existing cross-scope links, or prove who can change protection through its owning workflow.
Inspect actual enforcement before relying on installation protection settings. A proposed policy or deployment-stack flag alone proves no effective deny arrangement.
Keep native coded execution disabled while cross-scope coverage, attachment authority, or execution isolation remains unproven.
