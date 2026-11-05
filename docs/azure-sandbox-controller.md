# Durable Azure sandbox dispatch and cleanup

## Introduction

The sandbox Controller connects the [data-plane client](azure-sandboxes.md) to the installation database and shared worker exchange.
It reserves one create attempt, reconciles lost receipts, enforces admission gates, and checkpoints cancellation and resource removal.
It does not implement the complete hosted Provider or produce a test result.
No native command or deployment template selects it yet.
Actual Azure verification remains separate work.

## Concepts

### Required dependencies

NewController requires validated Config, Instances, Deadlines, Admission, and the actual workerexchange.Service.
The exchange supplies the shared database, fresh dispatch authorization, activation gates, revocation, and cleanup checkpoint checks.
The constructor accepts no substitute authorization callback or separate dispatch database.
After reopening the database, recreate the controller with the exchange's new Store.
The caller owns its client and database lifetimes.
Supply Commands for executable work. Without it, the controller provides resource control but cannot start a test command.

Instances is implemented by the production sandbox client.
Its inventory must return a complete bounded result or an error. A partial list is not authoritative.
The [sandbox deadline SDK](azure-sandbox-deadlines.md) now supplies durable independent timer control.
Effective timer authority, native composition, and actual instance admission remain required before deployment selection.
The local fixtures do not establish those production controls.

The controller accepts a Plan after approved source inspection and Ready disk-image resolution.
The [image manager](azure-sandbox-images.md) now supplies durable import and lookup. Source-queue composition remains separate work.
This controller does not execute uploaded code during source inspection or import.

### Immutable work and reservation

Dispatches use the azure-sandbox-dispatches bucket. Ordinary Job records use a separate bucket.
Each row binds the complete execution reference, project, configuration proof, original creation time, and exact input hash.
Reservation atomically asserts the exchange row and inserts the dispatch row.
The exchange must be gated, unrevoked, unactivated, and within its absolute deadline.
Its image, runner, run, test, and project must match the planned work.
Discovery also matches the exact source digest.

The source digest uses the OCI inspector's canonical sha256:HEX form.
The v2 proof binds this digest. Earlier staged proofs cannot silently restore under the new contract.
No native installation writes the earlier staged sandbox format.
Input bytes must match the shared exchange's SHA-256 checksum.
The reserved input hash also remains fixed across restarts and later record revisions.

Rows retain ticket and environment hashes without a usable ticket or copied work-item body.
Advance obtains the current ticket through ResumeDispatch before an initial create.
The ticket must retain the same execution, deadline, and hash.
Restoring a proof alone grants no dispatch authority.

### Deadline and create ordering

Ensure reserves a new plan. Advance continues an already reserved attempt.
The controller saves DeadlineAttempted before calling the deadline implementation.
It then requires verified independent deadline coverage before any create reservation.
A lost deadline acknowledgment permits verification, not another blind creation call.
An uncertain or missing deadline blocks dispatch.

DeadlineSpec binds Config and the immutable logical execution proof before a sandbox identifier exists.
The native deadline implementation must cover all matching late creations and lost create receipts.
It must survive controller loss and must not depend on sandbox idle suspension.
The ordinary Job ARM timer remains separate from the [sandbox-specific implementation](azure-sandbox-deadlines.md).
Deadline removal must confirm resource absence through that implementation. A delete acknowledgment alone is insufficient.

After deadline verification, Advance reauthorizes the gated exchange, checks current creation admission, and reads complete sandbox inventory.
An already matching instance before create reservation blocks creation as an ownership conflict.
The controller atomically records CreateAttempted while asserting the current exchange row.
Only the winning replica calls Create.
It never repeats that mutation after an uncertain response or process restart.

A definite rejected create remains distinct from a lost acknowledgment.
A successful response stores its exact receipt only after full configuration verification.
Cancellation can race the request. A late valid receipt remains recorded, with admission blocked by the saved cancellation barrier.

### Reconciliation and admission

A saved receipt selects an exact instance read.
Without a receipt, reconciliation requires one full configuration match from complete inventory.
Zero matches remain unavailable after an uncertain create. They do not prove that a delayed request cannot create a resource.
Multiple matches reject ownership. The controller chooses no arbitrary instance and creates no replacement.
Changed source, commands, resources, credentials, networking, or ownership settings reject adoption and resource control.

