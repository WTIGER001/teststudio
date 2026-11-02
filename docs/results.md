# Test results and timeouts

## Test status

Attempts and final logical test results use these states:

| State | Meaning |
| --- | --- |
| `passed` | The first attempt completes successfully. |
| `failed` | A check or assertion fails. Duration regression failures also use this state. |
| `timed-out` | The attempt exceeds its test or worker time limit. |
| `flaky` | A later attempt passes after an unsuccessful attempt. |
| `skipped` | The test does not execute because a dependency does not pass. |
| `unknown` | The worker cannot prove a complete result. |

Timeout results retain `failureKind: timeout` for diagnostic compatibility.
The state identifies the timeout separately from assertion failures.
HTTP and Playwright attempts use the same status contract.

Run counts expose `timedOut` separately from `failed`.
A timed-out logical test contributes one completed result, even when several attempts time out.
Each attempt remains in attempt history. Retry attempts do not increase final test counts.
A successful retry contributes to `flaky`, including a retry after a timeout.

## Run state and outcome

Run state describes execution progress. Run outcome summarizes logical test results.
A completed run can have a `timed-out` outcome.
If a completed run contains failures and timeouts, its outcome is `failed`. Its counts show both statuses.
Timeouts take priority over flaky results when no assertion failure exists.

An overall run deadline sets run state and outcome to `timed-out`.
The active test retains its timeout result when the worker can save it.
Selected tests that never dispatch do not receive invented timeout results.
Cancellation retains an unknown outcome. A lost worker remains incomplete with an unknown outcome.

## Rerun failures

Rerun failures selects failed, timed-out, and flaky logical tests from a finished run.
The new run retains the parent reference and uses the parent's published release.
Normal selection, dependencies, permissions, target protection, and admission checks still apply.
The original run and its evidence remain available.

Older results recorded as `failed` with `failureKind: timeout` appear as `timed-out` in API and UI reads.
Their displayed counts and outcomes follow the same rules as new results.
Reading these records does not rewrite stored history.
Failure reruns accept these older timeout records. Unknown interruptions remain unknown.

## Timing and evidence

Timeout measurements are censored. They do not train comparable duration baselines.
A slowdown failure remains distinct from a timeout.
Timeouts still collect screenshots, traces, and other evidence configured for unsuccessful attempts.
Timeout evidence also uses the configured failure retention extension.
Capture gaps remain separate from the observed test status.

## Local verification

Run the timeout status and failure-rerun workflow through TestStudio:

```sh
bash scripts/test-self.sh --test=STUDIO-TIMEOUT-001
```

Run the intentional Playwright timeout probe separately:

```sh
bash scripts/test-self.sh --timeout-probe
```

The probe succeeds when the retained test status is timed-out and its diagnostic media remains available.
