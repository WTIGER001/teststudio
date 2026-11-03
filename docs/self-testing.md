# TestStudio tests itself

TestStudio owns a new Playwright suite in `tests/selftest/playwright`.
The suite runs through the normal catalog, publication, execution, timing, and evidence APIs.
It copies no tests or configuration from another application.

## Run locally

Install Go, Node, npm, and Docker. Start the Docker daemon.
Run this command from the repository root:

```sh
bash scripts/test-self.sh
```

The script builds the current Go service and Angular UI.
It packages them with the pinned Playwright runtime and the new browser tests.
An isolated controller registers the catalog, publishes an immutable local image ID, and starts a normal TestStudio run.
Each test container starts a new TestStudio instance with its own SQLite database.
The instances use loopback endpoints inside the container. They have no external network access.
The suite tests themes, mobile layout, page help, catalog publication, HTTP assertions, native execution, timing, baseline reset, timeout reruns, and trends.
It also tests durable schedules, completion notifications, identity administration, scoped groups, and session controls.
Native check runs remain separate from the outer Playwright run. They cannot recurse into browser self-tests.

The script stops its controller after verification. It preserves its database and private artifacts under `.data/selftest-verification.*`.
It saves a local result summary and downloaded screenshots and traces under `output/selftest/{runId}`.
The printed installation path contains `installation.json` with its origins and immutable image reference.
No cloud account or remote deployment is required.

Use `--test=ID` to execute one normal workflow with its usual publication, execution, and evidence checks:

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
The preparer creates a project administrator and provider-managed membership with a fresh opaque session.
The browser checks installation and foreign-project rejection, hidden installation navigation, and read-only external membership.
Dedicated OIDC tests verify provider sign-in separately.

The preparer runs only inside the isolated self-test image. Its local fixture route belongs to the test wrapper.
The product HTTP server provides no fixture preparation route.
The wrapper awaits its owned service and helper processes during shutdown.
The normal worker cleanup removes the test browser container and its output volume.

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

External target networking, cloud Playwright execution, source preparation screens, and private trace viewing remain pending.

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
The check stops its replacement controller and preserves verification data under `.data/worker-recovery-*` and `output/worker-recovery/`.
