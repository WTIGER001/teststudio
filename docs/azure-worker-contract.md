# Azure job request and ownership contract

## Current boundary

The active Azure execution path now uses [isolated sandboxes](azure-sandboxes.md).
This ordinary Job contract remains reference material. Native coded Jobs remain disabled.

The new adapters/execution/azure package builds and verifies Container Apps Job requests through the official Go SDK.
It pins armappcontainers/v4 version 4.0.0 and management API 2025-07-01, matching the core Azure package.
These local contracts cover Go and Playwright execution and catalog discovery.
The [durable Azure provider](azure-worker-lifecycle.md) now implements asynchronous dispatch, status, cancellation, and checkpointed recovery.
The [trusted ACR authentication adapter](azure-registry-authentication.md) now supplies registry-specific token exchange.
The [independent deadline adapter](azure-worker-deadlines.md) now supplies durable timer resources.
The [topology inspector](azure-worker-network.md) now verifies environment and subnet bindings through scoped reads.
The [perimeter inspector](azure-worker-perimeter.md) now verifies the fixed NSG rules and default route.
The [firewall inspector](azure-worker-firewall.md) now checks actual attachment, dedicated policy, and complete group inventory.
The [DNS inspector](azure-worker-dns.md) now checks actual resolver policy, VNet association, complete rules, and default block.
The [selected private-zone inspector](azure-worker-private-dns.md) now verifies exact records, completed links, and absence of public fallback.
The [endpoint inspector](azure-worker-endpoints.md) now binds consumer-side approval, NIC ownership, subnet policy, and exact regional role addresses.
The [registry service inspector](azure-worker-registry.md) now verifies private configuration, regional authority, and service-side registry approval.
The [selected exchange inspector](azure-worker-exchange.md) now checks relay configuration, service approval, revision state, and HTTPS binding metadata.
The [workload inventory observer](azure-worker-workloads.md) now checks reported app and job placement across the configured subscription.
Complete network admission, effective read coverage, attachment authority, backend authority, identity role binding, and native wiring remain unfinished.
Azure coded execution remains disabled. The core deployment still supports native HTTP checks.

See [Azure core service](azure.md), [hosted lifecycle](hosted-execution.md), and [hosted preparation](hosted-discovery.md).
No cloud account or service call is part of this batch.

## Installation and execution identity

Config requires these explicit values:

| Field | Meaning |
| --- | --- |
| subscription | Lowercase subscription identifier. |
| resourceGroup | Resource group that owns generated jobs. |
| region | Canonical Azure region name. |
| workerEnvironment | Dedicated worker managed-environment resource identifier in that subscription. |
| controllerEnvironment | Separate trusted-controller managed-environment identifier in the same subscription. |
| pullIdentity | User-assigned image-pull identity identifier in that subscription. |
| registry | Exact installation registry hostname ending in azurecr.io. |
| repositoryPrefix | Lowercase repository namespace. Source images must be descendants of this namespace. |
| namePrefix | Two through six lowercase characters. Letters, digits, and single hyphens are permitted. |
| networkProfile | Required SHA-256 digest for the intended worker network policy. It does not prove network admission by itself. |
| exchangeOrigin | Canonical HTTPS DNS origin without credentials, port, path, query, or fragment. |

Environment and identity references can use other resource groups in the selected subscription.
Worker and controller environments must differ. Actual routing, identity authority, and environment isolation still need the network checker.
Every job uses the common managed reference for azure-container-apps.
Discovery requires its distinct preparation reference, catalog target, first attempt, and discovery operation.

Generated job names combine the prefix and 24 hexadecimal ownership characters. They remain shorter than 32 characters.
Tags retain complete ownership, installation, project, execution, configuration, operation, deadline, and specification digests.
The truncated name never substitutes for full ownership verification. Provider reconciliation rejects a conflicting existing resource.
The timing profile binds all configuration and the fixed Linux AMD64 resource class.
A changed network profile cannot reuse the previous comparable timing pool.

## Saved definition

The job definition fixes these settings:

- Manual trigger, one parallel replica, one required completion, and zero provider retries.
- Consumption workload profile with two CPU cores and 4 GiB memory.
- One immutable approved-source image from the exact registry namespace.
- One runner container with /runner/teststudio-job as its command.
- Fixed runner and private exchange arguments. Discovery adds its explicit operation argument.
- No init container, sidecar, probe, environment override, secret, shared volume, or mount.
- One user-assigned pull identity with lifecycle None. No system-assigned identity is permitted.
- Registry pull uses that identity. The definition contains no registry password.

The replica timeout uses the remaining ticket budget, rounded up to seconds, with a maximum of two hours.
The definition freezes that value. Recovery does not recompute it from a later clock.
Replica timeout limits replica lifetime. It does not establish the required independent absolute stop deadline across queue delay and controller failure.
That separate deadline remains required before dispatch can be enabled.

