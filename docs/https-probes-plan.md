# HTTPS probes feature and help plan

Status: Implementation authorized on 2026-11-06. Milestone 1 passes local verification. Milestones 2 through 6 are in progress; deployed behavior remains unchanged.

Make HTTPS probes easy to create, run, schedule, and investigate from one project overview.
Give each task contextual help with examples rendered from the same components as the application.
Start with one public website check, then support the user's nine selected website, admin, and search addresses across production, beta, and development.
Store those addresses through ordinary private project configuration. Keep product defaults, examples, and self-tests generic.

## Current behavior and gaps

| Area | Available now | Work to deliver |
| --- | --- | --- |
| Navigation | Project overviews contains HTTPS probes at `/projects/{projectId}/probes`. | Replace the filtered catalog presentation with a focused operational overview. |
| Creation | The HTTPS wizard supports Name, Request, Review, full URLs, reviewed target registration, Save draft, and Save and run. | Enter this type directly from Add HTTPS probe, retain return context, and provide clear next steps. |
| Checks | Native Go execution supports GET, HEAD, status, text, headers, JSON, response limits, and certificate lifetime. | Expose these existing capabilities through focused help and readable summaries. |
| Execution | Save and run publishes an immutable suite snapshot without activating it and selects the test and dependencies. | Make the executing installation and selected target clear. Do not imply the request runs in the user's browser or workstation. |
| Results | Run investigation retains HTTP diagnostics, timings, certificate expiry, attempts, and private evidence. | Summarize the correct target's latest observation and link to its normal investigation view. |
| Scheduling | Durable schedules select a suite, release policy, environment, and tests. | Carry the probe selection into a reviewed schedule and explain publication and activation. |
| Help | Create test, catalog, schedules, and run pages have text tutorials. The probes route uses catalog help. | Add a dedicated probes topic, an end-to-end tutorial, shared visual examples, and field highlights. |
| Target identity | Test definitions contain an endpoint alias. Runs retain their actual environment and definitions. | Persist the chosen test and target association so one definition used in several environments has separate rows. |

Reuse [native HTTP checks](http-checks.md), [test creation](create-tests.md), [schedules](schedules.md), [results](results.md), and [page help](help.md).
The new overview must use the same catalog, release, run, permission, notification, and evidence services.

## Tester workflow

1. Open a project and select HTTPS probes. See configured probes across its applications, with application and stage filters.
2. Select Add HTTPS probe. Open the existing dedicated HTTPS creation route directly.
3. Enter a title, select a suite, and paste the full URL. Reuse an exact matching registered target or review the new application and environment.
4. Review method, accepted status, target stage, protection, safety rationale, assertions, and effects before saving.
5. Choose Save draft or Save and run. Draft saving makes no website request. Running explicitly publishes the reviewed snapshot and creates one normal run selection.
6. Inspect the first result. See the actual response, timing, certificate information, and any assertion or execution failure.
7. Select Schedule checks. Review the selected test, target, release policy, cadence, overlap, owner, and repeated execution before enabling.
8. Return to HTTPS probes to inspect current observations, freshness, recent results, and schedule status. Open normal history and run investigation for details.
9. Edit and publish through the existing workflows. Show when a draft differs from the release used by a schedule.

Creating a probe must work without a browser worker, coded source preparation, or a service deployment.
Successful HTTP entry-page checks do not prove authenticated admin functions, search behavior, or dependent services work.

## Overview and investigation

### Main screen

Keep the existing project route and scope. Show the project context above the page.
Use the existing shared row and theme primitives where useful; keep probe-specific rendering in small focused components.

| Visible content | Meaning |
| --- | --- |
| Name and URL | Readable title, resolved target URL, application, and stage. |
| Latest result | Final logical test outcome, observation time, and readable failure reason. |
| Freshness | Current, stale, never checked, or unavailable, independent of the last outcome. |
| Response | Retained HTTP status and request duration. Missing measurements remain unavailable. |
| TLS | Certificate expiry and remaining lifetime, with the observation time clearly identified. |
| Recent checks | Up to 30 final logical observations with timestamps, outcomes, target scope, and links. |
| Schedule | Enabled, paused, none, blocked, or multiple; cadence and next due time where readable. |
| Actions | Run now and View history are primary. Edit, Schedule checks, schedule management, and Archive are available by permission. |

