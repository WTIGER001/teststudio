# Durable Azure job lifecycle

## Current boundary

The Azure adapter now implements the shared hosted Provider interface for Go and Playwright execution and discovery.
It uses the [job ownership contract](azure-worker-contract.md), shared worker exchange, and durable installation database.
Local fixtures exercise the production provider with a reopened SQLite database and simulated job controls.
The production SDK bridge also passes in-memory protocol checks without Azure credentials or connections.

Azure coded execution remains disabled in native commands and the Bicep package.
Concrete environment network admission, independent absolute deadline resources, registry authentication, and private exchange service packaging remain required.
Network, deadline, image inspection, and fresh dispatch authorization are mandatory constructor dependencies.
No default permissive implementation supplies these dependencies.
The core Azure package continues to support native HTTP checks.

## Durable phases

The azure-dispatches bucket retains one record per complete execution reference.
Its record contains the frozen proof, phase reservations, verified resource receipt, terminal completion, and cleanup state.
It contains no usable worker ticket, SDK credential, or application credential.
Revision checks and exchange assertions fence concurrent controllers before each dispatch mutation.

| Phase | Required evidence and behavior |
| --- | --- |
| Reserve | Match the gated exchange, reference, operation, and frozen deadline. One controller creates the journal. |
| Inspect | Check the supervised Linux AMD64 image, runner, project, and discovery capability. Check the worker pool. |
| Create | Confirm that the generated job name is absent. Reserve Create before its single SDK mutation. |
| Provision | Poll the exact owned definition. Keep executable input gated while provisioning is incomplete. |
| Deadline | Reserve deadline creation and verify the exact independent stop target before Start. |
| Authorize Start | Recover a ticket through current exchange authorization. Check the network and confirm empty execution history. |
| Start | Reserve Start before its single non-idempotent SDK mutation. Save or reconcile its exact child receipt. |
| Admit | Verify the owned execution template, running state, network admission, and independent deadline. |
| Observe | Retain provider phase and native completion. Preserve runner results and timing through the separate authenticated report. |
| Cleanup | Require revocation and saved result/evidence state. Confirm terminal work before removing owned resources. |

An existing resource at the generated name blocks fresh creation. The provider does not overwrite or adopt it as a new dispatch.
Concurrent Ensure calls retain the original replica timeout and proof. A later clock does not change that reserved specification.
Create can complete asynchronously. Ensure can return while admission remains pending.
Observe starts the owned job only after provisioning and the remaining dispatch checks succeed.

## Fresh authorization during provisioning

Worker exchange ResumeDispatch reconstructs the existing ticket only for unactivated gated work.
It checks the complete reference, current authorization, original deadline, and unrevised authorization records.
An assertion transaction detects concurrent revocation or permission changes before returning the ticket.
Activated, revoked, expired, reassigned, or otherwise unauthorized work receives no new ticket through this method.
Resumption does not change the input, deadline, nonce, worker instance, or activation state.
It is an internal controller method. No HTTP endpoint exposes it.

A recovered proof alone still cannot emit a Start request.
The provider needs the separately authorized ticket and a successful journal reservation.
Recovery after revocation never starts unfinished provisioning work.

## Uncertain mutations and history

Create and Start each have a durable attempted marker before the provider call.
Unknown Create responses retain pending admission while the exact definition becomes visible.
The controller never repeats an uncertain Create or Start.
Known provider rejection remains distinct from a lost response or outage.
A reserved Start with empty history remains unresolved. Empty history does not establish that the request never reached Azure.

The controller lists executions only under the validated parent job.
One matching template and ticket hash can reconstruct a missing receipt.
A list entry can omit resource identifier and type. The validated parent request supplies the missing parent context.
An identifier or type that is present must match the owned child.
Multiple executions, changed commands, injected environments, different tickets, and conflicting receipts reject ownership.
The provider does not select an arbitrary execution from an ambiguous list.

A saved terminal observation survives later execution-history expiry and job deletion.
A missing live execution without that observation remains unresolved.
Inventory returns current installation and profile records from the journal. Removed records stay fenced but leave recovery inventory.
Keep the old configuration available until its workers drain. New network or installation settings cannot take over its recovery proofs.

## Provider completion and test results

Azure execution responses do not expose a numeric process exit code in the selected API model.
The adapter therefore leaves ExitCode unavailable.
A verified Succeeded state uses NativeCompletion to identify its independent provider completion evidence.
The shared lifecycle accepts this evidence only for succeeded completion without an exit code.
Conflicting evidence, missing completion evidence, or native completion on a nonterminal phase is invalid.
Existing providers retain their numeric exit-code contract.

Native completion never creates a test result by itself.
The authenticated runner report still supplies functional state, measured duration, steps, logs, and evidence references.
A failed test remains failed even when its supervisor delivers the report successfully.
A failed provider cannot validate a passing runner report.
Missing reports retain unknown outcomes, unavailable timing, and explicit evidence gaps.
Discovery retains its separate report validation and requires independent successful provider completion.

