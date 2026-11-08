# Result trends

Trends use stored final logical results and immutable selected-test snapshots.
They require no generated HTML report or artifact download.
Each finished run contributes one observation per selected test.
Retry attempts do not increase logical result counts.

## States and gaps

Stack states in this order: passed, failed, timed-out, flaky, skipped, and unknown or incomplete.
Keep the same order and colors across overall, category, and test views.
The vertical axis shows test counts. Points follow run creation order and use UTC dates.
Adjacent known observations form connected areas. A selection gap breaks each area.

A selected test with no final result contributes an unknown count.
An incomplete or cancelled run cannot turn its unobserved tests into passes.
A run that does not select a matching category or test creates a visible gap.
A missing selection snapshot also creates a gap. It contributes no invented zero counts.
Duplicate final records for one test create an unknown observation instead of duplicate counts.
Active runs remain outside final-result trends.

## API

Use `GET /api/v1/projects/{projectId}/trends` with the following query parameters:

| Parameter | Operation |
| --- | --- |
| `since`, `until` | Required RFC 3339 boundaries for run creation time. The maximum period is 366 days. |
| `application` | Match frozen application context and count only its selected tests. Options use the same application scope. |
| `suite` | Match a suite identifier. |
| `environment` | Match an environment identifier. Use `none` for targetless runs. |
| `location` | Match an execution location. |
| `category` | Match the category in each selected-test snapshot. |
| `test` | Match a permanent test identifier. Also supply its suite. |
| `limit` | Scan 1 through 500 project run records. The default is 200. |
| `cursor` | Continue with an older bounded history page. |

The service checks project and run read permissions through the shared permission service.
Denied runs contribute no points, rankings, or filter suggestions.
The response contains points, aggregate counts, sample counts, context-run counts, rankings, suggestions, and an optional cursor.
It includes no endpoint credentials or artifact content.

The scan follows newest run identifiers first. Returned points follow creation time, then identifier, in ascending order.
Filters apply within the bounded scan. A page can contain no matching points and still provide an older cursor.
The cursor binds the project, reader, period, filters, and scan limit.
Changing those values requires a new first-page request.

`partial` means more project records remain outside the current scan.
Counts and rankings cover the displayed page. They do not claim to summarize the complete period.
An older page also remains a partial view even when it reaches the oldest stored record.
The interface keeps that notice visible from its cursor context.
Filter suggestions use historical snapshots and have a 200-entry limit per field.
Exact filters remain available when suggestions reach that limit.

## Rankings

Failure rankings sort tests by failed plus timed-out logical results.
Each row retains separate failure and timeout counts.
The response provides the first 20 rows. Equal counts use sample count, then stable suite and test identifiers.

Volatility measures state changes divided by comparable adjacent observations.
The result ranges from zero through one. The interface shows it as a percentage.
Compare observations only within the same suite, target environment, and execution location.
Unknown results, skipped tests, absent selections, and category changes break comparisons.
Another suite's run does not break a test's own comparison chain.
Tests with no comparison have unavailable volatility and remain outside this ranking.
The response provides the first 20 rows, sorted by volatility, comparison count, and stable identity.
Display names and category labels use the latest matching snapshot in the page.

## Interface

Open Trends from the project navigation.
Choose UTC dates and optional scope fields, then select Apply scope.
The URL preserves filters for reloads and sharing.
Category and test links in rankings open their corresponding views.

Hover or focus a chart run to inspect every state count and the total.
Select the run to open its original details.
Use View chart values for the accessible table and direct run links.
On narrow screens, the chart and wide tables scroll within their containers.
Light and dark themes use shared state colors, including a separate blue timeout color.

Use Older history to continue scanning or Newest history to return.
Request failures show an error instead of an empty healthy history.

## Local verification

Run the product workflow through TestStudio:

```sh
bash scripts/test-self.sh --test=STUDIO-TRENDS-001
```

The workflow creates real passes, failures, timeouts, a successful retry, and a run deadline.
It verifies counts, selection gaps, rankings, keyboard inspection, links, URL filters, page help, and both themes.
Focused Go checks cover aggregation, permissions, validation, and cursor boundaries on SQLite and PostgreSQL.
Pure chart checks verify area totals and missing-history breaks.

