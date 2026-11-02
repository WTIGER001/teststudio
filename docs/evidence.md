# Evidence capture and retention

Each run freezes a capture policy, project limits, and effective policies for its selected tests.
Project defaults apply when a suite has no override.
Catalog publication freezes the suite policy and test settings in the release.
A run override replaces those policies and requires `teststudio.evidence.override-capture`.
The worker checks that permission again before each attempt.

## Policy settings

Project and suite records accept `capture`. Project records also accept `captureLimits`.
Test definitions accept `capture`. Run creation accepts `capture` as an explicit override.
Use complete policy values when overriding capture settings.
The default policy enables logs, failure screenshots, failure traces, and no video.

| Policy field | Behavior |
| --- | --- |
| `logs` | Enable or disable process log artifacts. Structured events remain available. |
| `screenshots`, `trace`, `video` | Select `off`, `failure`, or `always`. |
| `maxBytes` | Set the run budget or a test budget for a test override. |
| `maxFileBytes` | Set the maximum bytes for one artifact. |
| `retentionDays` | Set the default evidence duration. |
| `kindRetentionDays` | Override durations by artifact kind. |
| `failureRetentionDays` | Extend retention for failed or flaky observations. |

The default run budget is 100 MiB. The default file limit is 64 MiB.
Project limits default to 10 GiB total storage, 1 GiB per run, 64 MiB per file, and 3650 retention days.
Policy values must stay within the project limits.
Native HTTP execution captures logs and events. Browser media requires a compatible browser worker.

## Quota accounting

Artifact admission reserves project, run, and test bytes in one transaction.
Competing runs cannot independently consume the same project capacity.
Duplicate uploads reuse their reservation. Failed uploads retain a reservation for retry.
Rejected uploads reserve no bytes.
Current project file limits and storage capacity also apply to an existing run.
Quota guards preserve the project's public revision and update timestamp.

## Retention processing

Run completion and lost-worker recovery queue a durable retention task.
The task uses recorded outcomes and the artifact's frozen policy.
It updates expiry metadata in bounded pages and resumes from a saved cursor after restart.
Failed or flaky observations can retain evidence longer than successful observations.
Processing failures remain in retention status and retry after one minute.
Retention state stays separate from test outcome and capture completeness.

## Pins and cleanup

Use `GET /api/v1/projects/{projectId}/runs/{runId}/retention` for pin state, retention processing, capture settings, and quotas.
Use `PUT /runs/{runId}/retention/pin` below the project prefix to pin or unpin evidence.
Send `If-Match` with the pin revision and a reason. Pin creation requires `teststudio.evidence.retain`.
A pin can have an expiry time or remain active until an authorized user removes it.
Pins protect retained files. They cannot restore files that storage has already deleted.
Pin creation waits until active deletion claims finish.

Cleanup processes finished runs after retention calculation completes.
It respects active pins and current upload leases.
Competing cleanup instances claim each artifact through a durable deletion lease.
Deletion failures use `cleanup-failed` state, retain their quota reservation, and retry after one minute.
Successful deletion releases project, run, and test reservations in one transaction.
The manifest remains with `expired` or `deleted` state and a deletion timestamp.
Compact test results remain unchanged.

Administrative deletion uses `DELETE /runs/{runId}/artifacts/{artifactId}` below the project prefix.
It requires `teststudio.evidence.delete`, the artifact's `If-Match` revision, and a reason.
Unpin the run before deleting evidence. The request queues deletion and returns `delete-pending` state.
Access grants and deletion requests create audit records.

Uploads use a bounded lease. Duplicate requests reuse pending or available manifests without another reservation.
A stale upload cannot replace a manifest after cleanup claims it.
Background indexing adds expiry records and project accounting to retained artifacts from earlier installations.
Capture configuration screens, retention screens, and cloud artifact adapters remain pending.
