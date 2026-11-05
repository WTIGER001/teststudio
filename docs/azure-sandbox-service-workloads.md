# Azure sandbox service workload identity binding

## Introduction

OpenRuntime includes SDKServiceWorkloads in creation, source import, and live admission.
It verifies reported relay and backend environment occupancy and service identity placement across Container Apps and jobs in the selected subscription.
It also checks the four selected identity resources and requires empty federated-credential inventories.
The stage derives its bindings from existing service settings. It adds no configuration field or deployment resource.

The shared ServiceWorkloadBinding reuses existing placement, identity projection, lifecycle, and identity-sharing checks.
The observer's existing sharing check uses the same pure identity-host verifier.
Ordinary Jobs execution remains disabled.

## Concepts

### Selected services and identity resources

The existing service settings define these expected bindings:

| Host | Identity | Lifecycle |
| --- | --- | --- |
| Relay | Relay image pull | None |
| Backend | Backend image pull | None |
| Backend | Backend startup | None |
| Backend | Backend runtime | Main |

Each identity resource must report its selected identifier, type, region, tenant, client, and principal.
Missing identifiers remain unavailable. Mismatched identifiers or tenants reject admission.
Each federatedIdentityCredentials collection must report one complete empty page.
Any returned credential rejects admission. Missing values, inaccessible lists, or continuation leave absence unproven.
The reader follows no continuation for these required empty collections.

### Reported environment occupancy

The reader obtains complete subscription inventories for Container Apps and jobs.
Both selected applications must appear in their configured private environments.
Current and legacy environment fields must agree when both are present.
Another app or any job in either selected environment rejects admission.
Duplicate, foreign, malformed, missing, or incomplete resource observations cannot establish occupancy.

The selected applications' inventory entries must retain their exact service identity sets, pins, and lifecycle settings.
The existing selected service stage still verifies full trusted templates, private environments, authentication, certificate metadata, and revisions.
The two stages share the same expected identity maps.

### Reported identity sharing

Every reported app and job must provide enough identity metadata to evaluate sharing.
An explicitly reported None identity is permitted for unrelated workloads.
Missing or unknown identity type leaves the inventory unavailable.
User-assigned references and any reported client or principal identifiers must be valid.

A selected service identity may appear only on its configured application.
Another resource with the same identity identifier, client identifier, or principal identifier rejects admission.
A system-assigned principal equal to a selected service principal also rejects admission.
Unrelated workloads in unrelated environments remain permitted when their reported identities establish no sharing.

This checks reported Container Apps and jobs in one subscription.
It does not establish exclusive identity use across other subscriptions, other host types, or resources hidden by deny effects.
Actual token issuance, effective permissions, identity write protection, and administrative control remain separate requirements.
Controller, registry-inspection, metadata, source-import, and timer host attachment remain unfinished.

### Bounds, cache, and shutdown

Two subscription collection GETs use API version 2025-07-01.
Four selected identity GETs and four federated-credential collection GETs use API version 2024-11-30.
The explicit metadata identity supplies all ten selected paths.
Construction obtains no token and calls no service.

Each workload collection permits at most 512 items and 16 pages.
The complete observation has a five-second bound and joins all reads before returning.
Each response retains the existing 1 MiB limit and scoped continuation rules.
Unsafe continuations, redirects, automatic retries, request mutations, and provider diagnostic exposure remain blocked.
Large or slow inventories remain unavailable when they exceed these bounds.

Creation and source import obtain fresh observations.
Concurrent live checks share successful evidence for at most 15 seconds from observation start.
Current failures invalidate prior success. Failures remain uncached.
Remaining protection, authorization, and sandbox-definition checks remain uncached.
Metadata gaps never establish sandbox absence or authorize another execution.
Runtime.Close closes the new reader after joining owned command work.

## Tutorials

### Verify the changed path locally

```sh
go test -race ./adapters/execution/azuresandbox -run '^TestSandboxServiceWorkload|^TestSandboxRuntimeServiceWorkloads' -count=1 -timeout=30s
```

Four new checks cover allowed unrelated workloads, sharing and occupancy drift, complete inventories, fresh failure, caching, and runtime admission.
Six affected native checks and three existing observer-placement checks also pass with race detection.
Evidence is output/azure-sandbox-service-workloads/summary.json, native.log, and shared.log.
Checks use local SQLite, fake credentials, and in-memory ARM responses.
No cloud service, browser, container, UI build, demo restart, resource mutation, or remote push is involved.

### Continue activation

Bind remaining trusted registry consumers and controller identities to their actual workloads and network paths.
Complete effective grants, deny effects, protected writes, native command role wiring, and deployment packaging.
Keep deployed token behavior, routing, private conversion, preview compatibility, and independent timer execution in separate deployment acceptance.

Related guides: [runtime](azure-sandbox-runtime.md), [trusted services](azure-sandbox-services.md), and [registry service](azure-sandbox-registry-service.md).
Acceptance 51 advances. The total remains 48 verified of 57, with nine open.
