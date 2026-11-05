# Azure observer host attachment

## Current boundary

ObserverHostGuard checks the selected controller app's reported observer attachment, identity lifecycle, image, placement, and current revision.
It follows the [reported subscription read-deny check](azure-read-deny.md) through the same bound observer credential.
Its existing [workload inventory](azure-worker-workloads.md#observer-identity-placement) now rejects reported sharing on another Container App or job without another scan.
Its identity reader also requires [federated-credential absence](azure-read-identity.md#federated-credential-absence) during bootstrap and each bound inspection.
It does not establish exclusive identity use, complete trusted API configuration, actual host token issuance, or complete effective authorization.
Those requirements remain part of native admission and separate hosted verification.
No Network is implemented. Azure coded jobs remain disabled.
Acceptance 51 and supporting packaging under 57 advance. The total remains 48 verified of 57, with nine open.

## Host settings

ObserverHostSettings requires these fields:

| Field | Requirement |
| --- | --- |
| identity | Complete preceding ReadIdentitySettings. |
| application | Selected API Container App resource identifier in the configured subscription. |
| image | Exact trusted service image with an immutable registry digest. It cannot use the untrusted test repository prefix. |
| pullIdentity | Separate core service image-pull identity resource identifier. |
| startupIdentity | Separate core API startup identity resource identifier. |
| runtimeIdentity | Separate core API Blob runtime identity resource identifier. |

The expected environment is Config.controllerEnvironment. It is not an independent host override.
The host cannot substitute the selected relay or trusted worker backend app.
Its three core identities must differ from the observer, worker pull, and three backend identities.
They must also differ from each other.
Binding checks the complete preceding configuration before bootstrap reads or credential selection.

The observed app must report the selected resource identity, region, successful provisioning, and running or ready state.
Its current and legacy environment fields must agree with the selected controller environment.
The app must use UserAssigned identity type without a system-assigned principal or tenant identity.
It must report exactly the four selected user-assigned identities, with complete and distinct client and principal metadata.
The observer entry must match the client and principal verified by the preceding identity stage.

## Lifecycle and revision checks

The four lifecycle entries must be explicit and unique:

| Identity | Lifecycle |
| --- | --- |
| Core image pull | None |
| Core API startup | None |
| Core API runtime | Main |
| Observer | Main |

The [Container Apps identity guide](https://learn.microsoft.com/en-us/azure/container-apps/managed-identity#control-managed-identity-availability) explains these lifecycle controls.
The host uses one main container, no init containers, and the selected service image.
It uses Single revision mode. The shared revision verifier checks the latest ready revision, health, activity, replicas, traffic, and template agreement.
Inactive revisions require explicit zero traffic and zero replicas.
Sidecars, extra identities, substituted metadata, missing lifecycle state, stale image references, and conflicting environments block the check.
These observations do not certify every app argument, environment value, grant, or administrative authority.
Identity changes require fresh app observations; an old revision identifier alone is insufficient.

## Bound session and request cost

ReadIdentityGuard.BindHostObserver accepts the explicit host settings and builds all 22 SDK readers with the verified observer credential.
It retains the existing single bootstrap inspection and performs no observer request during client construction.
Each Check refreshes the existing chain once, then reads the selected app and its revisions concurrently.
The workload stage first requires the selected host in the application collection and blocks shared or unknown observer identity placement.
BindObserver remains available for the preceding 21-reader diagnostic scope. It does not check host attachment.
Use the host-bound method when this attachment observation is required. Neither method enables native dispatch.

SDKObserverHostReader permits GET on only the selected app and its revisions collection.
It reuses the existing Container Apps SDK and API 2025-07-01 with no new dependency.
The shared service reader supplies bounded revision paging. Its environment, route, auth, and secret paths are outside this reader's allowlist.
Each operation has a five-second context and four-megabyte response limit. Revisions permit at most four records across three pages.
Retries and provider registration remain disabled. Unsafe pages, redirects, and incomplete revisions remain blocking errors.

The composed host check has a 135-second context and at most 2217 requests, including the identity reader's federation-absence GET.
The host-specific additional cost remains two GETs. Initial binding retains its separate 125-second bootstrap context and 2197-request ceiling.
Successful observations are not cached. Closing the session releases all 22 readers and prevents later checks.
Returned errors omit provider messages and private app configuration.

## Optional core packaging

The [Azure core package](azure.md) adds observerIdentityEnabled, which defaults to false.
When enabled, the package declares a sixth user-assigned identity and a subscription-level custom role and assignment.
The role permits exactly management reads and the existing DNS-association observation action, with no exclusions, data actions, conditions, or role-management authority.
The API main container receives the observer identity. Its pull, startup, and Blob runtime identities retain their existing roles and lifecycle controls.
The shared app module also gates the attachment on the API role. An artifact-role caller cannot expose it through that parameter.
The artifact app receives no observer identity, subscription read grant, or new application setting from the top-level package.

The observerBinding output contains resource identifiers, client, principal, tenant, role, assignment, selected host, image, and core identity references.
It contains no secret value or token. Combine those values with the complete installation network settings for the Go observer configuration.
The output is not a complete execution profile or admission certificate.
Enabling the option prepares identity resources only. Native observer startup and coded job dispatch remain unwired.
Provisioning requires subscription-level role-definition and role-assignment authority in addition to the existing core deployment permissions.
These templates are compiled locally only. No resource, role, or deployment is created in this work session.

## Targeted local verification

Three new focused race checks and one selected existing observer-session regression pass in 2.456 seconds.
They cover fresh host metadata, identity exposure, configuration binding, placement, revision state, exact requests, parent gating, and all-reader cleanup.
The initial fixture literal is corrected before those checks pass.
Fixtures use fake credentials and in-memory SDK transports. They open no network connection or listener and request no real token.

```sh
go test -race ./adapters/execution/azure -run '^(TestAzureObserverHost.*|TestAzureObserverBindsAllClientsWithoutBootstrapReuseAndClosesReaders)$' -count=1
```

The Bicep package compiles with the existing pinned compiler and no warnings or registry restore.
Its local contract check verifies the optional grant and API-only attachment with the existing identity and storage boundaries.
The checker can reuse a current compiled template through --compiled-template. This avoids another compile when only contract assertions change.
Review adds an API-role guard inside the shared app module. That semantic change receives one final compile and package check.
No unchanged Go suite repeats after the packaging-only correction.
Evidence is output/azure-observer-host/summary.json, final-package/service.json, final-package/compile.log, and final-package/summary.json.
No cloud operation, UI build, browser, image build, container, registry push, deployment, or demo restart runs.
External requests read only public Microsoft documentation.

Continue exclusive identity placement, complete trusted host configuration, resource-management authority, and full native admission wiring.
Keep deployed token behavior, effective coverage, protection enforcement, and cross-scope visibility in the separate hosted verification phase.
