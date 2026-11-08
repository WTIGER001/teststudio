# Implementation verification

The full plan remains required scope. Check an item only after its implementation and verification pass.

Implementation resumes at the user's request on 2026-11-03.
The acceptance count is 50 verified of 57.
Focus on implementation and targeted checks. The user now authorizes MMDev ECS deployment. Azure remains deferred.

## Status-page sharing copy buttons — 2026-11-07

Public URL and Iframe embed code now have their own copy buttons. A shared clipboard service performs the write; the shared button owns local busy and accessible success or fallback feedback. Visible read-only fields remain available for manual copying.
Page help and the feature guide describe the controls. The current published URL and exact displayed iframe markup are copied, including an active vanity name.
The UI compiles. STUDIO-STATUS-PAGES-001 passes in 6.709 seconds through normal local execution, run 20261108T025119.540693000-f97987f5b8ae0d58e4383b5e. It verifies real clipboard contents for both values, keyboard activation, denied clipboard access, visible fallback, successful retry, and desktop and 390-pixel sharing layouts, alongside the existing publication workflow.
The execution is cleaned and owned controllers and browsers exit. A separate clean checkout builds committed source without concurrent reliability changes. Unchanged packaging evidence from 0.5.0-dev is reused; no Go behavior or package configuration changes.
Release 0.5.1-dev uses committed 4e679d49a962d4f8803ed4669bdc7bc4301c7c27, build 969aad6abb87, created 2026-11-08T02:50:41.631Z with modified false. The ARM64 digest is sha256:309d4ee08088bd21c9cf6fcc121aefcea4b6d7b8f437feef8d08b1ca987baa63.
The reviewed CloudFormation change set modifies only TestStudio Service and Task. Task startup is 2026-11-08T02:54:03.467Z; matching UI publication is 2026-11-08T02:57:29Z.
Sixteen hosted checks pass for API identity, exact published UI and component bytes, health, expected task digest, completed stack and service rollouts, and completed CloudFront invalidation. Both teststudioapi and testing-server retain their completed rollouts.
The requesting account’s browser confirms both buttons beside its existing sharing fields on 0.5.1-dev. No page values or publication are changed. Its clipboard is not overwritten during the hosted inspection. The temporary tab closes and both original tabs remain.
Receipts and screenshots are output/aws-mmdev/status-copy-051 and output/selftest/20261108T025119.540693000-f97987f5b8ae0d58e4383b5e. The temporary build checkout is cleaned after evidence is retained. The patch and release documentation are pushed; concurrent reliability edits remain separate.
This refines 49 and 50. Acceptance stays **50 verified of 57**, with the same seven broad requirements open. Existing projects, memberships, probes, schedules, and the port 8040 demo remain unchanged.

## Shared reliability history: approved scope — 2026-11-07

The user approves the rollup and bucketed-pill approach for every existing and future probe and test type as part of common result capture.
The [shared design](trends.md#planned-shared-reliability-capture) defines canonical logical observations, durable aggregation work, hourly and daily summaries, longer-range display buckets, safe replay, retention, and access boundaries.
Every runner supports outcome and timing history. Continuous availability requires an explicit monitoring policy and freshness limit. Sparse test executions do not imply uptime.
The user extends the plan to a reusable [Angular component library](trends.md#reusable-angular-components) with a stylable status bar and complete page for other Angular sites. The planned contract includes typed data inputs, an optional client, scoped styling tokens, internal reuse, explicit public and private access, lifecycle cleanup, and independent-consumer verification.
The active plan and continuation notes retain the delivery sequence and distinguish this request from the pending status-page release.
This is a documentation milestone. Local consistency and link checks apply; it establishes no runtime, deployment, or cross-profile evidence.
Scope extends 34, 35, 36, 37, 39, 40, 41, and 47, with supporting 12, 13, 49, 50, 56, and 57. Acceptance stays **50 verified of 57**; no item advances to verified from planning.

## Focused status-page management and vanity URLs: MMDev delivery — 2026-11-07

Release 0.5.0-dev uses committed 29f540058510e5646d59b202b94094907a95355f, build a21537b24309, created 2026-11-08T02:38:53.301Z with modified false. Its ARM64 digest is sha256:e801ea09f23c383293fb79be86aea36c429ef417a47c1a9fcb166162521942e3. All branch commits are pushed.
The twelve portable-package checks pass. The reviewed CloudFormation change set modifies only TestStudio Service and Task. Existing foundation resources and testing-server remain unchanged.
Task startup is 2026-11-08T02:41:02.341Z; UI publication is 2026-11-08T02:44:15Z. CloudFormation reaches UPDATE_COMPLETE, CloudFront invalidation completes, and both services retain one healthy completed rollout. The API, published index, scripts, and new editor bundle match the selected artifact.
Thirty-four hosted checks pass through normal bootstrap OIDC and independent anonymous HTTP requests. They verify normalized names, private draft reservations, explicit publication, global duplicate rejection, availability checks from another project, live alias reports, no-store responses, iframe document access, rename snapshot isolation, permanent generated links, retired-name protection, and immediate unpublishing.
The owned generic page reuses the earlier explicit verification probe; no new test execution occurs. Its publication is withdrawn, its session is revoked, and the temporary helper is removed. Private receipts are output/aws-mmdev/status-vanity-050.
The requesting account’s actual browser shows the saved FoodMarketMaker page with separate Edit and Review publication links. Edit opens the focused Appearance step with the new vanity field and preserves the saved title and default period. No values are changed or published. The temporary verification tab closes and both original user tabs remain.
This refines 05, 12, 13, 25, 49, and 50. Acceptance stays **50 verified of 57**; broad worker and cross-profile requirements remain open. The port 8040 demo, existing probes, schedules, memberships, and page publication remain unchanged.

## Focused status-page management and vanity URLs: local delivery — 2026-11-07

Status-page details now summarize configuration, publication, sharing, and a saved draft preview. Dedicated create, edit, and publication-review routes follow Create project, with back and cancel destinations, step validation, preserved edits, and revision recovery.
The shared UI guide now defines when to use dedicated pages, short dialogs, and small inline controls. Page help and the status-page guide describe the current controls.
Optional vanity names normalize case and reserve globally in the same transaction as the page and audit. A competing save returns an explicit conflict. The advisory availability endpoint reveals only whether a name is available.
Publication activates the saved alias; draft renames preserve the current published address. Republishing retires the old alias. Permanent generated links remain compatible. Reservations bind the original page token, so deletion and recreation cannot take over old links.
Canonical OpenAPI 0.39.0 and the generated client describe the optional name and availability endpoint. Separate release and target evidence appears above.

The focused SQLite HTTP cases pass. The same PostgreSQL cases pass with race detection in 3.360 seconds, including concurrent claims with one winner, global duplicates, normalization, invalid names, draft and publication isolation, generated-link compatibility, and deleted-page takeover rejection.
The UI build and client synchronization check pass. STUDIO-STATUS-PAGES-001 passes through normal local execution in 6.497 seconds, run 20261108T023727.088360000-03fb9e6133bcc031ea0886d4.
It covers duplicate-name feedback and correction, dedicated authoring and publication routes, saved draft details, a 390-pixel authoring view, help and unsaved navigation, independent anonymous pages, actual cross-origin embedding, all availability periods, privacy, themes, rename activation, permanent links, revision conflicts, recovery, and unpublishing.
Screenshot review corrects stale duplicate feedback after a valid replacement. Owned local controllers, database containers, execution containers, and browsers exit; the port 8040 demo and user data remain unchanged.
Retained evidence is output/selftest/20261108T023727.088360000-03fb9e6133bcc031ea0886d4. Release publication and hosted checks are recorded above.
This refines 05, 12, 13, 25, 49, and 50. Acceptance stays **50 verified of 57**; the seven broad open criteria remain unchanged.

## Status-page sidebar access repair — 2026-11-07

The requesting FoodMarketMaker account runs the current 0.4.0-dev UI but retains the earlier Project administrator, Catalog editor, Publisher, and Operator bundle.
Its effective statuspages.read, statuspages.manage, and statuspages.publish permissions are absent, so the capability-gated sidebar correctly omits the feature. The release verification missed this compatibility-role onboarding gap.
After confirming the signed-in internal account in the user’s browser, normal audited APIs add these three permissions only in foodmarketmaker. No role conversion occurs.
Effective permission checks verify all three are allowed in that project while statuspages.delete remains denied. All status-page permissions remain denied in aws-core and System scope. Original role assignments are verified unchanged.
The actual requesting browser shows Status pages under Project overviews, opens the project’s Status pages route, and displays Create status page. No page is created or published during this repair.
Owned operator sessions close and temporary helpers are removed. Private receipts and screenshot are output/aws-mmdev/status-page-access.
This refines 10, 12, and 13. Acceptance stays **50 verified of 57**. The application version, deployed artifact, probes, schedules, and local demo remain unchanged. Documentation-only consistency and link checks pass; no build, browser suite, or deployment is repeated.

## Public status pages: MMDev delivery — 2026-11-07

Release 0.4.0-dev uses committed implementation 3fc524feb23198590a6140c7f2192249d295fdbc, immutable build 8f7db1066f2e, created 2026-11-07T22:37:40.930Z with modified false.
Its ARM64 registry digest is sha256:c0d43724e049e75681280abd7b502aea46bfff3dee852cb12aee93efca0f56b1. All implementation, operator, and final documentation commits are pushed.
The twelve portable-service checks pass and remove their owned containers and volumes.
The concrete reviewed change set adds PublicStatusHeaders and modifies Service, Task, UiDistribution, and the existing UiRecord alias dependency without replacement.
The alias guard verifies the existing hostname and only UiDistribution.DomainName as its source. Foundation resources and testing-server are preserved.
The selected healthy task starts at 2026-11-07T22:42:06.853Z. UI publication occurs at 2026-11-07T22:46:28Z. Build creation, task startup, and UI publication remain distinct.
CloudFormation reaches UPDATE_COMPLETE. Both services have one desired, one running, zero pending tasks and completed rollouts. CloudFront invalidation completes.
The public index, public document, primary assets, status-page bundles, and API version match the committed artifact.

Twenty-three focused hosted checks pass with normal bootstrap OIDC and independent anonymous HTTP requests.
The generic project status-8f7db1066f2e retains two independent page drafts, two explicit probe bindings, and one normal native HTTPS run.
Run 20261107T224337.430760610-e79998b43e37461d9dd398e9 passes in 140.412653 ms with complete evidence.
Checks cover anonymous draft rejection, preview allowlisting, published default range and background, available versus unknown coverage, private-data omission, no-store responses, all five periods, embedding headers, draft/published isolation, and immediate unpublishing.
Private UI framing denial remains active. Both verification pages are unpublished; the owned session is revoked and temporary helper removed.
Receipts are output/aws-mmdev/status-pages-040. Existing probes, schedules, memberships, the port 8040 demo, and unrelated workloads remain unchanged. No existing probe configuration is published automatically.
This extends 12, 13, 34, 49, and 50. Acceptance remains **50 verified of 57**. Broad coded-worker and cross-profile criteria remain open.

## Public status pages: local implementation — 2026-11-07

Projects can create independent status pages with saved drafts and separate published snapshots.
The dedicated Appearance, Probes, and Review workflow configures title, background, default shared period, ordered HTTP probe references, public probe titles, and optional descriptions.
Public URLs omit the authenticated shell and initialize no sign-in. Cross-origin iframe embedding uses the same minimal status view and explicit public response policy.
Each row shows recent pills, hover and keyboard timestamps, current state, and a right-hand availability donut.
Ranges cover rolling 24 hours, seven days, 30 days, 365 days, and all retained history. Availability uses monitored durations; unknown intervals remain separate and monitoring coverage stays visible.
A freshness ceiling prevents missing checks from becoming uptime. Target and assertion changes, infrastructure failures, and denied observations cannot establish successful availability. Late older logical runs cannot replace newer results.
The history scanner traverses retained records beyond the private overview's 1000-run limit. Native request-scoped permission reads reuse principal policy data without caching authority across requests, preserve exact run denies and frozen application scopes, and reject directory changes during calculation.
Public reports contain no private URLs, resource identities, run links, diagnostics, credentials, or evidence. Publisher revocation and unpublishing withdraw anonymous reads.
Project Contributor can manage and publish but cannot delete. Application-only authority cannot administer Project status pages. Mutations preserve revisions, audit records, drafts, and uncertain-save recovery.

The focused Go calculations and HTTP cases pass on SQLite and race-enabled PostgreSQL, including history older than a year and more than 1000 runs, denied-result redaction, changed targets, publication conflicts, and revocation.
The existing application-run and frozen-scope checks cover the shared scope helper. Eight selected UI utility checks pass. Canonical contract 0.38.0 generates the matching client.
The UI build passes. STUDIO-STATUS-PAGES-001 passes through normal TestStudio execution in 2.843 seconds, run 20261107T223233.325865000-9897ae3890872520b06d7718.
It verifies ordered rows, draft preservation, help, canceled navigation, explicit publication, anonymous viewing without authentication requests, real cross-origin iframe rendering, all windows, configured initial range, public privacy, independent dark pages, light desktop and narrow layouts, retained conflict inputs, recovery, and unpublishing.
Screenshot review corrects initial select binding and very small coverage rounding. The final dark page selects its configured year and displays <0.1% monitored instead of zero.
The local AWS package passes schema and eight packaging checks. Its separate /status/* behavior permits framing while the private UI remains denied.
Owned browsers, controllers, database containers, and execution containers exit. The user's demo and existing project data remain unchanged.
Version 0.4.0-dev is committed as 3fc524f and pushed. The release change set also requires CloudFormation’s dependent update of the unchanged UI DNS alias. The operator guard verifies its existing hostname, no replacement, and sole UiDistribution.DomainName dependency.
Separate publication and target verification are recorded above.
This extends 12, 13, 34, 49, and 50. Acceptance remains **50 verified of 57**.

## Scoped roles MMDev release — 2026-11-07

The user authorizes membership UI, committing and pushing all branch commits, and deployment to MMDev.
Implementation 28be433 includes the previously committed API 96e1bb8 and compact probe pills 457a4c5. All branch history is pushed to origin/codex/implementation.
Version 0.3.0-dev uses immutable build 00135cfbf35a, created 2026-11-07T21:40:22.138Z from committed source with modified false.
The ARM64 registry digest is sha256:4bfcc4c6c4d8052a3a58126fcd90f12508ee54081526cb3b6197961ab7de4a7c.
The twelve local portable-package checks pass. Owned package containers and volumes are removed.
The reviewed CloudFormation change set modifies only Service and Task. No foundation or existing testing-server resource changes.
The new task starts at 2026-11-07T21:42:34.514Z. UI publication occurs at 2026-11-07T21:46:35Z; these dates remain separate from build creation.
The stack reaches UPDATE_COMPLETE. Both teststudioapi and testing-server have one desired, one running, zero pending tasks and completed rollouts. TestStudio's selected task is HEALTHY with the expected digest.
CloudFront invalidation completes. Public index, primary scripts, stylesheet, membership bundle, and API version match the local committed artifact. Anonymous project access returns 401.

Fourteen focused hosted permission checks use normal bootstrap OIDC and three independent local-password sessions on PostgreSQL.
A generic project with Alpha and Beta applications verifies inherited Project Member execution, one-application Member execution, rejected sibling capabilities and reads, scoped group Developer authoring, and application explanation provenance.
Application Member starts native HTTPS run 20261107T214431.750002628-170090d8728a73be1477c8ad. It passes in 98.745277 ms with complete evidence and a matching private-download checksum.
After atomic promotion to Contributor, assigning Owner is rejected. Removing that assignment withdraws current-session run access and rejects its already-issued evidence URL. Independent inherited Member access remains.
All four owned sessions close. Three verification users are disabled. The temporary helper is removed, and receipts remain under output/aws-mmdev/roles-030.
Existing user memberships, nine private probes and schedules, the port 8040 demo, and unrelated workloads are preserved. No hosted compatibility conversion runs.
This extends evidence for 10, 12, 13, 49, 50, 54, and 55. Acceptance remains **50 verified of 57**. Coded workers, complete provider parity, and other broad cross-profile requirements remain open.

## Scoped membership interface — 2026-11-07

Project members and Application members manage direct standard user and group assignments with explicit scope, exact user verification, role descriptions, review, atomic replacement, and revision recovery.
System administration uses System access, System groups, System roles, and System Admin labels.
Application members shows readable Project assignments as inherited and read-only. Exact application explanations retain group paths and binding provenance.
Separate Project and Application capabilities prevent application-only managers from receiving Project or System controls.
Contributor cannot select Owner or change an Owner row. Ownership status describes system-managed, owned, and unavailable scopes.
Compatibility conversions preview replaced assignments and every added and removed action. Apply retains the receipt; Advanced recovery prepares a separately reviewed rollback.
Dirty forms survive help and reject canceled navigation. Stale assignments and unconfirmed conversions preserve entries and require inspection.

UI compilation and eleven selected utility checks pass. Normal local TestStudio execution verifies:

| Selection | Passing duration | Retained run |
| --- | --- | --- |
| STUDIO-PERMISSIONS-001 | 3.544 seconds | 20261107T213449.698330000-6e07407bfed5ef10c8be3280 |
| STUDIO-ROLES-001 | 5.798 seconds | 20261107T213603.668805000-61040884b373b5e972597e55 |
| STUDIO-APPLICATION-ACCESS-001 | 4.731 seconds | 20261107T213818.546428000-9bad3740a6101e5f8ad1fb29 |

The new workflow covers user and group assignment, atomic role change, inherited access, application explanations, canceled navigation, help, conversion and rollback, real application-only Contributor navigation, protected Owner rows, and sibling rejection.
The initial checks expose an unnecessary registry request after authority withdrawal; the workspace now requests applications only for a readable parent context. Test corrections use exact role cells, current help text, and expected forbidden responses.
Light desktop and dark mobile screenshots show contained layouts. All execution evidence is complete; owned controllers and containers exit. The user's port 8040 demo is preserved.
Version 0.3.0-dev is prepared through the version helper. The separate release and target verification are recorded above.
This advances the expanded scope of 10, 12, 13, 49, and 50. Acceptance stays **50 verified of 57**; broad cross-profile items remain open.

## Compact recent-check pills — 2026-11-07

HTTPS probe history replaces checkmark tiles with narrow vertical pills in one responsive row.
Passed, failed, timed-out, flaky, and other outcomes use the shared theme colors. Each pill retains its run link, readable label, timestamp, and previous-target or previous-definition details.
The guide and page help explain the colors and interaction. The patch version is 0.2.1-dev.
The UI build passes. STUDIO-PROBE-OVERVIEW-001 passes in 9.227 seconds through normal local TestStudio execution.
Run 20261107T153825.676380000-f359987ef19bbbc8498e69d0 verifies 30 real observations, distinct pass/fail colors, keyboard focus and activation, retained history, and a single row at 390 pixels without document overflow.
Light desktop and dark mobile screenshots pass visual review. Evidence is complete and execution is cleaned. Owned controllers and containers exit.
The first check used programmatic focus while expecting a keyboard-only focus ring. The corrected check uses Tab and Shift+Tab; no product correction was needed.
This refines 34, 49, and 50. The total remains 50 verified of 57. Target deployment is not checked or changed; the hosted release remains 0.2.0-dev.

## System and scoped role APIs — 2026-11-07

The user requests the API side of the role plan. Reader, Member, Developer, Contributor, and Owner now have cumulative native permission definitions alongside unchanged legacy role definitions.
System Admin keeps its compatible identifier. New ordinary project creators receive one managed Owner assignment atomically with the project.
Project roles inherit into applications. Server-resolved Application context controls assignments, registries, catalogs, sources, runs, evidence, trends, probes, schedules, and application-scoped notification channels.
Application-only members cannot obtain complete mixed runs or shared source and mutation payloads. Historical run authorization uses frozen context.
Contributor delegation excludes deletion and Owner changes, including indirect paths through groups, custom policies, direct grants, and legacy administration roles.
Atomic standard-role replacement checks both the previous role identity and its revision. Concurrent replacements produce one winner.
Usable-Owner guards cover assignment, group, user, credential, provider, role-policy, grant, and predefined-role changes. Ownership status exposes unavailable owners for authorized recovery.
Actor-bound conversion previews record potential role action changes, current assignments, and directory revision. Apply and rollback previews preserve custom roles, direct grants, audits, and recovery guards.
API contract 0.37.0 documents the new families and generates the matching identity client. The [access guide](access.md#standard-project-and-application-roles) records endpoints and compatibility behavior.

The initial selected PostgreSQL run passes 59 checks. A focused follow-up passes 18 checks for persistence across connection restart, competing replacements, Owner safeguards, and provider compatibility.
Two final PostgreSQL checks also verify delegation through legacy roles and preservation of the final group Owner path.
Matching SQLite checks pass. Race-enabled SQLite selections cover the new scoped APIs and the final delegation and Owner guards.
Actual local HTTP execution, evidence download, existing-session revocation, old artifact-URL rejection, automatic firing, and post-revocation schedule blocking pass through the normal API and services.
The scheduler checks current execution authority before its overlap-skip decision. Rejected mutations retain their prior state.
The generated-client synchronization check and documentation consistency checks pass. Private credentials and every owned database container are removed after verification.
Retained local summaries are under output/roles-api. No browser, UI build, release publication, hosted membership conversion, or deployment runs.
The user's port 8040 demo, hosted 0.2.0-dev release, existing memberships, and private schedules remain unchanged.

This advances the backend scope of 05, 07, 10, 12, 13, 14, and 45. UI milestone 4 and release/target milestone 5 remain pending.
The count stays 50 verified of 57. Open items remain 41, 46, 51, 53, 54, 56, and 57.

## System and scoped role planning — 2026-11-07

The user keeps Project and Group, selects System, and requests a plan for Owner, Contributor, Developer, Member, and Reader.
The clarified order is Reader, Member, Developer, Contributor, then Owner. Member runs tests; Contributor manages the scope except resource deletion and adding, removing, or changing Owners, including indirect group and policy changes.
The [role plan](roles-plan.md) defines cumulative Project and Application roles, separate System Admin authority, inheritance, delegation, group ownership, shared-resource isolation, and owner recovery.
It preserves narrow existing roles until explicit reviewed conversion and separates application authorization from current display filters.
Five implementation milestones map server scope enforcement, managed membership, migration, UI and help, and target verification to the stable acceptance numbers.
This delivery changes planning documentation only. Current application permissions, memberships, release identity, and deployment remain unchanged.
Documentation consistency, relative links, and the 57-item acceptance count are checked before commit. No application build, browser execution, or deployment is needed.
The count remains 50 verified of 57. Open items remain 41, 46, 51, 53, 54, 56, and 57.

## MarketMaker MMDev deployment launcher — 2026-11-07

The user selects the separate MarketMaker API checkout for a deployment entry point.
Its root deploy-teststudio-dev.py delegates to TestStudio's maintained generic publisher with fixed MMDev account, region, stack, and cluster defaults.
Source overrides and an explicit working directory preserve TestStudio's independence and permit invocation from another directory.
Prepare requires committed implementation and a changed product version for new runtime behavior. Already-published runtime inputs skip rebuilding and redeployment.
The explicit review, execute, and finish phases preserve concrete CloudFormation review and in-progress deployment recovery.
The API README records the commands. Six focused offline checks verify account rejection, working-directory handling, version enforcement, uncommitted runtime rejection, already-current reuse, and in-progress preservation.
An actual invocation from /tmp confirms 0.2.0-dev, build 6caa2b0ca508, is already deployed. No new image build, change set, or deployment is started.
The selected account is 253455159142. The live version and health pass; the core stack is UPDATE_COMPLETE, and both TestStudio and testing-server retain completed healthy rollouts.
This changes operator tooling only. The acceptance count remains 50 verified of 57, with the same seven broad items open.

## HTTPS probes release and private onboarding — 2026-11-06

Milestone 6 completes the requested six-milestone HTTPS probes delivery.
Release 0.2.0-dev, build 6caa2b0ca508, uses committed implementation source 6d2444fcb27a0d03af68c258e36da173adc38b77.
Build creation is 2026-11-06T16:02:07.885Z. The new task starts at 2026-11-06T16:06:11.177Z. Matching UI publication is 2026-11-06T16:09:03Z.
The service image is sha256:eb8ff64d74fcdc9eb06f2c5127bb8b202ee2029d39a0126dd9e0fa70e1651fdb.
Twelve local portable-package checks pass. The concrete CloudFormation change set modifies only TestStudio's Service and Task.
The resulting service is healthy with a completed rollout. The existing testing-server remains healthy with its completed rollout.
Public API metadata, UI index, main bundle, probe overview bundle, and setup bundle match the committed release. API and evidence health pass.

The user selects a new FoodMarketMaker project. It contains Website, Admin, and Search applications, one active native HTTPS suite release, and nine explicit target bindings across production, beta, and development.
All nine initial normal runs pass with HTTP 200, valid TLS, and complete private evidence. The observed initial durations range from 51.8 through 241.8 ms.
The existing requesting user's four normal project roles are verified. No installation administrator role is granted. Anonymous project API access returns 401.
A private evidence download matches its retained size and SHA-256 checksum. Every owned operator application session signs out.
Nine interval schedules use their actual retained AWS execution location, 300-second cadence, zero retries, skipped overlaps, and normal current-owner authority checks.
All nine first automatic firings at 16:18:26Z through 16:18:32Z pass with complete evidence. This proves actual recurring native execution on the selected AWS installation.
The first rejected schedule preview used a workstation location and created no run or schedule. The tracked operator helper now derives the location from the retained hosted run; the normal UI handoff already uses its binding's actual location.

Tracked publish-aws-core.py prepares, reviews, executes, and finishes owned image-only releases without depending on ignored workstation snippets.
Tracked manage-https-probes.py separates inspection, configuration, explicit execution, enablement, retained-result checks, and private-access verification.
Private configuration, credentials, and raw evidence stay outside Git. Retained receipts are output/aws-mmdev/https-probes-release.json and foodmarketmaker-probes.json.
The user's port 8040 demo, database, history, other services, and existing identities remain preserved.
This refines the accepted native HTTPS, results, scheduling, UI, help, access, and evidence workflows. Items 41, 46, 51, 53, 54, 56, and 57 remain open; the total stays 50 verified of 57.
Coded workers and source preparation are not installed by this release. Azure stays deferred. Resume the default worker queue without repeating the completed probe onboarding.

## HTTPS probe overview, actions, and multiple setup — 2026-11-06

Milestones 3 through 5 deliver the project operational overview, server filters, target-aware observations, recent logical checks, TLS and timing diagnostics, and scoped investigation links.
Creation saves explicit bindings; Run now reviews the active release and uses the normal estimate and run APIs with one request identity.
Schedule handoff preserves the exact test, environment, location, and release policy. Shared selection actions show their consequences before confirmation.
Multiple setup uses Addresses, Targets, and Review with current protection, explicit stages, and recoverable per-row saves. Existing binding adoption remains explicit and idempotent.
Visual help adds successful and failed diagnostics and the same schedule Review component used by the editor.
The UI build passes. STUDIO-PROBE-OVERVIEW-001 passes in 1.358 seconds in run 20261106T154952.367637000-5ffe357c21c6e949124d677a.
STUDIO-PROBE-ACTIONS-001 and STUDIO-PROBE-HELP-001 pass in 13.906 and 2.566 seconds in run 20261106T154749.647355000-82092f4dbdf96f9c6b1c276e.
STUDIO-PROBE-SETUP-001 passes in 1.288 seconds in run 20261106T154408.698159000-41b5d1fe777dfad234684740, including duplicate-free adoption and partial-save retry for nine generic addresses.
All recorded executions are cleaned and have complete evidence. The failed filter initialization is corrected and only its affected workflow is rerun.
Owned controllers and database fixtures stop. The user's demo and workloads remain unchanged.
This refines existing catalog, HTTPS, execution, scheduling, results, UI, and help criteria. No cross-profile criterion closes; the total remains 50 of 57.
STUDIO-PROBE-BULK-001 passes in run 20261106T160047.377724000-fbc74e1dca4ff433c8f4cc37 with complete evidence and cleaned execution. Bulk actions preserve request identities and successful rows across partial failure and reconcile a lost archive acknowledgement.
The active-definition boundary passes SQLite, PostgreSQL, and a focused overview rerun 20261106T155446.415808000-2d124bc412b4f78300ce0d21. Draft changes and previously published criteria never relabel old successes as current.
The user selects a new FoodMarketMaker project. Committed release publication and private onboarding remain milestone 6.

## Target-specific HTTPS probe bindings and summaries — 2026-11-06

Milestone 2 adds revisioned project-owned bindings for suite, test, environment, endpoint, and execution location.
Identical creation reuses deterministic identity. Mutations retain current catalog and target authority, reference assertions, audits, and revision conflicts.
The overview reads one indexed batch of at most 1000 newest project runs, filters exact current permissions, and reports partial coverage rather than invented percentages.
Target changes retain old observations with a boundary. Logical results preserve timeout and evidence distinctions. Schedule freshness accounts for due times and reporting grace.
The normal retained history API now accepts environment and location filters. Its client preserves those filters across route query changes.
OpenAPI 0.36 documents the new family and generates matching client types; the product release version remains separate.
Five focused API checks pass on SQLite. The active-definition boundary also passes PostgreSQL. PostgreSQL checks cover target separation, permissions, late completion, bounded coverage, and schedule freshness; cursor and scoped-history parity receive their own focused check.
The generated client synchronization check passes. The total remains 50 verified of 57; this refines 06, 13, 25, 27, 28, 34, 37, 39, 40, and 45.

## HTTPS probe visual help — 2026-11-06

Milestone 1 delivers dedicated probe and creation guides with complete create, run, inspect, activate, and schedule tutorials.
Request URL, Review, and retained HTTP diagnostics reuse application components with fixed example data.
Examples make no mutation or target requests. Current-page highlights preserve values, respect the modal drawer, and restore help scroll and keyboard focus.
The UI build passes. STUDIO-PROBE-HELP-001 passes in 2.572 seconds through normal local TestStudio execution with complete evidence and a cleaned execution record.
Run 20261106T152207.991011000-e5d322390d91e4005be1a202 verifies light and dark themes, mobile width, reduced motion, missing controls, and zero mutations.
The owned controller exits. Target-aware summaries, operational actions, multiple setup, and deployment remain in progress.
This refines 05, 33, 34, 49, and 50. The count stays 50 verified of 57.

## HTTPS probes feature and help planning — 2026-11-06
The user requests a complete plan for HTTPS probes and help with examples rendered from shared application components.
The [feature plan](https-probes-plan.md) defines six delivery milestones and their acceptance mappings and targeted checks.
It covers target-specific identities, bounded summaries, result freshness, observation coverage, creation, publication, scheduling, notifications, recovery, and private onboarding.
Visual help reuses presentational components with isolated example data. Current-page highlights preserve edits and respect the modal drawer.
The plan keeps application-specific addresses out of product defaults and tests. It preserves the existing run, release, permission, and evidence contracts.
The active plan and continuation guide prioritize this latest request before the default AWS worker queue.

This delivery changes documentation only. Consistency, local links, and the stable acceptance ledger are checked before commit.
No application build, browser workflow, probe creation, schedule enablement, version bump, or deployment occurs.
The count remains **50 verified of 57**. Open items remain **41, 46, 51, 53, 54, 56, and 57**.

## Agent continuation handoff — 2026-11-06

AGENTS.md now provides session startup, repository boundaries, focused verification, UI rules, deployment limits, release policy, and Git instructions.
The tracked continuation guide summarizes the deployed release, open acceptance items, next priorities, known gaps, and retained evidence.
The plan marks the latest UI and version work as delivered. Historical checkpoints remain reference material.
README and the plan link the continuation guide. Resume no longer depends on chat history or ignored helper scripts.
The new review records partial OpenAPI coverage for version and project-image endpoints. Their feature guides already describe the APIs.
Documentation consistency, local links, and the 57-item ledger are checked. No build, browser test, deployment, or version bump runs for this documentation update.
The acceptance count remains 50 verified of 57. No new acceptance item closes.

## Semantic release and build identity — 2026-11-06

VERSION starts the tracked product release at 0.1.0-dev. The footer displays the requested four-part form, 0.1.0.dev.
The release helper synchronizes the canonical version with UI package and lock metadata.
The footer shows an immutable build identifier and UTC creation date. Expand Build for source details.
The shared generator stamps the UI and Go API with the same metadata. The image includes standard release labels.
Unchanged build inputs retain their date and identifier. Raw unversioned Go builds report development without inventing a build date.
Mobile Navigation reveals the footer. Current-page help, release guidance, UI rules, and the active plan describe this behavior.

Three focused Node checks and two Go checks pass for release increments, tag syntax, linker metadata, fallback, and the version API.
The Angular production build passes. STUDIO-VERSION-001 passes through local TestStudio execution in 0.532 seconds.
Run: 20261106T141549.201293000-da61d7429e7ac5e9b3ab33b7. Evidence is complete and the execution ledger is cleaned.
The workflow verifies exact UI/API identity, UTC date, expanded details, reload stability, and mobile layout.
Two consecutive metadata generations retain the same date and build identifier. The release helper preserves synchronized version files.
Owned verification browsers, containers, and the local controller exit. No user browser or demonstration data changes.
The final release builds from committed source ca6968f, with working changes false.
A dedicated local package check verifies the actual service endpoint and image labels against the generated snapshot. Its containers exit.
MMDev deploys build e9dc1c0ca5e9 with creation date 2026-11-06T14:18:59.251Z. The stack and service rollout complete.
The public API and UI assets match the same build identity. The existing testing-server remains healthy.
This refines verified items 49 and 50. The verified count remains 50 of 57.

## Square project tiles and private images — 2026-11-06

Projects removes the redundant project total. Square tiles use bounded widths and responsive columns instead of stretching across the page.
Project managers can upload a screenshot or capture a website tab through the browser sharing prompt.
The focused image dialog previews the crop and readable title before an explicit Save image. Remove image restores the default tile.
Capture records one frame and stops sharing. Cancel, failure, and page departure stop active tracks.
The flow uses no hosted browser worker and starts no test run. Unsupported browsers retain the upload path.
Private image content requires current project read permission. Changes require project management permission and the exact cover revision.
The server validates and re-encodes bounded JPEG images. Separate database records preserve project settings revisions and ordinary project payloads.
Audit records retain change metadata without duplicating image bytes. Backup and transfer include the cover records.

Three targeted Go service and API checks pass for private access, revisions, upload validation, removal, and authenticated JPEG responses.
The focused local TestStudio workflow verifies square desktop tiles, narrow layout, upload, persistence, real browser tab capture, cleanup, and removal.
A private virtual display runs this capture check because headless Chromium does not provide the tested capture surface.
Visual review identifies and corrects image-button contrast without changing the screenshot overlay.
Page help, the project image guide, first-installation instructions, UI rules, and the active plan describe both image paths.
Run: 20261106T140014.789307000-4ac204a42ec7e6e01bf4e4e1. STUDIO-PROJECTS-001 passes in 1.688 seconds with complete evidence and cleaned execution records.
The image editor disables input until its current saved state loads, preventing an initial readiness race.
The owned capture display, browser processes, containers, and local verification controllers exit.
The matching API and UI deploy to MMDev. The stack reaches UPDATE_COMPLETE and the service rollout completes with one healthy task.
The public Projects index and changed assets match the built files. Anonymous image metadata access returns 401.
The existing testing-server remains healthy at one running task. The deployment creates no image or test data.
This refines verified items 49 and 50. The verified count remains 50 of 57.

## Dedicated project creation page — 2026-11-06

Projects shows the project list and a Create project action. The action opens the dedicated /projects/new route.
The existing wizard keeps project, application, target, and review steps. Progress stays in one row with contained scrolling on narrow screens.
Back to projects and Cancel setup use the existing unsaved changes guard. Help preserves entered values.
Creation permission remains installation-scoped. The creation route has no active project context.
The wizard keeps its existing review, partial-save receipts, and retry behavior.
Page help, first-installation instructions, UI rules, and the active plan describe this flow.

The Angular production build passes. STUDIO-WIZARD-001 passes through normal local TestStudio execution in 7.533 seconds.
Run: 20261106T131801.535325000-755c948292cb022d7125d961. Evidence is complete and the execution ledger is cleaned.
The check covers desktop and 390-pixel layout, help, cancelled navigation, preserved fields, setup recovery, and the application-scoped destination.
The first attempt exposed an outdated destination assertion. The corrected check retains the existing application scope.
Owned verification containers, browser processes, and local controllers exit after the check. No user browser or demo data changes.
The matching UI is published at teststudio.devfoodmarketmaker.com/projects/new. Public assets match the verified build.
The ARM64 service update reuses all three Go binaries. Its new task and both target groups are healthy.
The TestStudio rollout completes. The existing testing-server service remains healthy at one running task.
This refines verified items 05, 49, and 50. The verified count remains 50 of 57.

## Generic organization OIDC configuration — 2026-11-05

The user selects the existing Cognito issuer us-west-2_p1Epu0go8 and requires generic OIDC behavior.
The separate app-client template uses configurable identity choices and creates one public client in the selected external pool.
The client supports authorization-code flow and the canonical organization callback. Existing pool configuration and original clients remain unchanged.
The service, database, evidence, and runtime remain in MMDev. The external identity app client is in the user-selected MMProd pool.
The managed Organization sign-in provider is configured through the existing protected provider API.
No application authentication source, service image, or browser bundle changes.

The original administrator completes normal sign-in, provider registration, and sign-out.
Actual issuer discovery and application authorization initialization validate client, callback, PKCE, state, nonce, and the code flow.
The request contains no social-provider hint. Cognito returns its own sign-in page and controls its upstream choices.
Interactive authentication and explicit identity linking require the user. No human identity is inferred or linked through email.
Profile creation grants no project or installation roles. The existing administrator provider remains available.
The initial provider-specific staging client is removed after the generic replacement is configured.
Targeted template schema and actual configuration checks pass. No test run or browser process starts.
This advances hosted identity configuration under 54 and 57 without closing their broader requirements. The count remains 50 verified of 57.
See [MMDev organization sign-in](aws-mmdev.md#organization-oidc-sign-in) for exact configuration and receipts.

## Hosted UI address and repository integration — 2026-11-05

The user selects https://teststudio.devfoodmarketmaker.com. The ECS service keeps its name teststudioapi.
Reviewed change sets update only the Cognito client, service task public URL, CloudFront alias, and UI DNS record.
Both stacks report UPDATE_COMPLETE. One ECS task reaches its completed rollout.
Normal OIDC sign-in and private evidence access pass on the new hostname using the original run reference.
The owned probe environment changes to the canonical origin. No additional test run starts for this update.
The service image and static UI assets remain unchanged.

The user also authorizes a separate repository adapter for existing application tests and a generic authoring guide.
The adapter code stays in the application test repository. No existing application test enters the TestStudio source tree.
A new source archive preserves all 604 identifiers, recommendations, conditions, dependencies, and application boundaries.
The canonical publisher assembles both native manifests and discovers 505 Website and 99 FoodSearcher tests in isolated local workers.
This proves bundle and discovery compatibility, not remote application execution.
Source preparation on AWS, reviewed multi-origin browser routing, and a managed credential bridge remain explicit prerequisites.
The generic [test repository contract](test-repositories.md) documents source layout, manifests, selectors, dependencies, targets, secrets, and evidence.
The count remains 50 verified of 57. No cross-profile acceptance item closes.

## Authorized MMDev ECS core deployment — 2026-11-05

The user authorizes deployment into the MMDev ECS cluster, with service name teststudioapi.
The mm_dev profile resolves to account 253455159142 in us-west-2. The selected cluster is marketmaker-testing-dev.
The existing testing-server service remains healthy at one desired and running task.
The core template now supports an existing cluster, an explicit service name, and narrowly scoped database ingress.
A separate development foundation supplies dedicated networking, private encrypted PostgreSQL, ECR, Cognito, startup secrets, and an origin certificate.
Reviewed add-only change sets create 29 foundation resources and 31 core resources. The core stack creates no cluster.
Both stacks report CREATE_COMPLETE. ECS reports a completed one-task rollout and two healthy target groups.
Both CloudFront distributions report Deployed. The public address is https://teststudioapi.devfoodmarketmaker.com.

The image adds a reviewed public RDS CA bundle with pinned provenance. Database TLS keeps certificate and hostname verification enabled.
The builder reuses unchanged UI, trace viewer, and all three Go binaries.
The bootstrap helper preserves an existing confirmed administrator. Generated credentials stay in Secrets Manager and protected temporary request files.
Temporary credential files and the private Docker authentication directory are removed after use.

The focused cloud workflow passes actual Cognito OIDC sign-in, ordinary catalog publication, one native HTTPS execution, and private S3 evidence access.
The first setup request omits the publication idempotency key. The corrected request reuses its draft; no failed or duplicate test execution occurs.
Run 20261106T014449.692767879-e0841a5b4b501c410e0db21d passes in 167.768248 ms, with a 16.519076 ms TLS handshake.
It retains one attempt, four events, four log segments, and four artifacts. The downloaded evidence matches its size and checksum.
The public index and main JavaScript match the built bytes. Anonymous project access returns 401.
The evidence bucket reports AES256 encryption and all four public-access blocks enabled. The owned application session signs out.
The repeatable verifier preserves its original run reference and idempotency key. No browser or coded worker starts for this check.

Both templates pass focused schema checks. The recorder packaging contract remains valid across its four profiles and three rejected settings.
Actual AWS coded workers, preparation, lifecycle, two-provider permissions, storage policies, and recorder handoff remain separate work.
This advances 41, 46, 54, 56, and 57. No acceptance item closes; the total remains 50 verified of 57.
See [MMDev installation](aws-mmdev.md) and output/aws-mmdev for exact references, operation receipts, and continuation instructions.

## Readiness and priority audit before AWS authorization — 2026-11-05

The [active plan](plan.md#user-facing-priority-categories) records Critical, Important, and Nice to Have priorities.
No known local create, run, or result-inspection blocker remains in retained verification.
The seven Important open criteria follow order 51, 46, 54, 41, 56, 57, and 53.
Actual AWS verification remains separate work. Azure implementation and verification remain deferred.
The [deployment audit](deployment-readiness.md) distinguishes completed local implementation from the remaining provider gates.
The README now leads with local startup, tester guides, and the exact deployment limits.
It links implemented identity linking and pending project access. Detailed staged Azure references move into the Azure operator guide.
Database setup uses the supported environment reference rather than a credential-bearing command argument.
Validation guidance selects changed workflows and preserves passing evidence.
Documentation consistency and local link checks cover this update. No test, browser, build, deployment, or cloud operation runs.
No acceptance item closes. The total remains 50 verified of 57.

## Hosted recorder pairing API — 2026-11-05

The API can reserve an owner-private companion session without starting a server-side browser.
Five-minute pairing tokens rotate before claim. One desktop instance binds with a separate device credential.
Only hashes enter server storage. Pairing cannot grant ordinary application authority.
Frozen owner, project, suite, input, and deadline binding survive restart.
Current permissions and archive state apply to control and ordered source submissions.
Stop and cancel intent remain durable. Terminal snapshots accept exact retries and reject later replacement.
Maintenance closes expired, revoked, disconnected, or disabled sessions while retaining captured source.

Targeted local domain and HTTP checks pass. Existing native recording and maintenance checks also pass.
HTTP checks cover browser-header rejection, CSRF, session isolation, ordinary API denial, private draft conversion, and absence of test execution.
The OpenAPI contract and generated TypeScript client now describe the companion protocol.
The desktop executable, wizard connection flow, and complete browser verification remain unfinished.
The option stays disabled by default. No cloud, cluster, image build, browser process, or demo restart occurs.
This advances hosted authoring under 57. The count remains 50 verified of 57.

### Desktop connection and recovery command

The new teststudio-recorder command reuses the trusted native driver and fixed recording helper.
It accepts a connection code at its prompt and keeps that credential out of command arguments and ordinary output.
HTTPS is required except for numeric loopback verification. Redirects do not forward device credentials.
Private journals preserve device identity, launch intent, pending sequence, and final source before their dependent actions.
An operating-system lock excludes concurrent companions in the same directory.
Resume recovers retained source without replaying an uncertain browser launch.
Transport failure, revoked authority, deadline, and interrupted commands stop the owned recorder and retain local source.
Exact retries recover lost pairing, snapshot, and final acknowledgements.

Targeted local client and command checks pass, including a real native process driver with an authored helper fixture.
The helper exits and the directory lock releases after owner stop. No Chromium session, cloud call, cluster operation, or demo restart occurs.
The guided wizard, first-time setup, contextual help, and complete browser workflow remain unfinished.
See [desktop companion](recorder-companion.md). This advances 57; the verified count remains 50 of 57.

### Guided connection and real native capture

The authoring page now uses the shared stepper and focusable headings for Configure, Connect desktop, Record, and Review.
Native desktop mode omits the connection step. Generated identifiers stay under Advanced recording details.
Connection codes rotate before claim, disappear after pairing, and expire visibly. Reload retains status without retaining the private code.
Unsaved configuration uses the existing navigation guard. Reset configuration permits a destination change.
Stopping remains visible until the desktop closes and returns final source. Review opens the existing editor without starting a run.
Current-page Introduction, Concepts, Tutorials, and the user guides now describe these controls.

The UI build and focused progress and expiry utility checks pass.
TestHostedRecorderCompanionCompletesGuidedUIAndNativeCapture runs a new Playwright UI suite through normal TestStudio preparation, publication, execution, timing, and evidence.
The companion opens real native Chromium and records the independent local target's initial navigation.
The check covers rotation, reload, help, focus, dark mobile layout, light desktop layout, stop, review, and durable draft saving.
The recorded target receives no test run. The authoring run is 20261105T213644.732520000-8266daf9143ea86cc6552e14.
The first attempt identifies an incorrect semantic selector in the new help assertion. The corrected focused case passes in 26.84 seconds.
All sixteen owned processes exit. Owned images, execution containers, volumes, and private contexts are removed.
Evidence is output/playwright/recorder-companion-help-fix. The initial failure remains in output/playwright/recorder-companion-first.
The user demo, its browser, and its data remain unchanged. No cloud or cluster operation runs.
Standalone desktop distribution remains unfinished. This advances 57; the count remains 50 verified of 57.

### Standalone desktop package

The operator package builder now creates a fresh macOS or Linux archive for arm64 or amd64.
It copies only the binary, trusted helper, locked dependency declarations, setup script, README, and checksum manifest.
Testers install Node.js and run node setup-recorder.mjs. They do not need Go or a checkout.
The wizard displays the relative packaged command and updates its First-time desktop setup instructions.
The command now exits successfully after --help.

The native macOS ARM64 package passes archive checksum, exact file inventory, trusted helper bytes, executable mode, and native help checks.
Offline setup installs all three locked npm packages from cache and reuses the existing Chromium installation.
The first archive exposes macOS metadata entries. COPYFILE_DISABLE removes them from the corrected archive.
The corrected archive and verification record are in output/recorder/package-pOD1pj. Temporary verification files are removed.
Focused command generation and UI build checks pass. The unchanged recording lifecycle retains its passing browser evidence.
No new browser session, cloud operation, cluster operation, or user demo restart occurs for packaging.
Other desktop targets have no new native verification. Opt-in provider configuration and deployed-profile verification remain separate.
This advances 57; the total remains 50 verified of 57.

### Opt-in recorder deployment settings

Kubernetes chart 0.1.1 adds recordingCompanion: false. The AWS core template adds RecordingCompanion with default false.
Enabling either setting adds --recording-companion=true only to the API process.
The setting adds no browser, workload, port, volume, credential, or permission. Existing worker and source settings remain independent.
Kubernetes notes and operator guides describe trusted desktop distribution and the required updated service image.

Three focused Kubernetes profiles validate 41 resources. Two malformed values are rejected.
The enabled base manifest differs only by its API argument. Separate controller and preparation arguments exclude recording configuration.
Four ECS previews cover enabled and disabled settings with and without coded workers. Three malformed values are rejected.
The task previews differ only by the API argument. CloudFront retains HTTPS, authorization forwarding, POST delivery, disabled caching, and no UI rewrite.
The focused service command check starts two private API subprocesses. Default mode stays disabled; explicit enablement reserves pairing without a browser dependency.
Both subprocesses stop and release their listeners. The check takes 0.09 seconds, excluding compilation.

Evidence is output/kubernetes-recording-package and output/aws-recording-package.
No cloud, cluster, image build, or browser operation occurs in this packaging batch. The user demo remains running.
The next verification uses the dedicated local Kubernetes installation and one changed service image.
This advances 57; the verified count remains 50 of 57.

### Actual Kubernetes recorder handoff

Helm revision 15 now runs the updated service image with recordingCompanion enabled only on the API.
Both controllers and preparation report Ready and receive no recording flag. The selected gateway and worker runtimes remain unchanged.
Normal OIDC authoring creates a private recording through the installed API.
The production companion client and runner use the owned HTTPS ingress with the existing development certificate and verified hostname.
Only test-local address routing and certificate trust differ from ordinary desktop transport. No host DNS or system trust changes occur.
Real native Chromium records the independent local target's initial navigation.
The owner requests stop. The companion closes its browser, acknowledges final source, and the API retains ready.
Reviewed source includes the permanent tag, visible environment binding, and needs-review safety label.
Saving creates PostgreSQL draft 20261105T220101.897259977-94680fe9e468d63c7a720d59 at revision 1.

The native check passes first attempt in 6.46 seconds. The recording is desktop-8bdd35ed-8b4b-410c-a7c7-dd2c4365924e.
All 23 owned verification processes exit. The local target closes, the managed session signs out, and private connection files are removed.
The twelve original offline runs and seven application-networking runs keep their original identifiers. The new project has no target test run.
Checkpoint reinspection passes in about 0.2 seconds without another browser, target, or recording.
Evidence is output/local-k3s/recording. No AWS or Azure operation occurs. The user demo remains running.
See [local Kubernetes recording](local-kubernetes-recording.md). This advances 57; the verified count remains 50 of 57.

### First-frame theme contrast

Retained authoring screenshots identify a real transient contrast defect.
The foreground changes immediately during a theme switch, while the shared button background interpolates for 140 milliseconds.
An isolated Linux Chromium observation confirms light text over the previous light background on the first dark frame.
The shared button style now changes text and background together. Border transitions remain available.

A new focused Playwright case runs through normal TestStudio source preparation, publication, execution, timing, and evidence capture.
It verifies primary, secondary, and destructive buttons at the first dark and light frame with contrast at least 4.5.
It also verifies keyboard focus and retains color observations and screenshots.
Run 20261105T221401.036801000-bad2b0258e7fdd85749b16c0 passes. The complete source and execution check takes 10.90 seconds.
The first harness attempt uses the environment-worker slot for an offline fixture. The corrected harness selects the existing offline Playwright worker.
The UI build passes. The separate read-only macOS browser closes, its reported process exits, and the isolated rendering container is removed.
The TestStudio check removes its prepared image, execution containers, volumes, and private source contexts.

Local Helm revision 16 serves the exact updated compiled stylesheet through verified HTTPS.
The new image reuses the unchanged API, health, and preparation binaries. All expected replicas report Ready.
Checkpoint reinspection retains the existing recording, revision-1 draft, and nineteen old runs without another browser or target run.
Evidence is output/playwright/button-contrast. No cloud operation occurs.
This strengthens verified item 49. The count remains 50 verified of 57.

## Remote gateway transport milestone — 2026-11-05

The trusted gateway now supports execution-scoped TLS forwarding through a supervisor-owned Unix socket bridge.
It reuses the canonical browser and API policy engine. Application credentials remain inside the trusted gateway.
Cancellation and deadlines close hijacked browser tunnels as well as ordinary requests.
Targeted local checks cover HTTPS delivery, credential injection and redaction, denied routes, authorization, certificate validation, and tunnel shutdown.
The selected gateway checks pass with the race detector. Existing private command configuration checks also pass.
No cloud, cluster, image rebuild, browser session, or catalog test execution occurs for this transport milestone.
Hosted admission remains disabled until provider initialization, workload identity, network policy, and durable cleanup are wired.
This advances implementation under 51, 56, and 57. All seven remaining broad criteria stay open; the verified count remains 50 of 57.
See [remote transport](remote-gateway.md) for the component boundary and exact continuation instructions. Azure remains deferred.

### Private initialization and supervisor delivery

The exchange now separates gateway initialization, runner execution, and proxy transport authority.
It freezes the application credential reference outside runner input and binds one provider-confirmed gateway instance.
Activation waits for bound identity and private configuration delivery. Current execution authority still applies.
New local HTTPS checks verify role separation, preactivation denial, frozen references, restart, revocation, and authority withdrawal.
The supervisor creates its bridge before the child starts and removes the socket after delivery.
A new child-process check verifies actual socket use, stripped gateway environment, and prelaunch rejection of another execution.
These changed exchange and supervisor checks pass with the race detector.
Targeted command configuration and pinned-registry inspection checks pass. Runtime build-script syntax passes.
New runtime overlays advertise the gateway capability. No image rebuild, cluster deployment, cloud test, or browser process occurs.
The shared gateway process can fetch private configuration. Its complete provider installation path remains unverified.
Continue Kubernetes provider delivery and installation integration.
The total remains 50 of 57. This advances 51, 56, and 57 without closing their broader requirements.

### Kubernetes bundle lifecycle

The provider now owns the runner, trusted gateway, headless Service, and two scoped execution network policies.
The shared dispatch journal records resource hashes, creation attempts, UIDs, and independent Pod deadlines.
Gateway identity binding precedes private configuration. Readiness precedes admission.
Inventory includes all bundle resource kinds and partial creation.
Shutdown attempts both workloads. Removal requires saved results, complete or abandoned evidence, and guarded resource absence.
Lost acknowledgements do not replay creation. A changed resource retains ownership uncertainty and pending cleanup.
Missing dispatch authority remains pending instead of being reconstructed from annotations.

New local protocol checks cover full lifecycle, lost acknowledgement restart, partial creation cleanup, foreign gateway rejection, and policy loss.
A stalled runner observation still permits independent gateway shutdown. Unconfirmed runner termination keeps the bundle nonterminal.
Those changed provider checks pass with the race detector. Selected existing offline provider checks also pass.
Two focused shared lifecycle scenarios reject infrastructure failure before activation and prevent a reported pass from overriding it.
No image rebuild, cluster deployment, cloud test, browser session, or catalog execution occurs for this provider component.
Normal environment executor registration, source discovery admission, image capability inspection, service-role secrets, and Helm installation wiring remain unfinished.
See [Kubernetes gateway lifecycle](kubernetes-gateway.md) for current boundaries and exact continuation work.
The verified total remains 50 of 57. This advances implementation under 51, 56, and 57. Azure remains deferred.

### Normal environment admission and installation configuration

The normal hosted executor now enables environment work only through a configured gateway provider.
Registry checks verify the gateway platform, fixed entrypoint, and private delivery capabilities.
Source runtime checks require the gateway transport marker, supervisor build revision, and API-policy capability when needed.
Source assembly and isolated discovery support the existing reviewed environment profiles.
Discovery freezes its inspected capabilities and rejects missing support. Discovery receives no target or credential.
Recovery restores the frozen target policy and credential reference from the saved run.
Timing keys include the supervisor revision and gateway identity, keeping old and new runtime observations separate.

The API and trusted worker backend can resolve scoped private gateway credentials.
The controller receives gateway initialization authority without mounting application credential material.
Opt-in Helm values supply the separate gateway account, TLS and trust references, and bounded namespace resource permissions.
The additional write role excludes preparation. Application credentials mount only in the API deployment.

Targeted local checks pass for managed environment execution, capability rejection, source publication, frozen discovery, profile registration, and recovery input.
The managed lifecycle passes the race detector.
The enabled chart renders 24 schema-valid resources. Four unsafe configurations are rejected.
Rendered settings pass the real Go parser and match the protected base policy.
No image rebuild, cluster deployment, cloud test, browser session, or catalog run occurs during this integration batch.
The next step is the first actual browser and coded API flows on the dedicated local cluster.
The count remains 50 of 57. This advances 41, 51, 56, and 57 without closing their full deployment scope.

## Actual Kubernetes application networking — 2026-11-05

The dedicated local installation now passes newly authored environment-backed Go API and Playwright browser suites.
Normal OIDC sign-in, source upload, isolated discovery, reviewed publication, execution, and private evidence use the installed product APIs.
Helm revision 14 preserves PostgreSQL, shared evidence, both identity providers, and two execution controllers.

The API run is 20261105T190135.888651628-f45615b1da24f0d003db1922. Its measured logical duration is 20 ms.
It verifies authenticated HTTPS, reflected-credential redaction, rejected methods and paths, and denial of direct worker access to the target.
Seven private artifacts pass size, checksum, and credential-absence checks.

The browser run is 20261105T190237.146588449-543e82e7b812fb2d345278ba. Its measured logical duration is 263 ms.
It opens the independent local HTTP application, clicks its availability button, and checks the returned item count.
Eight private artifacts pass verification. Screenshots and a trace survive, including archive-content checks for the fixture credential.
Visual inspection confirms the screenshot shows ready: 3 items.

Both execution records reach cleaned. The runner Pods, gateway Pods, headless Services, and execution policies are absent.
CRI reports no running worker or gateway container. Temporary publisher archives and credential directories are absent.
The twelve existing offline run identifiers remain unchanged. No host browser, AWS test, or Azure test runs.

Actual integration fixes remove the stale targetless-suite discovery restriction and permit Docker's exact standard gateway PATH.
Typed Kubernetes list responses now restore omitted item types from their requested endpoints while rejecting conflicting explicit types.
Stopping a gateway preserves its worker's recorded failure outcome. The upgraded controller cleans the retained failed execution without another dispatch.
The local proxy policy now admits the trusted gateway role. A smaller idle preparation CPU reservation lets one worker and gateway fit together.

Targeted source, registry, bundle lifecycle, and failed-worker cleanup regression checks pass.
Only the changed service image rebuilds after concrete fixes. Runtime images and existing offline checks remain reused.
See [local gateway verification](local-kubernetes-gateway.md), output/local-k3s/gateway, and [deployment readiness](deployment-readiness.md).
This advances **41, 51, 56, and 57**. The total remains **50 verified, 7 open**.
Azure stays deferred.

### Actual gateway cancellation and interrupted-run recovery

Both targeted local gateway lifecycle cases now pass once each without rebuilding an image.
One new suite first reaches the existing authenticated target, then writes a marker in its private output volume.
The verifier checks that marker and both workload identities before cancellation or the owning controller's SIGKILL.

Cancellation run 20261105T190905.111792480-6395b63f6393d2b50ee25bda keeps an unknown interrupted attempt and unavailable duration.
It reaches cancelled and cleaned. Five available private diagnostic artifacts pass checksum and credential-absence checks.

Recovery run 20261105T191000.981246578-8b2e6eef59d96658294be7ea retains its original execution and passing observation measured at 12010 ms.
Its run remains incomplete with unknown outcome. Its execution evidence is complete, and cleanup reaches cleaned.
Five private diagnostic artifacts pass verification. The original controller Pod restarts, and both execution controllers return Ready.

Each case keeps one execution identifier. Both worker Pods, gateway Pods, Services, and execution policies are absent after cleanup.
CRI reports no running worker or gateway container. The twelve offline runs remain unchanged.
No host browser, cloud call, broad suite, calibration run, or image rebuild occurs.
See output/local-k3s/gateway/lifecycle and [local gateway lifecycle](local-kubernetes-gateway.md#verify-gateway-lifecycle-boundaries).
This advances **51, 56, and 57**. The total remains **50 verified, 7 open**.
Continue with ECS gateway implementation and targeted local checks. Azure remains deferred.

### ECS gateway transport components

The shared gateway now supports a provider-bound private task address without replacing its TLS certificate identity.
New local TLS checks verify the selected address, certificate hostname, certificate authority, and execution capability together.
Invalid address forms fail before connection. The address remains outside the uploaded child packet.
Task preparation can require an address that becomes known only after native launch.
Initialization and activation stay closed until BindGatewayAddress binds the verified address and immutable instance.
Preparation retries retain that address and original ticket. Replacement addresses fail.

The trusted gateway accepts a private certificate/key pair through bounded environment values, with mounted-file fallback when both variables are absent.
Valid material passes. Incomplete, invalid, empty, and oversized values produce sanitized errors.
Source runtimes and gateway images declare separate optional address and certificate capabilities.
Registry inspection exposes them, and the runtime build script validates its address marker.
An omitted certificate capability preserves the existing gateway timing fingerprint.

Selected local transport, exchange, certificate, and registry checks pass. The runtime script passes syntax validation.
No image build, new catalog run, browser, Kubernetes redeployment, or live cloud request occurs for these shared components.
Native ECS environment execution remains disabled until its full provider lifecycle and network integration are implemented.
See [ECS gateway integration](ecs-gateway.md). This advances implementation under **51, 56, and 57**; the total remains **50 verified, 7 open**.

### Native ECS gateway task and metadata contracts

NewGatewayPlan constructs the separate fixed Fargate gateway task from a frozen environment execution and its initialization ticket.
It reuses the existing immutable definition, launch, task ownership, deadline, and recovery-proof paths.
The gateway has distinct role, subnets, security group, tags, family, deadline name, and certificate secret references.
It runs without a task role, source or output mounts, elevated capabilities, interactive access, or restart permission.
Health uses the remote readiness command. Unknown health keeps admission pending, and unhealthy status marks an infrastructure failure.
Recovery proofs contain hashes and secret references, never raw tickets or private certificate values. Restored proofs cannot launch tasks.

The new ECS bootstrap mode reads only the canonical version-four task metadata endpoint.
It derives the task instance without forwarding initialization tickets, certificate values, or other private headers.
Alternate endpoints, redirects, encoded paths, explicit instance overrides, invalid ARNs, oversized responses, and trailing documents fail.
The gateway image declares its separate native task identity capability, and registry inspection exposes it.

Targeted local task, ownership, readiness, deadline, recovery, metadata, and registry checks pass.
Selected existing worker-plan, deadline, and recovery-proof checks also pass after the shared verifier changes.
No broad suite, image build, browser, Kubernetes redeployment, or cloud call occurs.
This is a task-contract milestone. Durable two-workload orchestration, actual network admission, and deployment packaging remain required.
The existing provider still rejects environment-backed ECS dispatch before creation.
Continue with role-specific provider journals and the existing cleanup state machine. Keep worker creation behind gateway readiness and activation.
This advances **51, 56, and 57**; the verified total remains **50 of 57**.

### Durable ECS gateway child lifecycle

The separate gateway provider now reuses the existing SDK registration, launch, deadline, observation, shutdown, and cleanup state machine.
Gateway rows use ecs-gateway-dispatches. Worker rows retain their original bucket and execution authority.
Frozen public trust and approved secret references remain part of the ownership proof. Raw tickets and certificate keys remain absent.
Native creation requires inspected transport, certificate, and task identity capabilities, plus the configured gateway network guard.
Verified task ownership, deadline, native network, and private IPv4 binding precede private initialization.
Unknown health permits initialization binding while retaining pending admission. Withdrawn authority and replacement addresses cannot acquire gateway authority.
Removal still requires revoked dispatch, confirmed termination, saved results, and complete or abandoned evidence.

Targeted local checks compose the real private exchange with authored AWS protocol fixtures.
Initialization order, capability rejection, address mismatch, deadline drift, authority withdrawal, restart, and checkpointed cleanup pass.
Lost registration, launch, and deadline acknowledgements recover without a second native creation attempt.
Selected existing offline worker lifecycle checks also pass after the shared state-machine extraction.
No cloud call, image build, Kubernetes redeployment, browser, or catalog execution occurs.
Parent orchestration, the concrete native gateway network guard, and installation wiring remain unfinished.
Environment-backed ECS executor registration remains disabled. This advances **51, 56, and 57**; the total remains **50 of 57**.

### Composed ECS environment lifecycle

NewEnvironment now composes the durable gateway and uploaded worker controls under one original execution reference.
Ensure creates only the gateway. Observe waits for verified address binding, private initialization, and healthy status before worker creation.
Worker dispatch recovers the original ticket under current authority and matches the frozen environment proof.
Source admission requires verified-address transport and an immutable supervisor revision.
Concurrent observations share one durable registration claim. Existing offline provider ownership remains separate.

Both roles retain independent task definitions and stop schedules. Inventory merges their journals into one execution, including partial bundles.
Shutdown attempts both roles even when worker observation fails. Removal retains the original result and evidence checkpoint.
Gateway infrastructure failure survives restart and prevents a worker pass from confirming the bundle.
Previously bound gateways do not request another private initialization after a worker submits results.
Lost launches remain unresolved until original receipts appear. They cannot trigger another worker or gateway launch.

Targeted local protocol checks cover startup order, current authority, concurrent dispatch, successful completion, partial cancellation, lost launches, independent shutdown, and cleanup.
Changed lifecycle checks and selected existing offline worker and recovery checks pass with the race detector.
No cloud test, image build, browser, Kubernetes redeployment, or catalog run occurs.
The native gateway network guard, production SDK selection, source preparation, and installation wiring remain unfinished.
Production environment-backed ECS registration remains disabled. This advances **41, 51, 56, and 57**; the total remains **50 of 57**.

### Native ECS gateway network admission

The native network guard now admits a separate gateway VPC through an explicit frozen network profile.
It verifies same-account and same-Region peering, narrow subnet routes, reciprocal transport groups, public NAT, and private exchange endpoint delivery.
Environment workers gain only gateway TCP 8044 alongside their existing private platform paths.
The worker DNS firewall remains unchanged. Offline mode still rejects gateway routes, direct target egress, and NAT.
The trusted gateway resolves application targets through its own VPC and retains the existing origin, request, credential, and destination guards.

Worker and gateway task checks share the native ECS attachment and EC2 interface verifier.
Public addresses, IPv6, extra groups, changed native identity, and mismatched address ownership fail.
Frozen settings copy and canonicalize gateway subnet and NAT identifiers before binding the immutable network profile.
Changing or removing these controls cannot reuse that profile.
The production SDK factory rejects incomplete gateway network configuration before any AWS request.
The worker package includes the additional peer, NAT, and internet gateway read permissions, without creating or enabling new network resources.

Targeted local native SDK protocol checks pass for the accepted network, task binding, configuration changes, route and group drift, native errors, and incomplete production selection.
Changed network checks and selected existing offline network checks pass with the race detector.
A fresh local package preview passes the existing package-to-admission check and confirms the additional Region-bound read permissions.
Retain that preview in output/ecs-gateway-network/admission-fixture.json. It uses the current template and does not replace older recorded evidence.
No cloud request, image build, browser, Kubernetes redeployment, or catalog execution occurs.
Next add SDK factory selection, source preparation, and dedicated environment deployment packaging.
Keep the existing offline pool separate. See [native gateway networking](ecs-gateway-network.md).
This advances **51, 56, and 57**; the total remains **50 verified of 57**.

### Native ECS SDK registration and source preparation

Settings now accept a complete optional environment pool beside the existing offline worker and network objects.
Both pools share the configured account, Region, cluster, architecture, registry namespace, exchange service, and controller backend.
Environment workers and gateways retain distinct VPCs, subnets, security groups, endpoints, and immutable network profiles.
Partial, shared, foreign, or mismatched configuration fails before SDK initialization.

The native factory retains the offline provider and constructs the composed environment provider with current private exchange authority.
API and controller commands register separate Go API, browser, and recovery drivers without replacing offline execution or discovery.
The preparation command supplies no application execution authority. Its runtime contains only offline discovery and registry access.
Environment and offline providers share the scoped registry client and existing native SDK boundaries.

Registry source assembly now requires the ECS verified-address capability before publishing environment source content.
Discovery freezes that capability into its private input. Unsupported environment output cannot reach catalog review.
New capable environment sources and legacy offline sources still reach review without a gateway, application target, credential, or test run.
Kubernetes retains its existing transport capability contract.

Targeted local factory, configuration, registration, source publication, and source review checks pass with the race detector.
Selected existing factory and registration checks also pass. No image build, browser, Kubernetes redeployment, live cloud call, or catalog run occurs.
Dedicated ECS environment network, gateway roles and certificate delivery, repository, and combined service settings remain to be packaged.
This advances **41, 51, 56, and 57**; the total remains **50 verified of 57**.

### Optional ECS environment deployment package

The worker template now includes 59 optional resources for separate environment worker and trusted gateway networks.
The existing offline worker resources retain their properties. EnableEnvironment and EnvironmentReady default to false.
Provisioning creates the repository, private subnets, narrow peering routes, scoped groups, two NAT paths, registry endpoints, exchange endpoints, and worker DNS controls.
Activation requires the sealed ready stage and an immutable gateway image digest.
This separation supports repository creation and endpoint acceptance before active configuration, including sealed installation upgrades.

The gateway execution role reads only its image repository and the two declared certificate secrets.
Optional key decryption requires the selected Region's Secrets Manager service and certificate secret encryption contexts.
Gateway creation and pull-role passing attach only to the controller. Preparation retains its source publication and offline discovery scope.
The retained gateway repository uses immutable tags and image scanning.
The generated settings preserve the offline objects and add the environment object only after activation.
Normal API and controller configuration consume that same secret. Existing tasks must roll to load a changed secret version.

Local schema, six configuration profiles, six rejected configurations, permission boundaries, default-resource preservation, and private range checks pass.
Resource-derived metadata passes the production settings loader, native network guard, and gateway task address verifier.
No cloud deployment, image build, browser, Kubernetes redeployment, or catalog execution occurs.
Retain the preview under output/ecs-environment-package. See [operator instructions](aws-environment-package.md).
Continue the hosted recorder companion next. Actual AWS verification remains separate, and Azure remains deferred.
This advances **51, 56, and 57**; the total remains **50 verified of 57**.

## Active application navigation correction — 2026-11-04

The sidebar now has one project selector and an explicit application selector.
Projects landing uses management cards without a duplicate project selector.
A persistent context band identifies the project and active application.
Application testing contains Overview, Suites, Test catalog, Test runs, and Trends.
Project overviews contains All applications and a dedicated HTTPS probes route.
Application routes retain scope through reload, browser Back, test creation, history, and investigation links.
Switches retain the main destination and clear the previous application's filters and run details.
Canceled unsaved-work switches restore the selector and retain the edited suite.
Shared suite badges explain project ownership. Catalog notices explain complete-suite publication, transfer, and saved selections.
Server catalog filtering scopes counts, facets, and categories, including tests that inherit suite links.
New runs validate application and environment agreement and freeze application associations.
Run lists, retained history, trends, options, and rankings use frozen scope. Shared run details retain the complete snapshot.
Unknown legacy targetless application scope remains available under All applications.

Validation passes six focused UI utility checks, Angular production build, generated API contract synchronization, and targeted Go scope and compatibility checks.
STUDIO-NAV-001 passes through normal TestStudio publication and execution with two new applications.
It verifies filtered tests, shared suites, filtered run lists, scoped trends, reload, Back, creation context, unsaved-work cancellation, keyboard navigation, and narrow light and dark layouts.
Run: 20261104T143026.653455000-ee7f64897d689cd1c0a78fe7. Evidence is complete and the execution ledger is cleaned.
The updated local demo retains its existing SQLite data and evidence at http://127.0.0.1:8040.
Visual inspection confirms the active application and grouped navigation. The owned browser and automation daemon exit.
Page help, application, catalog, result, retained-history, trend, UI, and plan documentation match the new scope rules.
This improves verified items 05, 21, 27, 29, 30, 47, 49, and 50. The count remains 48 verified, 9 open.
No cloud verification runs. Continue the remaining hosted adapter work after this usability correction.

## Global account and project navigation correction — 2026-11-04

Projects now appears before the project and application selectors.
The sidebar no longer uses Workspace as an ownership heading or places Your account beneath it.
The global header provides account access on desktop and narrow screens.
The account page and help explain that accounts apply across TestStudio. Project access is assigned separately.
Application scoping and project or installation administration keep their existing behavior.

Five focused navigation utility checks and the Angular production build pass.
STUDIO-NAV-001 passes with project-link ordering, no Workspace heading, no sidebar account link, and mobile global account navigation.
Run: 20261104T143706.209433000-1adab9175059225f76a47692. Evidence is complete and the execution ledger is cleaned.
Light and dark narrow-layout checks pass. The owned test container and browser processes are removed.
The verified build contains the navigation correction. The user demo requires a restart to replace its private UI snapshot.
Help, application guidance, UI rules, and the plan match the navigation correction.
This refines verified items 49 and 50. The count remains 48 verified, 9 open. No cloud verification runs.

### Served demo correction

The user's follow-up exposes an older UI snapshot still served by the running demo.
The earlier report of an updated user demo was incorrect. Compiling the source did not replace its private snapshot.
The demo now restarts through its owned launcher and serves the corrected navigation at the same address.
Its existing database, definitions, five retained runs, and evidence remain available. Setup does not seed another history.
A focused local Playwright check verifies Projects above the selectors and no Workspace heading or sidebar account link.
It also verifies the Parcel desk application context and global account navigation at a 390-pixel width without document overflow.
The demo guide explains the required stop, start, and browser reload sequence.
Acceptance items 49 and 50 retain their verified status. The total remains 48 of 57.

## Acceptance criteria

These stable numbers match the priority list supplied to the user.
Report progress against these numbers. Keep broad criteria unchecked until their full scope is verified.

- [x] **01.** The Go service and UI build from this repository without MarketMaker imports, packages, shared resources, or sibling checkouts.
- [x] **02.** No existing tests, fixtures, catalog records, configuration, credentials, reports, or history are copied into the product.
- [x] **03.** Installation starts with empty registries and can load only explicitly selected new examples.
- [x] **04.** Local durable operation works without a cloud account.
- [x] **05.** A new project, application, environment, and suite can be registered without editing Go or TypeScript source.
- [x] **06.** PostgreSQL and SQLite pass the same storage behavior checks, including claims, revision conflicts, and restart recovery.
- [x] **07.** SQLite restarts preserve users, memberships, assignments, custom grants, disabled roles, and sessions without a PostgreSQL service.
- [x] **08.** Local SQLite installation requires no database server. Export and import to PostgreSQL preserve identifiers and artifact references.
- [x] **09.** Local accounts, OIDC accounts, and mixed sign-in work through the same directory, permission checks, and managed sessions.
- [x] **10.** Users, groups, membership, custom roles, assignments, and effective permissions are manageable through authorized API and UI workflows.
- [x] **11.** `wtiger001/go-permissions` evaluates access on PostgreSQL and SQLite through the same service and identity contracts.
- [x] **12.** User, group, role, object, and project grant checks cover explicit denies, expired grants, inheritance, and assignment revocation.
- [x] **13.** Project administrators cannot grant installation privileges or affect another project's groups, assignments, results, or evidence.
- [x] **14.** Membership and grant revocation reach all instances within the documented limit. Disabled users and revoked sessions fail their next request.
- [x] **15.** Local credential changes, resets, disabling, and recovery preserve audit records and the last usable administrator.
- [x] **16.** OIDC sign-in supports configured providers with code flow, PKCE, state, nonce, issuer and audience validation, key rotation, and logout.
- [x] **17.** Duplicate email addresses from different issuer-subject pairs do not merge accounts or transfer access.
- [x] **18.** Unmapped, missing, incomplete, or outdated external groups do not silently preserve elevated project access.
- [x] **19.** Browser sessions keep provider tokens server-side. Expiry, revocation, callback rejection, and request-forgery protection pass focused checks.
- [x] **20.** Provider credentials remain outside browser configuration and application logs.
- [x] **21.** Catalog creation, editing, validation, publication, import, export, bulk actions, archive, and restoration work through both API and UI.
- [x] **22.** Declarative uploads, complete source bundles, and registered GitHub repositories can produce versioned releases through the UI.
- [x] **23.** One repository can provide multiple suites, including suites beside existing application source.
- [x] **24.** Test-only publication requires no API or UI deployment.
- [x] **25.** Concurrent catalog edits produce explicit conflicts. Source synchronization preserves service-managed metadata.
- [x] **26.** Failed or duplicate preparation events preserve the active release and do not create unintended test runs.
- [x] **27.** Runs retain their release across updates, activation changes, and rollbacks.
- [x] **28.** Catalog changes do not change existing run snapshots. Disabled tests and invalid dependencies block new dispatch explicitly.
- [x] **29.** Removed tests and archived suites retain their history and required artifacts.
- [x] **30.** TestStudio executes its own new Playwright suite through normal publication, execution, timing, result, and evidence contracts.
- [x] **31.** Go unit suites run without artificial web targets or production credentials.
- [x] **32.** Go suites preserve test identities, disable cached execution, and distinguish build failures from test assertion failures.
- [x] **33.** Adding an HTTPS check requires no browser or service deployment.
- [x] **34.** HTTPS checks run manually and on schedules, with configurable assertions, TLS checks, response-time history, and normal result records.
- [x] **35.** API suites run through guarded direct clients and produce the same history and trends as browser tests.
- [x] **36.** Package events, subtest aggregates, retries, and API steps do not inflate test counts.
- [x] **37.** Local and scheduled executions produce the same logical run and result records.
- [x] **38.** Protected environments reject unsafe selections and unsafe dependencies.
- [x] **39.** Timeouts have a separate test status and count. Failure reruns include them alongside failed and flaky results.
- [x] **40.** Native HTTP attempts support comparable timing history, bounded adaptive timeouts, slowdown criteria, and API and UI estimates.
- [ ] **41.** Go and browser runners apply the same timing contract across execution and deployment profiles.
- [x] **42.** Run details expose attempts, steps, context, logs, screenshots, traces, and configured attachments without requiring HTML reports.
- [x] **43.** Available evidence survives service restarts. Duplicate events and upload retries do not create duplicate attachments or rerun tests.
- [x] **44.** Capture gaps, failed uploads, expired files, and incomplete evidence remain visible beside the observed test outcome.
- [x] **45.** Artifact access respects project permissions. Private trace viewing requires no public upload or third-party viewer.
- [ ] **46.** Retention, quotas, and evidence pins work across local, S3, and Azure artifact adapters.
- [x] **47.** Result-state trends use stacked area charts with consistent state colors, final logical counts, accessible values, and visible missing-history gaps.
- [x] **48.** Manual and scheduled notifications use the same configured completion path.
- [x] **49.** Light, dark, and system modes pass browser workflow and keyboard checks.
- [x] **50.** Every current page provides a contextual help drawer with Introduction, Concepts, Tutorials, and verified keyboard and mobile behavior.
- [ ] **51.** Provider adapters support execution status, cancellation, deadlines, artifacts, and recovery consistently.
- [x] **52.** Replica failover creates no duplicate logical run. Notification retries retain their documented at-least-once behavior.
- [ ] **53.** Local, Kubernetes, and Azure installations operate without AWS services or credentials.
- [ ] **54.** Two newly configured OIDC providers work with the same internal directory and project-scoped permissions across deployment profiles.
- [x] **55.** Hosted identity and private report access work through installation-specific configuration.
- [ ] **56.** Newly written generic suites complete runs and produce progress, history, reports, and trends on each deployment target.
- [ ] **57.** AWS ECS and CloudFront, Kubernetes, Azure Container Apps, and local-server profiles pass the common acceptance scenarios.

## Current delivery: dedicated local k3s cluster

The separate teststudio-k3s Colima profile runs with four CPUs, 6 GiB memory, and a 20 GiB disk.
Verified k3d v5.9.0 creates one ARM64 k3s v1.35.5+k3s1 server node in about 17 seconds.
The node reports Ready=True. CoreDNS, local storage, metrics, and Traefik report Running.
The cluster API binds only to loopback. Its private kubeconfig has mode 0600 and uses the observed Docker API port mapping.
Helm connects from the host using that kubeconfig and its Secret driver. The default Docker context remains unchanged.
The existing demo remains on port 8040. No Azure work resumes.
Evidence is output/local-k3s/cluster-summary.json, nodes.json, system-pods.json, helm-connectivity.json, and cluster-create.log.
Local ownership and tool checksum metadata remain in .data/local-k3s. The cluster stays available for application verification.
The actual Helm installation now passes with a fresh PostgreSQL database and separate persistent evidence storage.
Release teststudio is deployed in namespace teststudio-local. Both Pods are Ready, and both persistent claims are Bound.
The service image uses the imported ARM64 manifest digest. No registry push occurs.
The UI, API health, and artifact health return 200 through TLS ingress with certificate verification.
Anonymous project access returns 401. Evidence is output/local-k3s/installation-checks.json and helm-install.log.
The first service start precedes database readiness. Kubernetes restarts it once, and the next start succeeds.
The local OIDC issuer now runs through a new independent fixture in the dedicated cluster.
The service supports mounted private OIDC authorities without changing native HTTP or system client trust.
The Helm chart separately mounts OIDC authorities and existing project-scoped native HTTPS bindings.
Focused checks verify valid private trust, rejected hostname and expiry, rejected untrusted authorities, unchanged default trust, and invalid-file errors.
The existing HTTP trust checks pass after extracting their certificate loader into the shared certificate package.
Normal OIDC code flow and PKCE now authenticate the installation administrator through actual TLS ingress.
Project, application, environment, suite, test, release, and run creation use the normal authorized API.
Run 20261105T135704.336870446-2120fdc092213d2c32b6024c passes with one newly authored HTTPS probe and complete evidence.
The probe duration is 2.293329 ms, with a 1.418414 ms TLS handshake and retained certificate expiry.
Four events, one attempt, four logs, and four artifacts remain available. A private artifact download passes its SHA-256 check.
After service Pod replacement, fresh OIDC sign-in reads the same single run and downloads its retained evidence without reexecution.
This verifies core persistence across that restart. It does not establish replica failover or coded-worker recovery.
Evidence is output/local-k3s/probe/summary.json and the adjacent retained run, event, attempt, artifact, and log records.
No browser, AWS, or Azure test runs. This advances the Kubernetes portions of 51, 55, and 56 without closing their full scope.
The next step is Kubernetes trusted gateway implementation for environment-backed browser and coded API suites.
The local cluster now runs the verified gVisor 20261005.0 package through RuntimeClass teststudio-isolated.
An actual non-root diagnostic Pod reports the gVisor kernel and exits zero.
The chart can use external private TLS routing while preserving the worker Service, controller permissions, and exact policy peers.
The dedicated local proxy exposes only the worker exchange host and path prefix.
A worker-namespace diagnostic reaches that exchange and has no Kubernetes API token.
Its connections to the API, artifact service, PostgreSQL, and local probe target are denied.
The positive node-side baseline confirms that those four destination services accept connections.
Four readiness retries allow policy convergence without relaxing the network policy.
The diagnostic Pod, public-CA ConfigMap, and temporary node helper are removed after the check.
Three focused routing profiles and four invalid configurations pass local rendering and schema validation.
The local TLS registry and separate preparation workload now run with different reader and writer credentials.
The reader cannot upload. Worker credentials contain no registry, database, or Kubernetes API credential.
Fresh Go and Playwright ZIPs use normal OIDC, source assembly, isolated discovery, reviewed publication, execution, and private evidence APIs.
Run creation now admits the configured Kubernetes driver while rejecting location mismatches.
The source-v2 Go runner preserves event-timestamp precision when framework elapsed time rounds to zero.
The source-v2 Playwright runner supplies writable output and the frozen capture policy across projects without source configuration workarounds.
Its trusted wrapper preserves suite-relative setup, teardown, server, snapshot, and TypeScript paths through a focused compatibility check.
Three targeted Go admission or existing behavior checks, three Go event checks, and four browser configuration checks pass.
The prior Go and incomplete browser observations remain immutable in history under their original runtime identities.
Go run 20261105T151433.206691439-698ad6db3525513bd828fd49 passes in 0.604332 ms with seven artifacts and seven log segments.
Playwright run 20261105T151447.417956930-cfc4c366f3c9ee354dd21551 passes in 263 ms with nine artifacts and six log segments.
Both evidence states are complete. Private downloads pass size and SHA-256 checks. Screenshots and a trace are retained.
Both execution ledgers are cleaned, both preparations have no pending cleanup, and Kubernetes and CRI show no remaining worker container.
Visual inspection confirms the browser's captured card shows Reviewed. No host browser or broad browser suite runs.
Evidence is output/local-k3s/coded/summary.json and adjacent run, preparation, log, artifact, execution, download, and cleanup records.
See [local source execution](local-kubernetes-source-execution.md) for installation details and continuation instructions.
This advances the Kubernetes portions of 41, 51, 55, and 56. The total remains 48 verified, 9 open.
See [local Kubernetes](local-kubernetes.md) for scoped commands and stop instructions.
No numbered acceptance item closes. The total remains **48 verified, 9 open**.

## Local Kubernetes lifecycle progress — 2026-11-05

Three newly authored lifecycle cases pass through normal source publication on the actual local cluster.
Cancellation records a cancelled run, an unknown interrupted attempt, unavailable duration, and an explicit evidence gap.
The timeout case records one timed-out result in 2008 ms with complete evidence and no failed-test count.
The controller receives SIGKILL only after its exact Pod UID, container, namespace, and live process are verified.
Kubernetes restarts the controller. The original run and execution identifiers remain unchanged.
Its recovered passing observation retains 20000 ms timing and complete evidence. The interrupted logical run remains incomplete.
All three execution records reach cleaned and their original worker Pods disappear.
The verifier stores checkpoints and reuses existing runs instead of repeating execution after an observation delay.
Actual asynchronous Pod deletion exposes an unnecessary recovery delay. The adapter now confirms absence within its existing call deadline.
Two focused checks cover delayed deletion and rejection of a foreign replacement. Both pass.
The new service deploys through Helm revision 8 with cached UI and trace assets.
An actual Rerun failures request includes the timed-out test and retains its parent link.
Its child run 20261105T154833.201607664-fc55cd1fb3946ebffd6803ef has one timeout, complete evidence, and cleanup on the first attempt.
No host browser, live AWS, or Azure verification runs. Independent deadlines and concurrent replica failover remain open.
See [local lifecycle](local-kubernetes-lifecycle.md) for exact identifiers, evidence, and continuation instructions.
This advances the Kubernetes portions of 41, 51, 52, and 56. The total remains **48 verified, 9 open**.

## Independent deadline and controller packaging — 2026-11-05

Actual run 20261105T155703.162689682-275ad94081254ddd2cb5b4a4 proves independent Kubernetes Pod deadline enforcement.
The guest supervisor and controller both remain stopped while Kubernetes records DeadlineExceeded and terminated exit code 137.
The original Pod UID and 15-second active deadline remain unchanged. Runtime control stays available.
Termination includes the configured ten-second grace and kubelet scheduling delay. Provider elapsed time does not become test duration.
The verifier restores the controller. The same run retains its timeout, explicit evidence gap, and cleaned execution.
Earlier fixture errors remain preserved. No failed observation is rewritten, and observation expiry never authorizes another execution.

The optional Helm controller package now separates execution replicas from the API and private worker backend.
It disables API execution and supplies dedicated controller identity, projected tokens, reader references, configuration, probes, and a disruption budget.
Filesystem profiles require actual shared ReadWriteMany evidence. The chart rejects a ReadWriteOnce configuration even for one separate controller.
Controllers receive no OIDC credential, writer reference, preparation journal, public Service, or Ingress.
Three focused package profiles and four rejected configurations pass. One targeted existing concurrent-claim check covers the added claim log.
The current cluster remains on combined Helm revision 8. Separate-controller deployment and concurrent failover remain unverified.
See [local lifecycle](local-kubernetes-lifecycle.md) and [controller packaging](kubernetes-controllers.md) for evidence and continuation steps.
This advances 41, 51, 52, and 56. The acceptance total remains **48 verified, 9 open**. Azure remains deferred.

## Acceptance 52 complete: actual controller peer failover — 2026-11-05

Helm revision 9 runs one API and two separate controllers on the dedicated local cluster.
The API execution worker is disabled. Shared NFS evidence uses the original artifact claim without copying bytes or recreating history.
An earlier browser screenshot remains downloadable with its stored size and checksum.
Run 20261105T161905.063920611-46e31390481d4b43ad92cdba records one winning controller claim.
The verifier stops that exact owner before SIGKILL and excludes replacement controllers while the original peer remains Ready.
The peer retains its Pod UID, container identity, process, and restart count through recovery and cleanup.
The original execution recovers a passing 20000 ms observation with complete private evidence. Its private download passes checksum verification.
The interrupted logical run remains incomplete with unknown outcome. There is one attempt and one execution, with no duplicate logical run.
Both controller replicas are restored and no test worker remains. This proves controller failure on one node, not node or storage fault tolerance.

The criterion's notification half passes one focused local check with an actual HTTP receiver and a second database connection.
It retries uncheckpointed acceptance with the same delivery and run identities, retains unknown acceptance, and executes the test exactly once.
This preserves the documented at-least-once behavior without repeating tests.
Evidence is output/local-k3s/controllers/summary.json and notification-recovery-check.log.
The complete saved lifecycle run set contains eight intended observations, including the two retained deadline-fixture errors.
See [local controller failover](local-kubernetes-controllers.md) for exact identities, storage, cleanup, rollback, and continuation instructions.
Acceptance **52 is complete**. The total is **49 verified, 8 open**: 41, 46, 51, 53, 54, 55, 56, and 57.
Other deployment profiles remain unverified. Azure implementation stays deferred.

## Acceptance 55 complete: hosted identity and private evidence — 2026-11-05

The actual local Kubernetes installation now authenticates two independently configured OIDC issuers against its shared PostgreSQL directory.
The second issuer uses a separate fixture process, signing key, realm, client, callback, certificate, and managed provider record.
Both fixtures emit the same verified email and subject. Their issuer links resolve to different users without transferring administrator authority.
A new provider-owned group maps the reviewer into only project k3s-coded through the normal Viewer assignment API.
The reviewer can inspect an existing run and download its private screenshot with matching size and SHA-256 checksum.
Foreign-project reads, installation user listing, and provider administration reject that reviewer.
Removing the group role immediately blocks the existing session's project, run, grant, and previously issued artifact URL access.
Restoring the role restores authorized reads. All verification sessions sign out. No test runs are created.
Installation-specific origins, callbacks, clients, managed providers, mounted private CA trust, and separate evidence routing remain authoritative.
The original issuer deployment and application image remain unchanged.
One focused fixture protocol check covers both default and realm configurations. No UI build or browser suite runs.
Evidence is output/local-k3s/identity/summary.json with issuer links, directory views, metadata references, and a private download.
See [local hosted identity](local-kubernetes-identity.md) for the complete acceptance audit and continuation instructions.
Acceptance **55 is complete**. The total is **50 verified, 7 open**: 41, 46, 51, 53, 54, 56, and 57.
Acceptance 54 keeps its cross-profile requirement. AWS and Azure deployment verification remain separate, and Azure implementation stays deferred.

## Remaining deployment audit — 2026-11-05

The seven remaining criteria are audited against current source, deployment packages, and retained local evidence.
Hosted admission explicitly rejects browser-target artifacts and required credential bindings.
The Kubernetes provider currently requires an empty gateway reference. Its worker policy allows only private exchange and DNS traffic.
Environment-backed browser and coded API execution therefore still require trusted gateway implementation.
The working local gateway uses a Docker lifecycle and Unix transport. It is not a remote Kubernetes or ECS gateway.
Native recording also requires the server's local desktop. Hosted authoring still needs a scoped local companion or streamed browser.
Azure native coded execution and discovery wiring remain unfinished and deferred.
Shared timing, evidence, and identity engines have implementation and local evidence. Their cross-profile verification gates remain open.

The active queue now prioritizes a complete Kubernetes gateway, its shared ECS contract, and the hosted recorder companion.
Reuse existing policy, credential, redaction, timing, source, and lifecycle contracts. Keep current admission barriers until the replacement boundary works.
No tests, cloud connections, application restarts, or new test runs occur for this documentation audit.
See [deployment readiness](deployment-readiness.md) for each criterion's implementation gap, verification gate, and continuation sequence.
The count remains **50 verified, 7 open**. No additional acceptance closes.

## Current direction: Azure deferred; local Kubernetes integration

The user defers Azure implementation and verification until a later deployment script and account setup.
The staged Azure changes are preserved. No Azure resource or account has been tested.
Kubernetes evidence includes local SDK protocol, Helm rendering, schema, and container startup checks.
A dedicated local cluster and actual TestStudio chart startup are now verified.
The user now authorizes dedicated local k3s/k3d verification with a separate kubeconfig.
Authenticated execution of one newly authored HTTPS probe now passes through that installation.
Isolated coded workers, cancellation, deadlines, artifacts, and restart recovery follow on that local cluster.
The default Docker workloads and demo on port 8040 remain preserved.
The active plan and repository instructions record this change of priority.
Acceptance **51** remains open. The total remains **48 verified, 9 open**.

## Current delivery: selected controller secret and evidence authority

The existing authority reader now includes startup-secret and execution-evidence grants with selected vault, Blob account, and private container metadata.
Two secret-only roles target the startup principal at individual database and configuration secrets.
Execution adds container-scoped Blob read, write, and delete authority for its separate runtime principal. Discovery receives no Blob grant.
The verifier rejects broader scopes, changed principals, secret writes, account-key access, grant delegation, public Blob access, and shared-key access.
AuthorityDefinitions now supplies 15 discovery objects or 17 execution objects. No resource or grant is created.
The existing authority cache retains its 15-second bound. Remaining domain authorization and effective-protection checks stay uncached.
Three new checks and eleven affected checks pass locally with race detection in 4.191 seconds.
For one shared vault, the reader observes 20 resources in discovery and 24 in execution. Separate vaults add one resource.
The central cold discovery composition observes 157 unique selected paths.
The same fixed profiles drive custom role definitions, scoped assignment objects, and metadata verification.
The reader copies execution account and runtime-identity pins before observing resources.
Checks retain secret-value isolation, mode separation, settings immutability, cache expiry, runtime admission fencing, and reader closure.
Affected checks retain native roles, scoped identities, cache sharing, uncached protection, explicit credentials, mode boundaries, and database admission.
Evidence is output/azure-sandbox-host-access/summary.json and local.log.
Checks use local SQLite, fake credentials, and in-memory metadata. Owned readers close through cleanup.
No cloud service, secret endpoint, Blob operation, database connection, browser, container, UI build, demo restart, resource mutation, or remote push runs.
The plan and affected operator guides match the behavior. No user page changes.
Effective authority, remaining import and timer identity attachment, protected writes, native command activation, and deployment packaging stay unfinished.
See [controller dependency authority](azure-sandbox-host-access.md).
Acceptance **51** advances. No acceptance item closes. The total remains **48 verified, 9 open**.

## Current delivery: controller private PostgreSQL network and reported service binding

The runtime now joins a selected PostgreSQL flexible server, controller private endpoint, NIC member, endpoint subnet rules, DNS, and reported approvals.
The source security group permits TCP 5432 only to that private address. The controller VNet now contains four exact selected subnets.
The server hostname must match the command's database-host binding. Public access must be disabled and password authentication enabled.
Complete embedded and separate approval inventories must agree on the selected endpoint and up to seven declared installation consumers.
Both documented completed connection states are accepted only with matching approval and ownership facts. Unknown and transitional states remain unavailable.
The resolver and reported subscription DNS inventory require the private database zone and link.
Fresh creation and import reads and bounded live evidence include the stage. Current failure invalidates earlier success.
Five staged resource bodies support later packaging without creating a server, endpoint, or credential.
Three new checks and eleven affected checks pass locally with race detection in 5.631 seconds.
The database reader observes ten selected paths. The source network reader now observes 17 paths.
The central cold discovery composition observes 152 unique selected paths.
The implementation pins PostgreSQL metadata to 2024-08-01 and reuses pure endpoint, subnet, rule, and private-zone verifiers.
Public Microsoft contract examples disagree on completed connection provisioning values; the reader accepts Ready and Succeeded with all other checks required.
Checks cover both modes, copied declared consumers, invalid settings, server, endpoint, port, source, DNS, approval and inventory drift, and reader closure.
Affected checks retain controller source, registry, host, resolver, scoped settings, and mode admission.
A focused settings recheck also passes in 1.594 seconds after tightening registry-consumer and zone alias rejection.
Evidence is output/azure-sandbox-host-database/summary.json, local.log, and settings-followup.log.
Checks use local SQLite, fake credentials, and in-memory metadata. Owned readers close through cleanup.
No cloud service, database connection, browser, container, UI build, demo restart, resource mutation, or remote push runs.
The plan and affected operator guides match the behavior. No user page changes.
Actual SQL and dependency access, effective authority, remaining identity attachment, native command activation, and deployment packaging stay unfinished.
See [controller private database](azure-sandbox-host-database.md).
Acceptance **51** advances. No acceptance item closes. The total remains **48 verified, 9 open**.

## Current delivery: PostgreSQL destination and TLS binding

The service and preparation commands now accept database-host to bind a protected PostgreSQL URL to one expected DNS hostname and verified TLS.
The guard rejects changed hosts, nonstandard ports, plaintext fallbacks, duplicate or alternate routing options, and nonempty PG environment overrides before storage opens.
Both staged controller commands require the configured host.databaseHost. Host admission rejects reported hostname argument drift.
The controller resolver approved list includes that name. The shared worker and converted-image profiles stay unchanged.
Four new checks and ten affected checks pass locally with race detection across configuration, service command, preparation, and sandbox packages.
The existing local controller still completes one queued HTTPS probe and releases its owned process and listener.
The connection URL remains in its protected reference. Validation errors exclude credentials and private parser diagnostics.
Validation reads no certificate, password, service, or provider configuration files.
The driver configuration retains the expected hostname, port 5432, verified TLS server name, and no plaintext or alternate-host fallbacks.
Runtime configuration passes in 1.275 seconds, service commands in 7.627 seconds, preparation in 1.990 seconds, and sandbox contracts in 2.230 seconds.
Checks use local SQLite, the existing owned loopback HTTPS target, pure configuration, and in-memory metadata.
Evidence is output/database-host-binding/summary.json and local.log.
Owned child processes, listeners, clients, and readers close through cleanup.
No cloud service, browser, container, UI build, demo restart, resource mutation, or remote push runs.
The plan and affected operator guides match the behavior. No user page changes.
Private database networking, actual dependency authority, remaining identity attachment, native command activation, and deployment packaging stay unfinished.
See [PostgreSQL destination binding](database-host-binding.md).
Acceptance **51** advances. No acceptance item closes. The total remains **48 verified, 9 open**.

## Current delivery: controller VNet, source perimeter, and resolver association

The runtime now checks controller VNet contents, source security rules, default routing, dedicated firewall policy, and actual resolver association.
The VNet contains exactly the selected source, registry endpoint, and firewall subnets with matching embedded observations.
Derived HTTPS dependencies include platform services, management endpoints, selected Key Vault hosts, and execution-only Blob storage.
Discovery adds no Blob dependency. Trusted controller network access stays outside the sandbox policy and shared worker profile.
Fresh creation and import reads and bounded live evidence include this stage. Current failure invalidates earlier success.
Fourteen staged resource bodies support later packaging. Database transport and native HTTPS-probe target routing remain unfinished.
Three new checks and eleven affected checks pass locally with race detection in 4.217 seconds.
The stage reuses the existing pure VNet, attachment, route, firewall, and resolver verifiers without ordinary Jobs settings.
Sixteen selected paths retain joined five-second bounds and scoped transport restrictions.
The central cold discovery composition observes 142 unique selected paths.
Checks cover both controller modes, dependency separation, invalid bindings, topology, route, firewall, resolver, current failure, and reader shutdown.
Affected checks retain explicit credentials, scoped modes, host and private registry admission, relay networking, and resolver composition.
Evidence is output/azure-sandbox-host-network/summary.json and local.log.
Checks use local SQLite, fake credentials, and in-memory metadata. Owned readers close through cleanup.
No cloud service, browser, container, UI build, demo restart, resource mutation, or remote push runs.
The plan and affected operator guides match the change. No user page changes.
Native command selection, dependency completion, effective authority, remaining identity attachment, protected writes, and deployment packaging stay unfinished.
See [controller source perimeter](azure-sandbox-host-network.md).
Acceptance **51** advances. No acceptance item closes. The total remains **48 verified, 9 open**.

## Current delivery: native controller private registry path

The runtime now joins the controller private registry endpoint, NIC members, endpoint subnet rules, private DNS, selected pull identity, and registry-scoped grant.
The selected endpoint must appear in shared placement.registryConsumers. Host-specific facts retain the separate host profile.
Source prefix and Container Apps delegation bind the endpoint rules to the selected controller subnet.
Selected registry approval and its embedded entry must agree. The complete registry service inventory remains independently required.
The reported subscription DNS inventory now guards controller, relay, and sandbox VNets against additional zone links.
Fresh creation and import checks and bounded live evidence include this stage. Current failure invalidates prior success.
Seven staged resource bodies support later packaging without creating resources. Native commands remain disabled for sandbox selection.
Three new checks and eleven affected checks pass locally with race detection in 4.277 seconds.
The reader observes 13 selected GET paths with joined five-second bounds and the existing transport restrictions.
The central cold discovery composition observes 128 unique selected paths.
The shared source-subnet verifier now serves relay and controller prefix and delegation checks.
The registry inventory fixture preserves the selected controller approval. Its pending-state check handles the expanded consumer set.
Evidence is output/azure-sandbox-host-registry/summary.json and local.log.
Checks use local SQLite, fake credentials, and in-memory metadata. All owned readers close during cleanup.
No cloud service, browser, container, UI build, demo restart, resource mutation, or remote push runs.
The plan and affected operator guides match the behavior. No user page changes.
Remaining controller networking, effective authority, remaining identity attachment, native command selection, and packaging stay unfinished.
See [controller registry path](azure-sandbox-host-registry.md).
Acceptance **51** advances. No acceptance item closes. The total remains **48 verified, 9 open**.

## Current delivery: native sandbox controller host and reported identity attachment

The native runtime now binds a required execution or discovery controller host to its fixed headless process and private environment.
Control, registry inspection, and metadata identities have Main lifecycle; pull and startup identities have None lifecycle.
Execution adds an evidence runtime identity and health probes. Discovery has no artifact adapter or HTTP probes.
A separate sandbox-controller-host/v1 profile binds complete mode-specific settings without fragmenting the shared converted-image pool.
The host stage verifies selected application, environment, authentication, complete revisions, and explicit route and endpoint absence.
The expanded subscription workload census checks reported controller occupancy, identity sharing, nine or ten identities, and empty federation inventories.
Two staged resource bodies support later packaging. Native --azure-sandbox-env command selection remains unavailable.
Four new checks and eleven affected checks pass locally with race detection in 4.223 seconds.
Fresh creation and import reads and bounded live evidence include the host stage. A current failure invalidates earlier success.
Runtime shutdown closes the host reader after owned work joins.
The workload reader now sizes joined results for the expanded identity set.
The central cold discovery composition observes 118 unique selected paths.
Checks retain scoped mode boundaries, settings, explicit credentials, selected authority, workload inventory, trusted services, and runtime shutdown.
Evidence is output/azure-sandbox-host/summary.json and local.log.
Checks use local SQLite, fake credentials, and in-memory metadata. No cloud service, browser, container, UI build, demo restart, or remote push runs.
The active plan and affected operator guides match the change. No user page changes.
Controller network and dependency admission, effective authority, remaining identity attachment, command activation, and deployment packaging remain unfinished.
See [controller host binding](azure-sandbox-host.md).
Acceptance **51** advances. No acceptance item closes. The total remains **48 verified, 9 open**.

## Current delivery: headless execution controller command

The server now accepts --serve-role=controller with an explicit numeric address.
It opens the shared database and artifact adapter, retains normal authorization, executes queued runs, and performs execution recovery.
Only health and liveness routes are exposed. Readiness checks the database; liveness remains independent.
It skips account and OIDC bootstrap, user and artifact APIs, worker exchange transport, discovery, source assembly, scheduling, notification delivery, and recording or session maintenance.
The existing completion hook still creates notification delivery records with the installation's UI origin.
The API can select --execution-worker=false while retaining its other configured background services.
Default combined local-server behavior stays unchanged.

The local controller supports native HTTP checks. Existing ECS and Kubernetes profiles register execution without discovery.
Hosted controller profiles require PostgreSQL. Native Azure sandbox selection and controller identity attachment remain unfinished.
Controller preflight rejects unrelated service flags, ambiguous providers, mismatched locations, and implicit listeners before opening storage.
Common worker backend and controller health use the same narrow handler.
The shared server lifecycle starts workers after listener binding and joins them before provider and storage cleanup.

Three new focused checks and eight affected checks pass with race detection.
The command package passes in 9.993 seconds and runtime configuration in 1.282 seconds.
The actual controller command consumes one queued HTTPS probe with the API execution worker disabled.
It verifies one logical result, unchanged sign-in state, retained notification planning, graceful process exit, and listener closure.
Shared checks retain worker backend health, direct TLS authority, startup, role parsing, provider separation, and server cleanup.
Evidence is output/headless-controller/summary.json and local.log.
Checks use SQLite, private filesystem evidence, an owned loopback TLS target, and project-scoped certificate trust.
Owned child processes, listeners, and clients close through cleanup. No cloud service, browser, container, UI build, demo restart, or remote push is involved.
The active plan and operator guides match the change. No user page changes.
Acceptance **51** advances. No acceptance item closes. The total remains **48 verified, 9 open**.

## Current delivery: separate hosted execution registration and discovery command

Common server registration now accepts separate execution and discovery providers.
Execution and discovery can be omitted independently. Recovery retains only the configured provider for each role.
Existing ECS and Kubernetes API calls explicitly retain their shared-provider behavior.
This removes a registration mismatch with the sandbox runtime's execution-only and discovery-only modes.

The preparation command now accepts --discovery-only with an existing ECS or Kubernetes profile and matching queue location.
The mode registers both inspectors and recovery drivers without image builders, recovery builders, runtime distributions, or a build directory.
It rejects mixed providers, local or Azure assembly settings, assembly-only mode, build flags, repository sync, and secret bindings before configuration reads or state creation.
The common discovery loop takes a private service copy with builder references removed.
It processes received discovery jobs, normal discovery recovery, and authorized automatic repository publication.
Unbuilt sources and assembly recovery remain available to the writer. Caller configuration stays unchanged.
The mode does not remove permissions from an existing cloud role. Registry read and discovery control remain deployment requirements.
Native Azure sandbox activation and controller host binding remain unfinished.

Five new checks and four affected checks pass with race detection.
The server command package passes in 1.561 seconds, preparation command in 1.509 seconds, and shared sources in 4.066 seconds.
Checks cover role separation, early rejection, handoff, legacy unbuilt input skipping, independent discovery ownership, recovery, and automatic publication without another build or sync.
Evidence is output/isolated-discovery-command/summary.json and local.log.
Source workflow checks use local SQLite, an owned loopback TLS exchange, and in-process provider doubles.
The worker loop and listener close through cleanup. No cloud service, browser, container, UI build, demo restart, or remote push is involved.
The active plan and operator guides match the change. No user page changes.
Acceptance **51** advances. No acceptance item closes. The total remains **48 verified, 9 open**.

## Current delivery: sandbox reported service workload identity binding

OpenRuntime now constructs SDKServiceWorkloads with the existing metadata identity.
Creation, source import, and live admission verify complete reported Container Apps and job inventories for the selected subscription.
The relay and backend must appear in their configured environments. Other apps or jobs in either environment reject.
The four service identities must appear only on their selected applications, with exact client, principal, identity set, and lifecycle bindings.
Sharing by identity identifier, client, principal, or system-assigned principal rejects.
Unknown reported identity metadata stays unavailable. Explicitly unrelated workloads remain permitted.
The four selected identity resources must match their region, tenant, client, and principal.
Their four federated-credential inventories must each report one complete empty page.

The stage derives existing service settings and adds no configuration field or deployment resource.
Shared pure placement, identity projection, lifecycle, and identity-host verifiers preserve the existing observer's behavior.
The selected service and workload stages share their expected identity maps.
Workload collections have 512-item and 16-page bounds. The joined observation retains the five-second bound and existing response guards.
Fresh creation and import observations and bounded shared live evidence preserve uncached protection and authorization.
Current failures invalidate prior success. Runtime shutdown closes the new reader.

Four new checks and six affected native checks pass with race detection in 2.848 seconds.
Three existing shared observer-placement checks pass with race detection in 1.830 seconds.
Evidence is output/azure-sandbox-service-workloads/summary.json, native.log, and shared.log.
Checks use local SQLite, fake credentials, and in-memory ARM responses.
No cloud service, browser, container, UI build, demo restart, resource mutation, or remote push is involved.
The active plan and affected operator guides match the change. User pages do not change.

Reported Container Apps and jobs do not establish exclusive identity use across other subscriptions, host types, or deny-hidden resources.
Remaining registry consumer network ownership, controller and remaining identity host attachment, effective grants, deny effects, and protected writes remain unfinished.
Native command wiring, packaging, and separate deployment acceptance remain required.
Acceptance **51** advances. No acceptance item closes. The total remains **48 verified, 9 open**.

## Current delivery: sandbox complete reported registry service inventory

OpenRuntime now constructs SDKRegistryService with the existing metadata identity.
Creation, source import, and live admission verify private access policies and complete reported approvals, private-link members, and replications.
The expected consumer set includes the sandbox and relay registry endpoints plus up to six explicitly configured trusted private endpoints.
The complete embedded and separately listed approvals must agree on each endpoint's connection identifier.
Missing or pending approval stays unavailable. Extra, duplicate, foreign, rejected, or disconnected consumers reject.
The selected region, data hostname, Premium SKU, registry group, required members, private zone, and home-region replica contract must match.
Registry policies require disabled public, administrator, anonymous, task-bypass, and export access, with fixed role and authentication modes.
The shared pure RegistryServiceBinding retains the existing guard's behavior without ordinary Jobs configuration.

The sandbox-placement/v9 profile binds additional consumers.
Invalid scopes, aliases, duplicate consumers, and stale profiles reject before credentials.
The reader copies the consumer list. Later caller edits cannot change its admitted expectations.
SDK metadata conversion preserves absence of the optional SKU tier while retaining an explicitly incompatible value for rejection.
Four selected GET paths use bounded complete pagination, joined reads, existing response guards, and redacted failures.
Fresh creation and import observations and bounded shared live evidence preserve uncached protection and authorization.
Runtime shutdown closes the new reader.

Six new checks and seven affected native checks pass with race detection in 3.117 seconds.
Four existing shared registry checks pass with race detection in 3.068 seconds.
Evidence is output/azure-sandbox-registry-service/summary.json, native.log, and shared.log.
Checks use local SQLite, fake credentials, and in-memory ARM responses.
No cloud service, browser, container, UI build, demo restart, resource mutation, or remote push is involved.
The active plan and affected operator guides match the change. User pages do not change.

Additional declared consumer endpoints still require actual network and workload ownership binding.
Effective grants, identity host attachment, deny effects, protected writes, command wiring, and deployment packaging remain unfinished.
Reported inventory does not prove private image pulls, publication, conversion, routing, or preview compatibility.
Acceptance **51** advances. No acceptance item closes. The total remains **48 verified, 9 open**.

## Current delivery: sandbox relay network and private registry binding

OpenRuntime now constructs SDKRelayNetwork with the existing metadata identity.
Creation, source import, and live admission verify the relay VNet and its four selected subnets, security rules, default route, firewall, and resolver.
The private registry endpoint and NIC retain exactly the login and regional data addresses.
Selected registry-side approval and its embedded connection must approve the same relay endpoint.
Private DNS records and the sole relay-network link bind registry resolution.
The shared subscription zone inventory now rejects additional reported links to either the sandbox or relay VNet.
The selected pull identity must match its tenant, client, principal, registry-scoped assignment, and fixed built-in AcrPull role.
Trusted relay platform access remains separate from sandbox connection rules and its deny-all firewall policy.

The sandbox-placement/v8 profile binds the required relayNetwork settings.
Aliases, overlaps, reserved addresses, foreign resources, and stale profiles reject before credentials.
RelayNetworkDefinitions supplies 21 reviewable resource bodies without mutation.
Fresh creation and import observations and bounded shared live evidence preserve uncached protection and authorization.
Current failures invalidate prior safe evidence. Runtime shutdown closes the new reader.
Shared pure platform, subnet, route, registry-approval, private-zone, and resolver helpers retain their existing callers.

Six new checks and twelve affected native checks pass with race detection in 3.572 seconds.
Two affected shared verifier checks pass with race detection in 3.048 seconds.
Evidence is output/azure-sandbox-relay-network/summary.json, native.log, and shared.log.
Checks use local SQLite, fake credentials, and in-memory ARM responses.
No cloud service, browser, container, UI build, demo restart, resource mutation, or remote push is involved.
The active plan and affected operator guides match the change. User pages do not change.

Complete registry consumer, replication, and member inventory remains required.
Effective grants, identity host attachment, workload placement, deny effects, protected writes, native command wiring, and packaging remain unfinished.
Reported configuration does not prove deployed routing, resolution, readiness, private conversion, or preview compatibility.
Acceptance **51** advances. No acceptance item closes. The total remains **48 verified, 9 open**.

## Current delivery: sandbox backend endpoint and readiness binding

OpenRuntime now constructs SDKBackendEndpoint with the existing metadata identity.
Creation, source import, and live admission join selected backend environment approval, endpoint, NIC, private address, subnet, security rules, and relay source subnet.
The environment's embedded and separately listed connections must agree on exactly the selected approved endpoint.
The private-link required member must match the NIC and any explicit endpoint IP configurations.
The address must equal the relay's configured backend address, used for forwarding and its existing TLS readiness probe.
The dedicated subnet and security rules permit only relay-subnet HTTPS to that address and reject unrelated endpoint references or broader access.

The sandbox-placement/v7 profile binds backend endpoint settings.
Invalid scopes, overlaps, reserved addresses, reused resources, and stale profiles reject before credentials.
BackendEndpointDefinitions supplies two reviewable subnet and security-group bodies without mutation.
Fresh creation and import observations and bounded shared live evidence preserve uncached protection and current authorization.

Five new focused checks and fourteen affected native checks pass with race detection in 3.120 seconds.
Two selected relay TLS checks pass in 1.393 seconds. The actual worker backend database-health check passes in 1.480 seconds.
The native protocol check uses an owned loopback TLS transport and the real relay readiness handler with the configured backend hostname.
It verifies fresh connections, credential-free requests, outage and recovery responses, and independent liveness.
The existing relay checks retain production private-address pinning, certificate trust, and lifetime coverage.
The worker health check uses local SQLite and confirms database loss changes readiness without changing liveness.
Evidence is output/azure-sandbox-backend-endpoint/summary.json, native.log, relay-readiness.log, and backend-health.log.
Owned TLS listeners close through test cleanup. No cloud service, browser, container, UI build, demo restart, resource mutation, or remote push is involved.
The plan and affected operator guides match the change. No user page behavior changes.

Readiness runs inside the relay network. Native admission verifies its configured probe and reported healthy active revision.
It does not perform a fresh controller-side HTTPS readiness request or add a controller route to the worker-facing private endpoint.
Relay VNet and private registry observations, workload placement, identity host attachment, effective grants, deny effects, and write protection remain unfinished.
Native command wiring and packaging remain unfinished. Actual routing, platform readiness, and preview compatibility remain separate deployment verification.
No native command or deployment template enables this runtime.
Acceptance **51** advances. The verified total remains **48 of 57**, with **9 open**.

## Previous delivery: sandbox relay and trusted backend service admission

OpenRuntime now constructs SDKServices with the existing metadata identity.
Creation, source import, and live admission require selected relay and backend service metadata.
Private environments, infrastructure subnets and groups, disabled platform authentication, and complete revision inventories must match.
The relay environment's approved worker-facing private connection and embedded connection inventory must agree.
Private-link metadata and the selected certificate bind its expected regional service and exchange hostname.
Certificate metadata must be valid beyond the supported execution window and contain no returned secret material.

Independent builders retain fixed relay-only and worker-backend templates without requiring Jobs configuration.
Immutable service images stay outside the uploaded-source namespace.
Relay and backend commands, arguments, origins, resources, probes, database secret reference, and Blob configuration must match exactly.
Four service identity resource, client, and principal pins remain distinct from one another and existing native control identities.
Pull and startup identities retain lifecycle None. Only the backend runtime identity receives Main exposure.
Actual application identities, lifecycle settings, and templates must match their intended bindings.
Exactly one healthy active revision serves each app. Inactive revisions require zero replicas and zero traffic across complete bounded pagination.

The sandbox-placement/v6 profile binds service configuration. Invalid scopes, aliases, images, dependencies, identities, and stale profiles reject before credentials.
ServiceDefinitions supplies four reviewable app and authentication resource bodies without mutation.
Fresh initial and import checks and bounded shared live evidence preserve uncached protection and current authorization.
Six new focused checks and thirteen affected native checks pass with race detection in 2.865 seconds.
Six affected shared relay and backend checks pass with race detection in 4.057 seconds.
Evidence is output/azure-sandbox-services/summary.json, native.log, and shared-services.log.
No cloud service, credential endpoint, browser, container, UI build, demo restart, resource mutation, or remote push is involved.
The plan and affected operator guides match the change. No user page behavior changes.

Backend endpoint routing, readiness, workload placement, identity host attachment, effective grants, deny effects, and write protection remain unfinished.
Native command wiring and packaging remain unfinished. Actual TLS, dependency access, and preview compatibility remain separate deployment verification.
No native command or deployment template enables this runtime.
Acceptance **51** advances. The verified total remains **48 of 57**, with **9 open**.

## Previous delivery: sandbox resolver policy and reported zone inventory

OpenRuntime now constructs SDKResolver with the existing metadata identity.
Creation, source import, and live admission require selected resolver policy, complete rules and links, and the actual VNet association.
The fixed approved list contains five exchange and registry domains. A root-domain block list covers remaining names.
Exactly two enabled rules retain their Allow and Block actions, list bindings, and priorities.
The policy link and actual VNet association must both identify the selected policy and network.

The metadata transport permits one exact empty POST read action for listDnsResolverPolicies.
Other actions, mutations, payloads, positive body lengths, filters, and foreign scopes reject.
Default metadata readers retain GET-only scopes. The metadata role gains no write action.
The reused subscription inventory reader now has an independent subscription-only constructor.
Complete reported zone and link observations reject extra links to the sandbox VNet while preserving unrelated networks.
Missing pages, count gaps, inaccessible metadata, and inventory limits retain uncertainty.
Existing DNS policy and zone-link verifiers are reused through independent bindings without Jobs configuration.

The sandbox-placement/v5 profile binds resolver settings. Invalid bindings and stale profiles reject before credentials.
ResolverDefinitions supplies six reviewable resource bodies without mutation.
Fresh creation and import checks and bounded shared live evidence preserve uncached protection and current authorization.
Six new focused checks and twelve affected native checks pass with race detection in 2.171 seconds.
Seven affected shared DNS and subscription inventory checks pass with race detection in 4.645 seconds.
Evidence is output/azure-sandbox-resolver/summary.json, native.log, and shared-dns.log.
No cloud service, credential endpoint, browser, container, UI build, demo restart, resource mutation, or remote push is involved.
The plan and affected operator guides match the change. No user page behavior changes.

Azure domain hierarchy also permits descendants of allowed domains. Independent selected-zone restrictions remain required.
Reported subscription inventory does not prove visibility across subscriptions or the absence of deny-hidden resources.
Trusted exchange service ownership, additional authority, host attachment, deny effects, and write protection remain unfinished.
Native command wiring and packaging remain unfinished. Actual resolution and preview compatibility remain separate deployment verification.
No native command or deployment template enables this runtime.
Acceptance **51** advances. The verified total remains **48 of 57**, with **9 open**.

## Previous delivery: sandbox VNet and firewall policy admission

OpenRuntime now constructs SDKTopology with the existing metadata identity.
Creation, image import, and live admission require selected VNet, subnet, firewall, public IP, policy, and complete rule-group observations.
The VNet reports one private address space and exactly the execution, endpoint, and firewall subnets.
Peerings, custom DNS servers, extra subnets, default public NAT, and mismatched embedded subnet contents reject.
The selected private firewall address binds the checked default route to the reported firewall, subnet, policy, and public IP attachment.

The firewall policy permits one fixed deny-all network rule group and only empty known service default groups.
The complete inventory and reported policy references must agree, including four bounded pages when needed.
Inherited policies, nonempty defaults, additional rules, changed actions, shared firewalls, and changed attachments reject.
The existing worker and relay policy verifier remains in use through its extracted definition-aware implementation.
Native sandbox verification uses independent bindings without Jobs configuration or platform allow rules.

The sandbox-placement/v4 profile binds private address space and selected firewall resources.
Invalid scope, overlap, subnet size, firewall address, and stale profiles reject before credentials.
TopologyDefinitions supplies five reviewable resource bodies without mutation.
Fresh creation and import checks and bounded shared live evidence preserve uncached protection and current authorization.

Six new focused checks and eleven affected native checks pass with race detection in 2.208 seconds.
Three affected shared firewall checks pass with race detection in 3.120 seconds.
Evidence is output/azure-sandbox-topology/summary.json, native.log, and shared-firewall.log.
No cloud service, credential endpoint, browser, container, UI build, demo restart, resource mutation, or remote push is involved.
The plan and affected operator guides match the change. No user page behavior changes.

Resolver policy, other linked zones, trusted exchange service ownership, additional authority, host attachment, and deny effects remain unfinished.
Native command wiring and packaging remain unfinished. Actual routing and traffic enforcement remain separate deployment verification.
No native command or deployment template enables this runtime.
Acceptance **51** advances. The verified total remains **48 of 57**, with **9 open**.

## Previous delivery: sandbox private endpoint and DNS bindings

OpenRuntime now constructs SDKPrivate with the existing metadata identity.
Creation, source import, and live admission require selected endpoint, NIC, endpoint subnet, security group, and private DNS observations.
The allowed address set must exactly match registry login, regional registry data, and private exchange NIC addresses.
The endpoints report the expected registry and relay environment services, approved connections, and selected interfaces.
The endpoint subnet contains only the selected endpoint and NIC IP configuration references.
Its complete security rules permit inbound HTTPS only from the execution subnet to the selected addresses.

Complete bounded inventories verify the two selected zones' records and links.
Expected A records, SOA metadata, short TTL bounds, exact VNet links, and reported counts must agree.
Aliases, wildcards, duplicates, automatic registration, public fallback, and unrelated records reject.
Pagination retains the selected host, path, version, and one bounded skip token.
Unsafe continuations reject before credentials. Missing, oversized, duplicate, looping, or unfinished inventories return no partial evidence.
The sandbox-placement/v3 profile binds private service, endpoint, subnet, address, and DNS settings.
PrivateDNSDefinitions supplies seven reviewable resource bodies without mutation.
Existing endpoint, NIC, subnet, and DNS verifiers are reused through independent bindings without Jobs configuration.

Fresh creation and import observations and bounded shared live evidence preserve uncached protection and current worker authorization.
Five new focused checks and twenty-three affected checks pass with race detection in 2.559 seconds.
One initial runtime admission check passes before that regression run.
Evidence is output/azure-sandbox-private/summary.json and targeted.log.
No cloud service, credential endpoint, browser, container, UI build, demo restart, resource mutation, or remote push is involved.
The plan and relevant operator guides match the change. No user page behavior changes.

Effective DNS resolution, other linked zones, topology, firewall enforcement, trusted exchange service ownership, additional authority, host attachment, and deny effects remain unfinished.
Native command wiring and packaging remain unfinished. Actual preview compatibility remains separate deployment verification.
No native command or deployment template enables this runtime.
Acceptance **51** advances. The verified total remains **48 of 57**, with **9 open**.

## Previous delivery: sandbox connection rules and routes

OpenRuntime now constructs SDKNetwork with the selected metadata identity.
Creation and source import obtain fresh selected security-group and route-table observations.
Live admission shares successful evidence for at most 15 seconds. Current failed full observations invalidate previous success.
The complete custom rule list denies inbound traffic and metadata access.
It allows only platform DNS and HTTPS to explicit private endpoint addresses before denying remaining outbound traffic.
No ordinary Jobs platform allow rules enter this sandbox connection policy.
The route table disables BGP propagation and retains only the selected default route through the private firewall address.
Extra rules, broader ports, changed destinations, additional routes, shared attachments, and unknown writable properties reject.

The sandbox-placement/v2 ownership profile binds the firewall and endpoint addresses.
Invalid addresses, duplicate destinations, and stale profiles reject before credential construction.
The network reader retains an independent destination-list copy.
NetworkDefinitions returns reviewable resource bodies for later packaging without resource mutation.
The runtime cannot omit this stage. It still requires the remaining uncached protection callbacks.

Five new focused checks and eighteen affected runtime, authority, placement, shared execution, and command authorization checks pass with race detection.
The targeted run completes in 2.211 seconds. Evidence is output/azure-sandbox-network/summary.json and targeted.log.
No cloud service, credential endpoint, browser, container, UI build, demo restart, resource mutation, or remote push is involved.
The plan and relevant Azure operator guides match the change. No user page behavior changes.

Effective routing, topology, firewall ownership and policy, private endpoint ownership, DNS, additional authority, host attachment, and deny effects remain unfinished.
Actual preview-service compatibility, role acceptance, timer execution, private conversion routing, token audience, and proxy trust remain separate deployment work.
Native command wiring and packaging remain unfinished. No native command or deployment template enables this runtime.
Acceptance **51** advances. The verified total remains **48 of 57**, with **9 open**.

## Previous delivery: selected sandbox and timer authority

OpenRuntime now requires identity and selected-grant settings for controller, registry inspection, metadata, and independent timer authority.
Identity resources, client identifiers, and principal identifiers remain distinct from one another and the import identity.
The concrete authority stage reads four identities, five custom roles, and six selected assignments through the shared bounded ARM reader.
Incorrect tenant, client, principal, scope, role, conditions, or permission sets reject admission.

The execution controller receives fixed sandbox data actions without image import. Discovery adds import only.
The timer identity receives read and stop only. It receives no command, create, import, resume, policy, secret, or file authority.
Workflow administration is limited to read, write, and delete in the separate timer group.
Controller identity attachment targets only the timer identity. Registry inspection retains a separate AcrPull assignment.
Canonical role and assignment objects support reviewable packaging inputs without any resource mutation.

Placement and authority now share the scoped ARM reader and bounded positive-cache implementation.
Selected authority evidence has a 15-second bound and coalesces successful concurrent observations.
Directory and worker permissions, command reservation, revocation, per-sandbox definition checks, and the remaining protection callback stay uncached.
The authority stage cannot be omitted from the runtime factory or substitute for the remaining protection checks.

Five new focused checks cover permissions, mode separation, role broadening, identity and scope changes, invalid settings, caching, and factory integration.
The five new checks and thirteen affected placement, runtime, shared execution, and authorization checks pass with race detection in 2.037 seconds.
Evidence is output/azure-sandbox-authority/summary.json and targeted.log.
No cloud account, identity endpoint, registry service, role mutation, browser, container, UI build, demo restart, or remote push is involved.
The authority, placement, runtime, provider, Azure, and plan guides match the change. No user page behavior changes.

Effective additional authority, deny effects, identity host attachment, protected network content, private exchange and DNS, command wiring, and packaging remain unfinished.
Actual custom-role acceptance, timer execution, private conversion routing, token audience, and proxy trust remain deployment verification work.
Acceptance **51** advances. The verified total remains **48 of 57**, with **9 open**.

## Previous delivery: reported sandbox placement and selected import authority

OpenRuntime now constructs a concrete ARM placement reader with a separate explicit metadata identity.
The stage runs before the remaining mandatory protection dependencies for creation, import, and live instance admission.
Initial observations read seven selected resources in parallel: group, connection, subnet, pull identity, registry, assignment, and AcrPull definition.
Native settings bind these resource identifiers and identity metadata into the existing network, timing, launch, and source-cache profiles.

Reported drift rejects changed scope, subnet attachments, delegation, group identity, import tenant or client, public registry access, and altered pull authority.
Unknown group or connection properties reject the pinned preview contract rather than adopting another ARM version shape.
Metadata failures remain unavailable. A missing metadata resource cannot establish worker absence or authorize a replacement.
The transport permits selected GET requests only, disables retries and registration, bounds responses, closes bodies, and redacts diagnostics.
Cancellation checks precede SDK credential acquisition. A targeted fixture catches and verifies that early cancellation barrier.

Live placement uses successful observations for at most 15 seconds from observation start.
Expired successful reads coalesce across twelve callers and refresh only group, connection, and subnet.
Failures invalidate live evidence. Older observations cannot overwrite a newer full check's cache result.
The additional protection callback and all existing sandbox-definition and worker authorization checks remain uncached.
This avoids seven ARM reads on every quarter-second status poll while retaining fresh initial admission.

Six new checks and eight affected runtime, shared execution, and authorization checks pass with race detection in 1.913 seconds.
Evidence is output/azure-sandbox-placement/summary.json and targeted.log.
No cloud account, identity endpoint, registry service, browser, container, UI build, demo restart, or remote push is involved.
The placement, runtime, source, provider, Azure, and plan guides match the change. No user page behavior changes.

Complete effective authority, protected network content, private exchange and DNS, timer authority, command wiring, and packaging remain unfinished.
The plan retains explicit deployment questions for private image conversion, actual routing, the token audience, and proxy trust.
Acceptance **51** advances. The verified total remains **48 of 57**, with **9 open**.

## Previous delivery: independent sandbox settings and concrete runtime assembly

The new settings loader uses the existing bounded file or environment configuration path.
It validates independent sandbox settings, explicit nonzero client and tenant identifiers, registry redirects, pull identity, and the separate timer scope.
Legacy ordinary Job environment fields reject. Local startup remains unchanged when no sandbox configuration is supplied.

OpenRuntime constructs the concrete read-only ACR client, sandbox image and command client, Logic Apps timer SDK, image manager, and provider.
Managed identity construction selects the exact configured control and registry client identifiers without a default credential chain.
Construction obtains no token and calls no service. Admission is mandatory, with import admission and checkpoint authorization also required for discovery.
The factory supplies no permissive admission default.

Execution mode cannot import images or control discovery references. Discovery mode cannot control ordinary execution references.
Wrong-mode status, cancellation, removal, and inventory preserve the rightful exchange.
Shutdown joins owned commands before closing created clients. Closed providers reject all new interface operations.
The shared ACR registry validator now provides one policy for client and runtime settings validation.

Five new focused checks verify configuration sources, rejected bindings before credential construction, actual SDK wiring, credential selection, modes, and shutdown.
The affected shared executor and pending discovery flows pass with explicit scoped provider modes.
The five new checks and two lifecycle regressions pass with race detection in 2.060 seconds.
One existing ACR scope and credential-forwarding regression passes after validator extraction in 1.359 seconds.
Evidence is output/azure-sandbox-runtime/summary.json and runtime.log.
No identity endpoint, hosted service, cloud account, registry service, browser, container, UI build, demo restart, or remote push is involved.

The runtime, provider, source, command, discovery, Azure, and plan guides match this library delivery. No user page behavior changes.
Actual admission, effective timer authority, command role wiring, and deployment packaging remain unfinished. Commands do not select this factory yet.
The plan records the published ARM version shape difference for the admission implementation and separate deployment verification.
Acceptance **51** advances. The verified total remains **48 of 57**, with **9 open**.

## Previous delivery: sandbox provider integration and shared execution lifecycle

The sandbox Provider now implements the common hosted interface with the existing OCI inspector, image manager, dispatch controller, and command bridge.
Durable launch intents retain pending conversions before a sandbox exists.
Proofs bind source inspection, the selected import identity, frozen work, the full reference, and the original deadline.
No usable ticket enters the launch journal. Normal execution cannot import a missing source.

Ensure reserves the intent. Observe advances pending preparation, resolves the owned sandbox, and delegates independent command completion.
Stop checks exact scope, records a barrier, revokes work, and resolves late controller reservations.
Remove requires the canonical result and evidence checkpoint. It marks the launch removed after owned instance and timer cleanup.
Recovery inventory includes pending conversions. Cleaned status survives worker-exchange deletion.

Seven new focused checks cover both runners, pending restart, cancellation, an in-flight create, concurrent lost-receipt recovery, input binding, and import identity changes.
The actual shared executor and exchange claim work, record start, accept a report, upload and collect a private log, and finish checkpointed cleanup.
The reports retain their original test duration. These checks exercise provider protocols, not hosted test processes.
The shared flow preserves the original prepared deadline and exposes an observation race during worker updates.
Bounded final assertion retries fix that race and retain immediate revocation checks.
A deterministic check distinguishes normal revision conflicts from revoked work. Concurrent stop reservations remain pending without claiming quiescence.
The seven new checks and three affected authorization regressions pass with race detection in 2.340 seconds.
Affected documentation links resolve. Evidence is output/azure-sandbox-provider/summary.json and targeted.log.

No hosted service, cloud account, registry operation, browser, container, UI build, demo restart, or remote push is involved.
No user page behavior changes. Provider, source, controller, discovery, Azure, and plan guides match this delivery.
Actual admission, effective timer authority, native SDK assembly, configuration, and packaging remain unfinished.
Acceptance **51** and supporting recovery under **52** advance. The verified total remains **48 of 57**, with **9 open**.

## Previous delivery: durable sandbox source conversion and lookup

The production data-plane client now requires a complete image plan for imports.
Its proof binds configuration, pull identity, immutable OCI metadata, project, runner, and discovery generation.
Readback verifies the source, fixed supervisor, empty command, ownership labels, and Dockerfile content.
Ready and Succeeded follow the pinned SDK's completed conversion states.

The new ImageManager shares the installation database and real worker exchange.
Trusted preparation requires gated discovery input, its original ticket, fresh authorization, and explicit import admission.
Reservation and the import marker assert current exchange and permission revisions.
Twelve competing controllers make one import call. Lookup processes cannot initiate imports.
Both runners retain their converted source across database reopen.

Lost acknowledgments reconcile one complete ownership match. Missing receipts, duplicate matches, and incomplete inventory cannot authorize another import.
Acknowledged image lookup uses the exact receipt and rechecks the provider's current definition.
Known failed conversions can receive a new generation only through a different authorized discovery request.
The transaction retains previous proofs and receipts. Earlier requests cannot become eligible again after another generation fails.
Current and archived source assets remain outside per-run deletion. Provider source retention remains open under 46.

Seven new focused checks and two affected client checks pass with race detection in 1.769 seconds.
Two selected launch-proof regressions pass in 1.422 seconds. Affected documentation links resolve.
They cover protocol drift, both runners, SQLite reopen, contention, lost receipts, current permissions, grant revisions, and failed-generation history.
The evidence is output/azure-sandbox-images/summary.json and targeted.log.
No cloud request, registry operation, browser, container, demo restart, UI rebuild, or remote push occurs.
No page behavior changes. The new image guide and affected operator and integration guides match this library delivery.

Native Provider composition, actual import and runtime admission, configuration, and deployment packaging remain unfinished.
Acceptance **51** and supporting recovery under **52** advance. The verified total remains **48 of 57**, with **9 open**.

## Previous delivery: independent sandbox command completion and guarded supervision

Sandbox plans now start a trusted idle supervisor and invoke executable work only through the fixed command API.
The v3 proof binds the image capability, command body, UID class, deadline, source, and immutable execution settings.
The runtime overlay advertises azure-sandbox-command/v1. OCI inspection retains it, and older images fail sandbox admission.
The overlay builder verifies the capability during image creation and cache reuse.
Existing ordinary supervisor invocation remains compatible.

Idle mode reads no work credentials or executable input. Its absolute UTC deadline is bounded to two hours.
Idle and command modes enforce Linux UID and GID 1000, restricted supplementary groups, no new privileges, empty capabilities, and protected process state.
Root startup drops its identities before input retrieval. Other identities reject execution.
The existing x/sys version becomes a direct dependency without an upgrade or new checksum.

The command client sends only the fixed quoted supervisor command and working directory to the selected instance endpoint.
Credential acquisition stays bounded while command response lifetime follows the original absolute deadline.
Redirect following and request replay remain disabled. Missing or malformed exit codes remain unavailable.
Only an explicit service-returned exit code enters control state. Raw stdout, stderr, and tickets do not enter that receipt.

Command reservation atomically asserts activated exchange work, the frozen input, dispatch ownership, and fresh authorization records.
Only one competing controller starts the command. Restart never replays an uncertain or acknowledged mutation.
Owned background contexts persist completion or uncertainty, request instance stop, and join before shared resources close.
Controller.Status maps an explicit zero exit and confirmed quiescence to successful hosted completion.
Known nonzero exits become failed after quiescence. Idle, unknown, and canceled completion cannot fabricate a passing result or test duration.
Cancellation suppresses late success and checks project ownership before interrupting an active connection.

Eleven new focused checks cover command protocol and lifecycle, image capability, OCI inspection, and supervisor modes.
The final eight command checks and two affected lifecycle regressions pass with race detection in 1.968 seconds.
The supervisor checks, including the existing error-redaction case, pass in 1.286 seconds. The new OCI check passes in 1.350 seconds.
Six selected earlier proof, transport, timer, and lifecycle checks also pass. Node syntax and generated workflow schema and references pass.
A final review finds a cancellation-scope bug and a concurrent stop acknowledgment assumption. Targeted checks verify the corrections and retained retry behavior.
Unrelated product suites, browser workflows, UI builds, and image matrices do not repeat.

The changed supervisor cross-compiles for Linux AMD64 and ARM64.
Three local ARM64 Linux probes verify root-to-1000 transition, restricted UID 1000 operation, and UID 1001 rejection.
All capability sets are empty and no-new-privileges is enabled in the running probes.
The probes use a cached TestStudio source base, a read-only binary mount, and no network. They build or pull no image.
Owned containers stop with PID zero and are removed. The corrected missing-object check also confirms the first probe's removal.
Owned local OCI TLS listeners close through normal test cleanup.

The [command guide](azure-sandbox-commands.md), supervisor guide, Azure guides, and active plan match the new contract.
Native source-image resolution, actual admission, full Provider configuration, and deployment packaging remain unfinished.
Actual Azure execution, command limits, converted runtime behavior, and permissions remain separate deployment evidence.
The user demo and page-help behavior remain unchanged. No cloud account, external registry, identity endpoint, or deployment is used.
This advances **51** and supporting recovery under **52**. The total remains **48 verified of 57**, with **9 open**.
Evidence is output/azure-sandbox-commands/summary.json and the retained targeted results, builds, identity observations, and cleanup records.

## Previous delivery: independent sandbox deadline SDK and workflow

The sandbox deadline interface now has a concrete Azure Logic Apps SDK implementation.
It reserves one workflow create call in the existing dispatch journal and verifies the exact owned definition before allowing sandbox creation.
The workflow uses the selected stop identity and regional data-plane API to find and stop all matching logical ownership labels.
It can cover delayed creation whose response never supplies a receipt.
Its bounded, manually checked pagination retains the selected group, API version, and ownership filter.
Sensitive HTTP and selection history is hidden. Runtime environments do not enter workflow variables.

The SDK uses an explicit management credential, fixed ARM scope, disabled retries and registration, bounded bodies, and rejected redirects.
Workflow identity, timing, policy, tags, and history settings participate in ownership verification.
Known timer rejection remains distinct from uncertainty. A missing timer never proves the worker execution absent.
Creation uncertainty remains durable until owned readback appears.
Removal requires revoked work, retained result and evidence checkpoints, settled instance ownership, and another instance absence check.
Timer deletion saves intent and backoff before the request and completes only after confirmed absence.
A lost deletion response reconciles after database reopen without another delete.

Five new deadline checks and three selected controller regressions pass with race detection in 2.006 seconds.
They use the production ARM SDK and scoped transport with local in-memory responses, real SQLite storage, and the real worker exchange.
Both runners and discovery cover lost creation responses, late timer appearance, definite rejection, unreserved resources, changed metadata, redirects, and checkpointed cleanup.
A later missing-timer classification correction repeats only its changed scenario and passes in 1.474 seconds.

An independent check validates the generated definition against the downloaded Microsoft workflow schema.
The schema catches variable type capitalization and its one-variable initializer limit. The corrected workflow passes.
The local checker also validates expression delimiters, variables, action references, sibling dependencies, dependency cycles, and HTTP history protection.
The generated definition has 37 actions, six initialized variables, and 23 expressions.
This checks schema and static references. It does not execute the workflow or establish runtime stop enforcement.
The schema download, checksum, and separate local validator environment remain retained with the evidence.
No product runtime dependency or existing SDK version changes.

The [sandbox deadline guide](azure-sandbox-deadlines.md), controller and discovery guides, Azure overview, and active plan describe the concrete implementation and remaining work.
Independent command completion is next, followed by durable image import, actual admission, native composition, and deployment packaging.
Effective identity grants, workflow protection, scheduling, filter behavior, and stop enforcement remain separate deployed evidence.
Native Azure coded execution remains disabled. The user demo and page-help behavior remain unchanged.
No account request, identity discovery, registry operation, listener, image, browser, container, or deployment runs.

This advances **51** and supporting cleanup recovery under **52**. The total remains **48 verified of 57**, with **9 open**.
Evidence is output/azure-sandbox-deadlines/summary.json, schema-validation.json, and the retained focused command results.

## Previous delivery: durable sandbox dispatch, receipt recovery, and cleanup

The Azure sandbox Controller now persists dispatch reservations in the shared installation database.
It uses the real worker exchange for fresh dispatch authorization, explicit activation gates, revocation, and result cleanup checkpoints.
One atomic reservation binds the exact reference, project, launch proof, and frozen input hash.
Current work-item bytes, source image, runner, run, test, project, and discovery source digest must match that reservation.

The controller saves deadline intent before external timer creation and requires verified coverage before reserving a sandbox create call.
Only the winning create reservation dispatches. A lost acknowledgment never triggers a replacement create.
Recovery resolves one full configuration match from complete inventory, then saves the exact returned instance identifier.
An empty inventory cannot discard an uncertain create. Duplicate matches reject ownership.
Already known receipts use exact reads and reject changed execution settings.

Cancellation saves a barrier before resource control. A concurrent initial dispatch cannot pass the same guarded revision.
Late receipts remain recoverable while canceled work stays unadmitted.
Stop acknowledgments remain distinct from confirmed Stopped or Suspended readback.
Cleanup requires revoked exchange work and the canonical result and evidence checkpoint.
It confirms quiescence, records deletion intent and backoff, verifies instance absence, and removes the independent deadline last.
Lost deletion and timer-removal responses retain their durable checkpoints through database reopen.
Selected inventory records validate before completed removal filtering. An invalid Removed flag cannot hide active work.

The integration review finds and corrects a source contract mismatch in the earlier staged plan.
The OCI inspector uses sha256:HEX source digests. The earlier sandbox validator and its fixture expect bare hexadecimal characters.
The v2 proof now accepts and freezes the canonical digest. Native sandbox records do not exist from the earlier staged format.
Wrapped dependency errors retain fixed classifications without exposing callback details or usable tickets.
Unchanged observations assert revisions without rewriting identical dispatch state.

Nine new lifecycle checks and three selected affected plan and response checks pass with race detection in 1.896 seconds.
They use real SQLite storage and the real worker exchange with local simulated provider controls.
Coverage includes both runners, discovery, database reopen, twelve concurrent callers, uncertain creation, in-flight cancellation, work-item changes, scope boundaries, admission loss, and cleanup checkpoints.
A later test teardown change joins its owned in-flight routine on every exit. Only that changed check repeats and passes in 1.434 seconds.
No unrelated product suite, UI build, image, registry operation, identity discovery, browser, listener, container, cloud test, or deployment runs.
The user demo remains unchanged. No product route or help-page behavior changes.

The controller reports resource facts. It does not implement the complete Provider or treat stopped execution as a passing test.
Concrete sandbox deadlines and admission, durable image import, independent command completion, native composition, and deployment packaging remain required.
The [controller guide](azure-sandbox-controller.md), Azure and discovery guides, and active plan describe these boundaries and continuation work.
This advances **51** and supporting recovery under **52**. The total remains **48 verified of 57**, with **9 open**.
Evidence is output/azure-sandbox-controller/summary.json and the retained focused command results.

## Previous delivery: isolated Azure sandbox launch and data-plane client

The active Azure coded execution path now uses Container Apps Sandboxes. The UI and API remain Container Apps.
Ordinary Jobs remain disabled because their execution isolation boundary is unresolved.
The existing plan permits another suitable Azure backend. Deployment targets and the stable 57 acceptance numbers stay unchanged.
Sandboxes and the selected API remain preview. This local client does not establish hosted acceptance.

The new Go package implements immutable image import, image reads, bounded image inventory, sandbox creation, status, stop, deletion, and bounded sandbox inventory.
It pins data-plane API 2026-02-01-preview and the regional endpoint format from Microsoft SDK 0.1.0b4.
Source inspection verifies the wheel checksum. No Python dependency is installed, executed, or included in the product.
The implementation follows the pinned SDK's dynamicsessions token audience. Separate deployment verification must confirm that audience against the actual service.

Launches require an approved source inspection result and Ready private disk image for the exact project and runner.
They fix the supervisor, resources, VNet connection, virtualization selection, Full outbound inspection, default Deny, and one exact exchange-host Allow rule.
Only the execution identifier and attempt-scoped worker ticket enter the environment.
No registry password, controller identity, application secret, shared volume, port, preset, or snapshot enters the launch.
Both execution runners and both discovery operations use this contract.
Proofs bind the launch definition and retain ticket and environment hashes without a usable ticket.
Restored proofs verify readback but cannot dispatch. Value and pointer diagnostics also omit the ticket.

The client uses explicit credentials, no redirect following, no environment proxy, no automatic mutation retries, and fixed error classifications.
Requests, credentials, responses, inventory pages, total items, continuation scope, and duplicate identifiers have explicit bounds.
Lost acknowledgments stay unavailable. Incomplete inventory returns no partial result or authoritative absence.
Accepted stop and delete responses remain acknowledgments. Provider state does not invent a passing result, exit code, or duration.

Ten focused package checks pass with race detection in 1.369 seconds.
They exercise the production HTTP client through an in-memory transport, including lifecycle requests, ownership mutations, lost responses, unsafe continuations, cancellation, and body cleanup.
A review extends diagnostic redaction to copied plan values and retains writable virtualization settings in ownership comparison.
Only those changed checks repeat during review. The final new-package check runs after the explicit virtualization binding changes its shared launch schema.
No unrelated product suite, UI build, image, browser, listener, credential discovery, registry operation, cloud test, or deployment runs.
The existing user demo is preserved. No product route or page-help behavior changes.
The [sandbox operator guide](azure-sandboxes.md), Azure guides, and active plan record the preview dependency and remaining integration requirements.

Native command selection, durable create reservation, lost-receipt reconciliation, independent sandbox deadlines, complete reports, and checkpointed cleanup remain required.
Idle suspension does not satisfy the absolute deadline contract. The existing ARM Job timer does not stop these sandbox instances.
Complete actual identity, VNet, egress enforcement, source conversion, proxy trust, and placement verification remain separate hosted work.
This advances acceptance **51**. The total remains **48 verified of 57**, with **9 open**.
Evidence is output/azure-sandbox-contract/summary.json, final-contract.json, final-contract.log, the focused review checks, and the retained SDK provenance.

## Previous delivery: Azure preparation egress and private journal network

Native Azure assembly now requires explicit network settings after the host and writer checks.
The source subnet selects a dedicated NSG and Standard NAT gateway, with private PostgreSQL, registry, and SMB destinations.
The policy retains the canonical Container Apps platform rules and adds trusted metadata, vault, and monitor HTTPS service tags.
The source has explicit disabled default outbound access, the selected NAT, successful environment delegation, and no unexpected user route or service endpoint.
The controller VNet reports its configured address spaces, default Azure DNS, and matching direct and embedded subnet metadata.
The NAT reports one static regional public IPv4 address, the exact source subnet, and matching bidirectional ownership.

The selected journal account, SMB share, approved file endpoint, NIC, private subnet, source-only NSG, and no-fallback DNS zone now bind to the same host account and share.
Public storage access, bypass rules, alternate consumers, wrong approvals, NFS, unexpected routes, and changed destination addresses block new publication.
Native composition adds two owned readers, bringing its total to thirteen. Construction makes no provider call.
The network stage refreshes its preceding host chain once and checks eleven selected resources, then the bounded private DNS inventory.
The pinned armstorage v1.8.1 direct dependency supplies account and share GETs. Existing SDK versions and checksums remain unchanged.

The separate preparation-network.bicep declares source egress resources, a private-only journal account and SMB share, its file endpoint, DNS, and account deletion protection.
Its outputs populate the private network settings. It retrieves no storage key or secret and declares no app or test job.
The native command also checks the current Linux mount table before database access.
Its journal path must be one writable CIFS mount of the configured account and complete share.
A private probe confirms file write and synchronization, directory synchronization, probe removal, and synchronization after removal.
Ephemeral, read-only, other-account, other-share, substituted-subpath, duplicate, and incomplete mounts reject startup.
This does not prove remote recovery or complete platform enforcement.

Five new network checks and three affected native and publisher checks pass under race detection in 3.389 seconds.
Four affected native assembly checks pass in 1.429 seconds.
The new mount parser check first passes in 1.525 seconds. After adding the local synchronization and cleanup scenario, only that check repeats and passes in 1.421 seconds.
The uncertainty check repeats after adding explicit missing approval-status handling. Its focused result passes in 1.709 seconds.
Its temporary directory and probe are owned by the test and cleaned before completion. Network fixtures use fake credentials and in-memory SDK transports.
The network package passes five local packaging checks. Its initial compile reports a hardcoded DNS suffix warning; an environment suffix resolves it.
The final Bicep compile passes without warnings. Checker refinements reuse that compiled template.
The worker and core package checks pass against retained compiled templates, without rebuilding either package or image.
Operator guides document network settings, setup sequence, actual mount requirements, SDK limits, remaining evidence, and continued isolated discovery requirements.
Documentation links and the stable 57-item ledger remain consistent. No UI, API contract, or page help behavior changes.
No actual registry, identity endpoint, cloud service, listener, image, container, browser, or deployment runs. The user demo remains unchanged.

Native Azure discovery and execution configuration remain the next implementation work.
Actual cloud authorization, routing, remote filesystem behavior, and replica recovery remain separate deployment evidence.
This completes no broad acceptance item. Acceptance 51 advances; the verified total remains 48 of 57, with nine open.
Evidence is output/azure-preparation-network/summary.json and its retained packaging results.

## Previous delivery: Azure assembly host and startup admission

Native Azure assembly now requires a host binding and compares selected reported app, environment, persistent storage, and reader identity metadata before queue startup.
The reader's selected subscription assignment and fixed custom read role must match its client, principal, and installation tenant.
The host app pins reader and writer metadata while keeping platform pull and secret-startup identities outside the main runtime.
The environment must bind the preparation source subnet with internal networking, disabled public access, and a Consumption profile.
The selected storage binding must name the configured read-write SMB account and share. NFS and read-only bindings reject.
The SDK returns a reduced storage projection without storage keys or credential references.

Shared service validation now supports explicit headless apps and absent or zero HTTP traffic weight on headless revisions.
Existing HTTP app and revision behavior remains required. The host compares app and revision against an independent fixed runtime definition.
Command, args, image, runtime digests, vault references, identities, journal mount, resources, and scaling must match.
Consistent malicious app and revision changes do not become the expected template.
One owned host reader extends the native publisher to eleven readers. Its preceding writer chain refreshes once per check.
Host observation completes before writer credential selection and before each new source assembly.
The native command rejects a missing host, mismatched runtime flags, or a different journal directory before state creation.

Startup waits up to three minutes for unavailable platform observations, with two seconds between attempts.
A definite ownership mismatch stops immediately. Cancellation stops the wait. No job is claimed during activation waiting.
The template outputs the complete host binding, including non-secret image, identity, storage, runtime, and vault-reference values.
The operator guide explains host settings, updated bootstrap, upgrade bindings, runtime matching, limits, and deployment evidence.
No UI, API, or current-page help behavior changes.

Four new host checks and five affected adapter regressions run under race detection in 2.576 seconds.
One new case initially expects unavailable for a foreign revision page; the shared reader correctly reports ownership.
Only that corrected check repeats, passing in 1.639 seconds. The remaining selected checks already pass and do not repeat.
A focused added NFS subcase passes in 1.533 seconds after separating a definite unsupported storage type from missing metadata.
The new startup wait check and three affected native assembly checks pass under race detection in 1.426 seconds.
The Bicep package compiles once without warnings. Its six packaging checks pass after a checker-only correction for an inlined image expression.
The corrected checker reuses the compiled template. No image, container, browser, listener, real registry, identity endpoint, deployment, or live provider request runs.

Selected reported bindings do not prove process origin, complete effective grants, network enforcement, storage durability, or identity exclusivity.
Source egress, storage network binding, actual SMB recovery, and complete isolated discovery remain unfinished or unverified.
Acceptance 51 advances; the ledger remains 48 verified of 57, with nine open.
Evidence is output/azure-preparation-host/summary.json and retained package results.

## Previous delivery: native Azure assembly and persistent workload template

The native preparation command now selects Azure through --azure-assembly-config or --azure-assembly-env, only with assembly-only mode and the Azure location.
It requires PostgreSQL and a separate explicit reader client identifier. Invalid modes, mixed providers, invalid runtime references, and invalid settings reject before state creation.
Startup opens the existing publisher with the exact reader identity and selects the separately verified writer for registry authentication.
Each new source build refreshes private-path and selected-grant observation before publication.
Both Go and Playwright builders retain owned recovery without registering any discovery inspector.
Cleanup and confirmed handoff preserve the existing short lease and owned journal behavior without another control-plane observation.

The separate preparation.bicep template declares one optional headless worker with four dedicated identities and separate registry, metadata, and named-secret grants.
The worker uses the current digest-addressed service image, PostgreSQL and private-settings vault references, one always-on replica, and a persistent Azure Files journal mount.
The mount requests private modes and uid and gid 65532. The template performs no secret-value or storage-key retrieval.
workerEnabled defaults to false so installation bindings can be established before the private configuration secret is populated.
The existing core package remains unchanged. Actual preparation placement and mounted filesystem behavior remain unverified.

Three new native command checks and the affected local-mode rejection check pass under race detection in 1.411 seconds.
One new explicit-reader check and the affected publisher readiness and closure check pass under race detection in 1.562 seconds.
The new Bicep template compiles once with the pinned compiler, without warnings, restoration, Azure CLI, credentials, or deployment.
Six packaging checks pass after correcting checker assumptions about the Docker COPY form and implicit identity dependencies; the compiled template is reused for both corrections.
The shared core checker also passes using its retained compiled template, without compilation or image builds.
No cloud service, real registry publication, listener, container, browser, or demo restart runs. No new test session needs cleanup.
Native and operator documentation match the available boundary. No UI, API contract, or current-page help behavior changes.

Next bind reported preparation app, environment, persistent storage, and reader identity metadata before native queue admission.
Effective authority, source egress, SMB durability, deployed recovery, and complete isolated discovery remain unfinished or unverified.
The native assembly process runs no uploaded code and enables no Azure coded execution. It cannot complete review without compatible isolated discovery.
Acceptance 51 advances; the verified total remains 48 of 57, with nine open.
See [native Azure assembly](azure-native-assembly.md). Evidence is output/azure-native-assembly/summary.json and its retained package results.

## Previous delivery: Azure preparation writer composition

Explicit preparation library settings now compose the existing registry observation chain, trusted preparation networking, selected writer identity, and registry-scoped AcrPush grant.
The factory reuses the existing registry builder and owns ten SDK readers. Construction makes no provider call.
Each registry client opening refreshes the path, identity, assignment, and fixed built-in role before selecting the verified managed identity client.
Publishing uses that separate credential, the configured tenant and repository prefix, and the selected regional data authority.
Changed scopes, principals, conditions, role actions, incomplete metadata, provider failures, and unselected SDK operations block opening.
Repeated publisher closure releases each owned reader once; closed publishers perform no provider operations.

Three new writer race checks pass in 2.142 seconds.
The new composed publisher check and one affected existing observer check pass in 1.839 seconds after extracting their shared registry factory.
Checks use fake credentials and in-memory SDK transports. No cloud request, real registry publication, listener, browser, image build, deployment, or demo restart runs.
No UI, API, or page help behavior changes. The operator guide, Azure guidance, and plan describe the library boundary and remaining native work.
Selected assignment observation does not prove exclusive or complete effective authority. Workload placement, persistent journal storage, bootstrap read authority, and native command wiring remain unfinished.
Azure coded execution remains disabled. Acceptance 51 advances; the verified total remains 48 of 57, with nine open.
See [Azure preparation writer](azure-preparation-writer.md). Evidence is output/azure-preparation-writer/summary.json.

## Previous delivery: durable source assembly and discovery handoff

The preparation service now supports an optional assembly-only role and a durable handoff to the normal discovery queue.
The writer publishes a validated immutable image, releases its owned journal without deleting content, and commits the handoff, lease removal, and audit atomically.
Assembly and discovery retain separate references. Claim checks now bind the exact reference as well as worker identity.
The API can discover the handed image and complete review, approval, publication, and owned discovery cleanup without a builder or registry writer.
Assembly-only workers run no discovery. Compatible discovery workers recheck retained source, expiry, current access, and normal artifact provenance.
Confirmed interrupted assembly resumes only its handoff; concurrent writers and repeated recovery create no duplicate build or discovery index.
Waiting cancellation, expiry, revocation, and cancellation during handoff block dispatch. Old jobs and the default combined workflow remain compatible.

The native preparation command adds --assembly-only for existing ECS and Kubernetes registry settings and rejects local mode before creating state.
Azure settings remain unwired. This establishes a queue boundary for a future trusted Azure writer without enabling native coded execution.
OpenAPI 0.34 and generated client types include the two optional assembly fields.
Preparation and repository views show Waiting for discovery; page help explains when review becomes available.

Six new behavior checks, eight targeted existing source regressions, generated-contract synchronization, and the Angular production build pass locally.
The first source selection exposes missing expiry validation after queue handoff. Retained source now revalidates before discovery, including before retention sweeping.
Only the four new source checks repeat after that fix, passing in 4.914 seconds. Registry preservation passes for both runners in 1.426 seconds; native rejection passes in 1.410 seconds.
Contract synchronization passes in 0.269 seconds. The final UI build passes in 2.830 seconds and reuses the private viewer.
A premature UI build after a script path error and a later one-line component formatting error are corrected before the final build; neither is used as final UI evidence.
No existing browser workflow repeats: the UI changes only label the new state and add help, with no new controls or layout. Existing keyboard, theme, and narrow-layout evidence is retained.
No cloud operation, image build, container, registry push, deployment, browser, or demo restart runs. Local TLS fixture servers close through test teardown.
See [separate source assembly and discovery](source-assembly-handoff.md). Acceptance 51 advances; the total remains 48 verified, nine open.

Evidence is output/source-assembly-handoff/summary.json and the targeted command results.

## Previous delivery: trusted preparation registry path observation

A separate PreparationRegistryGuard now checks the selected trusted registry path after the existing registry service and worker network observation chain.
It binds the approved preparation endpoint to its NIC, exact registry and data members, dedicated endpoint subnet and security rules, source subnet, controller VNet, and separate private zone.
Fresh VNet reads preserve worker separation, prefix containment, core-subnet separation, and direct-versus-embedded subnet agreement.
The fixed security rules allow exactly the selected private controller VNet prefixes to the two IPs on TCP 443, followed by inbound and outbound deny rules.
This preserves trusted controller and backend platform image pulls alongside preparation; writer authority remains separate.
Configured controller spaces copy and sort into the bound settings, match fresh VNet metadata, and cannot overlap workers.
The private zone has exact preparation addresses and one completed controller link, with no registration or public fallback under the existing DNS contract.
The two explicit-credential readers reuse the network and DNS SDKs. Static SDK definitions support installation review without a cloud mutation.

Four new targeted race checks pass after the final policy correction in 2.070 seconds.
Review adds fresh address-space separation, containment, caller-copy protection, and the existing canonical single-prefix representation helper.
Earlier verification repeats only changed branches. The final controller-prefix source policy receives all four directly affected checks.
No existing Go check, broad suite, UI, browser, package, image, container, provider call, publication, deployment, registry push, or demo restart runs.
The standalone guard has a 40-second context, including its existing 30-second registry parent, with nine ordinary additional requests and a twelve-request additional paging ceiling.
It repeats no complete parent inspection and caches no success. Parent failure blocks preparation reads; network failure blocks DNS.
The existing host-bound observer and its 22-reader limits remain unchanged.

The source queue also requires isolated discovery. These read-only checks do not enable publication, a preparation queue, native coded execution, or complete Network admission.
Actual preparation workload placement, effective DNS and egress, separate writer identity grants, protection-management authority, and native wiring remain required.
See [trusted preparation registry path](azure-preparation-registry.md). Acceptance 51 advances; the total remains 48 verified, nine open.

Evidence is output/azure-preparation-registry/summary.json and the targeted command results.

## Previous delivery: reported controller HTTP route absence

The host observer now requires an explicit empty controller-environment HTTP route collection on every successful Check.
Its mandatory RoutesAbsent method reads one selected page through the existing SDK client and observer credential.
Any route object rejects ownership. Missing or null values, null-only rows, failed reads, and incomplete empty pages remain unavailable.
It never follows continuations, even to the same collection, and returns no route names, hosts, targets, rules, or response bodies.
The staged profile supports no environment routing overrides, including routes for unrelated apps. The inspector changes no existing route.
The core package declares no such override. Installation tooling must select a suitable controller environment.

Three new focused race checks and seven affected host or protection regressions pass in 4.929 seconds.
They cover explicit absence, present and unknown routes, same and foreign continuations, exact scope, rejected queries and mutations, cancellation, freshness, and parent gating.
Four concurrent host reads now yield four ordinary GETs and at most six host-specific requests with revision paging.
Only the host ceiling increases to 2222 requests; its context remains 135 seconds with 22 readers.
Bootstrap remains 125 seconds and 2200 requests; diagnostic read-deny remains 130 seconds and 2216 requests.
No dependency, grant, additional reader, repeated parent scan, observation cache, package build, UI, browser, image, container, provider call, deployment, registry push, or demo restart is added.
See [reported controller route absence](azure-observer-host-routes.md). Acceptance 51 advances; the total remains 48 verified, nine open.
Reported absence does not prove effective visibility, prevent future route writes, establish external proxy behavior, or complete native execution admission.

Evidence is output/azure-observer-host-routes/summary.json and the targeted command result.

## Previous delivery: explicit reported host platform authentication

The host reader now requires the selected API app's authConfigs/current with an explicit false platform enabled flag.
It uses the existing SDK and returns only resource identity and that flag. Provider, registration, encryption, and login metadata do not reach callers.
Missing resources and unknown state remain unavailable. A changed enabled setting rejects the next bound inspection, while restored disabled state can pass.
The core app module now declares disabled platform authentication for both API and artifact roles, regardless of the optional observer setting.
TestStudio retains its own OIDC, local-account, session, and private-report authorization contracts.

Three new focused race checks and seven affected regressions run locally. Nine pass initially; one exposes a reader's missing-resource error category.
After that normalization is corrected, only the SDK privacy check and bound authentication check repeat and pass in 1.972 seconds.
The initial targeted selection takes 3.503 seconds. The changed Bicep module compiles once with pinned 0.48.1 and passes package contracts without warnings.
Three concurrent host reads replace two, increasing only the host ceiling to 2221 requests. Its context remains 135 seconds with 22 readers.
Bootstrap remains 125 seconds and 2200 requests; diagnostic read-deny sessions remain 130 seconds and 2216 requests.
No role grant, dependency, repeated parent scan, observation cache, UI build, browser, image, container, provider operation, deployment, registry push, or demo restart is added.
See [reported host platform authentication](azure-observer-host-auth.md). Acceptance 51 and supporting packaging under 57 advance; the total remains 48 verified, nine open.
Reported state does not establish deployed authentication behavior, other identity consumers, protection-management authority, execution isolation, or complete native admission.

Evidence is output/azure-observer-host-auth/summary.json and package/summary.json.

## Previous delivery: fixed reported API host runtime

The observer host now requires explicit runtime settings and an independently constructed expected core API definition.
App and active revision must match that definition, even when both report the same changed arguments or environment.
The contract binds API role, PostgreSQL, separate origins, Azure Blob runtime client, Key Vault and OIDC references, bootstrap identity, and local-account policy.
It rejects command overrides, additional arguments or variables, alternate storage, inline credentials, extra mounts, substituted ingress, probe drift, and replica changes.
The existing image, identity exposure, lifecycle, placement, and healthy revision checks remain required.
Optional OIDC client references follow the two existing package slots. Public clients need no secret.

The optional observerBinding package output now includes matching non-secret runtime settings from the API deployment parameters.
It exposes references and bootstrap identifiers, not secret values or tokens. The observer remains disabled by default, with no native worker startup.
Four new focused race checks and three changed host regressions pass in 2.918 seconds.
Four directly affected federation, protection, and identity-placement host checks pass in two targeted selections, taking 2.172 and 2.039 seconds.
The first dependent selection matches two tests; the two other intended checks run afterward under their exact current names.
The changed output compiles once with pinned Bicep 0.48.1 and passes local package contracts without warnings or registry restore.
No unchanged Go suite, UI build, browser, image, container, provider operation, registry push, deployment, or demo restart runs.

The guard reuses the existing two host GETs, 22-reader session, 135-second context, and 2220-request ceiling.
No new dependency, grant, scan, credential, or observation cache is added.
Reported configuration does not prove secret contents, actual image behavior, platform authentication settings, effective authorization, or management authority.
See [fixed host runtime contract](azure-observer-host-runtime.md). Acceptances 51 and supporting packaging under 57 advance; the count remains 48 verified, nine open.

Evidence is output/azure-observer-host-runtime/summary.json and package/summary.json.

## Previous delivery: reported protection-stack policy and membership

The identity stage now requires two selected deployment stacks with fixed policy and exact protected-root membership.
The network stack reports the five existing VNet and DNS roots. The distinct identity stack reports only the observer identity.
Both require Succeeded, denyWriteAndDelete, child coverage, no excluded principal, and the fixed effective exception sets.
The SDK reads only the two selected resources and drops templates, parameters, outputs, and deployment metadata from returned observations.
Failed or changed protection blocks credential construction and later host reads. Successful state is never cached.

Three new focused race checks and six changed identity, scan, and session checks pass in 3.668 seconds.
After correcting missing-metadata error categories, only the two affected checks repeat and pass in 1.925 seconds.
Local checks first exposed value-shaped SDK responses and an incompatible resource-group-only helper; both are corrected before the passing runs.
Seven concurrent identity-stage observations now yield ceilings of 2200 bootstrap, 2216 read-deny, and 2220 host requests.
Contexts remain 125, 130, and 135 seconds. The host session retains 22 readers and repeats no parent scan or bootstrap during a bound check.
The stable armdeploymentstacks/v2 v2.0.0 adds one module and two hashes without upgrading other dependencies.

The API exposes no exact deny-assignment link. These checks establish reported consistency, not direct ownership or stack-management authority.
The optional core package creates no protection arrangement. Actual enforcement, provenance, other consumers, complete host admission, isolation, and native wiring remain open.
See [reported protection stacks](azure-protection-stacks.md). Acceptance 51 advances; the total remains 48 verified of 57, with nine open.
No cloud operation, UI build, browser, image build, container, registry push, deployment, or demo restart runs.
External requests retrieve only public documentation and the public SDK module.

Evidence is output/azure-protection-stacks/summary.json and the targeted command results.

## Previous delivery: reported observer identity write protection

Acceptance **51** advances. The verified total remains **48 of 57**, with **9 open**.

ReadIdentitySettings now requires protectionAssignment at the exact selected identity resource scope.
The reader adds a selected GET through the existing authorization SDK and pinned preview API, with no list, mutation, alternate-scope, or optional fallback.
The shared resource-deny verifier requires explicit enforcement, Azure system protection, child coverage, one all-principal entry, and no excluded principals or conditions.
One permission denies the literal star action with exactly read and lock-delete exceptions.
Assignment, credential-write, deletion, and extra DNS-association exceptions reject the identity profile.
The existing network and zone profile keeps its three fixed exceptions through the same verifier.

The five identity observations run concurrently after the existing network protection chain.
Failed identity protection blocks credential construction and completed bound checks. Selected state refreshes after the subscription scan's earlier observation.
The subscription read-deny scan now requires six selected assignments, including identity protection.
Its DNS-association conflict check now respects operation scope. An identity-resource deny cannot block a VNet operation through that scope relationship.
Other inherited, subscription, resource-group, and VNet conflicts retain their existing checks.

One added GET increases identity and bootstrap ceilings to 2198, read-deny sessions to 2214, and host checks to 2218 requests.
Contexts remain 125, 130, and 135 seconds respectively. The host session retains 22 readers.
No parent scan, role grant, dependency, or packaging build is added.
The optional core identity package still declares no protection arrangement and cannot supply an automatic protectionAssignment.

Three new focused race checks, eight changed identity and scan checks, and one shared network-protection regression pass in 4.240 seconds.
They cover exact scope, enforcement, system protection, child scopes, exclusions, conditions, forbidden exceptions, SDK restrictions, factory gating, fresh selected state, and required scan coverage.
The bound host rejects changed protection, accepts a restored response, and blocks when the subscription list omits the sixth assignment.
The scan accepts an unrelated read-preserving identity deny and still rejects a VNet DNS-association deny.
Only affected read counts and contracts change in existing checks. No broad suite repeats.
Fixtures use fake credentials and in-memory SDK transports. They open no network connection or service listener and request no real token.
No cloud operation, UI build, browser, image build, container, registry push, deployment, or demo restart runs.
External requests read only public Microsoft documentation.
Evidence is output/azure-observer-protection/summary.json and the targeted command results.
See [identity write protection](azure-read-identity.md#identity-write-protection).

The read-only inspector creates no assignment and tests no prohibited mutation. Actual enforcement, resource-type support, preview compatibility, and owning protection-management authority remain unproven.
Existing foreign consumers, creation provenance, trusted host configuration, effective coverage, execution isolation, and native command wiring remain required.
All nine broad acceptance items remain open under their original scope.

## Previous delivery: observer federated-credential absence

Acceptance **51** advances. The verified total remains **48 of 57**, with **9 open**.

ReadIdentityReader now requires FederationAbsent alongside identity, assignment, and role observations.
The existing SDK reader permits one additional GET on the selected identity's federatedIdentityCredentials collection through pinned API 2024-11-30.
A successful response requires an explicit empty value array and no continuation. Any non-null credential object rejects ownership without evaluating its issuer or subject.
Missing or null arrays, null-only entries, continuations, denied reads, redirects, or canceled calls remain unavailable.
The reader follows no continuation and provides no credential metadata or provider response to the caller.

The guard refreshes all four observations concurrently after the existing protection stage succeeds.
Reported federation or unknown state blocks managed identity credential construction. Later diagnostic and host-bound checks use the selected observer credential for fresh absence checks.
A successful observation is not cached. Failed absence prevents later host reads without repeating bootstrap inspection.
The collection retains the existing five-second context and four-megabyte response limit. It sends no top, skip token, filter, or mutation.
No optional reader-method fallback is allowed.

One additional concurrent GET increases the identity and bootstrap ceiling to 2197 requests, the read-deny session ceiling to 2213, and the host-bound ceiling to 2217.
The composed contexts remain 125, 130, and 135 seconds. The host session still owns 22 readers.
No session reader, dependency, API version, role grant, or packaging build is added.
The existing subscription read role already covers the collection operation.

Three new focused race checks and four changed identity, coverage, and session checks pass in 3.179 seconds.
Coverage includes reported credentials, missing or null state, forbidden reads, continuations, exact SDK scope, cancellation, factory gating, and fresh complete bound host checks.
The bound host rejects a newly reported credential, accepts a later explicit empty response, and rejects a later incomplete response.
Only counters affected by the additional GET change in existing checks. No broad suite repeats.
Fixtures use fake credentials and in-memory SDK transports. They open no network connection or service listener and request no real token.
No cloud operation, UI build, browser, image build, container, registry push, deployment, or demo restart runs.
External requests read only public Microsoft documentation.
Evidence is output/azure-observer-federation/summary.json and the targeted command results.
See [federated credential absence](azure-read-identity.md#federated-credential-absence).

Other identity consumer types, future attachment and credential-write authority, trusted host configuration, effective coverage, and native command wiring remain required.
An empty collection response does not prevent a later privileged identity write or establish complete exclusivity.
All nine broad acceptance items remain open under their original scope.

## Previous delivery: reported observer identity placement in existing workload scans

Acceptance **51** advances. The verified total remains **48 of 57**, with **9 open**.

The workload SDK projection now retains reduced identity references, reported client and principal metadata, a system principal, and explicit observation state.
It still discards application and job configuration, environment values, secrets, templates, and unrelated settings.
Identity projection copies and sorts references. Missing or contradictory observations clear previous identity state rather than retaining stale success.
An explicit None identity records a known empty set. Missing objects, missing types, null entries, malformed references, and invalid metadata remain unknown.
Unrelated identity metadata can omit client and principal values. The selected observer entry requires both values and agreement with its configured binding.
Each workload has a 64-reference bound. Case-equivalent duplicate references remain unavailable.

NewSDKObserverWorkloadInventoryReader validates host-specific placement while reusing the existing subscription app and job operations.
BindHostObserver selects that reader during construction. The selected host must appear exactly once in its controller environment with the observer identity.
Other reported apps or jobs cannot attach that identity, even with lifecycle None. Reported client or principal aliases also reject ownership.
Unknown identity observations reject the host-bound check as unavailable. Earlier placement-only readers keep their original scope and compatibility.
Failed sharing or presence checks block the later host reads. Successful checks refresh the current collections without repeating bootstrap inspection.

This change adds no SDK resource, API version, permission, collection request, or session reader.
The existing 22-reader session retains its 135-second context and 2216-request ceiling.
Three new focused race checks and two selected existing host and placement checks pass in 2.410 seconds.
They cover complete paged SDK projection, reported sharing, aliases, unknown state, explicit empty identities, missing hosts, freshness, and bounded collection reuse.
Configuration canaries remain outside projected records. Complete bound checks verify no second subscription scan.
No unchanged broad suite repeats. No dependency changes or packaging rebuilds are needed.
Fixtures use fake credentials and in-memory SDK transports. They open no network connection or service listener and request no real token.
No cloud operation, UI build, browser, image build, container, registry push, deployment, or demo restart runs.
External requests read only public Microsoft documentation.
Evidence is output/azure-observer-placements/summary.json and the targeted command results.
See [observer identity placement](azure-worker-workloads.md#observer-identity-placement).

This establishes reported Container Apps and job checks only. Other resource types, cross-subscription use, credential federation, and future attachment authority remain unproven.
Complete trusted host configuration, effective coverage, actual token behavior, deployed enforcement, execution isolation, and native command wiring remain required.
All nine broad acceptance items remain open under their original scope.

## Previous delivery: observer host attachment and optional core identity packaging

Acceptances **51** and supporting packaging under **57** advance. The total remains **48 verified of 57**, with **9 open**.

ObserverHostGuard now checks the selected controller app's reported observer attachment, core identities, explicit lifecycle settings, image, and current revision.
Its host settings bind the complete preceding identity configuration, selected app, pinned service image, and separate core pull, startup, and runtime identities.
Worker, relay, backend, and observer identity substitution is rejected before bootstrap or credential selection.
The observed host must use the controller environment, UserAssigned identity type, exactly four distinct identity records, and explicit None or Main lifecycle values.
The observer client and principal must match the preceding verified identity. System identity, sidecars, init containers, image changes, and placement conflicts block observation.
Single revision mode and the shared revision verifier require a healthy current revision with matching template, replicas, and traffic.

BindHostObserver builds all 22 SDK readers with the same verified observer credential. The existing diagnostic binding retains its 21-reader scope.
Two selected concurrent GETs follow one existing parent inspection. No bootstrap or successful resource observation is reused during later checks.
The composed check has a 135-second context and at most 2216 requests, with two normal additional GETs.
The host reader permits only app and revision paths, with existing paging, response, deadline, retry, and provider-registration controls.
Close releases all 22 readers. This stage implements no Network and enables no native Azure coded job.

The core Bicep package adds an optional observer identity, fixed subscription custom role, assignment, and metadata output.
The option defaults to false. Only the API main container can receive this identity, including when the shared module is called directly.
The observer role grants management reads and the existing DNS-association observation without data or management-write authority.
Core startup, image pull, Blob runtime, and artifact identities retain their roles and lifecycle controls.
Enabling the option prepares identity resources. It wires no native observer startup or coded job dispatch.
The packaging checker now supports a current compiled-template input to avoid repeated compiler work for assertion-only changes.

Three new focused race checks and one selected existing observer-session regression pass in 2.456 seconds.
An initial fixture literal is corrected before the passing command.
The pinned Bicep compiler reports no warnings. Local package contracts pass first against the current compiled artifact.
Review then gates the optional attachment on the API role inside the shared module. That semantic change receives a final compile and package check.
No unchanged Go suite repeats for that packaging-only change. No dependency version changes.
Fixtures use fake credentials and in-memory SDK transports. They open no network connection or service listener and request no real token.
No cloud operation, UI build, browser, image build, container, registry push, deployment, or demo restart runs.
External requests read only public Microsoft documentation.
Evidence is output/azure-observer-host/summary.json and its final-package compile, template, and summary artifacts.
See [observer host attachment](azure-observer-host.md).

Exclusive identity use, complete trusted API configuration, actual host token issuance, effective authorization, and native command wiring remain unproven.
Keep resource-management authority, protection enforcement, provenance, existing cross-scope links, environment attachments, preparation networking, and execution isolation in full admission.
All nine broad acceptance items remain open under their original requirements.

## Previous delivery: reported subscription read-deny gate

Acceptance **51** advances. The verified total remains **48 of 57**, with **9 open**.

ReadDenyGuard now scans reported subscription deny assignments before the bound observer reads its existing resource chain.
The new SDK reader uses that same observer credential and the already-pinned preview authorization model.
It sends an unfiltered subscription GET, follows only bounded same-path pages, and requires all five selected protection assignments in the response.
Missing selected protection, unknown enforcement or principal metadata, incomplete arrays, duplicate identities, or unsafe pages cannot establish coverage.
The gate rejects potentially applicable enforced conflicts with management reads or the existing bodyless DNS-association operation.
It handles reported inherited scopes, direct principals, all-principal coverage, direct exclusions, and conservative unresolved group membership.
Conditions that could affect a conflicting permission remain unavailable. The check assumes no false condition or unknown group exclusion.
Write-only, delete-only, data-only, explicit audit, and proven read exceptions do not establish management read conflicts.
Overlapping wildcard exclusions can conservatively reject coverage when their general subset relationship remains unproven.

Each collection has a five-second context, 16-page limit, 1024-item limit, and the existing four-megabyte response limit.
The composed check has a 130-second context and at most 2212 requests. Normal added cost is one GET.
The session now owns and closes 21 readers. Bootstrap binding retains its separate existing bounds and performs no new observer request.
No observation is cached. Runtime mismatch rejects before the scan. Failed coverage prevents all later parent reads.
This stage implements no Network and enables no Azure coded execution.

Three new focused race checks and two changed observer-session checks pass in 2.716 seconds.
Coverage includes scope inheritance, principals, exclusions, conditions, enforcement, wildcard semantics, pagination, missing state, exact requests, parent gating, and fresh observations.
The session checks verify its distinct observer test token and all-reader cleanup with the added scan.
No unchanged broad suite repeats. No dependency version changes.
Fixtures use fake credentials and in-memory SDK transports. They open no network connection or service listener.
No real token, cloud operation, UI build, browser, image build, container, registry push, deployment, or demo restart runs.
External requests read only public Microsoft documentation.
Evidence is output/azure-read-deny/summary.json and the targeted command results.
See [reported subscription read-deny inspection](azure-read-deny.md).

Hosted API visibility, actual enforcement, host identity attachment, effective authorization, and complete native wiring remain unproven.
Keep cross-scope visibility, clean provenance, protection-management authority, environment attachments, preparation networking, and execution isolation in the complete admission scope.
All nine broad acceptance items remain open under their original requirements.

## Previous delivery: bound Azure observer SDK sessions

Acceptance **51** advances. The verified total remains **48 of 57**, with **9 open**.

ReadIdentityGuard.BindObserver now builds an owned SDK session using a credential for the verified observed client identifier.
It performs one complete bootstrap inspection, then rebuilds all 20 readers with the selected observer credential and fixed canonical settings.
The session retains no bootstrap reader or bootstrap credential. It provides no alternate-credential fallback.
Client construction makes no observer request. Each later Check refreshes the complete inspection chain through that observer credential.
Successful resource observations are not cached. SDK token caching remains separate from resource freshness.
Each check retains the 125-second context and 2196-request ceiling. Initial binding adds one separate bootstrap inspection with those same bounds.

The composition preserves existing SDK allowlists, API versions, pagination limits, response bounds, disabled retries, and disabled provider registration.
Changed runtime configuration rejects before a request. Missing state, changed grants, token failures, and transport failures remain blocking errors.
Close waits for active checks, closes owned readers in reverse order, releases references, and prevents later inspection requests.
Partial construction closes already-owned readers. The caller retains ownership of its bootstrap readers.
Observer implements no Network and enables no native Azure coded execution.

Two new focused race checks pass in 2.325 seconds. One selected existing credential check runs without failure in the initial targeted command.
The initial new helper expects the wrong token audience. Reusing the established credential fixture corrects the expectation.
Only the two changed checks repeat. No unchanged broad suite repeats.
Every bound SDK request carries a distinct observer test token, checked before routing into existing mutable in-memory fixtures.
Two successful full inspections repeat no bootstrap reads. A changed assignment fails the next fresh inspection.
Additional coverage includes frozen settings, runtime mismatch, nil credentials, bootstrap rejection, token failure, transport failure, cancellation, and all-reader cleanup.
All 20 readers close. Repeated Close and checks after closure produce no new request.
No real token, network connection, service listener, cloud operation, UI build, browser, image build, container, registry push, deployment, or demo restart runs.
Evidence is output/azure-observer/summary.json and the targeted command results.
See [bound observer sessions](azure-read-identity.md#bound-observer-session).

Continue effective read coverage, actual host attachment, complete native command wiring, deployed protection, provenance, attachment authority, and execution isolation.
One matching selected grant and successful local SDK fixtures do not prove complete effective authorization or cross-scope visibility.
All nine broad acceptance items remain open under their original scope.

## Previous delivery: observer identity and selected subscription read grant

Acceptance **51** advances. The verified total remains **48 of 57**, with **9 open**.

ReadIdentityGuard now binds a separate user-assigned identity to its reported client, principal, tenant, exact subscription assignment, and custom role.
It rejects reuse of the known worker pull and backend identities, delegated grants, conditions, different scopes, or substituted principals.
The fixed selected role permits only management reads and the existing bodyless DNS-association observation.
It permits no write, join, data action, permission exclusion, broader management action, or role-management action.
Returned installation definitions are fresh SDK values. They issue no cloud mutation.

Three concurrent selected GETs follow one fresh complete protection inspection. No parent observation repeats.
The composed context is 125 seconds, with at most 2196 requests including bounded reported-subscription inventory.
The reader requires an explicit credential, exact resource paths, and exact API versions. It disables retries and provider registration.
The dependency change adds stable armmsi v1.3.0 and its two checksum entries. Existing dependency versions remain unchanged.
The existing authorization SDK supplies the selected assignment and role reads.
Credential constructs an explicit ManagedIdentityCredential using the verified observed client identifier, with no default fallback.
It performs no token request. It does not prove host attachment, actual bootstrap credential identity, or effective read access.

Five new focused race checks and one selected existing protection regression pass in 4.003 seconds.
The initial observed command fails on the SDK GetByID disambiguation query. Scoped Get corrects the request without widening the transport boundary.
The same six targeted checks then pass. No broad suite repeats.
Coverage includes freshness, parent gating, principal changes, scope changes, extra authority, conditions, missing metadata, SDK scope, and frozen settings.
A private credential factory spy verifies client selection without a real managed identity credential or token request.
Fixtures use explicit fake credentials and in-memory SDK responses. They open no network connection or listener.
No cloud operation, browser, UI build, image build, container, registry push, deployment, or demo restart runs.
Evidence is output/azure-read-identity/summary.json and the targeted command results.
See [observer identity and selected read grant](azure-read-identity.md).

Continue native observer credential wiring and effective read coverage. One selected matching grant does not prove absence of other grants or effective denies.
Deployed protection, clean provenance or authoritative existing cross-scope visibility, protection-management authority, attachment controls, and native isolation remain required.
This stage implements no Network and enables no Azure coded execution. All nine broad acceptance items remain open.

## Previous delivery: selected VNet and private-zone deny observation

Acceptance **51** advances. The verified total remains **48 of 57**, with **9 open**.

The selected deny stage now reads actual resource-scoped assignments for the worker VNet, relay VNet, and their three selected private DNS zones.
Protecting only VNets does not protect records in already-linked zones. Review extends the same contract to those three zones.
Every assignment must report enforced mode, Azure system protection, child-scope coverage, all-principal coverage, no excluded principal, and no condition.
Its single management permission denies the literal star action with only fixed read, lock-delete, and bodyless DNS-association read exceptions.
Audit mode, join or record-write exceptions, wildcard exclusions, conditional permissions, missing protection, and substituted scopes cannot establish approval.
The parent configuration and all five assignment identifiers remain fixed. Broader-scope assignments cannot substitute for the selected resource scopes.

The SDK pins armauthorization/v3 v3.0.0-beta.3 and API 2024-07-01-preview to retain explicit effect and assignment-level condition fields.
No fallback assumes that an omitted enforcement field means enforced. Hosted API compatibility and actual resource provisioning remain unverified.
Only this module and its two checksum entries are added. Existing dependency versions remain unchanged.
Five concurrent selected GETs add no duplicate parent observation. The composed context is 120 seconds, with at most 2193 reads including bounded inventory.
The observer permits no mutation, list, credential, role-management, or alternate-version request.
It implements no Network, authorizes no dispatch, and enables no Azure coded execution.

Four new focused race checks pass for the final five-resource contract in 3.831 seconds.
Two selected existing reported-inventory checks pass in the earlier six-check command, which takes 4.624 seconds.
Only changed deny checks repeat after the protection scope expands to selected zones. No unchanged broad suite repeats.
The initial compile attempt corrects value-shaped SDK response handling before checks pass.
Coverage includes fresh state, parent gating, audit mode, principal exclusions, conditions, child scopes, join and record-write bypasses, action semantics, SDK scope, and unknown state.
No cloud request, browser, UI build, image build, container, registry push, deployment, or demo restart runs.
Only public documentation and SDK module metadata requests leave the local environment.
Fixtures use explicit fake credentials and in-memory SDK responses. They open no network connection or listener.
Evidence is output/azure-network-deny/summary.json and the targeted command results.
See [selected network and DNS deny observation](azure-network-deny.md).

Continue actual assignment provisioning and enforcement, clean installation provenance or authoritative existing cross-scope visibility, and protection-management authority.
The observed deny contract does not remove existing foreign links or prove who can change the owning protection arrangement.
Environment attachment, other perimeter protection, read coverage, trusted preparation networking, and complete native wiring remain required.
All nine broad acceptance items remain open under their original scope.

## Previous delivery: reported subscription private DNS links

Acceptance **51** advances. The verified total remains **48 of 57**, with **9 open**.

The private DNS inventory stage now lists reported zones across the configured subscription and validates their complete reported link inventories.
It detects additional reported links into the selected worker or relay VNet and requires the three existing selected links.
Unrelated valid zones and links remain permitted. Missing selected links, incomplete arrays, pending state, and inconsistent counters remain unavailable.
Duplicate identities, malformed references, foreign-subscription zones, and unexpected selected-VNet links reject ownership.
The observer compares case-insensitive identities and normalizes request paths without changing zone or resource-group names.

The SDK first permits only the subscription zone-list path, then adds only child link collections returned by that validated list.
It projects reduced zone and link references and reads no record values, tags, credentials, or unrelated service configuration.
It lists links even for reported zero-link zones. No dynamically discovered allowlist or successful observation persists between calls.
The bounds are 1024 zones, 4096 total links, sixteen pages per collection, eight concurrent zone reads, and 2048 requests per scan.
The SDK scan has a thirty-second context and five-second operation contexts. The composed guard has a 115-second context and 2188-read maximum.
The normal additional cost is one subscription list plus one link collection per reported zone, before paging.

Public Microsoft documentation confirms that private zones can link across subscriptions and tenants.
This reported single-subscription scan therefore cannot establish complete effective DNS coverage or authority to prevent later attachments.
It implements no Network, authorizes no dispatch, and enables no Azure coded execution.
The plan retains cross-scope coverage, effective read authority, attachment controls, and execution isolation as separate mandatory requirements.

Four new focused race checks and two selected existing checks pass locally in 3.379 seconds.
They cover fresh observations, parent gating, unexpected and unrelated links, zero-link zones, mixed-case identities, incomplete state, paging, request exhaustion, and SDK scope.
No broad suite, cloud request, browser, UI build, image build, container, registry push, deployment, or demo restart runs.
Only public Microsoft documentation requests leave the local environment during this batch.
Fixtures use explicit fake credentials and in-memory SDK responses. They open no network connection or listener.
Evidence is output/azure-private-dns-inventory/summary.json and the targeted command result.
See [reported private DNS inventory and its limits](azure-private-dns-inventory.md).

Continue effective read coverage, actual attachment and protection authority, trusted preparation networking, deployed trust, and complete native wiring.
All nine broad acceptance items remain open under their original scope.

## Previous delivery: selected relay private registry zone

Acceptance **51** advances. The verified total remains **48 of 57**, with **9 open**.

The relay private DNS stage now binds its selected registry zone, complete records, and exclusive VNet link to the preceding verified endpoint settings.
Its separate privatelink.azurecr.io resource cannot reuse the worker registry zone.
The registry login and selected regional data records derive from the preceding member addresses. There is no DNS address or region override.
The exact two A records and apex SOA must agree with reported resource counts.
The sole relay-network link must report completed propagation, disabled registration, and default resolution without public fallback.
Aliases, wildcards, IPv6, extra addresses, automatic records, unrelated links, and incompatible cache bounds reject observation.
Incomplete records, links, propagation, and counters remain unavailable.

Worker and relay profiles share the private-zone verifier, fresh definition generator, scoped SDK reader, and local fixture implementation.
Existing public worker APIs, selected resources, link names, and inventory limits retain their behavior.
The relay adds only its selected zone and two complete collection paths to its own reader.
Three concurrent observations add no duplicate registry service, endpoint, VNet, or source-subnet read.
The composed context is 85 seconds, with 78 normal and at most 140 reads for the explicit relay profile.
It implements no Network, authorizes no dispatch, and enables no Azure coded execution.
Selected records and a selected link do not establish the complete effective DNS view.

Four new focused race checks and five selected existing checks pass locally in 4.506 seconds.
They cover fresh derived addresses, parent reuse, dedicated settings, record and link bypasses, immutable definitions, exact SDK scope, and incomplete inventories.
Existing checks preserve worker record and SOA behavior, unavailable-state handling, public definitions, and SDK construction.
The relay endpoint regression verifies fresh service approval and unchanged parent read counts.
No broad suite, cloud request, browser, UI build, image build, container, registry push, deployment, or demo restart runs.
Fixtures use explicit fake credentials and in-memory SDK responses. They open no network connection or listener.
Evidence is output/azure-relay-private-dns/summary.json and the targeted command result.
See [relay private registry DNS observation](azure-relay-private-dns.md).

Continue authoritative effective VNet-wide zone visibility, read coverage, identity and attachment authority, preparation networking, deployed trust, and complete native wiring.
All nine broad acceptance items remain open under their original scope.

## Previous delivery: explicit relay registry consumer and endpoint ownership

Acceptance **51** advances. The verified total remains **48 of 57**, with **9 open**.

The registry service contract now accepts one explicit optional relayEndpoint beside its required worker and preparation consumers.
Omitting the relay setting preserves exactly two approved consumers and the original paging bound.
Selecting it requires exactly three approved consumers, distinct identifiers, independent embedded agreement, and no unknown fourth consumer.
The SDK bounds connections to the selected consumer count and one additional page. No read path or credential authority expands.

The relay endpoint stage requires that exact nonempty selected consumer before inspecting its endpoint, NIC, subnet, and NSG.
It uses the canonical worker endpoint and interface verification without another registry resource or collection read.
The registry and selected regional data members must use the exact two relay HTTPS destination addresses.
Its dedicated endpoint subnet must enable network policies, disable default outbound access, and contain only the selected endpoint and NIC configurations.
Its dedicated NSG permits relay-source TCP 443 to those two addresses and denies remaining inbound and outbound traffic.
Shared subnets or groups, foreign services, wrong members, alternate addresses, public IPs, route bypasses, and changed rules reject observation.
Reserved subnet addresses and destination-list mismatches reject configuration.

The relay DNS and firewall guards now forward the current validated VNet observation. Public Check behavior remains unchanged.
The new registry subnet must agree with that embedded inventory. Missing or inconsistent observations remain unavailable.
Four concurrent selected reads add no duplicate registry, VNet, or source-subnet observation.
The explicit three-consumer profile has an 80-second context, 75 normal reads, and at most 134 reads with bounded paging.
It implements no Network, authorizes no dispatch, and enables no Azure coded execution.

Six new focused race checks and four selected existing checks pass locally in 4.395 seconds.
They cover service approval, exact members and addresses, parent reuse, dedicated policy, SDK scope, consumer aliases, missing or unknown consumers, and legacy rejection behavior.
Only two changed checks repeat after review strengthens reserved-address and four-page-bound assertions. They pass in 1.623 seconds.
The fixture chain accepts an explicit relay consumer without changing existing two-consumer cases.
No broad suite, cloud request, browser, UI build, image build, container, registry push, deployment, or demo restart runs.
Fixtures use explicit fake credentials and in-memory SDK responses. They open no network connection or listener.
Evidence is output/azure-relay-registry-endpoint/summary.json and the targeted command results.
See [relay registry endpoint observation](azure-relay-registry-endpoint.md).

Continue relay private registry records, effective VNet-wide visibility, preparation networking, identity and attachment authority, deployed trust, and complete native wiring.
All nine broad acceptance items remain open under their original scope.

## Previous delivery: relay DNS policy and actual VNet association

Acceptance **51** advances. The verified total remains **48 of 57**, with **9 open**.

The relay DNS stage now observes its selected resolver policy, exact domain lists, enabled security rules, exclusive link, and actual VNet association.
Worker and relay profiles share one verifier, definition generator, scoped SDK reader, bounded inventory routine, and local fixture implementation.
Existing public worker APIs, names, exchange hostname declarations, and resource scope retain their behavior.
The relay requires dedicated policy and list identifiers that cannot reuse the worker policy or either worker list.
Registry region comes from the preceding selected registry profile without a separate override.

The relay allowlist contains the nine fixed platform domains and four selected registry names.
It omits public exchange and backend lookup declarations because the backend transport uses its pinned private address.
The root-domain block remains mandatory. Exact relay rules and one relay VNet link must agree with the actual VNet policy association.
Worker or controller links, shared policy, changed rules, broad names, bulk URLs, and incomplete observations reject or leave admission unavailable.
DNS hierarchy and CNAME behavior remain separate from exact declaration comparison. Effective private-zone visibility is still unproven.

Six concurrent selected observations add no duplicate parent read. The composed context is 75 seconds, with 71 normal and at most 129 reads.
The SDK accepts one selected bodyless association POST and otherwise only selected DNS GET paths.
No mutation, foreign action, resource substitution, retry, or provider registration is added.
The stage implements no Network, authorizes no dispatch, and enables no Azure coded execution.

Four new focused race checks and five selected existing checks pass locally in 4.106 seconds.
They cover fresh association, parent reuse, canonical settings, immutable definitions, domain and rule bypasses, exclusive links, SDK scope, and paging uncertainty.
Existing worker checks verify hierarchy declarations, incomplete observations, SDK construction, and prior bypass behavior.
The relay firewall regression verifies fresh ownership and network-observation reuse.
No broad suite, cloud request, browser, UI build, image build, container, registry push, deployment, or demo restart runs.
Fixtures use explicit fake credentials and in-memory SDK responses. They open no network connection or listener.
Evidence is output/azure-relay-dns/summary.json and the targeted command result.
See [relay DNS policy observation](azure-relay-dns.md).

Continue selected relay private records, effective VNet-wide visibility, and actual relay registry endpoint ownership.
The existing registry guard still permits exactly worker and preparation consumers. Extend its explicit consumer contract before admitting a separate relay endpoint.
Preserve unknown-consumer rejection, independent networking, identity and attachment authority, deployed trust, and complete native wiring.
All nine broad acceptance items remain open under their original scope.

## Previous delivery: relay firewall ownership and policy

Acceptance **51** advances. The verified total remains **48 of 57**, with **9 open**.

The relay firewall stage now binds its routed private address to the selected firewall, dedicated policy, public IP association, and firewall subnet.
Worker and relay checks share one resource verifier, policy generator, bounded SDK reader, inventory routine, and local fixture implementation.
Existing public worker APIs, rule names, and resource scope retain their behavior.
The relay policy permits HTTPS only from its selected source prefix to the fixed platform domains and retains the final application deny.
Inherited policy, populated default groups, classic rules, extra collections, broad destinations, alternate sources, and TLS interception reject observation.
Relay firewall, policy, and public IP identifiers cannot reuse the selected worker resources.
Its firewall subnet must contain the routed address, avoid both selected subnets, and exclude the selected private registry addresses.
Reserved first-four and final subnet addresses cannot identify the firewall.

The relay topology and perimeter guards now forward their transient validated VNet observation without another read.
The firewall subnet's direct observation must agree with that embedded inventory. Missing or inconsistent observations remain unavailable.
The VNet inventory now permits the exact required AzureFirewallSubnet name while preserving other canonical names and parent restrictions.
The stage adds five normal selected reads. The composed context is 70 seconds, with 65 normal and at most 121 reads.
It implements no Network, authorizes no dispatch, and enables no Azure coded execution.

Four new focused race checks and seven selected existing checks pass across two targeted local commands.
The first command exposes three failures caused by rejection of the reserved firewall subnet name.
After the naming correction, only those failed checks and the affected topology conflict check repeat. The final command passes in 4.431 seconds.
Coverage includes fresh ownership, parent reuse, policy bypasses, subnet consistency, canonical settings, scoped SDK requests, and bounded paging.
Existing worker checks cover resource substitution, policy bypasses, SDK setup, and platform declarations.
No broad suite, cloud request, browser, UI build, image build, container, registry push, deployment, or demo restart runs.
Fixtures use explicit fake credentials and in-memory SDK responses. They open no network connection or listener.
Evidence is output/azure-relay-firewall/summary.json and the targeted command results.
See [relay firewall observation](azure-relay-firewall.md).

Continue effective relay DNS, registry endpoint ownership, identity and attachment authority, deployed trust, execution isolation, and complete native wiring.
All nine broad acceptance items remain open under their original scope.

## Previous delivery: relay source security rules and routing

Acceptance **51** advances. The verified total remains **48 of 57**, with **9 open**.

The relay perimeter stage now binds its dedicated source NSG, source subnet settings, and default firewall route.
It uses the worker stage's canonical platform rule generator, security-rule comparison, route comparison, and scoped SDK reader.
The source subnet must attach its selected NSG and route table and disable default outbound access.
Configured NAT, service endpoints, sharing, and unrelated private endpoints reject observation.
The dedicated NSG attaches only to that subnet, flushes changed connections, and contains the exact thirteen relay rules.
Its private HTTPS destinations contain only the pinned backend address and selected private registry addresses.
The route table attaches only to the source subnet and disables BGP propagation.
Its sole relay-egress route sends 0.0.0.0/0 to the selected firewall address through VirtualAppliance.

The settings freeze and sort private registry addresses. They reject shared worker resources, public addresses, duplicates, and selected-subnet addresses.
Fresh definitions make no resource writes. Configured registry and firewall addresses still require actual ownership observation.
The relay topology guard now returns its current validated subnet observation to this stage. Public Check behavior remains unchanged.
Two concurrent selected reads add no duplicate subnet or VNet GET. The composed context is 60 seconds, with 60 normal and at most 113 reads.
This observer implements no Network, authorizes no dispatch, and enables no Azure coded execution.

Four new focused race checks and five selected existing checks pass locally in 3.718 seconds.
They cover parent reuse, immutable configuration, subnet and rule bypasses, route changes, provisioning uncertainty, and exact SDK scope.
Existing checks verify the shared traffic decisions, worker route comparison, SDK construction, definitions, and relay topology observation.
The first attempt stops at compilation because a fixture uses the wrong SDK NAT reference type. The corrected targeted command passes.
No broad suite, UI build, image build, browser, container, cloud request, registry push, deployment, or demo restart runs.
Fixtures use explicit fake credentials and in-memory SDK responses. They open no network connection or listener.
Evidence is output/azure-relay-perimeter/summary.json and the targeted command result.
See [relay security rules and routing](azure-relay-perimeter.md).

Continue actual relay firewall ownership and policy, effective DNS, registry endpoint ownership, effective identity and attachment authority, and execution isolation.
All nine broad acceptance items remain open under their original scope.

## Previous delivery: selected relay VNet observation

Acceptance **51** advances. The verified total remains **48 of 57**, with **9 open**.

The relay topology stage now observes the selected VNet address space, DNS settings, peering, and bounded embedded subnet inventory.
Its canonical private IPv4 prefix must contain the selected relay and backend endpoint subnets without overlapping the configured worker VNet.
Custom DNS, peering, default public NAT, BGP communities, external IP allocations, and extended locations reject observation.
Subnet identifiers must name unique children. Their private prefixes must lie within the VNet and must not overlap.
Both selected embedded subnets must agree with the current endpoint stage's direct observations.
Additional bounded subnets remain possible for later platform and registry paths. Their workloads and traffic paths receive no approval.

The endpoint guard now returns a transient validated subnet observation to its caller. Public Check behavior remains unchanged.
The composed relay stage adds one selected VNet GET without repeating subnet or backend environment reads.
The read-only SDK accepts an explicit credential and only the selected network resource and API version.
It permits no writes, secret reads, expansions, paging, foreign resources, or redirects.
The composed guard has a 55-second context, 58 normal reads, and a maximum of 111 reads with bounded parent paging.
It stores no successful admission cache and still implements no Network or native execution enablement.

Four new focused race checks and two selected existing endpoint checks pass locally in 3.376 seconds.
They cover fresh observations, parent reuse, alternate paths, prefix and inventory conflicts, incomplete state, exact SDK scope, and parent binding.
No unchanged broad suite, browser, UI build, image build, container, cloud request, registry push, deployment, or demo restart runs.
Fixtures use explicit fake credentials and in-memory SDK responses. They open no network connection or listener.
Evidence is output/azure-relay-topology/summary.json and the targeted command result.
See [relay VNet observation](azure-relay-topology.md).

Continue relay source NSG, route and firewall controls, effective DNS visibility, identity and attachment authority, and execution isolation.
All nine broad acceptance items remain open under their original scope.

## Previous delivery: pinned relay backend readiness

Acceptance **51** advances. The verified total remains **48 of 57**, with **9 open**.
The private backend endpoint observation batch commits as d953360.

The relay now exposes GET /readyz and checks the dedicated worker backend from its own network context.
The probe sends only a fixed GET /healthz through normal TLS verification. It carries no ticket, cookie, user identity, or forwarded caller header.
The backend health and liveness replies now identify role worker. A normal API health reply cannot establish private backend readiness.
The response must retain the exact bounded health schema, valid chain and hostname, and more than two hours and five minutes of certificate life.
Readiness uses HTTP/1.1 on a fresh TLS connection, one probe slot, a separate transport, and a three-second deadline.
Local health and liveness remain independent. Readiness failure exposes only a fixed unavailable response.

The optional backend-address pins both worker operations and readiness to one canonical private IPv4 address.
It preserves the original Host, TLS server name, and explicit backend port without a backend DNS lookup.
Production disables environment proxies, redirects, insecure TLS, and foreign dial targets.
The Azure relay definition now requires backendAddress, the matching command argument, and a four-second /readyz platform probe.
Backend endpoint observation rejects an address that differs from that relay pin.

Five new focused local race checks pass. Ten selected existing relay, command, backend, and Azure definition or endpoint checks pass for the changed contracts.
Only changed certificate, worker health, transport, and configuration cases repeat after review. No unchanged broad suite or fixture build repeats.
Owned TLS fixtures verify normal trust, certificate lifetime, fixed dial scope, worker forwarding, response bounds, cancellation, and concurrency separation.
The actual backend command check confirms the worker-role health reply and graceful process or listener cleanup.
The production dependency audit still reports only standard-library packages plus the existing relay, wire, identifier, and command packages.
Evidence is output/worker-relay-readiness/summary.json and targeted command results.
No cloud credential, cloud request, browser, UI build, image build, container, registry push, or deployment runs.
All owned listeners, request goroutines, idle transports, and the command-check subprocess close.

Both relay and service images require a later rebuild and approval for this new private readiness contract.
The local demonstration's ordinary API role and data remain unchanged.
Pinned backend transport avoids its hostname lookup. It does not establish full DNS isolation or remove worker, exchange, registry, or platform DNS requirements.
Deployed relay routing, VNet controls, certificate trust, effective authority, preparation networking, and execution isolation remain required before native enablement.
The complete goal and all nine broad acceptance items remain open under their stable scope.
See [relay backend readiness and address binding](worker-relay-readiness.md).

## Previous delivery: Azure private backend endpoint observation

Acceptance **51** advances. The verified total remains **48 of 57**, with **9 open**.
The backend service observation batch commits as 21c2372.

BackendEndpointGuard now refreshes backend service observation and checks a selected private endpoint in the relay VNet.
It binds consumer approval, complete reported service approval, NIC backlinks, exact private address, endpoint subnet membership, and selected security rules.
The service collection and embedded environment connections must agree. The NIC member must match the service's reported required member.
The endpoint subnet explicitly enables network policies, disables default outbound access, and permits no unrelated endpoint, allocation, route, or NAT attachment.
Its dedicated NSG permits only relay-subnet TCP 443 to the selected backend address before terminal inbound and outbound denies.
The relay subnet's actual prefix and Container Apps delegation must match the selected source scope.

The stage reuses the preceding call's validated environment observation instead of issuing a second environment GET.
Existing worker and new backend endpoint inspection now share scoped SDK access, endpoint, NIC, subnet, security, and reserved-address verification.
The new reader permits only five selected network resources and two backend service collections. It exposes no write or secret operation.
A normal composed check makes fifty-seven reads within a fifty-second context. Complete bounded paging permits at most 110 reads.

Four new focused local race checks pass. Four selected existing endpoint and backend checks also pass after extraction.
Checks cover fresh state, environment reuse, foreign services, member and address mismatch, policy bypass, pending approval, incomplete pages, immutable definitions, and read boundaries.
Only the changed definition check repeats after review strengthens its exact destination assertion.
Fixtures use explicit fake credentials and in-memory SDK transports. They open no network connection.
Evidence is output/azure-worker-backend-endpoint/summary.json and targeted command results.
No live cloud API, cloud credential, Azure CLI, UI build, image build, browser, container, service listener, registry push, or deployment runs.
No unchanged broad suite or fixture build repeats.

This is endpoint observation and intended policy. It proves neither actual packet flow nor complete relay networking.
The stage implements no Network and enables no coded Azure execution.
Effective read coverage, workload attachment authority, actual relay VNet and perimeter, private DNS, TLS trust, identity grants, backend infrastructure subnet policy, and execution isolation remain required.
The operator guide records the endpoint boundary and the remaining routing and trust work.
Continue effective relay routing, DNS and normal backend TLS verification before native service wiring.
See [Azure private backend endpoint observation](azure-worker-backend-endpoint.md).

## Previous delivery: selected Azure backend observation

Acceptance **51** advances. The verified total remains **48 of 57**, with **9 open**.
The staged backend package batch commits as 7825be6.

BackendGuard now refreshes the preceding exchange and subscription inventory stage and binds a selected trusted backend.
It compares actual app, environment, HTTPS origin, full known writable worker configuration, secret references, artifact settings, identity metadata, and active revision.
The app must use the worker-only command and its dedicated internal environment in the intended trusted subnet.
Only its runtime identity reaches the main container. Its observed runtime client identifier must match artifact settings.
Secret values, API mode, target bindings, changed storage, different ingress, and inconsistent revisions reject observation.

The backend reuses the preceding call's transient placement snapshot to reject other reported backend apps or jobs.
The selected backend must itself appear in that snapshot. No second subscription scan, persisted inventory, or successful cache is added.
This is reported placement, not proof of unrestricted read coverage or future attachment control.

The backend and exchange now share service SDK reads, application normalization, and revision comparison.
The shared private environment check explicitly requires internal networking.
Missing or null route and revision arrays remain unavailable. The exchange fixture now reports an explicit empty route array.
Artifact JSON comparison preserves meaning across formatting changes while rejecting duplicate keys and bounded-structure violations.
The new backend reader permits only five selected GET resources or collections. It has no secret, certificate, mutation, or unrelated-resource route.
A normal composed check makes fifty reads within a forty-five-second context. Complete paging remains bounded at 101 reads.

Four new focused local race checks pass. Four selected existing exchange and inventory checks also pass after shared extraction.
Only cases affected by snapshot reuse and stronger backend occupant coverage repeat. No unchanged broad suite or fixture build repeats.
Fixtures use explicit fake credentials and in-memory SDK transports. They open no network connection.
Evidence is output/azure-worker-backend-admission/summary.json and targeted command results.
No live cloud API, cloud credential, Azure CLI, UI build, image build, browser, container, service listener, registry push, or deployment runs.

This stage does not implement Network, authorize dispatch, or enable coded Azure jobs.
Effective read coverage, workload attachment authority, actual backend routing and TLS, identity grants, endpoint or subnet controls, DNS visibility, trusted preparation access, and execution isolation remain required.
The package and observer guides now distinguish intended infrastructure, reported state, and unfinished native admission.
Continue the actual private relay-to-backend path and remaining native authority checks.
See [Azure trusted backend observation](azure-worker-backend-admission.md).

## Previous delivery: staged Azure trusted backend package

Acceptance **51** and supporting packaging under **57** advance. The verified total remains **48 of 57**, with **9 open**.
The direct TLS backend batch commits as 38a5d7a.

The new deploy/azure/worker-backend.bicep package defines a dedicated internal Container Apps environment with public network access disabled.
Its only app overrides the service image entrypoint and runs the exact worker-only command on private port 8042.
The app uses platform HTTPS termination with the environment default domain. Platform user authentication is explicitly disabled.
It starts no UI, sign-in, source preparation, scheduler, notification worker, test executor, or coded job.
The existing core deployment remains unchanged and does not invoke this separate package.

Separate pull, startup, and runtime identities reuse the existing registry, secret, and Blob grant modules.
Pull and startup identities have lifecycle None. The main container selects the exact artifact runtime client identifier.
The startup grant reaches only the named database secret. The artifact grant reaches only the selected existing container.
Only a database secret reference and Azure artifact configuration enter the container. No target, OIDC, registry-write, or AWS credential is added.
Outputs bind intended backend origin, application identifier, and environment identifier. They contain no secret values.

A local Bicep 0.48.1 compilation passes schema validation without warnings or registry restoration.
Six focused package checks pass for private environment intent, HTTPS listener, exact command, identity separation, grant scopes, and probes or outputs.
A checker review strengthens the dependency assertions. It reuses the compiled template rather than recompiling or repeating Go fixtures.
Evidence is output/azure-worker-backend/backend.json, compile.log, and summary.json.
No live Azure API, account, cloud credential, Azure CLI, image build, UI build, browser, registry push, or deployment runs.
No owned browser, container, service process, or live test listener starts in this batch.

This is staged backend infrastructure, not native admission or coded worker enablement.
Actual private relay routing, endpoint approval, DNS, certificate trust, resource observation, exclusive occupancy, attachment authority, effective identities, and trusted storage paths remain required.
The intended package does not establish deployed isolation or authorize an untrusted test job.
The operator guide records those boundaries and leaves all nine broad acceptance items open.
Continue resource binding and private relay-to-backend admission, effective read coverage, environment attachment authority, and exchange networking.
See [staged Azure trusted backend](azure-worker-backend.md).

## Previous delivery: direct HTTPS worker backend

Acceptance **51** advances. The verified total remains **48 of 57**, with **9 open**.
The dedicated backend role batch commits as 105acdf.

The worker role now supports optional direct TLS with an exact HTTPS origin and mounted certificate chain and private key.
The backend validates leaf hostname, lifetime, server usage, and key matching before opening storage.
Incomplete or invalid material returns fixed errors without file names, private values, or certificate diagnostics.
The HTTPS origin parser is the existing relay validator, now exported for reuse. Relay production dependencies do not change.

The configured listener requires TLS 1.2 or later. Wrong SNI rejects the handshake, even when the certificate also covers that other name.
Wrong HTTP Host rejects before probe or worker routing. Plain HTTP reaches no worker operation.
The shared listener lifecycle supports already-configured TLS servers while preserving existing HTTP behavior and shutdown.
API and artifact roles cannot configure nonempty direct worker TLS settings. The API's optional worker listener retains external termination.
The relay still verifies issuer and chain trust normally. Leaf startup checks do not establish that external trust or network placement.

Two new focused local race checks pass, with invalid-material table cases and a complete relay-to-backend command flow.
The real command persists input claims, start observations, validated reports, and checked attachment uploads into the existing private stores.
Live account disablement rejects the next write through the relay.
Foreign Host, a foreign but certificate-covered SNI, untrusted roots, legacy TLS, and plaintext HTTP reject locally.
Three existing command startup and listener lifecycle checks pass after the TLS lifecycle change.
Only changed certificate and end-to-end checks repeat after review strengthens purpose, SNI, and plaintext coverage. No unchanged broad suite repeats.
Owned subprocesses, HTTP/TLS listeners, and idle transports close. Port checks confirm backend shutdown.
Evidence is output/worker-backend-tls/summary.json and targeted command results.
No browser, UI build, image build, cloud request, cloud credential, registry push, or deployment runs.

The operator guide documents read-only certificate mounts, normal relay trust, exact health authority, restart-based rotation, and image entrypoint overrides.
The service and relay image caches predate these code changes. Rebuild them only in the next relevant packaging batch.
Native read coverage, attachment authority, effective networking, private backend routing and placement, and full provider admission remain required.
Direct TLS enables no Azure coded job and closes no broad hosted acceptance item.
See [trusted backend direct TLS](worker-backend.md#direct-tls).

## Previous delivery: dedicated trusted worker backend

Acceptance **51** advances. The verified total remains **48 of 57**, with **9 open**.
The workload inventory batch commits as 903bf11.

The new --serve-role=worker starts one private worker listener with readiness and liveness probes.
It exposes no UI, sign-in, user API, artifact viewer, source preparation, schedule, notification, or execution worker.
An explicit option allowlist rejects unrelated configuration before storage opens. The listener address is required.
The role reuses the installation database, private artifact adapter, and normal permission service.
It loads no target bindings, HTTP trust bindings, OIDC providers, hosted dispatch settings, or local executors.

Execution requests retain fresh run, lease, current attempt, permission, project, suite, capture, environment, and catalog checks.
Discovery requests retain the location-specific lease registry, exact reference, permission, cancellation, and project checks.
Hosted dispatch and the new backend share the same authorization wiring. Source cleanup authority remains controller-side.
Shared storage initialization preserves existing PostgreSQL and SQLite adapter selection and database error redaction.
No user bootstrap or password-mode update runs in this role.

Six new focused local race checks pass. They cover role selection, explicit option rejection, execution permission loss, discovery activation and location, readiness, and command startup.
Three existing artifact-role and listener lifecycle checks also pass after shared storage extraction.
The startup check executes the real command branch in an owned subprocess and confirms standard health responses and stored sign-in preservation.
Its initial subprocess cleanup harness inherited the test context and killed the child before graceful cleanup.
The corrected harness owns a separate bounded lifecycle. Only the changed startup check repeats and passes.
Owned HTTP listeners and the subprocess stop. The final port check rejects new connections.
Evidence is output/worker-backend/summary.json and targeted command results.
No browser, UI build, image build, cloud credential, cloud request, registry push, or deployment runs.

Operator guidance covers exact role flags, shared storage, health overrides, and the required image entrypoint override.
The current service image cache predates this change. Rebuild it in the next relevant packaging batch rather than claiming it already contains the role.
Private HTTPS termination, restricted relay-to-backend routes, backend placement, effective network controls, and native authority remain required.
This role does not implement Azure Network or enable coded jobs.
Continue native coverage and attachment authority, effective networking, and trusted backend admission before provider wiring.
See [trusted worker backend](worker-backend.md).

## Previous delivery: bounded Azure exchange workload inventory

Acceptance **51** advances. The verified total remains **48 of 57**, with **9 open**.
The selected exchange binding batch commits as 716d412.

WorkloadInventoryGuard now refreshes the selected exchange stage before concurrent application and job subscription inventories.
The selected relay must appear in its expected environment. Any other observed app or job in that environment rejects inspection.
The check includes resources in other groups and stopped applications. It does not limit discovery to the relay's resource group.
Duplicate identities, conflicting environment fields, wrong kinds, foreign-subscription records, and contradictory placement reject observation.
Missing properties, absent placement, unknown references, missing selected relay, and uncertain list data remain unavailable.

The SDK permits only two configured subscription collection GET paths through the existing bounded ARM transport.
It reduces each page to resource and environment identifiers without retaining unrelated application or job configuration.
Each collection allows 512 items and sixteen pages. Missing or null value arrays, nil records, excess items, and unfinished paging cannot establish absence.
Foreign pages, other collections, token cycles, altered queries, secret actions, and mutations cannot widen read authority.
A failed subscription read triggers no smaller-scope fallback.

The shared HTTP guard now handles registered subscription paths without assuming a resource-group identifier exists.
Existing escaped-group and DNS association behavior remains verified through two selected checks.
Six new focused local race checks pass, including exact item bounds, cross-group occupants, legacy connected references, Unicode groups, and metadata reduction.
A review fix aligns duplicate keys with Unicode identity comparison. Only the two affected checks repeat.
Evidence is output/azure-worker-workloads/summary.json and the targeted command results.
No cloud credential, live cloud API, DNS probe, packet probe, TLS probe, deployment, registry push, UI build, browser, container, or image build runs.
No unchanged broad suite or runner fixture repeats.

This is reported inventory observation. A 200 response does not prove effective read coverage or prevent later workload attachments.
Native setup must still verify subscription read coverage, relevant access restrictions, and trusted environment join and write authority.
The guard does not implement Network or enable coded execution.
Actual exchange networking, restricted backend authority, effective DNS visibility, trusted preparation access, cross-job isolation, execution placement, and role binding remain required.
Continue native authority and effective network controls before complete admission and service wiring.
See [Azure exchange workload inventory](azure-worker-workloads.md).

## Previous delivery: selected Azure exchange and HTTPS binding

Acceptance **51** advances. The verified total remains **48 of 57**, with **9 open**.
The relay packaging batch commits as 707f54f.

ExchangeGuard now refreshes the preceding registry stage before eight selected environment, app, certificate, auth, approval, link, route, and revision observations.
The environment must be ready with public access disabled, the selected infrastructure subnet and group, and no environment runtime identity or telemetry credentials.
Service-side approval must match the worker exchange endpoint and its embedded observation.
The selected app must retain the immutable relay image, fixed command and origin arguments, one container, no runtime credentials, and lifecycle None for image pull.
The image repository stays outside the untrusted test source prefix.
Fresh SDK application and disabled-auth definitions support later packaging without Azure writes.

HTTPS ingress must bind only the configured worker hostname and selected uploaded certificate.
Certificate metadata must match the hostname and cover the maximum exchange deadline with an admission margin.
The selected revision list must contain one healthy latest active revision with the same template and all traffic.
Inactive revisions report no replicas or traffic. Environment HTTP-route inventory must be empty, and Easy Auth must be disabled.
Foreign identity, different origins, credentials, alternate commands, ports, routes, certificates, and approval reject inspection.
Missing or inconsistent snapshots, unsettled revisions, pending approval, and near-expiry certificate metadata remain unavailable.

The SDK permits only eight exact selected GET resources or collections through the existing bounded ARM transport.
It cannot request secrets, certificate export, app auth tokens, subscription discovery, resource-group discovery, writes, filters, or foreign resources.
Six new focused local race checks pass across targeted commands. One existing relay configuration check passes after pure origin-validator extraction.
The production dependency audit confirms no new storage, identity, secret, or cloud-SDK dependency in the relay.
Only changed checks repeat after definition readability and missing or duplicate revision and domain observation review fixes.
Evidence is output/azure-worker-exchange/summary.json and the focused command results.
No cloud credential, live cloud API, TLS probe, DNS probe, packet probe, registry push, deployment, UI build, browser, container, or image build runs.

This verifies the selected binding. It does not prove exclusive environment occupancy, attachment authority, actual network isolation, backend authority, or a served TLS chain.
Container Apps inventory scopes require separate complete workload observation and attachment controls.
Complete those controls, effective DNS visibility, trusted preparation networking, cross-job isolation, execution placement, role bindings, and native service wiring.
Azure coded execution stays disabled. The guard does not implement Network or release worker input.
See [selected Azure worker exchange binding](azure-worker-exchange.md).

## Previous delivery: minimal worker relay image packaging

Acceptance **51** and supporting packaging under **57** advance. The verified count remains **48 of 57**, with **9 open**.
The standalone relay batch commits as 763daf4.

The new builder compiles only teststudio-exchange and teststudio-health for an explicit Linux architecture.
It uses disabled CGO, readonly module inputs, trimmed paths, disabled VCS stamping, and recorded target settings.
It reuses the existing immutable certificate source and copies only its public CA bundle.
No UI assets, preparation executable, API executable, database directory, installation credential, or runner image enters the context.
Local-only mode requires a local Docker socket and refuses to pull a missing certificate-source image.
Image builds use no build network or implicit pull. No registry push occurs.

Source and compiler fingerprints preserve unchanged binary reuse. File hashes verify those cached artifacts.
Image identity binds actual binary and certificate bytes, architecture, Dockerfile, build tooling, and source policy.
A warm invocation reuses both binaries, the certificate bundle, and the exact immutable image without rebuilding.
Only Docker's fixed default PATH can appear in the image environment.
Public CA permissions are corrected so the non-root process can read the bundle.

The AMD64 relay package passes seven focused local checks on the ARM64 Docker host.
It starts as user 65532 with read-only root, no capabilities, no network, and no writable mounts.
In-image process health and liveness pass. Exported artifact digests and non-root file permissions match.
Both ELF64 executables match AMD64 and require no dynamic interpreter.
The exported filesystem contains only required artifacts and known Docker runtime files, including empty device placeholders.
Graceful shutdown exits successfully. Exact extraction and verification container absence is confirmed.

Initial checker assumptions reject Docker's default PATH and its empty console placeholder. Those expectations are corrected before final verification.
The certificate readability issue is fixed in the prepared context and covered by the final package check.
No unchanged relay Go check, broader Go suite, UI build, runner fixture, cloud credential, live cloud API, registry push, or deployment runs.
Evidence is output/worker-relay-image/build.json, cache-reuse.json, and runtime/summary.json.

This verifies the local AMD64 package. It does not prove backend TLS reachability or hosted service authority.
Continue Azure exchange environment approval, isolated application inventory, immutable relay configuration, and certificate/hostname binding.
Restricted backend placement, effective DNS visibility, trusted preparation networking, cross-job isolation, execution placement, and role bindings remain required.
Azure coded execution stays disabled. See [worker relay packaging](worker-relay.md#packaging-boundary).

## Previous delivery: standalone private worker relay

Acceptance **51** advances. The verified total remains **48 of 57**, with **9 open**.
The preceding registry service batch commits as 71b7801.
Inspection finds that the current worker listener runs inside the API process. A separate relay supplies the missing worker-facing process first.

The new teststudio-exchange command forwards only input, start, report, and attachment operations to one configured HTTPS backend.
It carries only the scoped worker ticket and instance identifier. Browser sessions, project headers, forwarding headers, and arbitrary caller authority do not pass.
Host, path, method, token shape, identifier, body size, response size, and eight-request concurrency bounds reject wider requests.
The controller retains actual authentication, authorization, activation, durable state, upload validation, and evidence storage.
The relay has no database, artifact-store, registry, identity-service, or cloud credential configuration.

The new lightweight workerwire package holds the shared packet, errors, and size constants.
Existing workerprotocol and execution symbols retain aliases. The identifier grammar moves unchanged into a small pure package.
The production command dependency audit now shows standard-library packages and only the four small relay and wire packages.
It links no storage driver, identity service, secret provider, or cloud SDK.

Seven new focused race checks pass, including the existing worker client over two local TLS listeners, exact operation forwarding, upload bounds, and shutdown.
The initial health fixture incorrectly uses an absolute request target. Correcting it to the real listener's relative target resolves that fixture failure.
After dependency extraction, the complete changed relay and command checks pass together.
Two existing durable input and report checks pass. No unchanged broad runner or UI suite repeats.
All owned local test listeners close. The command check verifies that its closed port rejects connections.
Evidence is output/worker-relay/summary.json and the focused command results.

A separate non-root scratch-image Dockerfile is staged. No image, container, registry push, deployment, browser, or UI build runs.
The local demonstration and data remain unchanged. No live cloud credential or cloud API is used.
The relay's process health does not establish backend readiness or Azure service admission.

Complete reproducible relay image packaging, private TLS ingress, restricted worker backend placement, and Azure environment and application authority checks.
Effective DNS visibility, trusted preparation networking, cross-job isolation, execution placement, role binding, and native configuration remain required.
Azure coded execution stays disabled. See [standalone private worker relay](worker-relay.md).

## Previous delivery: Azure registry service configuration and approval

Acceptance **51** advances. The verified total remains **48 of 57**, with **9 open**.
The preceding endpoint ownership batch commits as 24a42ca.

RegistryGuard now refreshes endpoint ownership before inspecting actual registry configuration and three complete service inventories.
The registry must be private, Premium, ready, and bound to the configured login and selected regional data hostname.
Admin passwords, anonymous pull, public access, task bypass, trusted-service bypass, IP exceptions, and artifact exports cannot widen access.
The observed legacy permission mode matches the current AcrPull packaging profile. ABAC role packaging remains separate work.
ARM-audience authentication stays enabled because the platform's managed-identity image pulls require it.
The trusted registry client keeps its separate registry-scoped audience without ARM fallback.

The required registry group, login member, selected regional data member, private zone, and complete replication inventory must agree.
Service-side connection approval must cover exactly the worker registry endpoint and a separate selected preparation endpoint.
The registry's embedded inventory must agree with the complete collection.
Foreign or duplicate connections, including identities differing only by case, reject inspection.
Missing policy status, pending approval, unfinished inventory, and different snapshots remain unavailable.
Trusted preparation NIC, subnet, DNS, and process placement remain separate required checks.

The production reader pins the stable armcontainerregistry/v3 3.0.0 SDK and API 2025-11-01.
Its transport permits only registry and selected child-collection GET requests.
Credential listing, regeneration, writes, filters, foreign scopes, redirects, and unsafe paging cannot widen authority.
Six new focused local race checks pass across targeted commands. Only two changed checks repeat after review fixes.
Evidence is output/azure-worker-registry/summary.json and the focused command results.
No live cloud credential, cloud API, DNS probe, packet probe, registry push, deployment, UI build, browser, container, or image build runs.

Registry service checks do not implement Network or enable coded Azure execution.
Exchange service authority, effective DNS visibility, trusted preparation networking, cross-job isolation, execution placement, and role binding remain unfinished.
Continue exchange environment, service approval, application inventory, and HTTPS authority inspection.
See [Azure worker registry service admission](azure-worker-registry.md).

## Previous delivery: Azure private endpoint and NIC ownership

Acceptance **51** advances. The verified total remains **48 of 57**, with **9 open**.
The previous selected-zone batch commits as 74b9184. This batch binds actual private endpoint resources to their DNS addresses.

EndpointGuard refreshes the complete preceding inspection chain before six concurrent endpoint, NIC, subnet, and NSG reads.
The registry endpoint must target the configured registry with group registry.
Its NIC binds distinct login and selected regional data members to their exact DNS addresses.
The exchange endpoint must target a separate managed environment with group managedEnvironment and one exact exchange address.
Consumer-side approval must be complete with no outstanding action. Service-side approval remains separate work.

The endpoint subnet is separate from worker and firewall subnets and excludes Azure-reserved addresses.
It requires enabled private endpoint policies, exact endpoint and NIC references, and a dedicated restrictive NSG.
The NSG permits only worker HTTPS to the selected addresses before explicit inbound and outbound denies.
Public attachments, foreign services, swapped addresses, alternate regions, forwarding, unrelated workloads, and permissive rules reject inspection.
Missing resources, unfinished approval, and incomplete inventories remain unavailable.
Fresh validated subnet and NSG definitions issue no Azure writes.

The SDK reader permits only six exact GET resources through the existing bounded ARM transport.
Wrong resource kinds, mutations, collections, foreign services, extra queries, redirects, and paging cannot widen read authority.
Seven new focused race checks pass across three targeted commands. Two selected existing perimeter checks pass after shared comparator extraction.
Evidence is output/azure-worker-endpoints/summary.json and the focused command results.
No cloud credential, cloud API, DNS probe, packet probe, registry push, deployment, UI build, browser, container, or image build runs.

Actual registry configuration and service-side approval, exchange authority, effective DNS visibility, attachment controls, and cross-job isolation remain unfinished.
Endpoint inspection does not implement Network or enable Azure coded execution.
Continue service-side registry and exchange inspection, then complete the remaining network boundary and native packaging.
See [Azure worker private endpoint ownership](azure-worker-endpoints.md).

## Previous delivery: Azure selected private zones, records, and links

Acceptance **51** advances. The verified total remains **48 of 57**, with **9 open**.
The previous resolver-policy batch commits as 5152995. This batch verifies the selected private DNS record and link inventories.

PrivateDNSGuard refreshes all previous network stages before reading the exchange and registry private zones, complete record sets, and links.
The exchange zone must equal the complete configured hostname. A parent zone cannot substitute for that authority.
Exact exchange, registry login, and regional data A records bind three distinct role addresses to the complete perimeter allowlist.
The only other permitted record is each zone's SOA. Aliases, wildcards, IPv6, extra addresses, mixed data, and foreign child identity reject inspection.
Positive and effective negative caching stay bounded to 300 seconds.
Worker links require completed state, explicit disabled registration, and no public fallback.
Global zone metadata counts must agree with the complete observed record and link inventories.
Missing records, unfinished links, unknown counts, and incomplete paging remain unavailable.

Fresh validated SDK definitions support later packaging without making Azure writes.
The production reader pins the stable private DNS SDK and API and permits only the two zones and their ALL-record and link collections.
A-only or name-filtered views cannot hide other record types. Foreign records, mutation verbs, POST exceptions, unsafe paging, and excess inventory remain rejected.
Six new focused local race checks pass. Two selected existing DNS and firewall paging checks pass.
Evidence is output/azure-worker-private-dns/summary.json and the focused command results.
No Azure credential, cloud API, DNS probe, packet probe, registry push, deployment, UI build, browser, container, or image build runs.

This is selected-zone inspection. It does not prove the absence of other linked zones that could alter the VNet's effective DNS view.
Effective-zone visibility and attachment authority remain required before complete admission.
Private endpoint identity, approval, subnet, NIC address ownership, service identity, registry data region, and isolated exchange service remain required.
Trusted preparation needs its own private registry endpoint and DNS path. Do not share worker links or peer workers into trusted networking.
Azure coded execution stays disabled. No complete Network interface or broad acceptance item closes from these local fixtures.
Continue endpoint ownership and effective-view controls before native enablement or infrastructure packaging.
See [Azure worker private DNS records](azure-worker-private-dns.md).

## Previous delivery: Azure DNS resolver policy and actual VNet association

Acceptance **51** advances. The verified total stays **48 of 57**, with **9 open**.
The previous firewall batch commits as 7d212a8. This batch implements actual resolver-policy and VNet association inspection.

DNSGuard refreshes all previous network stages before reading the selected policy, two domain lists, complete rules and links, and actual VNet association.
Exactly one enabled approved-list allow and one enabled root-domain block are required, with fixed priority and list authority.
The approved list binds platform names, exchange origin, registry login, explicit registry data region, and private-link names.
An explicit data region supports a registry and worker in different Azure regions without inferring endpoint location.
Extra domains, bulk references, disabled rules, changed actions, foreign lists, extra rules, and substituted resource identity reject inspection.
The policy links only the worker VNet. The VNet's own association inventory must identify the same policy.
Empty, missing, incomplete, or unfinished observations remain unavailable. Conflicting or foreign authority never establishes admission.

The production client pins the stable DNS SDK and API and preserves explicit credential, origin, resource, version, body, context, and query bounds.
Its one POST exception invokes the exact documented read-only worker VNet association action with no request body.
Other readers keep their existing methods and resources. A shared bounded inventory helper now serves DNS and firewall paging.
Fresh validated definitions support later packaging without making Azure writes.
Native hierarchy remains explicit: approved parents can permit descendants. Private zones and exact endpoint records remain required.

Seven new focused local race checks pass, including a nineteen-case declared DNS decision matrix.
Three selected existing firewall, perimeter, and topology checks pass for shared transport and paging compatibility.
Only the affected settings, registry-region declarations, association, and SDK checks repeat after making data region explicit.
Evidence is output/azure-worker-dns/summary.json and the focused command results.
No Azure credential, cloud API, DNS probe, packet probe, registry push, deployment, UI build, browser, container, or image build runs.

This stage does not implement complete Network admission or release worker input.
Private zone authority, private DNS records, endpoint ownership and region, isolated exchange, workload inventory, cross-job isolation, and execution placement remain required.
Live recursive resolution, platform CNAME behavior, identity compatibility, effective isolation, and deployment parity still need separate verification.
Azure coded execution stays disabled. No broad acceptance item closes from these local response and declaration fixtures.
Continue private DNS and endpoint verification before composing full Network admission or enabling native coded jobs.
See [Azure worker DNS policy inspection](azure-worker-dns.md).

## Previous delivery: Azure firewall resource and effective policy inspection

Acceptance **51** advances. The total remains **48 verified of 57**, with **9 open**.
The previous perimeter batch commits as f9be761. This batch verifies the actual firewall behind the selected default route.

FirewallGuard refreshes topology and perimeter before checking the firewall, subnet, public IP association, policy, and complete group inventory.
It binds the reported routed private IP to one Standard VNet firewall configuration and the selected subnet, public IP, and policy.
Classic rule collections, management configurations, hubs, substituted resources, unsafe subnet bindings, and alternate public-IP arrangements reject inspection.
The dedicated policy rejects inherited or child policy, shared firewall attachment, proxies, custom DNS, and unrelated capabilities.

One owned group contains a fixed worker HTTPS platform allowlist followed by an explicit application deny.
That final deny overrides Azure Firewall's built-in infrastructure fallback for this profile's application protocols.
Only empty documented default groups are permitted. Changed actions, rules, groups, destinations, sources, protocols, URLs, tags, and NAT reject inspection.
Fresh validated SDK definitions support later packaging without issuing Azure writes.

The production reader permits only the exact firewall resources and selected policy's bounded group listing.
The shared transport permits opaque pagination only on that collection. Other readers retain their existing resource and query boundaries.
Complete policy references and listed group inventory must agree. Missing groups and unfinished paging retain uncertainty.
Contexts, body reads, four-group inventory, four-page bounds, and token-cycle checks remain enforced.
Four independent attachment reads run concurrently after perimeter inspection. The normal full check makes thirteen bounded reads with no successful-admission cache.

Eight new focused local race checks pass. Two selected existing transport boundary checks pass.
Only the affected uncertainty check repeats after adding hidden-group, missing-group, and endless-paging handling.
Evidence is output/azure-worker-firewall/summary.json and the focused command results.
No Azure credential, cloud API, packet probe, registry push, deployment, UI build, browser, container, or image build runs.

This stage does not implement complete Network admission or enable coded execution.
Actual DNS filtering and private records, endpoint ownership, exchange isolation, workload inventory, cross-job isolation, and execution placement remain required.
Azure startup compatibility, identity behavior, effective packet isolation, and deployment parity still need separate hosted verification.
No broad acceptance item closes from these local SDK and declaration fixtures.
Continue DNS policy and endpoint inspection before native enablement or infrastructure packaging.
See [Azure worker firewall inspection](azure-worker-firewall.md).

## Previous delivery: Azure worker NSG and routing inspection

Acceptance **51** advances. The total stays **48 verified of 57**, with **9 open**.
The previous topology batch commits as f3f7634. This batch adds the actual security-rule and routing inspection stage.

PerimeterGuard performs fresh topology inspection before reading the selected NSG and route table.
It requires the exact thirteen-rule worker policy, explicit deny defaults, selected private endpoints, and bounded platform dependencies.
The NSG enables connection reevaluation and attaches only to the worker subnet.
Changed directions, priorities, protocols, sources, ports, actions, addresses, application groups, child ownership, and rule counts reject inspection.
Descriptions and documented metadata remain separate from traffic authority. Rule and endpoint order do not create false drift.

The route table requires one default route through the selected private firewall address and explicitly disabled BGP propagation.
Extra routes, service-tag prefixes, Internet next hops, foreign appliances, shared attachments, and BGP overrides reject inspection.
Pending rules, routes, resources, topology, or reads remain unavailable.
Immutable validated settings also produce fresh SDK definitions for later infrastructure packaging.
Those definitions issue no Azure write request.

The production SDK reader permits GET on only the exact NSG and route table through the pinned network SDK and version.
Topology and perimeter readers now share one bounded transport implementation while retaining separate resource allowlists.
Eight new focused race checks pass, including an independent 24-case declared traffic matrix.
Two selected existing topology SDK boundary checks pass after the transport refactor.
Only the changed endpoint normalization and rule rejection checks repeat with the new matrix.
Evidence is output/azure-worker-perimeter/summary.json and the focused command results.
No Azure credential, cloud API, packet probe, registry push, deployment, UI build, browser, container, or image build runs.

This stage does not implement Network, release worker input, or establish live platform compatibility.
Actual firewall ownership and effective policy, DNS filtering, private endpoint identity, exchange isolation, and cross-job isolation remain required.
Shared platform service tags and within-subnet communication cannot prove complete application isolation.
Verify metadata-deny compatibility with platform-only image pulls in the separate hosted verification phase.
Azure coded execution stays disabled. No acceptance item closes from these local declaration and response fixtures.
Continue effective firewall policy, DNS, and endpoint inspection before native enablement or deployment packaging.
See [Azure worker perimeter](azure-worker-perimeter.md).

## Previous delivery: Azure environment and network topology inspection

Acceptance **51** advances. The total remains **48 verified of 57**, with **9 open**.
The previous independent deadline batch commits as c3dd6da. This batch implements the actual environment and subnet inspection stage.

TopologyGuard checks the configured worker and controller environments, VNets, and infrastructure subnets through fresh SDK reads.
It requires a private workload-profile worker environment with one Consumption profile and separate controller networking.
It rejects shared infrastructure, overlapping or nonprivate prefixes, public ingress, extra profiles, telemetry credentials, and unexpected identity.
The worker VNet rejects peering, NAT bypasses, custom DNS, BGP community settings, and external IP allocations.
The worker subnet requires explicit private outbound configuration, exact NSG and route-table bindings, and the Container Apps delegation.
Service endpoints, subnet sharing, foreign attachments, conflicting subnet observations, and other configured bypasses reject inspection.
Missing addresses, unready provisioning, failed reads, and cancelled operations remain unavailable.

The production reader permits only GET on the six selected resources through pinned official SDKs and management versions.
It fixes endpoint, subscription, resource paths, and bounded contexts and response sizes.
The ARM client options now share the existing no-retry and no-provider-registration configuration across job, timer, and topology clients.
Seven new focused local race checks pass. Two selected existing SDK request checks pass.
A new fixture initially uses the wrong reader type; its corrected targeted invocation passes.
Only affected checks repeat after private-prefix and boundary updates.
Evidence is output/azure-worker-topology/summary.json and the focused command results.
No Azure credential, cloud operation, registry push, deployment, UI build, browser, container, or image build runs.

This is a required inspection stage. It does not implement Network or authorize execution by itself.
Full NSG rules, routing, DNS policy, private endpoints, exchange isolation, unrelated workload inventory, and cross-job isolation remain required.
The complete adapter must bind all policy settings into networkProfile. A topology-only digest cannot establish the final execution identity.
Azure coded execution stays disabled. No broad acceptance item closes from this local stage.
Continue the perimeter checks before native enablement, infrastructure packaging, or separate deployment verification.
See [Azure worker network admission](azure-worker-network.md).

## Previous delivery: independent Azure deadline resources

Acceptance **51** and supporting recovery under **52** advance. The total stays **48 verified of 57**, with **9 open**.
The previous ACR authentication batch commits as 6e943de. This batch implements the concrete deadline hook.

SDKDeadlines provisions a separate Logic Apps Consumption workflow for each owned job.
A future recurrence uses the frozen absolute deadline and repeats each minute to catch late execution visibility.
One fixed HTTP action requests stop on that exact job with a distinct managed identity.
The workflow holds no ticket, source credential, application secret, or executable input.
Accepted stop requests remain separate from observed termination. Azure scheduling and shutdown latency remain unverified.

The existing dispatch journal now reserves deadline creation and deletion before each mutation.
It binds the configuration profile and complete saved specification. Changed identity or proof cannot adopt an older timer.
Unknown creation responses reconcile through GET without another PUT. Preexisting resources are rejected.
Enabled state, succeeded provisioning, selected identity, exact schedule, and complete writable workflow content gate admission.
Unready, changed, foreign, or missing timers do not release work.

Cleanup requires revoked work, a durable result and evidence checkpoint, and independently confirmed job absence.
An unseen uncertain creation remains unresolved until its owned resource appears or an operator investigates.
Removal fences creation, reserves a thirty-second deletion backoff, and confirms workflow absence before completion.
Concurrent controllers retain one creation or deletion reservation. Removed timers stay fenced across database restart.
The production SDK pins the management endpoint, namespace, version, five-second operation contexts, and one-megabyte reads.
It disables automatic retries and resource-provider registration. Redirects, foreign operations, and provider diagnostics remain bounded.

Eleven new focused race checks pass across the targeted commands.
They cover both runners, execution and discovery, SQLite reopen, lost responses, concurrency, immutable configuration, SDK stop shape, and cleanup.
Two selected existing provider checks pass for journal compatibility and controller concurrency.
Initial fixture checks expose clock and resource-group encoding expectations. Only affected checks repeat after correction.
Evidence is output/azure-worker-deadlines/summary.json and the focused command results.
No UI build, browser, container, image build, source-runtime matrix, Azure credential, cloud operation, registry push, or deployment runs.
The demo and its data remain unchanged.

Azure coded execution stays disabled. Actual network admission, role binding, private exchange packaging, and native setup remain required.
Timer quotas, orphan monitoring, service delivery latency, and hosted identity authority need separate deployment verification.
Continue network admission before enabling native coded jobs. Keep acceptances 41, 46, and 51–57 open.
See [independent Azure deadlines](azure-worker-deadlines.md).

## Previous delivery: trusted ACR authentication and scoped source publication

Acceptance **51** advances. The count remains **48 verified of 57**, with **9 open**.
The previous durable Azure lifecycle batch commits as 10cd5f9. This batch implements its trusted registry authentication gap.

The new ACR client exchanges an explicit Azure credential through the registry-specific Entra scope and selected tenant.
It obtains registry refresh and exact repository tokens through bounded form POST requests to the configured registry.
No registry administrator password, ARM token fallback, or implicit credential discovery is used.
The existing OCI challenge parser checks exact realm, service, repository, and actions before invoking the optional token callback.
Basic and Bearer credential callbacks retain their existing behavior. Token callbacks cannot combine with the Credentials option.

Inspection permits pull only and rejects write requests. Separate preparation clients can request pull,push and publish through the normal OCI source flow.
The transport permits immutable digest operations and bounded upload-session actions within the configured namespace.
Registry redirects do not forward credentials. Approved Azure configuration-blob origins receive fresh credential-free GET requests.
The existing size, digest, non-root, runtime, ownership, source, approval, and upload-journal checks remain in place.

Caches remain in memory and are bounded by source identity, source expiry, registry token expiry, a 30-second margin, five-minute age, and 32 entries.
Concurrent inspections reuse grants. A changed or failed Azure credential clears older tokens instead of falling back.
Malformed, duplicate-key, expired, not-yet-valid, oversized, and foreign authentication responses fail with bounded private diagnostics.
Close seals new requests, clears tokens, and closes idle connections.

Five new focused race checks pass with in-memory OAuth and OCI responses for both Go and Playwright.
They verify inspection, source publication, exact scopes, completed upload journals, rotation, concurrent reuse, bounded caches, and credential-free blob redirects.
Four selected existing OCI namespace, digest, image, and redirect checks pass with race detection.
Evidence is output/azure-registry-auth/summary.json and the focused command results.
No UI build, browser, container, source-runtime build, image build, real registry, cloud operation, registry push, or deployment runs.
The simulated publication checks protocol and metadata only. It builds no runnable suite image.
The demonstration and its data remain unchanged.

Azure coded execution stays disabled. Actual network admission, independent absolute stop resources, registry identity binding, native setup, and private service packaging remain required.
Continue those controls before preparation enablement, remote gateways, recording, and separate deployment verification.
No broad acceptance item closes from local authentication fixtures alone.

## Previous delivery: durable Azure dispatch, recovery, and native completion

Acceptance **51** and supporting recovery under **52** advance. The total remains **48 verified of 57**, with **9 open**.
The previous request-contract batch commits as b62ea41. This batch implements the managed provider and production SDK bridge.

The azure-dispatches journal reserves Create and Start before each single mutation.
Provisioning stays asynchronous and gated. A later clock does not change the frozen proof or replica budget.
Current exchange authorization reconstructs the original short-lived ticket before Start. Revision assertions catch concurrent revocation or permission changes.
Activated, revoked, expired, ungated, and unauthorized exchanges receive no resumed ticket.
A recovery proof alone remains unable to create launch authority.
Lost responses reconcile the exact definition and execution history without another Create or Start.
Empty history remains unresolved. Multiple executions or changed ownership block admission and deletion.

Network admission, independent deadline verification, image inspection, and fresh authorization remain mandatory hooks.
The SDK bridge fixes public-cloud management scope, API version, owned job paths, response limits, and bounded pagination.
It disables automatic retries and provider registration. Redirects and foreign history links fail without forwarding credentials.
Exact owned cancellation works with unknown provider state. Pending provisioning can cancel without ever starting an execution.
Cleanup requires revoked work plus a saved result and complete or explicitly abandoned evidence.
Terminal receipts survive provider history expiry. Deletion retains a 30-second backoff and confirms absence before deadline removal.
Removed records remain fenced against redispatch and leave recovery inventory.

The shared lifecycle now accepts explicit native completion when a provider exposes no numeric process exit code.
It rejects conflicting completion evidence. Functional state, measured duration, and evidence still require the authenticated runner report.
Passing and failing runner reports retain their measured outcomes. Provider failure or missing reports retain unknown states and unavailable duration.
Existing numeric-exit providers retain their contract.

Targeted local race checks pass for both runners and both provider operations.
SQLite close/reopen, lost Create and Start responses, hidden history, concurrent controllers, admission drift, cancellation, and checkpointed deletion pass.
SDK checks pass for scoped single mutations, foreign pagination, redirects, unknown Start responses, and valid escaped resource groups.
Shared native-completion checks and existing wait, admission-loss, checkpoint, and numeric-exit compatibility checks pass.
Exchange checks cover fresh authorization and a revocation race detected by revision assertions.
Both native command packages compile without selecting tests.
Evidence is output/azure-worker-lifecycle/summary.json and the focused command results.
No browser, container, UI build, image build, source-runtime matrix, cloud operation, registry push, or deployment runs.
The demonstration and its data remain unchanged.

Concrete Azure environment network admission, independent absolute deadline resources, registry authentication, private exchange service packaging, and native wiring remain required.
Azure coded execution stays disabled until those controls exist. Local simulated hooks do not close deployment acceptance.
Continue those controls, then preparation, remote gateways, recording, and the separate deployment verification work.
No broad acceptance item closes from this local provider batch.

## Previous delivery: Azure job request and ownership contracts

Acceptance **51** advances. The count remains **48 verified of 57**, with **9 open**.
The previous goal batch commits ECS backend rotation support as 5afecb1. This batch continues the Azure adapter gap.

The new Azure package uses armappcontainers/v4 4.0.0 and API 2025-07-01 to construct and verify manual jobs and execution overrides.
Both Go and Playwright support normal execution and catalog discovery references.
Definitions fix one replica, one completion, zero provider retries, two CPU cores, 4 GiB memory, and Consumption resources.
They bind exact source digests, repository namespace, private exchange arguments, and the trusted supervisor command.
The sole pull identity has lifecycle None. Definitions reject runtime identity access, secrets, credentials, sidecars, init containers, and shared mounts.
Generated names fit the platform's shorter limit. Full ownership and specification hashes remain in tags.
The timing-pool digest includes installation configuration and the intended network policy.

Only a fresh, unexpired plan can emit ticket-bearing Start options. Definitions, recovery proofs, and plan diagnostics exclude usable tickets.
Restored proofs verify existing execution templates and cannot generate another launch.
Response checks retain legitimate provider metadata while rejecting writable drift and changed execution receipts.
The documented execution list can omit resource identifier and type; verification still requires the saved child receipt and matching execution name.
Provider completion and timestamps remain separate from runner outcomes, durations, and numeric process exit codes.
The selected execution API has no numeric exit-code field. Later lifecycle integration must preserve that distinction.

Six focused contract checks pass with race detection. The SDK wire check passes for both runners and both operations after one fixture correction.
Its first token audience expectation differs from the official SDK's public-cloud audience. Source inspection corrects the fixture, and only that failing check repeats.
The official SDK serializes exact create, start, and execution-stop requests into an in-memory transport. No Azure connection or credential discovery occurs.
The earlier three initial contract checks also pass before the complete new package's race run.
Final observation checks preserve missing optional provider timestamps, including stopped jobs that never start a replica.
An outdated negative case initially rejects a missing start time. Correcting that fixture leaves the targeted observation checks passing.
Evidence is output/azure-worker-contract/summary.json and the focused command results.
No UI build, browser, container, image build, source-suite run, cloud operation, registry push, or deployment runs.
The demonstration remains unchanged and available.

The adapter does not yet implement the managed Provider interface and is not wired into native commands.
Durable creation and Start reservations, lost-acknowledgment recovery, actual network admission, independent absolute deadlines, and registry authentication remain next.
Replica timeout does not replace absolute stop enforcement. The API cannot enforce the ECS user, read-only-root, or capability fields.
Keep strict supervised-image inspection and a separate Azure isolation design mandatory before enabling dispatch.
Private exchange service packaging, preparation, remote gateways, recording, and deployment verification remain required.
No acceptance item closes from request construction alone.

## Previous delivery: ECS admission during API backend rotation

Acceptances **51** and **57** advance. The count remains **48 verified of 57**, with **9 open**.
The earlier navigation refresh is verified in the existing user tab. It requires no further implementation.

Backend admission now requires at least one healthy target among one through 32 registered targets.
Known registration, initial-health, unhealthy, and deregistration transitions preserve admission when healthy capacity remains.
Every existing interface must retain the configured account, controller VPC, sole controller group, primary address, and private addressing.
Only draining with Target.DeregistrationInProgress permits an absent interface. Existing retiring interfaces retain all checks.
A cold controller can recover this state without an in-memory ownership history. Exact load-balancer egress remains mandatory.
Unknown health states, invalid reason pairs, absent active targets, and lost healthy quorum still block admission.
One EC2 request checks the complete backend set instead of one request per backend.
The 32-target local fixture confirms one lookup. Actual worker-interface verification remains a separate exact lookup.

The staged template now fixes target-level cross-zone routing, one-target health thresholds, disabled percentage thresholds, and 30-second connection termination.
It also fixes disabled client-IP preservation and proxy protocol. Admission rejects missing, duplicate, or changed required attributes.
The controller policy gains DescribeTargetGroupAttributes. User-code tasks receive no new authority.
Older installations must drain before applying the new attributes, read policy, and freshly built service image.

Targeted network race checks pass, including rollout, cold restart, ownership drift, lost healthy capacity, and maximum pool batching.
Nine local package previews and eleven rejected inputs pass. The generated package passes production admission with simulated AWS responses.
The existing provider admission and cleanup checks pass. Shared pending-admission and post-activation admission-loss checks pass.
The package alignment command first fails because Go runs from its package directory. Its corrected absolute fixture path passes.
Both native entrypoint packages compile. Their ECS-named test selection has no matching tests; it proves compilation only.
Evidence is output/ecs-rollout/package/summary.json and the focused command results.
No UI build, browser, image build, source-suite run, cloud operation, registry push, or deployment runs in this batch.
The user demo stays available. Live rotation, provider recovery, and deployment parity remain unverified.
Continue remaining Azure worker and gateway implementation next. Keep 41, 46, and 51–57 open.

## Previous delivery: staged AWS worker infrastructure and separate preparation

Acceptances **51** and **57** advance, with supporting private-journal recovery packaging under **52**.
The total remains **48 verified of 57**, with **9 open**. Local packaging does not close deployed acceptance.

The new worker template has 50 resources. The core template retains its 31 resources and gains optional worker references and source controls.
The worker package supplies an independent IPv4 VPC, exact endpoints and policies, DNS lists, internal TLS exchange, private connection, and deadline group.
It scopes controller creation, task management, tags, registry reads, role passing, and deadline management to the owned worker resources.
The platform image-pull role and stop-only Scheduler role remain separate from trusted controllers and untrusted test containers.
Immutable Go and Playwright repositories retain historical source images.

Prepare, connect, and ready stages withhold private settings until endpoint connection and acceptance sealing.
DNS ownership verification and explicit failure-mode configuration use documented operator steps supported by the pinned regional schema boundary.
No custom mutation function or broad DNS bypass is added. Runtime admission checks the actual verified and fail-closed state.
Generated settings contain resource references only. The API and preparation consume the same private document.
Explicit endpoint peer egress suppresses AWS default outbound rules. Worker ingress stays empty and broader endpoint destinations fail admission.

The separate preparation service uses its own role and security group, private PostgreSQL access, and retained encrypted EFS journals.
Its access point fixes user and group 65532 with mode 0700. IAM and TLS protect the mount; the filesystem denies plaintext, root, and access-point-free mounts.
Untrusted workers receive no journal mount or AWS task role. Preparation has no listener or load balancer.
New builds, new API source staging, and cleanup-only operation use separate controls. Registry writes stay on the preparation identity.
The service image gains a private owned journal mount directory. Its existing public and artifact entrypoints remain available.

Both templates pass local schema and dependency checks in two regions.
Nine stage or service previews and eleven rejected configurations pass in output/aws-workers-package/check.
The generated settings, groups, endpoints, policies, DNS rules, and private listener properties also pass production admission with simulated AWS responses.
Evidence is admission.log and endpoint-egress.log. Documented DNS operator prerequisites are simulated, not performed.
The changed ARM64 service image builds from cached UI and trace assets. No trusted runtime rebuild occurs.
Three local image probes pass: both packaged ECS entrypoints reject SQLite, and non-root preparation creates a private child on a real mounted volume.
An initial copied WAL-database read uses the wrong mode; immutable read mode corrects that checker failure.
All owned probe containers, named journal volumes, and anonymous temporary volumes are removed after failed and successful checks.
Evidence is image/summary.json and image/cleanup.json. Live EFS and IAM are not tested.
The local demo remains healthy. No browser, source suite, cloud account operation, registry push, or deployment runs.

Operator, worker, network, lifecycle, preparation, image, AWS, and plan guides match the staged package and current limits.
The current rollout batch replaces the all-targets-healthy restriction with locally verified transition admission.
Continue remaining Azure execution, gateway, recording, storage, and deployment verification work.

## Previous delivery: scoped ECS SDK setup, ECR authentication, and native wiring

Acceptance **51** advances. The total remains **48 verified of 57**, with **9 open**.
The optional SDK factory verifies STS account and ARN ownership, then constructs scoped regional ECS and network clients.
It retains the SDK credential chain for trusted controllers without copying ambient service endpoints, logging, proxies, or retry settings into control clients.
Control requests require expected HTTPS service hosts, bounded responses, request timeouts, and disabled automatic retries.
ECR authorization binds the configured registry and repository namespace. It remains in process memory rather than settings or worker input.
A bounded cache coalesces authorization and refreshes on SDK credential rotation or expiry. Failed refresh never falls back to stale authority.
Expected signed S3 configuration-blob redirects strip credentials and retain checksum and size verification.
Runtime closure releases owned SDK and registry connections.

Both native commands now accept ecs-workers-config or ecs-workers-env.
The API requires PostgreSQL, the ecs location, private port 8042, separate origins, and no local coded runner settings.
Origin checks normalize host case and default ports for both hosted providers.
The artifact role rejects ECS settings. Mixed ECS and Kubernetes configuration is rejected before storage initialization.
The canonical hosted driver wiring now accepts either provider name and retains approval, scoped discovery, and cleanup recovery.
Preparation uses the same registry builder and discovery workflow with a separate trusted process identity and durable private journal directory.
Omitting runtime flags retains cleanup-only configuration. API and preparation must retain identical worker and network settings while cleanup remains unresolved.
The example settings contain only independent resource placeholders. They enable no existing account resources.

Six focused ECR and real-SDK protocol checks pass with race detection in output/ecs-sdk-wiring/final-sdk-contract.log.
They use explicit fixture credentials and intercepted HTTP transports; the credential chain and external endpoints are never accessed.
A checksummed OCI image passes through the actual SDK authorization and credential-free S3 redirect path.
Account mismatch, endpoint drift, private errors, disabled retries, token rotation, cache expiry, and response limits are covered.
Both native binaries build. Twelve startup guards pass in output/ecs-sdk-wiring/startup/summary.json without initializing their database.
No browser, container, service image rebuild, trusted runtime rebuild, UI build, registry push, or live provider operation occurs.
The existing demo remains healthy and retains its data. Source-page help already describes configured locations without provider-specific restrictions.
Operator, network, lifecycle, preparation, exchange, AWS, and plan guides reflect the new optional flags and remaining package boundary.

Next package private routing, worker networking, least-authority control and deadline roles, optional service settings, and the separate preparation workload.
Preserve the required endpoint-group egress state, private durable journals, exact image approval, and cleanup-only operation.
Rebuild the service image once these inputs change. Keep target-profile verification in its separate session.

## Previous delivery: supported Fargate storage and non-root runtime volumes

Acceptance **51** advances, with supporting resource-class timing identity under **41**. The total remains **48 verified of 57**, with **9 open**.
Primary-source review finds two errors in the earlier draft: Fargate rejects tmpfs, and explicit ephemeral storage starts at 21 GiB.
The task definition now supplies two task-scoped volumes and an explicit 21 GiB allocation without host or persistent storage.
It retains non-root execution, a read-only root filesystem, fixed memory and CPU, private networking, and the independent deadline.
The resource class uses a new timing-pool identity. Per-mount tmpfs sizes and noexec behavior are no longer claimed.

The thin supervisor overlays now declare owned /tmp and /output image volumes and a task-volumes/v1 capability.
OCI inspection checks the capability and exact volume declarations. Source assembly preserves both in the immutable prepared image.
ECS image admission rejects absent or unsupported storage capability. Existing provider-independent legacy image inspection remains supported.
Older overlays require rebuilding and source preparation before ECS selection. This provider is not enabled in any production profile yet.

Targeted ECS and OCI race checks pass in output/ecs-network-admission/task-storage-contract.log.
Cases reject persistent host paths, changed mount destinations, unsupported tmpfs, changed allocation, and invalid storage metadata.
The two-runner source assembly check passes with the new capability in output/ecs-task-storage/source-compatibility.log.
API, preparation, and supervisor commands compile in output/ecs-task-storage/compile.log.
Both thin ARM64 overlays build from unchanged cached bases. Four focused local package checks pass in output/ecs-task-storage/package.
The probe verifies actual non-root volume initialization, writable scratch paths, and rejected writes to runtime and suite directories.
An initial probe incorrectly treats permission-bit access as proof of a writable filesystem. Actual write attempts correct that check.
The failed and final probes remove their owned containers and anonymous volumes. No browser or uploaded source suite starts.
No cloud operation, registry push, toolchain rebuild, service image rebuild, or UI build occurs. The demo remains available.

Next add the scoped SDK factory, rotating ECR authentication, command wiring, and private worker infrastructure.
Keep deployment, storage, timing, and hosted recovery acceptance open until profile-specific verification passes.

## Previous delivery: ECS network admission and private settings

Acceptance **51** advances, with supporting timing-pool identity under **41**. The total remains **48 verified of 57**, with **9 open**.
The adapter now checks actual EC2, ELBv2, Route 53 Resolver, and Route 53 Profiles responses through typed SDK interfaces.
A dedicated worker VPC keeps its strict DNS policy separate from trusted controller identity, notification, and target access.
Worker routing permits only local routes and the restricted S3 gateway. Exact groups permit only required HTTPS paths.
Private ECR policies bind platform pull authority and the repository prefix. The S3 policy permits only AWS-owned image-layer reads.
The PrivateLink service, internal TLS load balancer, private controller targets, exact DNS lists, and fail-closed firewall must match.
Profile overlays, public task interfaces, changed groups, inconsistent attachments, and unrecognized private diagnostics block admission.
Pool refreshes serialize and have bounded caching. Full task checks and refreshes have bounded contexts and single-attempt SDK calls.
The settings loader accepts one bounded file or environment reference and binds isolation settings into worker ownership and timing identity.

All 19 ECS package checks pass with race detection, including the existing lifecycle contracts and six new network/settings checks.
The final check includes 22 rejected pool mutations, six task-interface mutations, settings scope, cache expiry, and concurrent readers.
Evidence is output/ecs-network-admission/final-contract.log. No credentials are discovered and no live provider is contacted.
The four added SDK modules retain the existing AWS core SDK version. The demo remains healthy without another restart.
No browser, UI rebuild, service image, or trusted-runtime rebuild is needed for this batch.
See [ECS network admission](ecs-network-admission.md) for the exact contract, primary references, required read permissions, and remaining integration.

The primary-source review also identifies unsupported tmpfs mounts in the earlier Fargate task definition.
The next delivery above corrects those mounts. SDK initialization, ECR authentication, native settings wiring, and private infrastructure remain required.
Local contracts do not close deployed execution, timing, storage, or recovery acceptance.

## Current delivery: durable ECS dispatch and checkpointed recovery

Acceptance **51** advances with supporting replica recovery under **52**.
The total remains **48 verified of 57**, with **9 open** items. ECS remains disabled until production admission, settings, and private routing exist.

The provider now implements the shared lifecycle through typed SDK-compatible ECS and Scheduler clients.
Durable reservations and exchange assertions precede registration, launch, and deadline creation.
Only the winning reservation creates resources. Recovery lists and verifies existing resources without another creation request.
Exact definition and task ARNs, ticket-free proofs, terminal observations, and replay tombstones survive database reopen.

The shared status contract adds AdmissionPending for accepted tasks whose runtime or network interface is still becoming ready.
Execution and discovery keep executable input gated until admission completes. Admission loss after activation revokes and stops owned work.
The ECS constructor requires installation and actual-task network checks. No production network bypass is supplied.
Stop-time tag loss preserves exact saved ownership. Missing live tasks remain unresolved without a saved terminal receipt.

Cleanup requires revoked exchange ownership and the saved result/evidence or preparation checkpoint.
It verifies and deletes the exact deadline and immutable definition after confirmed termination.
Asynchronous definition deletion retains the saved result and backs off cloud reads for one minute.
Public control errors exclude raw provider diagnostics. Existing installation-transfer guards retain the exchange until cleanup finishes.

Targeted race checks pass for both runners and both operations, concurrent controllers, lost registration/launch/deadline responses, late task appearance, and missing live history.
Database-reopen recovery, checkpoint and revocation separation, changed schedules, delayed deletion, admission rejection, and private error handling pass.
The shared real exchange and ledger path passes delayed admission and admission-loss checks.
Targeted existing discovery and Kubernetes compatibility checks pass. API and preparation commands compile.
Evidence is output/ecs-worker-lifecycle: final-contract.log, restart-and-control.log, checkpoint-invariants.log, discovery-compatibility.log, kubernetes-compatibility.log, and compile.log.
No cloud account, registry, browser, UI build, service image build, or source-runtime rebuild occurs. The existing demonstration is preserved.

See [Durable ECS worker lifecycle](ecs-worker-lifecycle.md) for exact implementation boundaries and recovery limitations.
Next implement actual EC2 network admission, scoped SDK initialization, bounded private settings, private worker HTTPS routing, and deployment permissions.
Keep AWS disabled until these controls are wired and verified locally. Hosted deployment acceptance remains separate work.

## Previous delivery: ECS task, independent deadline, and recovery proof contracts

Acceptance **51** advances. The total remains **48 verified of 57**, with **9 open** items.
These contracts do not enable ECS workers or close provider acceptance.

The new ECS package builds pinned SDK task definitions and launch requests for offline Go and Playwright execution and discovery.
Task identity, immutable image, fixed supervisor arguments, ownership tags, account, region, execution pool, and deadline remain bound.
Tasks have no AWS task role or secret references. The platform image-pull role and independent deadline role remain separate.
Fixed non-root, read-only, resource, tmpfs, and restart settings reject unexpected mutations.
Launch requests have one task, no public address, no ECS Exec, and only the scoped exchange identity and ticket overrides.

The planner builds a separate UTC stop schedule for the exact task ARN.
Verification checks its identity, enabled state, exact stop target, role, time, and bounded retry policy.
The documented minute-scale Scheduler precision remains separate from exact runner and ticket deadlines.
SDK input follows the ECS StopTask field names. An owned TLS fixture verifies it against a signed production SDK request.

Serializable recovery proofs retain hashes and ownership without raw tickets. Restored plans verify resources but cannot create another launch request.
Changed claimants or execution pools invalidate restoration. Canonical subnet order preserves timing-pool identity.
Pinned ECS and Scheduler modules use the existing AWS core SDK and add no unrelated dependency upgrades.

Focused race checks pass for both runners, both operations, definition and deadline mutation rejection, configuration bounds, private diagnostics, and restored-proof restrictions.
The initial fixture has an oversized ticket canary; correcting its length lets it exercise the intended production contract.
The signed StopTask fixture uses only an explicit local credential callback and owned TLS endpoint. It closes after the check.
Current API, preparation-worker, and supervisor packages compile.
Evidence is output/ecs-worker-contract/contract.log and compile.log.
No cloud credential lookup, cloud account operation, registry push, browser, UI rebuild, service image build, or source-runtime rebuild occurs.
The local demonstration and its retained data remain unchanged.

Next implement durable provider dispatch and recovery, exact task observations, network admission, and checkpointed resource deletion.
Then add optional settings, private exchange routing, and AWS worker infrastructure.
StopTask removes task tags. Persist exact task and definition ownership before stop and preserve terminal observations across later resource expiry.
The finite RunTask idempotency window must not become permission to redispatch uncertain work.
See [ECS task and deadline contracts](ecs-worker-contract.md) for the implementation boundary and verified primary references.

## Previous delivery: Helm source preparation and private journals

Acceptances **51** and **57** advance with supporting preparation failover packaging under **52**.
The total remains **48 verified of 57**, with **9 open** items. Local package checks do not close hosted acceptance.

The optional preparation workload uses the immutable service image and a separate command, service account, projected token, and registry writer environment.
The API retains its registry reader. The chart rejects shared reader/writer Secret objects and embeds no credential values.
The preparation ConfigMap contains only worker and registry reference settings. It mounts no artifact claim or OIDC settings.
No Service, Ingress, or container port exposes the preparation process. Its ingress policy denies incoming traffic.
Existing bounded worker-control RBAC now binds both trusted controller accounts. Untrusted worker permissions remain unchanged.

A separate durable claim retains private registry upload journals. A mode-0700 child protects journal contents beneath the writable mount.
ReadWriteOnce journals use a single-replica Recreate rollout. Multiple replicas require shared ReadWriteMany storage and receive a disruption budget.
Cleanup-only mode keeps the process, provider authority, writer credentials, and journals while disabling new API staging and builder flags.
Source review, explicit publication, and approved execution retain their existing behavior.

The API adds --source-bundle-runners to advertise only supported source builders. Its default remains go,playwright.
The chart derives that list from selected immutable runtime references. The preparation command also derives synchronization profiles from enabled builders.
Targeted parser checks pass. Go-only chart settings emit only the Go upload profile.

Helm lint and six strict-schema profiles pass locally, including disabled preparation, both builders, Go-only staging, cleanup-only operation, shared journals, private CA, and Bearer credentials.
Twelve rejected configurations cover missing worker isolation, missing or mutable runtimes, foreign registry references, claim collisions, replica/storage mismatch, and credential errors.
Both rendered process settings load through the production Go parser and match the provider's exact protected network policy.
Nine native and packaged startup guards pass before database initialization or provider contact.
The service image reuses the UI, private trace viewer, and health binary. No source-runtime or browser dependency rebuild occurs.
Owned startup probe containers are removed and their absence is verified. The local demonstration remains healthy.

Evidence is output/kubernetes-preparation: package/summary.json, api-settings.log, preparation-settings.log, startup/summary.json, and startup/cleanup.json.
See [Kubernetes source preparation](kubernetes-preparation.md) for operator setup, storage privacy, drain behavior, and limits.
There is no registry push, browser run, cluster, AWS, or Azure operation.
Next implement AWS worker dispatch and private exchange packaging through local protocol fixtures.
Remote gateway profiles, Azure workers, and hosted deployment verification remain open.

## Previous delivery: hosted discovery and preparation queue integration

Acceptance **51** advances with supporting preparation failover work under **52**.
The total remains **48 verified of 57**, with **9 open** items. No hosted acceptance closes from local fixtures.

The preparation command now selects registry assembly and Kubernetes discovery through explicit settings and immutable runtime flags.
The API stages source and presents the existing review. Uploaded source assembly remains outside the API process.
The existing build checkpoint, discovery validation, exact artifact approval, and publication contracts accept registry distribution references.
The service image includes the separate preparation command. Older runtime overlays fail the discovery capability check before registry publication.

The exchange uses a distinct gated discovery operation with source-job authority and no measured test report or attachments.
Discovery logs, output size, source digest, suite, runner, and offline profile are bounded and validated.
The supervisor uses fixed discovery commands, owned process groups, stripped child credentials, redaction, and bounded delivery retries.
Provider success and process exit remain separate from valid report delivery. A failed provider cannot create a ready review.

The source service saves the ready review or failure before permitting provider deletion.
Cleanup verifies the exact preparation reference and uses provider termination and deletion preconditions.
Recovery adopts owned discovery Pods from the exchange without inventing a test-run ledger.
Cleanup-only recovery works with new builds disabled. Ready cancellation reuses the existing lease without a conflicting insert.
Interrupted work never dispatches or discovers again automatically.

Targeted race checks pass for the source workflow, both discovery schemas, supervisor behavior, Kubernetes protocol fixtures, registry assembly, and normal execution compatibility.
Follow-up checks pass for failed-provider reports, disabled-build recovery, ready cancellation, and normal exchange compatibility.
Native and packaged startup guards pass seven cases before opening SQLite or contacting a provider.
Both thin runtime package probes pass with the new discovery capability and fixed CLI operation.
The owned package probe containers are removed and their absence is verified.
The trace viewer, health binary, trusted Go base, and trusted Playwright base are reused.
The Angular production build includes the updated source help. No browser workflow rerun is needed for that text change.
No browser, external registry, cluster, AWS, or Azure verification runs. The existing demonstration is preserved.

Evidence is output/hosted-discovery: contract.log, provider-outcome.log, deferred-recovery.log, exchange-compatibility.log, startup, and runtime-package.
See [Hosted preparation](hosted-discovery.md) for exact native configuration, package command overrides, and recovery requirements.
The delivery above implements the Helm preparation deployment, durable private journal volume, and separate registry writer credential references.
Remote gateway delivery, other provider adapters, and deployment acceptance remain open.

## Previous delivery: registry source assembly and immutable publication

Acceptance **51** advances. The total remains **48 verified of 57**, with **9 open** items.
This delivery provides the builder adapter. The current delivery above adds hosted discovery and preparation-worker wiring.
No provider or complete source workflow acceptance closes from the local fixture.

The registry builder implements ArtifactBuilder for validated offline Go and Playwright snapshots.
It verifies a supervised registry runtime and appends a deterministic source layer without Docker, shell execution, or uploaded build commands.
Controlled manifests, retained provenance, private file ownership, opaque source-directory replacement, and bounded paths preserve source confinement.
The configuration retains the runtime environment, entrypoint, platform, supervisor, and original layers.
Artifact identity binds project ownership, preparation reference, source digest, filesystem digest, and the retained configuration.

Registry uploads use exact repository pull,push authentication, content-addressed deduplication, and fixed-origin upload locations.
Lost blob or manifest acknowledgements resolve through exact digest checks. Known sessions close before their journal clears.
Private durable journals precede creation and source transfer. Cancellation and restarted cleanup use only recorded owned sessions.
Unknown creation remains cleanup-incomplete and cannot automatically repeat. No private upload URL appears in public diagnostics.
Cleanup preserves published artifacts and shared base layers. Registry garbage collection remains an operator responsibility.

Targeted race-enabled checks pass for both runners, source confinement, deterministic images, controlled metadata, runtime preservation, dependency restrictions, scoped token challenges, deduplication, lost acknowledgements, cancellation, rotated credentials, interrupted checkpoints, and uncertain creation.
The existing OCI inspection and affected local assembly checks also pass. Evidence is output/registry-preparation/contract.log. Final scope and durability checks pass in final-contract.log and durable-journal.log.
The API, preparation-worker, and publisher packages pass compilation.
No external registry push, cloud account, cluster, browser, UI build, or runtime image rebuild occurs.
The local demonstration stays running with its existing data and evidence.

Next connect hosted discovery and the existing preparation queue to this builder.
The existing immutable-image validation already accepts registry distribution references. Current queue integration verifies that contract without weakening local validation.
Preserve source review, approval, catalog publication, activation, cancellation, and cleanup-only recovery.
See [Registry source assembly](registry-preparation.md) for the operator contract and exact remaining scope.

## Current delivery: Kubernetes Pod execution, private routing, and activation gate

Acceptance **51** advances. Timing **41** and replica recovery **52** gain implementation support.
The total remains **48 verified of 57**, with **9 open** items.
No live provider or deployment acceptance closes from this local fixture.

The first concrete provider uses one owned Kubernetes Pod per offline Go or Playwright attempt.
Its shared database reservation allows only one creation across concurrent controllers.
It retains the reference, specification hash, creation uncertainty guard, API scope, and terminal observation without persisting worker or API tokens.
Recovery discovers an uncertain creation, stops owned work, and reuses accepted private reports without creating another Pod.
Removed references remain replay tombstones.

The adapter verifies namespace restriction, runner service account, RuntimeClass, and exact namespace network policy before creation.
The returned Pod must match its complete reference, frozen image, deadline, and immutable specification.
Known API defaults normalize before comparison. Unexpected admission injection prevents executable input delivery.
Cancellation uses a guarded deadline update. Deletion requires terminal state and exact UID and resource-version preconditions.
A previously observed live Pod that disappears remains uncertain. No forced deletion manufactures proof of termination.
Inventory validates namespace resources and durable outstanding dispatches.
The timing key now includes cluster scope, RuntimeClass, node selection, and fixed resource profile.

The shared managed lifecycle freezes work behind an activation gate.
Workers cannot fetch input, bind an instance, or commit start progress until controller verification succeeds.
Activation rechecks run authority with storage guards. No worker endpoint exposes activation.
A rejected or uncertain provider stays blocked. Recovery never activates uncertain work.
An already accepted report remains recoverable after a later checkpoint failure.
Legacy direct transport preparation retains its existing behavior unless the trusted caller selects the gate.

Explicit file or environment settings connect the adapter to the API and private listener.
The mode requires PostgreSQL and the Kubernetes execution location. It rejects artifact-only service roles and local coded configuration.
The chart optionally supplies a dedicated restricted namespace, token-free runner service account, narrow controller RBAC, rotating controller token, private TLS ingress, and worker network policy.
The worker namespace and policy remain after Helm uninstall until explicit operator cleanup.
The default profile keeps workers disabled. No RuntimeClass, private ingress controller, registry secret, or TLS secret is invented.

Focused local TLS race checks pass for both runner report schemas, assertion outcomes, deadlines, concurrent creation, lost acknowledgements, checkpoint recovery, missing live Pods, admission mutation, ownership, mounted trust, rotating tokens, and timing-pool changes.
The real settings parser accepts the rendered configuration and matches the adapter's expected policy.
The activation gate and affected shared lifecycle checks pass with race detection.
Evidence is output/kubernetes-workers/provider.log.
The final terminal-state and rendered-policy checks pass in provider-final.log.

The targeted chart verifier passes two profiles, four rejected configurations, and the generated worker Pod's strict schema.
Evidence is output/kubernetes-workers/package/summary.json.
Five native and packaged startup probes reject unsupported database, service-role, or execution-location combinations before dispatch.
Evidence is output/kubernetes-workers/startup/summary.json and cleanup.json. Both owned probe containers are confirmed removed.
The image probe uses environment JSON because the local Docker daemon does not share the host temporary bind path.
It clears the image's default database environment argument when exercising the explicit SQLite rejection.

The changed ARM64 service image is sha256:2519658629aaceb903566f9065157c8373476fc1a8dc0c0d39cbba068f96786c.
Its exact inputs are output/kubernetes-workers/service-image.json. This is a local image identifier, not a distribution digest.
Native and service builds reuse the UI, private trace viewer, health binary, and source runtime images.
The local demonstration refresh preserves its database and private evidence.
No registry push, browser suite, source-runner matrix, cluster operation, or cloud account access occurs.

Next implement remote source assembly, registry publication, and hosted discovery through preparation.
A newly prepared registry artifact must retain normal approval and publication before execution becomes a usable hosted source workflow.
Trusted remote gateways, AWS and Azure adapters, and complete deployment acceptance remain required.
See [Kubernetes execution workers](kubernetes-workers.md) for exact operator inputs, platform dependencies, evidence, and continuation limits.

## Current delivery: immutable registry inspection for hosted workers

Acceptance **51** advances. The verified count stays **48 of 57**, with **9 open** items.
The new OCI registry adapter supplies the image inspection method for real provider adapters.
It admits only an explicit HTTPS registry, repository namespace, and sha256 distribution reference.
It verifies raw manifest and configuration digests, descriptor sizes, platform selection, source ownership, and the supervised runtime contract.
The shared lifecycle also compares inspected project ownership and retained source provenance before dispatch.
Both Go and Playwright metadata use the same path. An index must select exactly one supported configured platform.

Private Basic and Bearer credentials use a fresh callback for each inspection.
Bearer challenges must match the configured endpoint, service, and exact repository pull scope.
One scoped token serves that inspection. Another inspection can use rotated credentials.
Manifest and token redirects fail. Explicit configuration-blob redirect origins receive no registry Authorization header.
All returned bytes retain digest verification. Private values, response bodies, and signed query values stay outside errors.
Metadata, token responses, request counts, layer declarations, and elapsed time have bounds.
The adapter fetches no filesystem layers and starts no test code.

The targeted local TLS fixture and affected shared lifecycle checks pass with race detection.
They reject a mismatched project or retained source digest before dispatch.
The TLS fixture covers both runners, direct manifests, indexes, scoped authentication, credential rotation, exact metadata request counts, approved blob redirects, digest mismatch, unsafe references, unsupported runtimes, oversize responses, ambiguous platforms, and descriptor size mismatch.
Evidence is output/registry-inspection/contract.log.
No native service, UI, browser, runner image, or provider deployment rebuild is needed because this adapter is not wired into the API yet.
No cloud account or cluster is contacted.

Next connect this inspector to concrete isolated provider dispatch.
Private worker routing, provider lifecycle operations, remote source preparation, and immutable publication remain required.
See [Hosted registry inspection](registry-inspection.md) for the operator contract and exact current scope.

## Current delivery: shared hosted lifecycle and evidence recovery

Acceptance **51** advances. Timing identity **41** and replica recovery **52** gain implementation support.
The verified count remains **48 of 57**, with **9 open** items.
No hosted deployment acceptance closes from the local fixture.

The new shared managed executor requires approved registry distribution digests and inspected supervised runtime metadata.
It checks selectors, ownership, private origins, and platform-specific timing identity.
It currently rejects target-backed artifacts and credential bindings until trusted remote gateway delivery exists.
No provider adapter is enabled. Existing local execution remains the active demonstration path.

The driver uses the existing durable execution reference and lifecycle callbacks.
It checks dispatch authority around frozen packet creation and calls provider creation once.
Ambiguous acknowledgement, ownership mismatch, uncertain stop, and storage failures retain recovery state.
Recovery stops remaining owned work and reads accepted reports without creating another job.
Provider success represents delivery. Assertion outcome and measured test duration remain in the structured report.
A failed provider cannot validate a passing test report or invent a duration.

Ticket revocation precedes checkpoint publication.
Cleanup checks the result and evidence ledger before stopping or deleting provider resources.
Provider deletion precedes exchange cleanup. Final ledger completion records cleanup failure separately from the saved outcome.
The exchange preserves active upload fencing and staging reservation release.

Two integration corrections support restart recovery.
The run recovery input now derives FailedBefore from the attempt number instead of forcing true for the first attempt.
The exchange marks missing or temporarily unreadable attachments as internally retryable evidence.
The run checkpoint retains the result and defers cleanup while evidence collection remains retryable.
A temporary storage outage recovers its accepted bytes without another test dispatch.

Targeted race checks cover the shared lifecycle, both report schemas, duration preservation, cancellation, deadlines, private evidence, lost acknowledgement, uncertain stop, failed checkpoint, failed cleanup, timing admission, and ownership rejection.
Two focused run checks cover frozen attempt flags and pending evidence.
Three focused exchange checks cover immutable observations, active uploads, and staging quota release.
Evidence is output/job-supervisor/hosted-lifecycle.log and lifecycle-exchange.log.
Native service builds reuse the unchanged UI and private viewer. No runner image or browser suite rebuild is required.
The local demonstration retains its database and evidence during refresh.

Provider-specific creation, status, cancellation, isolation, private routing, registry preparation, and deployment verification remain required.
See [Hosted execution lifecycle](hosted-execution.md) for the concrete provider contract and continuation boundary.

## Current delivery: hosted runner supervisor and thin runtime overlays

Acceptance **51** advances. The ledger stays at **48 verified, 9 open, 57 total** items.
The new teststudio-job command fetches one scoped work item and invokes the existing fixed Go or Playwright runner inside its worker image.
It starts no uploaded code in the API and accepts no arbitrary command or ticket argument.
The ticket, execution, and optional instance identity use protected environment values.
The child gets a fixed toolchain environment without the job ticket, database settings, cloud credentials, identity headers, or personal proxy configuration.

The supervisor uses bounded retries for input, progress, report, and attachment delivery.
A single process retains its instance identifier. Exact retries preserve the start timestamp, report, manifest, and snapshotted bytes.
A lost report acknowledgement cannot rerun the test.
Start delivery occurs separately from stdout draining and is bounded to five seconds.
The local and hosted adapters now share the existing bounded output parser, including its 2 MiB retention and bounded progress-line behavior.
The existing local parser checks use its public pending-byte observation instead of private buffer access.

The owned runner process group receives termination before a bounded forced stop.
The supervisor waits for the runner, completes an active forced-stop callback, and stops remaining members of its group.
The local cancellation fixture starts a descendant and confirms that both runner completion and descendant reaping occur.
Provider PID namespace cleanup and sandbox controls remain separate requirements.

The collector verifies regular files through the rooted output directory and creates private read-only snapshots before manifest submission.
Escape links, duplicate paths, invalid metadata, capture-disabled files, oversized data, and output truncation stay explicit evidence gaps.
It removes its owned snapshot directory on delivery or an error.
Known job ticket values are removed from report metadata and captured process output.
The selected source runtime still requires the trusted gateway socket for target-backed work. No gateway or isolation boundary is invented by the supervisor.

The report envelope now retains the actual runner process exit status and an optional infrastructure failure discriminator.
A valid failed assertion delivers successfully while retaining its failed test outcome and nonzero runner exit.
Missing reports and invalid passing reports produce teststudio.worker/v1 observations with unavailable test duration and measured worker duration.
A worker infrastructure observation cannot create a passing test result.
Available process logs remain private evidence. Child stdout and stderr do not stream to provider console logs.
The supervisor's exit status describes report delivery. Test outcome remains in the accepted report.

Staging quota exhaustion before report acceptance permits an explicit attachment omission without changing the observed result.
Upload failure after acceptance keeps the report and records incomplete evidence separately.
The CLI's fixed delivery message uses that incomplete-evidence outcome even when no attachment remains pending.
The shared capture helper also fixes failure-only captures for a passing observation that fails its timing threshold.
The broker and supervisor use the same timing-aware capture decision.

The new runtime producer layers only the compiled supervisor onto a selected immutable source runtime.
It retains the original runner entrypoint, runtime labels, API policy, and source-base contract.
The layer adds a root-owned executable and defaults to user 1000:1000.
The current Go and Playwright overlays reuse the existing cached ARM64 bases. No Go toolchain, npm dependency, or browser runtime rebuild occurs.
Buildx selection now uses a shared helper with the service image producer. Its source participates in both relevant cache keys.
The producer loads images locally and performs no push or provider job creation. Its temporary build context is removed.
Produced overlays and immutable base aliases remain as intended build cache.

The Go overlay is sha256:2636d33fd2ac03dae1dbd5624ec3a339b14d705ae51d56efea637a45cb354f60.
The Playwright overlay is sha256:89611e083f27f68e494f92eaa12380268db301134562dc4efc018f4bc3fa9f34.
The final image identifiers and exact inputs are recorded in output/job-supervisor/go-runtime.json and playwright-runtime.json.
They are local image identifiers, not registry distribution digests.

Focused native supervisor checks use real child processes and local TLS with the production exchange handler and client.
They cover both runner schemas, passed and failed observations, missing reports, invalid passes, credential exclusion, oversized output, unsafe links, lost acknowledgements, quota gaps, upload failure, stale output, and cancellation cleanup.
The command configuration check verifies missing or invalid identity and credential-bearing flags without echoing values.
Two focused exchange cases preserve process-failure consistency and timing-triggered capture.
Five existing report and output checks preserve local behavior after the shared parser extraction.
All targeted Go checks pass with race detection.
Evidence is output/job-supervisor/supervisor.log, command.log, exchange.log, and compatibility.log.

Narrow overlay probes execute the installed supervisor configuration guard with no network, no capabilities, and a read-only root filesystem.
They confirm non-root image settings, unchanged source contracts and runner entrypoints, and missing-work rejection.
Evidence is output/job-supervisor/package/summary.json and cleanup.json.
All owned probe containers are removed. No test suite or browser starts during these probes.
The native API and preparation worker build with reused UI and private viewer assets.
The demonstration refresh preserves its database, evidence, and existing browser work.
No AWS, Kubernetes, Azure, full browser suite, or source-runner matrix runs.

Provider admission, private worker routing, exact image dispatch, trusted remote gateway delivery, authoritative status, cancellation, deadlines, discovery, and restart reconciliation remain required.
Remote source preparation and immutable registry publication also remain required.
The next batch connects the exchange and supervisor to the common provider lifecycle driver and provider adapters.
See [Hosted runner supervisor](job-supervisor.md) for the operational contract and current limits.

## Current delivery: private hosted worker exchange

Acceptance **51** advances. The ledger stays at **48 verified, 9 open, 57 total** items.
The new worker exchange binds a frozen work item, start event, report, and declared attachment bytes to one normal execution reference.
The service checks current controller ownership, attempt, test, lease, run deadline, project state, and the existing execution admission rules.
The normal ledger reference must remain prepared and uncheckpointed.
Ownership and project records guard relevant metadata writes.
The optional --worker-addr listener is separate from the UI and artifact listeners and stays disabled by default.
An artifact-only process rejects it. The listener starts no test process.

An installation-private signing key and exchange nonce derive a stable HMAC-SHA256 ticket.
Concurrent controllers reconstruct one ticket. The first worker instance claims the exchange; another instance cannot execute it again.
Later preparation cannot replace the input or extend the frozen deadline.
Preparation refreshes authority guards after a concurrent metadata update rather than retrying a stale revision.
Tickets are absent from packet bodies, formatting, JSON serialization, URLs, session cookies, and application identity records.
The selected work item excludes credential references, redaction values, unrelated selectors, native HTTP headers, source provenance, and controller callbacks.
It retains the existing 4 MiB bound, immutable artifact identity, selected test, target policy, capture settings, and timing values.

The HTTPS client rejects redirects, insecure origins, oversized responses, and invalid packet identity or expiry.
Its default transport uses ordinary TLS verification without environment proxy settings.
Request failures use fixed errors without upstream headers, bodies, URLs, or network diagnostics.
The handler rejects cookies, browser Origin headers, and query strings. Worker authorization remains separate from application sessions.

Start events and report submissions support exact retries after a lost acknowledgement.
Changed start events, reports, manifests, and instance claims conflict.
The report validator moves from the local container adapter into the shared execution package.
Both paths preserve selector, schema, test ownership, terminal state, timeout normalization, timing, Go observation, step, and process outcome checks.
A passing report cannot hide an incompatible failed process outcome.

Attachment manifests declare at most 32 files with exact sizes and SHA-256 checksums.
Capture mode, failure-only settings, filenames, media types, byte limits, and total attempt bounds validate before storage.
Project and run staging reservations bound simultaneous attempts. Exact report retries reserve no additional space.
The project staging limit follows current project settings. The run staging limit follows its frozen capture policy.
Staging is a separate bounded capacity from retained evidence. Normal evidence import still enforces its own quotas, retention, pins, and redaction.

Four uploads can stage concurrently per service process.
A bounded per-file lease protects backend writes. Revocation does not discard the metadata needed to track a write already in progress.
Collection verifies stored content again and can recover bytes written before a metadata acknowledgement failure.
Missing bytes remain an evidence gap and can be collected later without another execution.
Cleanup waits for live uploads and requires a saved result with complete or explicitly abandoned evidence.
Object deletion precedes atomic reservation release and exchange removal. A lost cleanup acknowledgement remains retryable.
Installation export now refuses unfinished exchanges even if another recovery index is missing.
The signing key remains protected installation authentication state in a completed backup archive.

Eight exchange tests and three live-run or HTTP tests pass with race detection.
They cover controller concurrency, one worker instance, restart reconstruction, private input, immutable messages, checksums, expiry, capture bounds, shared quotas, cleanup races, and exact cleanup retry.
A real TLS fixture uses the production run authorization callback, handler, and client.
It covers start progress, a deliberately lost report acknowledgement, attachment names with spaces, duplicate uploads, cancellation, stale ownership, expired leases, archived projects, and redirect rejection.
Three existing focused report cases preserve invalid-pass rejection, Go preparation failures, and legacy timeout interpretation after extraction.
The targeted export barrier covers normal execution work and unfinished exchange records without publishing a partial archive.
Its fixture now uses a fresh database instead of seeding and executing an unrelated installation.
Evidence is output/worker-exchange/service.log, run-http.log, report-compatibility.log, and transfer-barrier.log.
The quota fixture initially regenerates report timestamps during an exact-replay assertion. Keeping the same frozen report corrects the fixture.

The native listener check starts one fresh isolated process, checks private routing, and confirms that startup creates no ticket or signing key without trusted dispatch.
Evidence is output/worker-exchange/listener/summary.json and cleanup.json.
Its owned process stops and temporary database and storage are removed.
No browser, Docker test runtime, source-runner matrix, AWS, Kubernetes, or Azure test runs.
The native build reuses UI and private viewer assets. Existing trusted runner images remain unchanged.

Hosted runner supervision, provider job creation, gateway delivery, private provider ingress, authoritative status, cancellation, deadline enforcement, and restart discovery remain required.
Provider preparation must still publish and inspect immutable runnable artifacts outside the API.
The current transport library and listener do not enable hosted Go or Playwright runs alone.
The next batch adds the trusted supervisor and provider lifecycle integration against local protocol fixtures.
See [Private hosted worker exchange](worker-exchange.md) for the HTTP contract, quotas, and required integration order.

## Current delivery: Azure core packaging and separate service roles

Acceptance **57** advances. The ledger stays at **48 verified, 9 open, 57 total** items.
The new deploy/azure Bicep modules package independent API and artifact Container Apps with separate HTTPS origins.
They reference an installation environment, registry, Key Vault, and PostgreSQL connection secret.
The package creates five managed identities, private Blob storage, scoped permissions, a storage deletion lock, and two service apps.
It retrieves no secret values or account keys and depends on no AWS setting or credential.

The API runtime receives Blob writer authority on the evidence container. The artifact runtime receives reader authority there.
Separate startup identities receive read authority only on named Key Vault secrets.
The artifact startup identity can resolve only the database secret, without provider settings or client credentials.
Registry pull and startup identities use lifecycle None. Only the selected Blob runtime identity uses lifecycle Main.
Blob configuration selects each app's exact runtime client identifier.
Assignment names include the actual principal identifier to permit identity replacement without mutating an existing assignment's principal.
PostgreSQL, identity reachability, role propagation, TLS, network policy, and real managed identity access remain deployment verification requirements.

The service adds --serve-role=all, api, or artifacts. The existing combined role remains the default.
The API role binds only its listener and owns background work.
The artifact role binds only its listener and starts no run, schedule, source, notification, or maintenance worker.
It rejects initial sign-in, user bootstrap, provider, source, recording, and execution configuration flags.
It reads the database's current password-authentication mode without changing that mode.
Grant checks still validate current sessions, credentials, permissions, manifest ownership, and artifact retention.
The artifact listener adds the same process liveness and database readiness routes without exposing a UI, sign-in route, or catalog API.

Shared listener startup opens every selected port before starting background workers.
A failed later listener releases earlier ports and starts no workers. Cancellation stops owned listeners and workers.
Focused Go checks cover failed binding, shutdown, artifact liveness during database failure, and absence of application routes.
A separate focused artifact-role check preserves enabled password mode, reads a retained private log through a password-backed grant, then rejects that grant after mode disabling.
All focused checks pass with race detection. No complete API or browser suite repeats.

Local Bicep 0.48.1 compilation validates provider properties and all local modules without registry restoration or warnings.
Seven packaging checks verify the two roles, origin consistency, ingress, probes, fixed active replicas, identity lifecycle, named-secret permissions, private storage, and digest references.
The generated template has 15 top-level resources with nested app and permission modules.
Evidence is output/azure-package/check/summary.json, compile.log, and service.json.
The first compiler pass rejects runtime-derived resource locations and inline loop expressions. Explicit location input and supported local maps correct those issues.
The checker follows compiler-inlined output expressions rather than assuming every source variable remains in the emitted template.
The verified compiler download metadata remains in .data/azure-tools/download.json.

The required AMD64 build exposes the legacy Docker builder's cross-platform stage and scratch-image failure.
The producer now requires buildx for cross-architecture output and accepts --buildx=PATH for a trusted standalone executable.
It selects the current context's built-in builder, loads the image locally, and creates no additional builder daemon.
The build backend version and fixed certificate-source metadata participate in cache selection.
A stopped container supplies certificates from the pinned image. It never executes. The producer verifies its cleanup and removes the temporary context.
The final Dockerfile uses a single scratch stage with explicit non-root data and temporary directories.
The native fallback remains available for a host-architecture build when buildx is unavailable.
Verified standalone buildx metadata is recorded in .data/docker-tools/download.json.

The final local AMD64 image is sha256:b4f2827a1182b3df1eb8b10b71390f378e988663963083ee9fb1b934e982402c.
This is a local image identifier, not a pushed registry digest.
Four focused container checks verify API-only startup, artifact-only boundaries, persisted sign-in mode, and invalid role rejection.
Evidence is output/azure-package/roles/summary.json and cleanup.json.
Two configuration and temporary-volume checks pass on that same AMD64 image.
Evidence is output/azure-package/configuration/summary.json and cleanup.json.
The role check uses sequential single-process SQLite only for the fresh local fixture. Hosted apps still require shared PostgreSQL.
All owned containers and named or anonymous volumes are removed. No certificate-extraction container remains.
The native ARM64 fallback also builds and passes the two targeted configuration and temporary-volume checks.
Evidence is output/azure-package/native-configuration/summary.json and cleanup.json.
Its local image identifier is sha256:8b6e0cb88876b8ad9be2edfbf320586b2cab691e9a19a1f5effeef88ab829423.
UI, private viewer, and trusted execution runtime assets reuse unchanged outputs.
Each architecture builds only required service binaries. The health command's source remains unchanged.
The native API and preparation worker build. The demonstration refresh preserves its database, evidence, and existing browser work.
No browser, source-runner matrix, AWS, Kubernetes, or Azure deployment or live provider check runs.

The selected Container Apps schema exposes no container read-only root or capability-drop property.
The template uses the non-root image and states that platform limit. It makes no unsupported filesystem or worker-isolation claim.
Hosted preparation, execution, timing, cancellation, deadlines, provider retention, real identity, replica recovery, and full deployment parity remain open.
Next implement common private work-packet and result transport, then isolated provider dispatch and recovery adapters.
See [Azure core service profile](azure.md) for deployment prerequisites and remaining verification.

## Current delivery: ECS and CloudFront core packaging

Acceptance **57** advances. The ledger stays at **48 verified, 9 open, 57 total** items.
The new deploy/aws/service.yaml defines an independent 31-resource CloudFormation core profile.
Explicit inputs cover VPC and subnets, DNS, certificates, immutable ECR digest and architecture, PostgreSQL, identity settings, and the initial administrator.
The stack creates private retained UI and evidence buckets, two CloudFront distributions, a TLS load balancer, private Fargate tasks, and dedicated roles.
It creates no upstream application resources and inherits no MarketMaker identifiers or credentials.

The private UI bucket uses distribution-scoped Origin Access Control.
UI-only route rewriting preserves assets and query strings. API, authentication, readiness, and liveness routes bypass that function.
Private responses disable caching. Separate zero-duration error-cache settings retain upstream status codes and bodies.
Cookies, query strings, and viewer headers except Host reach the API and artifact service.
The artifact distribution cannot read the evidence bucket directly. Existing application grant authorization remains required.
UI content policy permits the separate artifact host without replacing artifact or trace policies.

The load balancer accepts CloudFront HTTPS sources and requires a generated origin credential on both listener routes.
Its default response denies access. Private tasks receive no public addresses or Docker socket.
The non-root service retains a protected root filesystem, no capabilities, a task-scoped temporary mount, separate liveness and readiness, and deployment rollback settings.
The image now declares /tmp as a volume with non-root ownership for ECS mount initialization.
The task execution role reads only selected startup secrets and the selected image repository.
The runtime role accesses evidence objects only within the new bucket's teststudio prefix.
Listing permission applies only to that dedicated bucket so missing-object errors remain distinguishable from denied access.
The operator supplies database reachability, certificates, DNS ownership, outbound routing, and appropriate secret key permissions.

The API adds --oidc-config-env and --artifact-store-env.
Both select bounded 64 KiB JSON through environment names and conflict with their file alternatives.
Missing, blank, malformed, oversized, unknown-field, and trailing-value configurations fail without echoing values.
Filesystem environment settings require an absolute directory. Provider settings accept references, not inline client secrets.
The complete initial provider list validates before database initialization. Invalid or repeated providers cannot leave partial initial identity records.
Existing providers remain unchanged on restart. The initial external administrator's existing one-time bootstrap behavior remains unchanged.

Focused runtime and artifact configuration checks pass with race detection, together with existing API-command checks.
The added provider configuration check covers two providers, distinct references, invalid later entries, duplicate identifiers, empty lists, and credential-bearing documents.
It passes with race detection. The final API command compiles, and the native API and preparation binaries build.
UI, trace viewer, health binary, execution runtimes, and gateway artifacts are reused.

Local cfn-lint 1.46.0 validates the template against two regional provider schema sets.
Seven packaging checks cover cross-resource routing, cache and error behavior, private storage, references, roles, mounts, process health, and the local UI function.
Evidence is output/aws-package/check/summary.json with schema.log and ui-routing.log.
The checker removes AWS environment settings and performs no account or AWS service operation.

The first updated service image passes all 12 focused local package checks.
Evidence is output/aws-package/service/summary.json and cleanup.json.
After complete initial-provider validation moves before database initialization, the final image repeats only the two changed configuration and volume checks.
Evidence is output/aws-package/final-configuration/summary.json and cleanup.json.
The final local arm64 image is sha256:4ab98ab5196c6afbc7112305d58b7dbd2d7622fca7b4e1e3768cb24e08a9b21b.
This is a local image identifier, not a pushed registry digest.
All owned check containers and named or anonymous temporary volumes are removed.
The demonstration refresh preserves its database, evidence, and existing browser work.
No browser suite, source-runner matrix, live AWS, Kubernetes, or Azure verification runs.

Hosted execution and preparation, infrastructure reachability, real identity, private report access, provider retention, replica failover, and deployment parity remain open.
The next package is Azure Container Apps. Isolated provider adapters follow the core packages.
See [AWS core service profile](aws.md) for operator setup, deployment limits, and later verification instructions.

## Current delivery: Kubernetes service packaging and lifecycle probes

Acceptance **57** advances. The ledger stays at **48 verified, 9 open, 57 total** items.
The new Helm chart packages the portable service for a dedicated Kubernetes namespace without inherited cluster, database, identity, or artifact settings.
Its values schema requires an immutable registry image digest, matching architecture, explicit HTTPS origins, existing Secret references, and initial identity configuration.
The API/UI and artifact hosts use separate Services and explicit TLS routes.
The chart references PostgreSQL and OIDC Secrets without creating or embedding their values.
It supports multiple initial providers and approves the explicitly mapped references for later UI provider management.
Bootstrap issuer selection must match one configured provider and names one exact external subject.

The pod runs non-root with a protected root filesystem, private data and temporary mounts, no capabilities, and no automatically mounted API token.
The chart grants no Kubernetes RBAC. Selector labels and the configuration checksum cannot be replaced by custom pod metadata.
Filesystem artifacts use retained private storage. Single-writer storage uses one replica and Recreate updates.
Multiple filesystem replicas require declared ReadWriteMany storage. Shared or object-storage profiles use rolling updates and a disruption budget.
Storage configuration changes replace the relevant JSON object without carrying filesystem-only defaults into Azure or S3 settings.
Ingress policy permits traffic from explicit proxy namespace and pod selectors. Hosted coded worker isolation remains separate work.

A new /livez route separates process liveness from /healthz database readiness.
The focused API check closes storage, observes readiness failure, and confirms continued liveness.
The API and preparation command add --require-postgres to reject a local database fallback before opening storage.
The targeted guard and API checks pass with race detection. The preparation command compiles.

Helm lint and local rendering pass. kubeconform validates 31 resources across four storage and replica renderings against pinned strict Kubernetes 1.37.0 schemas.
Cross-resource checks cover routing, service selectors, secret references, protected pod settings, stateful mounts, probes, multiple providers, and configuration rollout checksums.
Nine invalid configuration cases fail before deployment, including inline credentials, missing immutable digests, shared origins, unmapped references, foreign bootstrap issuers, and unsafe replicas.
Evidence is output/kubernetes-package/check/summary.json and the rendered manifests and schema logs.
Verified local tool versions and checksums are retained in .data/kubernetes-tools/downloads.json.
The verifier uses no kubeconfig and performs no cluster operation. Public schema downloads are local validation inputs.
An initial verifier-only Helm memory-driver setup error is corrected by clearing inherited driver overrides.
No product or cluster operation is involved in that correction.

Only the changed service binary and image build. UI assets, trace-viewer assets, health binary, and trusted test runtimes remain unchanged.
The current service image passes ten focused local package checks, including process liveness and the hosted PostgreSQL guard.
Evidence is output/kubernetes-package/service-2fl_jh9n/summary.json and cleanup.json.
All owned package-check containers and volumes are removed. Rendered files and the intended service image remain available.
No browser, source-runner matrix, Kubernetes cluster, AWS, or Azure test runs.
The chart's namespace, ingress, storage driver, real identity, replica failover, and hosted execution behavior still require their own deployment verification.
See [Kubernetes service profile](kubernetes.md) for operator inputs, commands, and remaining scope.

## Current delivery: portable service image and database environment

Acceptance **57** advances. The ledger stays at **48 verified, 9 open, 57 total** items.
The new deploy/service/Dockerfile packages the API, Angular UI, private trace viewer, and a bounded Go health command.
The final image runs as 65532:65532 on a scratch filesystem with trusted CA certificates and embedded timezone data.
Writable data and temporary storage use separate mounts. No Docker CLI, source bundle, user database, or verification fixture is copied into the service image.

The API and preparation command accept --database-env with explicit environment-name validation.
It conflicts with an explicitly supplied --database, preserves the ordinary local default, and rejects missing or blank selected variables.
The selected database value stays out of process arguments. Its initialization failures exclude private values and upstream diagnostics.
The focused configuration and existing API-command checks pass with race detection. The preparation command compiles.
The health command requires a bounded TestStudio ready response, rejects redirects, and enforces a two-second default deadline.
Focused health response, redirect, and deadline checks pass with race detection.

The new builder selects amd64 or arm64, disables Go workspace mode, preserves read-only module metadata, and compiles static Linux binaries.
Its source key includes the actual compiler inputs, binary hashes, asset hashes, and Dockerfile.
The canonical asset builder now supports --assets-only, so packaging reuses the UI and private viewer without building native workers or the self-test image.
Only the new service and health binaries and changed service image build. Trusted execution and gateway images stay unchanged.
Temporary assembly contexts contain only selected package files and are removed after each build.

The actual local arm64 image passes eight focused checks with a read-only root filesystem, no Linux capabilities, and a private writable volume.
Its API and UI become ready. The separate artifact origin serves no UI. The in-image health command succeeds.
Container recreation preserves SQLite migration state. Project, run, and schedule registries remain empty.
A newly generated malformed database canary stays absent from arguments and startup diagnostics.
The initial verifier exposes a cold SQLite reader-mode issue and case-sensitive Docker cleanup matching.
Both verifier issues are corrected. Exact resource inspection confirms cleanup even for the initial failed observation.
The verified current evidence is output/service-package/isolated-tzx0qcsx/summary.json and cleanup.json.
Earlier equivalent package evidence is output/service-package/local-6focctb1.
All owned verification containers and volumes are removed. The produced service image remains as the intended deliverable.

No browser or cloud checks run. The user's demonstration remains running without a rebuild or restart.
Cross-architecture execution, hosted worker dispatch and recovery, provider secrets, identity, storage lifecycle, and the four deployment profiles remain open.
See [Portable service package](service-image.md) for build, configuration, and verification commands.

## Current delivery: independent product material and provenance audit

Acceptance **02** is complete. The ledger has **48 verified, 9 open, 57 total** items.
The audit combines repository history, fixture and example review, known legacy source comparison, and existing empty-start and independent-build evidence.
The single initial root, beb1c7b, adds five planning and module files. It contains no imported tests, application records, credentials, configuration bundle, or history.
New product tests and fixtures enter through their own feature commits. The first self-test suite enters at 554739d.
The checked-in ZIP fixtures enter at 3b90043 and 8a18764. All ten members contain new generic source, catalogs, manifests, and runtime declarations.
The identity and HTTPS trust examples use sample endpoints, TestStudio references, and operator-selected values.
No tracked credential file, database, report bundle, imported run history, or existing application deployment configuration is distributed.

The read-only audit compares the complete tracked product inventory and ZIP members against nine known legacy checkouts.
References include their current HEAD, a HEAD ancestor before product creation, and 16942 current tracked working files.
No exact or whitespace-normalized matches appear.
The comparison records product content digests, checked source revisions, and explicit exclusions for missing, large, or symbolic-link working files.
A new temporary reference confirms that deliberate exact and whitespace-only overlap fails the audit. Its temporary files are removed.
Evidence is output/provenance/content-comparison.json and review.json.

The automated comparison is supporting evidence. It does not establish authorship or exclude arbitrary rewritten derivations by itself.
Repository origin, newly authored fixture contents, independent configuration, and the passing empty-installation contract provide complementary evidence.
The isolated build evidence remains .data/independent-build._o3cvq4v/result.json. It passes Go and UI builds with no external source directory or local package link.
The normal-start evidence remains output/empty-installation/local-qoUlbQ/summary.json. It reports passed checks and complete process cleanup.
Both existing records are inspected without repeating their builds or tests.
Generated local demo data and later user-selected uploads remain outside the initial repository-distribution audit.
See [Independent product material](product-provenance.md) for repeatable commands, scope, and limitations.
No application suite, browser, runner-image build, cloud request, or deployment runs for this milestone.
The demo remains running. Hosted execution, timing, storage lifecycle, identity, and deployment acceptance remain open under their stable numbers.

## Current delivery: shared SQLite and PostgreSQL storage behavior

Acceptance **06** is complete. The ledger has **47 verified, 10 open, 57 total** items.
The source audit maps the shared store, worker checkpoint ledger, and durable schedule claims to the required storage behavior.
Both selected backends use the same Store contracts and service-level claim records.
No missing storage implementation appears in the audited scope.

The same eleven focused checks pass with race detection on SQLite and local PostgreSQL.
They cover optimistic revision conflicts, one winning concurrent writer, and atomic rollback without leaked records.
Bound transactions read their own writes and commit or roll back sequence changes with metadata.
Swallowed failures cannot commit partial work. Stale revision assertions reject dependent changes without changing asserted records.
Scoped ascending and descending pages preserve boundaries and cursor ordering. Cursor continuation survives a real close and reopen.
Concurrent schema startup converges on usable storage. Concurrent installation identity creation returns one durable identifier across reopened connections.
Worker checkpoints retain claimant tokens, results, evidence state, cleanup failure, retry state, and queue removal.
A stale worker cannot finalize another claimant's checkpoint.
Competing schedule replicas create one normal logical run through one atomic firing transaction.
Live claims remain fenced. Expired claims transfer after reopening, preserve one firing, and coalesce missed occurrences without duplicate runs.

SQLite storage, ledger, and run checks take 1.554, 1.537, and 2.094 seconds respectively.
Local PostgreSQL checks take 1.912, 1.642, and 2.095 seconds respectively.
Evidence is output/storage-acceptance/summary.json, sqlite.log, and postgres.log.
The owned PostgreSQL container is verified absent. No browser or test-runtime image is needed.
No broad product suite or cloud profile runs. The user demo remains running without a rebuild or restart.
Existing identity restart, installation transfer, preparation recovery, and evidence restart acceptance remains valid without repetition.
SQLite remains the single-instance local default. PostgreSQL remains the hosted and multi-instance default.
Deployment-specific execution, timing, identity, and provider lifecycle acceptance remain open under their own ledger numbers.

## Current delivery: complete scoped grant and revocation coverage

Acceptance **12** is complete. The ledger has **46 verified, 11 open, 57 total** items.
A source audit maps every required grant dimension to existing permission-service and managed API behavior.
No missing behavior or permission bypass appears in this focused scope. No new authorization implementation is needed.

Nine existing focused checks pass with race detection on SQLite and local PostgreSQL.
They cover user and group grants, custom and built-in role assignments, inherited object rules, and exact project bindings.
Explicit user and group denies override applicable direct and inherited allows.
Expired denies retain their administrative records without affecting current access. Expired allows withdraw access.
Object grants apply only to the exact resource. Project grants do not cross into another project.
Inherited parent and child disabling withdraw access. Removing the last inheritance path withdraws access while alternate paths remain valid.
Exact assignment revocation reaches an already-created independent evaluator and preserves another project's binding.
Stale assignment removal fails after recreation. Grant, policy, and assignment changes retain atomic audit and revision behavior.
Read-only and exact-object denials also control managed grant reads, mutation capabilities, and list filtering.
All access decisions continue through wtiger001/go-permissions v0.0.31.

SQLite authorization checks take 1.668 seconds. SQLite API checks take 8.480 seconds.
Local PostgreSQL authorization checks take 1.839 seconds. Local PostgreSQL API checks take 6.137 seconds.
Evidence is output/access-acceptance/summary.json, sqlite.log, and postgres.log.
The owned local PostgreSQL container is verified absent after the check.
No browser, UI build, runner image build, source fixture, broad matrix, or cloud check runs.
The user demo stays running. See [Grant acceptance coverage](access.md#grant-acceptance-coverage) for the behavior map.
Storage parity, provenance, timing profiles, artifact-provider lifecycle, and hosted acceptance remain separate open requirements.

## Current delivery: complete repository publication through the UI

Acceptance **22** is complete. The ledger has **45 verified, 12 open, 57 total** items.
Declarative JSON and YAML uploads, complete Go and Playwright ZIPs, and registered GitHub sources produce immutable releases through the UI.
The existing source, native-upload, ZIP, isolated assembly, publication, and execution evidence remains valid.
The final gap closes with one actual local GitHub-to-release browser journey.

The browser registers a repository through Repository, Suites, and Review.
The production GitHub adapter resolves a fixed commit from a new local TLS provider fixture and downloads its archive once.
A separate real preparation worker assembles and discovers the Go suite from that exact snapshot.
The inspected preparation now updates automatically while work remains active. An explicit Refresh repository preparation action remains available.
Current project, detail, and job checks discard stale responses. Access loss uses the existing private-state clearing path.
The browser reviews source provenance and catalog changes, publishes with explicit activation, and reads the retained source.
Registration, synchronization, preparation, and publication create no test run.
One later explicit run executes that immutable Go release and retains one passed logical result with complete evidence.
The authoring journey itself runs through TestStudio's normal catalog, prepared Playwright source, environment gateway, timing, results, and evidence contracts.
The browser and Go executions share two local execution slots to avoid waiting on their own worker slot.

The initial execution finds a blank source page on an ordinary HTTP hostname.
The browser omits crypto.randomUUID outside a secure context. Existing generated identifiers and request keys previously call it directly.
A shared randomId helper now uses native randomUUID when available and secure getRandomValues otherwise.
The fallback preserves UUID version and variant bits. It never uses weak random values.
All existing UI UUID call sites use the helper. Native generation, the secure-byte fallback, and explicit missing-randomness failure pass focused checks.
The actual HTTP browser journey verifies the fallback under the original failing origin.

The Angular production build passes. The final integrated check passes in 30.794 seconds.
The UI observation takes 15909 milliseconds. Its separate Go result retains a valid zero-millisecond measured duration.
Evidence is output/playwright/repository-ui-zzqhf0me/summary.json, resources.json, and cleanup.json.
Review and retained-source screenshots receive visual inspection. Logs and traces remain private local evidence.
Owned browser and Go containers, gateways, volumes, both prepared source images, and private contexts are verified absent.
The verification process closes its API and TLS listeners, stops source and run workers, and revokes its newly created test-only session.
The initial failed execution also completes owned-resource cleanup.

Previously retained native upload evidence is output/selftest/20261103T112052.874370000-12768173eed23e629c9e7309/summary.json.
Previously retained Go and browser ZIP evidence is output/selftest/20261103T164151.211223000-fbde814a60efc9df0c742ab8/summary.json.
Both records remain completed, passed, and evidence-complete. They do not repeat.
The actual Go ZIP UI journey is retained under .data/bundle-ui-verification.Eh6NWc.
Its recorded browser, controller, worker, image, and context cleanup remains valid.
Supported dependency and execution profiles remain explicit. Arbitrary uploaded installers do not execute in the API.
No live GitHub or cloud requests run. Cached trusted Go, Playwright, and gateway images remain unchanged.
Only the changed UI and small prepared verification source images build.

## Current delivery: guided repository configuration

Acceptance **22** advances and remains open. The ledger stays at **44 verified, 13 open, 57 total** items.
The repository editor now uses the shared Repository, Suites, and Review stepper pattern.
Source identifiers generate from the name. Permanent identifier and reference-type controls move under Advanced repository options.
Periodic and signed push settings move under Advanced synchronization options.
Review shows repository identity, reference, access reference, all suite bindings, and automatic sync settings.
Enable state and automatic publication remain visible with future activation and execution consequences.
Automatic publication remains off by default. Synchronization never starts tests.

Back and completed step navigation preserve entries. Each suite can appear once.
Invalid hidden settings open and receive keyboard focus. Opening the editor focuses its first step heading.
The configuration uses feature-service state and pure validation utilities. No API or runner contract changes.
The page help and repository guide describe the same steps.

The Angular production build passes. STUDIO-GIT-001 passes in 2448 milliseconds through TestStudio's normal runner.
It covers generated identifiers, duplicate correction, review, hidden periodic validation, Back preservation, editing, durable sync queueing, Help, and mobile dark layout.
The first focused run detects missing initial focus. The editor corrects it without changing shared route focus behavior.
The failed and successful runs both complete owned execution cleanup.
Evidence is output/selftest/20261104T075856.861520000-f6ff1ec18e3235ac266f3a31/summary.json and cleanup.json.
Light desktop and dark mobile screenshots receive visual inspection.
Owned browser containers, volumes, and verification controllers are absent after completion.
Only the changed UI and self-test image build. Existing service binaries, runtime dependencies, and the private trace viewer are reused.

The remaining acceptance 22 step is the complete registered repository sync, preparation review, and UI publication journey against a local provider fixture.
Existing native-upload, ZIP, source-worker, and publication evidence remain valid. Do not repeat unchanged runner fixture builds.

## Current delivery: manual and scheduled runner parity

Acceptance **37** is complete. The ledger has **44 verified, 13 open, 57 total** items.
Manual creation and scheduled firing use the same admission, frozen selection, execution queue, worker, and completion path.
No separate scheduled runner or result format is introduced.

A focused contract matrix covers targetless Go and Playwright runs with passed, failed, timed-out, flaky, and skipped results.
Each trigger preserves the release, selected definitions, capture policy, targetless context, timeout, retries, and coded work item.
Prediction timestamps remain specific to each run. Comparable timing contexts and criteria remain equal.
Attempts and child steps produce one logical result per test. Both triggers retain readable logs, normal history, and trend counts.
The race-enabled matrix passes in 6.906 seconds.

One actual recorded website source prepares once and executes manually and through the durable scheduler.
Both runs use the same registered target and immutable source release through the current private gateway protocol.
Each run retains one passed logical test, measured timing, child steps, logs, one screenshot, and one trace.
Stored checksums match readable evidence. Screenshot decoding and trace archive inspection pass.
Normal history contains two entries. Trends contain exactly two passed observations.
The workflow passes in 23.095 seconds. Cached trusted runtimes and the gateway are reused.
Evidence is output/playwright/scheduled-website-tyizhg7i/summary.json and cleanup.json.
Owned containers, volumes, preparation contexts, and the prepared source image are verified absent.
The verification worker and fixture listener stop with the test process. The user's demo remains running.

Existing native HTTP and HTTPS evidence under acceptance 34 already verifies both triggers.
Existing guarded Go and Playwright API evidence under acceptance 35 verifies both triggers and normal history and trends.
Those unchanged workflows do not repeat. No cloud checks run, and no demo rebuild or restart is needed.
Hosted execution and timing parity remain separate open requirements under 41 and 51 through 57.

## Current delivery: guarded Go API execution and acceptance 35

Acceptance **35** is complete. The ledger has **43 verified, 14 open, 57 total** items.
The later manual and scheduled parity checkpoint closes acceptance **37**.

Go API suites now use go-environment, the registered endpoint policy, and the same trusted gateway as Playwright API suites.
Standard net/http clients use the loopback relay supplied to the selected test process after compilation.
The worker retains network=none, read-only source, bounded resources, and isolated dependency resolution.
The Go worker checks its prepared request rules, selected test, and authentication against frozen input before execution.
Go unit suites keep their targetless go-module profile and go-local-isolated-v1 identity.
Environment Go execution uses go-local-environment-v1 and joins the existing recovery and owned-resource ledger.

A complete new stdlib-only workspace is available in examples/go-api.
It prepares from a ZIP or repository manifest and uses normal publication, registered selection, results, history, and stacked-area trends.
Source and catalog validation accept Go environment profiles only with API type and explicit request rules.
The source JSON Schema now matches those profiles and no longer rejects valid API authentication bindings.
Draft 2020 checks validate normal and authenticated Go profiles and reject mismatched runner, profile, and missing-policy cases.

Go event evidence uses bounded JSON lines. The collector decodes event output before sensitive-field and reflected-value redaction.
Malformed or truncated event streams produce explicit capture gaps while leaving the observed state and measurements unchanged.
The existing private viewer treats these logs as text. Focused pure UI checks preserve other artifact previews and timeout reruns.
The common gateway initialization packet now carries the target policy and optional credential through private stdin and tmpfs.
No target policy or credential needs a process argument or Docker environment variable.
The packet and runtime work item each have a 4 MiB limit. Updated images declare private-config/v1 and the 4194304-byte runtime contract.
Legacy images fail API admission with an explicit upgrade requirement.

Targeted relay, frozen-input, redaction, credential-channel, image-capability, and artifact-preview checks pass locally.
The first real Go API run passes in 35.122 seconds. Its functional checks need no further repetition.
The final contract audit identifies the older 64 KiB input limit and operating-system argument limits for large reviewed policies.
The updated bounded channel and runtimes pass a second focused run in 35.701 seconds.
This final fixture uses all 64 request rules, long prefixes, and an encoded work item larger than 1 MiB.
It verifies authenticated manual and scheduled execution, policy denials, body limits, request budgets, disabled redirects, and direct-network and CONNECT denial.
The verification credential is generated outside the source image. Reflected values are removed before the test receives its response.
Retained logs contain no private canaries or credential values. Each run retains one passed logical result and its Go child steps.
History and trends contain exactly two passed tests. Only six permitted requests reach the newly created local fixture.
Evidence is output/playwright/go-api-large-policy-ozilpklo/summary.json and cleanup.json.
All owned source images, preparation resources, worker containers, gateways, volumes, listeners, and verification credential files are verified removed.

Only changed runtime and gateway images build. The passing authoring UI suite and unrelated functional suites do not repeat.
The demo accepts an immutable --go-runtime image, retains that choice, and enables Go preparation without reseeding existing data.
No cloud profile runs. See [Go API execution](go-testing.md#guarded-go-api-execution) for setup and current boundaries.
Continue the remaining scheduling parity, storage parity, access matrix, source workflow, artifact adapter, timing, and hosted deployment items.

## Current delivery: project API credentials through the trusted gateway

Acceptance **35** advances and remains open. The ledger stays at **42 verified, 15 open, 57 total** items.

Operators can register project-scoped api-credential references with exact approved origins and public or private network scope.
Environment registration maps up to 16 unique binding names to current project references under Advanced credentials.
The API validates project, purpose, identifiers, and permitted endpoint origin before saving mappings.
The new reference endpoint requires environments.manage and returns no secret value or source location.
The API authoring wizard supports Bearer, Basic, and reviewed API-key headers. Review retains binding names without credential values.
Manifest and catalog declarations must match. Missing-credential tests can explicitly omit their test-level binding.
Browser session credentials, native HTTP-check authentication, and environment-backed Go clients remain separate requirements.

The execution host resolves a selected reference before each attempt. Provider errors exclude private diagnostics.
Only the trusted gateway receives the resolved header through Docker exec stdin and a one-use private tmpfs socket.
The socket stays outside the worker volume and is removed after initialization.
Work-item JSON excludes resolved values and the selected secret reference. Stored runs retain only their environment reference mapping.
HTTPS is mandatory outside explicitly local environments. Network permission is the intersection of environment and credential scope.
The gateway injects only after method, path, payload, and origin admission. Redirects remain disabled.
Literal and common encoded credential reflections are removed before the worker sees upstream response headers or bodies.
The collector also removes known values. Recovery therefore needs no persisted earlier secret or current-secret resolution.
Authenticated admission requires the gateway-injection/v1 image capability.

Focused race checks pass for reference scope, private errors, HTTP rejection, operator origin validation, permissions, binding declarations, and configuration socket cleanup.
Gateway checks pass for Bearer, Basic, and API-key injection, spoofed-header replacement, off-origin rejection, reflection scrubbing, and missing authentication.
A public-only credential cannot inherit private network access from its environment.
Three pure UI checks pass for reference origin selection, unique bindings, request validation, and generated source declarations.
OpenAPI 0.32, the source schema, and generated client stay aligned.

The first real worker check finds a remaining legacy inspection rejection of all bindings.
Source inspection now accepts only validated API authentication bindings. Targetless and unsupported bindings remain rejected.
The focused worker check then passes in 24.404 seconds using one changed gateway image and the unchanged source runtime.
Manual and scheduled executions each retain one passed API result, normal history and trends, and redacted private evidence.
Evidence is output/playwright/api-credentials-re8kk9jt/summary.json and cleanup.json.
All owned preparation resources, source images, workers, gateways, and volumes are verified absent.

STUDIO-API-AUTHOR-001 passes in 2315 milliseconds through TestStudio's normal self-test path.
It creates an environment with an approved reference, rejects a different origin with Advanced focus, preserves review values, and saves an authenticated API draft.
The first UI check completes environment registration, then uses an unsupported single-environment read in its assertion.
The corrected assertion uses the existing environment list. Only this focused workflow repeats.
The narrow dark editor screenshot is inspected. Its worker, browser processes, volumes, and controller are verified absent.
Evidence is output/selftest/20261104T065542.477028000-af6b4329874072ad68068ed0/summary.json and cleanup.json.
No broad suite or cloud profile runs. See [API credentials](api-credentials.md) for setup, recovery, and limits.

Next implement environment-backed Go direct clients through the same reviewed request transport.
Then verify the remaining manual and scheduled runner parity before closing **35** or **37**.

## Current delivery: retained API diagnostic redaction

Acceptance **35** advances and remains open. The ledger stays at **42 verified, 15 open, 57 total** items.

API-typed coded tests now apply api-diagnostics/v1 at the trusted execution collector.
The collector sanitizes assertion messages, steps, status text, logs, evidence warnings, and supported attachments before checkpoint and upload.
Worker recovery uses the frozen test type and source policy through the same collector.
Default sensitive fields, authorization headers, cookies, URL user information, and sensitive query values are removed.
Detected values are also removed from reflected diagnostic text. Exact known-value replacement supports later credential resolution.
The manifest accepts up to 32 additional sensitive field names. The schema, OpenAPI 0.31, and generated client share this field.

Only bounded UTF-8 text and complete JSON enter API attachments. Invalid JSON and opaque files produce explicit evidence gaps.
The request runtime disables automatic media. The reporter rejects API media before copying it; the collector independently rejects it before transfer.
Browser media capture remains available. Each API diagnostic input stays within 2 MiB and the configured capture budget.
JSON preserves large numeric values and enforces a 64-level nesting limit.
The result records diagnosticsPolicy. Retained worker evidence records immutable redactionPolicy metadata.
Previously stored files remain unchanged and receive no retroactive guarantee.
The policy does not detect deliberately encoded or unlabeled private data in trusted published code.

Targeted race checks cover structured and escaped fields, headers, queries, reflected values, known values, bounded diagnostics, and unchanged timeout measurements.
The recovery collector check verifies sanitized logs and JSON without copying a declared private trace.
Reporter checks verify pre-copy omissions. A focused evidence check rejects policy relabeling during upload replay.
One changed runtime and gateway build supports the real local API check. No UI browser suite or cloud profile runs.
The first harness assertion incorrectly requires the worker policy marker on trusted scheduler event logs.
Its corrected assertion scopes the marker to worker diagnostics while checking every retained test artifact for private values.
The focused workflow repeats using the already built runtime and gateway.

The production UI and local server builds pass. No browser UI suite repeats for documentation changes.
The real worker check passes in 12.745 seconds. Manual and scheduled runs retain one passed API result each.
Retained logs and JSON contain none of the new private canaries. Unsupported binary evidence produces explicit incomplete-capture warnings.
Normal history and trends retain exactly two passed tests. Six permitted requests reach the new local fixture.
Evidence is output/playwright/api-diagnostics-q4u01s53/summary.json and cleanup.json.
Cleanup verifies all owned worker containers, gateways, volumes, and prepared source images are absent.
See [API diagnostic redaction](api-testing.md#diagnostic-redaction) and [API evidence](evidence.md#api-diagnostic-evidence).

Continue credential bindings, environment-backed Go clients, and remaining cross-runner scheduling parity before closing **35** or **37**.

## Current delivery: API authoring wizard and real local request runs

Acceptances **35** and **37** advance and remain open. The ledger stays at **42 verified, 15 open, 57 total** items.

Create test now includes API test on its dedicated type route.
New API draft uses the shared Name, Request, and Review stepper before opening the canonical source editor.
The wizard configures a title, generated identifier, request method, relative path, expected status, optional JSON, and reviewed request policy.
Named Advanced sections contain endpoint, path-prefix, request-count, and payload limits.
Review shows target-policy consequences and the separate needs-review safety label before source generation.
Request JSON becomes literal application/json data. UI values do not become executable expressions.
The default catalog uses empty arrays for optional lists.

Back and Help preserve entries. Invalid Advanced fields open and receive focus.
Unfinished wizard setup and source edits participate in navigation and reload protection.
Source creation, Save API draft, preparation, publication, and execution stay separate.
The shared editor retains complete API workspaces and their explicit manifest policy.
Canonical request validation now accepts one meaningful trailing slash without permitting repeated separators or dot segments.

A newly authored API suite completes real protected manual and scheduled runs through the capable local worker and gateway.
It checks read-only GET and POST, denied methods and paths, body limits, method overrides, redirect suppression, direct CONNECT rejection, and request budgets.
Only six permitted fixture requests arrive across both runs. Each run retains one logical API result and its request steps.
Normal retained history and stacked-area trend data count exactly two passed tests across those runs.
The functional assertions pass. The first harness exit reports a second deletion of an image already removed by builder cleanup.
Teardown now treats verified image absence as success. Exact Docker checks confirm all owned containers, volumes, and the source image are absent.
The successful execution is reused; no duplicate fixture build or API rerun is needed for that teardown-only correction.
Evidence is output/playwright/api-worker-teb7vb1l/summary.json and cleanup-rechecked.json.

STUDIO-API-AUTHOR-001 passes in 1467 milliseconds through TestStudio's normal self-test contracts.
It checks request configuration, Advanced focus, policy review, preserved values, Help, canceled navigation, saved code, manifest and catalog contents, and narrow dark layout.
Final verification also checks JSON review and focus transfer from the wizard to the source editor.
The initial UI check times out because an empty generated identifier exposes Advanced details before title entry.
A valid initial identifier fixes that interaction. Only the changed UI workflow repeats.
Evidence is output/selftest/20261104T061646.879778000-8c7e84b6d2df5172d5a211ed/summary.json and cleanup.json.
The final mobile screenshot is inspected. The owned controller, browser worker containers, and volumes are verified stopped and absent.
Pure generation and validation checks pass. The production UI and native server build pass.
The changed gateway builds once; the unchanged source runtime is reused. No cloud profile runs.

API, creation, editor, self-testing, and contextual help now describe the guided workflow and its limits.
Automatic diagnostic redaction, credential injection, environment-backed Go clients, and remaining cross-runner schedule parity stay required.
Continue those requirements before closing **35** or **37**.

## Current delivery: coded API request-policy foundation

Acceptance **35** advances and remains open. The ledger stays at **42 verified, 15 open, 57 total** items.

API-typed Playwright suites can declare a bounded request policy inside the existing environment execution artifact.
The trusted gateway enforces canonical paths, methods, complete request-body limits, response limits, request counts, and four-request concurrency.
API mode rejects CONNECT tunnels, protocol upgrades, compressed payloads, request trailers, and common method overrides under read-only rules.
It establishes upstream HTTP or verified TLS itself. The network-isolated worker uses a local relay and cannot obtain an unrestricted tunnel.
Redirect statuses remain visible. The gateway withholds Location and replaces redirect bodies with a fixed policy message.
The first redirect fixture exposes a destination URL in the original HTML body; the replacement fixes that disclosure.
HEAD retains its upstream content length. Nonredirect responses retain bounded assertion data.

Protected API execution requires a valid reviewed read-only policy in addition to safe test and dependency labels and current permission.
Later protection changes recheck that requirement at attempt admission.
Request restrictions participate in the immutable prepared artifact and timing definition.
Both runner and gateway require the explicit request-policy/v1 image capability. Admission and timing reject incompatible older images.
The worker compares its prepared manifest policy with the frozen request input, including defaults for optional rule fields.

The new starter endpoint accepts type=api and returns a complete workspace with a direct request fixture, reviewed rules, and needs-review safety.
It saves and queues preparation through the existing exact-revision source contracts without starting runs.
OpenAPI 0.30, the suite JSON Schema, and the generated client describe the policy and starter selector.
Default browser starter behavior stays available.

Targeted race checks pass for actual local gateway HTTP and TLS traffic, policy validation, compatibility, protected admission, starter API, and source snapshots.
The existing origin-only browser gateway regression passes in the initial focused batch.
The generated client check passes. Node worker syntax and the production UI build pass.
The changed source runtime and gateway images build once. Both actual image capability labels are verified.
The gateway image includes its pinned standard CA file. The local demo restarts with retained data.
These checks do not prove complete API worker execution or the dedicated authoring UI.
No browser suite or cloud verification runs in this batch.
Evidence is output/api-policy/checkpoint.json.
See [Coded API request policies](api-testing.md) for current behavior, operator setup, and limits.

Continue with the API choice and canonical editor starter wiring, then one targeted real API worker flow with the new images.
Complete automatic diagnostic redaction, credential bindings, environment-backed Go request clients, and remaining scheduled parity afterward.
Keep acceptance **35** open until its complete guarded execution and normal history and trends scope passes.

## Current delivery: private HTTPS trust and scheduled response history

Acceptance **34** is complete. The ledger has **42 verified, 15 open, 57 total** items.
The complete plan still requires native HTTP authentication bindings, bounded permitted redirects, and redacted response excerpts.
These requirements remain follow-up implementation scope even though the stable manual, scheduled, assertion, TLS, and history criterion now passes.

Native HTTP execution now receives its explicit project context.
The operator can configure certificate-authority bundles for one project and exact HTTPS origin through --http-trust-bundles.
Private trust does not enable private networking. Local stage or an explicit private-network policy remains required.
Trust configuration accepts bounded regular CA files and rejects private keys, leaf certificates, empty bundles, and trailing data.
Bindings preserve platform roots, hostname verification, certificate dates, TLS version limits, and configured minimum certificate lifetime.
The same CA cannot grant trust to another project, origin, or port.
CA material and filesystem paths remain outside result and browser configuration.

The selected trust identifier and certificate-set fingerprint form part of the frozen native execution profile.
Certificate order, repeated certificates, and formatting do not change that identity.
A changed bundle blocks queued dispatch with a readable trust-change reason. Newly created runs use separate compatible timing history.
HTTP diagnostics retain and show the trust name, identifier, and SHA-256 fingerprint beside the observed certificate and timing data.
Unavailable certificate information remains unavailable. Browser execution and provider trust configuration remain separate.

New local authorities and endpoint certificates verify trust boundaries, hostname mismatch, expiry, minimum lifetime, and private-network policy.
Real manual and scheduled HTTPS runs pass status, text, header, JSON, response-time, and certificate-lifetime assertions through the normal executor.
Both runs retain one logical result, complete evidence, and comparable response-time samples.
After trust rotation, the queued run remains blocked without another request. A fresh run succeeds with the new profile.
Focused race checks pass on SQLite and local PostgreSQL. Owned TLS fixtures stop through test teardown.
The exact PostgreSQL fixture and its launcher are verified absent.
The HTTP API and installation-transfer fixtures compile after the internal executor signature change.

The production Angular UI and native server build pass. The unchanged trace viewer remains cached.
The existing STUDIO-HTTPS-001 UI result is inspected and passes in output/selftest/20261104T001333.956188000-718375f9a5a5512ad70e2294/summary.json.
That retained run also contains a separate, earlier URL-focus failure. This evidence supports the HTTPS workflow result, not the whole historical run.
No browser suite, runner image build, cloud test, or external website runs for this delivery.
Evidence is output/http-trust/checkpoint.json and postgres.log.
The HTTP, backup, self-testing, operator example, and in-app run help describe trust scope, rotation, and retained diagnostics.
Continue required HTTP follow-ups, API and scheduled runner parity, storage, access, source, and provider packaging work.

## Current delivery: atomic admission and current dispatch safety

Acceptances **28** and **38** are complete. The ledger has **41 verified, 16 open, 57 total** items.
Acceptance **37** advances with matching actual manual and scheduled HTTP dependency runs. Other runner parity remains open.

Run creation now guards the reviewed suite, release, environment, and every selected working-test revision with the queued run.
The project guard remains in place. Saved selection and schedule checkpoints share the same transaction.
Conflicts revalidate manual requests with bounded retries. Scheduled conflicts retain their atomic firing behavior.
Large selections use the existing multi-batch transaction rather than exceeding the 100-change batch limit.

Dispatch validates the complete frozen dependency closure, including missing, cyclic, disabled, and duplicate selections.
Current lifecycle and safety restrictions remain authoritative without replacing saved definitions.
Native HTTP attempts now recheck current stage, network permissions, credential bindings, execution locations, and suite application links.
Browser execution retains its stricter origin check. Native HTTP endpoint edits retain their original frozen address.
A suite that newly requires a target blocks a previously targetless run.
Protection uses both the frozen and current environment state. Current permissions and published and working safety labels apply throughout the selection.
Blocked retries retain their first recorded observation and create no invented additional attempt or logical passing result.

Focused race checks pass on SQLite and local PostgreSQL for concurrent admission edits, current target policy, protected roots and dependencies, and large guarded selections.
The tests verify successful safe manual and scheduled dependency execution through the actual local HTTP executor.
They also verify atomic firing, normal notification completion, unchanged frozen snapshots, and the existing native endpoint snapshot contract.
Additional SQLite cases verify both directions of published-versus-working dependency safety changes.
Missing, cyclic, disabled, and duplicate frozen dependencies fail explicitly before admission guards are issued.
Retry cases preserve one failed attempt after safety changes, disablement, and protected-permission revocation.
The first environment-boundary fixture tries to change a permanent application reference. The test now uses a supported suite application-link edit.

The native server and production UI build pass. Help text changes require one UI build; the private trace bundle stays cached.
No browser suite, runner image build, or cloud verification runs for this batch.
Owned PostgreSQL fixture cleanup is verified. New local HTTP fixtures stop through test teardown.
Evidence is output/dispatch-safety/checkpoint.json and its focused PostgreSQL retry log.
Run help, catalog, result, and schedule guides explain blocked dispatch, frozen settings, and recovery.
Continue remaining local storage, access, source, HTTP, API, and scheduled runner parity work.

## Current delivery: provider credential error boundaries

Acceptance **20** is complete. The ledger has **39 verified, 18 open, 57 total** items.

OIDC discovery and credential resolution now return fixed errors without upstream bodies or resolver diagnostics.
A configured reference must resolve to a nonempty credential. Empty resolution blocks sign-in without a login transaction.
Existing token exchange and verification failures keep their fixed messages. Browser routes retain readable generic sign-in outcomes.
The service resolves credentials on the server and supplies them only to the token endpoint.
Provider records, browser choices, redirects, cookies, sessions, and audit records contain no resolved value.
Authentication routes do not log provider response bodies or resolved credentials.

New local canary checks cover discovery body and transport failures, resolver failures, empty resolution, successful token exchange, and rejected exchange.
The rejected provider deliberately echoes private diagnostic text. Returned errors and browser responses exclude that text.
OAuth client authentication can use either the supported server-side header or form mode. Neither mode sends a secret through the browser.
Focused API checks also verify private provider administration and readable callback outcomes.
The race checks pass. The first rejected-exchange fixture assumes header-only authentication; it now permits the supported server-side form fallback.
No cloud profile or browser suite runs. Owned HTTP fixtures stop through test cleanup.
The native server builds successfully. UI and trace builds stay cached.
Evidence is output/provider-credentials/checkpoint.json.

See [Provider credentials](providers.md#server-credentials) and [Self-testing](self-testing.md) for setup and focused checks.
Continue remaining local storage, access, catalog, and execution acceptance work.

## Current delivery: empty installation and selected examples

Acceptance **03** is complete. The ledger has **38 verified, 19 open, 57 total** items.

Normal server startup and restart create no projects or testing registries.
Initial authentication and built-in access setup remain separate from project, application, suite, environment, test, run, and schedule records.
The new optional examples/http/catalog.json contains independent native HTTP definitions without endpoints, credentials, or history.
The first-installation guide explains selecting definitions, reviewing the source, explicit publication, and a separate explicit run.
The existing full demonstration remains an explicit command with its own database and complete selected example package.

The focused real-server check starts three owned server processes against fresh local SQLite data.
It verifies empty startup and restart, empty project registries, preview without installation, one selected published definition, and no automatic run.
A final restart preserves exactly that selection. No demo project or self-test catalog appears.
All three owned processes stop and their identifiers are verified absent.
No browser, Docker container, fixture build, or cloud verification runs for this batch.
Evidence is output/empty-installation/local-qoUlbQ/summary.json.
The existing demonstration remains available with its retained data.

See [First installation](first-installation.md) and the focused command in [Self-testing](self-testing.md).
Continue the remaining storage, authorization, and execution contracts without broad regression repeats.

## Current delivery: installation backup and database transfer

Acceptance **08** is complete. The ledger has **37 verified, 20 open, 57 total** items.

The new teststudio-data operator tool exports private installation archives and restores them into an empty SQLite or PostgreSQL database.
Public identifiers, raw metadata, revisions, timestamps, immutable runs, audit history, and artifact references remain unchanged.
Typed native permission conversion preserves grant identifiers, inherited group access, exact denies, disabled roles, and sequence high-water marks.
Password hashes and server-side sessions remain usable after restoration.

Export opens SQLite read-only and uses a consistent PostgreSQL snapshot. Active execution, preparation, and recordings block export.
Private archives publish atomically without overwriting existing output. Artifact bytes and runtime configuration remain separate cutover responsibilities.
Import requires an empty destination. A checksum, ordered sections, schema versions, and exact counts protect transaction completion.
Corrupt or truncated input rolls back installation data. Existing destination data remains intact.

Focused local checks pass for SQLite restoration, SQLite-to-PostgreSQL transfer, PostgreSQL-to-SQLite restoration, access, and retained evidence.
A missing copied artifact stays explicitly missing without changing the original test outcome.
The actual operator command builds and its help works. Existing storage transaction checks pass after the read-snapshot refactor.
No browser suite, fixture image rebuild, or cloud deployment runs for this operator-only change.
Both owned PostgreSQL fixtures and their launcher processes stop and are verified absent.

Evidence is output/installation-transfer/checkpoint.json and the focused transfer logs under .data.
See [Installation backup and database transfer](data-transfer.md) for cutover commands, storage copying, credentials, and format limits.
Continue remaining storage parity and local acceptance work. Hosted profile verification remains separate.

## Current delivery: coded timing and logical count parity

Acceptance **36** is complete. Acceptance **41** advances and remains open across deployment profiles.
The ledger has **36 verified, 21 open, 57 total** items.

Go, Playwright, and API-typed coded tests now recheck runtime compatibility before each attempt and managed dispatch.
Runtime drift blocks execution before another test starts. Original predictions remain unchanged; a new run selects the current history.
Runtime contexts include inspected operating system and architecture independently of the declared toolchain label.
Unlabelled approved Playwright images use their actual immutable local image identifier instead of a shared unknown profile.
Known compatible labels retain history across source-only releases. The self-test image declares its runtime and explicit build architecture.

Unavailable measurements stay censored even when the functional result passes. They cannot train history or become measured zero values.
Actual zero measurements now enter clean history. Zero-median predictions retain the configured absolute noise guard and historical tail.
Regression messages avoid dividing by zero. A configured zero noise guard does not make a zero measurement regress against itself.
Preparation and worker coordination remain separate from execution duration.

Focused race checks pass for frozen adaptive budgets, slowdown retries, unavailable measurements, real zeros, runtime drift, and separate contexts.
They pass on SQLite and local PostgreSQL. Core timing checks and the zero-noise edge check pass.
A direct local image inspection verifies the actual Docker metadata format and platform binding without starting another browser.
The first coded zero check exposes the old sample filter. Both recording and prediction now retain valid zero observations.

New API-typed Playwright self-testing preserves each API operation and assertion as a step within one logical result.
The registrar respects the declared API type instead of forcing every definition to browser.
STUDIO-API-STEPS-001 passes in 415 milliseconds with 64 recorded steps and one result.
STUDIO-TIMEOUT-001 passes in 2958 milliseconds with 96 recorded steps and one result.
Their normal TestStudio run has two registered tests and exactly two completed logical results.
Focused Go runner checks execute a newly written aggregate and an independent subtest. Package and ancestor events remain diagnostics.
Coded retry checks retain two attempts and three steps per attempt while producing one flaky logical result.
The same shared reduction applies across Go, browser, and API-typed coded results.

Evidence is output/selftest/20261104T042513.552815000-000b05a7679da65d8998edf6/summary.json and cleanup.json.
The concise milestone record is output/coded-timing/checkpoint.json.
Owned browser containers, output volumes, controller, and exact PostgreSQL fixture are stopped and verified.
The final native server build passes. UI and trace outputs remain cached after the help update build.
No broad suite or cloud profile runs. The user's demo data and browser remain intact.

Timing, Go, result, and self-testing guides describe runtime drift, valid zero values, and logical count boundaries.
Run details help explains the changed timing and recovery behavior.
Acceptance **41** still requires the applicable deployment profiles. Provider storage acceptance remains open under **46**.
Continue remaining local acceptance work and provider packaging without treating local checks as hosted verification.

## Current delivery: provider artifact storage selection

Acceptance **46** advances and remains open. The ledger stays at **35 verified, 22 open, 57 total** items.

Added filesystem, S3, and Azure Blob selection through the operator's --artifact-store-config JSON file.
The default filesystem startup remains independent of cloud credentials. Selected profiles reject mixed settings and credential-bearing fields.
Relative local directories resolve beside their configuration file. Local evidence remains outside the public UI directory.
AWS credentials use the configured SDK chain. Azure uses its default credential chain or an explicit managed identity client identifier.
Provider resources and credentials stay outside catalog, run, and UI contracts.

Both object adapters use the existing evidence service for permissions, quotas, manifests, pins, retention, and upload recovery.
Cloud uploads stage bounded input in private temporary files, calculate SHA-256, and remove the owned files after each call.
S3 supplies a transfer checksum, optional expected owner, and optional KMS key. Azure uses block blobs and transfer validation.
Keys preserve project and run identity beneath a configured installation prefix. Unsafe and oversized keys fail before a provider request.
Missing objects use the shared missing-content error. Missing buckets and containers remain configuration errors.
The server continues private artifact delivery through its existing isolated origin and authorization grants.

Focused race checks pass against in-process HTTPS protocol fixtures with real SDK clients and test-only identities.
They verify signing or bearer authentication, content and checksum preservation, prefixes, bounded writes, escaped names, and idempotent deletion.
Common evidence checks pass for duplicate admission, quotas, pins, expiry, and durable manifests on SQLite and local PostgreSQL.
Settings and staging checks pass for invalid profiles, credential rejection, relative paths, cancellation, private permissions, and owned-file cleanup.
Password compatibility checks pass after required SDK dependency updates.
Native server compilation and a Linux ARM64 build pass. UI and trace assets remain cached. No browser fixture image rebuilds.
An actual local startup verifies the configuration flag, selected directory, ignored default directory, and rejection of public storage paths.
Its owned server stops. The exact PostgreSQL fixture is removed. No verification browser starts.

The concise milestone record is output/artifact-storage/checkpoint.json. Startup evidence is output/artifact-storage/startup.json.
Operator setup, identity requirements, migration, provider lifecycle limits, and verification status appear in artifact-storage.md.
The README, evidence guide, and self-testing guide link the supported profiles.

Protocol fixtures do not prove provider IAM, managed identity, KMS, storage lifecycle rules, or deployed retention behavior.
No cloud resources or deployments are tested. Acceptance **46** stays open for that required verification.
Continue timing parity and provider execution packaging while keeping hosted acceptance separate.

## Current delivery: durable evidence upload recovery

Acceptance **43** is complete. The current ledger has **35 verified, 22 open, 57 total** items.

Upload admission now records durable recovery work with the pending manifest and quota reservations.
Finalization removes that work with its manifest update. Recovery claims expired upload leases and verifies existing size and checksum.
Verified bytes become available without another upload or quota reservation. Missing or corrupt content remains upload-failed.
Temporary storage read failures keep pending work and visible errors for retry.
Bounded scans advance beyond live leases. Competing instances respect the current upload claim.
The retention migration indexes older pending manifests without changing their accounting.
A duplicate pending upload no longer satisfies capture completion. Its execution checkpoint remains available for bounded collection retries.
Recovered evidence does not create another test run or change the original logical outcome.

Focused race checks pass on SQLite and local PostgreSQL.
They cover database and filesystem reopening after interrupted manifest finalization, identical upload replay, current leases, and late completion fencing.
They also cover missing and corrupt bytes, storage read outages, competing recovery claims, paged work, older indexes, and worker checkpoint retention.
Pending upload retry preserves one attachment, one attempt, one logical run, and one byte reservation.
The production UI and native server builds pass. Unchanged trace assets and trusted runtimes remain cached.

The existing local controller restart check now verifies private evidence bytes after another restart with dispatch disabled.
It compares size, checksum, bytes, attachment identities, attempt identities, and the unchanged incomplete outcome.
Its first invocation exposes missing ADAPTER-TIMEOUT fixture coverage. New product-owned diagnostic fixtures provide the declared exact selectors.
Only the affected runner image rebuilds. The corrected controller restart scenario passes.
The check stops its controller, replacement controllers, browser container, and output volume. Exact cleanup checks find none remaining.
The earlier failed check also leaves no owned container or volume. The local PostgreSQL fixture is removed.
No broad browser suite or cloud profile runs. The user's demo and browser remain intact.

Restart evidence is output/worker-recovery/20261104T035538.601559000-afba5699c513d0beff0b98dd/summary.json and cleanup.json.
The concise milestone record is output/evidence-recovery/checkpoint.json.
User and operator behavior appears in evidence.md. Self-testing instructions and Run details help describe recovery without another test execution.

Acceptance **46** retains local, S3, and Azure retention, quota, and pin parity scope.
Timing parity and provider deployment verification also remain open. This milestone does not claim hosted adapter acceptance.

## Current delivery: preparation recovery and retained test history

Acceptances **26** and **29** are complete.
The current ledger has **34 verified, 23 open, 57 total** items.

Preparation recovery now advances through bounded lease pages.
A worker scans one page before normal preparation work. An explicit recovery sweep scans all pages.
One hundred live leases no longer hide an expired job on a later page.
Recovery records interruption, cleans owned resources, and preserves the active release without running tests.
Terminal preparation replay returns the original published or failed record without another inspection or queued job.
Concurrent publication returns one release. Existing bundle checks preserve failed-build and cancelled outcomes during cleanup.
The focused race checks pass on SQLite and local PostgreSQL. No fixture images or cloud profiles are rebuilt for these checks.

Retained history implementation adds current lifecycle context, saved run snapshots, and scoped evidence entry.
It supports archived suites and a missing working definition with a retained revision.
The shared history route opens from catalog titles, More > History, published definitions, and run-result titles.
Runs retain original title, release, result, target, timing, and evidence state. Missing observations display not-recorded.
Bounded older-run scans use actor-bound, scoped, expiring cursors. Exact run denies omit rows.
The page reuses the revision renderer. Technical definitions, revisions, and identifiers use named disclosures.
Scoped evidence entry validates the requested test against the saved run snapshot.
Refresh preserves a tester's later manual evidence selection.

New API and run-domain checks pass with the race detector on SQLite and local PostgreSQL.
They cover archive, working-row removal, retained bytes, pinned evidence, immutable snapshots, current exact permissions, and bounded scans.
Generated-client synchronization and the production build pass. Seven focused UI utility checks pass.
STUDIO-HISTORY-001 passes through normal TestStudio execution with complete evidence in 2.073 seconds.
It uses two actual HTTP runs and fresh browser diagnostic media attached separately through the evidence service.
The check covers saved titles, archive, missing working definitions, revisions, screenshot focus, private trace bytes, help, themes, and mobile width.
It also verifies that current permission loss hides rejected history. No extra logical runs appear.
STUDIO-BROWSE-001 passes in 4.430 seconds with the shared revision renderer and renamed Revisions action.
Two early history checks require fixture corrections: a grant cannot target a deleted object, and revocation must follow the initial load.
The corrected selector passes. Only the affected failed workflow reruns. No broad suite or remote profile runs.

History evidence is output/selftest/20261104T034449.754604000-244062383774ad54492c2264/summary.json and cleanup.json.
Browse evidence is output/selftest/20261104T034140.144453000-c9f1d856dade4c45f83d183f/summary.json and cleanup.json.
The concise milestone record is output/test-history/checkpoint.json.
All owned verification controllers, browser containers, output volumes, and exact PostgreSQL fixtures are stopped and verified.
The user's demo data and browser remain intact.

History does not extend normal artifact retention or reconstruct deleted revision records.
Source omissions use archive. A missing working-row fallback is an additional recovery read, not a production deletion feature.
Evidence restart recovery, provider retention parity, timing parity, and hosted acceptance remain separate open scope.

## Feature log

Record feature commits and targeted validation here.
Each entry records its verification checkpoint. The acceptance ledger defines current verified completion.

### Complete catalog workflows and shared paged API

Acceptance **21** is complete. The ledger has **32 verified, 25 open, 57 total** items.
Catalog creation, editing, validation, publication, import, export, bulk review, archive, restoration, duplication, and history have authorized API and UI paths.
The UI now uses the server list contract for search, filters, stable sorting, category branches, counts, metadata options, and pagination.
It loads 30 matches at a time. Load more tests appends another page. Facets and counts cover the complete authorized view.
Selected identities survive filtering and page changes. Catalog or access changes invalidate an old cursor and require reload with fresh selection.
Read errors display recovery controls instead of an empty registry claim.
Project and suite lists omit exactly denied definitions before building rows, categories, options, or counts.
Whole-suite export checks exact read and export authority for every included definition. It cannot bypass a list denial.
Exports include unloaded pages. JSON file imports remain bounded, validated previews with explicit apply.
Reload catalog clears a rejected plan and preserves the import document for another review.
File reading disables transfer actions. Export access rejection clears the old document and download link.
The generated-client boundary preserves null dependencies and bindings instead of rewriting source-owned facts during metadata edits.
The backend provides the canonical filtering and category rules. Duplicate browser filter and sort implementations are removed.
A shared authenticated cursor helper preserves existing access cursor keys and context while adding a separate catalog namespace.
Cursors bind actor, project, route, query, page size, and authorized snapshot. They expire after 15 minutes.
The self-test registrar follows all suite pages before updating definitions.
OpenAPI 0.28 describes catalog lists, test writes and history, lifecycle actions, duplication, transfer, drafts, validation, and retained release reads.
The generated client, current-page help, interface rules, catalog list guide, and transfer guides match the workflow.

Focused catalog list and HTTP checks pass with the race detector on SQLite and local PostgreSQL.
They cover combined filters, category boundaries and totals, stable duplicate identifiers across suites, pagination, validation, and exact read or export denials.
Opaque cursor checks reject actor, route, project, filter, tampering, expiry, catalog edits, and permission changes.
Existing direct-grant, access-page, and permission-source checks pass after shared cursor extraction.
The targeted original transfer API check passes. It preserves immutable run snapshots through import and bulk archive.
Typed query, nullable definition, and loaded identity merge checks pass. Generated-client synchronization and the production build pass.

STUDIO-BROWSE-001 passes after server filtering replaces local filtering.
It verifies category navigation, metadata filters, selection, sort, lifecycle, history, keyboard disclosure, and themes.
STUDIO-TRANSFER-001 and STUDIO-CATALOG-001 pass through normal TestStudio execution with complete evidence.
The transfer workflow creates 35 fresh checks, follows pages, preserves hidden selections, rejects a stale cursor, and explicitly reloads.
It downloads a full JSON export, reads a new file, rejects invalid assertions, handles a concurrent import conflict without partial writes, and reviews again.
It imports and publishes without activation. The original run and active release stay unchanged.
The catalog workflow creates a native test through the wizard, validates and activates its release, runs it, and inspects retained results.
Their actual browser work takes about 2.7 and 4.2 seconds. Unchanged server binaries and the trace viewer remain cached.
The first transfer pass needs an export-completion wait in the test harness. File reading also receives explicit busy feedback.
A type check requires the metadata display helper to accept retained null tags. The corrected build and workflows pass.
Evidence is output/selftest/20261104T030946.163033000-c5a028b98583370072d766d6/summary.json and cleanup.json.
The earlier browse pass is output/selftest/20261104T030440.276266000-19155287270245cf01475d91/summary.json.
The concise milestone record is output/catalog-transfer/checkpoint.json.
Both browser runs, their controllers, browser containers, output volumes, and the exact owned PostgreSQL fixture are stopped and verified.
No broad suite or cloud verification runs.

The list currently computes its complete authorized view before slicing a page. This milestone does not claim indexed catalog search.
Source preparation completion, removed-test history, timing parity, evidence recovery, retention adapters, and hosted acceptance retain their separate scope.

### Reviewed publication and complete source metadata preservation

Acceptance **25** is complete. Acceptance **21** advances and remains open.
The current ledger has **31 verified, 26 open, 57 total** items.
Validate and preview now stores an actor-owned review of one draft revision and its active release context.
The review lists added, changed, removed, and unchanged definitions, explicit safety changes, source settings, timing, capture, and readable schedule effects.
Publish release and Publish and activate remain distinct choices. Neither action starts a test run.
Pinned schedules retain their release. Future unpinned execution follows activation. Hidden schedules remain hidden.
The publication transaction checks current authority, project, suite, draft, working catalog, active release, and schedule configuration.
Routine scheduler progress does not invalidate a review. Changed schedule configuration does.
Record guards serialize the relevant publication context. Rejected work cannot partially publish or activate a release.
Same-key committed retries return the original release. Concurrent same-key submissions retain one release.
Uncertain responses preserve the in-page request key and selected effect. Replacement preview and the other effect stay disabled until resolution.
Access rejection hides the review. A stale review requires fresh validation and review before another publication.
Prepared-source publication retains its existing source review. Existing trusted clients retain their original publication request shape.
OpenAPI 0.27, the generated client, page help, and user guides describe the new reviewed operations and compatibility behavior.

Focused publication review domain and HTTP checks pass with the race detector on SQLite and local PostgreSQL.
They cover differences, schedule privacy, exact test denials, foreign context, expiry, stale catalog, draft, project, active release, and schedule changes.
Same-key sequential and concurrent recovery, changed-effect rejection, and unchanged activation after rejected work pass.
Generated-client synchronization and the production Angular build pass.
STUDIO-PUBLISH-001 passes through normal TestStudio execution in about 1.4 seconds.
It reviews changes and schedule pins, rejects a concurrent catalog edit, and regenerates the review.
It then simulates a lost response after commit and recovers the same publication without another release.
The original run keeps its release and definitions. Publication without activation keeps the active selection. Mobile layout stays within the viewport.
The first focused UI pass identifies a long pinned-release identifier overflow. Scoped wrapping corrects it.
Evidence is output/selftest/20261104T024320.646208000-175a56010e1548a3268efb8a/summary.json and cleanup.json.
The concise milestone record is output/publication-review/checkpoint.json.
Both UI runs, their browser containers and output volumes, their controllers, and the owned local PostgreSQL fixture are stopped and verified.
No cloud tests or broad suites run.

The acceptance 25 audit combines current revision conflict checks with prepared-source and registered repository synchronization.
The prepared-source check now covers every managed metadata group through a second discovered source version.
Title, description, category, tags, owner, explicit lifecycle, safety, rationale, conditions, timeout, timing, and capture settings survive synchronization.
Source-owned selectors update from discovery. Direct executable edits remain rejected.
A ready repository review refreshes metadata without rebuilding the unchanged source. An obsolete review conflicts before application.
Atomic bulk metadata conflicts preserve all unselected values and create no partial updates.
The five selected catalog and source checks pass locally with the race detector. Existing UI bulk and source conflict evidence remains applicable.
These checks use controlled source and discovery adapters. They do not claim live GitHub or hosted execution verification.
Items **22** and **26** retain their separate broader source preparation scope.

Continue catalog API list contracts and complete transfer workflow evidence under acceptance 21.
Keep evidence recovery, retention adapters, timing parity, and deployment acceptance in the full remaining plan.

### Category browsing, metadata filters, and direct lifecycle actions

Acceptance **21** advances and remains open. The ledger stays **30 verified, 27 open, 57 total**.
The catalog now provides flat table and category navigation views over the same authorized working definitions.
Slash-separated paths form a hierarchy. Parent selections include descendants. Exact category scope remains available under Advanced filters.
Branch counts follow the other active filters without hiding sibling navigation entries.
Search, suite, readable application names, type, and sort controls remain visible.
Advanced filters contains lifecycle, category, child-category scope, tag, owner, runner, safety, and source controls.
Source matches executable or registered definition source identity. Stable suite and test identities resolve equal sort values.
Selections survive filtering and view changes. The bulk bar identifies selected tests hidden by the current filters.
Common Edit, Enable, Disable, and Restore actions stay on each applicable row.
More contains History, Duplicate, and Archive. Existing revision and permission checks protect these unchanged server mutations.
Small standalone filter, category, and table components replace embedded page markup.
Pure utilities handle paths, tree counts, filtering, sorting, readable options, and hidden selection counts.
Application names load after current permission readiness and cannot block the test list. Failed name reads retain identifier fallback with a message.
The table preserves readable test-name width and scrolls within its own container.
Catalog help, interface rules, the catalog guide, and self-test documentation match the implementation.

Four targeted pure checks and the Angular production build pass.
STUDIO-BROWSE-001 passes through normal TestStudio execution with complete evidence in about two seconds.
It covers nested and adjacent categories, combined filters, readable application names, stable sorts, preserved selections, and empty-result recovery.
The same workflow verifies duplicate, enable, disable, archive, restore, revision history, category keyboard disclosure, and light mobile and dark desktop layouts.
The first focused pass finds a permission-readiness race in application label loading. The corrected workflow passes.
Screenshot review identifies cramped test-name cells. The final focused pass verifies a readable minimum width and bounded page overflow.
Evidence is output/selftest/20261104T022644.041563000-3b0a14670db4c5d56df95e09/summary.json and cleanup.json.
The concise milestone record is output/catalog-browsing/checkpoint.json.
All three focused verification runs have confirmed controller, browser container, and output-volume cleanup.
No Go behavior changes, broad suite, or cloud verification runs occur for this interface milestone.

The browser currently filters the authorized working catalog returned by the existing list operation.
Server list filtering and cursor contracts, complete transfer workflow evidence, and publication change review remain under the existing catalog plan.
Do not mark acceptance 21 complete from this browsing milestone alone.

### Private trace viewing and complete run evidence inspection

Acceptances **42** and **45** are complete. The current ledger has **30 verified, 27 open, 57 total** items.
Run details expose attempts, recorded steps, the execution snapshot, logs, screenshots, traces, and configured attachments without HTML reports.
Shared test and attempt scopes preserve the investigation workflow from acceptance 44.
Available traces now open in the installation-hosted Playwright 1.64.1 viewer on the separate artifact origin.
Recorded actions and DOM snapshots replay archived styles and images without contacting the original website.
Current project permissions, session state, grant expiry, artifact identity, and availability protect each server read.
The scoped service worker also checks authorization before returning cached recorded content and blocks external network fetches.
Expired or revoked views close. Refresh trace access requests a new grant. Download trace requires its separate current permission.
Evidence already delivered to a browser cannot be recalled.
Invalid or oversized archives and installations without a viewer retain an authorized local download fallback.
The viewer requires HTTPS or localhost with service-worker support.

The trusted bundle contains pinned assets, checksums, and license notices outside the main UI output.
The Go service loads immutable verified bytes at startup and needs no Node process to serve them.
The demo launcher and isolated self-test image package this bundle. Hosted deployment packaging remains open.
The new trace guide covers API response fields, viewer limits, operator setup, proxy log redaction, and refresh behavior.
Page help and the results, evidence, demo, and self-testing guides match the implemented workflow.

Targeted private-trace API and archive checks pass with the race detector.
Existing artifact origin, permission, expiry, and missing-content checks pass after shared authorization extraction.
Pure URL validation, service-worker authorization and network checks, TypeScript checking, and the production Angular build pass.
STUDIO-TRACE-001 passes through normal TestStudio publication, execution, timing, and evidence contracts.
It uses fixture-backed failed-attempt metadata with fresh real browser screenshot and trace captures.
The original fixture server stops before viewing. The browser verifies archived CSS and image replay, steps, context, logs, screenshots, and a JSON attachment.
It also verifies actual server-side grant expiry, clean viewer closure, renewed access, separate download authorization, and keyboard return focus.
Earlier focused passes expose and correct nested preview focus and snapshot policy issues.
A later expiry race requires closing the viewer before a rejected cached response reaches its parser.
The final harness permits repeated expected 401 responses only for the exact expired authorization URL.
Evidence is output/selftest/20261104T021538.903511000-9ae6bffb5462461d36e93110/summary.json and cleanup.json.
The concise milestone record is output/private-trace/checkpoint.json.
All seven focused verification runs have confirmed controller, browser container, and output-volume cleanup.
The user demo remains preserved. No AWS, Kubernetes, or Azure verification runs.

Items **43** and **46** remain open for complete recovery, retry, retention, and storage-adapter scope.
Next, continue remaining catalog actions and category navigation, followed by timing parity and evidence recovery.

### Scoped investigation, failure recovery, and explicit capture gaps

Acceptance **44** is complete. The current ledger has **28 verified, 29 open, 57 total** items.
Capture gaps, failed uploads, expired or missing files, and incomplete capture remain visible beside the observed outcome.
The Playwright reporter now names requested screenshots, traces, and video that the suite does not produce.
Those gaps do not change the measured test state. Failure-only capture does not invent a gap for an ordinary passing or skipped test.
A real isolated worker probe passes its test and returns two explicit missing-media gaps. Its container and volume are removed.
Targeted storage and run checks verify failed uploads, missing files, expiry, and capture failure without rewriting a passing observation.

Inspect test now scopes attempts, logs, and artifacts together. All attempts remains the default, including unsuccessful retries.
Selected attempts and artifacts use readable cards. Failed recorded steps open automatically. The first results column keeps Inspect test visible on mobile.
Shared field rules constrain long select values. Technical log filters and execution events use named Advanced disclosures.
Screenshot previews receive focus and return it to their triggering button. Scope changes and rejected evidence reads hide stale previews or content.
Refresh preserves the selection and action error. Refresh access reloads current capabilities and the run.
A lost rerun response retains its in-page request key. Retrying returns the existing child. Original run links preserve the parent context.
The continuation ends on page exit or reload. The guide directs users to retained history before starting another rerun.
Filtered evidence pages bind their cursor to the actor, run, collection, test, and attempt. A bounded scan can retain a continuation cursor.

STUDIO-DIAGNOSTICS-001 passes through normal TestStudio self-test execution with complete evidence.
It uses new fixture-backed browser failure metadata and freshly captured private screenshot and trace files for UI checks.
A separate normal HTTP timeout and successful rerun verify response-loss recovery without creating a duplicate child.
The test checks keyboard focus, dark mobile layout, readable step width, constrained selects, permission errors, and visible retention states.
Visual inspection catches table squeezing that a document overflow check misses. Scoped cards correct that problem.
Evidence is output/selftest/20261104T013400.731748000-9d83080068096b0149207e5f/summary.json and cleanup.json.
The real missing-media probe is TestContainerMissingRequestedMediaPreservesPassingObservation, run 20261104T012756.134194000-2fd3601107e1db07a965eaea.
Targeted race checks cover filtered pagination, log retry isolation, quota and upload failure, missing content, expiry, and observed outcome preservation.
Three pure utility checks and the reporter policy check pass. The production Angular build passes.
The checkpoint and selected checks are recorded in output/run-investigation/checkpoint.json.
Owned verification controllers, browser processes, worker containers, and output volumes stop. Verification uses no cloud or public website.
Items **42** and **45** remain open for their remaining evidence and private trace-viewer scope.

### Environment-backed recorded website execution locally

The first single-origin local website flow is complete. The full ledger remains **27 verified, 30 open, 57 total**.
This work advances **22, 34, 35, 36, 38, 41, 42, and 44** without closing their broader pending scope.
Recorded source now prepares through playwright-environment and retains its endpoint policy in the verified immutable artifact.
The review source visibly replaces supported start-origin navigation with TESTSTUDIO_BASE_URL. Private history retains the raw recording.
Offline browser suites keep their separate targetless profile. Preparation still installs no additional source dependencies or scripts.
Browser execution stays unprivileged with network=none. A trusted Unix-socket gateway restricts upstream connections to one registered origin and port.
The profile supports browser traffic and Playwright request fixtures, including HTTP CONNECT clients. Off-site redirects are blocked.
It permits all methods to that endpoint. Additional origins, custom request policies, credential-binding injection, and hosted execution remain open.
Gateway health is checked before dispatch. Normal owned-resource cleanup and recovery include the gateway container.
Admission rejects incompatible localhost targets. Dispatch rechecks permissions, safety, current environment policy, and worker ownership.
The UI enables recorded draft preparation, shows target endpoints on run review, and explains source binding and local addressing.
Page help, user guides, operator setup, suite JSON Schema, OpenAPI 0.26, and generated client types describe the profile.

Targeted race checks pass for source conversion, gateway HTTP and TLS behavior, target admission, changed policies, revoked execution, and protected safety.
A new real native recording closes all 19 owned processes before execution verification.
The first local integration run catches socket-volume ownership. A readiness check and corrected ownership fix that setup error.
The next pass catches Playwright's CONNECT transport for HTTP API requests. Exact-destination HTTP tunneling fixes that compatibility gap.
The final recorded flow passes normal preparation, publication, and execution with one logical result and complete evidence.
It retains 23 steps, a 157 ms test duration, logs, one screenshot, and one trace.
Evidence is output/recorded-website/checkpoint.json, original-recording.json, run.json, and the retained artifact files.
STUDIO-CREATE-001 and STUDIO-WIZARD-001 pass together through TestStudio's normal self-test execution.
Their evidence is output/selftest/20261104T005645.164758000-7c3098f80687727af244e78a/summary.json and cleanup.json.
The production Angular build and generated API contract check pass.
Owned verification browsers, controllers, worker containers, gateway containers, and output volumes stop after checks.
No AWS, Kubernetes, Azure, or public website verification runs.

### Project creator access and project administration boundaries

Acceptance **13** is complete. The current ledger has **27 verified, 30 open, 57 total** items.
Project administrators cannot assign installation privileges or affect another project's groups, assignments, runs, results, or evidence.
Targeted API checks cover original and newly bootstrapped project administrators on SQLite and local PostgreSQL.
They reject foreign group changes, shared-group assignment, foreign assignment and grant writes, installation role assignment, and installation-only permissions.
They reject foreign run details, attempts, events, logs, artifact metadata, cancellation, and evidence grants.
Using an owned project path for a foreign run or artifact returns not found.
Existing custom-policy and inherited-role checks also reject cross-project delegation and installation escalation.

Ordinary authorized project creators now receive four existing project-scoped roles: administrator, editor, publisher, and operator.
Creation commits the project, managed assignment metadata, audit records, and directory revision together.
Disabled setup roles, unauthorized actors, duplicate identities, and failed concurrent creates leave no partial access.
Explicit denies remain effective. Revocation survives later project edits and independent service reads.
Full installation administrators retain their existing authority without automatic project self-assignment.
The project-administrator role itself remains unchanged. Existing projects receive no upgrade-time role changes.
Shared assignment identity and state types preserve the existing metadata shape, revisions, and regular revocation controls.
The wizard explains initial creator access on Review. User help and access documentation describe the roles, conditions, and API behavior.

Four targeted creator domain checks and six relevant API checks pass with the race detector on SQLite and local PostgreSQL.
Command selection is recorded in output/project-access/checkpoint.json.
STUDIO-OWNER-001 passes through normal TestStudio execution with complete evidence.
It uses an ordinary creator to register project resources, publish and run a native check, and delegate a scoped reviewer role.
The first UI pass reaches successful execution but uses an incorrect Add role button label in its final test step.
The corrected selector passes without another application change or repeated backend suite.
Evidence is output/selftest/20261104T003512.986489000-2efe2d2ead9eea0f371bcf8a/summary.json and cleanup.json.
The production UI build passes. Verification browser, controllers, worker containers, output volumes, and the local PostgreSQL container stop after checks.
No AWS, Kubernetes, or Azure verification runs.

Items **05**, **10**, and **12** advance through normal creator setup and durable role handling. Item **12** remains open for its broader scope.
The next active batch is environment-backed recorded website execution.

### Plan reconciliation after tester walkthrough

The active plan now separates implemented decisions, bounded delivery batches, and remaining deployment choices.
A code review confirms that project creation saves the project and audit entry without an initial project assignment.
Project creator access now precedes recorded website execution in the active queue.
The planned owner experience uses project-scoped administration. Atomic creation, audit, denies, and installation boundaries remain required verification.
This is a planned correction. The creation handler and project service do not change in this documentation update.

The next browser batch binds reviewed recordings to registered environments through a separate execution profile.
Offline suites retain their current isolation. Hosted recording remains a later, separate design decision.
The plan preserves normal logical results, timeouts, inclusive reruns, timing, evidence gaps, and private artifacts.
It records the ordinary tester role, target-management permissions, redirects, partial saves, and local network reachability as explicit workflow considerations.

The repository, current Angular UI, accessible SVG charts, database implementations, and source workers are recorded as selected decisions.
The plan removes the obsolete chart-library proposal and dependency-selection questions already resolved by repository manifests.
Azure Container Apps with Jobs is the planning baseline. Cloud packaging and acceptance remain open.
MMDev remains the later AWS verification destination, not an application dependency or permission to test cloud profiles in this session.
Historical continuation commands are labeled as reference. Repository instructions point to the active queue.
The demonstration and recording guides link to the reconciled implementation scope.

Documentation validation checks local links, anchors, guide references, and the stable acceptance numbering.
No application code, data, browser state, or deployment changes. No build or browser suite is required.
The ledger remains **26 verified, 31 open, 57 total**. Planning changes alone complete no additional acceptance item.
The next functional work is project creator access under the remaining access boundaries, followed by recorded website execution.

### Continuing usability and documentation requirements

The plan now makes usability and documentation part of every changed workflow's completion.
The shared UI contract defines tester language, context, steppers, Advanced options, review, preserved work, and accessible recovery.
Simple tasks keep short interactions. Consequences remain visible before configuration, execution, or delivery changes.
A documentation coverage table separates tester, API user, operator, and contributor needs.
The completion checklist requires affected help, user guides, contracts, operational instructions, known limits, and targeted evidence.
The README provides a task-based guide index.
Page help maintenance covers current labels, wizard steps, Advanced sections, side effects, access, and recovery.
The main plan's older navigation and page inventory now align with the current sidebar and planned-page status.
Repository instructions carry these requirements into future work.

Remaining work stays explicit: apply patterns as complex editors change, use tester feedback, and complete operational documentation with provider work.
Recorded browser execution remains the next functional priority and must ship with its user and operator guidance.
The documentation requirements do not claim every future workflow or deployment profile is already documented or verified.

Validation checks 72 local documentation links and anchors, plus all 27 route help-topic references.
The stable acceptance list still contains items 01 through 57, with 26 verified and 31 open.
Whitespace checks pass. No application behavior, browser state, or demo data changes.
No UI build, browser suite, cloud check, or new verification process is required for these prose changes.
This records continuing requirements under the existing plan. It does not mark an additional acceptance item complete.

### Quick HTTPS probes from complete URLs

The HTTPS Request step now accepts a complete website URL by default.
HTTPS uses port 443 automatically. Users do not need to enter a port.
The wizard derives the origin, relative path, and query without contacting the website.
A valid URL entered as the title also fills the address field.
Use a registered environment preserves the existing endpoint-alias workflow.
Both modes share check parsing, assertions, timing, safety review, and catalog saving.

One matching registered target reuses its original stage, protection, and endpoint settings.
Several matches require an explicit choice. Disabled targets remain visible and cannot run.
Linked suite applications constrain target matching.
New addresses show target registration and any new application on Review before saving.
The application belongs to the selected suite context or an explicit application choice.
Advanced target options exposes stage and protection.
New plain HTTP targets require an explicit Local stage choice.
Existing management permissions govern new application and environment records.
Save draft stores reviewed configuration and the test without contacting the website.
Save and run uses the normal local run request after saving the reviewed records.
The suite's active release remains unchanged.
Successful record receipts stay available during in-page retries.
Unknown create outcomes use the existing stable identifier and intended-data reconciliation.
Target configuration locks during partial save recovery. Project changes do not redirect delayed writes to another project.

The first URL check finds that validation can report an error before focus moves to the invalid field.
Validation feedback now waits for rendered fields. The URL control also exposes its parsed validity.
The corrected URL and shared wizard checks pass.
Three focused utility checks pass for omitted ports, paths, queries, explicit ports, invalid URLs, protected targets, disabled targets, prefixes, and ambiguous matches.
STUDIO-URL-001 passes in 3.664 seconds using only local fixtures.
It verifies protected target reuse, draft-only saving, invalid-address focus, Back preservation, reviewed new application and target creation, a successful native run, and dark mobile layout.
STUDIO-WIZARD-001 passes in 7.050 seconds after the shared focus change.
STUDIO-HTTPS-001 passes in 2.972 seconds with the registered-environment choice and retained advanced assertions.
Final URL and wizard evidence is output/selftest/20261104T001458.396810000-58e498dfae0bc7a51c0ecbb9/summary.json and cleanup.json.
HTTP assertion evidence is output/selftest/20261104T001333.956188000-718375f9a5a5512ad70e2294/summary.json and cleanup.json.
The production Angular build passes. Generic URL placeholders keep the independent product examples neutral.
The local demo receives the updated private UI snapshot and retains its data and history.
The user's open tab is not refreshed. No public website receives a verification request.
All owned verification browsers, controllers, containers, and volumes stop after checks.
No cloud verification or unrelated broad suite runs for this delivery.

The ledger remains **26 verified, 31 open, 57 total**.
This extends **05**, **33**, **49**, and **50**, and advances **21**, which remains open.
Continuation: implement environment-backed execution for recorded browser journeys, then continue the complete acceptance ledger.

### Tester sidebar and guided automation

The sidebar keeps Applications, Suites, Test catalog, Test runs, and Trends visible under Testing.
The selector is labeled Project because it chooses a project.
Schedules and Notifications move to a collapsed Automation section.
Import and sync tests and Suite releases move to a collapsed Advanced testing section.
Project and installation administration remain separate collapsed scopes.
The current destination opens its section. Create test routes keep Test catalog marked as current.
Native disclosure controls, mobile menu behavior, and capability gates remain in use.
The import page and its contextual help use the same plain-language navigation label.

Suites now provides a dedicated project route with scoped View tests and Create test actions.
New suite reuses the existing Name, Options, and Review registration wizard.
Suite settings reuses the canonical manager, conflict handling, and retained lifecycle controls.
Unsaved creation or settings changes require a choice before route navigation.
Project changes reset the local suite creation state.
Configuration avoids application registry requests when project read permission is unavailable.
The new page provides Introduction, Concepts, and Tutorials in contextual help.

Schedule creation and editing now uses Tests, Timing, and Review steps.
Identifiers generate from names. Advanced sections hold selection pins, execution location, overlap, limits, and retries.
Review keeps the target, cadence, selection, enable control, and recurring execution consequences visible.
Preview still shows future times without creating a run.
Notification creation and editing now uses Destination, When to notify, and Review steps.
Advanced sections hold identifiers, filters, and retry budgets.
Review keeps the destination, scope, enable control, and future message consequences visible.
Back preserves values. Existing server permissions, ownership, conflicts, and save contracts remain unchanged.
Registered notification credentials stay server-side.

The production Angular build passes. Ten targeted utility checks pass.
STUDIO-NAV-001 passes for sidebar groups, Suites creation, scoped test creation, dirty settings, active destinations, keyboard controls, and mobile navigation.
STUDIO-SCHEDULE-001 passes in 20.673 seconds with the new stepper and retained firing, selection, timing, and result contracts.
STUDIO-NOTIFY-001 passes in 26.827 seconds with guided channel configuration, delivery, retries, history, scope, and mobile checks.
Evidence is output/selftest/20261104T000316.634912000-62d687eaa13df30dbe9e5e41/summary.json and cleanup.json.
A targeted navigation repeat passes after final suite state and help changes.
Final navigation evidence is output/selftest/20261104T000523.976849000-fba71fdf431646d7af6f9d5d/summary.json and cleanup.json.
All owned verification browsers, controllers, containers, and volumes stop after checks.
The user demo serves the updated private UI snapshot and retains its data, examples, and evidence.
The user's open browser tab is not refreshed or changed by this delivery.
No cloud verification or unrelated broad suite runs for this interface work.

The ledger remains **26 verified, 31 open, 57 total**.
This extends **05**, **48**, **49**, and **50**, and advances **21**, which remains open.
Next interface work: accept a full URL directly when creating a quick HTTPS probe.
Current Save and run still requires a registered target environment.
Then resume environment-backed recorded browser execution and the remaining full acceptance ledger.

### Guided configuration and reviewed local execution

Project setup now uses Project, Application, Target, and Review steps.
Users can include a first application, optional target, and empty Availability suite.
The wizard lists every selected resource before creation. Setup creates no tests and starts no runs.
Generated identifiers stay editable under Advanced details. Back preserves entries.
Successful setup steps remain confirmed after a later failure. Retry continues the remaining work.
An uncertain save reconciles its stable identifier and intended data through the existing read contract.
A first-attempt conflict does not adopt or overwrite another record.

Application, environment, and suite registration share Name, Options, and Review steps.
The target URL, stage, and protection remain visible. Endpoint aliases stay under Advanced options.
The related-resource wizard preserves its fixed application association.
HTTPS creation uses Name, Request, and Review with the canonical catalog save and validation services.
A target picker filters enabled environments by the suite's linked applications.
Named Advanced sections contain response assertions, TLS, timing, and identifier controls.
Invalid hidden options reopen and receive focus. Step headings receive focus after navigation.
Review shows the request, target protection, safety label, rationale, and advanced settings.
New tests keep needs-review safety until the user makes a deliberate choice.
Save draft remains separate from execution.
Save and run validates and publishes a suite snapshot without changing its active release.
It runs the selected test and dependencies through the normal local run API.
Save and publication receipts survive in-page retries. Publication and run requests retain idempotency keys.
An interrupted run response does not create a second logical run on retry.
Run creation now uses Selection and Review steps.
Saved selections, test identifiers, limits, and retries stay under Advanced options.
The existing saved-selection preview and frozen-release contracts remain intact.
Contextual help and walkthrough documentation describe the new steps.

The production Angular build passes. Five focused utility checks pass.
The changed browser workflows pass through normal local TestStudio execution:
- STUDIO-WIZARD-001: 6.953 seconds. First-project setup, denied child registration, uncertain target save, preserved entries, invalid Advanced options, mobile review, interrupted run response, and normal run review.
- STUDIO-APPS-001: 2.470 seconds. Shared registration, locked application association, scoped navigation, and mobile view.
- STUDIO-SELECTIONS-001: 4.221 seconds. Saved-selection preview and retained run snapshots with the new stepper.
- STUDIO-HTTPS-001: 2.618 seconds. Advanced assertions, validation, HEAD editing, and retained diagnostics.
- STUDIO-CATALOG-001: 3.878 seconds. Draft publication, reviewed native execution, timing estimate, and evidence.

Final wizard evidence is output/selftest/20261103T235018.292921000-0ea939035701cda28c317f16/summary.json and cleanup.json.
Application evidence is output/selftest/20261103T234854.252019000-63afe8a2be4fbd7d50638ecd/summary.json and cleanup.json.
Other targeted evidence is output/selftest/20261103T234601.999273000-6911eea4d76742eeed63862b/summary.json and cleanup.json.
The first pass found early suite-selection loading and disabled-control validation defects. Both are corrected.
The second wizard pass found an unsupported read path in its final test assertion. The corrected check passes.
Only failed workflows repeat after corrections. Unchanged Go fixture binaries remain cached.
Repeated --test options now select several affected workflows in one build and run.
All owned verification browsers, controllers, containers, and volumes stop after verification.
The existing local demo receives the new private UI snapshot. Its database, examples, and history stay retained.
No cloud verification or unrelated broad test matrix runs for this delivery.

The ledger remains **26 verified, 31 open, 57 total**.
This extends **05**, **33**, **49**, and **50**. Catalog work advances **21**, which remains open.
Continuation: resume environment-backed execution for recorded website journeys, then the remaining full ledger.

### Applications sidebar and project application views

Applications now appears before Test catalog in the project's visible Testing sidebar section.
The link requires project read permission and stays active on application detail routes.
The dedicated list opens each application's environments and linked suites.
The view preserves the current model. Environments belong to an application; project suites can link to several applications.
Archived and shared suites remain represented. Unassigned suites remain outside an application's linked list.
View tests opens a suite-filtered catalog. Create test carries an active suite through the test-type chooser.
The first local browser check identifies a dropped query during fast type selection.
Preserving query parameters fixes that handoff without waiting for suite data to load.

Application, environment, and suite registration reuse the extracted Project settings form and its existing service.
Related registration fixes the application field to the current application.
Unsaved registrations require confirmation. Context changes clear stale view state and fence delayed navigation.
Configuration loading waits for current project capabilities and skips unreadable environment and suite registries.
Missing read access shows an explanation instead of zero resource counts.
The application routes provide contextual Introduction, Concepts, and Tutorials help.

Three focused utility checks pass for sidebar gates, shared suites, unassigned suites, environment ownership, and immutable view data.
STUDIO-APPS-001 passes through normal TestStudio execution with complete evidence.
It covers sidebar discovery, registration, persisted associations, suite-filtered tests, creation handoff, unsaved navigation, help, dark mobile layout, limited read capabilities, and project switching.
The read-capability check controls client capabilities and verifies that restricted registries receive no requests. It does not claim new backend authorization coverage.
Evidence is output/selftest/20261103T231947.114208000-31c311c7670744d0abb77a2b/summary.json and cleanup.json.
The production UI build passes. Existing application binaries and the trusted browser runtime remain unchanged.
No broad test matrix, native recorder check, or cloud verification repeats for this interface change.
The user's demo retains its database, examples, runs, and evidence while receiving the updated private UI snapshot.
The ledger remains **26 verified, 31 open, 57 total**. This extends **05**, **49**, and **50**, and advances catalog navigation under **21**.

### Dedicated test creation, native recording, and retained source editing

Create test now opens /projects/{project}/tests/new instead of expanding the catalog page.
The chooser offers Playwright browser tests, HTTPS probes, and Go suites.
Each type has a dedicated route and a focused workflow.
The HTTPS creation form and catalog edit form share one editor component and save service.
Saving a new probe returns to the catalog with its draft confirmation.
Unsaved forms and source buffers prompt before navigation. Empty projects link to suite setup.
The new creation routes provide contextual Introduction, Concepts, and Tutorials help.

The user chooses browser recording first. Playwright creation now leads with the recorder.
Explicit local server configuration opens the trusted native Playwright browser and Inspector.
Users record actions and assertions, stop the recorder, and review generated code before saving a source draft.
The converter adds a permanent test tag and matching catalog selector. Recorded tests start with needs-review safety.
Sessions retain their generated code and remain private to their actor until saved through normal source permissions.
History resumes active polling after page reload. Identical session identifiers reuse existing recordings.
A 20-minute limit, current-authority maintenance checks, and helper input closure stop abandoned recording processes.
The adapter retains source privately across restarts and closes only owned processes.
The local demo configures recording. Ordinary installations require explicit loopback-only local mode and trusted recorder paths.
Hosted recording still needs a companion or streamed browser.
Recorded website drafts remain saveable, but cannot use the isolated execution profile.
Environment-backed browser execution is the next priority and remains required work.

Retained Playwright ZIPs can open as unsaved editable copies.
Copies preserve nested manifest paths, package declarations, helper source, and executable flags.
Binary or oversized sources return explicit editor limits. Original ZIPs and releases remain unchanged.
The API contract and generated client advance to version 0.25.0.
Explicit sidebar link labels also remove hidden-link name ambiguity in the shared browser navigation helper.
The native browser dependency installation no longer invalidates source-runtime cache fingerprints.

Focused source and API checks pass for retained source preservation, unfinished drafts, authority, revisions, and current execution guards.
The recording API check passes for request-forgery protection, owner privacy, replay, durable code, reviewed source, and no unintended runs.
The maintenance check passes for expired and revoked sessions with retained captured code.
The real native recorder check passes in 2.54 seconds after a corrected readiness condition removes a redundant 20-second wait.
It records a local navigation, converts actual Playwright output to tagged source, and verifies all 18 observed owned processes exit.
Native evidence is output/recording/native.json.
STUDIO-CREATE-001 passes through TestStudio with complete evidence in 2.269 seconds.
It covers the chooser, direct typed routes, native probe creation, unsaved navigation, help, recording stop and reload, saved source, Go import controls, retained ZIP copies, sidebar navigation, and dark mobile layout.
The UI check uses a controlled recorder helper. The separate native check verifies the actual installed Playwright recorder.
Evidence is output/selftest/20261103T195823.296411000-23808cbc945be1ec50eaa8ea/summary.json and cleanup.json.
The owned verification controller, browser container, and output volume are removed. The latest immutable self-test image remains cached.
The production UI build and generated contract check pass. No broad workflow matrix or cloud tests run.
The user's demo restarts with its existing database, examples, history, evidence, and updated private UI snapshot.
The ledger remains **26 verified, 31 open, 57 total**. Items **21** and **22** advance and remain open.

### In-app Playwright editor and saved source workspaces

The Sources page now provides an in-app TypeScript editor and an independent browser starter.
A draft contains the complete suite workspace, catalog, configuration, and optional source or helper files.
The editor provides syntax colors, line numbers, bracket matching, undo, search, and keyboard focus escape.
Saved drafts persist in the installation database. Explicit revision checks protect concurrent edits.
Unfinished code remains saveable. Preparation validates catalog structure and the worker reports code or discovery errors.
Unsaved edits disable suite switching and prompt before page navigation or browser reload.
Conflict responses preserve the local buffer. Users can copy edits, discard changes, and reopen the current saved revision.

The API generates pinned package declarations and the suite manifest.
Preparation snapshots the exact saved revision and uses the existing isolated bundle worker and common source review.
Current source authority, catalog import authority, suite state, and draft revision apply inside the queue transaction.
Later draft edits cannot change retained preparation bytes, releases, or existing runs.
Saving and preparation never create a test run. Publication and execution remain separate steps.
The current browser profile remains offline and isolated. Website environment support and browser recording remain open work.

The Sources page now puts destination selection and browser authoring before repository administration.
It explains HTTP definitions, complete ZIP bundles, repository sources, and the editor workflow.
The native upload form no longer says ZIP bundles and repository connections are unavailable.
Contextual help and playwright-editor.md document the implemented controls and limits.
The API contract and generated client advance to version 0.23.0.

Three focused API and source checks pass locally in less than one second per package.
They cover authority, project boundaries, sessions, request-forgery protection, revision conflicts, unsafe paths, unfinished source recovery, and immutable preparation bytes.
TestPlaywrightEditorLocalStarterBuildsPublishesAndRuns passes through real local assembly, discovery, publication, execution, and complete evidence in 9.14 seconds.
Its owned assembled image, browser containers, and volumes are removed. The trusted runtime stays cached.
The record is output/playwright-editor/local-starter.json.
STUDIO-EDITOR-001 passes through TestStudio with complete evidence in 1.574 seconds.
It covers editing, saved reloads, helper files, a frozen preparation revision, cancellation, concurrent edits, unsaved navigation, help, dark theme, keyboard focus, and mobile overflow.
The first two attempts identify a missing source-preparation fixture flag and an unsuitable hidden-link harness locator.
The fixture enables source preparation. The navigation guard check uses the visible application brand link.
Before broader self-tests, revisit navigationLink hidden-link name resolution. No broader workflow matrix runs here.
Evidence is output/selftest/20261103T192406.578680000-672e1c4c5bbe69b902f4a10d/summary.json and cleanup.json.
All three verification controllers, browser containers, and output volumes are removed.
The Angular production build and generated contract check pass.
The user's local demo restarts with its existing database and the updated private UI snapshot.
The ledger remains **26 verified, 31 open, 57 total**. Items **21** and **22** advance and remain open.

### Grouped sidebar navigation

The user's walkthrough feedback replaces the flat sidebar with Workspace, Testing, Project administration, and Installation administration sections.
Daily testing links stay visible. Native disclosure controls collapse administration links until requested or selected by the current route.
The sidebar opens the current administration section and marks the current link for assistive technology.
A compact Navigation button controls the menu on narrow screens. Link selection closes that menu.
The sidebar and link list use focused components. A pure utility retains existing capability checks and route grouping.
Navigation styles now stay inside the sidebar components instead of applying to every navigation element.
The existing Playwright workflows use a shared helper to open native disclosures and mobile navigation before selecting links.
Permission absence checks include hidden links, so collapsed sections do not conceal a permission regression.

Two focused navigation utility checks pass for section grouping, route ownership, exact account matching, project switching, and capability filtering.
The Angular production build passes.
STUDIO-NAV-001 passes through normal TestStudio execution with complete evidence.
It verifies collapsed defaults, keyboard disclosure, current-page expansion, active-link semantics, mobile menu closure, and narrow layouts in both themes.
The first check identifies an ambiguous Workspace accessibility label. Distinct navigation region names correct it.
Evidence is output/selftest/20261103T190457.595672000-3c8197424f65a7a7161a7165/summary.json and cleanup.json.
The owned verification controller, browser container, and output volume are removed.
The user's demo restarts with its existing database, history, evidence, and a private snapshot of the updated UI.
Unchanged application binaries are reused for the corrected browser check and demo restart.
No unrelated workflow matrix or cloud verification runs for this change.
This improves the existing interface acceptance work. The ledger remains **26 verified, 31 open, 57 total**.

### Stable demo UI assets during development

New demo starts copy compiled UI assets into a private content-addressed directory under the demo installation.
The server serves that snapshot instead of the mutable development build directory.
Later development builds cannot replace the files used by a running snapshot.
Unchanged snapshots are verified and reused. Altered snapshots and symbolic links are rejected.
Copy verification rejects a build that changes during setup.
A focused local filesystem workflow replaces the source build and verifies that the first snapshot retains its original index and lazy chunk.
The same workflow verifies reuse, a separate snapshot for changed output, altered-cache rejection, and symbolic-link rejection.
Evidence is output/demo-ui/result.json. Temporary fixtures are removed after verification.
The user's current demo stays running. No Angular build, browser test, cloud check, or demo restart runs for this change.
The ledger remains **26 verified, 31 open, 57 total**.

### Saved catalog selections and frozen run reviews

Suites now store shared saved selections with explicit identifiers or schema-versioned filter expressions.
Filters support text, categories and descendants, tags, owner, application, test type, runner, safety, and source.
The catalog UI creates, inspects, edits, archives, restores, and resolves these definitions through generated API operations.
Explicit definitions require valid distinct identifiers. Unsupported filter versions and conflicting definition shapes fail validation.
Edits use revision checks and current suite catalog authority inside the shared assignment transaction.
Lists paginate and omit explicit selections whose identifiers fail current exact read checks.

Resolved previews pin one retained release, selection definition and revision, root identifiers, and dependency closure.
Previews expire after 15 minutes and belong to the creating actor.
Empty matches, missing or disabled dependencies, revoked read permissions, and current disabled test state block execution explicitly.
Run creation accepts savedSelectionPreviewId and asserts the saved definition revision when committing its frozen run snapshot.
The run form requires a preview before timing estimates or execution. Run details shows the retained selection name and revision.
Later active releases, saved definition edits, and archival do not rewrite existing runs.
Failure reruns use the parent's frozen release and failed identifiers instead of the saved preview.

Focused API checks pass for frozen releases, retained definitions, stale and expired previews, current lifecycle, actor ownership, dependency closure, read revocation, archive, and restoration.
An actual failed saved-selection run reruns its failed identifiers after the saved selection is archived.
Ordinary run evidence and legacy timeout reruns remain covered by targeted checks after the run request change.
The Angular production build and generated OpenAPI client synchronization pass.
STUDIO-SELECTIONS-001 passes through normal TestStudio execution with complete evidence.
It creates explicit and filtered selections, handles a revision conflict, reviews dependencies, and executes an older reviewed release after a new activation.
It also verifies the stored run selection after later definition edits.
Evidence is output/selftest/20261103T183650.273193000-92ddbebefae1e75c5eb9494c/summary.json and cleanup.json.
The owned controller, browser container, and output volume are removed after verification.
Only the affected browser workflow and selected local backend checks run. No cloud verification runs.
Items **21** and **28** advance. Category-tree navigation, broader catalog controls, and the full admission requirement keep them open.
The ledger remains **26 verified, 31 open, 57 total**.

### Reviewed bulk metadata editing

The existing bulk preview route now supports explicit metadata changes for 1 through 30 tests from one suite.
Editable fields include category, tags, owner, title, description, safety label, rationale, and operating conditions.
The focused editor requires explicit field selection and shows field validation beside its controls.
Unselected fields stay unchanged. Empty optional values clear the selected fields.
Visible before and after values supplement the retained full definitions in the common review panel.
Source-owned executable definitions, selectors, lifecycle states, active releases, and historical runs remain unchanged.
Safety selections require an explicit rationale. Changed labels require current publishing authority.
Plan reads and apply operations recheck exact test read authority. A stored preview cannot retain revoked access.
Import and bulk application now share permission and metadata transactions. Stale plans cannot partially apply.
Concurrent replays retain their idempotent result without adding revisions.
Selection and metadata changes invalidate displayed or pending previews.
OpenAPI records the bulk request, plan read, and apply contracts. Generated client synchronization passes.

Focused local catalog checks verify prepared source facts, metadata clearing, stale-plan atomicity, current publishing and update denials, and concurrent replay.
The focused API check rejects executable metadata fields and exact read denial after preview creation.
The Angular production build passes.
STUDIO-BULK-METADATA-001 passes through normal TestStudio execution with complete evidence.
It edits two selected tests, reviews changed values, rejects a stale apply, clears optional fields, and preserves the active release and historical run.
Evidence is output/selftest/20261103T181950.320666000-96aed8860fd546b2041e375b/summary.json and cleanup.json.
The owned controller, browser container, and output volume are removed after verification.
Only the affected browser workflow and selected backend checks run. No cloud checks run.
Item **21** advances. Category-tree navigation, saved selections, broader filters, and remaining catalog requirements keep it open.
The ledger remains **26 verified, 31 open, 57 total**.

### Suite settings, archive, and restoration

Project settings now opens a focused suite manager with editable name, description, supported applications, and target requirements.
The manager displays the active release, current authority, and readable affected schedules with their release pins.
Settings updates preserve execution policies and the active release. They do not publish or run tests.
Archive and restoration preserve the suite identifier, releases, run snapshots, retained evidence, and schedule pins.
Archive blocks new dispatch and new attempts through the existing execution checks.
Read checks omit denied schedules from the affected list and denied suites from the suite collection.
Settings, lifecycle, and legacy PUT routes check current exact suite read and management authority inside a shared assignment transaction.
Lifecycle response construction shares the mutation transaction. A rejected response read cannot leave a committed update.
Revision conflicts keep the form and block another mutation until the user reloads current values.
The contextual help provides suite lifecycle concepts and a management tutorial.
The OpenAPI additions and generated TypeScript client remain synchronized.

The focused TestSuiteLifecycle API checks pass locally in under one second.
They verify missing and stale revisions, duplicate applications, archive dispatch rejection, restoration, retained run snapshots, artifact downloads, and schedule pins.
Exact read, management, and schedule denials pass. Rejected changes preserve the stored suite revision.
The targeted STUDIO-SUITES-001 Playwright workflow passes through normal TestStudio execution with complete evidence.
It verifies saved settings, a concurrent edit conflict, mandatory reload, archive, retained release access, restoration, schedule pins, and page help.
Evidence is output/selftest/20261103T180829.702720000-ab5aa2952544dc82f9fc933b/summary.json and cleanup.json.
The owned controller, browser execution container, and output volume are removed. Unchanged native binaries are reused for the browser correction.
No unrelated browser workflow, broad fixture matrix, or cloud verification runs for this feature.
Items **21** and **29** advance. Their broader catalog and removed-test requirements remain open.
The ledger remains **26 verified, 31 open, 57 total**.

### Usable local demonstration

The immediate demonstration checklist is complete. The full acceptance ledger remains **26 verified, 31 open, 57 total**.
Run `node scripts/demo.mjs start` to build and start an explicit, separate local demonstration.
The launcher reuses unchanged UI, server, preparation worker, and trusted browser runtime outputs.
It starts loopback listeners and uses a dedicated SQLite database, artifact directory, and local target.
New generic HTTP checks demonstrate passing, failing, and timed-out results.
A new Parcel desk Playwright check produces screenshot evidence through normal source publication and execution.
The setup records three HTTP runs and one browser run only once.
The local browser walkthrough edits and publishes a catalog, starts a manual run, previews screenshot evidence, and opens stacked trends.
After target recovery, failure reruns select the failed and timed-out checks. The result contains one pass and one timeout.
The manual run uses three clean healthy-check observations for its timing baseline.
Restart verification preserves all six runs, the catalog edit, and screenshot metadata without duplicate setup runs.
Saved-port checks and an owned-process readiness message prevent seeding into another local server.
An occupied-port check confirms startup stops before server startup or seeding.
Use Ctrl+C or the authenticated local stop command. Demo history and source images remain available for resumption.
The browser session, owned service processes, and execution containers and volumes are cleaned after verification.
Evidence is `output/playwright/demo-checkpoint.json`, `demo-port-conflict.json`, workflow snapshots, and screenshot captures.
The README and demo.md document startup, the walkthrough, restart behavior, and intentional failures.
No broad test matrix, Linux fixture rebuild, cloud account, or cloud deployment check runs for this delivery.
The pending suite lifecycle edits remain separate work. Broader acceptance items remain open.

### Independent repository build

Item **01** passes an isolated build of the current TestStudio source tree.
The copy contains 733 repository source files and excludes ignored dependencies, generated outputs, and sibling checkouts.
The Go server builds with workspace mode disabled, read-only module metadata, and network module access disabled.
All 459 compiled package dependencies resolve inside the temporary copy, the Go toolchain, or the ordinary published-module cache.
There are no module replacements, MarketMaker or Mapag package imports, or external source directories.
A clean offline npm installation and Angular production build pass inside the same temporary copy.
The npm lockfile contains no local-path dependencies. Installed package links remain inside the isolated copy.
Evidence is .data/independent-build._o3cvq4v/result.json and package-dependencies.json.
The temporary source, dependency installation, and build outputs are removed after verification.
No tests, browser matrix, or cloud checks run for this build-only acceptance item.
The ledger is **26 verified, 31 open, 57 total**.

The user prioritizes a usable local demonstration over further feature expansion.
The complete 57-item plan remains required later scope. The immediate delivery follows demo-readiness.md.

### Retained release comparison, activation, and rollback

Added the Suite releases page with retained catalogs, immutable provenance, exact commit links, definition inspection, and release comparisons.
Comparisons preserve permanent test identities and separate definition changes from artifact, timing, and evidence policy changes.
The registry exposes the current active release, suite revision, archived state, and exact activation authority.
Archived suites retain readable release history.
Activation now checks current permission inside the shared assignment transaction and records previous and selected release identifiers.
Concurrent suite changes produce a conflict. The UI requires a refresh before another activation.

Item **27** passes the full local publication, activation, and rollback workflow through the UI and normal run API.
Runs created before and after a definition update keep their original release and title after repeated activation changes and rollback.
A future unpinned run follows the rolled-back release. An explicit run pin still executes the other retained release.
An existing schedule release pin remains unchanged.
The same shared run snapshot contract retains coded artifacts, with earlier two-image execution evidence under item 24.

Focused catalog and generated-client checks pass. Pure comparison and immutable source-link checks pass.
The Angular production build and STUDIO-RELEASES-001 pass with complete evidence and cleaned execution.
Evidence is output/selftest/20261103T171931.090298000-b214991be576d067b3830d33/summary.json.
The owned browser container, volume, and controller are absent after verification.
The first targeted test uses an unsupported suite detail URL at its final archive step. The corrected test uses the existing collection API.
The retry reuses every unchanged binary and runs only the affected selector.
The self-test UI input fingerprint also ignores standalone TypeScript test files that the application build does not import.
No repeated Go or browser source-image integrations, broad matrix, live provider, or cloud tests run.
The ledger is **25 verified, 32 open, 57 total**. Items **21** and **29** advance but remain open.
The full acceptance scope remains required.

### Browser-confirmed CLI sign-in for local and OIDC accounts

Added native publisher login with browser, no-open, and client-name options.
The terminal generates a private proof bound to the server time and displayed client name.
The browser requires normal account sign-in, a matching terminal code, explicit approval, current origin, and CSRF protection.
Anonymous polling creates no durable authorization records. Approved or declined requests expire after five minutes.
The server atomically consumes approval and creates one independent managed CLI session.
It preserves authentication and credential revisions, user revocation epoch, and the original authentication and session expiry bounds.
The browser remains signed in. Private provider tokens never enter the CLI or confirmation response.

The terminal saves only managed TestStudio credentials in the existing private session-file format.
Failed file delivery cancels the issued session. Successful replacement attempts to revoke the previous same-origin CLI session.
User session administration displays each CLI client name and retains ordinary revocation controls.
A new confirmation page provides contextual Introduction, Concepts, Tutorials, keyboard dismissal, and mobile layout.

Focused local checks pass for proof tampering, client-name binding, wrong codes, concurrent exchange, replay, denial, expiry, account disabling, parent revocation, private output, and failed file delivery.
The signed-in browser session remains valid after CLI issuance and cancellation.
The existing local publisher review and publication check and session CSRF/revocation check pass.
Generated client checks and the Angular production build pass.

STUDIO-CLI-001 runs the actual publisher binary through normal TestStudio publication and execution.
It passes local browser approval, a new local OIDC provider, denied approval, private file modes, independent session credentials, and expiry bounds.
It also passes help input preservation, keyboard close, dark theme, mobile width, and owned publisher process cleanup.
Evidence is output/selftest/20261103T170106.595113000-222b227f16dac6a53ad47c83/summary.json.
Evidence is complete. Execution is cleaned. Its owned browser container, volume, and controller are absent.
The first image build misses the new binary's Docker allowlist entry. The corrected retry reuses unchanged UI and binaries.
Only this new browser selector runs. No broad matrix or cloud verification runs.
The ledger remains **24 verified, 33 open**. This extends the implemented behavior for **09**, **19**, and **50**.
The full acceptance scope remains required.

### Offline Playwright source bundles

Added a trusted generic browser runtime with pinned Playwright packages and no application tests, UI, or controller files.
The runtime build reuses the existing trusted dependency layers without downloads or reinstalls in this checkpoint.
The source assembler validates exact runtime versions, lockfile roots, required packages, and package shadowing before assembly.
It uses the existing playwright-node profile, fixed copy instructions, isolated discovery, and normal publication workflow.
Uploaded build scripts and dependency installers do not run during assembly.

ZIP staging and repository suite bindings now accept Go or Playwright profiles through the Sources UI and API.
The preparation worker accepts independent Go and Playwright runtime flags and keeps recovery for both profiles.
Workers leave unsupported runtime jobs queued and scan later pages for work they can process.
The publisher CLI uses the same source assembler for either runner.

A newly written browser bundle imports shared application source, builds, discovers, publishes, and passes a normal targetless run.
It records one passed logical result and complete evidence in about eight seconds.
Verification is TestBrowserSourceBundleBuildsPublishesAndRunsWithRuntimeDependencies in internal/sources/browser_bundle_local_test.go.
Run identifier is 20261103T164017.169602000-e7ff513fa532ce3fa4073713.
Its owned image, discovery resources, browser container, volume, and private context are removed.
An initial profile mismatch is corrected by using the existing inspector profile before the passing run.

Focused dependency, controlled-builder, source, queue-pagination, and compilation checks pass.
The Angular build and updated STUDIO-BUNDLES-001 pass for both Go and browser uploads, retained source, queued jobs, and cancellation.
Evidence is output/selftest/20261103T164151.211223000-fbde814a60efc9df0c742ab8/summary.json.
Only the affected browser selector runs. No repeat of the Go image integration or broad matrix runs.
The ledger remains **24 verified, 33 open**. Item **22** advances and remains open for complete source workflow evidence and broader dependency preparation.
All checks run locally. No cloud deployment or live GitHub tests run.

### Signed push events and durable periodic synchronization

Added signed GitHub push notifications through a provider-neutral verification port and a separate provider endpoint contract.
The receiver authenticates exact body bytes before parsing and verifies the pinned repository identity and watched reference.
Duplicate delivery identifiers return their original sync. Conflicting payloads are rejected.
Delayed pushes queue resolution of the current reference, which prevents old payload commits from replacing the current source.
Unwatched references and deletions create no work. Signing secrets remain separate server-side references.

The existing service scheduler now processes durable repository timers.
Atomic source and timer updates prevent competing local replicas from queueing duplicate firings.
Missed intervals and pending source work coalesce. Blocked authority leaves a visible diagnostic and advances the next timer.
The Sources UI edits push and interval policy and displays the webhook address, next due time, and trigger history.
Source summaries retain current worker and recovery states.

Focused local checks cover signatures, payload tampering, project-scoped secrets, duplicate deliveries, delayed pushes, scheduler contention, coalescing, disabling, and revoked authority.
The signed provider endpoint passes without a browser session. Unsigned requests are rejected.
The Angular build and updated STUDIO-GIT-001 pass with periodic settings and retained next-due state.
Evidence is output/selftest/20261103T163150.954443000-181a4e4001c3776a1f6876e3/summary.json.
Evidence is complete, execution is cleaned, and the owned browser container, volume, and controller are absent.
Only the affected UI selector runs. No repeated Go image integration, broad self-test matrix, live GitHub, or cloud verification runs.
The ledger remains **24 verified, 33 open**. Items **22**, **25**, and **26** advance.
Playwright source assembly and dependency preparation profiles remain required.

### Registered GitHub sources and shared suite snapshots

Added provider-neutral repository registration, manual sync, and private GitHub App access.
The API queues work. The separate preparation worker pins the repository identity and commit, then fetches one bounded archive.
Several bound suites share its exact immutable bytes and full application context.
Credentials remain server-side and outside images, browser configuration, and diagnostics.

The Sources page registers settings, multiple suite bindings, private credential references, and publication policy.
It exposes durable sync history, preparation diagnostics, and the canonical catalog review.
Repeated requests and unchanged commits reuse work. Explicit force requests create fresh preparation.
Ready reviews refresh managed metadata without repeating assembly or discovery.
Current authority, generation, worker leases, and latest sync sequence fence stale work and activation.
Source-scoped denies protect retained details, preparations, reviews, and source downloads.
Failed preparation preserves active releases. Registration, sync, and publication create no test runs.

Item **23** passes one real local two-suite repository workflow.
Both newly written suites import shared application source from a nested module.
Each builds a separate image, publishes a release, and passes a normal targetless run with one logical result and complete evidence.
The test removes its owned images, containers, volumes, and contexts. It reuses the cached trusted runtime.
Verification is TestRepositorySnapshotRunsTwoSuitesBesideApplicationSource in internal/sources/repository_local_test.go.
Run identifiers are 20261103T162229.539610000-5f6f05a43d498d033dccee67 and 20261103T162234.448321000-ea7d79c4fca9e3d39295a05c.
The integration test takes about 22 seconds. Its initial fixture path error is corrected before images are built.

Focused source, API, archive, GitHub protocol, publication, and generated-client checks pass in about four seconds.
GitHub protocol checks use local TLS fixtures. They make no live GitHub requests.
The Angular production build and STUDIO-GIT-001 pass.
The UI self-test covers registration, two bindings, setting changes, queued history, dark theme, and mobile width.
It fixes a mobile action-row overflow found by its first targeted run.
Evidence is output/selftest/20261103T162436.812696000-0512635ac07bcde78895b448/summary.json.
The execution is cleaned with complete evidence. Its owned browser container, volume, and controller are absent.
The retry reuses unchanged binaries and rebuilds only changed UI and image inputs.
Items **22**, **25**, **26**, and **29** advance but remain open for their broader required scope.
The ledger is **24 verified, 33 open, 57 total**.
Push webhooks, periodic sync, Playwright assembly, and additional dependency profiles remain required work.
No AWS, Kubernetes, or Azure checks run.

### ZIP uploads, separate durable builds, and targeted self-test reuse

Added project-scoped ZIP upload and exact source download through authenticated API operations.
The Sources page selects a complete Go bundle and manifest path, then displays durable preparation progress.
A separate teststudio-prepare process assembles the image. The API only queues build work.
The worker retains ownership before assembly and checkpoints its image before isolated discovery.
Build metadata, runtime, bounded logs, review, publication, cancellation, and source downloads use the common source workflow.

Bundle retries return the original job. Preparation and publication create no unintended runs.
Current authority checks remain inside the shared transaction and worker heartbeat.
Ready withdrawal and interrupted-job recovery clean owned images and private contexts without repeating builds or tests.
Expired staging removes ZIP bytes while retaining summaries, logs, original failures, and completion timestamps.
Published snapshots retain their exact ZIP bytes and source provenance.

Focused source, API, builder, generated-client, expiry, and context-recovery checks pass.
The Angular production build passes.
A real local browser workflow uploads a ZIP, waits for its separate worker, downloads source, reviews, and publishes.
A separate explicit normal run passes. Reload retains published history. The mobile dark view has no document overflow or browser errors.
Evidence is `.data/bundle-ui-verification.Eh6NWc/result.json`, `browser.json`, and `cleanup.json`.
Cleanup confirms browser and controller exit, no owned containers, volumes, suite images, or private build contexts.

Added permanent STUDIO-BUNDLES-001 for ZIP staging, exact source, duplicate replay, cancellation, and retained history.
The targeted workflow passes through normal TestStudio publication, execution, timing, and complete evidence capture.
Evidence is `output/selftest/20261103T151752.455247000-97c8c9cdf744e8bad8b0705a/summary.json`.
The execution is cleaned. Its owned container and volume are absent after verification.
An initial test observation depended on Chromium's response-body cache. The final test reads durable preparation state through the API.

The self-test launcher now fingerprints source inputs and verifies cached output hashes.
It reuses unchanged UI builds, native and Linux binaries, and immutable runner images.
The local base image is pulled only when absent.
A warm build check reuses every artifact in about 0.25 seconds without executing tests again.
No broad matrix, AWS, Kubernetes, or Azure tests run for this milestone.
The ledger remains **23 verified, 34 open**.
Items **22**, **23**, **25**, **26**, and **29** advance. GitHub preparation and remaining source profiles keep those broad criteria open.

### Offline Go source bundles and reusable build interface

Added bounded ZIP validation before extraction, including paths, entry types, duplicate names, file conflicts, CRC checks, and required source context.
Manifest selection preserves the full repository context and supports separate manifest paths within one bundle.
The source digest covers the original archive bytes, including shared file changes.
Regular executable helper files retain execution permission. Links and special entries are rejected.

Added an artifact builder interface separate from inspection and test scheduling.
The local implementation assembles files on a trusted immutable Go runtime with fixed copy instructions.
Uploaded Dockerfiles cannot control assembly. No source-defined build commands or dependency downloads run during assembly.
The CLI provides build-bundle and retains a private ownership reference before assembly.
Recovery removes only the matching image tag. Published releases still require retained execution images.
Go discovery and normal execution use included vendor dependencies with downloads disabled.
The runtime build script sends only its binary directory as build context.

Focused archive and builder tests pass. The offline vendor discovery and execution check passes.
The publisher package compiles and its managed-session checks pass.
A real local workflow assembles a new ZIP with a nested manifest and vendor dependency, publishes it, and passes a normal run.
Evidence is `.data/bundle-verification.GoxFng/result.json` and `cleanup.json`.
The check removes its controller, containers, volumes, and assembled suite image.
Items **22** and **23** advance but remain open.
At this assembly checkpoint, direct UI staging, durable server builds, GitHub preparation, and Playwright assembly remain pending.
The subsequent ZIP upload milestone above delivers direct staging and durable server builds.
No cloud checks or broad test matrix run for this milestone.

### Durable prebuilt preparation and publication

Added durable, project-scoped image preparation through the API, Sources page, and publisher CLI.
HTTP requests enqueue work. The worker records ownership before inspection and retains bounded diagnostics.
Preparation rechecks current permissions. Cancellation and expired-lease recovery remove owned resources without rerunning tests.
Recovery remains available when new dispatch is disabled.

Prepared catalogs publish atomically with project-bound immutable artifact approval.
Execution accepts newly published artifacts without restarting the API or changing its image list.
Source synchronization preserves managed metadata and archives omitted source-owned definitions without deleting history.
Manual edits cannot assign source ownership or replace discovered facts.
Duplicate creates an independent draft without inherited source ownership.
The shared publication path retains revision conflicts, explicit activation, and replay behavior.

The UI provides runner options, image preparation, progress, retained logs, cancellation, history, and shared review.
The CLI provides managed local sign-in, private session storage, preparation, status, review, publication, cancellation, and logout.
At this checkpoint, CLI OIDC browser sign-in, ZIP builds, GitHub preparation, and browser verification of the new preparation controls remain pending.
The subsequent bundle milestone adds offline Go ZIP assembly through the CLI.

Targeted local source, HTTP API, CLI, and generated-client checks pass.
The ownership regression check and Angular production build pass.
A cached-image workflow publishes two image digests and passes one normal run from each through the same controller process.
The second publication preserves a title changed through the catalog API.
Evidence: `.data/prebuilt-verification.Y3B1AO/result.json` and `cleanup.json`.
Cleanup confirms controller exit and no owned containers or volumes.
Item **24** is verified. Items **21–23** and **25–29** receive partial implementation progress and remain open.
No AWS, Kubernetes, or Azure checks run in this milestone.

### Prepared suite manifests and isolated artifact inspection

Added versioned suite and source-origin schemas and shared bounded document decoding.
The local artifact inspection port reads immutable Go and Playwright images in separate preparation namespaces.
Go discovery compiles source declarations without executing initialization or test bodies.
It distinguishes runnable declarations from helpers, TestMain, and non-running examples.
Dynamic Go child paths retain explicit parent-only verification.
Playwright discovery preserves exact project selection, inherited tags, and source locations.
Both prepared formats execute through the existing managed runners.

At this inspection checkpoint, the local CLI provides inspect and cleanup operations with private files and saved resource references.
Server publication is added in the later durable preparation milestone above.
The inspection interface remains separate from future source builders.

Targeted package checks and real local inspection checks pass before the shutdown checkpoint.
The latest retained fixture results are `.data/discovery-verification.WqyUvr/go.json` and `playwright.json`.
Both contain two inspected entries. Owned fixture resources are absent after those checks.
Queued preparation, publication, CLI authentication, and interface work are delivered in the later milestone above.
Remaining cancellation and recovery scenarios belong to that integration work.
The complete acceptance ledger remains required.

### Native source uploads and atomic release publication

Added the Sources page and suite-scoped JSON and YAML upload APIs.
Uploads retain the stored UTF-8 source and its server-computed SHA-256 digest.
Strict document parsing rejects duplicate keys, aliases, merges, unsupported tags, trailing documents, and excessive structure.
The initial native workflow uses the existing 30-test import limit and a 1 MiB source limit.

The review shows discovered identifiers, categories, types, safety labels, and before and after definitions.
The upload actor explicitly publishes and chooses whether to activate the release.
Publication uses the existing catalog import, draft, release, and activation contracts in one shared authorization and metadata transaction.
Source-bound plans reject ordinary import application outside that transaction.
Stale reviews and publication failures preserve catalog records and the active release.
Concurrent retries return one immutable release and preserve the original activation choice.
Publication never creates a test run.

Release artifacts reference retained source snapshots and digests.
Draft and publication checks validate source ownership, availability, and stored content digests.
Published snapshots survive activation changes and rollback.
A bounded maintenance worker expires abandoned staging and removes its source text and import plan.
Published source retention currently has no automatic deletion.

The composed Angular page includes upload history, retained source inspection, explicit activation, accessible change review, and contextual help.
The generated OpenAPI 0.13 client describes five source operations.
Dictionary generation uses index signatures so recursive JSON assertion values compile correctly.

Source and catalog checks pass on SQLite with race detection.
Source, catalog transfer, and source API checks pass on local PostgreSQL with race detection.
Checks cover stale revisions, publication rollback, actor binding, project isolation, invalid documents, staging expiry, and concurrent publication.
The generated client check and Angular build pass.
Permanent self-test STUDIO-SOURCES-001 passes through normal TestStudio publication, execution, timing, and evidence handling.
Run `20261103T112052.874370000-12768173eed23e629c9e7309` verifies upload review, exact digests, detached-plan rejection, immutable releases, explicit execution, malformed files, and mobile help.
Desktop light and mobile dark screenshots were inspected.
The current-page help regression also passes in run `20261103T111917.109619000-3e8ce2db188b82120631a393`.
Owned controllers, browser containers, volumes, and the local PostgreSQL test container are absent after verification.

ZIP preparation, GitHub connections, complete source synchronization, source-owned metadata, prebuilt and CLI publication, build jobs, and configurable source retention remain required.
The complete source acceptance criteria remain unchecked. The full product plan remains active.

### OIDC acceptance, provider sign-out, and complete claim handling

Added optional exact end-session endpoints and post-logout callback display to managed provider settings.
Local session revocation and cookie clearing precede provider integration.
The interface offers an explicit provider action after TestStudio sign-out.
Client-identifier logout sends no provider token or client secret through the browser.
A separate HttpOnly browser cookie, opaque offer, state, expiry, provider revision, and one-time records bind the protocol.
Discovery must match the configured endpoint. Callback completion never clears a newer session.
Provider failure and cancellation retain local revocation.

Sign-in preserves approved local destinations, including query and fragment.
Fixed feedback covers cancellation, expiry, provider unavailability, and rejected identity proof.
Normal sign-in replaces the prior browser session. Authentication and API redirects use no-store responses.
Provider HTTP clients retain shorter timeouts and cap background key requests at ten seconds.
Logout discovery has a separate five-second deadline.
Discovered protocol endpoints require HTTPS outside loopback, and redirect policies reject HTTPS downgrades.

The shared verifier uses the SDK’s original signature verification and basic identity validation.
Additional checks cover authorized party for all audience forms, required issue time, bounded subject, token use, and access-token hash.
A signature key-set adapter isolates unsupported distributed metadata from SDK identity parsing while retaining original claims for application validation.
No signed input is rewritten before verification. Invalid identity bindings cannot change memberships.
Valid identities with incomplete group data withdraw only that provider’s old memberships.
Missing, null, malformed, overflowing, paged, distributed, complete-empty, and unmapped cases have focused coverage.
Native permission checks prove immediate loss after external membership expiry.
Unresolved email verification cannot activate pending project access.

The permanent STUDIO-AUTH-001 self-test runs through normal TestStudio publication and execution.
It verifies provider settings, deep links, key rotation, browser session replacement, token-free browser storage, optional logout, denial, outage, callback feedback, and stale-group withdrawal.
Run `20261103T104942.021902000-31381e3e1f8f5a36ec4c0254` passes with complete evidence and cleaned execution resources.
Desktop light and mobile dark screenshots were inspected.
Provider administration regression run `20261103T103654.904212000-3cc587e46077e07ac9e7e690` passes.
Account-linking regression run `20261103T103835.895016000-b62c15eecef928f3a4c21de4` passes.

Focused OIDC, session, API, and generated-client checks pass on SQLite and local PostgreSQL, including race detection.
Idle, absolute, and authentication expiry reject the next normal and evidence request.
The UI build and pure provider and authentication-navigation checks pass.
OpenAPI 0.12 and the generated client contain 74 JSON operations.
Three focused authentication acceptance criteria are now verified.

Deployment-wide identity parity, remaining authorization and privilege-change behavior, sources, execution parity, cloud profiles, and the remaining full plan stay required.

### Pending project access without invitations

Added project-owned pending access preparation, curated enabled provider choices, bounded reads, and revision-controlled cancellation.
Records bind an allowed provider revision and issuer, mandatory verified email, exact project, permitted role, approving owner, and audit reason.
Preparation creates no user or sign-in identity and sends no invitation.
Project managers require no installation directory or provider management authority.
Provider choices contain no client credentials or secret references.

Normal OIDC sign-in supplies verified provider claims through a focused activation port.
Activation rechecks provider configuration, identity ownership, the project, current owner authority, and the actual recipient.
Role fingerprints retain predefined state and inherited custom policy revisions.
Changed approvals become blocked and require explicit new preparation. Email matching never merges internal accounts.
Identity-linking proof does not apply pending access.
Configured first-sign-in provisioning remains authoritative.

Refactored direct role changes into a shared transaction-bound helper.
Pending activation uses the same permission service and exact project assignment contract.
Native assignments, metadata, activation, match removal, and audit commit together.
Consumed approvals cannot regrant a subsequently revoked role.
Cancelled, blocked, and activated history remains readable within its exact authority.
Exact pending-object, role, and activated-recipient denies remain effective.

Project access provides focused preparation, history, cancellation, and read-only components.
Editors retain their opened provider revision across ordinary refreshes.
Conflicts and unreadable responses block repeat submission until explicit state and choice review.
Revoked authority clears private edits. Contextual help, shared themes, mobile layouts, and keyboard submission accompany the workflow.
OpenAPI 0.11 and the generated identity client contain 73 JSON operations.

The new STUDIO-PENDING-001 self-test runs through normal TestStudio publication and execution.
It covers a project-only manager, stale provider revisions, stale cancellation, unconfirmed preparation, two real local OIDC issuers, and mandatory verified email.
It verifies activation without merging accounts, later revocation, provider-change blocking, read-only recovery, private-edit clearing, keyboard submission, both themes, and mobile layout.
Run `20261103T095240.042580000-d952f2d4955758b43dc8116f` passes with complete evidence and cleaned execution resources.
Desktop and mobile screenshots were inspected. Mobile headings use a stacked layout.
Account-linking regression run `20261103T095157.662269000-9e854c6640282437e7e1aae0` also passes.

Focused API, assignment, identity, and generated-client checks pass on SQLite and local PostgreSQL with race detection.
Additional checks verify inherited-role changes, cross-instance cancellation races, exact recipient privacy, and native-role rollback after a late batch failure.
The UI build and pure pending-access and shared resource-validation checks pass.

Remaining OIDC acceptance, source management, execution parity, deployment verification, and the remaining full plan stay required.

### Fresh account proof and explicit identity linking

Added session-owned identity link operations, enabled provider choices, current-state reads, cancellation, and revision-controlled confirmation.
Requests require a reason and expire after five minutes or the earlier session or proof deadline.
Password accounts verify the current password through the durable limiter.
OIDC accounts prove the existing account through a fresh provider authentication.
Loopback operator confirmation remains explicit and unavailable to remote or internal fixture sessions.

Both OIDC steps reuse the normal code exchange and token verifier.
They request fresh authentication and require a bounded integer authentication time.
Source proof must resolve to the same internal user. Target proof never provisions users or synchronizes group memberships.
Purpose-bound state, nonce, PKCE, current provider revisions, and the initiating session protect callbacks.
New proof transactions supersede earlier callbacks without letting stale failures invalidate the new proof.

Proof exposes the exact issuer and subject for a separate confirmation.
Already linked identities retain their owner. No email matching, account merging, or identity transfer occurs.
One database transaction links the identity, audits the reason, rotates the current session, and invalidates older sessions.
Rotation preserves the original absolute session and authentication deadlines. An outer transaction rollback preserves the original session.
The new session can recover the completed operation after an unreadable confirmation response.

Your account provides focused source verification, target verification, exact proof review, cancellation, and completion views.
Conflicts and unknown mutation outcomes block repeat changes until explicit state review.
Submitted password values clear. Contextual help explains the complete flow.
OpenAPI 0.10 and the generated client contain 68 JSON operations.

The new STUDIO-LINKS-001 self-test runs through normal TestStudio publication and execution.
It covers password proof, incorrect credentials, cancellation, OIDC source ownership, target proof, and explicit confirmation.
It also covers unreadable confirmation recovery, session rotation, linked sign-in, keyboard confirmation, both themes, mobile layout, and unavailable-view recovery.
Run `20261103T092724.197236000-6ce7a6549b9d40fd8d58b93f` passes with complete evidence and cleaned execution resources.
Desktop and mobile screenshots were inspected.

Focused identity, session, local authentication, OIDC, API, and generated-client checks pass.
The linking, session, OIDC, and local authentication checks also pass on local PostgreSQL with race detection.
The UI build and pure linking utility checks pass.

Pending access, remaining OIDC acceptance, source management, deployment profiles, and the remaining full plan stay required.

### Managed providers, permanent group ownership, and browser sign-in

Added installation provider list, registration, detail, and revision-controlled configuration APIs.
Exact provider-object denies remain authoritative. Project administration grants no installation provider management.
Configuration changes share one permission and metadata transaction with audit, directory revision, and administrator protection.
Disabling or changing the only usable issuer rolls back configuration, revision, and audit changes.

Provider settings include exact issuer, client identifier, provisioning, scopes, claim mappings, membership lifetime, and explicit disabled state.
Duplicate scopes, invalid references, invalid mapping values, missing explicit state, and stale revisions fail without changes.
Enabling requires active mapped groups and projects.
Mapped destinations require their actual group and membership management authority. Exact membership denies remain effective.
Provider ownership can be declared only when creating a new group through the canonical group API.
Ownership remains permanent. Ordinary local membership edits reject provider-managed groups.

Operator-approved credential references use the new `--oidc-secret-refs` flag.
References in an operator-supplied initial provider file are also approved.
The management API accepts references without resolving their values. Runtime sign-in rejects unapproved reference resolution.
Provider responses and audit records contain no resolved credentials.
The OpenAPI 0.9 contract and generated client contain 63 operations.
String dictionary schemas retain their value type. Empty mappings and zero lifetime remain explicit in responses.

Identity providers and provider details provide focused registration, editing, group creation, mapping, and read-only workflows.
Conflicts and unconfirmed responses preserve fields and block repeat submission until explicit state review.
Revoked management clears private edits and views. Retained reading permits an explicit read-only refresh.
Contextual help, keyboard controls, shared themes, and narrow-screen layouts accompany both pages.

The real browser workflow found and fixed provider links that Angular canceled because their handler returned false.
The navigation guard now returns no value while preserving the concurrent-authentication check.
Saving the current sign-in provider clears private pages and refreshes public provider choices after session invalidation.
Users without project access receive an explicit Access not granted state.

The self-test wrapper provides two new loopback issuers with discovery, RSA signatures, one-time codes, PKCE, and approval pages.
The browser registers both providers through the UI and completes private-client and public-client sign-in.
It verifies exact mapped project access, invalidation after provider edits, and restoration through fresh claims.
Equal email and subject values at different issuers remain separate users. The second user receives no transferred project access.
A signed-token test also verifies equal emails with different subjects at one issuer and retains the first user's grants.
This evidence completes the duplicate-email identity-isolation acceptance item.

Full HTTP, access, OIDC, client, and startup suites pass with race detection on SQLite.
Focused provider, OIDC, mapping, and administrator checks pass on disposable PostgreSQL with native permission storage.
Generated-client checks, targeted vet checks, five pure utility checks, TypeScript checks, and production builds pass.
The final provider workflow passes: `20261103T084637.615214000-2a2d5073c236199857963814`.
Help and new navigation pass: `20261103T082839.290013000-75b8061caf2929159ccf9c42`.
The shared local-account regression passes: `20261103T083955.845681000-63851417aac7effa2121790f`.
Actual restart verification preserves provider settings, provider-group ownership, and revision guards: `output/access/access-verification.GQRpsL/summary.json`.
Visual review confirms loaded editor fields and controls in light mode and at 390 pixels in dark mode without document overflow.
Owned resource cleanup is verified before the feature commit.

Explicit fresh-authentication account linking, pending project access, provider logout, source releases, hosted profiles, and the remaining plan stay pending.

### Provider-bound authentication and membership state

OIDC login transactions now bind their stored provider revision and assert it after discovery.
Callbacks recheck the revision before remote discovery, before token exchange, and after token validation.
Configuration changes reject callbacks before provisioning. Consumed transactions cannot be replayed.
New-account creation asserts the provider revision with the user, exact identity link, and audit transaction.
Provider records require installation ownership and matching permanent identifiers.

Provider membership synchronization asserts its revision in the membership and audit transaction.
Membership provenance retains the revision. Directory lookup rejects stale, disabled, missing, and incorrectly scoped providers.
Local membership remains independent. Membership screens and administrator safeguards use the same current-state check.
An outdated external membership cannot act as the last usable administrator.

OIDC sessions retain an explicit authentication method and provider revision.
Creation asserts that revision with the user and session update.
Browser and evidence access reject stale provider sessions. Restoration revives neither older cookies nor older memberships.
Session administration lists only sessions accepted by the shared validator.
Provider names local and password remain distinct from authentication methods.
Legacy sessions without a method require sign-in again. Legacy OIDC memberships require a validated provider refresh.
These changes leave new sessions durable across ordinary restarts.

The OpenAPI 0.8.1 membership schema includes optional server-managed provider provenance.
Focused loopback tests cover provider edits before callbacks, during discovery, and during token exchange.
They cover explicit session modes, stale evidence access, membership withdrawal, validated refresh, and local membership independence.
OIDC, session, membership, administrator guard, HTTP access, and startup checks pass with race detection on SQLite.
Focused OIDC, session, membership, source, and access checks also pass on disposable PostgreSQL with native permission storage.
The new provisioning checks reject obsolete provider revisions and foreign issuer proof without identity writes.
Active session lists exclude earlier provider revisions and include freshly validated sessions.
Generated-client checks, targeted vet checks, and production Angular builds pass.
The final access administration browser regression passes: `20261103T075239.360985000-1f5f6084c07b354f0118415a`.
The permission explanation regression passes: `20261103T074852.739282000-dac8bd5618676a1986354b80`.
Both runs retain complete evidence and cleaned execution records.
Actual SQLite restart verification passes with explicit authentication methods: `output/access/access-verification.JAX4Zs/summary.json`.
Owned controllers, browser containers, output volumes, and disposable database containers exit after verification.
These changes establish provider configuration boundaries. They do not complete provider administration or full OIDC acceptance.
Provider management, fresh account linking, pending project access, source releases, hosted profiles, and the remaining plan stay pending.

### Effective access decisions and source explanations

Added self and authorized-user explanation APIs for exact project, installation, and resource contexts.
The decision uses the live permission service. Read-only library probes verify each direct or inherited role path.
The probes preserve actual principal, root assignment, binding, grant owner, disabled state, denies, expiry, and field semantics.
No second policy matcher or native permission writes are introduced.
Reasons distinguish allows, explicit denies, missing allows, and disabled users.

Source details require authority over their actual administration scope, principal, role, group, and managed grant.
Restricted rows omit private identifiers, scopes, names, bindings, expiry timestamps, and field details.
Source identities use inspector-bound opaque authentication codes. Native grant identifiers remain private.
Bounded continuations bind current directory, group, role-path, decision, actor, subject, action, and resource context.
Changed directory state and processed grant expiry reject stale continuation.
Empty partial scans retain the complete decision and permit further scanning without a total-source ceiling.

Explain project access and Explain installation access provide registered action selection and self-inspection.
Authorized managers can inspect another registered user.
The interface separates the complete decision from source-scan progress.
Cards show direct, assigned, inherited, expired, and restricted paths with precise scopes and expiry.
Changed fields discard reports and invalidate late responses. Failed refreshes and continuations clear private state.
Both pages provide contextual help, keyboard controls, shared themes, and narrow-screen layouts.

The OpenAPI 0.8 contract and generated client contain 59 identity and access operations.
Required query parameters now require the query container in generated request types.
The built-service restart verifier preserves explanation identities and verifies inactive and restored inherited access.
Its retained summary is `output/access/access-verification.ShC9BP/summary.json`.

Focused source, privacy, stale-context, grant, custom-role, predefined-role, and client checks pass with race detection on SQLite.
Source checks also pass on disposable PostgreSQL, including actual foreign assignment bindings and scoped source redaction.
Generated-client checks, targeted vet checks, pure presentation checks, TypeScript checks, and production builds pass.
The normal Playwright workflow verifies active denies, expiry, direct and inherited paths, self-inspection, restricted groups, and installation denial.
It verifies bounded continuation, explicit conflict recovery, discarded late responses, and revoked administrative inspection with retained evidence.
The final explanation workflow passes: `20261103T073014.785326000-005a226812b3e6f723ee4679`.
Contextual help and navigation pass: `20261103T072814.022520000-78634560dda57dbe23d9955b`.
Predefined role regression passes: `20261103T073122.920006000-e4b99c8f38befc1986e5d421`.
Visual review confirms loaded source cards and decisions in light mode and at 390 pixels in dark mode without document overflow.
Owned controllers, browser containers, output volumes, and disposable databases exit after verification.
This evidence completes the authorized user, group, membership, role, assignment, and effective-permission workflow acceptance item.
Identity linking, provider workflows, source releases, hosted profiles, and the remaining full plan stay pending.

### Predefined role overrides and fresh authorization state

Added inspection and revision-controlled state changes for all six predefined roles.
Definitions remain code managed. Disabling applies across the installation while retaining assignments and default permissions.
Management requires installation system and role authority, delegation, and current authority for every default action.
Exact role-object denies and project boundaries remain authoritative.
Public revisions represent native state before its first managed change and advance through subsequent changes and recovery.
Native overrides, management metadata, reasons, audit, and directory revisions share one policy transaction.
The last-administrator guard rejects disabling the only usable administration path.
Independent connection checks prevent competing role disabling and alternate account removal from removing both usable paths.

Authorization now reads current predefined state before using a library evaluation engine.
The engine receives an immutable disabled-role set. Its internal cache cannot retain an older role state across later checks.
Storage errors prevent reuse of an earlier allowed decision.
Role choices, assignment inspection, and administrator guards use the same fresh state.
Permission decisions still use the permission library.

Installation roles provides global state controls, default permission inspection, managed timestamps, and current revision values.
Project role pages retain read-only predefined state.
Reason forms preserve their opened revision during visible refresh. Conflicts and unconfirmed responses block repeat submission.
Revoked controls clear private state. Users with retained inspection authority can explicitly refresh a read-only view.
Contextual help includes state-change tutorials. Catalog columns retain readable identifiers on narrow screens.

Explicit loopback recovery now restores disabled predefined administration, updates override metadata, and invalidates earlier administrator sessions atomically.
Ordinary startup preserves disabled roles.
The built-service verifier passes across actual SQLite restarts with users, memberships, assignments, direct grants, disabled custom roles, predefined overrides, and sessions.
It also verifies retained access restoration, ordinary startup behavior, explicit recovery, and stale post-recovery revision rejection.
Its summary is retained in `output/access/access-verification.kOrzP6/summary.json`.
This evidence closes the SQLite restart-persistence acceptance item.

Focused override, immediate state, fail-closed, recovery, bootstrap, assignment, custom policy, and administrator guard race checks pass on SQLite and disposable PostgreSQL.
The OpenAPI 0.7 contract and generated client contain 57 identity and access operations.
Generated-client checks, targeted vet checks, four form utility checks, and production Angular builds pass.
STUDIO-BUILTIN-001 verifies real cross-project access withdrawal and restoration, retained assignments, code definition stability, and disabled assignment choices.
It verifies conflict reasons, unconfirmed responses, read-only inspection, lost management with retained reads, complete read revocation, help, and administrator protection.
The final workflow passes with complete evidence: `20261103T061102.231047000-236701edf597558d334165a3`.
The final custom-policy regression passes: `20261103T060841.568656000-2b23ec72ec1e773361e901a7`.
Help passes: `20261103T060302.025951000-e57122d704c6a612817185e0`.
Assignment administration passes: `20261103T060318.354154000-4c5c5955b3533d676fd0b480`.
Visual review confirms loaded role cards and confirmation forms in both themes and at 390 pixels, with no document overflow or unexpected browser errors.
Owned controllers, browser containers, output volumes, and disposable database containers exit after verification.

Complete effective-access source explanations, identity and provider workflows, source releases, hosted profiles, and the remaining full plan stay pending.

### Direct user and group grant interface

Added Project permissions and Installation permissions pages through the generated access client.
Focused services manage bounded lists, exact context permission choices, creation, expiry editing, and reason-based removal.
Server capabilities control actions. Principal activity, exact objects, explicit effects, UTC expiry, and retained public revisions remain inspectable.
Expiry editors preserve their opened revision and unsaved values during visible refresh and conflict.
Changing the creation object invalidates permission choices until explicit refresh.
Permission catalog inspection accepts role read or permission read authority without granting delegation.
Both pages provide contextual Introduction, Concepts, and Tutorials help.
Shared theme tokens and contained table regions support light, dark, and mobile layouts.
The mobile editor constrains native select width while retaining readable permission descriptions.

Unconfirmed saves and removals preserve entered reasons and block repeat submission until explicit state review.
Shared reason actions now apply the same uncertainty guard to other administration pages.
Unreadable error bodies retain their HTTP rejection status. Revoked access clears private editor and list state even with unreadable responses.
Scope changes discard old editors and late responses.

STUDIO-GRANTS-001 verifies actual exact-object access, inherited group deny precedence, expiry, deny removal, recreation conflicts, and retained reasons.
It verifies unconfirmed successful saves and removals, read-only sessions, unreadable revoked-access rejections, administrator protection, and bounded continuation.
Editor and list screenshots pass visual review. Both themes and mobile widths pass without document overflow or unexpected browser runtime errors.
The final grant workflow passes with complete evidence: `20261103T054254.219959000-9b9843d7565b8b368a65afe9`.
Expanded help passes: `20261103T054307.513038000-2605cdc5ba5b126228486c7a`.
Role assignment regression passes: `20261103T054324.533775000-77f7006d6f403d91f7516e52`.
Managed local account regression passes: `20261103T054337.613978000-91e050a3ec98e3f8ead552ba`.
Focused direct grant, custom role, and preview race checks pass on SQLite and disposable local PostgreSQL after the catalog authority change.
Four pure form utility checks, generated-client checks, targeted vet checks, and production Angular builds pass.
Owned controllers, browser containers, output volumes, and disposable PostgreSQL containers exit after successful and failed verification runs.

Complete effective-access source explanations, built-in override administration, identity and provider workflows, and the remaining full plan stay pending.

### Managed direct user and group grant APIs

Added scoped direct grant listing, creation, detail, expiry updates, and revision-controlled removal.
Rules retain user or group ownership, exact project and object scope, allow or deny effect, and UTC microsecond expiry.
Shared rule validation keeps custom role and direct grant semantics aligned, including supported UTC years.
The permission library evaluates current grants, role inheritance, active membership, deny precedence, and expiry.
Reads and mutations check scope, principal, grant, and object authority. Shared groups require authority in their installation scope.
Inactive principals and archived projects reject new or renewed access while permitting authorized removal.
Native grant identifiers stay private. Unrelated native identifiers remain stable.
Retained management records preserve public identifiers and advance revisions through removal and recreation.
Exact native inspection rejects missing or inconsistent state. Unmanaged native duplicates are not adopted.
Native changes, management metadata, audit reasons, and directory revisions commit atomically.
Project revision assertions protect new and renewed grants against concurrent archive changes.

Strengthened the usable-administrator definition to include permission inspection, role administration, and membership administration.
Direct denies cannot remove those essential actions from the only usable administrator.
Independent connection checks verify that competing administrator denies commit at most one removal path.
Rejected changes preserve native state, audit history, and the directory revision.

Focused direct grant, custom policy, assignment, account guard, and generated-client race checks pass on SQLite and disposable local PostgreSQL.
Checks cover read-only sessions, filtered object denies, foreign scopes, expired group denies, disabled users, archived groups, and stale revisions.
Native grant discovery scans beyond 1000 existing grants without an arbitrary total ceiling.
The built-service check verifies actual SQLite restarts, retained direct grants and expiry, deny precedence, existing-session access revocation, and recreated revisions.
Its summary is retained in `output/access/access-verification.J4pSuu/summary.json`.
The OpenAPI 0.6 contract and generated client now contain 54 identity and access operations.
Pure interface utility checks, generated-client checks, targeted vet checks, and the production Angular build pass.
The custom-policy Playwright regression passes through normal TestStudio execution with complete evidence: `20261103T052916.550828000-2bbf5c6e6d855ce26c4200ee`.
Owned verification controllers, browser containers, output volumes, and disposable database containers exit after checks.

Direct grant screens, complete effective-access source explanations, built-in override administration, and the remaining full plan stay pending.

### Custom policy interface and temporary access preview

Added installation and project Roles and permissions pages, scoped detail views, and focused policy editor components.
Editors show registered action descriptions, explicit effects, exact object scope, precise UTC expiry, and bounded inheritance choices.
Server-computed creation, management, and delegation capabilities control the interface.
Predefined and custom assignment choices share existing guarded workflows. Custom choices expose exact policy links and bounded continuation.
The editor retains its opened revision through visible refreshes. Conflicts preserve entries and require explicit reload before another save.
Lost role read access clears private details and open editor values, including after a rejected preview.
State actions identify their target and impact, require reasons, and retain last-administrator protection.
Both pages provide contextual Introduction, Concepts, and Tutorials help with preserved unsaved values.

Added a proposed policy preview through the generated API client.
Current and proposed decisions use the permission service in one transaction that always rolls back temporary policy, assignment, audit, and directory changes.
The preview checks subject administration and permission inspection authority. Existing grants, group roles, denies, and disabled users remain effective.
Changed decisions appear by default. Unchanged permissions remain available, and altered inputs invalidate the displayed preview.
The OpenAPI 0.5 contract and generated identity client now contain 44 operations.

STUDIO-POLICY-001 verifies real policy creation, exact rules, expiry, inheritance, group assignment, conflict retention, and explicit disabling and restoration.
It verifies preview rollback, stale comparisons, read-only controls, lost access during editing, bounded pages, and the last usable administrator guard.
Light, dark, 390-pixel, and contextual help workflows pass with complete evidence and no unexpected browser runtime errors.
The final policy workflow passes: `20261103T050900.119419000-aa688a7b9b8fe9b0bf7d7235`.
Help passes: `20261103T050204.102684000-a29eef838ee0d20a41ed1588`.
Assignments pass: `20261103T050221.000765000-34bc3f75005bf38763decca3`.
Stacked trends pass: `20261103T050451.649733000-d5d95cd39733ed77f297b592`.
Existing help, assignment, and stacked-trend browser workflows pass after the shared navigation and row-header changes.
Visual review confirms readable exact identifiers, theme tokens, and contained table scrolling.
Focused policy, capability, preview, assignment, administrator guard, and generated-client race checks pass on SQLite and disposable local PostgreSQL.
Three pure interface utility checks and the production Angular build pass.
Owned browser containers, output volumes, PostgreSQL containers, and verification controllers exit after checks.

Direct principal grant management, built-in override administration, complete effective-access source explanations, and the remaining full plan stay pending.

### Managed custom role APIs

Added custom role creation, detail, bounded listing, and revision-based updates through the identity API.
Policies retain permanent global identifiers and owning scopes, names, descriptions, explicit rule effects, exact object rules, expiry, inheritance, and disabled state.
The service validates registered actions and stored resource ownership. Project policies cannot contain installation actions or inherit installation roles.
Custom delegation requires current rule actions and explicit delegation authority through inherited policies.
Edits also require authority over affected parent roles. A child update cannot bypass an ancestor's delegation deny.
Existing assignment APIs now support managed custom roles with exact route bindings and existing group ownership guards.
Disabling withdraws native rules and outgoing inheritance while retaining the logical definition.
Enabling revalidates and restores the retained policy. Unchanged rules preserve their native grant identifiers.
Installation changes preserve the last usable administrator against the proposed native state.
Policy metadata, ownership index, native permission changes, audit, and directory revision share one transaction.

Added registered action descriptions and exact-context grantability through the permission catalog API.
The OpenAPI 0.4 contract and generated identity client now contain 42 operations.
Focused race checks pass on SQLite and disposable local PostgreSQL for role CRUD, exact object boundaries, inheritance, deny precedence, expiry, and disabled access.
They also verify stale edits, immutable ownership, rejected server fields, ancestor restrictions, bounded filtered pages, cursor binding, and last-administrator rollback.
The built-service check passes across actual SQLite restarts with a new managed user and inherited group assignment.
It verifies retained definitions, exact-object access, disabled inherited access, and explicit restoration.
Evidence is retained in `output/access/access-verification.KpIKdZ/summary.json`.
Generated-client checks and the production Angular build pass. Owned controllers and disposable PostgreSQL containers exit after verification.

Custom role editing and assignment controls, policy previews, effective-access explanations, direct principal grants, built-in override APIs, and the remaining full plan stay pending.

### Atomic custom policy storage

Added a separate policy transaction without widening the existing PostgreSQL assignment-only transaction contract.
Custom role definitions, grants, inheritance, assignments, disabled overrides, and metadata now share a transaction on SQLite and PostgreSQL.
The shared directory revision locks before authorization and rejects competing changes through independent connections.
The permission library evaluates pending policy writes. Rejection rolls back policy, audit, and directory state together.
Added transaction-only single-grant removal and inheritance removal for operations absent from the pinned service interface.
Removal preserves unrelated grant identifiers and alternate inheritance paths.
PostgreSQL closure rebuilds in the same transaction as direct-edge removal.
Both stores reject missing inheritance endpoints, cycles, and reserved custom role definitions.
PostgreSQL role display names can repeat, matching SQLite. Permanent role identifiers remain unique.
Literal project and object grants remain visible in PostgreSQL inspection outside a transaction.
The library still evaluates scope, expiry, explicit denies, and role lifecycle state.

Focused race checks pass on SQLite and disposable local PostgreSQL.
They verify pending role and grant reads, complete rollback, exact object boundaries, grant pagination, and concurrent policy conflicts.
They also verify retained grant identifiers, alternate inheritance paths, and removal rollback.
Existing assignment, administrator guard, and managed local-account checks pass on both databases.
No acceptance item is closed by storage work alone.
Custom policy APIs, delegated administration, effective-access explanations, interfaces, and the remaining full plan stay pending.

### Managed local account interface and browser acceptance

Added managed password sign-in beside configured OIDC choices and the separate explicit loopback development action.
Required password changes replace private pages with a restricted form. Workspace requests wait until that restriction clears.
Your account exposes the caller's profile, bounded external identities, credential state, and own-password change without installation directory authority.
Desktop and mobile navigation provide access to the account page.
Authorized User details provide provisioning, reset, enable, and disable through the generated identity client.
Credential editors preserve their opened revision through visible refreshes.
Conflicts clear submitted secrets, retain reasons, and require explicit review before another save.
Self changes clear private workspace state and return to sign-in without requesting the revoked account page.
Authentication changes serialize requests and reject stale session and project responses.
All new pages provide Introduction, Concepts, and Tutorials help with shared light and dark themes.

Added STUDIO-LOCAL-001 through normal TestStudio publication, execution, timing, and private evidence storage.
It verifies password-only administration, required changes, last-administrator protection, concurrent reset conflicts, and reason retention.
It also verifies ordinary password changes, cookie rotation, disabled reset behavior, revoked sessions, and mode restart invalidation.
A successful server change with unreadable response requires explicit refresh and produces no duplicate password submission.
An unauthorized response clears the private page even when its response body is unreadable.
Required sessions make no ordinary private page requests. Account and credential controls respect actual project-only permissions.
The isolated test wrapper supports local authentication restarts. Production exposes no mode-switching endpoint.
The final account run passes with complete evidence: `20261103T040250.747799000-7b0b5f4c1113f37c89911ca8`.
Expanded help passes: `20261103T040013.406913000-af17d4812ad197fa33aa703f`.
The existing user-administration workflow passes: `20261103T040104.171381000-bf887077304f8d5e04e9891a`.
Focused password-form utilities, generated-client checks, and the production Angular build pass.
Focused local-account, bootstrap, recovery, OIDC flow, and session checks pass with race detection.
Retained screenshots pass visual review in both themes and at 390 pixels. No document overflow or unexpected browser runtime errors remain.
Owned browser containers, output volumes, and verification controllers exit after checks.

Identity linking, custom policies, pending project access, provider management, and deployment verification remain pending.
The remaining acceptance criteria stay required scope.

### Users, scoped groups, and session administration interface

Added Users and User details for pending profile registration, permitted profile edits, account disabling, restoration, and active session revocation.
Details expose linked identities, readable local and external membership, account metadata, and bounded session pages without session credentials.
Added separate installation and project group directories, metadata editing, local membership management, UTC expiry, archive, and restore.
Installation navigation uses installation capabilities independently from project capabilities.
Editors preserve permanent identifiers and scope, strip server-owned fields, retain unsaved values on conflicts, and provide explicit reload actions.
Lifecycle and removal actions display the target and impact, require a reason, and preserve server administrator protection.
Shared collection state discards stale responses after route changes and clears cached data after authorization failures.
Added four contextual help guides with Introduction, Concepts, and Tutorials.
The new interface uses the generated OpenAPI client and shared light and dark theme tokens.
Validation: focused utilities, generated-client checks, the Angular production build, and scoped API race checks pass.
The membership read and legacy user response checks pass on SQLite and local PostgreSQL.
STUDIO-ACCESS-001 passes through TestStudio's normal publication, execution, timing, and private evidence workflow.
It verifies actual profile and membership conflicts, reason validation, UTC expiry, local membership removal, retained archive and restore, pagination, and scope changes.
It also verifies final-administrator rejection and revocation of a second real session while preserving the current browser session.
An isolated preparer uses product directory, role, membership, and session services for project-administrator and external membership checks.
The browser verifies hidden installation navigation, rejected installation and foreign-project views, and read-only provider-managed membership through the actual server.
UTC expiry remains correct in a non-UTC browser timezone.
Expected HTTP rejections remain explicit; unrelated browser errors still fail the workflow.
Light, dark, and 390-pixel views pass workflow and document-width checks. Visual inspection confirms the layout.
STUDIO-HELP-001 passes with all four new guides and existing keyboard, focus, dismissal, theme, and mobile behavior.
STUDIO-NOTIFY-001 passes again after the shared fixture and shutdown changes.
Owned verification controllers and browser containers stop after checks. The disposable PostgreSQL service also stops.
Identity linking, pending project access, personal profile UI, role and grant editors, deployment verification, and the remaining full plan stay pending.

### Managed local account backend

Added opt-in `--local-accounts` alongside OIDC and the separate loopback development administrator.
Managed credentials use permanent internal user identifiers and grant no implicit project access.
Added redacted credential reads, administrator provisioning and reset, enable, disable, local sign-in, and own-password change APIs.
The OpenAPI contract and generated client now contain 37 identity and access operations.

Passwords use the pinned Argon2id implementation, independent salts, bounded memory concurrency, Unicode normalization, and explicit length limits.
Durable bounded counters include unknown identifiers and serialize reservations across connections.
Login errors conceal credential existence. Origin checks protect login; authenticated password changes also require the session token.
Public password deployments require HTTPS. Loopback HTTP also requires a loopback listener.

Resets create restricted sessions until the password changes.
A successful change rotates the cookie and atomically saves the credential, account session generation, audit, and replacement session.
The response sends no cookie before commit.
Credential revisions reject password proof obtained before a reset.
Durable mode revisions prevent old cookies from reviving after authentication is disabled and enabled again.
OIDC provider identifiers cannot be confused with the password authentication method.
Credential responses and audit records contain no passwords or hashes.

Last-administrator checks now include usable local credentials and actual authentication mode.
The common directory revision locks before record changes, including ordinary identity and session writes.
Targeted password, limiter, session, credential, administrator, assignment, bootstrap, and client checks pass with race detection on SQLite and local PostgreSQL.
The generated client passes the TypeScript check.
The built-service smoke passes across actual restarts with complete redacted evidence in `output/local-accounts/local-accounts-verification.dWb0S1/summary.json`.
It verifies password-only administration, forced change, cookie rotation, last-administrator protection, mode revocation, generic errors, and transport rejection.
Owned test controllers and the PostgreSQL container exit after verification.

Managed password screens, browser acceptance, identity linking, custom policy management, hosted verification, and the remaining full plan stay pending.

### Scoped assignment interface

Added Project access and Installation access screens with predefined role choices and permission details.
The generated client handles user and group assignment creation, bounded pages, and exact revision-based removal.
The service returns per-assignment removal authority. Shared-group controls use that result instead of inferred project privileges.
Forms preserve creation conflicts and stale-removal reason fields until explicit review.
Scope changes invalidate old requests and close previous editors and actions.
Self-revocation clears unavailable collections and refreshes the project directory and navigation capabilities.
Both pages provide contextual help and shared light and dark themes.
Mobile tables preserve readable columns inside a keyboard-focusable horizontal scroll region.

Added `STUDIO-ROLES-001` to normal TestStudio self-tests.
The isolated fixture now includes a project administrator and a read-only access reviewer.
The browser verifies real assignment mutations, recreation conflicts, exact project removal, shared-group restrictions, role delegation, and administrator protection.
It also verifies blocked installation and foreign-project views, read-only controls, self-revocation, bounded pages, preserved help edits, and both themes.
The final assignment run passes with complete evidence: `20261103T025912.895503000-47da3217fdbc9caf1d095c9f`.
Expanded help passes: `20261103T025926.025821000-16446d2f91cf5b6422877d15`.
The identity workflow passes with the updated fixture and navigation: `20261103T025943.041337000-44d75a7033dd3d027f5a5f64`.
Retained screenshots pass visual review. No unexpected browser runtime errors or document overflow remain in these workflows.
Owned controllers, browser containers, and output volumes are gone after verification.

Targeted API and generated-client checks pass on SQLite and local PostgreSQL with race detection.
Five focused pure utility checks and the Angular production build pass.
The utility runner accepts selected test files, compiles runtime dependencies through the pinned TypeScript package, and removes temporary output.
Custom policy editors, effective-access explanations, pending project access, managed local credentials, and the remaining full plan stay pending.

### Scoped predefined assignment APIs

Added predefined role choices and bounded assignment APIs for installation and project scopes.
The route determines exact bindings. Unknown body fields cannot widen them.
User and group assignments require role, principal, membership, and delegation checks.
Group mutations also require authority in the group's stored ownership scope.
Project administrators can delegate predefined project roles without receiving each delegated action themselves.
Installation groups can hold different project roles. Project administrators cannot change those shared groups' assignments.
Single-assignment removal uses a reason and current revision. Retained revision history prevents stale removal after recreation.
Legacy bootstrap assignments remain visible before their first managed mutation.
Assignment, management metadata, audit, and directory changes share one transaction.
Last-administrator checks use the proposed permission state and serialize against competing account disabling across independent connections.

Added nine operations to OpenAPI and generated identity clients. The generated contract now contains 30 operations.
Targeted assignment, guard, bootstrap, and client checks pass on SQLite and local PostgreSQL with race detection.
The generated client passes the TypeScript check.
The built-service smoke verifies assignment persistence, revocation, and recreation across actual local restarts.
It retains evidence in `output/access/access-verification.7FBjxj/summary.json`.

Ordinary startup now preserves revoked administrator roles.
Bootstrap creation, role assignment, initialization markers, and audit records share a transaction.
Existing local and external bootstrap profiles retain their disabled state and current assignments during marker adoption.
Explicit loopback recovery restores the local role and invalidates earlier sessions atomically.
Recovery cannot validate a stale assignment-removal revision.

Assignment interfaces, custom policy management, pending project access, managed local credentials, and the remaining full plan stay pending.

### Local and OIDC account decision

The plan now supports local accounts, OIDC accounts, or both in one installation.
Both sign-in methods use the same internal directory, sessions, groups, and permission service.
Managed local credentials, password changes, administrator resets, and sign-in controls remain pending.
The current loopback development administrator does not satisfy the managed local account requirements.

### Atomic assignment storage

Added transaction-bound record reads, writes, assertions, pages, and counters.
SQLite permission assignments now share the metadata transaction.
PostgreSQL uses its existing native assignment tables inside that same transaction.
PostgreSQL rejects unrelated policy writes through the assignment transaction adapter.
Authorization uses a fresh permission service and transaction-bound identity directory.
The directory revision advances before authorization reads and serializes competing identity and assignment changes.
Targeted checks pass with race detection on SQLite and local PostgreSQL.
They verify pending permission evaluation, explicit denies, exact revocation, idempotent assignment, concurrent conflicts, and complete rollback with audit and directory records.
Existing access API and administrator guard checks also pass on both databases.
The administrator-facing assignment API, editors, and custom policy transactions remain pending.

### Direct project membership decision

Use owner-managed project membership instead of invitations.
Existing users receive explicitly scoped role assignments from an authorized project owner.
Pending access requires an allowed provider and a verified email before activation.
Pending access cannot merge accounts or grant installation administration.
Invitation delivery, acceptance links, and invitation expiry are outside the initial plan.
Assignment administration and pending access activation remain incomplete.

### Generated identity client and direct membership reads

Added deterministic TypeScript request types, response types, operation methods, and route templates from the identity OpenAPI contract.
The shared adapter preserves session, request-forgery, revision, identifier escaping, and explicit query semantics.
Added generation and stale-output checks to the Go tooling.
Added direct, authorized membership reads with current revisions and source metadata for explicit editor reloads.
Local and external sources use the same ownership and member-read checks as membership pages.
User read views supply missing legacy session generations without rewriting stored revisions.
Validation: generator checks, client-path checks, focused API race tests, and local PostgreSQL checks pass.
The Angular build accepts the generated types and adapter.
The full plan and remaining API contract families stay pending.

### Identity administration API and durable session invalidation

Added user profiles, account state, linked identity inspection, active session inspection, and single-session revocation through authorized APIs.
Added permanent group ownership, local membership edits, provenance and expiry views, archive, and restore.
Stored ownership drives checks. Project administrators cannot reparent groups or edit shared installation groups.
Provider-managed membership remains read-only. New profiles and memberships do not grant project access.
Current profile access requires authentication without installation directory authority.
Identity writes save revisions and audit data atomically. Destructive lifecycle actions require reasons.
Shared directory versions fence competing administrator removals across service instances.
The permission library evaluates proposed directory state, including alternate group paths, explicit denies, and configured sign-in paths.
Opaque encrypted cursors hide filtered identifiers and bind actor, collection, resource, scope, and archive filters.
Added a partial OpenAPI contract for all 20 implemented identity administration operations.
Other route families and role administration remain pending in that contract.
Sign-in saves its timestamp atomically. Account disabling advances a session generation, and restoration cannot revive old sessions or artifact access.
Added explicit loopback-only operator recovery with audit records and invalidation of earlier local sessions.
Validation: focused race checks pass on SQLite and local PostgreSQL.
Checks cover cross-project and installation boundaries, source ownership, object denies, bounded pages, forged cursors, stale revisions, administrator races, and session secrets.
Existing OIDC and permission checks pass after the shared directory and session changes.
The built-service smoke verifies actual SQLite restarts, persisted profiles and membership, cursor continuity, ordinary session continuity, explicit recovery invalidation, and loopback restrictions.
Owned controllers and the disposable PostgreSQL service stop after verification.
At this stage, Users and Groups screens, identity linking, role and grant administration, hosted verification, and the remaining full plan stay pending.

### Notification configuration and delivery interface

Added project channel configuration with registered secret names, exact scope filters, trigger selection, distinct timeout outcomes, and bounded attempt budgets.
New channels start paused. The editor retains unsaved values during polling and help, and closes them on project changes.
Revision errors retain edits and provide an explicit configuration reload.
Channel controls pause, enable, archive, and restore without removing retained history.
Project and run views share delivery history, frozen summaries, attempt responses, explicit retries, and cancellation of waiting work.
Test outcomes remain separate from delivery outcomes. Unknown acceptance and expired checkpoint leases remain explicit.
Bounded history exposes coverage and pauses automatic refresh after older pages load.
Added Notifications help with Introduction, Concepts, and Tutorials.
Added the permanent STUDIO-NOTIFY-001 workflow and a fresh isolated receiver inside the Playwright worker.
The fixture uses new project bindings and fresh secrets. It requires no external network or cloud credentials.
Validation: focused notification and timeout-rerun utilities pass. The Angular production build passes.
STUDIO-NOTIFY-001 passes through normal publication, execution, timing, and evidence storage.
It verifies real manual and scheduled delivery, transient retries, permanent Discord failures, explicit retries, waiting cancellation, stable identifiers, and retained history.
It also verifies project reference isolation, preserved edits, contextual help, light and dark themes, and mobile document overflow.
Browser diagnostics contain no runtime errors. Visual inspection confirms readable light and dark layouts.
STUDIO-TIMEOUT-001 passes again and preserves the original timed-out result after a successful UI failure rerun.
STUDIO-HELP-001 passes again with the new Notifications guide and existing keyboard, dismissal, theme, and mobile checks.
Verification workers finish with complete evidence and cleaned resources. Owned controllers and browser containers stop after checks.
Hosted secret adapters, cloud deployment checks, and the remaining full plan stay pending.

### Durable completion outbox and notification API

Run creation freezes channel scope and outcome rules without exposing resolved secrets.
Terminal transitions save delivery work atomically for manual, scheduled, cancelled, timed-out, and interrupted runs.
Delivery workers claim durable attempts, retain bounded retry budgets, and recover unknown acceptance after restart.
Stable identifiers allow receiver deduplication. Recovery preserves at-least-once uncertainty without rerunning tests.
Destination and Discord global cooldowns respect provider delays.
Pausing and destination changes invalidate earlier delivery epochs. Explicit retry records new authorization while retaining the original snapshot.
Every send checks current channel ownership, project state, notification management, and run access.
Added scoped configuration, reference, delivery, attempt, retry, stop, archive, and restore APIs.
Delivery history uses query-bound cursors, bounded scans, per-run access checks, and visible partial coverage.
The local service loads environment or mounted-file bindings and starts durable delivery workers by default.
Validation: focused race checks pass on SQLite and local PostgreSQL.
Actual local HTTP receivers verify manual and scheduled delivery, retry delays, permanent failures, policy snapshots, revocation, and uncertainty after restart.
The built-service smoke workflow verifies native publication, real scheduling, retained 503 retries, stable identifiers, configured links, and secret-free API data.
The notification interface, its Playwright workflow, hosted secret services, and the remaining full plan stay pending at this backend checkpoint.

### Notification transports and scoped local secret bindings

Added purpose-scoped environment and mounted-file secret bindings with project isolation and bounded reads.
Reference listings expose identifiers and names without resolved values or source configuration.
Native probes and notification senders share DNS and address policy checks.
Added generic JSON and Discord webhook transports with stable delivery identifiers, confirmation, bounded requests, and redacted diagnostics.
Discord delivery disables mentions and respects rate-limit responses.
Validation: local race checks pass for secret resolution, network policies, webhook payloads, redirects, permanent failures, and rate-limit parsing.
Existing native HTTP executor checks pass through the shared network guard.
Durable delivery, API and UI workflows, hosted secret adapters, and the remaining full plan remain pending at this transport checkpoint.

### Schedule editor, previews, and firing history

Added project navigation and focused schedule editor, listing, and firing-history components.
The editor selects suites, targets, locations, release pins, tests, run limits, retries, cadence, timezones, and overlap behavior.
Preview validates the normal execution selection and shows five future UTC times with timing readiness. It creates no run.
Polling updates runtime state without changing unsaved editor fields or configuration revisions.
Enable, pause, archive, and restore use the same scoped API and revision checks.
Triggered firings open normal run details. Blocked and skipped firings retain visible reasons and coalescing notices.
Run snapshots link back to their schedule and retain the original due time.
Project changes close stale editors. Structured edits preserve request options and exclude server-owned state.
Added Schedules help with Introduction, Concepts, and Tutorials.
Validation: the Angular production build and focused form transformation checks pass.
STUDIO-SCHEDULE-001 executes the real local scheduler through TestStudio's own Playwright runner.
It verifies preview, firing, unsaved editing, stable revisions, pause across a due interval, normal timing and evidence, blocked catalog changes, archival, and restoration.
It also verifies cron previews, preserved help edits, project changes, and light and dark mobile containment.
The expanded contextual help workflow passes with the new page.
Notification transports, environment exclusivity, provider execution adapters, and the remaining full plan remain pending.

### Durable schedule API and atomic run creation

Added project schedules with bounded intervals, five-field cron expressions, explicit timezones, release pins, and overlap policies.
The service scheduler claims durable occurrences and recovers expired claims after restart.
One transaction queues the normal run, checkpoints its firing, and advances the schedule.
Configuration changes fence in-flight claims. Runtime updates preserve public configuration revisions for editors.
Missed occurrences coalesce without a catch-up burst. Independent schedules retain independent overlap rules.
Scheduled runs use current owner permissions and normal catalog, target, timing, evidence, and dispatch checks.
Triggered, skipped, and blocked firings retain project-scoped history.
Archival preserves identifiers and history. Restored schedules stay disabled.
Validation: targeted race checks pass on SQLite and local PostgreSQL.
Checks cover competing schedulers, restart recovery, disable races, stable editor revisions, release activation and pins, overlap, and revoked owners.
API checks verify preview, revision conflicts, firing links, archival, project isolation, and object denies.
The schedule interface, its browser workflow, and completion notifications remain pending at this backend checkpoint.

### Isolated local Go source runner

Added an immutable Go image manifest with permanent identifiers and explicit package and test selectors.
Publication rejects recursive packages, injected patterns, and overlapping parent and child registrations.
Compilation stays separate from test duration. Execution disables result caching and records structured test events.
Parent aggregates and independently registered child tests preserve logical counts. Dynamic descendants remain recorded steps.
Assertions, panics, skips, timeouts, build failures, and missing tests retain distinct observations.
Timeouts retain their separate status and count. Normal failure reruns include them.
Preparation problems block the run without assertion retries or invented execution duration.
Runtime labels separate toolchain and platform timing history without resetting compatible source revisions.
The shared local container transport retains bounded evidence and selects recovery by the durable runner profile.
Go diagnostics appear in attempt history with toolchain, platform, build duration, and available exit codes.
Validation: focused race checks pass for selector rules, actual Go source execution, catalog publication, timing, runs, and transport contracts.
Actual Docker checks pass for normal and diagnostic outcomes, cancellation, retained checkpoint recovery, ownership, and resource cleanup.
Focused Go catalog, dispatch, timeout rerun, and recovery checks pass on SQLite and local PostgreSQL.
The full API workflow publishes a new suite, runs four logical checks, learns timing, and verifies diagnostic selections and timeout reruns.
The Angular production build passes. The existing Playwright timeout self-test passes through the shared transport.
Local browser checks verify Go diagnostics in light and dark themes, mobile containment, unavailable build results, and clean browser logs.
External Go targets, bindings, hosted workers, and source preparation screens remain pending.

### Shared native HTTP definition validation

Added shared publication and execution validation for HTTP methods, statuses, response assertions, and numeric thresholds.
Limits bound text, headers, JSON paths, and expected JSON content.
HEAD body assertions remain invalid. Header names stay unique without case differences.
Invalid new drafts cannot replace an active release.
Validation: catalog and HTTP executor race checks pass, including real HEAD requests and response-time failures.
Publication and activation checks pass on SQLite and local PostgreSQL.
The HTTPS editor and diagnostics interface remain under verification at this checkpoint.

### Declarative HTTP editor and retained network diagnostics

Added focused request, status, body, header, JSON, response-time, and certificate-lifetime controls to the native check editor.
Pure parsers validate field values and preserve exact body text and structured JSON values.
HEAD conflicts remain visible. Users clear body assertions explicitly instead of losing fields when changing the method.
Metadata-only edits retain assertions. New publications keep earlier run snapshots unchanged.
Catalog filters now include test type independently of the runner.
New catalog actions clear previous success messages before reporting their own result.
Attempt details show retained probe timing, network phases, certificate expiry, and lifetime at observation.
Missing phases stay unavailable. Measured zeros remain valid.
Validation: the Angular production build and six focused editor, filtering, and diagnostics utility checks pass.
The HTTP self-test creates, validates, publishes, edits, and executes checks through the existing API and UI.
Checks cover rejected fields, retained assertions, HEAD conflicts, real GET and HEAD requests, and immutable earlier results.
The existing basic catalog publication and native execution workflow also passes.
Local screenshots verify light diagnostics and the dark mobile editor without document overflow or browser runtime errors.
Both workflows retain complete evidence and cleaned worker records. Owned containers, volumes, and controllers stop after checks.
Scheduled checks, credential bindings, project trust settings, response excerpts, and response-time history remain pending.

### Result-state trend data and rankings

Added an authorized project trend API with bounded history pages and scoped cursors.
Aggregates use immutable selected-test snapshots and final logical results. Retry attempts do not inflate counts.
Passed, failed, timed-out, flaky, skipped, and unknown observations remain distinct.
Missing selections create gaps. Missing final results remain unknown.
Added separate failure and volatility rankings with explicit sample and comparison counts.
Volatility compares adjacent known results within the same suite, target, and execution location.
Per-run permission checks exclude denied observations and metadata.
Validation: focused aggregation and API race tests pass on SQLite. API parity checks pass on local PostgreSQL.
Checks cover counts, retries, ambiguity, gaps, comparison boundaries, filters, pagination, object denies, and permission revocation.
Interface completion, cross-project rollups, and HTTPS timing charts remain pending at this checkpoint.

### Stacked area trends and ranking interface

Added project navigation and a focused Trends page with persisted UTC period and resource filters.
Connected stacked areas show final states with a separate timeout layer and consistent light and dark colors.
Selection gaps break each path. Missing observations remain unknown instead of passing.
Hover and keyboard focus expose exact state counts. Accessible tables link to original run records.
Failure and volatility rankings show sample and comparison counts with category and permanent-test history links.
Bounded history pages expose partial coverage. Errors remain distinct from valid empty scopes.
Added Trends page help and extended the shared help workflow to cover the new route.
Validation: the Angular production build and three chart and scope utility checks pass.
The chart geometry check verifies area totals and missing-history breaks.
STUDIO-TRENDS-001 creates real passes, failures, timeouts, a successful retry, and a run deadline through TestStudio.
It verifies logical counts, selection gaps, volatility, ranking links, preserved filters, original runs, themes, and mobile layout.
The workflow also verifies unavailable history errors and valid empty scopes.
The focused help workflow passes with the new guide. Browser checks report no runtime errors.
Local visual checks verify light and dark rendering and keyboard values without document overflow.
Owned browsers and verification controllers stop after checks. Worker containers and output volumes are removed.
Cross-project rollups, HTTPS timing history, and the remaining full plan stay pending.

### Durable metadata storage

Implemented versioned SQL records, atomic revision checks, scoped cursors, durable sequences, and SQLite WAL persistence.
The PostgreSQL dialect uses the pgx driver. Hosted validation remains pending.
Validation: `go test ./internal/storage -count=1` passes.
Tests cover competing writers, transaction rollback, cursor isolation, and reopening a database.

### Persistent identities and scoped permissions

Pinned `go-permissions` v0.0.31 and added its complete SQLite store interface.
Added persistent users, local and expiring external memberships, group archival, and scoped built-in roles.
All authorization checks use the library evaluator.
Validation: `go test ./adapters/permissions/sqlite ./internal/identity ./internal/authorization -count=1` passes.
Tests cover two project bindings, deny precedence, expired grants, revocation, inheritance cycles, and reopened database access.
Administrative API workflows, OIDC sessions, and PostgreSQL permission parity remain pending.

### Catalog drafts and immutable releases

Added project, application, environment, and suite services with atomic audit records.
Added permanent test identities, retained revisions, lifecycle actions, catalog drafts, validation, immutable publication, activation, and idempotency.
Protected selection validates dependencies. Dispatch checks current test state against retained releases.
Validation: `go test ./internal/catalog ./internal/projects -count=1` passes.
Tests cover immutable catalogs, stale edits, invalid publication, authorization, unsafe dependencies, and native HTTP definitions.
Catalog import, bulk changes, source preparation, and UI administration remain pending.

### OIDC and server-managed sessions

Added provider discovery, PKCE S256, state, nonce, token verification, replay protection, and provider group synchronization.
Added issuer-subject identity links and opaque browser sessions with expiry, revocation, and CSRF checks.
Provider tokens stay outside browser storage. Incomplete group claims remove previous external membership.
Validation: `go test ./adapters/auth/oidc ./internal/sessions -count=1` passes.
Tests use two newly written signed-token providers and cover identity isolation and invalid authentication responses.
Provider administration, bootstrap recovery, account linking, and complete multi-instance authorization checks remain pending.

### Local service and authenticated catalog API

Added the Go executable, explicit local authentication, OIDC bootstrap, PostgreSQL permission schema, and static UI hosting.
Added versioned project, application, environment, suite, catalog, draft, revision, publication, and activation routes.
Added cookie authentication, mutation CSRF checks, revision headers, and API error responses.
Validation: `go test ./internal/httpapi -count=1` and `go build -o bin/teststudio ./cmd/teststudio` pass.
API tests cover sign-in, project creation, catalog publication, stale edits, archive history, CSRF rejection, invalid JSON, and UI asset routes.
Source publishing, execution, evidence APIs, access administration, and deployment verification remain pending.

### Active principal reporting

Aligned SQLite principal lookup with the selected library's PostgreSQL grant lookup.
Expired grants and explicit denies no longer appear as active direct access.
Validation: `go test ./adapters/permissions/sqlite -count=1` passes.

### Angular catalog UI and theme modes

Added an independent Angular 22 application with project setup, application and environment forms, suites, and catalog workflows.
Added test editing, search, lifecycle filters, archive and restore, draft validation, and explicit release publication.
Added light, dark, and system themes with preference persistence and semantic HTML.
Validation: `npm --prefix web run build` passes.
Browser verification covers new generic records, publication, archive and restoration, theme persistence, system changes, keyboard focus, and narrow-screen overflow.
Captured light, dark, and mobile screenshots under ignored `output/playwright/`.
Verified that the named browser, Chrome children, and automation daemon exit after cleanup.
Run pages, evidence views, catalog imports, complete access administration, and advanced catalog workflows remain pending.

### Deployment verification targets

The user selects the MMDev account for AWS verification.
Account and profile values remain external installation configuration.
Azure verification credentials and a remote Kubernetes target remain unspecified.
Local implementation and provider contract checks continue independently.

### Bounded artifact filesystem

Added a filesystem adapter with atomic writes, checksums, byte limits, and root-confined file access.
Validation: `go test ./adapters/artifacts/filesystem -count=1` passes.
Tests cover successful retrieval, oversized uploads, invalid paths, and escaping symlinks.

### Native HTTP execution

Added bounded GET and HEAD probes, response assertions, TLS checks, phase timing, deadlines, and cancellation.
Redirects stay disabled. Target resolution validates network scope and rejects metadata endpoints.
Private networks and plain HTTP require explicit environment conditions.
Validation: `go test -race ./internal/execution -count=1` passes.
Tests cover JSON and header assertions, redirect isolation, TLS trust, certificate lifetime, cancellation, timeouts, and metadata restrictions.
Run coordination, response credentials, evidence indexing, and deployment workers remain pending.

### Verification scope

The user requires almost all initial functional testing to run locally.
Use SQLite, local PostgreSQL, real local browser workflows, local workers, and local artifact storage.
Provider contract checks and local deployment packaging precede any cloud smoke check.
MMDev remains the later AWS smoke target.

### Durable evidence metadata and log ingestion

Added provider-neutral artifact ports, persistent manifests, run quotas, checksums, ordered log metadata, and execution events.
Duplicate segments are idempotent. Changed payloads conflict. Failed uploads remain explicit and can retry without rerunning a test.
Missing content updates its manifest state. Blob files remain outside SQL rows.
Validation: `go test -race ./internal/evidence -count=1` passes.
Tests cover concurrent duplicate admission, byte quotas, retry recovery, project isolation, and missing content.
Retention cleanup, complete streaming ingestion, and cloud blob adapters remain pending.

### Durable native run coordination

Added immutable execution snapshots, idempotent run requests, transactional queue claims, worker leases, progress counts, and retries.
Added queued and running cancellation, lost-worker recovery, dispatch authorization checks, and failure reruns.
Each attempt remains separate from the final test count. Capture completeness remains separate from test outcome.
Validation: `go test -race ./internal/runs -count=1` passes.
Tests cover environment changes, competing workers, flaky reduction, cancellation, worker loss, permission revocation, and capture failures.
Coded suite workers, complete execution profiles, scheduling, and run UI integration remain pending.

### Run API and private evidence delivery

Added authenticated run creation, bounded history filters, attempt pages, event pages, cancellation, and failure reruns.
Added bounded log pages and a tail view. Continuation cursors bind the selection and collection version.
Added short-lived evidence grants on a separate HTTP origin. Grants recheck sessions and current permissions.
Logout, permission revocation, expired links, and missing content prevent access.
HTML and other active content download as attachments. The artifact response uses a sandbox content policy.
The executable starts the native worker pool and a separate artifact listener.
Validation: targeted race tests pass for the HTTP API, evidence, runs, and sessions.
Storage pagination tests and the Go executable build pass.
API tests execute a real local HTTP target and retrieve retained evidence through the artifact origin.
Cloud artifact adapters, report viewers, complete log ingestion, and retention cleanup remain pending.

### Run history and evidence UI

Added run history filters, bounded pagination, and a run form for published suites and registered targets.
Added live run details, execution snapshots, progress, final counts, raw attempts, events, and filtered log tails.
Added cancellation, failure reruns, artifact previews, and explicit download links.
Artifact previews support text, raster images, and video through separate-origin grants.
Validation: the Angular production build passes.
Local browser checks create a fresh project, application, environment, suite, test, and published release.
Checks verify passing execution, failed execution, failure rerun recovery, cancellation, log preview, and file download.
Light, dark, and narrow-screen views use the shared theme system. The final browser page has no console errors.
The named browser, Chrome children, automation daemon, local target, and local TestStudio server stop after verification.
Step timelines, coded runner evidence, trace viewing, and complete result comparison remain pending.

### Local PostgreSQL parity and concurrent startup

Added a disposable local PostgreSQL test profile with loopback binding, isolated databases, and automatic container cleanup.
Catalog, run, evidence, HTTP API, session, and OIDC fixtures now use the same checks with either database.
Added serialized metadata migrations and a product-owned PostgreSQL permission adapter around the upstream store.
Permission migrations use a dedicated connection and work with a one-connection service pool.
Shared checks verify cross-instance membership, grants, explicit denies, expiration, role overrides, inheritance, revocation, and disabled users.
The upstream PostgreSQL store permits inheritance cycles. The product adapter rejects them in a serialized transaction.
SQLite inheritance writes use a graph revision to reject competing edges that form a cycle.
Validation: the local PostgreSQL functional profile passes with race checks.
Targeted SQLite and PostgreSQL checks pass for concurrent migration and opposing inheritance writes.
The Go executable builds. Test containers and browser processes are absent after cleanup.
SQLite-to-PostgreSQL transfer, complete administrative workflows, and cloud deployment verification remain pending.

### Catalog transfer and atomic bulk API

Added versioned JSON exports, native HTTP import previews, revision guards, bounded atomic apply, and duplicate identities.
Added lifecycle bulk previews with explicit per-test revisions and current permission checks.
Plans retain before values and field errors. Catalog changes invalidate unapplied previews.
Definitions, revisions, audit records, plan completion, and guards commit in one transaction.
Concurrent apply retries return the same completed plan without duplicate revisions.
Validation: targeted race tests pass for catalog, HTTP API, and affected run behavior on SQLite and local PostgreSQL.
Checks cover preview isolation, stale plans, complete rollback, authorization, invalid dependencies, duplicate identifiers, and retained run snapshots.
The Go executable builds. Catalog transfer UI, YAML, source uploads, and complete bulk metadata edits remain pending.

### Catalog transfer, selection, duplication, and history UI

Added JSON file upload, export download, validation previews, before/after inspection, and explicit atomic apply.
Added suite filtering, checkbox selection, and bulk lifecycle previews.
Added independent duplication and revision history panels with complete retained definitions.
Preview and history panels receive keyboard focus. Invalid plans disable apply.
Validation: the Angular production build passes.
Local browser checks verify upload, download, import application, disable, archive, restore, duplicate creation, and retained history.
An invalid external path produces a field error and cannot apply.
Light, dark, and mobile views pass visual checks. The final browser page has no console errors or document overflow.
The test browser, automation daemon, and local server stop after verification.
Category trees, saved selections, source-owned metadata, full bulk metadata editing, and YAML remain pending.

### Capture policies, shared quotas, and durable retention calculation

Added project defaults, suite and test overrides, and permission-controlled run capture overrides.
Runs freeze effective capture settings and current project limits.
Added file, test, run, and shared project quotas with atomic reservations and idempotent retry accounting.
Added non-mutating revision assertions for quota configuration guards.
Added per-kind durations and longer retention for failed or flaky observations.
Durable retention tasks page through manifests and resume after service restart.
Completion and lost-worker recovery queue the same retention calculation.
Dispatch checks current capture limits, archive states, environment protection, and override permission.
Validation: targeted race tests pass on SQLite and local PostgreSQL for storage, evidence, runs, catalog, and HTTP API.
Repeated SQLite startup checks pass after bounded lock retries. The Go executable builds.
Tests cover shared quota contention, frozen policies, log opt-out, current limits, override revocation, and retention task resumption.
Capture configuration screens, storage cleanup, evidence pins, browser media capture, and cloud artifact adapters remain pending.

### Evidence pins, retryable cleanup, and upload leases

Added durable run pins, timed pins, audited deletion requests, retention status, and quota reads.
Short-lived artifact grants honor pins, current permissions, session state, and expiry.
Cleanup skips active runs, pending retention tasks, pinned runs, and active uploads.
Deletion claims survive restart. Failed storage deletion remains explicit and retries without changing observed results.
Successful deletion preserves manifests and releases each quota reservation once.
Uploads use a bounded lease. Duplicate requests do not create competing writers.
Late upload completion cannot revive expired content. Legacy indexing preserves existing run accounting and adds project reservations once.
Validation: targeted race tests pass for evidence and HTTP API on SQLite and local PostgreSQL.
Tests cover pin access, explicit permission denies, deletion races, retry after restart, legacy indexing, quota release, and late uploads.
The Go executable builds. Retention UI, capture configuration screens, media capture, and cloud artifact adapters remain pending.

### Retention controls and storage visibility UI

Added retention status, pin state, storage usage, captured policy details, and periodic reconciliation views.
Added reasoned pin creation, timed updates, removal, and explicit artifact deletion forms.
Pin editing retains its revision while dirty. Conflict messages support reload and review.
Expired and deleted content has no active access controls. Active pins disable deletion controls.
The Capture summary remains separate from file retention. Test outcomes remain unchanged after deletion.
Validation: the Angular production build and the targeted retention API check pass.
Local browser checks verify pins, timed expiry, pin removal, deletion, manifest retention, and exact quota reduction.
Keyboard focus returns to the artifact heading. The final page has no console errors or document overflow.
Light, dark, and mobile views pass visual inspection. Test browser, daemon, server, and database containers stop after checks.
Editable capture settings, media capture, cloud adapters, and remaining plan areas remain pending.

### Timing baselines, adaptive timeouts, and slowdown failures

Added project and suite timing defaults, test overrides, and immutable release and run timing snapshots.
Native HTTP attempts use monotonic elapsed time. Logical results retain cumulative retry duration and separate functional state.
Comparable baselines isolate definition, target, environment, location, and runtime context.
Clean first-attempt results train bounded history. Slow, failed, retried, and interrupted attempts do not train history.
Five observations enable default slowdown failures and bounded adaptive timeouts.
Added side-effect-free run estimates, authorized baseline reads, and audited revision-controlled baseline resets.
Generation checks prevent old queued work from repopulating reset baselines.
Final results and observations commit atomically. Worker heartbeat reads avoid unnecessary lease revision contention.
Validation: targeted race tests pass on SQLite and local PostgreSQL for timing, runs, catalog, and HTTP API.
Checks cover actual local HTTP slowdowns and adaptive cancellation, policy modes, retry totals, context changes, concurrency, permissions, and resets.
Timing UI integration and browser verification remain pending at this checkpoint.

### Timing estimates, result comparison, and baseline controls UI

Added pre-run estimates with readiness, expected duration, thresholds, attempt timeouts, and complete retry timeout budgets.
Changed requests clear stale estimates. Unknown history stays explicit and short run deadlines show a warning.
Run details show worker elapsed time, estimated remaining execution, actual durations, slowdown ratios, and cumulative retry time.
Functional results remain visible beside duration regression failures.
Added complete test timing overrides, timeout ceiling editing, field validation, and normal release publication.
Added current baseline inspection and revision-controlled resets with reasons.
Reset confirmation preserves the selected run's original predictions and results.
Validation: the Angular production build and four focused timing utility tests pass.
New default inheritance checks pass with race detection on SQLite and local PostgreSQL.
Local browser checks verify five clean observations, estimates, a real slowdown failure, baseline reset, policy editing, publication, and budget warnings.
The checked views have no console errors or document overflow. Light, dark, and mobile views pass visual inspection.
The owned browser, automation daemon, local service, target server, and database containers stop after checks.
Coded runner timing, deployment profiles, editable capture settings, and the remaining full plan stay pending.

### Stacked area trend design

Updated the plan and interface contract to use stacked area charts for result states.
The chart stacks passed, failed, flaky, skipped, and unknown or incomplete counts.
Final logical results provide counts. Retry attempts do not inflate totals.
State colors, stack order, accessible values, and missing-history gaps form part of acceptance.
Validation: documentation consistency checks pass. No result-state chart implementation exists yet.
Trend implementation and browser verification remain pending.

### Local Playwright worker and TestStudio self-tests

Added an explicitly enabled local container worker with permitted immutable images and targetless suite admission.
Added bounded reporter contracts, real test-start progress, framework timing, steps, browser diagnostics, and private media capture.
Unknown workers, missing reports, wrong identities, and unexplained process failures cannot produce passing results.
Preparation stays separate from execution. Playwright uses a separate timing profile and normal retry reduction.
Containers use no host mounts or external network. Cleanup removes the owned browser container and anonymous output volume.
Added new TestStudio Playwright workflows for themes, mobile layout, catalog publication, HTTP runs, timing history, and reset.
Added separate intentional failure and timeout probes, plus cancellation checks.
Added explicit local registration, immutable publication, and a standalone verification command.
Registration preserves permanent identities, lifecycle state, service-managed metadata, and comparable history.
Added collapsible recorded steps to the standard attempt view.
Validation: targeted race checks pass for the adapter, runs, catalog, and HTTP API.
The coded worker storage checks pass on SQLite and local PostgreSQL.
The normal self-test suite passes three workflows through TestStudio with complete evidence.
A repeated registration reuses all three baselines. Intentional failure and timeout runs retain the expected failure kinds and media.
Real container checks verify test-start events, timeout, failure, cancellation, and cleanup.
The Angular build passes. Browser inspection verifies recorded steps without console errors or mobile document overflow.
The owned browser, controller services, worker containers, and database container stop after verification.
External target profiles, cloud execution, source screens, orphan reconciliation, and private trace viewing remain pending.

### Durable local worker recovery and scoped cleanup

Added persistent installation identity, execution references, pre-launch records, and installation-scoped container and volume labels.
Normal workers checkpoint observations and evidence before removing resources.
Expired worker leases retain incomplete outcomes while recovery imports available diagnostics.
Recovery does not launch replacement tests or rewrite existing attempts and final results.
Durable claims prevent competing controllers from processing one reference.
Inventory checks reopen cleanup for late resources. Healthy workers and other installations remain untouched.
Cleanup retries and bounded evidence retries survive restart. Artifact identity and log sequence remain stable during replay.
Added paginated authorized execution history. Responses omit internal claim tokens.
Validation: focused race checks pass for worker identity, checkpoints, lost leases, competing claims, cleanup retries, and evidence replay.
The hard-stop check kills its own active controller and restarts the same installation with new dispatch disabled.
The check verifies incomplete outcome, recovered diagnostics, and removal of the owned browser container and output volume.
The focused checks pass on SQLite and local PostgreSQL. Recovery status UI verification remains pending at this checkpoint.
Cloud worker profiles and the remaining full plan stay pending.

### Worker recovery status and unavailable timing UI

Added paginated worker records with checkpoint, recovery, cleanup, and capture-gap details.
Pending resources refresh every five seconds. Access failures stop automatic requests.
Recovered attempt labels preserve the original run outcome. Missing durations remain unavailable instead of zero.
Worker coordination time stays separate from test execution duration.
Validation: focused backend race tests and five timing utility checks pass. The Angular production build passes.
Local browser checks verify recovered diagnostics, light and dark themes, and the corrected mobile table.
The checked views have no console errors or document overflow. Owned browser and controller processes exit after checks.
Cloud profiles, trend implementation, and the remaining full plan stay pending.

### Contextual help drawer

Added Help to the shared header, including the sign-in screen.
Route metadata selects guides for projects, the catalog, project settings, run history, and run details.
Each guide provides Introduction, Concepts, and numbered Tutorials for current controls.
The native dialog slides from the right, scrolls independently, and preserves unsaved page edits.
Close, Escape, and the desktop backdrop dismiss help and restore focus.
Explicit focus boundaries keep keyboard navigation inside the drawer across browser runtimes.
Shared theme tokens, full-width mobile display, and reduced-motion support apply throughout.
Added the permanent STUDIO-HELP-001 self-test and a targeted normal-workflow selector.
Validation: the production Angular build and script syntax check pass.
The targeted Playwright workflow passes through normal TestStudio publication, execution, timing, and evidence storage.
Checks cover all six guides, normal navigation, preserved edits, focus, dismissal, themes, mobile overflow, and reduced motion.
The run retains complete evidence and a cleaned worker. Browser checks report no console errors.
Owned verification browsers, controllers, and worker containers stop after checks.
The remaining full plan stays pending.

### Separate timeout results and inclusive failure reruns

Added timed-out status for native HTTP, Playwright reports, worker deadlines, attempts, and final results.
Counts expose timedOut separately from failed. Mixed runs retain both counts without retry inflation.
A passing retry remains flaky. Timeout observations stay censored and do not train timing baselines.
Run deadlines retain known active-test timeout results for later reruns.
Failure reruns include failed, timed-out, and flaky tests through the API and UI.
Older failed results with timeout reasons receive compatible read views without rewriting stored history.
Timeouts retain unsuccessful-attempt media capture and failure retention extensions.
Added separate timeout badges and counts, page help guidance, and the timeout layer in the stacked-area trend contract.
Validation: focused race tests pass for runs, HTTP execution, API history, report contracts, timing, and retention rules.
Timeout aggregation, deadlines, legacy reads, and reruns pass on SQLite and local PostgreSQL.
The Angular build and rerun utility check pass.
STUDIO-TIMEOUT-001 triggers a real local timeout, reruns it through the UI, and verifies a clean pass with original history retained.
The intentional Playwright timeout probe retains zero failures, one timeout, complete evidence, and diagnostic media.
Real container checks verify distinct assertion and timeout results with retained steps, screenshots, and traces.
Mobile inspection verifies the separate timeout display without document overflow. Browser diagnostics contain no runtime errors.
Owned verification controllers and browser containers stop after checks.
The remaining full plan stays pending.
