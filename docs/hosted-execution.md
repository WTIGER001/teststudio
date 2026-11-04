# Hosted execution lifecycle

## Current scope

The shared hosted lifecycle driver is implemented in internal/hostedexecution.
It implements the existing managed executor and timing-profile contracts.
It connects a provider implementation to the private worker exchange and the durable execution ledger.

The optional [Kubernetes Pod adapter](kubernetes-workers.md) and [ECS adapter](ecs-workers.md) now connect this lifecycle to concrete providers.
ECS private worker infrastructure now has [staged packaging](aws-workers-package.md). Target-profile execution and recovery remain unverified.
The default local API keeps its existing local runners.
Complete hosted execution and deployment acceptance remain incomplete.
Native [hosted preparation](hosted-discovery.md) and its Helm deployment are implemented. Trusted gateways, remaining adapters, and deployment verification remain required.
The shared driver currently admits offline Go and Playwright work only.
It rejects environment-backed artifacts and credential bindings until trusted gateway delivery exists.

See [Hosted runner supervisor](job-supervisor.md) for worker packaging.
See [Private worker exchange](worker-exchange.md) for frozen work, tickets, reports, and staging limits.

The lifecycle now supports an explicit provider AdmissionPending state before controller activation.
Execution and discovery wait within their existing deadlines while the worker gate stays closed.
Admission loss after activation revokes the ticket and starts owned cleanup.
The optional ECS provider uses this state while its task and network interface become ready.
See [ECS provider lifecycle](ecs-worker-lifecycle.md) for its implementation boundary and remaining installation controls.

## Admission and timing

The installation supplies one provider, runner, profile name, private HTTPS origin, and artifact approval callback.
The origin contains no path, query, fragment, or user information.
The provider inspects the selected registry distribution digest.
The [OCI registry adapter](registry-inspection.md) supplies bounded metadata, platform, and supervised-runtime verification.
Local image identifiers and mutable tags cannot launch hosted work.

The driver requires the selected test in the artifact's discovered catalog.
Go uses the existing package and test selector parser.
Playwright uses its permanent discovered test tag.
The registry response must match the exact image reference, runner, Linux platform, runtime label, and worker-exchange/v1 supervisor contract.
An artifact with another project's explicit ownership cannot dispatch.
Inspected project ownership must match the approved artifact.
A retained provenance digest must match the inspected source digest.

Timing profiles include the provider profile, runtime identity, operating system, and architecture.
Observed test duration comes from the validated report.
Provider duration never becomes test duration.
An unavailable observation keeps unavailable timing and an explicit gap.
An enforced deadline produces the separate timed-out status when no valid test observation exists.

## Durable dispatch

The run service creates the durable execution reference before calling the executor.
Hosted execution requires the existing dispatch, checkpoint, and cleanup callbacks.
The reference includes installation, project, run, test, attempt, worker, and profile.
A deterministic portable job name derives from the execution identifier.
The complete reference remains the ownership authority.

The driver checks dispatch authority before preparing the packet and again before creating the job.
The work packet keeps its immutable input and discovered selector.
Its deadline is bounded by the attempt, caller, and current exchange authority.
The provider receives the approved image, fixed runner, private origin, reference, and scoped ticket.
The packet stays behind a controller activation gate until provider creation and identity verification succeed.
A pending or rejected provider cannot release uploaded code through that gate.
The ticket does not serialize with the job or appear in its diagnostic string.

The driver calls Ensure once for this dispatch.
An ambiguous response leaves the durable execution and private staging available for recovery.
It revokes the worker ticket before returning the unresolved observation.
Recovery discovers that same execution. It does not create another job.
A provider can return ErrRejected only when it proves that no creation was accepted.

Control calls have five-second bounds inside the enclosing lifecycle deadline.
Polling defaults to 250 milliseconds.
The driver forwards one acknowledged test start through the normal run callback.
Provider-specific execution settings can add a timing profile component for the selected execution pool.
It checks the complete reference, image, deadline, phase, and exit code on each provider observation.

