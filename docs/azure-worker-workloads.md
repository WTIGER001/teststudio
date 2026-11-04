# Azure exchange workload inventory

## Current boundary

WorkloadInventoryGuard checks reported subscription-wide application and job placement after [selected exchange binding](azure-worker-exchange.md) succeeds.
The selected relay must appear in its exchange environment. No other observed app or job can share that environment.
The reader uses the documented subscription collection APIs. It does not substitute resource-group discovery for subscription inventory.

This is an observed inventory stage. It does not implement Network, authorize dispatch, or release worker input.
A completed API list does not prove the caller's full read coverage or restrictions on future attachments.
Effective read coverage, join and write authority, network isolation, and backend authority remain separate requirements.
Azure coded execution stays disabled in native commands and infrastructure packaging.

## Inventory scope and placement

The guard preserves the exact ExchangeSettings and Config used by the preceding stage.
It refreshes those prior checks before requesting the two inventories concurrently.
No additional tester configuration, application selector, or account discovery is introduced.
The operator's configured subscription remains the only collection scope.

The application inventory must contain exactly the selected relay in the selected exchange environment.
Other applications in that environment reject observation, including stopped apps and apps owned by another resource group.
A job attached to the exchange environment also rejects observation, whether or not it has started.
The exchange environment is a service boundary, rather than a general test execution pool.

Unrelated apps and jobs can occupy other environments in the selected subscription.
Legacy app environment references and connected environments remain readable.
When both app environment fields exist, they must agree.
Resource groups can use their supported Unicode names. Resource identity comparison ignores letter case.
Duplicate keys use the same Unicode case-fold equivalence as identity comparison.
Duplicate identities, different resource kinds, contradictory names, foreign-subscription records, and conflicting environment aliases reject ownership.

Missing properties, missing placement, unknown environment references, and unsupported resource identity shapes leave observation unavailable.
A missing selected relay also remains unavailable.
A selected relay observed in a different environment rejects the intended binding.
A list item cannot hide an unknown placement by appearing outside the selected resource group.

## Minimal retained data

SDKWorkloadInventoryReader reduces each page immediately to WorkloadPlacement:

| Field | Meaning |
| --- | --- |
| resourceId | The observed application or job resource identifier. |
| environmentId | Its resolved managed or connected environment identifier. |

Only those two fields leave the reader.
Application configuration, job templates, environment values, secret references, identities, and unrelated settings do not enter the returned inventory.
This stage persists no subscription inventory or application configuration.
The SDK still receives the provider's list response while decoding each page. The adapter cannot make that server response omit configuration fields.
It calls no secret-list, token, credential, export, or mutation API.

The selected application's full configuration remains the preceding exchange stage's responsibility.
This inventory stage checks placement. It does not replace app, revision, certificate, or backend verification.

## Completeness, permissions, and attachment authority

A complete reported list requires every page within the configured bounds.
An explicit empty jobs array can establish an empty reported job list.
A missing or null value array, nil item, unfinished page, or exceeded bound cannot establish absence.
The adapter returns unavailable instead of accepting a truncated or uncertain view.

The controller needs subscription-wide Microsoft.App/containerApps/read and Microsoft.App/jobs/read authority for this inventory scope.
The SDK requires an explicit TokenCredential and makes no permission grant or implicit credential lookup.
Resource-group-only access cannot establish complete subscription coverage.
Permission-denied responses remain unavailable. The adapter never falls back to a smaller scope after a failed subscription read.

A 200 response alone does not establish effective permission coverage across the subscription.
Native setup must verify that scope, including relevant conditional or denied access, before using these observations as complete admission evidence.
The current local fixtures verify the reader contract. They do not verify a deployed principal's authorization.

Inventory is also a point-in-time observation.
Installation controls must restrict environment join and workload write authority to the intended trusted infrastructure actors.
Otherwise an actor could attach a workload after a successful observation.
Do not treat an empty list as proof of those controls.
The current batch verifies no effective role assignment, deny assignment, join restriction, or native attachment policy.
Those conditions remain open before composing Network.

