# Private hosted worker exchange

The worker exchange transports one attempt's input, start observation, report, and attachments between a controller and a hosted worker.
It uses shared installation storage and the existing private artifact store.
It runs no test code inside the API.

The transport, HTTPS client, and [hosted runner supervisor](job-supervisor.md) are implemented and pass local checks.
Provider dispatch, gateway delivery, and provider recovery adapters remain unfinished.
No hosted Go or Playwright execution is enabled by this feature alone.
Keep acceptance 51 and the related hosted acceptance items open.

## Ownership and tickets

A trusted executor prepares an exchange after the normal run service registers its execution reference.
The reference binds installation, project, run, test, attempt, controller owner, and execution profile.
The run must remain active with the same current test and attempt, an unexpired controller lease, and current execution authority.
Current project, suite, environment, protected-target, capture, and catalog checks still apply.
A completed, cancelled, expired, reassigned, or checkpointed attempt receives no new worker access.

Preparation freezes one input digest and deadline. A later call cannot replace that input or extend its deadline.
Concurrent controllers reconstruct the same ticket for the same prepared exchange.
An installation-private signing key and random exchange nonce derive the ticket through HMAC-SHA256.
The key stays in the database. A worker receives only its own opaque ticket.
Ticket formatting and JSON serialization omit the secret value.

The first input request binds one worker instance identifier.
A retry from that instance can fetch the same input. Another instance cannot claim or execute the same exchange.
Provider adapters must disable automatic execution retries that create another instance for an already claimed attempt.
Use result recovery and explicit product retries instead of silently running an accepted attempt again.

The deadline cannot exceed the run deadline and cannot extend more than two hours from preparation.
The future dispatcher must select the effective attempt wall deadline, including its permitted startup allowance.
The test's selected timeout remains in the work item. Queue and startup time do not become an invented test duration.
Expired tickets stop worker reads and writes. Trusted controller collection can still recover retained reports and bytes afterward.

## Work item

The selected work item has the existing 4 MiB maximum.
It carries one test identifier, selector, runner, type, symbolic required bindings, immutable artifact identity, capture policy, timing settings, and approved target policy.
Unrelated discovered selectors, catalog descriptions, native HTTP headers, source provenance, credential references, redaction values, and controller callbacks are excluded.
Real target credentials remain in the trusted gateway and controller context.
The worker receives no application session, database connection, signing key, or provider credential from this packet.

The selected artifact must identify the test's discovered selector and an immutable image.
A project-owned artifact cannot cross project ownership.
A target policy must match the source's target-backed profile and pass the existing target validation.
Preparation and provider admission remain responsible for immutable source approval and sandbox configuration.
The transport does not replace those controls.

## Listener and HTTPS client

The API can expose a separate listener with --worker-addr.
It is disabled by default and is not part of the UI or artifact listener.
An artifact-only process rejects this flag.

A local operator can select a private loopback port:

```sh
bin/teststudio --worker-addr 127.0.0.1:8042
```

Hosted profiles need a private worker route with trusted TLS termination and explicit network reachability.
The current provider core packages do not yet provision that route.
Azure also needs a separate private HTTPS service arrangement for this listener.
These settings belong to the provider adapter batch before any real hosted execution.

The Go client requires an HTTPS origin, execution identifier, worker instance identifier, and ticket.
Its default transport uses ordinary certificate verification and no environment proxy configuration.
Redirects are rejected. A redirect cannot forward the ticket to another origin.
Errors expose fixed protocol outcomes and exclude upstream bodies, request headers, and network diagnostics.
Trusted test transports can supply local certificate trust without changing the production default.

Requests use Authorization: Bearer TICKET and X-TestStudio-Worker-Instance: INSTANCE.
Never place a ticket in a URL, normal application session, command argument, or browser request.
The server rejects query strings, cookies, and browser Origin headers.
All responses use no-store. The private listener serves no UI or user sign-in flow.

## HTTP operations

Each operation starts at /internal/worker/v1/executions/EXECUTION_ID.
The execution identifier selects an existing server-owned record. The caller cannot supply another ownership context in the body.

| Method and suffix | Request and result |
| --- | --- |
| GET /input | Claims or reloads input for the same instance. Returns executionId, deadline, and the selected input object. |
| POST /started | Sends one at timestamp. An exact retry succeeds. A changed timestamp conflicts. |
| PUT /report | Sends report and files. The first valid submission freezes both. An exact retry succeeds. A changed submission conflicts. |
| PUT /attachments/FILE | Uploads one file declared in the frozen report manifest. The body must match its exact byte count and SHA-256 checksum. |

Ordinary reports keep the teststudio.go/v1 or teststudio.playwright/v1 contract.
The envelope also records processExitCode and an optional infrastructureFailure.
The supervisor uses teststudio.worker/v1 only for an explicit unknown or timed-out infrastructure observation with unavailable test duration.
That discriminator must match its failure kind. It cannot accompany a passing observation.
Its test, selector, terminal state, timeout reason, timing, steps, and Go observation use the same validator as local container reports.
The report body has a 2 MiB limit. Its transport envelope adds at most 64 KiB for file descriptors.
Unknown JSON fields and trailing JSON are rejected.

