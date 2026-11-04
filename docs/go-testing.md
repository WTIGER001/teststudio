# Local Go suites

## Introduction

The local Go worker compiles source tests inside an immutable container image.
Unit suites need no target, browser, network access, or application credentials.
The installation administrator enables exact images through `--go-images`.
Publication and execution use the normal catalog, release, run, timing, and evidence APIs.
External targets, dependency bindings, hosted Go workers, and source preparation screens remain pending.
Prepared manifests can now declare a module root and catalog for isolated discovery and normal execution.
See [Prepared suite inspection](prepared-suites.md) for that contract and its local CLI.

## Catalog and selectors

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
