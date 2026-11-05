# Azure sandbox runtime assembly

## Introduction

Settings and OpenRuntime now assemble the concrete sandbox provider dependencies.
They use the existing ACR client, sandbox data-plane client, Logic Apps timer SDK, image manager, and provider bridge.
The settings have no ordinary Job environment fields.
Execution and discovery use explicit runtime modes.

This is a library integration. The teststudio commands do not load these settings yet.
The [placement stage](azure-sandbox-placement.md) now verifies reported group, subnet, import identity, and selected pull-grant metadata.
Remaining authority and protected network admission, effective timer authority, command role wiring, and deployment packaging remain unfinished.
The factory requires admission dependencies. It supplies no permissive default.
Do not treat client construction or local protocol checks as deployment acceptance.

## Concepts

### Settings

LoadSettings accepts one file or one environment variable reference through the standard configuration loader.
It limits the document to 64 KiB and rejects unsupported fields, trailing JSON values, and incompatible configuration sources.
No reference returns no sandbox settings. Local startup behavior stays unchanged.
The loader validates all fields before credential construction.

| Field | Contract |
| --- | --- |
| worker | The existing sandbox Config: subscription, resource group, region, group, registry namespace, exchange origin, VNet connection, virtualization type, and network profile. |
| mode | Required execution or discovery. No combined native mode or implicit default. |
| tenantId | Nonzero installation tenant identifier for the registry exchange. |
| controlClientId | Explicit nonzero managed identity client identifier for sandbox control and timer administration. |
| registryClientId | Separate explicit nonzero client identifier for read-only registry inspection. It differs from controlClientId. |
| metadataClientId | Third explicit nonzero client identifier for selected ARM metadata reads. It differs from control, registry, and import clients. |
| placement | Required group-network and selected import-grant bindings described in the placement guide. |
| pullIdentity | Exact user-assigned identity resource used by the sandbox service for source conversion. |
| blobRedirectOrigins | Optional exact HTTPS registry-data or Azure Blob origins accepted by the existing registry policy. |
| timer.resourceGroup | Separate resource group for independent stop workflows. It differs from the sandbox group resource group. |
| timer.namePrefix | Validated installation prefix used to derive each owned timer name. |
| timer.stopIdentity | User-assigned timer identity in the selected subscription. Its resource differs from pullIdentity. |

PlacementSettings.NetworkProfile derives worker.networkProfile from the placement bindings and worker configuration.
The protected network fingerprint remains in placement.policyProfile. A mismatched derived worker profile rejects startup.

Registry validation uses the same pure policy as the ACR client constructor.
Foreign origins, duplicate origins, public exchange IPs, and invalid identity resource paths reject configuration.
Source references still require immutable digests during inspection and launch.
Legacy workerEnvironment and controllerEnvironment fields reject as unsupported JSON.
The existing assembly-only preparation settings remain separate. Their migration is not part of this runtime loader.

### Concrete clients and credentials

OpenRuntime requires the real worker exchange with its shared database and authorization callback.
It constructs an ACR inspection client with publishing disabled.
It constructs the sandbox client for image, instance, and command operations.
It constructs SDKDeadlines with the same control credential and the selected independent workflow scope.
It constructs SDKPlacement with the explicit metadata credential and composes it before the additional protection checks.
The resulting image manager and provider share the controller's exact configuration and exchange.

Default credential construction uses ManagedIdentityCredential with each exact configured client identifier.
It uses no default credential chain, CLI login fallback, or environment-selected client identifier.
Construction requests no token and calls no service.
Library callers can inject a credential constructor and transports for controlled integration checks.
Injected credentials and transports remain the caller's responsibility. The runtime owns its created clients and default connection resources.
Credential-construction failures return a fixed unavailable error without underlying diagnostics.

The factory now checks selected reported group placement, import identity tenant metadata, and the chosen pull grant during admission.
Complete effective identity authority, credential host placement, actual network enforcement, and timer authority remain required separate checks.
Configuration syntax and distinct identifiers cannot replace that observation.

### Runtime modes

Execution mode uses image lookup and cannot import source images.
It accepts execution references and rejects discovery dispatch, status, cancellation, deletion, and inventory.
Discovery mode enables authorized conversion and accepts only discovery references.
It also requires ImageAdmission and the real AuthorizeDiscoveryCleanup callback.
The opposite workload cannot revoke a rightful exchange through this provider.

The library Provider constructor still permits an unscoped mode for explicit compositions.
OpenRuntime always selects a required scoped mode from Settings.
Wire native execution and source discovery roles separately.
Do not pass one execution-only runtime as both providers in a combined command registration.
That command integration remains unfinished.

### Admission and shutdown

Admission is mandatory for every runtime. ImageAdmission is also mandatory for discovery mode.
The existing controller calls the placement stage and these dependencies before creation, during live admission observation, and before import.
Live shared ARM placement evidence has a 15-second bound. The additional protection and per-sandbox checks remain uncached.
OpenRuntime does not substitute a successful construction result for those checks.
Native commands must compose actual group, VNet, private exchange, source identity, and timer authority checks before enabling dispatch.

Stop new requests and join caller-owned operations before runtime shutdown.
Runtime.Close cancels and joins owned command work before closing registry, timer, and sandbox connection resources.
Repeated Close calls are safe.
The provider rejects new inspection, dispatch, observation, inventory, stop, and removal calls after Close.
The runtime does not close the shared database, worker exchange, or caller-owned artifact store.

## Tutorials

### Verify assembly locally

```sh
go test -race ./adapters/execution/azuresandbox -run '^TestSandboxRuntime|^TestSandboxProviderUsesSharedExecutorReportsEvidenceAndCleanup$|^TestSandboxProviderResumesPendingDiscoveryThroughCommandAndCleanup$' -count=1 -timeout=30s
```

The new checks verify file and environment parity, rejected bindings before credentials, actual client selection, mode boundaries, and shutdown.
An in-memory sandbox read and a real Logic Apps SDK GET verify the selected control credential and distinct token audiences.
No credential endpoint, Azure account, registry service, browser, or container is used.
The existing shared execution and discovery checks also pass with their explicit provider modes.
One affected ACR regression retains repository scope, read-only access, redirect credential separation, and closed-client behavior.
Evidence is output/azure-sandbox-runtime/summary.json and runtime.log.

### Complete native activation

Implement actual admission against explicitly selected service contracts.
The pinned SDK uses 2026-02-01-preview for sandbox group and data-plane operations.
The current Microsoft ARM template reference also publishes a distinct 2026-07-01 group shape with environmentId.
The preview quickstart uses group defaults, identity, and a VNet child without a managed environment.
Do not silently combine these versions when implementing the admission reader or deployment template.
Actual service compatibility remains part of the separate deployment phase.

Then wire the scoped runtimes to the source queue and execution commands.
Keep the API on image lookup. Keep import and discovery in the selected trusted role.
Package identity bindings, group, VNet connection, private exchange, import authority, and independent timers.
Keep the existing staged Jobs implementation disabled.

Related implementation: [provider](azure-sandbox-provider.md), [images](azure-sandbox-images.md), and [timers](azure-sandbox-deadlines.md).
Contract references: [Microsoft preview quickstart](https://learn.microsoft.com/en-us/azure/container-apps/sandboxes-quickstart-bicep) and [versioned ARM reference](https://learn.microsoft.com/en-us/azure/templates/microsoft.app/2026-07-01/sandboxgroups).
Acceptance 51 advances. The verified total remains 48 of 57, with nine open.