## SDK and transport boundary

The reader uses the existing armappcontainers/v4 4.0.0 SDK and API 2025-07-01.
Its HTTP transport permits GET on exactly these two paths:

```text
/subscriptions/SUBSCRIPTION/providers/Microsoft.App/containerApps
/subscriptions/SUBSCRIPTION/providers/Microsoft.App/jobs
```

The shared transport now handles these subscription paths without assuming a resource-group segment.
Existing escaped resource-group reads and DNS association behavior retain targeted passing evidence.
The change does not expand any existing reader's resource allowlist.

Individual resource reads, resource-group collections, other subscriptions, other providers, filters, field selection, secret actions, and mutations reject before transport.
Redirects, unsafe route encoding, altered versions, extra query keys, and unrelated continuation paths also reject inspection.
Automatic retries and provider registration remain disabled.
Provider diagnostics become fixed control errors without response bodies.

Each collection permits at most 512 items and sixteen pages.
An item limit is an uncertainty bound. It never permits partial absence claims or silently removes excess resources.
The two collection operations each have a five-second context and use the existing four-megabyte response ceiling per page.
Only placement records accumulate across pages; full SDK models do not accumulate in the returned result.
The shared pager rejects foreign origins, other collections, foreign subscriptions, repeated tokens, and unsafe queries.
Unfinished bounded paging remains unavailable.

A normal full check makes forty-five reads, including the preceding forty-three observations.
The full paging bound is ninety-four reads. The guard has a forty-second context.
Successful observations are not cached.
Close the reader to close idle transport connections.

## Targeted local verification

Six new focused race checks cover fresh observations, cross-resource-group occupants, missing placement, duplicate identity, item bounds, paging, and SDK scope.
They verify explicit empty jobs, Unicode resource groups, legacy connected placement, and immediate reduction of unrelated configuration.
Missing arrays, null lists, nil rows, access failure, redirect, oversized response, and endless paging remain unavailable.
Foreign continuations, changed collection paths, extra occupants, conflicting references, and duplicate IDs reject observation.
Two selected existing private-DNS and DNS association checks pass after the shared transport path adjustment.
Only two changed checks repeat after the Unicode duplicate-identity review fix.

```sh
go test -race ./adapters/execution/azure -run '^TestAzureWorkloadInventory|^TestAzurePrivateDNSSDKScopesSelectedZoneInventoriesAndEscapedGroups$|^TestAzureDNSSDKPinsReadInventoryActionAndEscapedResources$' -count=1
```

Evidence is output/azure-worker-workloads/summary.json and the targeted command results.
No cloud credential, live cloud API, DNS probe, packet probe, TLS probe, deployment, registry push, UI build, browser, container, or image build runs.
No unchanged broad suite or runner fixture repeats.
Acceptance 51 advances. The total remains 48 verified items out of 57.

## Next implementation

Verify effective inventory read coverage and environment attachment authority in the native installation boundary.
Inspect actual exchange VNet, subnet, routing, and restricted worker backend placement.
Complete effective DNS visibility, trusted preparation networking, cross-job isolation, execution placement, identity roles, and complete profile binding.
Then compose Network and add native service wiring before separate deployment acceptance.

## Primary references

[Container Apps operations](https://learn.microsoft.com/en-us/rest/api/resource-manager/containerapps/container-apps?view=rest-resource-manager-containerapps-2025-07-01) documents subscription and resource-group inventory scopes.
[Jobs subscription inventory](https://learn.microsoft.com/en-us/rest/api/resource-manager/containerapps/jobs/list-by-subscription?view=rest-resource-manager-containerapps-2025-07-01) documents the jobs collection and environment reference.
[Container roles](https://learn.microsoft.com/azure/role-based-access-control/built-in-roles/containers) documents environment join authority and workload permissions.
The pinned SDK defines the actual list operations and observed placement fields.
