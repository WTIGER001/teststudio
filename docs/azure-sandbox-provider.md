# Azure sandbox provider bridge

## Introduction

The sandbox Provider implements the shared hosted execution interface.
It connects approved OCI inspection, durable image preparation, sandbox dispatch, independent commands, and checkpointed cleanup.
The shared Go and Playwright executors use this bridge without a second execution or report format.
Discovery can wait for conversion before a sandbox exists.

No native command or deployment template enables this provider yet.
The [runtime factory](azure-sandbox-runtime.md) now supplies independent settings and concrete SDK assembly.
The [placement stage](azure-sandbox-placement.md) now checks reported group, subnet, import identity, and selected pull authority.
The [authority stage](azure-sandbox-authority.md) now checks selected control, registry inspection, metadata, and timer identity grants.
The [network stage](azure-sandbox-network.md) now checks complete selected security rules, the default firewall route, and exclusive subnet attachments.
Remaining authority and protected network admission, effective timer authority, command wiring, and native packaging remain unfinished.
Local protocol verification does not establish deployed Azure behavior.

## Concepts

### Composition

NewProvider requires a Controller with Commands, an ImageManager, and an OCI Inspector.
The controller and image manager must share the same worker exchange and sandbox configuration.
The production OCI client implements Inspector. The sandbox client implements image, instance, and command operations.
The existing deadline SDK supplies independent timers.
Admission remains a required controller dependency. Import admission remains required when the image manager permits conversion.
These injected dependencies do not establish actual identity or network controls by themselves.

AllowImport belongs in the separate preparation workload.
Only discovery uses Prepare. Ordinary execution uses Lookup even when its manager permits import.
A missing prepared source does not cause an execution process to import an image.
The existing shared executor still applies its artifact approval, selector, and supported target-binding rules.
This bridge does not add hosted recording or application-target gateway support.

### Durable launch before conversion

The azure-sandbox-launches bucket stores one launch intent per logical reference.
The proof binds the complete reference, inspected immutable source, operation, original deadline, image cache key, and frozen work hash.
The image key also binds the selected import identity. Reconfiguration cannot adopt an earlier pending launch under another identity.
The proof stores a ticket hash. It stores no usable ticket.
Initial reservation asserts the exchange and current authorization revisions.

Ensure returns after durable reservation. An incomplete conversion or uncertain acknowledgment remains pending for Observe.
Recovery obtains the original gated ticket through ResumeDispatch. It never creates another logical work packet or deadline.
The discovery image manager can advance conversion. Execution can only read a prepared conversion.
Ready or Succeeded conversion permits construction of the fixed sandbox Plan.
The controller then reserves the timer and sandbox creation through its existing lifecycle.
The provider checks that a controller record matches its launch proof before delegating status or mutations.

After sandbox reservation, status does not require another registry inspection or image import.
Image, timer, and sandbox uncertainty cannot produce admission readiness.
Before activation, unavailable startup observations remain pending within the original deadline.
After activation, unavailable control observations remain explicit errors for the shared recovery path.
Known rejected preparation or sandbox creation produces a terminal failed control state without an invented exit code.
Provider status never invents a test result or duration.

### Admission and command completion

The shared executor activates the worker exchange only after verified sandbox admission.
Controller.Status then starts the once-reserved supervised command.
Normal input claims, progress, and uploads can update the exchange during admission observation.
The observer retries conflicting final revision assertions up to eight times.
It rereads revocation and ownership each time. A revoked exchange cannot become ready through this retry.
Independent exit receipts and verified quiescence remain necessary for successful command completion.
Authenticated runner reports and uploaded evidence remain separate inputs to the shared result collector.

### Cancellation and cleanup

Stop validates the exact launch reference before saving a durable stopping barrier.
It revokes the exchange before reading the latest controller record.
An in-flight controller reservation must either precede that revocation or fail its exchange revision guard.
A late sandbox receipt remains owned and stoppable. It cannot activate canceled work.
No controller record after successful revocation establishes that this launch has no reserved sandbox.
An uncertain controller create still remains uncertain. Empty provider inventory does not turn it into absence.

A saved stop request can remain pending while observation confirms termination.
An accepted stop never supplies an exit result or passing outcome.
Remove requires the canonical result and evidence checkpoint.
It fences pending dispatch, delegates owned instance and timer removal, and marks the launch removed last.
Shared source images remain outside per-run deletion.
Completed control observations survive deletion of the cleaned worker exchange.

Inventory includes pending conversion intents as well as existing sandbox dispatches.
It validates exact installation and profile scope and rejects contradictory controller records.
Each provider call has a five-second bound. Inventory has a complete 10,000-record scan limit per journal.
Close joins the controller's owned command work. Call it before closing shared clients, artifact stores, and the database.

## Tutorials

### Verify the integration locally

```sh
go test -race ./adapters/execution/azuresandbox -run '^TestSandboxProvider|^TestSandboxControllerRetriesAdmissionObservationAfterConcurrentWorkUpdate$' -count=1 -timeout=35s
```

The local checks use SQLite, private filesystem artifacts, and in-memory provider controls.
The actual shared executor and worker exchange perform activation, input claims, start reporting, submission, private log upload, collection, and cleanup.
Both runner contracts retain their reported duration and complete one logical command without another image import.
These are provider protocol checks. They do not execute a hosted Go or browser process.

Additional checks cover pending conversion restart, scope, cancellation, in-flight creation, concurrent lost-receipt recovery, identity binding, and retained completion.
A deterministic observation check distinguishes a normal exchange revision change from actual revocation.
Evidence is output/azure-sandbox-provider/summary.json and targeted.log.
No cloud account, registry operation, browser, container, demo restart, or remote push is involved.

### Continue native setup

Use OpenRuntime for concrete OCI, image, sandbox, timer, and provider assembly with explicit scoped settings.
Wire execution and discovery command roles separately. Native commands do not select this factory yet.
Implement actual import and runtime admission before any command can enable dispatch.
Retain separate API and preparation identities. Keep installation credentials outside uploaded code and execution sandboxes.
Use the existing durable source queue, worker leases, evidence checkpoints, and orphan recovery.
Package the selected group, VNet connection, import authority, private exchange, and independent timer authority.
Verify live service compatibility and deployment acceptance in the separate hosted phase.

Related contracts: [images](azure-sandbox-images.md), [dispatch](azure-sandbox-controller.md), [timers](azure-sandbox-deadlines.md), and [commands](azure-sandbox-commands.md).
Acceptance 51 and supporting recovery under 52 advance. The verified total remains 48 of 57.
