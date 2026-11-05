# Hosted runner supervisor

The supervisor connects an isolated hosted job to the [private worker exchange](worker-exchange.md).
It fetches one frozen work item, starts its fixed runner, forwards start progress, and submits the report and owned evidence.
It runs inside the worker image. It never executes uploaded test code inside the API.

The supervisor, runtime overlay producer, and targeted local verification are implemented.
The optional [Kubernetes worker profile](kubernetes-workers.md) now supplies dispatch, private ingress configuration, and provider recovery.
Native remote preparation and discovery now connect to the existing source queue and approval workflow.
The optional Helm preparation package is implemented. Trusted remote gateway delivery, other providers, and hosted acceptance remain unfinished.
Keep acceptance 51 and the related hosted acceptance items open.

The overlay also advertises the [versioned sandbox command capability](azure-sandbox-commands.md).
Sandbox idle and command modes enforce the unprivileged Linux identity before accessing work credentials.
Ordinary job invocation retains the contract below. Rebuild the changed overlay before selecting the new sandbox capability.

## Job contract

The provider starts /runner/teststudio-job with these arguments:

```sh
/runner/teststudio-job --runner=go --exchange-origin=https://PRIVATE_WORKER_HOST
```

Use playwright for a prepared Playwright image.
The image and selected test must use the same runner.
The origin requires HTTPS, no embedded credentials, and no path, query, or fragment.
The supervisor accepts no ticket argument or arbitrary command argument.

Supply these values through the job's protected environment:

| Variable | Purpose |
| --- | --- |
| TESTSTUDIO_JOB_EXECUTION | The registered execution identifier. |
| TESTSTUDIO_JOB_TOKEN | The opaque exchange ticket for that attempt. |
| TESTSTUDIO_JOB_INSTANCE | An optional unique worker instance identifier. If omitted, the supervisor generates one at startup. |

A retry inside the same process retains the instance identifier.
A new instance cannot claim an already claimed attempt. Provider execution retries must not silently run the test again.
Use normal product retries and result recovery instead.

The supervisor sends canonical bounded input to the existing runner through stdin.
Go uses /runner/teststudio-go. Playwright uses the installed Node executable and /runner/run.mjs.
The prepared suite remains at /suite. The owned output directory remains /output.
The output directory must exist and contain no previous report.json or report.tmp.
A stale report blocks execution before the next process starts.

The child receives an explicit runtime environment for its toolchain, writable temporary directories, timezone, and packaged browser location.
The supervisor does not forward the job ticket, database settings, identity endpoint credentials, cloud keys, or personal proxy settings.
Target-backed runners continue to require the trusted gateway socket at /output/.browser-egress.sock.
The provider must supply that gateway and its network policy before enabling those profiles.
The supervisor does not create a gateway or establish a sandbox by itself.

## Progress and process ownership

The shared output parser drains stdout and stderr while retaining at most 2 MiB per stream.
It recognizes one matching test.started event with the selected test, selector, and a bounded timestamp.
Oversized lines cannot grow the progress buffer without limit.
Start delivery uses the original timestamp on every retry.
A slow progress request does not stop stdout or stderr from draining.

The runner process uses an owned process group on Linux.
Cancellation first sends termination, then applies a bounded forced-stop grace period.
The supervisor waits for the owned runner and stops remaining members of that group.
The provider still owns the job's complete PID namespace, process limits, and final resource removal.
Test code cannot gain a broader process or network boundary from this library alone.

The runner wall allowance uses the selected timeout plus the existing 60-second startup and cleanup allowance.
The exchange deadline remains the outer bound. A shorter parent deadline or cancellation stops the runner.
The runner's own test timeout remains responsible for normal timed-out test observations.
If the hard exchange deadline ends before delivery, the controller must use provider status and explicit evidence gaps.
The supervisor does not invent test duration from job startup or delivery time.

## Reports and infrastructure failures

