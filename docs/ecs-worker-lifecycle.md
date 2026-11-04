# Durable ECS worker lifecycle

## Current scope

The ECS provider now implements the shared hosted execution and discovery lifecycle through typed ECS and Scheduler clients.
It reserves creation in PostgreSQL or SQLite records, verifies provider receipts, observes owned tasks, stops work, and reconciles cleanup.
The existing immutable [task and deadline contracts](ecs-worker-contract.md) remain the resource specification.

The provider now has optional [server and preparation settings](ecs-workers.md), scoped SDK construction, and rotating ECR authentication.
Its constructor requires an image inspector and both installation and task network-admission checks. No default network bypass exists.
The [production network checker and bounded settings loader](ecs-network-admission.md) are implemented with local protocol verification.
Private exchange infrastructure and the worker deployment package remain to implement.
The current AWS service profile continues to support native HTTP checks only.
Keep all hosted acceptance items open until their complete scope is verified.

## Durable creation and ownership

Each attempt reserves one ecs-dispatches record before a provider creation request.
The record retains its ticket-free proof, creation markers, exact task-definition ARN, exact task ARN, deadline state, and terminal observation.
It stores no worker ticket, provider credential, or raw provider diagnostic.
Current exchange ownership, operation, deadline, activation gate, and revocation state guard new creation markers.
Concurrent controllers share the same reservation. Existing reservations perform observation instead of another registration or launch.

The provider disables SDK retries on these calls and supplies the frozen launch and schedule client tokens.
It saves a marker before each registration, launch, or deadline creation.
A lost response remains uncertain. Recovery never repeats that creation request.
Definition recovery lists the exact family across active, inactive, and deleting revisions and verifies immutable content and ownership.
Task recovery lists that family for running and stopped desired states, then verifies the exact definition, claimant, overrides, and ticket hash.
Pagination has explicit bounds and rejects repeated cursors. Multiple matching owned resources remain ambiguous.

An empty listing does not prove an uncertain request never succeeded.
The journal remains pending until an exact resource appears or an operator resolves the uncertainty.
Provider rejection before creation is recorded separately and permits cleanup of any already confirmed resources.
Completed records remain as replay tombstones. Ordinary cleanup never deletes the dispatch reservation.
Database backup and restoration must preserve these reservations with their execution and source records.

## Admission before executable input

Fargate task creation can finish before the task reaches RUNNING and before its network interface can be checked.
The shared status contract now exposes AdmissionPending for that interval.
Execution and discovery poll within their existing wall deadlines while the exchange gate remains closed.
Provider creation, an immutable task definition, the independent stop schedule, and task network admission must all pass first.
Worker instances cannot activate that gate themselves.

Each active observation checks task identity, fixed overrides, platform, resource class, and the independent deadline.
The required task network checker verifies the actual interface and isolation before admission.
If admission is lost after activation, the controller revokes the ticket and stops the owned worker.
Cleanup observations remain available after revocation, including when deadline creation never finished.
That recovery path cannot reactivate the ticket or create another task.

## Stop, terminal receipts, and missing history

Stop records controller intent before calling the exact task ARN.
StopTask can remove tags while the task still reports a stopping state.
Cleanup verification therefore uses the saved ARN, exact definition, startedBy identity, group, environment identifiers, and ticket hash.
It still rejects changed ownership, credentials, or commands.
Normal admission requires the expected ownership tags.

Only STOPPED proves terminal state. Intermediate task states and the stop request itself do not prove termination.
The provider saves the terminal phase and container exit code before permitting resource deletion.
The saved terminal receipt remains available if ECS later expires its task history.
A missing task without that receipt remains unresolved and blocks cleanup.
Provider duration never becomes a measured test duration.

See [StopTask](https://docs.aws.amazon.com/AmazonECS/latest/APIReference/API_StopTask.html) and [task listing](https://docs.aws.amazon.com/AmazonECS/latest/APIReference/API_ListTasks.html) for the underlying service contracts.

## Checkpointed cleanup

Normal execution cleanup requires the exact saved result and complete or explicitly abandoned evidence.
Discovery cleanup requires the source service's saved preparation checkpoint or valid recovery claim.
Both require a revoked matching exchange record.
Ticket revocation alone does not replace either checkpoint.

The provider confirms termination, verifies and deletes the owned deadline, then deregisters and deletes the exact immutable task definition.
Changed schedule ownership or task-definition content blocks deletion.
Schedule deletion confirms absence after the delete request. An unconfirmed creation that remains absent is still uncertain.
The dedicated Scheduler group and scoped controller permissions must prevent unrelated writers from reusing owned schedule names.

Task-definition deletion is asynchronous. DELETE_IN_PROGRESS keeps cleanup pending and preserves the saved result.
The provider checks again after one minute instead of repeating cloud reads on each worker reconciliation tick.
AWS documents that deletion can take up to one hour after a task stops.
See [definition deletion](https://docs.aws.amazon.com/AmazonECS/latest/developerguide/delete-task-definition-v2.html).
The AWS missing-definition response is accepted only for the exact recorded ARN and its recognized missing-resource form.
Other provider errors remain unavailable and retain the journal.
Public methods return bounded product errors and exclude raw SDK diagnostics.

The shared transfer barrier already rejects outstanding worker exchanges and source leases.
Those records remain until provider cleanup finishes, so delayed deletion also keeps installation transfer blocked.

## Focused local evidence

The new checks use authored in-memory ECS and Scheduler protocol fixtures with an owned local SQLite database.
No account credentials or cloud clients are discovered. No AWS, cluster, Azure, registry, or browser operation runs.

Evidence is output/ecs-worker-lifecycle:

- contract.log covers initial lifecycle checks.
- final-contract.log covers ECS contracts and shared lifecycle checks with race detection.
- restart-and-control.log covers database reopen, lost acknowledgements, concurrent controllers, admission, private errors, and delayed deletion.
- checkpoint-invariants.log checks persisted result and ownership invariants.
- discovery-compatibility.log and kubernetes-compatibility.log preserve the affected existing paths.
- compile.log confirms current API and preparation commands still compile.

Both runners and both operations pass checkpoint-gated cleanup, tag-loss recovery, and replay fencing.
The shared driver passes delayed admission and admission-loss checks through its real exchange and execution ledger.
The UI, service images, trusted runtimes, and browser dependencies remain unchanged.

## Continue implementation

Keep the implemented network checker mandatory. Its dedicated worker VPC and PrivateLink path separate worker egress from trusted controller access.
The corrected task contract uses supported task-local volumes, an explicit 21 GiB allocation, and a new timing-pool identity.
Require the inspected non-root volume capability before execution admission.
The scoped SDK factory and private settings now connect the API and preparation commands.
Preserve explicit account and region scope, disabled creation retries, current credentials, and private error handling.
Package private worker HTTPS routing, controller permissions, platform-only pull permissions, and the independent deadline role.
Connect registry assembly and hosted discovery through the existing preparation queue.
The core service template must keep ECS worker settings disabled until its private resources and permissions are packaged and checked locally.