Provide search by title and URL; filter by application, stage, latest result, freshness, and schedule state.
Default to attention first, then name, with a stable identifier tie-breaker.
Filters and paging belong in the route so refresh, Back, and shared links retain the selection.
Show counts for matching authorized results only. Do not display decorative totals unrelated to the current query.
Keep loading, empty installation, no matches, permission denial, partial history, and server errors distinct.
Refresh must preserve filters and action errors. Suppress stale responses after a project change or access loss.

On narrow screens, stack each row's information without hiding the URL, result, freshness, or primary action.
Provide text and accessible names with state colors. Recent-history markers must work with keyboard and touch.
Use stacked area charts for state trends. A compact sequence of recent checks is an observation list, not an uptime chart.

### Result semantics

Preserve passed, failed, timed-out, flaky, skipped, cancelled, incomplete, and unknown distinctions from normal result contracts.
Show queued or running work separately from the most recent completed observation.
Do not convert blocked execution, missing observations, access denial, or upload gaps into an endpoint outage.
Evidence completeness stays separate from functional outcome.
A recovered passing retry remains flaky; attempts do not increase the number of logical checks.

Latest means the newest selected logical run by its retained creation order, not the last event to arrive.
A late completion from an older run cannot overwrite the newer row's selected observation.
Show observation and refresh timestamps so a last successful check cannot appear to be a current live measurement.

For enabled schedules, derive expected observations from the durable cadence and firing history.
Allow the firing's frozen run deadline plus a documented 60-second reporting grace before marking an unobserved occurrence stale.
For interval and cron schedules, apply the same rule using their actual due times, overlap skips, and coalesced downtime records.
Display blocked or missed execution reasons alongside stale state. A paused schedule keeps its prior result and observation date but makes no current health claim.
Manual-only probes show Last checked and Manual only; do not invent a cadence or freshness guarantee.

Initially label any percentage Check success rate, with a selected window, numerator, denominator, and coverage.
Count passed, failed, timed-out, and flaky completed logical outcomes once; only passed enters the success numerator.
Show other outcomes and missing expected checks separately. Exclude them from that denominator and disclose the exclusions.
Show no percentage for an empty or incompletely scanned window. Zero observations never means 100 percent.
Do not label this metric uptime or infer continuous availability between samples.

### Investigation

View history opens the existing retained test-history workflow with the probe's target scope.
Extend that workflow's server filters before using it where environment or execution-location scoping is missing.
Link the selected observation to the existing run and attempt inspection view, retaining its frozen release and target.
Show assertion failures, certificate validation failures, response limits, and attempt timeouts distinctly.
Retain DNS, connection, TLS, and first-byte phases without adding overlapping phases into a second total.
Historical certificate lifetime stays calculated at observation time; a current countdown must be separately labeled and derived from the retained expiry.

## Probe identity and data contracts

### Persistent association

Add a small project-owned probe binding that references a suite, stable test identifier, environment, endpoint alias, and execution location.
Give it a stable identifier, revision, display name, and archive state. It stores no copied executable definition, result, credential, or alternate scheduler.
Use the binding to represent a configured monitor even before its first run or schedule.
Prevent duplicate active bindings for the same project, suite, test, environment, endpoint, and location combination.
Keep suites project-owned and allow the existing multiple-application links. Environments remain application-owned.

Bind the environment identity rather than only its current hostname. Record target revisions and resolved origins with each observation.
If an environment's URL changes, show Target changed and a history boundary. The old origin's success cannot become the new origin's current result.
Keep existing timing-profile separation and frozen criteria for origin, trust, and execution-profile changes.
Archive preserves history and does not silently disable a test used elsewhere.
If archiving should pause a schedule, show the exact affected schedules and require their normal mutation authority.

