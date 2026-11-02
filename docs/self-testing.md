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
The suite tests theme persistence, mobile layout, catalog publication, native HTTP execution, timing history, and baseline reset.
Native check runs remain separate from the outer Playwright run. They cannot recurse into browser self-tests.

The script stops its controller after verification. It preserves its database and private artifacts under `.data/selftest-verification.*`.
It saves a local result summary and downloaded screenshots and traces under `output/selftest/{runId}`.
The printed installation path contains `installation.json` with its origins and immutable image reference.
No cloud account or remote deployment is required.

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

These commands succeed only when TestStudio retains a failed result with the expected failure kind and diagnostic media.
They do not add intentional failures to the normal workflow suite.
The adapter also has focused Docker checks for timeout, failure, real test-start events, and cancellation cleanup:

```sh
teststudio_selftest_image=$(docker image inspect teststudio/selftests:local --format '{{.Id}}')
TESTSTUDIO_PLAYWRIGHT_IMAGE="$teststudio_selftest_image" go test -race ./adapters/execution/localplaywright -run TestContainer
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
Completion, failure, timeout, and cancellation remove the owned container and its anonymous output volume.
A worker-side deadline also bounds execution after a controller disconnect.

External target networking, cloud Playwright execution, source preparation screens, orphan reconciliation after abrupt process loss, and private trace viewing remain pending.
