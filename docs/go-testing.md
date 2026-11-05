# Local Go suites

## Introduction

The local Go worker compiles source tests inside an immutable container image.
Unit suites need no target, browser, network access, or application credentials.
The installation administrator enables exact images through `--go-images`.
Publication and execution use the normal catalog, release, run, timing, and evidence APIs.
Registered HTTP API targets and credential bindings use the separate go-environment profile described below.
Hosted Go execution uses the protected provider profiles. Non-HTTP integration bindings remain pending.
See [Kubernetes workers](kubernetes-workers.md) for the actual local cluster verification and remaining provider scope.
Prepared manifests can now declare a module root and catalog for isolated discovery and normal execution.
See [Prepared suite inspection](prepared-suites.md) for that contract and its local CLI.

## Catalog and selectors

For environment-backed API checks, use [Guarded Go API execution](#guarded-go-api-execution).

Put suite source and module metadata under `/suite` in the image.
Preload dependencies in the image module cache and set `GOMODCACHE` to its path.
Dependencies must be available before execution. The worker disables module and toolchain downloads.
Place this manifest at `/suite/teststudio-go.json`:

```json
{
  "schemaVersion": "teststudio.go.catalog/v1",
  "tests": [
    { "id": "GO-SUM-001", "package": "./unit", "test": "TestSum" },
    { "id": "GO-CASE-001", "package": "./cases", "test": "TestRows/first" }
  ]
}
```

Each catalog entry uses runner `go` and type `unit`, `integration`, or `api`.
Its selector joins the explicit package and test path with `::`.
The first manifest entry uses `./unit::TestSum`.
The immutable artifact's discovered selector map must match the entry.
The worker also verifies the selected identifier and selector against the image manifest.

Register a parent aggregate or its independent child tests.
Overlapping registrations within the same package block publication.
Sibling child registrations remain independent logical tests.
Dynamic descendants appear as recorded steps. Package events do not create extra logical results.
The worker uses anchored selection patterns for each subtest path segment.
Recursive packages, parent traversal, arbitrary commands, and injected flags remain invalid.

## Runtime and timing

Package preparation runs `go test -c -mod=readonly` with a separate 45-second preparation limit.
The binary runs from its package directory through `go tool test2json`.
Execution uses `-test.v=test2json`, `-test.count=1`, an exact selection, and the frozen effective timeout.
Test execution never uses cached test results. Compilation may use an available build cache.
Go's [test2json contract](https://pkg.go.dev/cmd/test2json) defines the event stream.
The [test flags](https://pkg.go.dev/cmd/go#hdr-Testing_flags) define selection, count, and timeout behavior.

Test duration excludes compilation and container preparation.
When Go rounds its reported duration to zero, valid selected-test event timestamps preserve the measured sub-millisecond interval.
An equal start and finish time remains zero. Positive framework durations remain unchanged.
The selected test's final event supplies its finish timestamp.
The source-v2 Go runtime identifies this timing change and starts a separate comparable baseline.
Rebuild and publish the trusted runtime, then prepare a new suite release to use it.
Retained runs keep their original runtime and observations.
Recorded diagnostics retain build duration, toolchain, platform, process exit codes, and output truncation.
Exit codes remain unavailable when their process never starts.
A start event moves live progress from preparation to execution.

Set the image label `teststudio.runtime` to identify the toolchain, platform, and execution flags.
The example image uses `go1.26.8-linux-arm64-cgo0-source-v1` on ARM64.
Keep this label stable across compatible source revisions.
Change it when toolchain, architecture, compilation flags, or execution semantics change.
The image identifier remains frozen in the release. It does not reset comparable timing history by itself.

## Results and evidence

Assertion failures use `failed` with reason `assertion`.
Panics use `failed` with reason `panic`. Their duration observations remain censored.
Timeouts use `timed-out` with reason `timeout` and a separate count.
Failure reruns include timed-out results. A successful retry produces a flaky logical result.
Runtime skips use `skipped` without assertion retries.

Build failures and missing selected tests retain unknown attempts with distinct diagnostic reasons.
These errors block the run. They do not trigger assertion retries or train duration baselines.
An all-skipped run keeps an incomplete, unknown outcome.
Missing, duplicate, corrupt, and incomplete observations cannot produce passes.

Build output, structured test events, and standard error use private evidence storage.
Each output stream retains up to 2 MiB. Parsing continues after raw capture reaches its limit.
An oversized event line or an incomplete event stream prevents a passing observation.
The runner retains at most 500 recorded steps.
Capture settings and normal evidence limits still apply.

## Isolation and recovery

The Go profile shares the local container transport with Playwright.
It uses no host source mounts, external network, or production credentials.
The container runs as an unprivileged user with a read-only root filesystem and bounded resources.
Temporary storage permits compiled test execution and remains private to the container.
Writable cache and temporary files stay within that bounded storage.

Each attempt checkpoints its result and evidence before cleanup.
Recovery uses the profile in the durable execution record, even when new Go dispatch is disabled.
An interruption retains uncertainty. Recovery captures available observations without rerunning tests.
Ownership checks protect other installations and runner profiles from cleanup.

## Verify locally

Run the new source suites through a separate local TestStudio controller:

```sh
bash scripts/test-go-self.sh
```

The script builds the runner and image, publishes its manifest, and executes normal and diagnostic selections.
It checks subtest counts, retained metadata, clean timing learning, retries, timeout reruns, and preparation failures.
It stops its controller and verifies worker resource cleanup.
The retained installation stays under `.data/go-verification.*`.
Its summaries stay under `output/go-selftest`.
Diagnostic packages intentionally fail or do not compile. Their nested module excludes them from the product test suite.

Run focused container checks with the built image:

```sh
teststudio_go_image=$(docker image inspect teststudio/go-selftests:local --format '{{.Id}}')
TESTSTUDIO_GO_IMAGE="$teststudio_go_image" go test -race ./adapters/execution/localcontainer -run TestGoContainer
```

## Frozen timing compatibility

Queued runs retain the runtime profile used for their timing prediction.
The service blocks dispatch if that profile changes before an attempt. Start a new run for the new profile.
Framework measurements stay separate from compilation and worker coordination.
Measured zero durations can train history. Missing measurements remain censored.
See [Test timing](timing.md#runtime-compatibility-and-zero-measurements) for the shared rule.

## Guarded Go API execution

Use runner go, type api, and buildProfile go-environment for registered HTTP API targets.
The complete new example lives in examples/go-api. It uses only the Go standard library.
Copy it to a new suite workspace, set the manifest suiteId to the destination suite, and review its catalog and policy.
Upload that complete ZIP through Create test → Go suite or Import and sync tests.
Prepare, review, publish, and run through the normal source workflow. The example initially retains needs-review safety.
Unit suites keep go-module and require no target or artificial API endpoint.

The manifest's browser object is the existing shared endpoint-policy field. Its name remains compatible with published Playwright source contracts.
For Go API suites, browser.requests is mandatory. Declare the registered endpoint, allowed methods and path prefixes, and request and response limits.
The server validates the source manifest, discovered catalog, and selected environment before dispatch.
The worker compares the frozen policy and authentication with the prepared source again before compiling or running a selected test.
Policy changes require a new prepared release. They cannot alter an already queued run.

Standard net/http clients use the local relay address supplied in TESTSTUDIO_API_BASE_URL or TESTSTUDIO_BASE_URL:

```go
client := &http.Client{Timeout: 5 * time.Second}
response, err := client.Get(os.Getenv("TESTSTUDIO_API_BASE_URL") + "/health")
```

Close response bodies and idle client connections. Use relative paths appended to the supplied relay address.
The selected test process receives the address after compilation. Proxy environment variables are cleared for that process.
The relay forwards to the owned Unix gateway. The worker keeps network=none, its read-only image, and existing resource limits.
It cannot reach a registered target directly. CONNECT, off-origin requests, unsafe methods, oversized bodies, and excess requests remain blocked.
Redirect statuses remain visible, but the gateway withholds Location and the original redirect body.
General Go source remains trusted published code. The network boundary enforces access; a library wrapper is not its sandbox.

For authenticated tests, declare the matching API binding in the manifest and catalog.
The gateway resolves the environment reference and injects the reviewed header. Go source and process input receive no resolved secret.
Protected environments still require safe selections, safe dependencies, protected execution authority, and reviewed read-only rules.
See [API credentials](api-credentials.md) and [Request policies](api-testing.md).

The profile retains ordinary Go build diagnostics, exact selectors, uncached execution, root and child steps, and framework measurements.
Build and coordination durations remain separate from measured test duration. Child events never add logical test counts.
Manual and scheduled execution produce the same result, evidence, timing, history, and trend contracts.
Go event logs use application/x-ndjson. The collector decodes event output before redacting sensitive fields and reflected values.
Invalid or truncated event streams produce an explicit evidence gap. Test outcomes and durations remain unchanged.

Build the updated Go source runtime once with scripts/build-go-runtime.sh.
Configure teststudio-prepare with --go-runtime and its immutable image identifier.
Enable source bundles or approved prepared Go images on the API, and configure its --browser-gateway-image.
The gateway flag names the shared environment transport for both Go and Playwright.
Environment Go execution uses go-local-environment-v1. Offline Go retains go-local-isolated-v1.
The updated API runtime advertises teststudio.api-policy=request-policy/v1 and teststudio.work-item-bytes=4194304.
The gateway advertises teststudio.gateway-config=private-config/v1. Older images fail admission with an explicit upgrade requirement.

For the demo, pass --go-runtime with the already built immutable image to node scripts/demo.mjs start.
The launcher retains that choice for later starts and reuses the image. It does not rebuild it or reseed existing data.

Run only the focused real worker check after changing this profile:

```sh
TESTSTUDIO_GO_RUNTIME_IMAGE="$teststudio_go_image" \
TESTSTUDIO_BROWSER_GATEWAY_IMAGE="$teststudio_gateway_image" \
go test ./internal/sources -run '^TestGuardedGoAPIWorkspaceBuildsPublishesAndRunsManuallyAndOnSchedule$' -count=1
```

Set both variables to immutable images already available locally. No cloud account or public target is needed.
The fixture checks authenticated manual and scheduled runs, large reviewed policies, denied bypasses, private evidence, history, trends, and owned-resource cleanup.
