# Azure worker private DNS records

## Current boundary

The [relay private DNS observer](azure-relay-private-dns.md) now reuses the verifier, definitions, scoped SDK reader, and fixtures with a separate registry zone.

The Azure adapter now inspects the two selected private DNS zones after [resolver policy inspection](azure-worker-dns.md).
It checks complete record and link inventories, exact role addresses, ready links, registration settings, and public fallback settings.
The [endpoint stage](azure-worker-endpoints.md) now binds actual endpoint and NIC resources to the selected DNS addresses.
Correct DNS records alone do not establish private endpoint ownership.
This stage inspects selected zones. It does not prove that no other linked zone can change the VNet's effective DNS view.

PrivateDNSGuard.Check does not implement Network or release worker input.
Service-side approval, VNet-wide DNS visibility and attachment controls, isolated exchange authority, workload inventory, and cross-job isolation remain required.
Azure coded execution stays disabled in native commands and infrastructure packaging.
Local fixtures do not perform recursive DNS resolution or establish hosted propagation, cache behavior, CNAME compatibility, or deployment parity.

## Explicit zone and address settings

PrivateDNSSettings requires these values:

| Setting | Requirement |
| --- | --- |
| dns | The exact immutable settings used by the resolver-policy stage. |
| exchangeZone | A private zone whose name equals the complete configured exchange hostname. |
| registryZone | The selected private zone named privatelink.azurecr.io. |
| exchangeIp | The selected exchange endpoint's private IPv4 address. |
| registryIp | The selected registry login endpoint's private IPv4 address. |
| dataIp | The selected regional registry data endpoint's private IPv4 address. |

Zone identifiers use the selected subscription, explicit resource group, and canonical zone name.
A parent exchange zone cannot substitute for the zone at the exact exchange hostname.
The exchange cannot use the registry's azurecr.io namespace.
The three role addresses must be unique and equal the complete three-address allowlist in the previous perimeter settings.
Those earlier settings require private addresses inside the worker VNet and outside worker and firewall subnets.
This initial profile selects one exchange endpoint and one registry data region. Extra endpoints require explicit profile support.

ZoneDefinition, LinkDefinition, and RecordDefinitions return fresh validated SDK values for later packaging.
They issue no Azure write request. Caller mutations cannot change the saved guard.
The full network profile must include these settings and every remaining admission control before native enablement.

## Selected private authority and records

Both zones must report the expected identifier, name, private-zone type, global location, and successful provisioning.
Each zone has one worker-network link to the exact worker VNet.
The link must finish provisioning and reach Completed.
RegistrationEnabled must explicitly be false.
Public NXDOMAIN redirection and unknown resolution policies reject inspection.

The registry link definition explicitly selects Default resolution policy.
The custom exchange-zone definition omits ResolutionPolicy because Azure restricts that setting to private-link zones.
Absent and Default observations both represent no public fallback for this inspection.
Extra links, controller links, and unknown registration settings cannot establish the intended private boundary.

The exchange zone contains exactly two record sets: its SOA and one apex A record with exchangeIp.
The registry zone contains exactly three record sets: its SOA, the registry login A record, and the selected regional data A record.
The registry record names derive from the exact configured registry and explicit registryDataRegion.
Each A record contains exactly its selected role address.
The definitions use a sixty-second A-record TTL. Inspection accepts positive TTL values from one through 300 seconds.
This permits bounded provider-managed record TTLs without allowing long positive caches.

The inspector rejects extra names, wildcard records, CNAMEs, IPv6, alternate addresses, mixed record data, foreign child identifiers, and auto-registered records.
Reported FQDNs must agree with the exact relative record and zone when supplied.
Record metadata and inventory order do not change DNS authority.
SOA records must contain valid required timing values and no mixed record content.
The effective negative caching bound, the lower of SOA TTL and SOA minimum TTL, must not exceed 300 seconds.
SOA contact and host values describe the record. They do not replace the selected Azure private-zone resource as the authority check.

Root zone record counts, link counts, and registration counts must agree with the complete observed inventories.
Missing records, unfinished links, unavailable counts, and unsettled count observations leave admission unproven.
DNS propagation and existing caches still require separate hosted verification.

## Remaining effective-view and endpoint checks

The [reported subscription inventory](azure-private-dns-inventory.md) now checks direct zone and link lists for additional worker or relay links.
That scan does not establish cross-subscription or cross-tenant coverage, effective read authority, or protection against later attachments.
The [selected deny observer](azure-network-deny.md) now checks enforced, unconditional assignment metadata on both VNets and the three selected zones.
Actual enforcement, installation provenance, existing cross-scope links, and protection-management authority remain unproven.

