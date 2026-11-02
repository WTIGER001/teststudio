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