Existing catalog HTTPS definitions remain available. Do not guess monitor bindings from every compatible environment or create a Cartesian product.
Provide an explicit Add existing check flow that chooses the definition and target.
Historical runs may inform a suggested target, but adoption requires review and does not execute anything.
Archived, disabled, removed, or inaccessible references produce clear retained states and block new execution as appropriate.

### Read and mutation API

Use a proposed project API family under `/api/v1/projects/{projectId}/probes` for bindings, summaries, and scoped observation history.
Document exact request and response schemas in OpenAPI during implementation, before wiring the page.
Use the normal catalog, publication, run, and schedule endpoints for their existing mutations.

The overview response must provide bounded paging, matching counts, applied window, server time, last refresh, and explicit history completeness.
Resolve latest observations, HTTP diagnostics, schedule references, and aggregate states on the server without one browser request per row.
Use a bounded indexed query or durable derived projection. Avoid rescanning all project runs for each probe.
Treat a projection as rebuildable from retained authoritative records; report rebuilding or partial coverage instead of inventing healthy results.
Do not rerun checks to repair summaries. Document backfill, retention, restart recovery, and SQLite/PostgreSQL parity.

Parse resource context before checking authority with the existing permissions service.
Evaluate project, suite, test, environment, run, and schedule visibility before returning their data.
A catalog-only reader may see an authorized binding while its result remains unavailable; do not leak denied run outcomes, counts, or schedule names.
Recheck current authority for every mutation, including publication, protected execution, target creation, and schedule changes.
Use revisions for edits and archive actions, stable idempotency receipts for create/run workflows, and explicit conflict feedback.
Bind cursors and caches to user, scope, filters, window, and relevant revisions. Do not cache private summaries publicly.
URLs must follow the existing credential-free address rules. Preserve the executor's omission of query strings from retained HTTP diagnostic URLs.
Use a redacted URL display for summaries and examples; authorized definition editing can still show the configured query.
Never put secrets into labels, help examples, or diagnostics.

## Creation and ongoing actions

Keep Name, Request, and Review. Add HTTPS probe selects the type without adding another parallel creation form.
Retain the generated identifier under Advanced details and preserve all existing validation and unsaved-work behavior.
Make application and target stage easy to see; keep their consequences on Review even when uncommon options are collapsed.

Use these values in the introductory example: GET, accepted status 200, 10-second timeout ceiling, and 14-day minimum certificate lifetime.
Label them example choices, not measured guarantees or automatic production-safe settings.
Leave response-time failure assertions and content assertions unset until the tester selects them.
Preserve existing target protection, timeout policy, and suite defaults unless the user explicitly changes them.
GET and HEAD never grant safety automatically.

Explain that redirects are not followed. A redirect probe may accept its expected status and assert the Location response header; test its destination separately.
Keep status and response headers available for HEAD, while blocking conflicting body assertions without discarding edits.
Keep TLS validation mandatory. Certificate expiry checks, response-time assertions, adaptive slowdown, and attempt timeouts remain distinct.

Saving a reviewed target, catalog definition, probe binding, release, or run can succeed before a later step fails.
Extend the current receipts and retry flow to report each saved resource and resume without duplicates.
Keep successful work locked during an in-page retry, and explain what reload loses and which saved resources remain recoverable.
After a successful first run, offer View result, Return to HTTPS probes, and Schedule checks.

Run now previews the chosen test, dependencies, release, exact target, and protection through the normal run workflow.
Use the active release by default. If none exists or a draft differs, offer the existing reviewed publication flow.
Never run a saved draft implicitly or change the active release as a side effect of a plain Run now action.
Retain explicit combined Save and run behavior and its nonactivating publication semantics.

Edit definition and Edit target remain distinguishable. Changes show their effects on other bindings and shared suite schedules before publication or mutation.
Disable test, Pause schedule, and Archive probe have different effects and must use those explicit labels.
Bulk actions select explicit bindings, show targets and shared effects, and retain successes when only part of a batch can proceed.

## Scheduling and notifications

