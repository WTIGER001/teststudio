# Durable schedules

## Schedule definitions

Schedules belong to one project and select a published suite through the normal run request contract.
Each definition records its owner, enabled state, cadence, target, selected tests, release policy, time limits, and retries.
The creator or latest configuration editor becomes the execution owner.
Every firing checks the owner's current run permissions and target protection.
The worker repeats normal dispatch checks before each attempt.

Leave `releaseId` empty to follow the suite's active release at each firing.
Set `releaseId` to pin a published release.
The created run freezes its actual release, definitions, target, timing criteria, and capture settings.
Disabled tests, unsafe dependencies, missing bindings, and unavailable profiles block normal run admission.
The schedule records blocked firings with their reason. It tries again at its next due time.
The firing transaction guards the reviewed suite, release, environment, and selected-test revisions with its run and queue entry.
A concurrent configuration edit cannot commit a stale selection or a partial scheduled run.
Queued execution and retries apply the same current-policy checks as manual runs. See [Dispatch safety](results.md#dispatch-safety-and-frozen-runs).

## Shared execution and results

Manual and scheduled requests use the same admission, queue, worker, and completion contracts.
Supported local runners include native HTTP and HTTPS, targetless Go and Playwright, recorded websites, and guarded API suites.
Targetless suites need no environment. Target-backed suites require the same registered environment and current authority as manual runs.
Each firing freezes its actual release, target, capture, timeout, retry, and timing settings.
Prediction timestamps belong to each run. Baselines can advance between separate runs.

Attempts and child steps remain within one final logical result per selected test.
Timeouts retain their separate status. A successful retry produces one flaky result.
Schedules retain normal logs, screenshots, traces, history, trends, and explicit evidence gaps when the profile supports them.
The run records its schedule identifier, firing identifier, and due time for investigation.
Hosted profile verification remains separate from this local parity evidence.

## Cadence and overlap

Interval schedules accept 5 through 2592000 seconds and use UTC.
Five-field cron schedules use an explicit IANA timezone. UTC is the default.
Cron fields are minute, hour, day of month, month, and day of week.
The preview returns the next five times and a normal run timing estimate.
Cron parsing uses the pinned [robfig cron parser](https://pkg.go.dev/github.com/robfig/cron/v3).
Nonexistent daylight-saving times do not fire. A repeated local hour can contain two distinct UTC occurrences.

After downtime, one firing represents the earliest pending occurrence.
The scheduler advances directly to the next future time and records whether it coalesces missed occurrences.
Interval schedules preserve their original cadence phase.
The scheduler does not create a burst of historical runs after a restart.

The default overlap policy is `skip`.
It skips a firing while the previous run from the same schedule remains active.
The `allow` policy permits another run while that run remains active.
Other schedules and manual runs do not inherit this overlap rule.
Project and environment exclusivity controls remain separate implementation work.

## Claims and transactions

The service scheduler scans durable project work records. It requires no provider timer service.
Each scheduler claims an occurrence with an atomic revision check and a 30-second lease.
Expired claims can transfer to another scheduler instance.
One transaction saves the firing, advances the schedule, queues the normal run, and records its idempotency receipt.
Concurrent replicas cannot create duplicate logical runs for the same occurrence.
Disabling, archiving, or editing a claimed schedule changes its revision and prevents the old claimant from committing a run.
An interrupted transaction leaves no queued run or completed firing.

Public revisions identify configuration changes.
Internal revisions also protect claims and firing updates.
Frequent runtime updates do not invalidate an editor's configuration revision.
Competing configuration edits return a conflict.
History keeps triggered, skipped, and blocked firings with their scheduled time, observation time, and run reference.

## API

Use these routes below `/api/v1/projects/{projectId}/schedules`:

| Method | Route | Operation |
| --- | --- | --- |
| GET | `/` | List readable schedules with bounded pages. |
| POST | `/` | Create a schedule. |
| POST | `/preview` | Validate cadence and estimate the run selection. |
| GET | `/{scheduleId}` | Read configuration and current execution state. |
| PUT | `/{scheduleId}` | Update with a configuration `If-Match` revision. |
| GET | `/{scheduleId}/firings` | Read bounded firing history. |
| POST | `/{scheduleId}/archive` | Disable and archive with `If-Match`. |
| POST | `/{scheduleId}/restore` | Restore with `If-Match`. The restored schedule stays disabled. |

Schedule permissions control creation, updates, reads, and archival.
Object denies also apply to listing and firing history.
Archive preserves the identifier and history. Use `archived=true` to include archived definitions in lists.
Notification delivery remains a separate completion workflow. Schedules do not imply that a notification transport is configured.

## Local operation

The local server processes schedules by default.
Set `--scheduler=false` on instances that only serve requests or execute queued work.
Keep at least one scheduler instance active for each database.
SQLite supports one local service instance. Hosted replicas use PostgreSQL.
The same scheduling implementation supports both databases.
The initial functional verification uses local databases and local test targets.

Focused checks cover concurrent claims, restart recovery, release pins, permission changes, overlap, configuration revisions, and disable races.
They also verify normal run snapshots, execution, timing, results, evidence, and project-scoped API access.

## Interface

Open Schedules from the project navigation.
Create or edit a definition with its suite, target, location, release pin, and explicit test identifiers.
Preview shows five future occurrences and current timing readiness without creating a run.
Form changes clear the previous preview. Polling preserves unsaved editor fields.
Use Enable and Pause for future firings. Existing runs keep their normal cancellation controls.
History shows the due time, observed time, coalescing notice, owner, and blocked or skipped reason.
Run links open normal timing, result, attempt, and evidence views.
Archive keeps firing history. Include archived schedules to restore a definition in the paused state.
Configuration conflicts provide an explicit reload action.
Changing projects closes the previous editor.
The page has contextual Introduction, Concepts, and Tutorials help.
The editor and history use shared light and dark theme tokens and contained mobile table scrolling.

Run the focused browser workflow through TestStudio:

```sh
bash scripts/test-self.sh --test=STUDIO-SCHEDULE-001
```

## Guided configuration

Open Automation, then choose Schedules.
Create and edit uses Tests, Timing, and Review steps.
Enter a name and choose the suite and target. New identifiers generate from the name.
Advanced selection options contains the identifier, execution location, release pin, and explicit test selection.
Continue to Timing. Choose an interval or cron expression and timezone.
Advanced run options contains overlap, run limits, and retries.
Continue to Review. Inspect the target, cadence, selection, and limits.
Preview schedule shows future times and timing readiness without creating runs.
Choose Enable this schedule only when it should repeat. Save schedule remains the final action.
Back preserves all entries. Configuration conflicts still require a stored reload.
