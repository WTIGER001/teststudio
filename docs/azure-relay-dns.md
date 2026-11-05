# Azure relay DNS policy observation

## Scope

RelayDNSGuard checks the relay VNet's resolver policy after [relay firewall observation](azure-relay-firewall.md).
It observes the selected policy, domain lists, complete rule and link inventories, and the VNet's actual policy association.
The worker and relay share one verifier, definition generator, SDK reader, bounded inventory routine, and local fixture implementation.
Existing public worker APIs, rule names, exchange hostname declarations, and resource scope retain their behavior.

This stage implements no Network, authorizes no dispatch, and enables no Azure coded execution.
It checks reported resolver configuration. It performs no DNS lookup or live Azure request during local verification.
Effective private-zone visibility, record ownership, registry endpoint ownership, identity and attachment authority, and complete native wiring remain required.
Acceptance 51 advances. The verified total remains 48 of 57, with nine open.

## Explicit settings and declarations

RelayDNSSettings requires these values:

| Field | Requirement |
| --- | --- |
| firewall | Exact RelayFirewallSettings used by the preceding guard. |
| policy | Dedicated relay DNS resolver policy identifier. |
| allowList | Dedicated approved-domain list identifier. |
| blockList | Separate dedicated root-domain block list identifier. |

Identifiers must belong to the installation subscription and use canonical resource names.
The relay cannot reuse the worker policy or either worker domain list. Its allow and block lists must differ.
Parent configuration substitution, wrong resource kinds, foreign subscriptions, and list aliases reject configuration.
Registry data region comes from the preceding selected registry profile. There is no independent relay region override.
PolicyDefinition, DomainListDefinition, RuleDefinition, and LinkDefinition return fresh validated SDK values without resource writes.

The relay approved list contains thirteen declarations:

- The nine fixed platform domains used by the shared firewall profile.
- The selected registry login and regional data hostnames.
- Their corresponding registry and regional data names beneath privatelink.azurecr.io.

The list omits the public exchange hostname and backend hostname. The relay's pinned backend transport does not need those lookups.
It adds no arbitrary application name, registry wildcard, Blob wildcard, or other data region.
The [worker DNS guide](azure-worker-dns.md#native-hierarchy-and-private-dns-requirements) explains DNS hierarchy and CNAME limits.
Declared names can permit descendants. This stage does not claim exact-host-only DNS matching or verified live CNAME behavior.

## Rules, association, and missing observations

The policy must report its selected identity, region, and successful provisioning.
Both domain lists must expose exact inline contents and successful provisioning. Bulk URLs are rejected and never followed.
List order does not change policy. Duplicate, missing, null, and unexpected domains cannot establish the selected configuration.
The block list contains only the DNS root, dot.

The policy contains exactly two enabled rules:

| Rule | Priority | Requirement |
| --- | --- | --- |
| relay-allow-approved | 100 | Allow only the selected approved-domain list. |
| relay-default-block | 65000 | Block only the selected root-domain list. |

Changed actions, priorities, list references, rule names, disabled rules, duplicate rules, and extra rules reject observation.
The complete link inventory contains only relay-network, linked to the selected relay VNet.
Worker, controller, foreign, duplicate, and additional links reject observation.
The VNet's listDnsResolverPolicies result must independently identify only the selected relay policy.
A declared link cannot substitute for this actual association observation.
Missing links, no association, unfinished provisioning, or incomplete inventories remain unavailable.

The preceding VNet and source NSG checks retain Azure-provided DNS and the selected platform DNS path.
This policy check does not establish all private zones visible to that VNet or prevent later network and identity changes.
Native installation controls must still prove effective visibility, attachment authority, read coverage, and deployed resolution.

## SDK scope and bounds

NewSDKRelayDNSReader creates the shared SDKDNSReader with six selected paths and an explicit credential.
It uses armdnsresolver and management API 2025-05-01.
GET permits only the policy, two domain lists, security-rule collection, and virtual-network-link collection.
Only the selected relay VNet's listDnsResolverPolicies action uses POST. It reads association inventory and accepts no request body.
Other POST actions, mutations, bulk URLs, child actions, foreign VNets, alternate APIs, and foreign hosts remain rejected.
The read-action exception does not widen other readers' authority.

Rules permit at most two entries and two pages. Links permit one entry and two pages.
The VNet association permits one policy and one complete response. A continuation remains unavailable and is not followed.
Individual operations have five-second contexts and four-megabyte response limits. Retries and provider registration remain disabled.
The six observations run concurrently after the parent firewall succeeds.
The composed context is 75 seconds, with 71 normal reads and at most 129 reads with bounded parent paging.
An explicit relay registry consumer adds one possible parent page, for a maximum of 130 reads. It adds no normal read.
No successful observation is cached or persisted. No duplicate parent read is added.

## Targeted local verification and next work

Four new focused race checks and five selected existing checks pass locally in 4.106 seconds.
They cover fresh association, parent reuse, exact domains and rules, worker or controller substitution, shared policy, SDK scope, and incomplete paging.
They also verify fresh definitions, immutable parent settings, the root-domain block, and omission of backend and exchange lookups.
Existing worker checks cover domain bypasses, DNS hierarchy declarations, incomplete observations, and scoped SDK construction.
The selected relay firewall regression verifies current parent ownership and network-observation reuse.
Fixtures use explicit fake credentials and in-memory responses. They open no network connection or listener.
No broad suite, cloud request, browser, UI build, image build, container, registry push, deployment, or demo restart runs.

```sh
go test -race ./adapters/execution/azure -run '^(TestAzureRelayDNS.*|TestAzureDNSRejectsDomainRuleAndAssociationBypasses|TestAzureDNSSDKPinsReadInventoryActionAndEscapedResources|TestAzureDNSDeclaredDecisionsPreserveHierarchyAndDefaultBlock|TestAzureDNSPreservesPendingMissingAndIncompletePolicyState|TestAzureRelayFirewallBindsFreshOwnershipAndReusesNetwork)$' -count=1
```

Evidence is output/azure-relay-dns/summary.json and the targeted command result.
Continue selected relay private records, effective VNet-wide zone visibility, and actual relay registry endpoint ownership.
The [relay registry endpoint observer](azure-relay-registry-endpoint.md) now requires an explicit third consumer in the registry service contract.
It binds the selected endpoint, NIC, member addresses, subnet, and dedicated NSG while preserving unknown-consumer rejection.
Independent worker, preparation, and relay networking remains required.
