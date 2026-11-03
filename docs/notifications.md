# Durable completion notifications

## Policy and run snapshots

Notification channels belong to one project.
Each channel defines a transport, secret reference, enabled state, retry budget, and selection filters.
A project supports at most 16 unarchived channels.
Filters select suites, environments, stages, triggers, and outcomes. Empty scope filters include all values.
Stage selection uses the registered stage. Target protection does not imply a production stage.
Default outcomes include failed, timed-out, flaky, and unknown results.
An optional evidence-gap rule also selects incomplete capture without changing the observed test outcome.

Run creation freezes eligible channel definitions and filters in a private notification plan.
Manual, scheduled, and failure-rerun requests use this same path.
New channel settings do not apply retroactively to runs already queued.
Outcome selection occurs when the run finishes.
The completion transaction saves the run and its durable delivery work together.
Queued cancellation, queued deadlines, and lost-worker recovery also use this completion path.

Messages contain run identity, suite, environment stage, trigger, state, outcome, counts, evidence state, and the configured UI link.
Links require normal application authentication. Messages contain no artifact grants, logs, screenshots, response bodies, or resolved secrets.
Transport failures do not change functional test results.

## Delivery and retries

Workers claim delivery records with atomic revisions and 30-second leases.
Each claim creates a retained attempt record before sending.
A 2xx response confirms delivery.
Transient failures use exponential backoff, bounded jitter, and provider retry delays.
Shared destination cooldowns respect rate limits. Discord global limits also pause other Discord destinations.
Automatic attempts use the frozen budget. Channels accept 1 through 10 attempts; the default is 6.

Each delivery keeps one stable identifier across retries and restarts.
The webhook header `X-TestStudio-Delivery-ID` exposes that identifier for receiver deduplication.
Delivery has at-least-once semantics.
A worker can stop after remote acceptance and before its checkpoint. A retry can then produce a duplicate message.
Recovery records the interrupted attempt as unknown and preserves its acceptance uncertainty.
A later confirmed delivery does not erase that earlier uncertainty.
Notification recovery does not rerun tests or create another logical run.

Delivery states include pending, sending, retrying, delivered, failed, blocked, and cancelled.
Attempt history separately identifies interrupted acceptance as unknown.
Failed and blocked delivery records remain available after their work leaves the automatic queue.
An authorized administrator can retry with an explicit revision.
Retry preserves identity and history and adds another bounded attempt budget.
A delivery supports at most 1000 lifetime attempts, including explicit retries.
Pending or waiting retries can also stop explicitly.

## Pause, destination changes, and permission checks

Every attempt checks the current channel, project state, and channel owner's current permissions.
The owner needs notification management and access to the specific run.
Revocation blocks delivery. Error text contains fixed diagnostics and HTTP status numbers.
The service does not record webhook URLs, provider response bodies, or resolved credentials in delivery metadata.

Pausing or archiving advances the channel's delivery epoch.
Reenabling does not silently revive earlier queued messages.
An explicit retry can authorize an earlier message after the original channel is enabled again.
Retry requires the original transport and secret reference. Use a new run to apply a different destination.
Requests already sent cannot be recalled.
Archive preserves channel identity and delivery history. Restore leaves the channel disabled.

## Secret configuration

Register project and purpose bindings in an installation file:

```json
[
  {
    "id": "availability-receiver",
    "name": "Availability receiver",
    "projectId": "example",
    "purpose": "notification-webhook",
    "environmentVariable": "TESTSTUDIO_ALERTS_WEBHOOK",
    "network": "public"
  }
]
```

The environment variable contains the complete webhook URL.
Alternatively, use an absolute `file` path for a mounted secret. A binding uses exactly one source.
File values resolve again for each attempt. Environment changes require a process restart.
Binding configuration loads at startup through `--secret-bindings`.
The API exposes reference identifiers and names. It does not expose values, file paths, or environment variable names.
Bindings enforce project and purpose boundaries.
See [Notification transports](notification-transports.md) for network scopes, Discord behavior, and read limits.
Hosted secret service adapters remain pending.