The supervisor reads one bounded regular report file and applies the common report validator.
A matching passed, failed, timed-out, or skipped observation retains its actual test outcome and measured duration.
The transport envelope records the actual runner process exit code.
A nonzero runner exit cannot accompany a passing report.

A missing or invalid report produces teststudio.worker/v1 metadata.
Its observed outcome is unknown, or timed-out when the supervisor's runner allowance expires.
It carries an explicit infrastructure failure, zero unavailable test duration, and observed worker duration.
The supervisor preserves available private process logs without calling that condition an assertion failure.
A failed process that falsely writes a passing report therefore becomes an infrastructure observation.

The envelope has processExitCode and optional infrastructureFailure in addition to report and files.
Exit code -1 means that no ordinary process exit status exists.
The infrastructure discriminator must match the worker report's failure kind.
The broker validates that discriminator and the common identity, timing, size, and result rules before acceptance.
An infrastructure envelope cannot create a passing test result.

The supervisor's own exit status describes delivery.
It exits successfully after a valid report is accepted, including a valid failed assertion or an explicit unknown infrastructure observation.
The application reads the report to determine the test outcome.
Provider success alone never means that the test passes.

## Private evidence

Runner stdout and stderr do not stream directly to provider console logs.
The supervisor captures bounded process output as private log evidence when logs are enabled.
Its console messages contain only fixed delivery status text.
Known job ticket values are removed from report metadata and captured process output.
The child receives no ticket through its runtime environment.

Attachment collection accepts owned regular files through a rooted output directory.
Escape links, duplicate paths, invalid metadata, unsupported capture kinds, oversized files, and excess total bytes become explicit evidence gaps.
The selected capture mode controls screenshots, traces, video, and logs.
A timing policy that fails a slow passing observation also permits requested failure-only evidence.
The exchange and supervisor share this capture decision.

Accepted files enter a private temporary snapshot with read-only file permissions.
The supervisor computes each snapshot's exact size and SHA-256 checksum before submitting its manifest.
Upload retries reopen the same snapshot. They do not reread a mutable runner file or rerun a test.
The supervisor removes its owned snapshot directory after delivery or an error.
Final controller collection still applies the normal evidence quotas, retention, pins, and API diagnostic redaction.

Input, start, report, and upload operations use bounded retries for unavailable storage or transient conflicts.
Permanent authorization and invalid-message failures do not retry as if another execution were safe.
The report and start timestamp remain fixed across exact retries.
A lost report acknowledgement does not create another report or execute another test.

If staging capacity is exhausted before report acceptance, the supervisor omits attachments and adds a fixed evidence-gap message.
It then submits the same observed test outcome without consuming attachment capacity.
If an attachment upload fails after report acceptance, the delivery outcome records incomplete evidence.
The supervisor can exit successfully with that accepted result. The controller must preserve the missing evidence as a separate gap.
These cases do not turn a passing observation into an assertion failure.

## Build a supervised source runtime

Use an existing immutable TestStudio source runtime as the base.
The producer supports Go and Playwright source-base/v1 images with the 4 MiB work-item contract.
The base architecture selects the supervisor's Linux architecture.
The Azure profile requires an AMD64 base. The current locally verified overlays use ARM64 cached bases.

Build a thin overlay:

```sh
node scripts/build-job-runtime.mjs --image=IMMUTABLE_SOURCE_RUNTIME --buildx=.data/docker-tools/docker-buildx
```

An installed Docker buildx plugin can replace the explicit executable option.
The producer uses the current Docker context's built-in builder and loads the result locally.
It creates no additional builder daemon, registry push, provider job, or deployment.
The final overlay build has no network-enabled build step and needs no package installation.
It compiles only the supervisor and retains the cached Go toolchain or Playwright browser runtime.

