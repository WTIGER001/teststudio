# Azure relay private registry DNS observation

## Scope

RelayPrivateDNSGuard observes the selected private registry zone after [relay registry endpoint observation](azure-relay-registry-endpoint.md).
It checks the zone, complete record inventory, and complete link inventory against the verified registry member addresses and selected relay VNet.
Worker and relay profiles share one verifier, definition generator, bounded SDK reader, and local fixture implementation.
Existing worker public APIs, zone names, link names, and read limits retain their behavior.

This stage inspects one selected zone. It does not prove that no other visible zone changes effective resolution.
It implements no Network, authorizes no dispatch, and enables no Azure coded execution.
Effective zone visibility, read coverage, attachment authority, preparation networking, identity grants, and complete native wiring remain required.
Acceptance 51 advances. The verified total remains 48 of 57, with nine open.

## Explicit settings and derived records

RelayPrivateDNSSettings requires these values:

| Field | Requirement |
| --- | --- |
| endpoint | Exact RelayRegistryEndpointSettings used by the preceding guard. |
| registryZone | Dedicated private zone identifier named privatelink.azurecr.io in the installation subscription. |

The zone must differ from the selected worker registry zone. The same DNS zone name therefore needs a separate resource group.
Wrong names, foreign subscriptions, queries, missing identifiers, and parent-setting substitution reject configuration.
The constructor preserves the complete preceding settings and their copied address lists.
There is no independent DNS address or region override.

ZoneDefinition, LinkDefinition, and RecordDefinitions return fresh validated SDK values without resource writes.
The two A records derive from the selected registry login name, service data region, and verified endpoint addresses:

| Relative name | Address |
| --- | --- |
| REGISTRY | Selected relay endpoint registryIp. |
| REGISTRY.REGION.data | Selected relay endpoint dataIp. |

The zone contains those two A records and one apex SOA record.
It adds no backend or public exchange record. The relay backend transport retains its separately verified pinned address.
The selected zone and complete inventories must agree on three records, one VNet link, and zero registration-enabled links.

## Records and exclusive relay link

The zone must report its selected identity, global location, resource type, and successful provisioning.
Each A record must report its exact child identity and relative name and contain only its selected private IPv4 address.
Record order does not change authority. Duplicate records, wildcards, aliases, IPv6, extra addresses, automatic registration, and unrelated record types reject observation.
Reported FQDNs must agree with the selected names when present.
Positive record TTLs must be between one and 300 seconds. Fresh definitions select 60 seconds.

The SOA must retain its expected apex identity and valid required fields.
Its effective negative caching bound, the lower of SOA TTL and minimum TTL, must not exceed 300 seconds.
Extra traffic-bearing record fields cannot accompany the SOA or selected A records.
The [worker private DNS guide](azure-worker-private-dns.md) documents the shared record comparison and caching requirements.

The complete link inventory contains only relay-network, attached to the selected relay VNet.
The link must report successful provisioning and completed propagation. Registration must explicitly be disabled.
Default resolution policy retains no public fallback. NxDomainRedirect rejects observation.
Worker or controller links, extra links, changed authority, and registration reject observation.
Missing links, pending propagation, missing records, unknown counters, and inconsistent resource counts remain unavailable.

## Read scope and bounds

NewSDKRelayPrivateDNSReader creates the shared SDKPrivateDNSReader with only the selected zone and its two complete inventory paths.
It requires an explicit credential and uses armprivatedns with management API 2024-06-01.
It permits GET only on the selected zone, its ALL record collection, and its virtualNetworkLinks collection.
Worker zone reads, individual record reads, mutations, secret actions, filters, alternate APIs, foreign resources, and foreign pages remain rejected.

Records permit three entries and at most three pages. Links permit one entry and at most two pages.
Each operation has a five-second context and a four-megabyte response limit. Retries and provider registration remain disabled.
The three selected observations run concurrently after the current endpoint chain succeeds.
They add no duplicate registry service, endpoint, VNet, or source-subnet read.
The composed explicit relay profile has an 85-second context, 78 normal reads, and at most 140 reads with bounded paging.
No successful observation is cached or persisted.

## Targeted local verification and next work

Four new focused race checks and five selected existing checks pass locally in 4.506 seconds.
They cover derived endpoint addresses, parent reuse, fresh observations, dedicated settings, record changes, aliases, registration, fallback, links, SDK scope, and incomplete paging.
Existing worker checks cover record and SOA rules, incomplete authority, SDK construction, and immutable public definitions.
The selected relay endpoint regression verifies fresh service approval and unchanged parent read counts.
Fixtures use explicit fake credentials and in-memory responses. They open no network connection or listener.
No broad suite, cloud request, browser, UI build, image build, container, registry push, deployment, or demo restart runs.

```sh
go test -race ./adapters/execution/azure -run '^(TestAzureRelayPrivateDNS.*|TestAzurePrivateDNSRejectsAliasesWildcardRecordsAndForeignAddresses|TestAzurePrivateDNSPreservesIncompleteAuthorityAndBoundedPaging|TestAzurePrivateDNSSDKScopesSelectedZoneInventoriesAndEscapedGroups|TestAzurePrivateDNSSettingsAndDefinitionsFreezeRoleAddressBindings|TestAzureRelayRegistryEndpointBindsFreshApprovalWithoutRegistryRereads)$' -count=1
```

Evidence is output/azure-relay-private-dns/summary.json and the targeted command result.
The [reported subscription inventory](azure-private-dns-inventory.md) now detects additional reported links into worker and relay VNets.
Continue authoritative effective VNet-wide zone visibility and attachment controls for worker and relay networks.
A selected-zone record list or Resource Graph snapshot cannot establish that complete view or its authorization coverage.
Keep live propagation, registry CNAME behavior, image pulls, and packet isolation in the separate hosted verification phase.
