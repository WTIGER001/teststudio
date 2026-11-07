# HTTPS probes

HTTPS probes checks websites and endpoints through TestStudio's native Go runner.
Open Project overviews, then HTTPS probes. This page spans the project's applications.
Checks run on the installation, not in your browser or on your workstation.
No browser worker or source preparation package is required.

## Create a first probe

Select Add HTTPS probe to open Name, Request, and Review directly.
Choose a suite and paste the complete website URL.
Review the application, stage, and protection of any matching target. Multiple matches require an explicit choice.
A new address lists its application and target configuration before saving.

Start with GET and accepted status 200 for a page that serves directly.
Example advanced choices are a 10-second timeout ceiling and a 14-day minimum certificate lifetime.
Keep content and response-time assertions unset until you choose them.
TLS trust, hostname, and certificate dates always require validation.
Redirects are not followed. Accept an intentional redirect explicitly and test its destination separately.
A homepage or admin entry page returning 200 does not prove authenticated functions or application dependencies work.

Review the safety label and rationale. GET and HEAD do not automatically establish safety.
Save draft stores configuration without requesting the website.
Save and run publishes a suite snapshot without activating it, then explicitly runs the selected test and its dependencies.
Adding from this page also saves a binding to the selected target.
A partial save retains successful resources. Retry on the same page without creating duplicates.
After reload, inspect saved configuration and runs before repeating an uncertain action.

## Add existing checks or several addresses

Add existing check chooses a saved native definition and an explicit target environment.
It does not change the definition, publish, or run it.
An identical binding reuses its stable identity. The same definition can have separate bindings in several environments.

Add multiple probes opens Addresses, Targets, and Review.
Paste up to 30 unique complete URLs, one per line. Choose the destination suite.
On Targets, name each probe and choose its application, stage, protection, and safety rationale.
The application must be linked to the suite when the suite has explicit application links.
Existing matching environments retain their stage and protection. Choose ambiguous targets explicitly.
Plain HTTP requires an explicitly selected Local stage in this setup workflow.

Review every row before Save probe configuration.
New checks use GET, accepted status 200, a 10-second timeout, and a 14-day certificate lifetime assertion for HTTPS.
New targets and test drafts are saved through their ordinary APIs. Saving starts no run and enables no schedule.
After a partial failure, Retry unfinished rows skips successful probe bindings and retains saved targets and tests.
Saved resources survive reload. Unsaved entries and the in-page retry continuation do not.

## Read the overview

Search by name or URL. Filter by application, stage, result, and history window.
Advanced filters contains freshness, schedule state, and archived bindings.
Filters stay in the route for refresh, Back, and shared links.
Results sort attention first, then name and stable identifier.

Each row shows the target, latest authorized logical observation, observation date, recent results, and matching schedules.
HTTP diagnostics includes response status, measured duration, available network phases, and certificate expiry.
Missing measurements remain unavailable. Historical certificate lifetime uses the observation time.
Changing an environment URL preserves old observations and marks them Previous target. Their success does not establish health at the new address.
Draft changes are marked separately. The row URL uses the active published definition. Activating different request or assertion criteria keeps earlier results marked Previous definition until a comparable new check completes. Pinned schedules may continue an earlier definition; their original observations remain in retained history.
An active run appears separately from the previous completed observation. Late completion of an older run does not replace a newer logical run.

Passed, failed, timed-out, flaky, skipped, cancelled, blocked, and incomplete records retain their meaning.
Execution problems and missing evidence do not become endpoint outages.
Manual only makes no monitoring freshness guarantee. Paused schedules retain their old observation.
For enabled schedules, overdue observations allow the frozen run budget and a 60-second reporting grace before becoming stale.
Blocked firings remain visible. Missing checks never count as successes.

The recent sequence shows up to 30 final logical observations as compact vertical pills in one row.
Green means passed, red failed, blue timed out, amber flaky, and neutral other outcomes.
Hover for the status and timestamp. Each pill has a screen-reader label and keyboard focus; select it to open its retained run.
Previous target and Previous definition labels remain available in the pill details.
Check success rate counts passed, failed, timed-out, and flaky outcomes once per logical run; only passed enters its numerator.
Other outcomes are shown separately. Empty or partial history has no percentage.
This is a sampled check success rate, not continuous uptime.
The overview reads at most 1000 newest project runs once per refresh. It reports partial coverage when more records exist.
Open retained history to search older observations. Definitions and schedules have a 1000-row overview configuration limit.

