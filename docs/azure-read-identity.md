# Azure observer identity and selected read grant

## Current boundary

ReadIdentityGuard binds one observer identity to its reported principal, tenant, client identifier, subscription assignment, and custom role.
It follows the [selected network and DNS deny observer](azure-network-deny.md). Each inspection refreshes that complete preceding chain once.
The observer identity remains separate from worker pull, backend pull, backend startup, and backend runtime identities.

This stage implements no Network and enables no native Azure coded execution.
It observes the selected grant. It does not prove effective authorization, complete visibility, or absence of other grants.
The initial guard requires an explicit bootstrap credential. A bound observer session then uses the verified observer credential throughout its SDK chain.
Binding does not prove that the bootstrap credential itself uses the observer identity.
The [reported read-deny scan](azure-read-deny.md) now gates each bound inspection before its existing resource chain.
The [host-bound observer](azure-observer-host.md) additionally checks the selected controller app's reported identity exposure and current revision.
Its workload stage now rejects [reported app and job identity sharing](azure-worker-workloads.md#observer-identity-placement) while retaining the same collection requests.
The identity stage now requires an explicit empty federated-credential collection before constructing a credential or completing a bound inspection.
Identity attachment, token availability, deny coverage, protection-management authority, and cross-scope visibility remain installation requirements.
Acceptance 51 advances. The total remains 48 verified of 57, with nine open.

## Explicit settings

ReadIdentitySettings requires these fields:

| Field | Requirement |
| --- | --- |
| protection | Complete preceding NetworkDenySettings. |
| identity | Selected user-assigned identity resource identifier in the configured subscription. |
| clientId | Canonical, nonzero client GUID. |
| principalId | Canonical, nonzero service principal GUID, distinct from clientId. |
| tenantId | Canonical, nonzero tenant GUID. |
| assignment | Selected subscription-root role assignment identifier with a GUID name. |
| role | Selected subscription-root role definition identifier with a GUID name. |

The identity resource must differ from the four known pull and backend identity resources.
Its observed name, resource type, location, identifier, client, principal, and tenant must match the selected configuration.
The assignment must report that principal, ServicePrincipal type, selected role, and exact subscription scope.
Assignment conditions, condition versions, and delegated managed identity references are rejected.
Missing required identity or grant metadata remains unavailable. A conflicting identity or policy rejects ownership.

## Fixed selected role

The role must report CustomRole type and exactly the selected subscription as its assignable scope.
One permission entry permits exactly these actions:

- */read
- Microsoft.Network/virtualNetworks/listDnsResolverPolicies/action

The second action supports the existing bodyless DNS-association observation.
The role has no exclusions, data permissions, permission conditions, write actions, join actions, or role-management actions.
Action order and letter case do not change semantics. Duplicate actions reject observation.
The cosmetic role name does not determine authority.

RoleDefinition and AssignmentDefinition return fresh SDK values for installation tooling.
They perform no creation, assignment, update, or other cloud operation.
The assignment definition omits response-only scope metadata. Installation tooling uses the selected assignment identifier to select its scope.
Microsoft documents [role definitions](https://learn.microsoft.com/en-us/azure/role-based-access-control/role-definitions) and [role assignments](https://learn.microsoft.com/en-us/azure/role-based-access-control/role-assignments).

## Credential and SDK boundary

Credential performs one complete inspection and constructs ManagedIdentityCredential with the verified, observed client identifier.
It provides no default-credential fallback. This method does not request or redeem a token.
SDK calls made later with that credential can request tokens. Host identity attachment and successful authentication require separate verification.
The [managed identity credential API](https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/azidentity#ManagedIdentityCredentialOptions) defines explicit user-assigned identity selection.

SDKReadIdentityReader requires an explicit credential and permits only four selected GET paths.
The identity reader pins armmsi v1.3.0 and API 2024-11-30.
The existing armauthorization/v3 v3.0.0-beta.3 supplies role assignment API 2022-04-01 and role definition API 2022-05-01-preview.
The scoped role Get method preserves the exact query. GetByID adds an SDK disambiguation query outside this transport contract.
The shared transport rejects mutations, other collections, alternate resources, extra query parameters, secret operations, and redirects.
Provider registration and retries remain disabled. Each selected read has a five-second context and four-megabyte response limit.

Four selected reads run concurrently after protection succeeds: identity, assignment, role, and federated-credential absence.
Parent observations do not repeat within one inspection.
The composed guard has a 125-second context and at most 2197 requests, including the bounded reported-subscription scan.
Normal additional cost is exactly four GETs. Successful observations are not cached or persisted.
Credential construction uses the identity from that same inspection.
Returned errors use fixed public categories and omit provider messages, condition text, and credential factory details.

## Federated credential absence

The observer profile uses explicit managed identity authentication and accepts no federated identity credential under its selected user-assigned identity.
The pinned armmsi v1.3.0 client reads only this additional collection path with API 2024-11-30:

```text
SELECTED_IDENTITY/federatedIdentityCredentials
```

The [Microsoft list API](https://learn.microsoft.com/en-us/rest/api/managedidentity/federated-identity-credentials/list?view=rest-managedidentity-2024-11-30) defines that identity-scoped collection.
The reader sends no top, skip token, tenant, or filter parameter. It permits no child credential GET, mutation, or other identity collection.
It fetches exactly one page with a five-second context and the shared four-megabyte response limit.
A successful response must contain an explicit empty value array and no nonempty nextLink.
Any non-null credential object rejects ownership, regardless of issuer, subject, audience, or other metadata.
The reader does not evaluate an allowlist or assume that a reported credential is unusable.
Missing or null arrays, null-only entries, and any continuation remain unavailable.
Continuations are never followed, even when they name the same collection. An incomplete empty page cannot prove absence.
Provider errors, denied reads, redirects, and canceled requests remain blocking errors through fixed public categories.

ReadIdentityReader requires FederationAbsent as part of its contract. There is no optional missing-method or skipped-observation fallback.
The guard checks absence alongside its identity and selected grant reads after the existing protection stage succeeds.
Credential construction therefore requires that observation too. Later bound checks refresh it through the selected observer credential.
The reader returns no issuer, subject, audience, credential name, response body, or federated configuration to the caller.
Successful absence is not cached or persisted. It does not prevent a later administrative write or establish complete identity exclusivity.
Future credential-write authority and protection of the identity resource remain separate installation requirements.

## Bound observer session

ReadIdentityGuard.BindObserver performs one complete bootstrap inspection and obtains a credential for the verified observed client identifier.
It rebuilds all 21 SDK readers with that same credential and preserves their canonical settings and existing transport restrictions.
The rebuilt chain includes subscription read-deny inventory, topology, worker networking, endpoints, registry, exchange, workload inventory, backend, relay networking, DNS inventory, protection, and identity.
The bootstrap readers and credential remain outside the returned session.

Binding creates clients without requesting an observer token or performing an observer SDK request.
Observer.Check performs the complete fresh inspection through the selected observer credential.
Later checks reuse the SDK clients and their normal token cache. They do not reuse successful resource observations or repeat bootstrap inspection.
The final identity and assignment reads therefore also use the selected observer credential.
Changed immutable runtime settings reject before an SDK request. Missing identity metadata and rejected grants remain blocking errors.
Token acquisition and transport errors remain unavailable and omit private provider details.
There is no alternate credential or bootstrap fallback after binding.

Each Check now has a 130-second context and 2213-request ceiling, including the bounded subscription read-deny scan.
Binding adds one separate bootstrap inspection with a 125-second context and 2197-request ceiling.
Bind once per owned session. Do not bind again before every Check.
Close the session after its final inspection. Close waits for active checks, closes owned readers in reverse order, and releases their references.
Repeated Close calls are safe. Checks after Close return unavailable without a credential or transport request.
If client construction fails, binding closes the readers that it already creates.
The caller retains ownership of the bootstrap readers and closes them separately.

```go
observer, err := guard.BindObserver(ctx, config, nil)
if err != nil {
    return err
}
defer observer.Close()
return observer.Check(ctx, config)
```

The optional transport parameter supports trusted installation integration and local protocol fixtures.
Production uses nil for the existing restricted HTTPS transport. Fixture tests supply an in-memory transporter and a private credential factory spy.
The public binding API accepts no credential override. Token requests occur only through later SDK calls.
BindHostObserver extends the session to 22 readers and checks selected host attachment after the existing chain succeeds.
Its bounds and explicit host configuration appear in the [host attachment guide](azure-observer-host.md).
Observer implements no Network. Native commands still require complete admission and installation wiring before coded jobs can run.
Passing these requests does not establish complete effective authorization or visibility across subscriptions and tenants.

## Targeted local verification of identity

Five new focused race checks and one selected existing protection regression pass in 4.003 seconds.
They cover fresh grants, parent gating, changed principals, foreign scopes, conditions, extra authority, exact SDK paths, unknown metadata, and settings isolation.
A private credential factory spy checks the verified client selection without constructing or requesting a real managed identity token.
The initial targeted command exposes the GetByID query mismatch. The scoped Get correction preserves the transport boundary and passes the same checks.
No unchanged broad suite repeats.
Fixtures use explicit fake credentials and in-memory SDK transports. They open no network connection or listener.
No cloud request, UI build, browser, image build, container, registry push, deployment, or demo restart runs.

```sh
go test -race ./adapters/execution/azure -run '^(TestAzureReadIdentity.*|TestAzureNetworkDenyRequiresFreshEnforcedProtectionAfterInventory)$' -count=1
```

Evidence is output/azure-read-identity/summary.json and the targeted command results.
The bound session now connects this credential to actual inspection clients. Effective access remains a separate requirement.
Keep clean creation provenance, existing cross-scope visibility, deployed protection, attachment controls, preparation networking, and execution isolation in the native admission scope.

## Targeted local verification of session binding

Two new focused race checks pass in 2.325 seconds. One selected existing credential check also runs without failure in the initial targeted command.
The initial new helper expects the wrong token audience. It now reuses the existing fixture credential's exact audience validation.
Only the two changed checks repeat after that fixture correction. No unchanged broad suite repeats.
The complete chain receives a distinct observer test token. The bootstrap capture records selected routes without storing credentials or freezing response bodies.
Later SDK requests reach those original mutable fixtures through the observer transporter, with the distinct token checked before fixture adaptation.
Two successful inspections reuse no bootstrap reads. A changed assignment then fails a fresh third inspection.
Checks also cover client construction without observer requests, frozen settings, runtime mismatch, nil credentials, token failure, transport failure, cancellation, and reader cleanup.
All owned readers close. The read-deny extension increases the count from 20 to 21 and updates these targeted session checks.
Repeated cleanup and checks after closure produce no new request.
No real token, network connection, service listener, cloud operation, browser, image build, container, registry push, deployment, or demo restart runs.

```sh
go test -race ./adapters/execution/azure -run '^TestAzureObserver' -count=1
```

Evidence is output/azure-observer/summary.json and the targeted command results.

## Targeted local verification of federation absence

Three new focused race checks and four changed identity, coverage, and session checks pass in 3.179 seconds.
They cover reported credentials, missing or null state, forbidden reads, continuations, exact SDK scope, canceled requests, freshness, and factory gating.
The complete host-bound check rejects a newly reported credential, accepts a later explicit empty response, and rejects a later incomplete response.
Host reads remain blocked on failed absence. Bootstrap inspection does not repeat during those bound checks.
Only counters affected by the additional GET change in the existing checks. No broad suite repeats.
The method reuses the identity reader and SDK dependency. No additional session reader, role grant, API version, or packaging build is needed.
Fixtures use fake credentials and in-memory SDK transports. They open no network connection or listener and request no real token.
No cloud operation, UI build, browser, image build, container, registry push, deployment, or demo restart runs.
External requests read only public Microsoft documentation.

```sh
go test -race ./adapters/execution/azure -run '^(TestAzureReadIdentityFederation.*|TestAzureReadIdentityBindsFreshPrincipalAndSubscriptionGrant|TestAzureReadIdentityCredentialFactoryPinsVerifiedClientWithoutTokenRequest|TestAzureReadDenyGatesFreshParentAndRejectsMissingSelectedProtection|TestAzureObserverBindsAllClientsWithoutBootstrapReuseAndClosesReaders)$' -count=1
```

Evidence is output/azure-observer-federation/summary.json and the targeted command results.
Other identity consumer types, future attachment and credential-write authority, full trusted host configuration, and complete native admission remain required.
