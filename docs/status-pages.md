# Public status pages

Each project can create separate status pages for different audiences. A page contains an ordered selection of current HTTP probe bindings or direct test references.
Each row has its own public title, optional description, bucketed history pills, and an availability donut or outcome summary. A page supports up to 100 probes; project page lists continue through pagination.
The page title, optional centered logo, background color, and default time range are configurable. One viewer selector changes every row together. Outcome rows show pass rate and excluded skipped/unknown counts; explicitly monitored rows show time-based availability.
Public pages have standalone URLs and support iframe embedding without sign-in.

## Optional logo

Choose Logo source on the Appearance step: No logo, Upload image, or Image URL.
Uploads accept PNG, JPEG, and WebP files up to 10 MB. The browser preserves transparency, removes source metadata, and resizes the image to fit 800 by 320 pixels, reducing it further when needed to keep the stored PNG data URL within 256 KiB.
The API also accepts validated PNG or JPEG data URLs within 256 KiB and 1600 by 1600 pixels. Logo alternative text supports up to 256 UTF-8 bytes; the viewer defaults to the page title plus “logo”.
References require a public HTTPS image URL up to 2048 bytes, without embedded credentials. The service never fetches the URL. Viewers load the image directly from its host without sending the status-page address as a referrer. A referenced file must stay publicly reachable; changing that file changes what viewers see without a new TestStudio publication. Upload an image when the published logo must remain fixed.
The logo centers above the title, preserves its proportions, and fits within 320 by 128 CSS pixels. Missing logos add no header space. An image that cannot load shows Logo unavailable while probe status remains visible.
Choose Remove logo or No logo to clear it from the draft. Saving, replacing, and removing a logo remain private until reviewed publication. Unpublishing withdraws the page report; it cannot withdraw an independently hosted image or copies already received by a viewer.
Uploaded logos are public page content after Publish. Private Project cover images stay separate and are never selected automatically.

## Vanity URLs

Set Vanity URL on the Appearance step to use an address such as /status/customer-health.
Names use 1–63 letters, numbers, or hyphens and ignore case. They start and end with a letter or number. Generated 40-character hexadecimal addresses are reserved.
Availability feedback is advisory. Saving atomically reserves the normalized name across the installation, including other projects and concurrent saves. No other project's identity is disclosed by the availability check.
A saved draft reserves its name but does not make the page public. Publishing activates the saved vanity address. Changing a draft leaves the currently published URL working until publication.
Publishing a changed name retires the previous vanity address. The permanent generated URL continues working. Retired names stay reserved for the original page, including after deletion, so an existing embed cannot be taken over by another page.
Unpublishing withdraws both addresses. The sharing controls show the currently published vanity URL and embed code; the permanent generated URL remains available under its disclosure.

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
Shared result capture maintains compact observations and hourly and daily summaries for all runner types. Reports combine saved durations and counts with exact partial edges, without scanning complete project runs on each refresh. Aggregation that cannot finish returns unavailable rather than a misleading partial percentage.

## Publishing

Create a private draft on the dedicated setup page. Choose appearance, ordered checks and public text, then review and save. Saving returns to the status-page detail screen. Use Edit status page for the focused editing workflow.
Review publication opens a separate preview and confirmation page. The detail screen stays focused on configuration summary, sharing, and saved preview.
Preview the saved draft before Publish. Publishing makes only the reviewed snapshot public; later draft edits stay private until published.
The public projection contains public titles, descriptions, statuses, observation times, and duration aggregates.
It contains no target URLs, project or user identifiers, run links, request data, diagnostics, credentials, or private evidence.
Hover or focus a pill for its interval, result counts, downtime, and unknown coverage. Select it to inspect details. Public pills do not link to private runs.

Reader and higher project roles can inspect status-page configuration. Contributor and Owner can manage and publish; only Owner can delete.
Existing compatibility role bundles predate these permissions and do not acquire them automatically. If Status pages is missing, inspect the selected project’s effective statuspages permissions. An authorized administrator can grant statuspages.read, statuspages.manage, and statuspages.publish explicitly for that project without converting the account to Owner or changing its existing roles.
System Admin retains management and recovery authority. Application-only membership does not grant project status-page administration.
Public reads recheck the publisher's current authority, referenced probes, and per-run access. Disabling or revoking the publisher withdraws the page until an authorized publisher publishes again.
Unpublish withdraws the public URL immediately. An already loaded embed clears its display on its next refresh, normally within sixty seconds.
Publishing and editing use revisions and audited mutations. A conflict preserves unsaved work and requires inspection before retry.

## Embedding

Use Copy Public URL or Copy Iframe embed code beside the sharing fields. Each button confirms when its value is copied. If browser clipboard access is unavailable, select and copy the visible read-only field.
The public page refreshes once per minute without executing probes.
Use the configured monitoring schedules to produce observations; viewing a page never runs a check.
The public document permits framing. Signed-in TestStudio pages retain their framing protection.
The AWS package includes a separate /status/* behavior and response policy for public documents. Existing private UI and evidence policies remain separate.

### Native Angular embedding

The reusable [Angular library](../web/projects/status/README.md) provides teststudio-status-bar and teststudio-status-page for external Angular sites.
Hosts can supply typed report data or use the optional public reporting client. Styling tokens and templates support host appearance while preserving accessible states and unknown coverage.
Public reads retain explicit publication and withdrawal. The public API permits credential-free cross-origin reads; private endpoints retain their own restrictions.
Private hosts supply reports through their normal authenticated services. Component installation does not publish a test or grant access.
The public URL and iframe remain supported alongside native Angular embedding.

## Include other test types

On Checks, use Add HTTP probe for an existing binding or expand Other test types for any registered test.
Choose its environment and execution location. Leave the location blank for the installation default. Use No target for targetless tests.
Execution outcomes are the default for direct tests. Report this test as a monitor uses the page freshness limit; it creates no schedule and runs no tests.
Review shows the metric for every row before saving. Publishing remains a separate reviewed action.
Hour, six-hour, daily, weekly, monthly, and yearly pills keep long histories compact. A successful latest result cannot hide an unsuccessful observation earlier in the bucket.
Monitoring definitions and source/target changes retain separate contexts. Old configurations do not establish current availability.

## Inspect individual checks

Published pages retain the bucketed overview and offer Individual checks on each row. The table shows one final logical result per execution, with completion time, outcome, and measured duration. It does not aggregate results or infer uptime between them. Select a pill and Inspect period to inspect that interval; Return to full period restores the table range. Load older checks continues through retained detail with no fixed three-hour or six-hour restriction.
Public tables omit private run identities, targets, diagnostics, and evidence. Private Test history offers Inspect check under current run authority. The shared Angular components support the same table. See [individual check APIs](trends.md#individual-check-api) for boundaries, pagination, and retention.