The files list contains path, size, and sha256 for every report attachment, with no duplicates or undeclared paths.
At most 32 files are accepted. Each file is bounded by the selected file policy and 64 MiB.
Their combined size cannot exceed the selected attempt capture budget.
Paths and filenames must be plain filenames. Media types, streams, capture kinds, and failure-only capture settings are validated.
The client escapes a valid filename for its URL segment. Spaces in an attachment path remain supported.

Successful reads return 200. Successful writes return 204.
Unavailable or invalid credentials return 404 without exposing record existence or owner details.
Invalid messages return 400. Conflicting claims or immutable messages return 409.
Exhausted staging capacity returns 507. Other storage failures return a fixed 503 outcome.
The supervisor reports an explicit evidence gap if staging capacity prevents an attachment upload.
It preserves the observed test result when permitted evidence is unavailable.

## Private evidence staging

A frozen submission reserves its declared bytes before upload.
Project and run staging counters prevent simultaneous attempts from independently consuming the full shared staging allowance.
The project allowance uses the current project capture limit. The run allowance uses its frozen capture budget.
An exact report retry reserves no additional bytes.

Staging capacity is separate from retained evidence capacity.
During handoff, private staging and retained evidence can temporarily use storage at the same time.
Final evidence import still enforces the existing project, run, test, and file quotas, retention rules, pins, and redaction policies.
Plan storage capacity for both bounded stages. Private staging is not itself a user-visible artifact manifest.

The service stages and verifies each upload before writing its content-addressed private object.
At most four uploads stage concurrently per service process.
Per-file leases protect active writes. Cleanup cannot delete a manifest or its bytes while a live upload lease remains.
Backend writes have a shorter deadline than their leases.
A revoked in-flight upload can finish its bookkeeping without restoring access to the worker.
This prevents a late write from becoming an untracked object after cleanup.

Collection verifies the retained bytes again against the frozen size and checksum.
It can recover bytes written before a controller or upload acknowledgement failure.
Missing bytes remain an explicit evidence gap. Collection can retry without executing the test again.
API reports continue through the existing text and JSON diagnostic redaction policy.
The trusted collector supplies controller-held redaction values. Those values never enter the worker packet.

## Completion and recovery integration

The provider managed executor must use this order:

1. Register the normal execution reference and recheck dispatch authority.
2. Prepare the bounded exchange and create one provider job through an idempotent provider operation.
3. Deliver only the assigned ticket and worker instance identity to the trusted supervisor.
4. Fetch work, execute the immutable runner under the selected isolation profile, and report start progress.
5. Submit the report and upload its declared files with exact retries after lost acknowledgements.
6. Reconcile provider status and the report. Validate the actual runner or provider failure consistently with the shared report contract.
7. Revoke worker access before publishing collected results or evidence.
8. Invoke the normal result and evidence checkpoint hooks.
9. Remove staged objects and release staging reservations after evidence completes or is explicitly abandoned.
10. Finish the normal worker cleanup ledger only after transport and provider cleanup succeed.

A provider failure cannot turn a passing report into a passing execution merely because the report exists.
The shared validator rejects a passing report paired with an incompatible failed process outcome.
Provider cancellation, hard deadlines, gateway credentials, network isolation, job discovery, and restart reconciliation still require their adapters.
The client library alone establishes none of those provider behaviors.

Cleanup is idempotent after a lost acknowledgement.
It requires a saved result and completed or explicitly abandoned evidence.
A failed object deletion or live upload keeps enough metadata for cleanup retry.
Quota release and exchange removal commit together.
Installation export refuses unfinished exchanges, including records that remain after a recovery interruption.
Completed installation archives retain the private signing key as protected server authentication state.
Keep those archives restricted to installation backup operators.

## Targeted local verification

Focused checks cover controller concurrency, one instance claim, secret-free input, immutable messages, expiry, wrong ownership, checksums, bounded capture, shared staging quotas, and safe cleanup.
A TLS fixture uses the real run authorization callback and production HTTP client.
It verifies start progress, a lost report acknowledgement, filename escaping, upload replay, cancellation, stale owners, and redirect rejection.
The existing local report cases verify Go preparation failures, invalid pass reports, and legacy timeout normalization after validator extraction.
The export barrier check verifies that an unfinished exchange publishes no partial archive.

The native listener check uses one fresh local process and no browser or Docker runtime:

```sh
node scripts/build-selftest.mjs --local-only > output/worker-exchange/build.json
python3 scripts/verify-worker-listener.py --build output/worker-exchange/build.json --output output/worker-exchange/listener
```

It checks explicit listener startup, absence of UI and unauthenticated work, lazy key creation, and owned process cleanup.
The focused evidence remains under output/worker-exchange.
No AWS, Kubernetes, Azure, source-runtime matrix, or browser suite runs for this batch.

The [shared hosted lifecycle](hosted-execution.md) uses controller-only state polling and checkpoint-gated cleanup.
Missing attachment bytes remain retryable internal evidence instead of immediately becoming a permanent omission.