## Cancellation and deletion

Stop first fences further dispatch in the journal.
A job still provisioning can be stopped logically without issuing Start.
An already started execution requires its exact owned definition, receipt, and execution template before cancellation.
Ownership checks remain separate from status interpretation.
An unknown or degraded provider status can still receive exact-execution Stop when ownership is verified.
Changed ownership blocks that cancellation. It never selects a different resource or a whole-job stop fallback.

Remove requires a revoked exchange and a durable result checkpoint.
Evidence must be complete or explicitly abandoned. Discovery uses its source-cleanup authority instead of a test-result checkpoint.
Unknown live history or an incomplete terminal observation prevents deletion.
The provider rechecks the current definition and execution ownership before deleting a present job.
The deletion marker survives a lost response. A 30-second journal backoff bounds repeated reads during deletion.
Cleanup confirms job absence before asking the deadline adapter to remove its exact stop resource.
The removed tombstone prevents another dispatch. Available terminal completion remains readable.

## SDK bridge and permissions

NewSDKClient requires an explicit trusted credential and fixes the public Azure management endpoint and API version.
It disables automatic retries and resource-provider registration.
Requests stay within the configured subscription, resource group, and generated job namespace.
Only exact job reads, creation, deletion, execution listing, Start, and exact-execution Stop paths are permitted.
Secret-listing, debugging, alternate endpoints, and other management paths are outside this transport.
Supported resource-group escaping retains the same owned path.

Each SDK operation has a five-second context bound. Responses have an 8 MiB limit.
Execution listing permits at most four bounded pages and 100 records per page.
Two executions already establish ambiguity, so listing stops there.
Pagination links must retain the exact parent job and fixed HTTPS endpoint.
Redirects and unexpected locations fail without forwarding credentials.
Raw provider diagnostics do not leave the adapter.

The eventual controller identity needs these scoped job actions:

- Microsoft.App/jobs/read and Microsoft.App/jobs/write.
- Microsoft.App/jobs/start/action.
- Microsoft.App/jobs/executions/read.
- Microsoft.App/jobs/stop/execution/action.
- Microsoft.App/jobs/delete.

Keep independent deadline authority separate. It must verify the exact job resource, deadline, and complete specification before admission.
Its removal hook must establish owned-resource removal or authoritative absence. A successful request acknowledgment alone is insufficient.
The untrusted worker receives neither controller authority nor deadline authority.
Registry writes remain on separate trusted preparation infrastructure.

## Targeted local verification

Focused race checks pass for both runners and both provider operations.
They cover provisioning, database reopen, fresh authorization, frozen budgets, lost Create and Start responses, late visibility, and concurrent controllers.
They also cover admission loss, changed ownership, duplicate executions, exact cancellation, saved terminal history, deletion backoff, and checkpointed cleanup.
SDK fixtures verify scoped single mutations, rejected pagination and redirects, and no automatic retry after an uncertain Start.
Shared lifecycle checks retain measured passing and failing outcomes with native completion and reject missing or conflicting observations.
Exchange checks reject activated, revoked, expired, ungated, permission-revoked, and concurrently invalidated resumption.

```sh
go test -race ./adapters/execution/azure -run 'TestAzureProvider|TestAzureSDKControl' -count=1
go test -race ./internal/hostedexecution ./internal/workerexchange -run 'TestHostedNativeCompletion|TestResumeDispatch' -count=1
```

Evidence is output/azure-worker-lifecycle/summary.json and the focused command results.
No UI build, browser, container, image build, source-runtime matrix, live provider call, registry push, or deployment runs.
Both native command packages compile. No tests are selected for that compilation check.
Local fixture success advances acceptance 51 and supporting recovery work in 52. The verified total remains 48 of 57.
Live Azure lifecycle, identity, network isolation, absolute deadline enforcement, and deployment parity remain unverified.

## Primary references

[Create or update](https://learn.microsoft.com/en-us/rest/api/resource-manager/containerapps/jobs/create-or-update?view=rest-resource-manager-containerapps-2025-07-01) defines asynchronous job provisioning.
[Start](https://learn.microsoft.com/en-us/rest/api/resource-manager/containerapps/jobs/start?view=rest-resource-manager-containerapps-2025-07-01) defines the execution template and receipt.
[Execution list](https://learn.microsoft.com/en-us/rest/api/resource-manager/containerapps/jobs-executions/list?view=rest-resource-manager-containerapps-2025-07-01) defines history and pagination.
[Delete](https://learn.microsoft.com/en-us/rest/api/resource-manager/containerapps/jobs/delete?view=rest-resource-manager-containerapps-2025-07-01) defines accepted deletion and absence responses.