## API

Use these routes below `/api/v1/projects/{projectId}/notifications`:

| Method | Route | Operation |
| --- | --- | --- |
| GET | `/references` | List registered notification reference names with management permission. |
| GET | `/channels` | List readable channel definitions. Use `archived=true` to include archived channels. |
| POST | `/channels` | Create a channel. |
| GET | `/channels/{channelId}` | Read a channel. |
| PUT | `/channels/{channelId}` | Edit with `If-Match`. |
| POST | `/channels/{channelId}/archive` | Disable and archive with `If-Match`. |
| POST | `/channels/{channelId}/restore` | Restore in the disabled state with `If-Match`. |
| GET | `/deliveries` | Read bounded delivery history. Optional `run` selects one run. |
| GET | `/deliveries/{deliveryId}` | Read delivery state and its frozen summary. |
| GET | `/deliveries/{deliveryId}/attempts` | Read bounded attempt history. |
| POST | `/deliveries/{deliveryId}/retry` | Retry stopped or waiting delivery with `If-Match`. |
| POST | `/deliveries/{deliveryId}/cancel` | Stop pending or waiting delivery with `If-Match`. |

Read operations enforce notification permissions and current access to each referenced run.
Object denies remove delivery data from history pages.
History cursors remain bound to their project and run filter. Bounded scans expose partial coverage.
Mutations require notification management, current resource context, session checks, and revision checks.

## Local operation and verification

The service processes notification work by default.
Set `--notification-worker=false` on nodes that only queue work or serve requests.
At least one worker instance must remain active for the database.
SQLite supports one local service instance. PostgreSQL supports multiple API and delivery worker instances.

Run the built service with a new local receiver and fresh records:

```sh
bash scripts/test-notifications.sh
```

The verification publishes a new native suite, runs it manually and on a schedule, and checks actual delivery retries.
It verifies stable delivery identifiers, retained failures, final counts, configured links, and secret-free API data.
The script stops its controller and local receiver. It retains its isolated database, artifacts, and summaries for inspection.
Focused race checks also cover leases, replica claims, uncertainty, policy snapshots, permissions, cooldowns, and cancellation.
## Interface

Open Notifications from the project navigation to manage channels and inspect delivery history.
The editor selects registered secret names. It never requests a webhook URL or displays a resolved credential.
New channels start paused. Filters select exact suite, environment, and stage values, run triggers, outcomes, and evidence gaps.
Timeouts remain a separate outcome. Empty scope filters include all values.
The editor requires at least one trigger and one outcome or the evidence-gap rule.
Changes apply to future runs. Open edits remain intact during polling and page help.
Revision conflicts retain edits and offer an explicit stored-configuration reload.
A project change closes the previous editor.

Channel controls pause, enable, archive, and restore configuration. Restoration leaves the channel paused.
Delivery history remains available after configuration changes and archive.
Run details show the same delivery view, restricted to that run.
Project history also accepts a `run` query parameter.

Expand Frozen run summary for captured scope, counts, and destination references.
Select Attempts for retained responses and checkpoint diagnostics.
Unknown acceptance remains explicit after later confirmation.
Expired sending leases show Recovery pending while a worker reconciles them.
Retry delivery preserves the identifier and test result. Stop retries cancels only pending or waiting delivery work.
These actions require notification management permission and current record revisions.

History uses bounded pages and exposes partial coverage.
Automatic refresh pauses after older history loads. Refresh deliveries returns to current history.
Tables scroll within narrow screens. Shared theme tokens support light, dark, and system preferences.
Page help provides Introduction, Concepts, and Tutorials.

Run the focused browser workflow through normal TestStudio execution:

```sh
bash scripts/test-self.sh --test=STUDIO-NOTIFY-001
```

Its isolated receiver verifies real generic and Discord requests without external network access.
The fixture registers only new project bindings and exposes no destination values through application APIs.
Hosted secret service adapters remain pending.