## Observation, cancellation, and cleanup

A successful provider job proves supervisor delivery, not a passing assertion.
A valid failed assertion retains its failed outcome and measured duration.
A provider failure cannot validate a passing report.
Available private attachments can remain beside the resulting unknown observation.

Cancellation and hard deadlines use a fresh bounded controller context to stop the owned job.
The controller waits for terminal state or authoritative absence.
An uncertain stop or mismatched resource remains available for reconciliation.
It cannot pass checkpoint and deletion as a completed execution.

Worker tickets are revoked before the result checkpoint.
The checkpoint saves the normal run result and available evidence.
Incomplete retryable evidence retains private staging and provider resources.
The existing reconciliation budget eventually permits an explicit evidence abandonment.

Cleanup checks the ledger before any provider deletion.
The ledger must show a saved result and complete or explicitly abandoned evidence.
Cleanup revokes the ticket, stops the owned job, removes its resources, and then clears private exchange staging.
Exchange cleanup still waits for active upload leases and releases staging reservations.
The final callback records cleanup completion or retryable cleanup failure in the ledger.
A cleanup failure does not replace a saved test outcome.

## Restart and evidence recovery

Recovery validates the durable reference and immutable source image.
It revokes further worker access and stops a remaining owned job.
It reads the accepted report and checksummed bytes from private staging.
It never invokes Ensure.

Recovery reconstructs FailedBefore from the attempt number.
The first attempt keeps false. Later attempts keep true.
This preserves the frozen input hash instead of changing capture semantics on restart.

The exchange exposes controller-only state without reading attachment content during polling.
A missing or temporarily unreadable attachment marks EvidencePending on the internal report.
That flag never comes from worker JSON.
The run service preserves the result checkpoint and retries evidence collection.
A later collection can succeed without another test execution or another attachment identity.

## Provider adapter requirements

| Operation | Required behavior |
| --- | --- |
| InspectImage | Inspect the exact approved distribution digest and return verified runtime metadata. Never substitute a mutable tag. |
| Ensure | Create at most one job for the durable reference. Use provider idempotency and inspect an existing name before reuse. |
| Observe | Return authoritative ownership, image, absolute deadline, phase, and supervisor exit status. Pending work has no exit code. |
| Stop | Stop only resources with the complete matching ownership reference. Bound pending creation and repeated stop requests. |
| Remove | Remove only owned resources. Confirm removal before success. Preserve uncertain operations for retry. |
| Inventory | Discover only the requested installation and profile. Preserve complete ownership metadata and reject ambiguous resources. |

ErrNotFound means authoritative absence after provider consistency handling.
A temporary empty list or delayed resource observation cannot produce that result.
ErrRejected means no provider creation was accepted.
Other dispatch errors preserve the reference for recovery.
Raw provider errors and credential-bearing responses stay outside user reports.

The provider must enforce a deadline independently of the controller and supervisor.
It must prevent automatic test retries and unwanted replacement workers.
It must enforce private exchange routing, process and network isolation, resource limits, and identity separation.
Runtime image pulls can use platform identity. Untrusted test code receives no cloud SDK credentials or application session.
The provider must preserve private evidence and owned temporary-resource cleanup after crashes.
These adapter requirements are not established by the shared Go interface alone.

## Local verification

Focused race-enabled checks use the production exchange, SQLite ledger, and private filesystem store with an in-process provider fixture.
They cover both report schemas, assertion failure, provider failure, private evidence, cancellation, deadlines, and ordered cleanup.
They also cover a lost creation acknowledgement, uncertain stop, failed checkpoint, failed deletion, and temporary evidence storage outage.
Admission, timing identity, foreign resources, and premature deletion have focused checks.
Two focused run checks preserve the original attempt flag and retryable evidence state.
Three exchange checks preserve frozen reports, active upload fencing, and staging reservations.

No provider API, cluster, cloud account, browser, or source-runtime build is used.
The evidence paths are recorded in [Implementation verification](implementation.md).
