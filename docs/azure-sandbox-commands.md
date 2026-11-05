# Supervised sandbox commands and completion

## Introduction

The sandbox client and controller now execute a fixed supervised command and retain the data-plane service's explicit exit result.
Controller.Status maps that result and confirmed instance quiescence to the shared hosted status contract.
Authenticated test reports, evidence, and measured test durations remain separate worker-exchange facts.
Native configuration, durable source-image resolution, actual admission, and the complete Provider factory remain unfinished.
No native command selects Azure sandbox execution yet.

## Concepts

### Versioned image and idle launch

The runtime overlay advertises teststudio.sandbox-supervisor=azure-sandbox-command/v1.
The OCI inspector retains this capability. New sandbox plans require it alongside the existing supervisor and non-root storage contracts.
Older images cannot silently enter this execution mode.
The overlay builder verifies the label when creating or reusing an image.
It still reuses unchanged binary and image inputs through its existing content hashes.

The v3 plan binds the image capability, trusted idle entrypoint, fixed command body, deadline, environment hashes, and execution ownership.
Earlier staged sandbox proofs cannot restore as v3. No native installation writes those earlier formats.
Timing pool identity also binds the capability and UID 1000 execution class.

The guest starts /runner/teststudio-job with sandbox-idle-until set to the absolute UTC deadline.
Idle mode accepts no runner, exchange origin, discovery operation, or executable work.
It reads no worker ticket and performs no work-item request.
Its deadline must be future, canonical UTC, and at most two hours away.
Cancellation or expiry ends idle mode. Idle process termination does not establish command success.

### Linux identity

Idle and command modes enforce the isolated Linux identity before reading work credentials.
Root startup clears supplementary groups and changes real, effective, saved, and filesystem identities to UID and GID 1000.
Existing UID 1000 execution must retain GID 1000 and no unrelated supplementary group.
Other identities fail before work starts.
The guard sets no-new-privileges across Go runtime threads, disables retained capabilities, requires empty capability sets, and disables process dumping.
These modes require the static CGO-disabled Linux build used by the runtime producer.
macOS and Windows reject sandbox supervision. Their ordinary local workflows retain existing behavior.

The test runner receives the existing explicit child environment.
Provider credentials and the supervisor's worker ticket do not enter that child environment.
The provider still owns complete instance isolation and final quiescence.

### Fixed command request

The client calls the selected instance's executeShellCommand endpoint with a fixed command and /suite working directory.
Arguments select the approved runner, private exchange, sandbox-command mode, and optional discovery operation.
Every argument is quoted. No uploaded command, shell fragment, image override, or usable ticket enters the request body.
The guest uses its already bound environment ticket.

Credential acquisition retains a five-second limit.
The synchronous command response uses the original absolute ticket deadline, up to two hours, through a separate owned HTTP transport.
It does not inherit the five-second status-request limit or short response-header timeout.
Redirect following and request-body replay remain disabled. Responses remain bounded.
Only an explicitly present integer exitCode between zero and 255 is accepted.
Missing, null, malformed, oversized, or uncertain responses remain unavailable.
Stdout and stderr are discarded from this control result. Private logs continue through the worker exchange.

### Durable command reservation

StartCommand requires an admitted running instance and explicitly activated exchange work.
It reauthorizes the current reference and atomically asserts dispatch, input, permission, and exchange revisions.
The saved marker binds one command hash and one instance receipt before the network mutation.
Competing controllers cannot reserve another command for the attempt.
An uncertain response or restart never replays that mutation.

The controller owns its background command context and waits for its routines during Close.
Close cancels owned requests, preserves uncertainty, and attempts owned stop.
Close the controller before its SDK clients and shared database.
A successful response saves the exit code and control observation time before requesting instance stop.
That timestamp is not a measured test duration.

### Cancellation and outcomes

Cancellation persists the ownership-checked barrier before interrupting the command connection.
A rejected project reference cannot cancel another project's connection.
Concurrent cancellation can remain pending and uses the existing durable retry path.
Cancellation before an independent result suppresses later success classification.
Missing acknowledgments retain explicit unknown state and trigger owned stop without another command request.

Common hosted status becomes succeeded only after exit code zero and verified Stopped or Suspended instance state.
A known nonzero code becomes failed after the same quiescence check.
Uncertain, canceled, rejected, or idle-only termination remains stopped without a fabricated successful exit.
Actual test results and evidence still pass the shared exchange validation before collection.
An exit receipt alone cannot bypass instance cleanup, and a suspended instance alone cannot supply a passing result.

## Tutorials

### Verify the changed contract locally

Run the command checks:

```sh
go test -race ./adapters/execution/azuresandbox -run '^TestSandboxCommand' -count=1
go test -race ./cmd/teststudio-job -count=1
go test -race ./adapters/registry/oci -run '^TestOCIInspectionRetainsVersionedSandboxCapability$' -count=1
```

The controller checks use real SQLite storage, the real worker exchange, and local simulated provider controls.
They cover both runners, discovery, concurrent reservation, restart, uncertain responses, cancellation, result mapping, and scope boundaries.
The OCI check uses owned local TLS fixtures. Their listeners close when the check ends.
No Azure request or account credential is used.

The changed supervisor cross-compiles for Linux AMD64 and ARM64.
Local ARM64 Linux probes use a cached TestStudio base and the new binary through a read-only mount.
They verify root-to-1000 transition, restricted UID 1000 operation, and rejection of UID 1001.
The probes use no network and build or pull no image.
Their containers stop, report PID zero, and are removed. A corrected case-insensitive missing-object check confirms cleanup.
The existing user demo remains untouched.

Evidence is output/azure-sandbox-commands/summary.json and its retained unit, protocol, build, identity, and cleanup records.
Actual converted root filesystem behavior, command limits, identity enforcement, TLS trust, and service compatibility still require deployment verification.

### Continue native integration

Connect durable image import and Ready-image lookup to trusted source preparation.
Use the [provider bridge](azure-sandbox-provider.md) for shared lifecycle and report collection.
Bind actual instance and deadline authority, admission, and the concrete native SDK factory.
Reuse Controller.Status and the existing hosted executor, report validation, evidence, cancellation, and cleanup contracts.
Keep explicit observation gaps after lost command acknowledgments. Do not replace them with a repeated attempt.

This advances 51 and supporting recovery under 52. The total remains 48 verified of 57.
