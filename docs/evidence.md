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
Storage cleanup, evidence pins, and capture configuration screens remain pending.
