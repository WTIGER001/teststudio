# Interface design and verification

The Angular application uses standalone components, OnPush change detection, and signal-based services.
Components delegate workflows to focused feature services.
Pure utilities handle catalog filtering and counts.

## Themes

Light, dark, and system preferences use shared semantic CSS tokens.
The browser stores the selected preference under `teststudio.theme`.
An external initialization script applies the theme before the first render.
System mode follows operating-system changes without a page reload.
The build disables inline critical-style scripts to preserve the server content policy.

## HTML and access

The shell uses header, navigation, main, and complementary landmarks.
A keyboard skip link moves focus to the main content.
Forms use associated native labels, required fields, and compatible identifier patterns.
Tables use captions and scoped column headings.
Status labels include text. Color is not the only signal.
Visible focus and reduced-motion preferences apply in both themes.
Narrow layouts wrap navigation and controls. Wide tables scroll inside their own container.

## Page help

The header provides Help on every page, including sign-in.
A right-side drawer shows the current page's Introduction, Concepts, and Tutorials.
The drawer keeps page edits intact and restores focus when closed.
Close, Escape, and the desktop backdrop dismiss it.
Section controls move keyboard focus into the chosen section. Help content scrolls independently.
The drawer supports both themes, full-width mobile display, and reduced motion.
Guide structure and maintenance instructions are in `docs/help.md`.

## Trend chart contract

Result-state trends use stacked area charts across overall, category, and test views.
Stack passed, failed, timed-out, flaky, skipped, and unknown or incomplete counts together.
Timeouts use a separate blue status color in both themes.
Keep stack order and state colors consistent in light and dark themes.
Use final logical results for counts. Retries do not add results to the chart.
Show state counts and totals in tooltips and an accessible data table.
Missing history creates visible gaps. Incomplete results do not become passing observations.
Trend implementation and browser verification remain pending.

## Verified local workflows

A browser check creates a new generic project, application, suite, environment, and HTTPS definition.
It validates and publishes a catalog release, then archives and restores the test.
It verifies theme persistence, operating-system theme changes, visible keyboard focus, and a 390-pixel viewport.
The final page has no console errors.
The test browser and its automation process exit after verification.
Run history provides environment, suite, state, and selection filters with bounded pagination.
The run form selects a published suite, environment, test identifiers, time limit, and retries.
The time limit accepts 1 through 3600 seconds, consistent with the run API.
Run details show the execution snapshot, progress, final counts, attempts, events, and filtered logs.
Counts and status badges separate timed-out tests from failed tests.
Rerun failures includes failed, timed-out, and flaky results.
Active runs refresh every two seconds. Cancellation and failure reruns use explicit actions.
Artifact previews support text, raster images, and video. Other content uses a download link.
The browser retrieves artifact content through short-lived access links on a separate origin.
Local browser checks verify successful runs, failure reruns, cancellation, text previews, and file downloads.
The run pages support light and dark themes and a 390-pixel viewport without document overflow.

Catalog transfer supports JSON upload, export download, field-specific validation, and explicit preview application.
Bulk controls select tests within one suite and preview disable, enable, archive, or restore actions.
Duplicate creates a new draft identity. History displays retained definitions in revision order.
Preview and detail headings receive keyboard focus when they open.
Local browser checks verify these actions, invalid import rejection, theme rendering, downloads, and mobile overflow.

Run retention shows pins, expiry, processing state, storage usage, and the frozen capture policy.
Pin forms preserve unsaved edits while polling. Revision conflicts provide a reload action.
Artifact deletion uses an explicit reason form and shows cleanup state without changing test outcomes.
Completed deletion returns keyboard focus to the artifact heading.
Local browser checks verify pin creation, timed update, removal, deletion, quota reduction, and retained results.
Light, dark, and mobile retention views pass visual checks without document overflow.

Timing estimates show expected execution, baseline readiness, slowdown thresholds, attempt timeouts, and retry timeout budgets.
Changing a run request clears its stale estimate. Incomplete history remains explicit.
Run details compare actual and expected durations while retaining functional outcomes and cumulative retry time.
The catalog editor validates timing overrides and publishes them through the normal release workflow.
Baseline controls show current history and require a reason and revision for reset.
Local browser checks verify a real slowdown failure, baseline reset, frozen historical criteria, policy editing, publication, and budget warnings.
The checked timing views support light, dark, and mobile layouts without document overflow or console errors.

Playwright attempts expose collapsible step lists with durations, categories, and errors.
The list has a bounded scroll region. Native HTTP attempts keep their existing observation view.
Local self-tests run through TestStudio and retain screenshots, browser diagnostics, logs, and failure traces.
Browser inspection verifies the recorded-step view without console errors or mobile document overflow.

Run details show worker resource state, saved checkpoints, recovered observations, and explicit capture gaps.
Pending cleanup refreshes every five seconds. Worker records use bounded pagination.
Recovered attempts retain the original run outcome. Missing measurements show Duration unavailable.
Resource references remain inside an expandable section. The worker table scrolls within narrow screens.
Local browser checks verify interrupted-run diagnostics in light, dark, and mobile layouts.
The checked views have no console errors or document overflow. Owned verification processes exit after checks.
