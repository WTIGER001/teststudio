# Durable Azure sandbox image preparation

## Introduction

The sandbox ImageManager stores source conversion ownership in the installation database.
Trusted preparation can reserve an import. Execution processes can resolve an existing conversion without starting another import.
The production data-plane client implements ImageControl. No Azure SDK process runs inside the API or test sandbox.
Native Provider composition, actual import admission, configuration, and deployment packaging remain unfinished.
This library does not enable Azure execution.

## Concepts

### Immutable source and ownership

ImagePlan requires the approved OCI inspection for one project, runner, and immutable image digest.
It also requires Linux AMD64, the common supervisor, sandbox command capability, discovery capability, and non-root task-storage contract.
The expected pull identity must be a selected user-assigned identity in the configured subscription.
The cache key binds that identity, source metadata, execution profile, and sandbox configuration.
Different projects, sources, identities, or profiles cannot share a cache entry.

Each import generation binds its requesting discovery reference.
The request uses the fixed supervisor entrypoint, an empty command, and ownership labels with the complete proof fingerprint.
It supplies no registry password or custom Dockerfile.
Readback must retain the exact digest, entrypoint, command, labels, and empty Dockerfile content.
The returned snapshot contains a definition hash. It does not retain raw provider diagnostics or credentials.
The SDK does not report the import identity in its disk-image model. Request binding does not replace verification of actual identity authority.

### Authorization and reservation

Prepare requires AllowImport and the existing gated discovery exchange.
It validates the frozen work input, source digest, ticket, deadline, project, and runner.
ResumeDispatch and the normal authorization callback provide fresh permission checks.
Initial reservation and the import marker assert the current exchange and authorization record revisions.
Revocation or a changed grant prevents an unreserved import.
The separate BeforeImport admission hook must establish actual group and import-identity controls before native integration.
The hook does not replace source-queue authorization.

ImageBucket is azure-sandbox-images. Each current record contains one source proof and its import lifecycle.
The import marker commits before the collection PUT. Only its winning controller makes that call.
An existing matching resource without a prior import marker blocks creation and adoption.
Lookup mode cannot initiate an import. It can save verified receipts and status observations in the shared journal.
Each Prepare or Lookup call has a five-second bound, including local storage and provider requests.

### Pending conversion and recovery

A returned Building image remains pending. Only Ready or Succeeded permits a sandbox launch.
Lookup reads the saved exact receipt again. A cached Ready observation alone cannot authorize execution.
A missing acknowledged image remains unavailable. The manager does not silently replace it.

A lost import response retains the original marker.
Recovery searches complete bounded image inventory for the saved ownership proof.
Exactly one match can supply the missing receipt. No match remains uncertain. Multiple matches reject ownership.
An incomplete or failed inventory cannot prove absence and cannot authorize another import.
Restart and a new discovery request preserve this rule.

### Known failure and history

An explicit import rejection or verified Failed image remains a known failure.
The same discovery request cannot repeat its conversion.
A different authorized discovery request can reserve a new generation after a known failure.
The transaction first retains the old proof, receipt, and outcome in azure-sandbox-image-history.
Archived requests remain ineligible after later generations fail. They cannot alternate requests to repeat an old mutation.
Pending, missing, blocked, and uncertain conversions do not qualify for this retry path.

Converted images are source assets shared by later runs. Per-run cleanup does not delete them.
Source retention and provider quotas must include current and archived conversion assets before hosted acceptance closes.
Automatic image deletion is not part of this manager.

## Tutorials

### Verify the changed behavior locally

```sh
go test -race ./adapters/execution/azuresandbox -run 'TestSandboxImageImport|TestSandboxClientUsesPinnedAPIForImageAndExecutionLifecycle|TestSandboxClientRejectsInvalidReferencesBeforeAcquiringCredentials' -count=1
```

These checks use local SQLite, in-memory provider transports, and explicit fake credentials.
They cover both runners, restart, Ready lookup, twelve-controller contention, lost responses, ambiguous inventory, ownership drift, and current authorization.
They also verify known-failure generations, retained history, stale grant revisions, and lookup processes without import permission.
Evidence is output/azure-sandbox-images/summary.json and targeted.log.
No cloud account, registry, browser, container, demo restart, or remote push is involved.

### Connect trusted preparation

Keep AllowImport in the separate preparation workload. Use lookup mode in the execution provider.
Reuse the existing OCI inspector, preparation authorization, durable source queue, and worker exchange.
Represent pending conversion before sandbox creation through the complete native Provider.
Then construct the sandbox Plan from a freshly resolved Ready image and the approved inspection.
Preserve the original absolute discovery or execution deadline.
Connect the [dispatch controller](azure-sandbox-controller.md), [independent timer](azure-sandbox-deadlines.md), and [command completion](azure-sandbox-commands.md).
Implement actual import and runtime admission before native configuration can enable these paths.
Keep live Azure verification in the separate deployment phase.

The wire contract follows the pinned SDK recorded in [sandbox source provenance](azure-sandboxes.md#public-service-contract).
Acceptance 51 advances. Acceptances 41, 46, and 51–57 remain open.
