# Test results and timeouts

## Test status

Attempts and final logical test results use these states:

| State | Meaning |
| --- | --- |
| `passed` | The first attempt completes successfully. |
| `failed` | A check or assertion fails. Duration regression failures also use this state. |
| `timed-out` | The attempt exceeds its test or worker time limit. |
| `flaky` | A later attempt passes after an unsuccessful attempt. |
| `skipped` | A dependency prevents execution or the runner records an explicit skip. |
| `unknown` | The worker cannot prove a complete result. |

Timeout results retain `failureKind: timeout` for diagnostic compatibility.
The state identifies the timeout separately from assertion failures.
HTTP, Playwright, and Go attempts use the same status contract.

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

## Investigate one test

Select Inspect test beside a final result. The investigation heading receives keyboard focus.
Test evidence scope selects the test. Attempt evidence scope selects all attempts or one attempt.
These controls filter Attempt history, Run logs, and Run artifacts together.
All attempts is the default. It preserves unsuccessful attempts when a later retry passes.
Selected attempts and artifacts use readable cards. Failed recorded steps open automatically. Final logical counts stay unchanged.
Final result summaries stay compact. Failure details and recorded steps retain the complete diagnostic text.

Select View beside a trace to inspect recorded actions and page snapshots inside TestStudio.
The viewer uses the installation artifact origin. Refresh trace access renews authorization. Download trace requires its separate permission.
Configured JSON attachments open as bounded text previews. Other attachment types retain authorized downloads.
See [Private trace viewing](trace-viewer.md) for requirements and operator setup.

Select View beside a screenshot to open its private preview. The preview heading receives focus.
Close preview returns focus to its View button when that button remains available.
A scope change closes the old preview. Unavailable, expired, or missing files retain their state without unusable access buttons.
Advanced log filters contains manual stream and test filters. The manual test filter appears only when no shared test scope is selected.
Advanced execution events retains run-level context across the selected test and its attempts.
Refresh reloads evidence pages while retaining the selected test and attempt.
Load more attempts and Load more artifacts continue bounded scans when the matching evidence is on later pages.

An access error hides content from the rejected section. It does not mean that the run produced no evidence.
Refresh logs and Refresh evidence retry their respective reads.
A run-level access rejection also hides the previous run snapshot. The saved server records remain intact.

## Rerun failures

Rerun failures selects failed, timed-out, and flaky logical tests from a finished run.
The new run retains the parent reference and uses the parent's published release.
Normal selection, dependencies, permissions, target protection, and admission checks still apply.
The original run and its evidence remain available. The child page provides an Original run link.

A lost rerun response can leave a successfully created child run on the server.
Retry Rerun failures on the same page to reuse the request key. The server returns the existing child instead of creating another run.
The page keeps this error across its Refresh action. Permission errors provide a separate Refresh access action.
Refresh access reloads current project capabilities and the run without deleting saved records.
The in-page request key does not survive leaving or reloading the page. Check run history before starting another rerun after that change.

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

## Evidence query filters

The run prefix is `/api/v1/projects/{project}/runs/{run}`.

| Route | Filters |
| --- | --- |
| GET /attempts | Optional test identifier and attempt number. |
| GET /artifacts | Optional test identifier and attempt number. |
| GET /events | Optional test identifier and attempt number. |
| GET /logs | Optional test, attempt, and stream. Existing tail, limit, and cursor controls remain available. |

Use the query names `test` and `attempt`. An attempt requires a test identifier and a value from 1 through 3.
Omit attempt to include every retry. Invalid filters return a validation error.
Current run and evidence permissions apply before filtering. A filter cannot grant access to another project or run.
Filtered collection cursors bind the actor, project, run, collection, test, and attempt.
Log cursors also bind the selected attempt and retained log snapshot.
A collection request scans at most 200 metadata rows. A filtered page can be empty while its cursor permits another scan.
Collection limits remain 1 through 1000. Log pages remain limited to 100 segments with existing content bounds.
Changing the selection requires a new first page. Do not reuse a cursor after changing filters.