Observe checks current deadline coverage and instance admission before clearing AdmissionPending.
It asserts the latest dispatch and exchange revisions before returning an admitted state.
Unchanged observations use revision assertions instead of writing another copy of the same state.
Revoked work, expired deadlines, canceled dispatch, incomplete checks, and placement failures keep admission pending.
The controller never calls Activate. The shared execution coordinator retains explicit activation and its own current authorization check.

### Cancellation and cleanup

Stop first saves a cancellation barrier. Initial dispatch can no longer pass the atomic create guard.
It resolves and verifies the owned instance before requesting stop.
Accepted stop responses do not set Quiesced.
Only verified Stopped or Suspended readback confirms quiescence.
Failed and unknown states do not establish stopped execution.
An observed quiescent instance cannot silently resume under the same record.
Stop retries have a persisted 30-second backoff.

Remove requires revoked exchange work and the canonical result and evidence cleanup checkpoint.
Normal runs use the existing worker ledger. Discovery uses the exchange's configured discovery cleanup authorization.
Remove saves the cancellation barrier again before deciding whether any create is still unresolved.
This fresh revision prevents a concurrent create reservation from being mistaken for never-created work.

Deletion requires verified quiescence for the exact saved instance.
Accepted deletion remains pending until an exact read confirms absence.
A persisted 30-second backoff prevents repeated deletion requests after a lost or delayed response.
A later confirmed absence can finish that step without another deletion request.
Missing inventory cannot discard a create attempt whose receipt remains unresolved.

Only settled instance ownership permits independent deadline removal.
Lost deadline-removal acknowledgments keep the dispatch unfinished and permit checkpointed retry.
The dispatch becomes Removed after instance and deadline cleanup both complete.
The retained proof and receipt remain available for audit.
Reopened completed cleanup performs no further instance operation.

### Outcome boundary

DispatchState now includes independent [command completion facts](azure-sandbox-commands.md).
Controller.Status combines explicit service-returned exit codes with confirmed instance quiescence through the shared hosted status contract.
It does not infer measured test duration or derive command success from a test report.
The [provider bridge](azure-sandbox-provider.md) now connects source resolution and the shared lifecycle.
Actual admission, concrete native SDK assembly, and configuration remain unfinished.
Lost command acknowledgments retain explicit observation gaps and never restart the same logical attempt.
Close joins owned command routines before SDK clients or the shared database close.

### Inventory and errors

Durable inventory filters the installation and runner profile.
It validates selected records before excluding completed removal, so an invalid Removed flag cannot hide active work.
It has a five-second deadline and a 100-page limit, with 100 rows per page.
An exhausted bound or invalid selected record returns an error without partial references.

Dependency calls have five-second limits. The concrete client retains its own request, response, and paging bounds.
Recognized wrapped control errors retain their fixed product classifications.
Private callback details and usable tickets do not enter returned errors or dispatch observations.

## Tutorials

### Verify changed behavior locally

```sh
go test -race ./adapters/execution/azuresandbox -run '^TestSandboxController' -count=1
```

The new checks use real SQLite storage and the real worker exchange with simulated instance, deadline, and admission controls.
They close and reopen the database for receipt recovery and interrupted cleanup.
Twelve concurrent controller calls reserve one external create and one deadline creation.
A coordinated in-flight create verifies cancellation without sleeps and joins its owned routine on every exit.
Checks also cover both runners, discovery, work-item changes, duplicate receipts, scope boundaries, missing checkpoints, and lost cleanup acknowledgments.
The selected earlier plan and response checks cover the changed source proof and shared state validation.
Retained evidence is output/azure-sandbox-controller/summary.json and the individual command results.
No actual registry, identity endpoint, cloud service, listener, browser, container, or deployment runs.

### Continue the native worker path

Bind the sandbox deadline SDK and actual admission composition before selecting the controller in native commands.
Connect durable image import and Ready-image resolution to the existing trusted preparation queue.
Use the implemented command execution and service-returned completion receipt handling without replay after an uncertain start.
Connect the shared hosted Provider, source discovery, evidence, cancellation, and cleanup contracts.
Package installation-specific identities, sandbox group, VNet connection, import authority, and independent deadline resources.
Confirm actual service audience, virtualization selection, source conversion, certificate trust, and effective policy in the separate deployment phase.
Keep ordinary Job environment settings outside the new sandbox configuration path.

This advances acceptance 51 and supporting recovery under 52.
The total remains 48 verified of 57, with nine open.
