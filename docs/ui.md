# Interface design and verification

## Usability and documentation contract

This document is the shared design reference for current and future TestStudio pages.
Apply these rules when adding or changing a workflow. Reuse the existing components and feature services.
Record exceptions, their reason, and remaining work in the implementation ledger.

### Design for tester tasks

Lead with the task: create a test, run it, understand its result, or inspect its evidence.
Use the same visible names in navigation, headings, actions, help, and user guides.
Keep project, application, suite, target, and protection context visible where they affect the task.
Projects own applications and suites. Suites link to applications and contain tests. Environments belong to applications.
Carry the selected suite and application into the next workflow.
Generate identifiers and retain sensible defaults. Keep identifier editing under a named Advanced section.
Accept familiar input directly. A quick HTTPS check accepts a complete URL without requiring separate setup pages.
Empty states explain the next available action. Access failures and unavailable data must not look like empty registries.

### Choose the smallest useful interaction

Use a wizard with a labeled stepper for configuration that spans several decisions or creates related resources.
Group steps by the tester's decisions. Keep a short form or direct action for a simple task.
Use dedicated routes for substantial authoring workflows. Small edits can remain beside their context.
Keep common controls visible. Put uncommon filters, tuning, pins, and identifiers under clearly named Advanced sections.
Keep target protection, safety, enable state, and execution or notification consequences visible on review.
Do not hide consequences under Advanced options.
Show the final resources and effects before saving related configuration or starting execution.
Keep Save draft, publish, activate, run, and enable actions distinct. Label combined actions explicitly.
Do not add a confirmation to every ordinary action. Use review where it resolves a meaningful decision or consequence.

### Preserve work and explain recovery

Back and completed step navigation preserve values. Lookup refreshes and help must not reset edits.
Validate the current step before continuing. Explain the correction beside the field or in a clear error message.
Reveal invalid Advanced options and move focus to the field after rendering.
Move focus to the new step heading when the step changes.
Identify successful steps after a partial save. Explain what retry does and prevent duplicate records or runs.
State when an in-page continuation ends on reload. Do not imply unsaved fields survive a reload unless they do.
Keep revision conflicts, permission changes, unavailable data, and failed execution distinct.
Provide a relevant retry, reload, or return action when recovery is available.

### Keep every supported workflow documented

| Audience | Required documentation |
| --- | --- |
| Tester | Current-page Introduction, Concepts, Tutorials, and a linked user guide for the workflow. |
| API user | Request and response contracts, permissions, validation, revisions, side effects, and retry behavior when relevant. |
| Operator | Dependencies, configuration, installation, upgrades, backup, restore, and recovery for supported profiles. |
| Contributor | Shared UI patterns, affected services and components, targeted validation, and remaining work. |