## Run and investigate

Run now reviews the active release, selected check and dependencies, exact target, execution location, 60-second run limit, and zero retries.
The normal run estimate checks admission before Start reviewed run creates the run.
If no active release exists, Review publication opens the existing catalog workflow.
Running never silently activates a draft.

View history retains the binding's environment and execution-location filters.
Open a retained run to inspect attempts, logs, events, and private evidence through the normal investigation page.
Authorization, retention, pins, and capture gaps use the existing evidence contracts.
See [results](results.md), [test history](test-history.md), and [native assertions](http-checks.md).

## Review several actions

Select explicit probe rows and choose Review selected runs or Review selected archival.
Selection survives filtering, and the selection bar reports hidden rows.
Review lists each target, stage, test, and execution location before applying anything.
Runs use each suite's reviewed active release and the normal run APIs; each target receives a separate run identity.
Archival preserves tests, schedules, and history. Pause schedules separately to stop monitoring.
After partial completion, retry unfinished actions on the same page. Successful rows are skipped and their saved run links remain available.
Reopening an unfinished batch preserves its request identities. After reload, inspect stored records before repeating uncertain actions.

## Schedule and stop checks

Schedule checks opens the existing schedule editor with the suite, exact test, environment, and execution location prefilled.
The example interval is 300 seconds, with no retries and overlapping firings skipped.
Saving assigns execution ownership to the editor. Every firing and dispatch checks that owner's current authority.

Use the active suite release or pin a reviewed published release.
Save and run did not activate its release. If there is no active release, the handoff can pin the latest observed published release.
To follow active releases, review all shared suite changes with Validate and preview before Publish and activate.
Preview schedule lists future times without contacting the target.
Enable this schedule and Save schedule explicitly authorize repeated execution.

Open the named schedule to edit its cadence or pause it.
Actions on schedules that select several tests or all eligible tests show their shared scope before confirmation.
Several matching schedules remain listed separately; there is no ambiguous pause-all action.
Pausing prevents future firings. Already queued runs can finish and require normal cancellation separately.

Archive probe removes the binding from the default overview and preserves tests, runs, and schedules.
Review explicitly explains that archival does not stop monitoring. Pause its schedules separately.
Restore probe leaves schedule state unchanged. Disable test is a separate catalog action.
Optional notifications use existing project completion channels and filters. See [schedules](schedules.md) and [notifications](notifications.md).
Select Status pages under Project overviews to publish reviewed probe summaries without sign-in. See [public status pages](status-pages.md) for time-based availability, embedding, and withdrawal.
Incident escalation, maintenance windows, and redirect following remain separate capabilities.

## Visual help

Select Help for Introduction, Concepts, and Tutorials.
The HTTPS guides show Request, Review, successful and failed HTTP diagnostics, and schedule review using shared application components.
Examples use fixed reserved-domain data. They make no target requests, save no resources, and enable no schedules.
They follow the current theme and fit narrow screens.

Show me actions locate a visible control without changing wizard steps, fields, or Advanced sections.
The modal drawer closes before the real control receives focus.
Return to help restores the tutorial scroll position. Escape or Dismiss highlight removes the highlight.
Unavailable controls explain the required step. Help and examples preserve unsaved work.

## API and operation

The `/api/v1/projects/{projectId}/probes` family lists summaries and creates explicit bindings.
Read and revision-controlled update use `/{probeId}`. References are permanent; add a new binding to change them.
Binding creation requires current catalog update authority on the suite and test and environment read authority.
Run and schedule data require their own current exact permissions. Denied records do not enter returned counts or history.
Identical creation replays reuse identity. Updates require a positive If-Match revision.
The canonical [OpenAPI contract](../contracts/openapi.yaml) specifies request, response, paging, conflict, and error behavior.

Bindings use normal durable record storage on SQLite and PostgreSQL, including backup and transfer.
Summaries derive from retained authoritative records. No separate projection job or recovery replay is needed.
Refreshing summaries cannot execute tests. The existing durable scheduler uses robfig cron parsing and transactional claims.
No additional queue or PostgreSQL-only job dependency is installed.
See the [delivery plan](https-probes-plan.md) for milestone evidence and deployment status.
