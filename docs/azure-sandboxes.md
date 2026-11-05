# Azure isolated sandbox execution

## Introduction

The adapters/execution/azuresandbox package implements the Container Apps Sandboxes data-plane client and supervised launch contract.
This is the active Azure execution implementation path. The UI and API remain Container Apps.
Ordinary Jobs remain disabled because the current implementation does not establish isolation between executions.
The existing Job contracts remain reference material. They do not establish a sandbox deployment.

Sandboxes and the selected API are preview services. Hosted acceptance remains open.
No native command selects this package yet. No deployment template enables sandbox execution yet.
The [durable controller](azure-sandbox-controller.md) now supplies dispatch reservation, lost-receipt recovery, cancellation, and checkpointed cleanup.
The [independent deadline SDK](azure-sandbox-deadlines.md) now supplies owned timer creation, verification, and checkpointed removal.
The [command bridge](azure-sandbox-commands.md) now supplies independent exit receipts and common hosted status mapping.
The [image manager](azure-sandbox-images.md) now supplies durable import and verified converted-image lookup.
The [provider bridge](azure-sandbox-provider.md) now connects pending image preparation and the shared execution lifecycle.
The [runtime factory](azure-sandbox-runtime.md) now supplies concrete SDK assembly and independent settings.
The [placement stage](azure-sandbox-placement.md) now checks reported group, subnet, import identity, and selected pull authority.
The [authority stage](azure-sandbox-authority.md) now checks selected identity and least-authority role profiles for control and timers.
Native activation still requires complete authority and protected network admission, timer authority, source-queue command wiring, and packaging.
Local contract checks do not establish deployed isolation, service availability, identity authority, certificate trust, or complete product operation.

## Concepts

### Public service contract

The client pins API 2026-02-01-preview.
Its wire contract follows azure-containerapps-sandbox version 0.1.0b4, published through the Microsoft Container Apps SDK path.
The SDK is inspected as source. It is not installed, executed, or added as a product dependency.
The Go client uses the existing Azure credential interfaces and standard HTTP library.

The public wheel has SHA-256 a4eff5ffe8ef941132b517ef768064626d0f78c4c21ebd7268d74935e1444f0a.
Source provenance remains in output/azure-sandbox-contract/sdk/provenance.json.

The regional endpoint is https://management.REGION.azuredevcompute.io.
The data-plane path is /subscriptions/SUBSCRIPTION/resourceGroups/GROUP/sandboxGroups/SANDBOX_GROUP.
This path differs from the ARM resource path. It has no providers/Microsoft.App segment.
Configuration cannot replace the endpoint, region suffix, subscription, or group during a request.

The pinned SDK uses https://dynamicsessions.io/.default as its default token scope.
Some public examples use the management.azuredevcompute.io audience instead.
This implementation follows the pinned SDK scope. Later deployment verification must confirm the actual audience before enabling execution.
It does not try another audience after a rejected request or discover credentials from the environment.

### Source images

Image import accepts one distribution digest under the configured registry and repository prefix.
It uses an explicit user-assigned pull identity in the selected subscription.
It supplies no registry password. The runtime launch receives no pull or controller identity.
The imported image fixes the supervisor entrypoint and clears the inherited command.
Completed image reads must match their requested identifier, approved source, fixed supervisor, and complete ownership proof.
Both Ready and Succeeded are completed image states in the pinned SDK contract.
Complete bounded image inventory supports later reconciliation of a lost import acknowledgment.
The image manager reserves imports through the existing discovery exchange before dispatch.
Lookup processes cannot import. A lost receipt never permits automatic replacement.

A launch requires the approved OCI inspection result for its exact image, project, and runner.
It requires the common supervisor, non-root task-storage contract, Linux AMD64, source digest, and explicit discovery support when requested.
The converted disk image must be Ready or Succeeded and retain the exact original immutable reference.
Service conversion does not replace source inspection or prove effective runtime user and filesystem controls.

### Execution request

Each launch selects one private disk image and one VNet connection.
Configuration also requires an explicit virtualization selection through the SDK's vmmType string field.
It supplies no guessed virtualization default. Separate deployment verification must confirm the supported value and effective runtime boundary.
The fixed resource class uses two CPU cores, 4 GiB memory, and 40 GiB disk.
The entrypoint invokes trusted idle mode until the absolute deadline.
The command API later invokes the fixed supervisor with runner and exchange arguments. Discovery adds its explicit operation argument.
No snapshot, preset, shared volume, exposed port, application secret, or runtime identity enters the request.

The environment contains only the execution identifier and its short-lived worker ticket.
The outbound policy uses Full inspection, default Deny, and one exact exchange-host Allow rule.
The request explicitly disables proxy bypass.
The hosted gateway contract still applies to application targets and secrets. This client does not grant direct target access.
Later deployment verification must establish the VNet connection, effective egress enforcement, and proxy certificate compatibility.