Cross-project rollups and HTTPS response-time and certificate history remain pending.

The Application selector controls the page scope. Application routes and ranking links retain it after reload.
Choose All applications to inspect project-wide history. Shared run details show their complete snapshot.

## Shared reliability capture

The user approves shared reliability history for every probe and test type on 2026-11-07.
The shared result lifecycle now captures reliability observations for every runner. The bounded trends API above retains its per-run investigation view. The reliability endpoint supplies complete bucketed reports without that scan limit.
Native HTTP, Go, Playwright browser, API, recorded journeys, and future check types must use the same result capture and aggregation contract.
Capture belongs to the shared result lifecycle. Status pages, private history, trends, and reports consume that history instead of building separate capture paths.
Recording reliability history does not publish a check or grant anonymous access. Existing publication and permission rules still apply.

### Shared observations and metric meaning

Record one canonical final observation per logical test execution. Keep attempts, assertions, steps, and package events beneath that observation.
Persist a durable aggregation work item with the result, or use an equivalent transactional checkpoint. A worker failure must not lose accepted results or require test execution again.
Preserve stable test and run identities, scope, runner and runtime context, definition and target fingerprints, observation times, final and functional outcomes, failure kind, measured duration, and timing assessment.
Keep credentials outside fingerprints and summaries. Preserve frozen run criteria and the existing timing-baseline eligibility rules.
Retain passed, failed, timed-out, flaky, skipped, and unknown counts separately. Duplicate delivery and retries must not inflate logical counts.
Interrupted or selected-but-unobserved tests remain unknown or incomplete. Do not invent a target failure or a passing result.

Every check supports outcome counts, explicit pass-rate denominators, timing summaries, and bucketed history over retained periods.
Reports show excluded skipped and unknown counts alongside a pass rate. They do not disguise missing execution as success.
Time-based availability requires an explicit monitoring policy with a freshness limit and outcome classification. Any check type can use that policy when configured as a monitor.
An occasional unit test or manual browser run does not establish continuous uptime. Without a monitoring policy, a period with no execution has no observations and no availability percentage.
Target failures and monitoring infrastructure failures remain distinguishable. Infrastructure failures and stale observations cannot establish uptime.

### Incremental summaries and retention

Start with indexed hourly and daily summaries. One summary table can hold both resolutions; weekly, monthly, and yearly reports combine compatible smaller buckets.
Store additive outcome counts and monitored available, unavailable, and unknown durations. Calculate percentages from their summed numerators and denominators.
Store timing sums and eligible sample counts for means. Retain censored and unavailable measurements separately. Percentile reporting needs a mergeable distribution or retained samples; never average percentiles.
Use UTC storage boundaries and explicit report boundaries. Distinguish rolling 30-day and 365-day ranges from calendar months and years.
Calculate partial range edges from finer retained data. Do not prorate a daily percentage to invent an hourly result. Disclose unavailable finer detail after retention.
Combine materialized history with a bounded live tail. Reuse summaries across viewers instead of rescanning complete project runs for each refresh.
Advance monitored intervals through freshness expiry even when no new result arrives. Stop the interval at the next valid observation or its freshness limit.
Process duplicates, late results, corrections, restarts, and concurrent workers through idempotent checkpoints and bounded bucket rebuilds. Preserve logical observation ordering.
Key summaries by compatible scope, definition, target, monitoring policy, runtime context where relevant, and aggregation version. A policy change must not silently reinterpret retained history.
Compact observations, authorization provenance, and hourly and daily summaries remain retained without automatic expiry in this release. Diagnostic artifact retention stays separate. Aligned historical summaries survive removal of original observations; exact partial edges and recent-check drill-down require their finer retained data. All time starts at the first readable retained observation or summary.
Recheck current authority when reading reports. Summaries and caches must preserve project, application, and per-run restrictions, explicit denies, revocation, and publication withdrawal.
Do not combine differently protected observations into a summary that cannot enforce its source access rules.

### Bucketed pills and reporting