The overlay runs as 1000:1000 and installs /runner/teststudio-job as a root-owned executable.
It declares teststudio.task-storage=task-volumes/v1 and image volumes at /tmp and /output.
Both directories belong to 1000:1000. Their modes are 1777 and 0700.
These declarations initialize non-root task-local storage under the [ECS worker contract](ecs-worker-contract.md).
The remaining runtime and prepared suite stay on the read-only root filesystem.
It retains the original runner entrypoint, runtime identity, API policy, and source-base labels.
The provider explicitly selects the supervisor entrypoint when launching a hosted job.
Ordinary local runner invocation remains available without changing existing demonstration images.
A project preparation must use the supervised base before the resulting suite image can launch this supervisor.

The producer hashes compiler settings, Go inputs, binary output, base identity, build backend, Dockerfile, and producer sources.
Unchanged inputs reuse the binary and overlay image.
Temporary build contexts are removed. Produced images and immutable base aliases remain as intended build cache.
The returned local image identifier differs from the registry distribution digest required by provider manifests.

## Local verification

The focused supervisor checks use local TLS, the production client and handler, and owned native child processes.
Fixtures exercise both report schemas without starting a browser or repeating complete source-runner suites.
Checks cover passed and failed observations, missing reports, invalid passes, credential exclusion, output draining, unsafe links, exact retries, quota gaps, and cancellation cleanup.
The CLI check rejects invalid or credential-bearing arguments without echoing private values.
Focused existing report and output cases preserve the local container contract after parser extraction.

The package check executes the supervisor's configuration guard and a focused volume probe in each overlay:

```sh
python3 scripts/verify-job-runtime.py --go-runtime output/job-supervisor/go-runtime.json --playwright-runtime output/job-supervisor/playwright-runtime.json --output output/job-supervisor/package
```

It checks metadata, non-root image configuration, retained runner entrypoints, the installed binary, and owned container removal.
The volume probe checks ownership and directory modes, writes only to /tmp and /output, and rejects writes to /runner and /suite.
It uses actual write attempts. Permission-bit inspection alone does not prove a read-only filesystem.
It uses no network, test suite, or browser.
Evidence remains in output/job-supervisor with targeted Go logs and package cleanup records.
No AWS, Kubernetes, or Azure functional check runs for this batch.

## Discovery operation

The provider adds --operation=discover for a source preparation job.
The frozen packet must name that same operation. Execution and discovery packets cannot substitute for each other.
The supervisor invokes the fixed Go or Playwright runner with --discover and empty standard input.
It accepts one bounded discovery.json file from an initially empty output directory and rejects extra outputs.
Discovery uses the same owned process-group shutdown, stripped child environment, private-value redaction, and bounded delivery retries.
Its submission contains discovery data or an explicit worker, timeout, or cancellation failure. It contains no test result or attachments.
Runtime overlays advertise teststudio.discovery-supervisor=worker-exchange/v1. The registry builder requires this capability before publishing source.
See [Hosted preparation](hosted-discovery.md) for source authority, review, approval, and checkpointed provider cleanup.

## Remaining integration

The [shared hosted lifecycle](hosted-execution.md) implements controller ordering and recovery through the existing managed executor contract.
The optional Kubernetes provider implements that lifecycle with explicit configuration.

Provider adapters must admit the exact prepared immutable image, create one owned job, and supply only its scoped work identity.
They must establish private worker HTTPS ingress, resource limits, network isolation, and trusted gateway delivery.
They must distinguish provider status, runner status, test outcome, and evidence completeness.
They must reconcile cancellation, hard deadlines, orphaned jobs, and persisted reports after controller restart.
Revoke worker tickets before publishing collected results or evidence.
Keep staging cleanup and provider cleanup ahead of the final worker ledger completion.
Remote gateway profiles, remaining providers, and hosted verification remain required work.

A managed worker can receive a retryable input response while provider verification remains pending.
The existing bounded input retry policy handles that activation gate without invoking uploaded code early.