Idle suspension uses Disk mode with a two-hour interval.
This is housekeeping, not an absolute execution deadline.
The native provider must install and verify an independent stop arrangement before allowing work activation.
The arrangement must also cover an uncertain create response and controller failure.
The existing Job timer targets the ARM Job API. The [sandbox timer](azure-sandbox-deadlines.md) uses the separate data-plane stop contract.

### Recovery and ownership

The immutable proof binds configuration, API version, audience, source image, canonical source digest, disk image, reference, operation, deadline, and launch definition.
The v3 proof requires the canonical sha256:HEX digest and versioned sandbox command capability.
It retains ticket and environment hashes. It retains no usable ticket.
A restored proof can verify readback but cannot create another execution.
Expired tickets and cancellation during credential acquisition prevent dispatch.
Timing pool identity binds the sandbox backend, configuration, network profile, and fixed resource class.

Snapshots retain only an identifier, state, and configuration hash.
Raw environment values do not enter returned snapshots, plan diagnostics, or saved proofs.
The ownership check compares the complete returned writable configuration after documented provider metadata removal.
Empty arrays and absent arrays normalize consistently. Unknown writable settings remain differences.
The writable vmmType field is not discarded as provider metadata. A reported override requires an explicit matching launch binding.
Changed commands, credentials, network settings, source references, resources, labels, ports, and mounts reject adoption.
The durable provider must also bind the verified snapshot to its saved resource receipt.

Creating a sandbox uses PUT on a collection. The caller does not choose the returned identifier.
An HTTP method name does not establish idempotency.
There are no automatic retries, resume calls, replacement launches, or provider registration requests.
A lost acknowledgment remains unavailable. The future provider must reconcile complete inventory under its saved proof.
Multiple matches remain uncertain. An incomplete list cannot establish absence.

Stop and delete address one validated identifier in the configured group.
Accepted responses do not prove termination or removal. Later status and absence checks remain required.
Stopped and suspended instances do not establish successful test completion.
Provider state never supplies a numeric process exit code, test duration, or passing test result.
Authenticated runner reports and retained evidence remain separate requirements.

### Request bounds

Each operation has a five-second limit, including credential acquisition and response reading.
Create also observes the ticket's absolute deadline.
Requests have a 64 KiB body limit. Successful responses have a 4 MiB limit.
Owned HTTP connections use TLS 1.2 or later, no environment proxy, and no redirect following.
Errors return fixed product classifications. They omit response bodies and credential errors.
Every response body closes. Closing the client releases its owned idle connections.

Inventory has a 20-second limit, eight-page limit, 500 items per page, and 1,000 total items.
It accepts complete arrays or value envelopes.
Continuation URLs must retain the exact endpoint, collection path, and API version.
Foreign scopes, filters, cycles, duplicate identifiers, malformed envelopes, and exhausted bounds return unavailable without partial results.
Continuation checks run before acquiring a credential for another request.

## Tutorials

### Check the contract locally

Run the focused new package checks:

```sh
go test -race ./adapters/execution/azuresandbox -count=1
```

These checks use an in-memory HTTP transport and an explicit fake credential.
They make no Azure request and start no container, browser, listener, or identity discovery process.
They cover both runners, discovery, image import, image inventory, create, read, stop, delete, and bounded execution inventory.
They also cover ownership changes, credential-free recovery, lost responses, redirects, error redaction, cancellation, and response cleanup.
Current retained evidence is output/azure-sandbox-contract/summary.json, final-contract.json, and final-contract.log.

### Continue native integration

Use the [durable dispatch controller](azure-sandbox-controller.md) for sandbox creation and lifecycle mutations.
Connect the durable image manager and verified Ready-image lookup to native source preparation.
Reuse the existing source queue, database, approved OCI inspector, worker exchange, and preparation authorization.
Reconcile returned identifiers under complete ownership proofs. Never retry an ambiguous create as new work.
Bind the independent sandbox deadline SDK to verified native identity grants and workflow protection.
Use the implemented Provider for command status, terminal runner reports, deletion checkpoints, and pending-launch recovery.
Use OpenRuntime for concrete SDK assembly. Complete actual admission before command selection.
Add explicit managed identity, group, VNet, source-import, and deadline packaging before native command selection.
Keep all installation credentials in trusted processes.
Keep actual Azure verification separate from this local implementation session.

## Sources

- [Microsoft sandbox overview](https://learn.microsoft.com/en-us/azure/container-apps/sandboxes-overview)
- [Microsoft sandbox egress controls](https://learn.microsoft.com/en-us/azure/container-apps/sandboxes-egress-policies)
- [Microsoft preview setup guidance](https://github.com/MicrosoftDocs/azure-docs/blob/main/articles/container-apps/sandboxes-get-started.md)
- [Pinned SDK package](https://pypi.org/project/azure-containerapps-sandbox/0.1.0b4/)

Acceptances 41, 46, and 51–57 remain open. This contract advances 51 without enabling native execution.