The selected API has no job container user override, read-only root setting, or capability-drop field.
The package does not invent those fields or claim ECS storage controls apply to Azure.
The provider requires supervised Linux AMD64 image inspection before creating a job.
Native wiring must use the strict OCI inspector, including its non-root identity checks.
Use the existing OCI inspector's strict image and source checks. Uploaded code cannot choose a replacement entrypoint or image configuration.
Container-scoped writable storage remains distinct from mounted shared journals. Live isolation still requires deployment verification.

## Start and recovery

Start emits the complete fixed execution template with only the execution identifier and short-lived worker ticket as environment values.
The saved job definition, recovery proof, and plan diagnostics contain no usable ticket.
Expired tickets cannot produce a start request.
Restoring a proof preserves the original definition and its ticket hash. It cannot recreate a start request.
Authorized Azure infrastructure operators can inspect execution environments. Worker tickets remain bounded to their exchange and deadline.
No application credential or provider SDK credential belongs in that environment.

The Azure Start operation has no caller-selected execution name or idempotency field in this SDK contract.
The provider now reserves that mutation durably before calling the SDK. It never repeats an uncertain Start after a lost acknowledgment.
Reconcile the owned job's bounded execution history and preserve an unresolved state if authority or history is missing.
A numeric request sequence or HTTP retry is not a logical-run idempotency guarantee.
Automatic provider retries stay disabled. The production SDK bridge also disables automatic resource-provider registration.

## Response verification

Definition verification requires the expected resource identifier, name, type, successful provisioning, tags, identity, configuration, and template.
Only documented provider metadata is omitted from specification comparison.
These fields include system metadata, managed-identity identifiers, provisioned outbound addresses, eventstream endpoint, and reported ephemeral storage.
Empty arrays normalize to absent arrays. Other writable differences reject ownership.

Start receipts require an execution name and its exact child identifier under the owned job.
Execution verification binds a response to that saved receipt and compares the complete execution template and ticket hash.
The documented execution-list response can omit resource identifier and type. Those optional fields must match when present.
The saved receipt and validated parent request remain necessary ownership evidence.
Missing templates, changed commands, extra containers, injected environments, secret references, and different tickets fail verification.
Unknown and degraded states remain unavailable. They never become successful observations.
Provider timestamps remain optional, including when a stopped or failed execution never starts a replica.
Missing timestamps remain unavailable. Contradictory reported timestamps fail verification.
Provider timestamps remain provider facts. They do not become measured test durations.

The execution model exposes completion state but no numeric process exit code.
Observation therefore retains provider completion without manufacturing an exit code or a test result.
The shared lifecycle now preserves independent native provider completion and the authenticated runner report.
A fresh dispatch ticket requires current authorization through ResumeDispatch. A recovery proof alone still cannot start work.
Missing or inconsistent runner reports must keep explicit evidence gaps and must not create passing tests.

## Targeted local checks

The new package's focused race checks cover the request contract, ownership mutations, ticket-free recovery, execution states, and timing identity.
The SDK fixture receives HTTPS requests in memory. It uses an explicit fake credential and makes no network connection.
It checks exact management endpoints, API version, resource paths, serialized create and start bodies, and exact-execution cancellation.
Both runners and both execution operations pass that fixture.
The first fixture expects the wrong management token audience. Reading the SDK's public-cloud configuration corrects that expectation.
Only the failing SDK fixture repeats after its correction.
Later targeted observation checks preserve absent optional timestamps, including executions that stop before a replica starts.
An outdated missing-start rejection case is corrected during that check.
No unchanged product suite, UI, image, or deployment check runs.

```sh
go test -race ./adapters/execution/azure -count=1
```

Current evidence is output/azure-worker-contract/summary.json and the focused command results.
The local contracts advance acceptance 51. The total remains 48 verified of 57.
The [lifecycle guide](azure-worker-lifecycle.md) records the subsequent local provider checks.
Real Azure execution, identity isolation, networking, status consistency, cancellation, deadlines, recovery, and profile parity remain unverified.

## Primary references

[Jobs schema](https://learn.microsoft.com/en-us/azure/templates/microsoft.app/2025-07-01/jobs) defines the selected resource fields.
[Job naming](https://learn.microsoft.com/en-us/cli/azure/containerapp/job) defines the shorter name restrictions.
[Identity lifecycle](https://learn.microsoft.com/en-us/azure/container-apps/managed-identity) defines None for platform-only identity use.
[Start API](https://learn.microsoft.com/en-us/rest/api/resource-manager/containerapps/jobs/start?view=rest-resource-manager-containerapps-2025-07-01) defines execution overrides and receipts.
[Execution list](https://learn.microsoft.com/en-us/rest/api/resource-manager/containerapps/jobs-executions/list?view=rest-resource-manager-containerapps-2025-07-01) defines optional resource metadata and provider observations.
[Container support](https://learn.microsoft.com/en-us/azure/container-apps/containers) specifies Linux AMD64 images.
[SDK source](https://github.com/Azure/azure-sdk-for-go/tree/sdk/resourcemanager/appcontainers/armappcontainers/v4.0.0/sdk/resourcemanager/appcontainers/armappcontainers) supplies the pinned request, response, and serialization types.