Schedule checks opens the existing schedule workflow prefilled with the binding's suite, test ID, environment, and execution location.
Keep the selected IDs visible in the probe handoff, rather than hiding the fact that only one check was selected.
Start with an example five-minute interval, no retries, and skip overlapping firings. Keep enablement explicit.
Show who will own execution and that each firing rechecks their current permissions.

Offer Follow active release or Pin this published release. If following active, require an active release containing the selected check.
When needed, route through Validate and preview and Publish and activate, showing all suite changes and affected schedules.
Do not activate unrelated draft changes automatically to make a schedule work.
Return from publication with the schedule draft preserved, or provide clear saved-resource recovery if navigation requires leaving it.

Associate schedules by their actual suite, selected tests, environment, and location. Derive references instead of copying schedule state into the binding.
A broad schedule can cover several probes. Pausing it affects all of them and must show that scope.
Multiple matching schedules appear as Multiple schedules with a readable list. Never choose one silently or pause them all with an ambiguous button.
Preview must not execute a check. Editing cadence and pausing must retain revisions, firing history, and existing idempotency rules.
Once a schedule is paused, a previously queued run may still finish; display it and offer normal cancellation separately.

Use existing project notification channels and outcome filters for optional notifications.
Explain current completion-based behavior, retry semantics, private links, and notification delivery failures separately from probe results.
Incident grouping, consecutive-failure escalation, recovery-only messages, maintenance windows, and public status pages are later work.
Do not present View incident as implemented simply because the reference screenshots contain it.

## Help with examples rendered from shared components

### Dedicated topics and tutorials

Give the overview a `https-probes` help topic. Give the HTTPS creation route specific guidance while preserving general Create test help for other types.
Keep Introduction, Concepts, and Tutorials on every affected page, including schedule handoff and history.

| Tutorial | Complete task and visual examples |
| --- | --- |
| Create your first probe | Choose or create a suite, enter an example URL, understand assertions and safety, then save or explicitly run. Show Request and Review examples. |
| Read the first result | Identify HTTP status, duration, certificate expiry, assertion failures, timeout, and incomplete evidence. Show successful and failed result examples. |
| Repeat a check | Explain nonactive Save and run publication, choose active or pinned release, preview cadence, and enable a reviewed schedule. Show schedule Review. |
| Monitor several stages | Reuse a definition where appropriate, bind each target explicitly, and group website, admin, and search surfaces by stage. Show separate target rows. |
| Change or stop monitoring | Edit without rewriting history, publish deliberately, pause the intended schedules, and archive without deleting retained runs. |
| Recover from problems | Correct invalid input, retry partial saves, handle conflicts and revoked permissions, and distinguish stale observations from endpoint failures. |

Use reserved example domains and synthetic data. Mark every example Example data, with fixed observation times and no relationship to the user's real project.
Explain redirects, mandatory TLS checks, protection, certificate thresholds, timeout distinctions, expected execution location, and public entry-page limitations beside the relevant task.
Link the feature guides and exact visible controls. Current help should describe only delivered milestones; keep planned features in this plan until available.

### Component design

Extend the help model with optional typed example blocks and registered local highlight actions.
Render known Angular components from a fixed registry; never accept arbitrary HTML, dynamic code, or untrusted component names.
Keep existing text-only topics compatible.

Extract small presentational request, review, result, and schedule-summary components from real screens where needed.
Pass immutable display models into them. Both production screens and help examples use the same label definitions, formatting utilities, validation descriptions, and styles.
Do not instantiate entire feature pages or inject workflow services merely to render an example.
Examples must never call APIs, load project data, register resources, navigate to external sites, publish, run checks, or enable schedules.
Use unique input IDs and accessible example captions so the page and drawer do not duplicate IDs or confuse label targets.

Begin with read-only examples. Add isolated editable practice fields only where they materially clarify a task, with a visible Reset example action and no persistence.
Use the current theme tokens, responsive layouts, keyboard focus, and reduced-motion behavior.
Keep explanatory text beside each example. Do not shrink an entire desktop page to an unreadable thumbnail.
No screenshot asset generation or image service is required for this feature.