Keep the compact pill display across check types. Individual-result pills remain useful at the closest view; longer views represent explicit time intervals.
Start with hourly pills for 24 hours, six-hour pills for seven days, daily pills for 30 days, and weekly pills for 365 days.
Use monthly or yearly display buckets for longer ranges while bounding the visible pill count. Display resolution does not require a matching storage table.
Keep empty intervals visible. Distinguish no executions from monitored unknown time, and mark unfinished buckets as partial.
Keep current status separate from historical bucket status. A successful latest result must not erase an earlier failure in the bucket.
Preserve any unsuccessful observation or unavailable duration in the bucket details and visible status indication. Show monitoring gaps even when downtime also occurs.
Expose boundaries, outcome counts, downtime, availability where applicable, and coverage through keyboard and pointer details. Drill into finer retained periods before opening authorized run details.
Private drill-down retains normal run permissions. Public projections retain their allowlist and never expose private runs or evidence.
Use the same underlying totals for pills, donuts, accessible values, and exports. Retain stacked area charts for result-state trends.

### Reusable Angular components

The user also requests stylable Angular components for other Angular sites on 2026-11-07. Version 0.1.0-dev of @wtiger001/teststudio-angular now implements this contract; Angular 22.2.4 is the verified peer version. See the [library guide](../web/projects/status/README.md).
Deliver a versioned Angular library with standalone, OnPush components and signal-based state. Use the same library inside TestStudio to keep internal and external rendering aligned.
Provide a compact status bar for one check and a complete status-page component for a selected report. Both support every check type through the shared reporting contract.
The complete page composes the same bar and summary primitives. It supplies range selection, ordered rows, applicable availability and coverage, accessible details, and drill-down events without taking over the host router or application shell.
The public selectors are teststudio-status-bar, teststudio-status-page, and the optional teststudio-status-summary. The independent example verifies package installation and one deduplicated Angular runtime. Other peer versions remain unverified.
Build an installable Angular package with a stable public entry point. Consumers must not import this repository's application internals or install a second Angular runtime.
Follow [Angular library guidance](https://angular.dev/tools/libraries/creating-libraries) and [host styling guidance](https://angular.dev/guide/components/host-elements).

Separate typed report models and render-only components from an optional injectable reporting client. A host can supply report data from its own service or use the client against a configured TestStudio base URL.
Server reports supply bucket boundaries, counts, durations, coverage, freshness, and metric meaning. Components render these values without rebuilding rollups or deciding monitoring eligibility independently.
Inputs configure the report source, time range, density, optional headings and summaries, refresh behavior, locale, and display timezone. Outputs report range changes, bucket selection, loading failures, and requested drill-down for host-owned navigation.
The bar supports one row and the page supports multiple rows. Multiple instances need unique accessible control identifiers and independent scopes.

Expose a documented styling contract with namespaced CSS custom properties for status colors, typography, pill width and height, spacing, radius, and surfaces. Inherit host fonts and provide light, dark, and system defaults.
Provide template customization for titles, descriptions, and detail content where CSS alone is insufficient. Keep state labels, keyboard access, unknown coverage, and visible failure indications intact.
Scope styles to components. Require no global CSS reset, private selectors, or deep overrides. Host styles must not leak into other components through the library.
Allow embedding as a normal Angular component without an iframe. Keep the existing standalone public URL and iframe path available.

Public clients consume only explicitly published allowlisted reports. Component installation, a page identifier, and CORS configuration grant no additional data access.
Support cross-origin public report reads with an explicit read-only CORS policy and no browser credentials. Configure and verify that policy independently from iframe framing headers; private endpoints retain their own restrictions.
Private reports use host-provided data or an explicitly supported authenticated adapter with normal server authorization. Do not place administrator keys, worker credentials, or long-lived service tokens in browser bundles.
Clear withdrawn reports after unpublishing or access revocation. Distinguish loading, stale data, no observations, and unavailable access; none may appear healthy by default.
Bound refresh rates, share identical in-flight reads where appropriate, and clean up timers and requests on source changes and component destruction. Rendering a component never executes a check or initiates sign-in.
Keep browser-dependent behavior safe for server rendering and hydration. Do not require window or document during server rendering, and prevent duplicate polling after hydration.

### Delivery and focused verification

1. Add shared durable capture and summary primitives at result acceptance. Verify actual HTTP, Go, browser, and API paths on SQLite and PostgreSQL with explicit generic fixtures.
2. Add incremental hourly and daily processing, safe replay, backfill of retained results, retention boundaries, and exact partial-range handling. Compare summaries with direct calculations, including missing checks, timeouts, retries, late results, policy changes, and access revocation.
3. Add shared bucket queries, pill history, drill-down, and matching report totals. Update contracts, contextual help, and feature guides. Use explicit normal TestStudio browser IDs for changed workflows and preserve existing result-state charts.
4. Package the Angular status bar and complete page, then use them inside TestStudio and in an independent generic Angular consumer. Verify installation from the built package, both data modes, custom themes, multiple instances, keyboard and narrow layouts, server rendering, refresh cleanup, public cross-origin reads, and withdrawal. Keep the consumer independent of TestStudio application internals. Select an explicit normal TestStudio browser ID for the integration workflow. Publishing the package and verifying other sites remain separate delivery steps.
5. Check representative multi-year data volumes and bounded refresh work. Record query cost and freshness, then verify deployment profiles separately. Add coarser stored summaries only when measured workloads justify them.

Local implementation and focused verification refine acceptance 34, 35, 36, 37, 39, 40, 41, and 47, with supporting access, UI, and help scope under 12, 13, 49, and 50.
Cross-profile evidence remains under 56 and 57. The count remains 50 verified of 57. Target publication remains separate from these local checks. The upstream logo implementation is integrated locally and its fields and centered rendering are preserved.

### Reliability API and operation

GET /api/v1/projects/{project}/suites/{suite}/tests/{test}/reliability reads one test in an explicit environment and execution location.
Omit environment for targetless checks. An omitted location uses the installation execution location.
Choose window=24h, 7d, 1m, 1y, or all. Explicit since and until must appear together and define a half-open range.
Resolution accepts hour, 6h, day, week, month, or year. Automatic resolution keeps the response compact. Requests exceeding 120 display buckets fail validation before scanning history.
The report uses the existing probes row-array name for compatibility across all test types.
Outcome reports default to maxAgeSeconds=0. An explicit value from 60 through 86400 opts into a monitoring report without changing schedules or frozen execution rules.
Monitoring matches the active definition, source artifact, timing policy, target, and credential references. Older configurations do not establish availability for the current configuration.
Pass-rate denominators include passed, failed, timed-out, and flaky logical outcomes. Skipped and unknown counts remain visible and excluded.
The API preserves measured duration sums and sample counts, censored counts, unavailable measurements, and the number of distinct configuration/runtime contexts. It publishes no averaged percentiles.

Result acceptance commits observations and dirty-series work atomically. Terminal runs also capture selected tests with no final result as unknown.
A recoverable worker backfills retained completed runs and refreshes hourly and daily summaries in bounded day batches. Duplicate capture and replay do not add observations.
Closed bucket reads use saved summaries. Partial edges use compact observations. Full-report percentages derive from duration totals; no percentage averaging or interpolation occurs.
Each summary retains run provenance and a smaller aggregate grouped by frozen authorization context. Native permission checks permit that aggregate only when every relevant policy is independent of individual run objects.
Any object-specific grant or deny uses per-run provenance instead. Grant expiry, application scope, publisher revocation, and directory changes remain enforced.
Aggregation backlogs return 503 with retry guidance. They never return a partial percentage as complete.
Database metadata version 2 includes all six reliability tables in installation export and import. See [database transfer](data-transfer.md).

Open a test's Test history page for Reliability history. Select the environment and period, inspect a pill, and use Inspect period to narrow the range.
Advanced reliability settings controls location and an explicit monitoring policy. These controls never dispatch work.
The existing per-run history and stacked area trends remain available for investigation.

Run STUDIO-RELIABILITY-001 for private history and the independent Angular consumer, and STUDIO-STATUS-PAGES-001 for public publication and iframe compatibility.
The retained local receipts and performance measurements are recorded in the [acceptance ledger](implementation.md).