This reader requests only the selected zones and their children.
It does not enumerate all zones in every resource group, subscription, or tenant that could be linked to the worker VNet.
Resolve effective linked-zone visibility and the authority to attach or change zones before composing complete network admission.
Do not substitute a Resource Graph snapshot or the selected-zone record list for an authoritative effective-view check.
Keep coded execution disabled while that condition remains unproven.

The endpoint inspector now verifies selected resource identity, consumer-side approval, subnet policy, NIC ownership, and regional address members.
It binds those observations to the three DNS role addresses and the perimeter allowlist.
Actual service configuration and service-side approval remain required.
The exchange service must expose only the isolated worker exchange contract through its selected private path.
Public DNS lookup success or an A record alone cannot prove any of those conditions.

Trusted preparation also needs private registry access once public registry access is disabled.
Keep its endpoint and DNS path in trusted controller networking.
This worker profile requires exclusive worker links and cannot add a controller link to the same selected worker zone.
Provide the trusted path through separate verified resources. Do not solve preparation access by peering workers into the controller network.
Record the complete role and endpoint arrangement in infrastructure packaging before enabling coded execution.

## SDK and inventory boundary

SDKPrivateDNSReader pins armprivatedns 1.3.0 and management API 2024-06-01.
It requires an explicit Azure TokenCredential and uses the existing fixed public management endpoint.
It permits GET on exactly the two selected zones, each ALL-record collection, and each virtual-network-link collection.
The ALL collection prevents an A-only or name-filtered view from hiding aliases or other record types.
Writes, deletes, POST actions, zone-wide discovery, individual foreign records, record filters, alternate versions, and foreign resources are rejected.
The DNS policy reader's association POST exception does not apply to this client.
Automatic retries and resource-provider registration remain disabled.

Exchange records allow at most two items. Registry records allow at most three items.
Each record inventory permits at most three pages. Each link inventory permits one item and two pages.
The shared pager rejects foreign origins, different zones, token cycles, extra queries, and excess inventory.
Incomplete paging remains unavailable.

A normal complete check makes twenty-five reads, including nineteen prior-stage reads and six selected-zone observations.
The six new observations run concurrently after the resolver policy succeeds.
The full paging bound is thirty-six reads. The guard has a twenty-second total context.
Each SDK operation retains its five-second context and four-megabyte response limit.
No successful-inspection cache hides later changes.
Provider diagnostics remain fixed control errors.

The controller needs read authority on the selected private zones, their record sets, and their links.
The reader requires no DNS writes, zone attachment, registry credential, secret, or worker identity authority.
Close the reader to close idle transport connections.

## Targeted local verification

Six new focused race checks cover exact selected authority, role addresses, complete records, link readiness, immutability, and typed SDK scope.
They reject wildcard records, aliases, IPv6, alternate addresses, long caching, auto-registration, controller links, and public fallback.
Inventory checks cover missing records, unsettled counts, safe pagination, foreign pages, token cycles, and bounded unfinished paging.
SDK checks cover escaped resource groups, rejected mutations, filtered lists, foreign records, and unrelated POST actions.
Two selected existing DNS and firewall paging checks pass without broader reruns.

To reproduce these local checks:

```sh
go test -race ./adapters/execution/azure -run '^TestAzurePrivateDNS' -count=1
```

Evidence is output/azure-worker-private-dns/summary.json and the focused command results.
No Azure credential, cloud API, DNS probe, packet probe, registry push, deployment, UI build, browser, container, or image build runs.
Acceptance 51 advances. The verified total remains 48 of 57, with nine open items.

## Primary references

[Private endpoint DNS](https://learn.microsoft.com/en-us/azure/private-link/private-endpoint-dns) distinguishes DNS resolution from endpoint access and ownership.
[Private zone links](https://learn.microsoft.com/en-us/azure/dns/private-dns-virtual-network-links) defines registration and completed resolution links.
[Public fallback](https://learn.microsoft.com/en-us/azure/dns/private-dns-fallback) defines NxDomainRedirect and the supported API version.
[Private-zone override behavior](https://learn.microsoft.com/en-us/troubleshoot/azure/dns/troubleshoot-private-dns-zone-override-nxdomain) explains authoritative NXDOMAIN and setting restrictions.
[Registry private endpoints](https://learn.microsoft.com/en-us/azure/container-registry/container-registry-private-link) defines the registry private zone and regional data records.
The pinned private DNS SDK defines the inspected resource and record fields.
