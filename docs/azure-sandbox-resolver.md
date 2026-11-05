# Azure sandbox resolver policy admission

## Introduction

OpenRuntime now constructs SDKResolver with the existing metadata identity.
Creation, source import, and live admission require resolver policy, actual VNet association, and reported subscription private-zone link observations.
The implementation reuses existing DNS policy and subscription inventory verifiers through independent bindings.
It requires no ordinary Jobs configuration or additional credential.

These are reported configuration checks. They do not establish effective DNS resolution or visibility outside the selected subscription.
Native commands and deployment templates still do not enable sandbox execution.

## Concepts

### Settings and definitions

placement.network.resolver requires policy, allowList, and blockList.
Each field identifies the matching resource type in the selected subscription.
The two domain lists must differ.
Invalid scope, type, missing settings, or stale ownership profiles reject before credential construction.
The sandbox-placement/v9 profile binds resolver settings with required trusted services, topology, both endpoint paths, DNS, and placement bindings.
Older staged profiles require regeneration. No native command currently loads them.

ResolverDefinitions returns six reviewable bodies: policy, two domain lists, two security rules, and one VNet link.
It makes no service request or resource mutation. Complete packaging remains unfinished.

### Allowed domains and hierarchy

The approved list contains five domains derived from the selected worker configuration:

- The private exchange hostname.
- The registry login hostname.
- The registry's regional data hostname.
- The registry login hostname under privatelink.azurecr.io.
- The regional data hostname under privatelink.azurecr.io.

There are no Jobs platform, identity, or unrelated service domains in this profile.
The list must match completely. Duplicates, additional domains, and remote domain-list URLs reject.
The block list contains only the root domain, represented by a single dot.

Azure domain rules follow DNS hierarchy. Allowing a listed domain also allows its descendants under that rule.
Do not describe this policy as an exact-host DNS filter.
The independent private stage still requires fixed records, no wildcard or alias records, disabled registration, and no public fallback.
The sandbox's Full inspection and exact exchange-host HTTP allow rule remain independently required.
Actual resolver behavior, cross-subscription visibility, and protection against unobserved DNS changes remain separate work.

### Rules and actual association

The complete rule inventory contains exactly two enabled rules:

| Rule | Priority | Action | Domain list |
| --- | --- | --- | --- |
| sandbox-allow-approved | 100 | Allow | Selected approved list. |
| sandbox-default-block | 65000 | Block | Selected root-domain list. |

Changed priorities, Alert actions, disabled rules, duplicate or extra rules, and different list bindings reject.
The complete resolver link inventory contains only sandbox-network targeting the selected VNet.
The actual listDnsResolverPolicies observation must report exactly the selected policy for that VNet.
Missing or unfinished association observations remain unavailable.
Policy, list, rule, and link identities, regions, types, and successful provisioning states must match.

### Read-only action and request bounds

The selected policy, domain lists, rules, and links use API version 2025-05-01.
GET remains the default transport method for metadata readers.
This resolver reader permits an empty POST to the selected sandbox VNet's listDnsResolverPolicies path.
The relay network reader independently permits the same read action for the selected relay VNet.
That action reads association metadata. Other actions, GET on that action, PUT, request bodies, positive body lengths, filters, and foreign hosts reject.
The transport checks the selected request before acquiring credentials and checks it again before transmission.
The metadata authority profile already includes this read action. It gains no write action.

Rule and link collections retain the existing bounded pagination guards.
Each allows at most two pages and its exact expected item count.
The actual association response must include a non-null value array and no continuation.
Responses have a 1 MiB bound and close their bodies. Redirects, automatic retries, and provider registration remain disabled.
Returned errors omit provider bodies and credential diagnostics.

### Other reported private zones

The reused inventory reader lists private zones in the selected subscription with API version 2024-06-01.
It then observes complete link inventories for every reported zone.
Zone and link identities, successful states, reported link counts, duplicate detection, and registration counts must agree.
The two sandbox private zones and their sandbox-network links must be present.
The relay registry zone and its relay-network link must also be present.
Any other reported zone or link targeting either guarded VNet rejects admission.
Unrelated zones linked to unrelated VNets remain permitted.

The subscription observation retains limits of 1024 zones, 4096 links, 16 pages per collection, and 2048 HTTP requests.
It processes at most eight zones concurrently.
The outer native observation has a five-second bound and joins its selected reads.
Large or slow subscriptions can exceed these bounds and remain unavailable.
Incomplete pages, count gaps, inaccessible metadata, or missing selected zones never qualify as evidence of absence.

This checks the selected subscription's reported inventory.
It does not prove that other subscriptions contain no link to either VNet, or that deny effects hide no relevant resource.
Effective observer authority, deny inspection, protected write boundaries, and deployment ownership remain required.
No claim of complete tenant-wide DNS visibility is made.

### Cache and shutdown

Creation and image import always obtain fresh observations.
Successful evidence can supply concurrent live callers for at most 15 seconds from observation start.
A failed current full observation invalidates prior success. Failures are not cached.
Remaining protection callbacks, sandbox definitions, domain permissions, and worker authorization remain uncached.
Metadata 404 responses remain unavailable and never establish sandbox absence or permit another execution.
Canceled or closed readers return unavailable.
Runtime.Close closes the resolver reader and its inventory resources after joining owned command work.
Injected caller transports remain caller-owned.

## Tutorials

### Verify locally

```sh
go test -race ./adapters/execution/azuresandbox -run '^TestSandboxResolver|^TestSandboxRuntimeResolver' -count=1 -timeout=40s
```

Six new focused checks cover domain definitions and settings, policy drift, other zone links, the empty read action, caching, and automatic runtime composition.
Eighteen selected native checks pass with race detection in 2.171 seconds.
Seven affected shared DNS and subscription inventory checks pass with race detection in 4.645 seconds.
Evidence is output/azure-sandbox-resolver/summary.json, native.log, and shared-dns.log.
The checks use local SQLite, fake credentials, and in-memory ARM responses.
No cloud service, browser, container, UI build, demo restart, resource mutation, or remote push is involved.

### Complete activation

The service stage now checks selected relay and backend metadata, fixed definitions, identity exposure, and revisions.
The backend endpoint stage now binds selected approval, NIC address, network policy, and the relay's configured readiness address.
The [relay network stage](azure-sandbox-relay-network.md) now verifies the selected VNet, egress policy, registry endpoint, DNS, and pull grant.
The [registry service stage](azure-sandbox-registry-service.md) now verifies access policies and complete reported consumer, member, and replication inventories.
Complete trusted consumer network ownership, additional grants, deny effects, credential host attachment, and protection bindings.
Then wire scoped execution and discovery commands and package their dependencies.
Verify actual resolution, routing, preview compatibility, private conversion, role acceptance, timer execution, token audience, and proxy trust during separate deployment acceptance.

Primary contracts: [Microsoft resolver policy](https://learn.microsoft.com/en-us/azure/dns/dns-security-policy)
and [Microsoft DNS security rule schema](https://learn.microsoft.com/en-us/azure/templates/microsoft.network/2025-05-01/dnsresolverpolicies/dnssecurityrules).
Related guides: [private bindings](azure-sandbox-private.md), [topology](azure-sandbox-topology.md), and [runtime](azure-sandbox-runtime.md).
Acceptance 51 advances. The verified count remains 48 of 57, with nine open.
