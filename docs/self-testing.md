# TestStudio tests itself

TestStudio owns a new Playwright suite in `tests/selftest/playwright`.
The suite runs through the normal catalog, publication, execution, timing, and evidence APIs.
It copies no tests or configuration from another application.

## Run locally

For atomic run admission and protected dispatch, use the focused local checks:

```sh
go test -race ./internal/runs ./internal/catalog \
  -run '^(TestRunCreationAtomically|TestProtectedManualAndScheduled|TestQueuedHTTPDispatch|TestRunAdmissionSupports|TestExecutionGuards|TestAttemptAdmissionStops)' -count=1
```

The checks create new local records and targets. They exercise concurrent edits, protected dependency selection, queued policy changes, and blocked retries.
The safe dependency cases execute actual HTTP requests through normal manual and scheduled runs.
Use scripts/test-postgres.sh with the same package and test arguments for disposable local PostgreSQL parity.
No browser or execution image is required. See [Dispatch safety](results.md#dispatch-safety-and-frozen-runs).

For OIDC credential boundaries, run the focused adapter and API checks:

```sh
go test -race ./adapters/auth/oidc ./internal/httpapi \
  -run '^(TestOIDCProviderAndResolverErrorsExcludePrivateText|TestOIDCClientCredentialStaysInServerTokenExchange|TestProviderAdministrationKeepsOwnershipRevisionsAndCredentialValuesPrivate|TestOIDCCallbackErrorsReturnReadableSignInOutcomes)$' -count=1
```

New local provider fixtures exercise discovery failures, credential resolution, successful exchange, and rejected exchange with private diagnostic text.
The checks verify server-only credentials, fixed errors, private stored metadata, and readable browser failure outcomes without a browser suite.

For the empty-installation and optional-example contract, use a built server:

```sh
node scripts/verify-empty-installation.mjs .data/selftest-build/controller
```

This focused check starts only owned local server processes with fresh SQLite data and no cloud environment.
It checks empty startup, empty restart, one selected HTTP example, explicit publication, and unchanged definitions after restart.
It starts no browser or container. Each owned server stops before the summary records cleanup.
Evidence stays under output/empty-installation. It does not touch the running demonstration.
See [First installation](first-installation.md) for the tester workflow.

Install Go, Node, npm, and Docker. Start the Docker daemon.
Run this command from the repository root:

```sh
bash scripts/test-self.sh
```

The script builds changed Go and UI inputs and reuses verified unchanged outputs.
It records source fingerprints and output hashes under `.data/selftest-build`.
It reuses the immutable runner image when its complete input fingerprint still matches.
The pinned base image is pulled only when absent.
The cache does not skip requested tests. Each invocation still creates a fresh verification installation.
It packages them with the pinned Playwright runtime and the new browser tests.
An isolated controller registers the catalog, publishes an immutable local image ID, and starts a normal TestStudio run.
Each test container starts a new TestStudio instance with its own SQLite database.
The instances use loopback endpoints inside the container. They have no external network access.
The suite tests themes, mobile layout, page help, catalog publication, HTTP assertions, native execution, timing, baseline reset, timeout reruns, and trends.
It also tests durable schedules, completion notifications, identity administration, scoped groups, role assignments, and session controls.
Effective-access checks exercise actual decisions, inherited source paths, expiry, redaction, bounded scans, and revoked inspection.
The native source workflow checks YAML uploads, server digests, reviewed changes, atomic publication, conflicts, retained source, explicit runs, and mobile help.
Native check runs remain separate from the outer Playwright run. They cannot recurse into browser self-tests.

The script stops its controller after verification. It preserves its database and private artifacts under `.data/selftest-verification.*`.
It saves a local result summary and downloaded screenshots and traces under `output/selftest/{runId}`.
The printed installation path contains `installation.json` with its origins and immutable image reference.
No cloud account or remote deployment is required.

Use `--test=ID` to execute one normal workflow with its usual publication, execution, and evidence checks.
Repeat `--test=ID` to select several affected workflows in one build and run.
Unselected workflows do not execute. Unchanged Go fixture binaries remain cached.

```sh
bash scripts/test-self.sh --test=STUDIO-HELP-001
```

The identifier must exist in the self-test catalog. Diagnostic flags use their separate selection.

## Identity interface fixtures

Run the identity workflow with:

```sh
bash scripts/test-self.sh --test=STUDIO-ACCESS-001
```

The workflow creates new profiles and groups through the normal service API.
It checks real revision conflicts, reason forms, archive, restore, bounded pages, and session revocation.
The browser uses a non-UTC timezone and verifies the stored UTC membership expiry.

The self-test image includes a dedicated directory preparer for authority checks.
It uses product identity, membership, role, and session services against the worker's fixed disposable database.
The preparer creates a project administrator, a read-only access reviewer, and provider-managed membership with fresh opaque sessions.
The browser checks installation and foreign-project rejection, hidden installation navigation, and read-only external membership.
Dedicated OIDC tests verify provider sign-in separately.

The preparer runs only inside the isolated self-test image. Its local fixture route belongs to the test wrapper.
The product HTTP server provides no fixture preparation route.
The wrapper awaits its owned service and helper processes during shutdown.
The normal worker cleanup removes the test browser container and its output volume.

## Assignment interface checks

Run the focused assignment workflow with:

```sh
bash scripts/test-self.sh --test=STUDIO-ROLES-001
```

The workflow uses actual API mutations and browser-issued role actions.
It verifies exact project scope, duplicate creation, stale removal after recreation, preserved reason fields, and administrator protection.
Project-administrator and read-only sessions verify navigation, shared-group controls, scoped delegation, and rejected installation or foreign-project views.
Self-revocation clears assignment data, project choices, and unavailable navigation without another sign-in.
Bounded pages, contextual help, light and dark screenshots, and mobile width checks use normal retained evidence.

## Focused bundle workflow

Run only the ZIP staging and cancellation workflow after source upload changes:

```sh
bash scripts/test-self.sh --test=STUDIO-BUNDLES-001
```

This workflow enables source-bundles on its own isolated controller.
It verifies exact source bytes, queued state, duplicate upload replay, cancellation, reload, and mobile layout.
It does not start a source builder inside the browser worker container.
The separate local preparation worker has focused source and builder checks.
A local browser workflow also verifies real assembly, publication, and a normal run with the cached Go runtime.

Check build reuse without starting another test:

```sh
node scripts/build-selftest.mjs
```

Unchanged builds print reuse messages and the verified immutable image identifier.
Changes to browser tests rebuild the image while preserving unchanged Go and UI outputs.
Changes to Go test files do not rebuild application binaries.
Do not run multiple self-test scripts against the shared build cache concurrently.

## Reuse a local installation

The worker stays disabled unless the administrator supplies permitted immutable images through `--playwright-images`.
The initial profile accepts local, targetless suites with no external bindings.
It does not accept arbitrary commands or mutable image tags from a run request.

After building the image, enable its immutable identifier on a local TestStudio service:

```sh
teststudio_selftest_image=$(docker image inspect teststudio/selftests:local --format '{{.Id}}')
bin/teststudio --local --playwright-images "$teststudio_selftest_image"
```

In a separate terminal, register and execute the suite:

```sh
teststudio_selftest_image=$(docker image inspect teststudio/selftests:local --format '{{.Id}}')
TESTSTUDIO_SELFTEST_IMAGE="$teststudio_selftest_image" node scripts/selftest.mjs
```

Set `TESTSTUDIO_ORIGIN` for another loopback origin.
Set `TESTSTUDIO_SELFTEST_PROJECT` for another project identifier.
Registration reuses permanent test identifiers in the `teststudio-selftests` project by default.
It preserves lifecycle state, timing policies, and service-managed metadata on existing self-tests.
It rejects conflicting suite ownership and test sources.
Each invocation publishes a new immutable release and explicitly starts a selected run.
Reusing the installation preserves comparable timing history across source releases.
The standard UI can inspect and rerun these tests after registration.

## Verify failure handling

Run the intentional assertion failure separately:

```sh
bash scripts/test-self.sh --diagnostics
```

Run the intentional timeout separately:

```sh
bash scripts/test-self.sh --timeout-probe
```

The assertion probe requires a failed result. The timeout probe requires a separate timed-out result and count.
Both probes retain the expected diagnostic reason and media.
They do not add intentional failures to the normal workflow suite.
The adapter also has focused Docker checks for timeout, failure, real test-start events, and cancellation cleanup:

```sh
teststudio_selftest_image=$(docker image inspect teststudio/selftests:local --format '{{.Id}}')
TESTSTUDIO_PLAYWRIGHT_IMAGE="$teststudio_selftest_image" go test -race ./adapters/execution/localcontainer -run TestContainer
```

## Worker contract

The local worker uses one permitted image and one discovered permanent test tag for each attempt.
The reporter requires exactly one selected test. TestStudio owns retries.
A missing report, mismatched identity, or unexplained failed worker exit cannot produce a passing result.
Framework duration excludes container and service preparation.
A real test-start event changes preparation into execution.
Timing baselines use a separate Playwright runtime profile.

Results retain assertion outcomes, runner status, expected status, bounded steps, and attempt timing.
Screenshots, traces, video, browser diagnostics, and process output use the existing private evidence store and capture rules.
Worker output, file transfers, attachment counts, file sizes, and total evidence bytes have bounds.
The worker uses no host mounts. File transfer accepts regular files through a bounded archive reader.
The container has a read-only image, an unprivileged user, resource limits, and no external network.
Completion, failure, timeout, and cancellation remove the owned container and its installation-scoped output volume.
A worker-side deadline also bounds execution after a controller disconnect.

The separate [website profile](browser-execution.md) supports one registered local target with guarded browser and request fixtures.
Cloud Playwright execution and private trace viewing remain pending.

## Restart and cleanup recovery

The controller stores an installation identity and an execution reference before launching a container.
Containers and named output volumes carry installation, execution, run, test, attempt, and worker labels.
Normal execution saves the observation and available evidence before removing these resources.
A failed checkpoint leaves resources available for recovery.

A replacement controller marks expired worker leases incomplete and reconciles their references.
It stops abandoned browser work, imports available diagnostics, and removes only resources with matching ownership labels.
Recovery retains the original run outcome. It cannot turn an incomplete run into a passing run or launch a replacement test.
Healthy worker leases remain active. Another installation's resources remain untouched.
Recovery claims prevent concurrent controllers from processing the same reference.
Inventory checks find late resources associated with completed references.
Cleanup and transient evidence failures retain retry state across restart.
Capture retries have a three-attempt limit. Permanent gaps remain explicit after resource removal.

Worker cleanup remains available when new Playwright dispatch is disabled.
The authenticated `/api/v1/projects/{projectId}/runs/{runId}/executions` route exposes paginated status and recovered observations.
Internal claim tokens stay outside API responses.

Verify a hard controller stop and restart against an existing self-test image:

```sh
go build -o bin/teststudio ./cmd/teststudio
teststudio_selftest_image=$(docker image inspect teststudio/selftests:local --format '{{.Id}}')
TESTSTUDIO_SELFTEST_IMAGE="$teststudio_selftest_image" TESTSTUDIO_RECOVERY_SERVER=bin/teststudio node scripts/test-worker-recovery.mjs
```

The check kills only the controller that it starts.
It waits for the normal worker lease to expire and restarts the same installation with dispatch disabled.
It verifies an incomplete outcome, recovered diagnostics, and removal of the owned container and volume.
It restarts the controller again with dispatch disabled, downloads private evidence, and compares its size, checksum, and bytes.
Run, attempt, and attachment identities remain unchanged. Only one logical run exists.
The product-owned runner diagnostics provide exact ADAPTER-FAIL and ADAPTER-TIMEOUT selectors.
They use fresh in-memory pages and no external target.
The check stops its replacement controller and preserves verification data under `.data/worker-recovery-*` and `output/worker-recovery/`.


## Managed provider and OIDC workflow

Run the provider workflow through the normal publication and execution path:

```sh
bash scripts/test-self.sh --test=STUDIO-PROVIDERS-001
```

The isolated test wrapper hosts two new loopback issuers.
They use discovery, RSA-signed tokens, one-time codes, PKCE, and a separate approval form.
A private client checks an operator-approved server credential. The public client uses no client secret.
The credential exists only in the wrapper and its owned Go process environment.
The product server exposes no fixture routes.

The browser registers both providers through the UI.
It creates a provider-owned project group, configures exact claims, and verifies real project access.
Equal emails and subjects at different issuers produce separate users. The second user receives no transferred project access.
Provider edits invalidate earlier sessions and memberships. Fresh validated sign-in restores the permitted group path.
Self-provider edits refresh public sign-in choices after authentication expires.
The workflow also checks conflicts, unconfirmed saves, revoked management, retained read-only views, help, and both themes.
The wrapper and all fixture state end with the owned test container.

## Configuration wizard checks

STUDIO-WIZARD-001 covers first-project setup, generated identifiers, preserved entries, and reviewed HTTPS creation.
It checks hidden invalid options, mobile layout, partial setup recovery, and duplicate run prevention after an interrupted response.
STUDIO-APPS-001 covers the shared registration wizard and locked application association.
STUDIO-HTTPS-001 covers advanced assertions and retained diagnostics.
STUDIO-CATALOG-001 and STUDIO-SELECTIONS-001 cover reviewed normal runs and saved selection previews.
Select only these workflows for wizard changes. Cloud verification remains separate.

## Ordinary project creator workflow

STUDIO-OWNER-001 starts from an isolated identity with only project creation authority.
It completes project, application, target, and suite setup through the UI.
It creates and publishes a native check, runs it locally, and adds an existing project reviewer.
It verifies managed project assignments, limited project discovery, and the absence of installation authority.
The fixture session is private to its disposable local TestStudio instance.
Use the targeted selector after changing initial project access or its guided setup.

## Recorded website execution checkpoint

A fresh native Playwright recording supplies the start navigation for the local website workflow.
The native recorder check verifies that all 19 owned processes stop. It preserves the generated source as private verification evidence.
The reviewed draft adds independent assertions, a local button interaction, a direct API request, and an off-site redirect check.
A separate preparation worker assembles and discovers the saved source. Normal publication produces the immutable browser artifact.
The normal run service selects its registered environment and executes one logical test through the Unix-socket gateway.
The successful result contains 23 steps, a 157 ms test duration, complete logs, one screenshot, and one trace.

The check names are TestRecordingLocalPlaywrightWritesSourceAndClosesOwnedBrowsers and TestRecordedWebsiteBuildsPublishesAndRunsThroughGuardedGateway.
The second check accepts TESTSTUDIO_NATIVE_RECORDING_SOURCE, TESTSTUDIO_BROWSER_GATEWAY_IMAGE, TESTSTUDIO_BROWSER_RUNTIME_IMAGE, and an optional TESTSTUDIO_BROWSER_OUTPUT directory.
Use the newly retained recording JSON as the source. Keep runtime and gateway references immutable.
Unit checks cover HTTP forwarding, HTTPS tunneling, Playwright HTTP tunneling, denied origins, redirects, target admission, and changed policy between attempts.
STUDIO-CREATE-001 covers the reviewed target expression and preparation action in the application.
STUDIO-WIZARD-001 covers run review with the registered endpoint list.
Evidence is output/recorded-website/checkpoint.json and output/selftest/20261104T005645.164758000-7c3098f80687727af244e78a/summary.json.
The corresponding cleanup records verify that owned controllers, containers, and volumes stop.
These checks use local fixtures. They make no public website or cloud verification request.

## Tester investigation and failure recovery

Run only the affected workflow:

```sh
bash scripts/test-self.sh --test=STUDIO-DIAGNOSTICS-001
```

The test creates new private browser media through an owned helper process. That process closes its browser before returning.
An isolated fixture helper writes failure and timeout observations through product metadata and evidence contracts.
Those observations verify presentation. They do not claim execution of a prepared browser suite.
The workflow inspects failed steps, selects retry-specific logs, opens a private screenshot, and checks expired, missing, and failed-upload states.
It also executes a real native HTTP timeout, loses the response after rerun creation, and retries the same in-page request.
The existing child returns without another run. Refresh preserves the error and selected scope. Permission errors hide rejected content.
The mobile check measures step readability and select bounds, alongside document overflow and keyboard focus.

The separate ADAPTER-NOMEDIA probe runs a passing check with browser media disabled in its source configuration.
TestContainerMissingRequestedMediaPreservesPassingObservation requests screenshots and traces, then verifies two capture gaps without changing the pass.
Select that Go check with TESTSTUDIO_PLAYWRIGHT_IMAGE set to the cached immutable self-test image.
The reporter unit check is `node --test runners/playwright/reporter.test.mjs`.
All probes use local fixtures. The checkpoint records selected backend checks and exact cleanup evidence.

## Private trace workflow

Run `bash scripts/test-self.sh --test=STUDIO-TRACE-001` after private trace viewer changes.
The focused workflow uses fresh browser captures and fixture-backed failed-attempt metadata in its disposable installation.
It checks attempts, recorded steps, execution context, logs, screenshots, a JSON attachment, and the installation-hosted viewer.
The original fixture website stops before inspection. The viewer must replay recorded CSS and images from the archive.
The check expires the real server grant, refreshes access, requests a separate download, and verifies keyboard return focus.
It permits no external network responses. Its isolated browser and controller stop with normal self-test cleanup.
This fixture does not replace the separate real prepared-suite execution checks.

## Catalog browsing workflow

Run `bash scripts/test-self.sh --test=STUDIO-BROWSE-001` after catalog browsing or row-action changes.
The workflow creates new HTTP definitions through normal APIs in its disposable project.
It covers nested and sibling categories, combined metadata filters, readable application names, sorting, and preserved hidden selections.
It also covers duplicate, enable, disable, archive, restore, revision history, keyboard category disclosure, and mobile themes.
This workflow does not execute the HTTP definitions. Execution and publication retain their existing focused workflows.

## Reviewed publication workflow

Run `bash scripts/test-self.sh --test=STUDIO-PUBLISH-001` after reviewed publication changes.
The workflow publishes and executes a new local HTTP check before editing its working definition.
It reviews changes and pinned or unpinned schedules, rejects a concurrent catalog edit, and creates a fresh review.
It simulates a lost response after publication commits, then verifies same-key recovery without another release.
It also checks the preserved earlier run, mobile layout, and publication without activation.
Schedules remain disabled. No hosted services or cloud verification run.

## Paged catalog and transfer workflow

Run `bash scripts/test-self.sh --test=STUDIO-TRANSFER-001` after catalog paging or transfer changes.
The workflow creates 35 new native checks through normal APIs.
It verifies paging, server search, preserved selection, stale cursor rejection, and explicit reload.
Export must include unloaded records and produce a real JSON download.
The browser loads a new file, rejects invalid assertions, handles a stale import atomically, reloads without losing the document, and reviews again.
It then imports and publishes without activation. The existing run and active release remain unchanged.
Use STUDIO-BROWSE-001 for category navigation, combined filters, lifecycle actions, keyboard behavior, and themes.

## Retained test history

Run STUDIO-HISTORY-001 for the [retained history workflow](test-history.md).
It creates two fresh native HTTP runs and separate diagnostic screenshot and trace attachments.
The media helper closes its owned browser before returning.
A selector-scoped fixture can remove the archived working definition inside that disposable database.
The production service exposes no corresponding deletion endpoint.
The workflow checks saved snapshots, current permissions, retained private evidence, help, focus, themes, and mobile width.
STUDIO-BROWSE-001 covers the shared revision renderer and renamed Revisions action.

## Upload finalization checks

Focused evidence recovery checks interrupt the request after durable bytes are written and before its manifest commits.
They close and reopen database and artifact handles, then recover without uploading the same bytes again.
Additional cases cover missing content, corrupt bytes, transient reads, active leases, bounded pages, and older pending manifests.
Execution checks hold a duplicate upload open. Its checkpoint remains until the attachment becomes available.
These checks require no browser build or cloud account.

## Artifact storage contracts

Run the affected local contracts with this command:

```sh
go test -race ./adapters/artifacts/... ./internal/artifacts -count=1
```

S3 and Azure cases use SDK clients against in-process HTTPS fixtures with test-only identities.
The fixture owns no cloud resource and uses no configured account credentials.
It closes its HTTP servers when the check finishes. No browser starts.
The common lifecycle check can use the local PostgreSQL helper through the existing test database setting.
See [Artifact storage](artifact-storage.md#verification-status) for verified behavior and provider acceptance limits.

## Coded timing and logical counts

The self-test image carries an explicit runtime label and target architecture.
Unchanged source-only releases can retain compatible timing history.
STUDIO-API-STEPS-001 registers an API-typed Playwright test with several API operations and nested assertions.
Its reporter retains those operations as steps within one registered logical result.
STUDIO-TIMEOUT-001 verifies timeout counts, attempts, and failure reruns through the normal application.
The registrar respects an explicit API test type and defaults other workflows to browser.

Focused Go runner checks execute a new aggregate and an independently selected subtest.
Package and ancestor events remain diagnostics. They do not create additional logical test results.
Coded execution checks cover passing retries, step evidence, unavailable measurements, valid zero durations, and changed runtimes.
See [Test timing](timing.md) and [Results](results.md) for their contracts and remaining deployment scope.

## Installation transfer

Run focused SQLite archive and operator command checks with this command:

```sh
go test -race ./internal/transfer ./cmd/teststudio-data -count=1
```

Run the cross-database scenario with the local PostgreSQL helper:

```sh
bash scripts/test-postgres.sh ./internal/transfer -run TestSQLiteToPostgreSQLInstallationTransfer
```

The fixture creates a new project, access policy, local credential, session, completed HTTP run, and pinned artifact.
It verifies raw metadata, effective access, preserved native grant identifiers, sequence continuity, and retained bytes.
The reverse transfer also preserves metadata and access. A damaged archive leaves the destination empty.
No browser or cloud account is required. See [Database transfer](data-transfer.md) for operator cutover steps.
