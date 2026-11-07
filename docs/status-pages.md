# Public status pages

Each project can create separate status pages for different audiences. A page contains an ordered selection of current HTTP probe bindings.
Each row has its own public title, optional description, recent result pills, and an availability donut.
The page title, background color, and default time range are configurable. One viewer selector changes every row together.
Public pages have standalone URLs and support iframe embedding without sign-in.

## Availability

Availability is estimated time up divided by known monitored time. It is not the percentage of checks that passed.
A completed observation applies from its completion until the next observation or its freshness limit, whichever comes first.
The default freshness limit is fifteen minutes. Advanced monitoring settings can change it from one minute through one day.
Missing history, stale intervals, changed targets or assertions, and infrastructure failures are unknown. Unknown time is shown separately in the donut and monitoring coverage.
Failed assertions, target timeouts, and flaky results count as unavailable. Duplicate attempts do not add observations.
Late completion of an older logical run cannot replace a newer observation.

Ranges are rolling 24 hours, seven days, one month (30 days), one year (365 days), and all retained history.
All time starts at that probe's first retained observation. No data has no percentage.
A result just before a range boundary can contribute until it becomes stale or is replaced.
The history calculation traverses retained project history without the private overview's 1000-run limit. A request that cannot finish returns unavailable rather than a misleading partial percentage.

## Publishing

Create a private draft. Choose appearance, ordered probes and public text, then review and save.
Preview the saved draft before Publish. Publishing makes only the reviewed snapshot public; later draft edits stay private until published.
The public projection contains public titles, descriptions, statuses, observation times, and duration aggregates.
It contains no target URLs, project or user identifiers, run links, request data, diagnostics, credentials, or private evidence.
Hover or focus a pill for its status and local timestamp. Public pills do not link to private runs.

Reader and higher project roles can inspect status-page configuration. Contributor and Owner can manage and publish; only Owner can delete.
System Admin retains management and recovery authority. Application-only membership does not grant project status-page administration.
Public reads recheck the publisher's current authority, referenced probes, and per-run access. Disabling or revoking the publisher withdraws the page until an authorized publisher publishes again.
Unpublish withdraws the public URL immediately. An already loaded embed clears its display on its next refresh, normally within sixty seconds.
Publishing and editing use revisions and audited mutations. A conflict preserves unsaved work and requires inspection before retry.

## Embedding

Copy the published URL or its iframe markup. The public page refreshes once per minute without executing probes.
Use the configured monitoring schedules to produce observations; viewing a page never runs a check.
The public document permits framing. Signed-in TestStudio pages retain their framing protection.
The AWS package includes a separate /status/* behavior and response policy for public documents. Existing private UI and evidence policies remain separate.