Update the affected documentation in the same delivery as the behavior.
Use current control labels and task-based examples. Write short, active sentences.
Explain defaults, target requirements, safety, permissions, persistence, and failure recovery where they affect a task.
Mark unsupported behavior and unverified deployment profiles explicitly.
Keep shared rules here. Feature guides explain their own behavior and link to shared concepts.
Use [Page help](help.md) for guide structure and [the plan](plan.md#workflow-completion-checklist) for the completion checklist.

### Verify the changed workflow locally

Implement first, then select the smallest useful checks for the changed behavior.
Use pure utility checks for meaningful parsing or transformation rules.
Use a targeted TestStudio Playwright workflow for changed interaction or execution behavior.
Check keyboard focus, relevant themes, narrow layout, preserved values, and errors in the affected path.
Reuse passing evidence when the relevant behavior stays unchanged.
Document edits require consistency and link checks. They do not require a UI build or browser run.
Keep cloud acceptance separate from local verification. Close every verification browser and preserve the user's tabs.

## Implementation patterns

The Angular application uses standalone components, OnPush change detection, and signal-based services.
Components delegate workflows to focused feature services.
Pure utilities handle catalog filtering, stable sorting, category paths, category counts, and hidden selection counts.
The catalog uses shared rows for its table and category navigation views.
Common search, suite, application, type, and sort controls stay visible. Metadata and lifecycle filters use Advanced filters.
Row editing and applicable lifecycle actions stay visible. History, duplication, and archive use More.
View and filter changes preserve selection. The bulk bar reports selected tests hidden by filters.
Bulk metadata uses a focused editor, signal-based service, and pure field validation utilities.
Review previews show compact before and after values beside optional complete definitions.
The catalog manages reusable saved selections and versioned filters.
Run forms require a resolved preview for saved selections. Run details shows the retained selection name and revision.

Project settings provides suite inspection, editable settings, archive, and restoration.
The focused manager shows readable affected schedules and preserves release pins.
Revision conflicts require a reload before another mutation. See [Suite settings](suites.md).

## Sidebar navigation

The Project selector chooses the current project. Workspace contains Projects and Your account.
Testing contains Applications, Suites, Test catalog, Test runs, and Trends.
Applications provides application environments. Suites provides test groups and scoped catalog links.
Automation contains Schedules and Notifications.
Advanced testing contains Import and sync tests and Suite releases.
Project administration and Installation administration group access controls by ownership scope.
Automation, Advanced testing, and administration start collapsed. Their current page opens the relevant section.
Creation routes keep Test catalog marked as the current destination.
Native disclosure controls support keyboard operation. Active links expose the current page.
Existing capability checks still control each protected link.
On narrow screens, Navigation opens and closes the menu. Selecting a link closes it and returns space to the current page.
Sidebar styles stay within focused components and use shared theme tokens.

## Run investigation

Use Inspect test to move from a logical result to its attempts, logs, and artifacts.
Keep one shared test and attempt selection across these evidence sections.
Default to all attempts so a passing retry cannot hide an earlier failure.
Use cards for focused attempts and artifacts. Keep primary investigation actions in the first results column.
Constrain select controls to their field width. Verify readable content width as well as document overflow.
Open failed recorded steps within the selected test. Keep technical log filters and execution events under named Advanced disclosures.
Move focus to the investigation heading and artifact preview heading after the corresponding action.
Close preview returns focus to its triggering control when it still exists.
Preserve investigation filters and action errors during Refresh. Hide rejected data after permission loss.
Explain ambiguous mutation responses and reuse the same request key for an in-page retry.
State the reload limit of that continuation. Keep saved server records separate from temporary page state.

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
The project Trends page provides overall, category, and permanent-test views with shareable UTC filters.
The chart uses connected stacked areas and breaks paths at selection gaps.
Hover and keyboard focus expose state counts. An accessible values table links to original runs.
Failure and volatility rankings retain sample counts and link to test and category history.
History pages expose partial coverage. Request failures remain distinct from valid empty scopes.
Local self-tests verify real outcome history, gaps, ranking links, URL persistence, themes, and mobile overflow.
Cross-project rollups and HTTPS response-time and certificate history remain pending.

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
HTTP attempts expose expandable network phases and certificate observations.
Missing phases remain unavailable. Certificate lifetime uses the retained observation time.
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
Its native HTTP controls support GET, HEAD, statuses, text, headers, JSON fields, response-time limits, and certificate lifetime thresholds.
Field errors block saving. Changing to HEAD preserves conflicting body fields for explicit correction.
The type filter separates HTTPS, API, browser, and other definitions independently of their runners.
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

## Schedules

The project schedule page provides interval and cron configuration, execution selection, release pins, timing previews, and firing history.
Public configuration revisions stay stable during runtime polling.
Unsaved fields remain intact while schedules fire. A project change closes the previous editor.
Pause controls future firings. Archive preserves identifiers and history. Restore leaves the schedule paused.
Triggered firings link to normal runs. Blocked and skipped firings retain their reasons.
The page includes contextual help and supports light, dark, and narrow layouts.
The focused self-test verifies the real local scheduler through TestStudio's publication, execution, timing, and evidence workflow.

## Notifications

The project notification page provides channel configuration and durable delivery history.
The editor selects registered secret names and exact scope filters. Timeouts remain distinct from failed outcomes.
New channels start paused. Settings apply to future runs, and open edits survive polling and page help.
Archive preserves history. Restore leaves the channel paused.
Run details include a delivery view restricted to that run.
Delivery state, frozen test outcome, HTTP responses, and unknown acceptance remain separate.
Explicit retry and stop controls operate on delivery work without rerunning tests.
History shows bounded coverage and pauses automatic refresh after older pages load.
The page includes contextual help and shared light and dark theme tokens.
Tables scroll inside their containers on narrow screens.
The focused self-test verifies manual and scheduled completion, real retries, Discord confirmation, stopped delivery, and preserved history.
Checked light and dark views have no document overflow or browser runtime errors.

## Identity administration

Users and Groups use focused page services, small standalone components, and an OpenAPI-generated identity client.
Installation capabilities control Users and Installation groups navigation independently from project capabilities.
Project groups retain an explicit ownership scope. Editors cannot change that scope.
User details show profile state, linked identities, readable group membership, and active sessions.
Group details show membership origin, activity, UTC expiry, and permitted local membership controls.
External membership remains read-only. Archive preserves records and removes effective membership access.
Restore can reactivate retained membership and its assigned access.
Lifecycle actions and session revocation use explicit reason forms with target and impact text.
Revision conflicts preserve unsaved values and offer explicit reload controls.
Scope changes close old editors. Failed reads remain distinct from empty bounded pages.
Each new page includes contextual help with Introduction, Concepts, and Tutorials.
Shared tokens support light and dark themes. Mobile detail actions sit below the heading, and tables scroll within their panels.
The focused self-test verifies real local API mutations, conflicts, membership, archive, restoration, administrator protection, pagination, and session revocation.
The checked views have no document overflow or unexpected browser runtime errors.

## Role assignments

Project access and Installation access provide scoped predefined role administration.
Focused services handle permissions, bounded collections, creation, and reason-based removal.
The shared generated client supplies the same API contract as the service.
Role choices expose registered actions and current delegation capability.
Assignment rows use server-provided removal capabilities, including shared-group ownership checks.
Creation and removal conflicts preserve entered values and require explicit review.
Self-revocation refreshes project choices and capabilities, then clears unavailable private collections.
Changing scope closes previous editors and invalidates old responses.
Each page includes contextual help and uses shared light and dark tokens.
Narrow screens keep readable table columns inside a keyboard-focusable horizontal scroll region.
The focused self-test exercises real project-administrator and read-only sessions, mutations, conflicts, pagination, help, and administrator protection.

## Predefined role state

Installation roles also provides state controls for all six predefined roles.
These controls change disabled state across the installation. Project role pages remain read-only for predefined state.
The server supplies current management capability and checks installation authority, delegation, and every default action.
Reason forms identify the global impact and retain their opened revision through refreshes.
Conflicts and unconfirmed responses require explicit state review before another submission.
Revoked authority clears private controls. Contextual help explains disabling, restoration, and retained assignments.

## Direct permission grants

Project permissions and Installation permissions provide scoped direct user and group grant administration.
Structured creation fields show principal ownership, exact objects, registered action descriptions, effect, precise UTC expiry, and change reason.
Object changes invalidate action choices until explicit refresh.
Expiry editors retain their opened revisions and preserve unsaved values during list refresh and conflicts.
Removal displays the exact rule and its effect on access before requesting a reason.
The interface blocks repeat submission after an unconfirmed change until explicit state review.
Authorization failures clear private editor and list data, including responses with unreadable bodies.
Scope changes discard old editors and responses.
Server capabilities control creation, expiry editing, and removal. Read-only sessions retain inspection controls.
Each page provides contextual help. Shared tokens support both themes, and labeled table regions contain horizontal scrolling on narrow screens.

## Guided configuration

Suites provides a first-class list with View tests, Create test, and Suite settings actions.
New suite uses the existing Name, Options, and Review registration wizard.
Schedule creation and editing uses Tests, Timing, and Review steps.
Advanced selection options contains identifiers, release pins, explicit test identifiers, and execution location.
Advanced run options contains overlap, limits, and retries.
Notification creation and editing uses Destination, When to notify, and Review steps.
Advanced identity options contains the permanent identifier.
Advanced filters and delivery options contains scope filters and automatic attempts.
The final review keeps enable controls and their future execution or delivery consequences visible.
These flows reuse the existing services, permissions, conflict handling, preview, and save contracts.