Shared components reduce visual drift; they do not eliminate tutorial maintenance.
Changing a shared control, label, validation rule, default, or action updates its example and prompts review of the corresponding tutorial in the same change.

### Show me on this page

Add optional actions to locate a visible control on the current page, such as Website URL, Accepted status codes, or Review.
Use stable semantic anchor identifiers registered by the active page, not translated text matches or fragile CSS selectors.
The help dialog is modal. Close it before focusing the real page, show a nonmodal highlight, and provide an explicit Return to help action restoring the section and scroll position.
Escape clears the highlight and returns focus appropriately. Respect reduced motion.
Do not switch wizard steps, open hidden Advanced sections, change fields, submit forms, or navigate as a side effect of locating a control.
When a control is unavailable, explain which step or permission is required and keep the user's edits intact.
Remove highlights on route change, page destruction, and access loss.

## Delivery milestones and acceptance

These milestones refine the stable acceptance ledger. They add no new product acceptance numbers and close no cross-profile requirement by themselves.

| Milestone | Deliverable and completion condition | Acceptance items | Smallest useful verification |
| --- | --- | --- | --- |
| 1. First-probe help | Dedicated probe and creation topics; complete create, run, inspect, activate, and schedule instructions; shared Request and Review examples; safe current-page highlights. Examples produce no requests or mutations and help preserves edits. | 05, 33, 34, 49, 50 | UI build and one new focused normal TestStudio browser workflow for examples, field highlights, focus, themes, narrow layout, and preserved values. |
| 2. Target identity and summaries | Revisioned bindings, explicit adoption of existing checks, bounded overview/history API, correct target and run ordering, permission filtering, restart recovery, and complete contract documentation. | 06, 13, 25, 27, 28, 34, 37, 39, 40, 45 | Selected Go service/API tests for target separation, late completion, denial, conflicts, projection recovery, and SQLite/PostgreSQL storage behavior. |
| 3. Operational overview | Focused project screen, target-aware rows, server filtering and paging, recent results, freshness, TLS and timing summaries, and result help examples. Existing history and run links preserve target scope. | 34, 39, 40, 42, 44, 45, 47, 49, 50 | One new focused normal TestStudio browser workflow using controlled pass, failure, timeout, stale, partial, and never-checked cases; focused summary utility checks where needed. |
| 4. Reviewed actions and automation | Add, adopt, edit, Run now, publication handoff, schedule creation, pause, archive, shared-effect review, optional notification links, and partial-save recovery. No accidental activation, extra run, or broad schedule pause. | 21, 25, 27, 28, 29, 33, 34, 37, 38, 48, 50 | Focused run/schedule service checks and one normal TestStudio browser workflow for reviewed handoff, response-loss retry, and shared-schedule consequences. |
| 5. Multiple-target setup | A dedicated Add multiple probes route uses Addresses, Targets, and Review. It accepts explicit URL rows, validates duplicates and stage/application choices, and records per-row receipts. Draft saving makes no target requests. Run and enable remain explicit reviewed actions. | 03, 05, 21, 25, 33, 34, 38, 50 | A focused normal TestStudio browser workflow with nine new generic addresses across three surfaces and stages, including one partial save and retry. |
| 6. Release and selected-domain onboarding | Commit verified implementation, stamp matching artifacts, publish through the authorized release process, then create or adopt the user's nine private project probes without duplication. Verify one explicit initial run per selected target and schedule enablement separately. | 33, 34, 49, 50; bounded progress under 57 | Inspect deployed version/assets/help, verify each selected run identity and actual result, and inspect the first enabled schedule firing without repeating the full local workflow suite. |

Milestone 1 is the first implementation milestone after this plan. Begin with its read-only examples and tutorial; add current-page highlights after those work.
Milestones 2 through 4 complete the single-probe operational workflow before adding the multi-address setup in milestone 5.
Ship verified increments with complete help for the behavior they expose. Do not show empty or misleading planned controls between milestones.

### Targeted browser selection

