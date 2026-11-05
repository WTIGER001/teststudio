# Azure preparation writer

## Current boundary

The preparation publisher composes the existing registry observation chain, the trusted preparation path, and a selected registry writer grant.
It opens an ACR publishing client only after those checks succeed.
The [native assembly profile](azure-native-assembly.md) now uses this boundary in a separate preparation command and headless deployment template.
The publisher itself starts no queue, executes no uploaded source, and implements no execution Network.
Acceptance 51 advances. The verified total remains 48 of 57, with nine open.

## Configuration

LoadPreparationSettings reads one JSON file or one named environment variable through the shared bounded loader.
It rejects unsupported fields, invalid ownership, invalid identity identifiers, and incompatible registry settings.
The document has worker and writer fields. worker contains the existing Azure Config.
Native operation requires readerClientId, [host settings](azure-native-assembly.md#host-settings-and-observation), and [network settings](azure-preparation-network.md) for separate credentials and workload binding.

PreparationWriterSettings requires these writer fields:

| Field | Requirement |
| --- | --- |
| registry | Complete [preparation registry settings](azure-preparation-registry.md). |
| identity | Selected user-assigned managed identity in the installation subscription. It must differ from the configured worker pull identity. |
| clientId | Nonzero managed identity client identifier. |
| principalId | Nonzero service principal identifier, distinct from clientId. |
| tenantId | Nonzero tenant identifier. |
| assignment | One role assignment directly beneath the selected registry resource. Subscription or resource-group assignments reject. |

AssignmentDefinition returns the selected principal's registry-scoped AcrPush assignment body for installation review.
The caller must select the registry scope when provisioning that assignment. The method performs no provider operation.
The built-in role identifier is fixed; configuration cannot select another role.
Microsoft documents its two actions in the [AcrPush role definition](https://learn.microsoft.com/en-us/azure/role-based-access-control/built-in-roles/containers#acrpush).

## Fresh grant and credential checks

PreparationWriterGuard first refreshes the complete preceding registry and preparation path once.
After success, three concurrent GETs read the selected identity, assignment, and built-in role definition.
The identity must match its configured resource, region, client, principal, and tenant.
The assignment must select that service principal, the exact registry scope, and the fixed AcrPush role.
Conditions, delegated identities, alternate principals, and broader assignment scopes reject this contract.

The reported role must remain BuiltInRole AcrPush with root assignability and exactly the pull and push actions.
Additional actions, exclusions, data permissions, and role conditions reject.
The standard global and subscription-qualified forms of the same built-in role identifier are accepted.
Successful checks remain uncached. A changed selected grant blocks the next opening.

OpenRegistry checks the path and grant before constructing an explicit ManagedIdentityCredential for the verified client identifier.
It uses no default credential, credential chain, registry password, or observer credential for registry authentication.
The resulting client uses the configured tenant, repository prefix, publishing mode, and selected regional data redirect authority.
Existing [ACR authentication restrictions](azure-registry-authentication.md) apply to its later registry requests.
Opening a client requests no registry token and publishes no content.

## Composition and ownership

PreparationSettings.OpenPublisher requires an explicit observation credential and optionally accepts a restricted test transport.
The observation credential must have permission to read the selected resources. Its authority remains the caller's installation responsibility.
The factory reuses the existing registry chain rather than maintaining a second implementation.
Construction creates ten readers for writer-only observation, eleven with host observation, or thirteen for native network admission. It makes no provider call.
Each publisher OpenRegistry call performs fresh observation before selecting the separate writer credential.

The caller closes each returned ACR client and the publisher after use.
Publisher Close releases every owned reader once. Further openings return unavailable without provider calls.
Closing the publisher does not close previously returned registry clients; their ownership remains with the caller.
Library callers can call OpenRegistry before each attempt or use publisher Check before reusing the selected registry client.
The native process checks the source network, journal path, host, and preceding writer chain during startup and before every new build.
A retained client alone proves no continued grant validity. Host observation succeeds before selecting a publishing credential.
The registry service still enforces authorization on subsequent registry operations.

The writer stage has a 45-second total bound, including the preparation path's 40-second bound.
Its three SDK operations each retain the shared five-second context and four-megabyte response limit.
Only the selected three GET paths and API versions are permitted. Lists, mutations, alternate paths, extra queries, and redirects reject.
Provider diagnostics remain outside public errors.

## Remaining installation work

This checks one reported assignment. It does not enumerate additional, inherited, delegated, or effective permissions.
The identity's absence from other workloads, federation state, and protection against later reassignment remain unverified.
Writer-only settings exclude the worker pull identity. Native host settings also separate preparation pull, startup, reader, and writer resources.
Installation verification must still exclude unrelated controller, observer, and backend uses.
Reported private resources do not prove effective packet routing, process placement, or complete DNS visibility.
Native wiring, source and journal templates, reported placement and network checks, and an actual journal mount probe now exist.
Actual placement enforcement, source egress, filesystem behavior, and complete effective read authority remain unverified.

The native trusted writer uses [assembly-only handoff](source-assembly-handoff.md).
Discovery, review, approval, and complete untrusted execution admission remain separate requirements.
The current source service still requires compatible isolated discovery before source approval.
This publisher does not enable Azure coded execution or close deployment acceptance.

## Local verification

Three new writer checks and one publisher composition check pass with fake credentials and in-memory SDK transports.
They cover fresh reads, exact grant scope, changed authority, incomplete responses, selected credentials, configuration, SDK restrictions, and reader ownership.
One existing observer check passes after the shared registry factory extraction.
No provider request, registry publication, token request to a real identity service, listener, browser, deployment, or demo restart runs.
