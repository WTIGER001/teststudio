# Azure reported subscription read-deny inspection

## Current boundary

ReadDenyGuard checks reported subscription deny assignments before the [bound observer session](azure-read-identity.md#bound-observer-session) reads its resource chain.
The selected subscription role alone does not establish effective access. A separate deny can block an otherwise granted operation.
This stage detects reported conflicts with the observer profile's unconditional management reads and DNS-association observation.
It also requires the five selected network and zone protections to appear in the subscription response with their expected metadata.

It does not establish complete effective authorization or complete cross-subscription and cross-tenant visibility.
Hosted API coverage, actual authentication, host attachment, deployed enforcement, and native command wiring remain unverified.
The observer implements no Network and enables no Azure coded job.
Acceptance 51 advances. The total remains 48 verified of 57, with nine open.

## Why the selected grant needs a separate check

Azure evaluates deny assignments separately from grants and considers applicable inherited assignments.
The [Azure RBAC evaluation guide](https://learn.microsoft.com/en-us/azure/role-based-access-control/overview#how-azure-rbac-determines-if-a-user-has-access-to-a-resource) explains that sequence.
The resource and resource-group permissions endpoints report caller permissions at their selected scopes.
Those endpoints alone do not establish the subscription-wide read coverage needed by the existing workload and private-zone inventories.
See the [resource permissions API](https://learn.microsoft.com/en-us/rest/api/authorization/permissions/list-for-resource?view=rest-authorization-2022-04-01).

The [deny-assignment guide](https://learn.microsoft.com/en-us/azure/role-based-access-control/deny-assignments) documents subscription lists and inherited scope behavior.
It also distinguishes direct principals, groups, exclusions, and all-principal assignments.
This implementation uses the already-pinned preview SDK to retain explicit enforcement and condition metadata.
Successful local responses do not prove hosted support for that API or its complete view.

## Fixed scan and request bounds

SDKReadDenyReader requires an explicit credential and lists only this path:

```text
/subscriptions/{selected-subscription}/providers/Microsoft.Authorization/denyAssignments
```

It pins API 2024-07-01-preview through the existing armauthorization/v3 v3.0.0-beta.3 dependency.
It sends no principal, atScope, tenant, or other filter that can narrow the subscription response.
Pagination permits only the same host, selected path, API version, and bounded skip token.
Foreign links, repeated tokens, alternate versions, mutations, and extra query parameters reject ownership.
Retries and provider registration remain disabled.

One collection has a five-second context, at most 16 pages, and at most 1024 assignments.
Each response retains the shared four-megabyte limit. Missing or null value arrays remain unavailable.
Explicitly empty arrays cannot satisfy the five required selected protection assignments.
The reported scan uses the same bound observer credential as the rest of the session.

The composed guard has a 130-second context and at most 2212 requests, including the preceding chain's bounded inventories.
Normal additional cost is one GET. The scan and its successful observations are not cached.
Binding still performs its separate 125-second bootstrap inspection before constructing the 21-reader session.
The bootstrap guard itself does not perform this new scan. Each bound Check performs it before refreshing the existing identity and resource chain.

## Scope and principal contract

Each assignment must report its identifier, GUID name, type, scope, explicit enforcement effect, child-scope setting, principals, and permissions.
The identifier must name that exact scope and GUID. Duplicate identifiers, foreign subscription scopes, and malformed identities are rejected.
Supported scopes are the selected subscription and its descendants, tenant root, and reported management-group ancestors.
Ancestor assignments that explicitly exclude child scopes do not apply to this subscription.
Audit assignments do not establish an enforced read block. Missing or unknown effect remains unavailable.

SystemDefined with the zero GUID represents all principals. The selected ServicePrincipal also matches directly.
Other direct users, devices, and service principals do not match the observer.
Group and ForeignGroup entries remain potentially applicable. This stage makes no Graph membership request.
A direct exclusion of the selected service principal removes that assignment from consideration.
A group exclusion does not prove that the observer is excluded. Unknown principal metadata remains unavailable.
This conservative group handling can block a deployment that a complete directory-membership evaluation could permit.

## Management action contract

The observer profile requires unconditional management read coverage throughout the selected subscription.
The check therefore considers reported read conflicts outside the selected resource groups too.
A narrower resource placement cannot substitute for that coverage while subscription-wide inventories remain required.

Action matching treats the star as a case-insensitive wildcard that can cross slash boundaries.
Patterns that can match an action ending in /read conflict unless a supported NotActions exception removes that conflict.
The separate Microsoft.Network/virtualNetworks/listDnsResolverPolicies/action must also remain outside the denied set.
The existing fixed protection profile preserves both operations and passes this scan.
Write-only and delete-only action patterns do not conflict with these observer operations.
Data-only denials do not block management reads in this check.

A literal star or */read exception removes the corresponding general read conflict.
Identical action exceptions and matching exceptions for literal actions also remove their corresponding conflicts.
For other overlapping wildcard exclusions, this stage can conservatively reject coverage rather than claim a general policy-subset proof.
Unknown patterns, missing permission state, or unsupported metadata remain unavailable.
Conditions on a potentially blocking assignment or permission remain unavailable. The scanner does not assume those conditions are false.
Unconditional potentially applicable conflicts reject ownership through the fixed public error category.
Provider messages, condition contents, and principal metadata are not included in returned errors.

## Targeted local verification

Three new focused race checks and two changed observer-session checks pass in 2.716 seconds.
They verify inherited scopes, direct and uncertain group applicability, exclusions, conditions, action wildcards, read exceptions, and missing enforcement metadata.
They also verify same-scope pagination, missing arrays, null items, foreign pages, alternate APIs, page exhaustion, redirects, and explicit credentials.
Missing selected protection and reported read conflicts prevent later parent reads.
Successful checks refresh both the scan and parent chain. Runtime mismatches reject before requests.
The observer integration uses its distinct test token for the scan and closes all 21 readers.
No unchanged broad suite repeats.
Fixtures use fake credentials and in-memory SDK transports. They open no network connection or service listener.
No real token, cloud operation, browser, UI build, image build, container, registry push, deployment, or demo restart runs.
External requests read only public Microsoft documentation.

```sh
go test -race ./adapters/execution/azure -run '^(TestAzureReadDeny.*|TestAzureObserver.*)$' -count=1
```

Evidence is output/azure-read-deny/summary.json and the targeted command results.
Continue hosted visibility and enforcement verification, observer host attachment, protection-management authority, and complete native admission.
Keep existing cross-scope DNS visibility, clean installation provenance, environment attachment controls, preparation networking, and execution isolation in that scope.
