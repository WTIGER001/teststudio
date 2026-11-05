# Test timing

Each attempt records UTC start and finish times and measured execution milliseconds.
Native HTTP uses monotonic elapsed time. Go and Playwright use their framework execution durations.
Go uses valid selected-test event timestamps when its framework rounds elapsed time to zero.
The source-v2 runtime preserves that precise measurement without replacing genuine zero durations or positive reported durations.
The final logical test also records total execution milliseconds across its attempts.
Queue delay, evidence writes, and worker coordination do not enter this measurement.
Interrupted attempts retain elapsed time. Their timing assessment has a censored status.
Skipped tests have no measured execution duration.
Timeouts use the separate `timed-out` result state and `timedOut` count.
Their timing stays censored. Rerun failures includes them alongside failed and flaky results.
The complete status contract is in `docs/results.md`.

## Comparable history

A baseline belongs to a project, suite, permanent test identifier, environment, execution location, runner, and runtime profile.
Definition and target fingerprints separate changed assertions, endpoints, and credential references.
Display labels, catalog releases, source revisions, timing rules, and timeout limits do not reset a baseline.
Changed runtime measurement or capture behavior requires a new runtime identity. Older observations remain in their original baseline.
Fingerprints store no resolved credential values.

The default window uses the latest 50 clean observations within 30 days.
A clean observation passes on its first attempt and stays below the known slowdown threshold.
Failed, retried, interrupted, and slow observations do not train the baseline.
Each observation and its final result commit in one transaction.
Concurrent workers retry revision conflicts without losing observations.
The store retains at most 1000 observations per context.

## Slowdown rules

Project defaults flow into suite drafts. Suite defaults override project defaults.
Each test can override the complete timing policy.
Publication freezes these rules. Each new run freezes its own prediction and thresholds before queueing.

The default rule requires five comparable observations before enforcement.
The slowdown threshold uses the largest value from these three calculations:

- Twice the median execution duration.
- The median plus 100 milliseconds.
- The historical 95th percentile.

The absolute increase and historical tail reduce failures from small timing changes.
The policy supports `off`, `warn`, and `fail` modes.
The default `fail` mode converts a passing functional result into a duration regression failure at or above the threshold.
The result retains its original functional state in `observedState`.
A functional failure retains its original failure kind. Timing remains a separate assessment.
A passing retry remains flaky. The result retains the count of slow attempts.

## Timeouts and estimates

Cold and warming tests use the catalog timeout ceiling.
Ready tests use an adaptive timeout by default.
The recommendation uses three times the historical 95th percentile, plus 250 milliseconds, with a minimum of one second.
The service rounds the recommendation up to whole seconds and caps it at the catalog timeout ceiling.
A fixed timeout policy keeps that ceiling while exposing the recommendation.
These limits constrain each attempt. The overall run deadline remains independent and includes queue time.

Estimates report baseline coverage, expected execution time, a high execution estimate, and the sum of attempt timeout budgets.
Expected execution time sums test medians. The high estimate sums historical 95th percentiles.
Neither estimate includes queue delay, preparation, evidence capture, or retry execution.
The timeout budget includes all permitted retries.
An incomplete history leaves the aggregate prediction unavailable. Partial observations remain visible per test.
A budget warning identifies an overall run deadline below the attempt timeout budget.
Estimation uses the same admission and dependency selection as run creation. It creates no run or audit event.

## API

Use these routes below `/api/v1/projects/{projectId}`:

| Route | Method | Operation |
| --- | --- | --- |
| `/runs/estimate` | POST | Predict timing for a normal run request. Requires `runs.create`. |
| `/timing-baselines/{id}` | GET | Read the current baseline and revision. Requires `runs.read`. |
| `/timing-baselines/{id}/reset` | POST | Clear samples with a reason and `If-Match`. Requires `catalog.publish`. |

A reset increments the baseline generation and records an audit event.
Previously queued runs keep their original criteria. They cannot repopulate a reset baseline.
A reset does not change historical results or their frozen predictions.

## Interface

Use Estimate timing on the run form before starting a run.
Changing the request clears its estimate. Starting the run freezes a new prediction from current history.
The run detail page shows the frozen prediction, actual duration, slowdown ratio, and functional result.
Attempt history retains separate retry durations. Final results also show the sum across attempts.
Live remaining execution stays unavailable when history is missing or the current attempt exceeds its typical duration.
Worker elapsed time includes coordination and evidence capture. It does not replace per-test execution timing.

The catalog editor supports timeout ceilings and complete timing policy overrides.
Invalid sample windows block saving. Publish a new release to activate changed timing rules.
Use the run's timing history panel to inspect the current baseline and its revision.
An authorized publisher can reset history with an explicit reason.
The panel confirms the new generation. Historical predictions and results remain visible.

## Playwright timing

The isolated local Playwright profile uses framework durations and a separate runtime context.
Container startup and service preparation do not train the execution baseline.
A reporter event identifies the actual test start for progress updates.
Adaptive timeouts constrain the framework test. A separate bounded worker deadline includes preparation and collection.
The overall run deadline still limits the complete request.
Playwright failures, retries, and interruptions follow the same baseline training rules as native checks.

A missing worker report sets `durationUnavailable`. The interface shows Duration unavailable instead of zero milliseconds.
Measured zero durations remain valid. Worker coordination time stays separate in `workerDurationMs` when available.
Recovered observations retain the original run outcome and do not train timing baselines.

## Go timing

The local Go profile uses structured test durations and its image runtime label.
Compilation duration stays separate from test execution.
Source image revisions preserve comparable history when the runtime label and test definition remain compatible.
Toolchain and platform changes require a new runtime label.
Panics, timeouts, unknown observations, and interruptions remain censored.
Build and discovery problems retain unavailable test durations and do not train history.
See [Local Go suites](go-testing.md) for selectors, diagnostics, and local verification.

## Runtime compatibility and zero measurements

Go, browser, and API tests use the same frozen prediction, adaptive timeout, slowdown assessment, and clean-history rules.
Before each attempt and managed dispatch, the service checks the current runtime against the run's frozen timing profile.
A changed or unavailable runtime blocks execution before another test starts.
Create a new run to review the current runtime's history. Existing predictions remain unchanged.
Recovered diagnostics do not dispatch tests and retain their separate recovery rules.

Known coded images use their teststudio.runtime label to separate toolchains, platforms, and execution flags.
The inspected operating system and architecture also enter the context, independently of the declared label.
Source-only image updates can share history when that label and the test context remain compatible.
Explicitly approved Playwright images without a label use their immutable image digest as a separate timing context.
They cannot share unproven compatibility across image changes. Go images still require a valid runtime label.
The product's self-test image declares its pinned framework, base image identity, and target architecture.

Duration unavailable stays censored even when the functional outcome is passed.
It does not train a baseline or appear as a measured zero.
A recorded zero duration remains valid. It can train history when the observation meets the normal clean-pass rules.
A zero median uses the configured absolute noise guard and historical tail. Its slowdown ratio remains unavailable.
If the absolute guard is zero, another measured zero remains normal. A positive duration can still exceed that threshold.
Regression messages show the baseline milliseconds instead of inventing a ratio or dividing by zero.

Focused checks cover Go, Playwright, and API-typed coded results, retries, worker overhead, runtime drift, and zero observations.
They run on SQLite and local PostgreSQL.
Cross-deployment timing acceptance remains open until provider profiles pass their separate verification.