Register new workflows in `tests/selftest/catalog.json` before invoking their IDs.
Suggested new IDs are `STUDIO-PROBE-HELP-001`, `STUDIO-PROBE-OVERVIEW-001`, `STUDIO-PROBE-ACTIONS-001`, and `STUDIO-PROBE-SETUP-001`.
These IDs are planned and are not currently runnable catalog entries.
Use `bash scripts/test-self.sh --test=REGISTERED-ID` with the actual selected ID after registration.

Reuse retained evidence for unchanged behavior. Rerun existing `STUDIO-URL-001`, `STUDIO-HTTPS-001`, `STUDIO-SCHEDULE-001`, `STUDIO-HISTORY-001`, or `STUDIO-HELP-001` only when the implementation changes their behavior or leaves a specific unresolved concern.
Do not use live FoodMarketMaker sites as deterministic self-test fixtures or run a full catalog matrix by default.
Close and verify all owned browsers, processes, containers, and servers. Preserve the user's port 8040 demo and history.

## Documentation and release completion

### Implementation ownership

| Source area | Planned responsibility |
| --- | --- |
| `web/src/app/features/help/` | Typed example registry, dedicated topics, modal-safe highlights, and accessible example rendering. |
| `web/src/app/features/catalog/https-wizard.*` and `probe-target.*` | Reuse creation, reviewed target selection, receipts, and direct typed entry. Extract presentational fields and summaries for examples. |
| New focused `web/src/app/features/probes/` components and services | Overview, filters, target binding setup, server reads, and handoffs to existing workflows. Keep components below 450 lines. |
| `web/src/app/features/runs/` and `features/test-history/` | Shared diagnostic rendering and retained target-scoped investigation. |
| `web/src/app/features/schedules/` | Prefilled reviewed handoff and readable shared-schedule effects, using the existing editor and service. |
| `internal/runs/`, catalog, storage, and a focused probe binding service | Authoritative observation reads, bindings, bounded summaries, current authority, storage parity, and recovery. Keep HTTP execution in the existing executor. |
| `internal/httpapi/`, `contracts/openapi.yaml`, and generated clients | Typed API handlers, documented permissions, revision and paging rules, and generated client updates. |
| `tests/selftest/` | New generic controlled scenarios through normal TestStudio execution. |

Update `docs/http-checks.md`, `docs/create-tests.md`, `docs/help.md`, `docs/ui.md`, `docs/schedules.md`, and affected history, result, notification, API, and operator guides with each behavior milestone.
Update routes and help topics together. Keep component ownership, recovery rules, projection rebuild instructions, and target-history semantics documented.
Add the feature guide to the README and synchronize this plan's completed milestones, continuation notes, and ledger entries with retained evidence.

Follow [versioning](versioning.md) before publishing changed behavior. Build from committed source and retain one shared UI/API/image identity.
Documentation-only planning needs consistency, local link, and stable-ledger checks. It needs no build, browser run, version bump, or deployment.
Actual MMDev publication uses the authorized account and TestStudio-owned resources, current-state inspection, and concrete change-set review where infrastructure changes are needed.
The feature does not require an AWS infrastructure change merely to add native HTTP probes.
Keep hosted evidence separate from local verification. Azure stays deferred, and this work does not establish coded-worker readiness or full acceptance 57.

## Completion checklist

- A tester can create, review, run, inspect, schedule, edit, pause, and archive a target-specific probe through the supported UI without source changes.
- Production, beta, and development observations never mix, even when they share one test definition.
- An empty installation stays empty until the user explicitly creates or adopts probes.
- Unknown, stale, blocked, paused, timeout, retry, and incomplete evidence states stay honest and explain the next available action.
- Recent history and percentages disclose window and coverage without claiming continuous uptime.
- Every affected help topic explains the full task and includes readable examples that reuse real UI components.
- Opening examples or locating fields preserves edits, performs no mutations, and works with keyboard, themes, narrow layouts, and reduced motion.
- Current authority, review, revision conflicts, immutable history, explicit execution, and partial-save recovery remain enforced.
- The nine selected private project probes have explicit targets, actual normal-run evidence, and reviewed schedule settings after onboarding.
- Local evidence, deployed behavior, target observations, and still-open cross-profile acceptance are reported separately.
