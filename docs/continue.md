# Continue TestStudio implementation

Last reviewed: 2026-11-10. These notes describe the last verified state, not a live infrastructure inventory.
Read this guide before historical continuation commands elsewhere in the plan.

## Local delivery: application ownership and environment configuration

The user adopts one application owner per suite, an application catalog, an empty Default suite for each new application, and environment configuration contracts on 2026-11-10. This supersedes earlier shared-suite guidance.
Application creation commits its empty Default suite atomically. Existing applications have an explicit Create default suite action. Existing shared or unassigned suites require a reviewed assignment in Project settings. Incompatible current tests, probes, and schedules block assignment. Existing sole owners are permanent; historical definitions and runs remain unchanged.
The application catalog uses an application route and server-scoped counts. Project catalog navigation asks for an application or opens the sole application. Administrative aggregate APIs and legacy history remain readable.
Application > Environment configuration defines typed nonsecret parameters, defaults, named base URLs, and approved secret bindings. Its readiness matrix covers explicit requirements and readable enabled tests, including saved browser endpoints. Incomplete saves, review, unsaved edits, conflicts, help, and scoped access remain explicit. Secrets stay operator-managed. Application-only reference listings restrict target origins.
Run review, enabled schedule saves, dispatch, and attempt authorization enforce requirements for selected tests and dependencies. Missing or invalid parameters and unavailable secrets name the blocked entry without exposing provider details. Effective parameter changes block queued runs. Native HTTP paths and text assertions accept escaped parameter references. Updated coded runtimes receive TESTSTUDIO_PARAMETERS; existing hosted runtimes are not upgraded here.
API contract is 0.49.0. Targeted SQLite and race-enabled PostgreSQL checks pass for ownership, atomic creation, conflicts, selection closure, frozen parameters, exact access, secret availability, native parameter execution, and retained history. Source import and generated-client checks pass. The Angular build and three scope utility checks pass.
Normal STUDIO-CONFIG-001 passes in 2.348 seconds and STUDIO-SUITES-001 passes in 1.809 seconds in the retained run 20261110T151310.627004000-5eb43b34523cb66b54c17937. Normal STUDIO-APPS-001 passes in 3.077 seconds in run 20261110T151736.456039000-a7ba656238ae2abb00ab399f. The earlier application fixture failures use obsolete labels and capability queries; the final application run supersedes them. Reuse output/selftest under those run identifiers. The dark narrow configuration screenshot is visually reviewed. Owned controllers, browsers, execution containers, and the temporary PostgreSQL container close.
This refines 05, 12, 13, 21, 25, 27–29, 34, 38, 49, and 50. Acceptance stays 50 verified of 57. No release, push, deployment, hosted data migration, or demo update occurs. MMDev remains 0.14.0-dev, build 14cc815281d1. Preserve the user's port 8040 demo, its database and private UI snapshot, and existing workloads.
See [Environment configuration](environment-configuration.md). Publishing, reviewed target migration, and target verification remain separate work.

## Completed delivery: direct saving throughout the product

The user explicitly removes the user-facing draft and publication concepts for tests, suites, source authoring, and status pages. Status pages use Save and Enabled. This overrides earlier UI rules requiring separate draft/publication actions. Preserve internal immutable execution versions, history, authority checks, revision conflicts, unsaved edits, source preparation, and explicit test execution. Save never starts a test run.
The user also reports a saved News recording missing from the Simple Browser suite catalog. Source saving must register the test immediately, retain its application context, and automate preparation and activation. Recover existing saved source through normal authorized APIs; do not read another actor's private live recording. Preserve the three existing catalog tests and all user data. Missing hosted preparation/execution workers must remain an honest readiness limitation, not extra tester publication steps.
Product 0.14.0-dev from committed 042780f is deployed as build 14cc815281d1. UTC build creation is 2026-11-09T21:13:45.177Z, task startup is 2026-11-09T21:16:30.391000Z, and UI publication is 2026-11-09T21:19:54Z. The reviewed change set modifies only Service and Task with LocalAccounts=false. Both existing ECS services retain completed healthy rollouts. Hosted API metadata and actual index and main bundle match the selected release. SSO-only automatic Organization sign-in and the private recording worker remain unchanged.
The user's two existing saved recordings are recovered through normal exact-revision suite source APIs. Both appear under Simple Browser and retain their application context. Existing project catalog definitions and revisions remain unchanged. No target run starts. The hosted catalog and help expose no draft or publication controls. The hosted Enabled switch withdraws and restores access, enabled editing updates public content on Save, and narrow dark layout passes.
The temporary public status page is disabled and its verification project archived. Owned application sessions close and all seven owned browser processes exit. Local owned recording resources and test controllers close. User browsers, the port 8040 demo, and unrelated containers remain intact. Private credential state and temporary verification helpers are removed. Reuse output/aws-mmdev/direct-save-0140 and output/playwright/direct-save-0140. Hosted coded preparation and execution workers remain absent; the two saved browser tests honestly show Getting ready. Do not describe them as runnable on MMDev.
Normal STUDIO-PUBLISH-001 now verifies direct saving, stale edit rejection, immutable prior runs, and schedule pins. It passes with STUDIO-SOURCES-001 in run 20261109T212119.481148000-037295a1991f02c6f36ac6dd. STUDIO-RELEASES-001 passes retained version comparison, rollback, conflicts, and unchanged historical pins in run 20261109T212200.844868000-e1fbc588e5d72f5beb140189. No full catalog executes. Acceptance remains 50 verified of 57; items 21, 25–28, 49–51, 56, and 57 retain their stated scope. No Git push occurs.

## Completed delivery: selecting elements and recording checks

Product 0.13.0-dev adds visible outlines, explicit item-or-group counts, and repeated-card detection for ordinary containers. A story needs no special HTML label. Group counts include displayed items outside the current viewport and exclude hidden items. Check selection never activates a link or changes a control; Interact with website restores ordinary recording. Disabled controls can be selected through the same bounded point selection.
Checks now include visible, minimum, maximum and exact counts, contained and exact text, field value, enabled and disabled controls, checked and unchecked inputs, page address, and page title. The menu limits control checks to compatible elements. Current text, value, count, and state mismatches remain in review. Password values are not offered for inspection. Existing source review, one-draft saving, explicit preparation and execution, uncertainty recovery, and tab ownership remain.
Normal STUDIO-HOSTED-RECORDER-001 passes in 13.969 seconds, run 20261109T185138.923030000-84a33498a8efbfe67326a97c. Its saved draft prepares, publishes, and passes an explicitly selected replay against a different local environment in 222 ms. The complete harness takes 45.00 seconds. The fixture uses unlabeled div cards, a hidden draft card, form values, disabled controls, and checkbox changes. Reuse output/playwright/recording-selection-checks-final. All owned recording resources and three observed host processes close. Focused native capture, bounded action validation, generated contract synchronization, and UI compilation pass. API contract is 0.47.0.
The first UI check exposes disabled-control selection relying on native click dispatch. Point selection now resolves the element without clicking it. The normal workflow passes after that correction. Existing recording origin policy, sandbox, identities, and source-retaining cleanup remain. Product 0.13.0-dev from committed db131a5 is deployed as build 5d81af4f2f8d. UTC build creation is 2026-11-09T18:52:46.059Z, task startup is 2026-11-09T18:55:59.529Z, and UI publication is 2026-11-09T18:59:41Z. The reviewed core change set modifies only Service and Task with LocalAccounts=false. The same private worker installs the new recording overlay and committed source while preserving both reviewed starting-origin policies, gateway, sandbox, transport identities, and capacity.
The actual hosted production News page selects 100 similar story cards without requiring a headline as the group selector. Three currently intersecting cards are outlined. Switching to one item outlines only that item. At least 1, page title, and page address checks record and save as one draft. No target test executes on MMDev. The temporary project is archived, application session closes, all seven owned automation processes exit, and the worker has zero recording containers or volumes. Both existing ECS services retain completed healthy rollouts. SSO-only automatic Organization sign-in remains configured.
Reuse output/aws-mmdev/recording-checks-0130 and output/playwright/recording-checks-0130. Refresh TestStudio and start a fresh recording to use the new controls. Private helpers and credential state are removed. The port 8040 demo and user browsers remain separate. No Git push occurs. This refines 21, 49, 50, and 51; acceptance remains 50 verified of 57. Broad hosted target execution remains open.

## Completed correction: production recording startup

The latest loading-screen report uses the production website. The earlier private policy covers only its development starting origin. Production uses a different startup API. A browser probe reproduces the loading screen with single-origin access and renders News with its reviewed dependencies.
The private map now retains development and adds a separate production starting origin, each with four observed dependency origins. Analytics remains blocked. The installed canonical policy hash is 8d6fb0684b258315c22a513264245fbdebece6b244d23b8ace30079a3bcec4b5. Preserve the complete ignored map at .data/recording-site-loading/origins.json for future upgrades.
A controlled worker upgrade installs committed 3d943e7 on the same host with unchanged immutable browser and gateway images, sandbox policy, transport identities, and capacity. Existing source is retained; interactive recording requires a fresh session. The API and UI remain 0.12.1-dev, build 9875b5cc8762. SSO settings stay unchanged.
An actual owned hosted recording renders the production homepage and News stories, records the News click, and adds an at-least-one assertion. The check cancels its recording, archives its temporary project, and closes its application session. Host inspection finds no recording containers or volumes. All seven owned automation processes exit. Both existing ECS services retain completed healthy rollouts.
Reuse output/aws-mmdev/recording-prod-startup and output/playwright/recording-prod-startup. This is an operator configuration correction; no application release or target test execution occurs. Items 12, 13, 21, 45, and 51 retain their existing scope. Acceptance remains 50 verified of 57; broad hosted execution remains open. No Git push occurs.

## Completed delivery: SSO-only automatic sign-in

The user explicitly selects SSO-only automatic Organization sign-in. Product 0.12.1-dev from committed a8f9432 is deployed as build 9875b5cc8762. UTC build creation is 2026-11-09T16:24:26.667Z, task startup is 2026-11-09T16:27:41.783Z, and UI publication is 2026-11-09T16:30:34Z. Both reviewed change sets modify only Service and Task; LocalAccounts=false is explicit. Both existing services retain completed healthy rollouts. The recording worker stays unchanged.
Public sign-in options contain only organization, disable both local modes, and select organization for automatic redirect. The duplicate cognito provider is hidden from the login page and retains explicit operator recovery. Existing accounts, issuer/client configuration, mappings, and permissions are preserved. Provider saves retain existing authentication invalidation and revision checks.
Seventeen hosted checks pass. A fresh browser reaches the configured Organization authorization endpoint with the correct client, callback, code flow, nonce, and S256 challenge without rendering the TestStudio login form. Cancelling the actual owned transaction returns to the original path, query, and fragment. Cancellation reload, dismissed feedback, signed-out reload, dark narrow layout, Help, and closed login binding pass. No user authentication or project grant is fabricated; no test run starts on MMDev.
Reuse output/aws-mmdev/signin-0121 and screenshots under output/playwright/signin-0121. Explicit hidden-provider operator authentication passes. Owned operator sessions close, the cancelled login transaction closes, all seven automation processes exit, and private helpers are removed. User browsers, the demo, and existing worker workloads remain intact.
The initial 0.12.0 hosted check exposes an initial-router race under slow route loading. Automatic SSO stays temporarily off during its correction. The patch seeds navigation from the browser address and preserves absolute recovery destinations. Normal STUDIO-SIGNIN-001 passes with delayed route chunks in 8.250 seconds, run 20261109T162255.201509000-5aefdb1a1275206b6f316ddc. Its evidence and cleanup are retained under output/selftest. Earlier normal STUDIO-AUTH-001 and STUDIO-LOCAL-001 pass in 4.208 and 8.092 seconds. Focused Go race, generated contract, six utility checks, and UI compilation pass.
API contract is 0.46.0. This refines 09, 16, 19, 20, 49, 50, and 55. Acceptance remains 50 verified of 57. Cross-profile identity, hosted coded preparation and execution, and broad deployment acceptance remain open. No Git push occurs.
For future operator checks, create a fresh HTTP sign-in client for each provider authorization. An existing Cognito SSO cookie can bypass its password page. Refresh CSRF from the current application session before closing it. Use scripts/publish-aws-core.py --local-accounts=false for an explicit reviewed hosted selection; omission preserves the installed setting.

## Completed delivery: recording checks and workspace

Product 0.11.1-dev from committed e904b05 is deployed as build 8106a41d6681. Build creation is 2026-11-08T20:41:03.467Z, task startup is 2026-11-08T20:43:45.124Z, and UI publication is 2026-11-08T20:48:06Z. Both reviewed core change sets modify only Service and Task. Both existing services retain completed healthy rollouts.
The recording worker retains verified source 5ba5d66 on the same private host, transport identities, runtime, and complete approved origin map. Reuse its separate package checkpoint at output/aws-mmdev/recording-workspace-0110/worker-upgrade.json. The API/UI patch needs no further worker upgrade.
The actual selected News page records at least one item, exactly 100 current stories, headline text, and page address. A separate recording tab and resizable dialog work; changing to 390 × 844 changes the real browser viewport. Saving from both tabs preserves one revision-1 draft. The installed UI explains the missing preparation worker. No target test runs on MMDev.
Fifteen hosted checks pass. Reuse output/aws-mmdev/recording-workspace-0111 and screenshots under output/playwright/recording-workspace-final. The verification project is archived, its application session closes, both owned recording resource sets are removed, all seven owned automation processes exit, and temporary credential state and helpers are removed. The initial 0.11.0 view race is corrected and a fresh hosted session connects.
Locally, normal STUDIO-HOSTED-RECORDER-001 covers counts, text, address, invalid inputs, dialog Escape and focus, two-way tab control, presets and custom dimensions, reload, source review, help, themes, narrow layouts, uncertainty recovery, save-confirmation recovery, and cancellation. The final expanded authoring run takes 10.489 seconds. With TESTSTUDIO_REPLAY_RECORDED_CHECKS=1, the exact saved draft also prepares, publishes, and passes a selected run against a different local environment in 198 ms with complete evidence. The full harness takes 46.90 seconds and closes all owned resources. Reuse output/playwright/recorded-checks-replay-verified.
API contract 0.45.0 describes the new bounded actions. Native adapter, action validation, environment rebinding, generated contract, UI build, and focused race checks pass. Saved recordings expose preparation directly when available.
This refines 21, 22, 25, 41, 42, 49, 50, 51, 56, and 57. Acceptance remains 50 verified of 57. Hosted preparation and coded execution remain the next substantial gap. Count groups depend on the website's retained structure; review their displayed matches. Azure remains deferred. Preserve the port 8040 demo, user configuration, pending recordings, and unrelated services. No Git push occurs.

## Resume in this order

1. Read AGENTS.md and check the branch and working tree. The logo delivery branch is codex/status-page-logo. The original checkout retains codex/implementation and concurrent reliability edits.
2. Read the active delivery sections at the start of [the plan](plan.md).
3. Read the Acceptance criteria section in [the ledger](implementation.md). It contains the authoritative 57 checkboxes.
4. Continue the latest user request first. Otherwise select one next milestone below and read its implementation before changing it.
5. Implement the milestone, run targeted local checks, and update the ledger and these notes with verified results.

Do not restart completed setup or broad acceptance matrices merely because this is a new session.
Missing ignored receipts require focused inspection or recovery. They do not prove the feature needs reimplementation.

## Current correction: recording website startup dependencies

The selected website stalls on its loading screen because its separate startup API is blocked by the recorder's single-origin policy. Direct browsing loads News; reproducing the recorder policy blocks initialization; an exact approved dependency list restores News.
The worker now supports a generic operator origin-policy file, frozen network targets and resource fingerprints, and compatible cleanup of older leases. Credentials, metadata protection, private-address restrictions, and exact origin and port enforcement remain intact.
The normal STUDIO-HOSTED-RECORDER-001 fixture now requires a module from a second origin to construct its News interface. The focused workflow passes in a 26.76-second harness and closes all owned resources. Reuse output/playwright/recording-cross-origin.
Go race checks, actual PostgreSQL policy persistence, container isolation and deadlines, command startup, and source-retaining shutdown pass locally.
Product 0.10.2-dev from committed 000ed2b is deployed as build c9d8e9d2d4b2. The worker upgrade uses a separate package checkpoint and preserves the same private host and transport identities. The reviewed core change set modifies only Service and Task. Both services retain healthy completed rollouts.
The actual selected website now loads inside the hosted recorder. News navigation reveals real stories; a story visibility check records, reviews, and saves exactly one revision-1 draft. No target test executes. Ten hosted checks verify the matching artifact, upgraded worker policy, private access, preserved services, and complete recording cleanup.
Reuse output/aws-mmdev/recording-origins-0102 and screenshots under output/playwright/recording-site-loading. The verification project is archived, its session and automation browser close, and temporary credential state is removed. The private operator map remains .data/recording-site-loading/origins.json; it contains one starting website and four reviewed public dependency origins. Its installed hash is d28ed1bf5cdc71fe9d5bef9088f147065fd0fe0595da0a435d446347e1b2f345.
Preserve the original infrastructure receipt and worker-upgrade.json. Future worker updates use scripts/deploy-aws-recording.py upgrade with an explicit complete origin map and a new package checkpoint. Graceful shutdown retains captured source; start a fresh recording afterward. This refines 12, 13, 21, 45, 51, 56, and 57; acceptance remains 50 verified of 57.

## Current correction: shared form field alignment

Descriptions caused shared grid fields to stretch their labels and controls to match a taller neighbor. The shared .field layout now starts its content at the top and retains normal control heights.
Normal STUDIO-HOSTED-RECORDER-001 passes at 1280, 900, and 390 pixels with wrapped help, light and dark themes, and the existing recording, review, recovery, saving, cancellation, help, and keyboard workflow.
Run 20261108T174217.731170000-0c6c67ceb614d9e3f4b3361f takes 9.863 seconds; the focused harness takes 38.64 seconds. All owned browser containers, gateways, volumes, and three observed host processes close.
Reuse output/playwright/form-field-alignment. The initial failed check used an incorrect description locator and is superseded by this passing run.
Patch 0.10.1-dev from committed fe99e8a is deployed as build 6c6cec452746. The reviewed change set modifies only Service and Task. Nine hosted checks verify matching API, UI, stylesheet, service health, private access, unchanged recording host, and actual field geometry at all three widths.
Reuse output/aws-mmdev/form-alignment-0101 and screenshots under output/playwright/hosted-form-alignment-0101. The verification project is archived, its owned session closes, and the owned Edge browser stops. No recording or test run starts. The private recording host is unchanged.
This refines 49 and 50. Acceptance remains 50 verified of 57.

## Azure planning: direct installation first

On 2026-11-08 the user selects an installation into their own Azure subscription before Marketplace delivery.
Follow [the Azure deployment plan](azure-deployment-plan.md). Container Apps and isolated Sandboxes remain the intended direction; AKS is excluded.
Keep the future Managed Application option viable through reusable infrastructure and installation parameters. Marketplace publication and billing remain deferred.
The exact tenant, subscription, region, identity settings, and budget remain unselected. No Azure deployment or target verification has occurred in this planning pass.
The user reports new-account free-service benefits and permits alternative databases, including Azure SQL. Evaluate the free pilot choices in the deployment plan first; PostgreSQL and SQLite are the only current adapters.
Preserve staged Azure code and concurrent recording work. Do not restart historical inspector work.
Acceptance stays 50 verified of 57; no item closes from planning.

## Current priority: recording without tester installation

The user approves an in-page recording browser for testers who need simple guidance and no developer setup.
Follow [the hosted recording plan](hosted-recording-plan.md). This direction supersedes desktop-companion packaging as the main tester onboarding path.
The local proof now captures a generic News click and visible-story check, shows plain-language review, and saves through the existing source draft service.
Normal STUDIO-EMBEDDED-RECORDER-001 passes in 4.123 seconds, run 20261108T155702.239245000-2124450c8345cdca54c6958c. Its complete focused harness takes 19.87 seconds.
Help, active and saved reload, uncertain input, lost save confirmation, Advanced source access, cancellation, focus, themes, narrow layout, and all eleven observed process shutdowns pass.
Reuse receipts and screenshots under output/playwright/embedded-news-complete. Canonical OpenAPI 0.44.0 describes this source; the current hosted patch is 0.10.2-dev.
The native preview remains loopback-only. Milestone 2 now implements a separate [private recording worker](recording-worker.md) with mutually authenticated TLS, durable capacity and cleanup leases, fenced restart recovery, current authority checks, and bounded idle behavior.
The actual recording browser runs sandboxed in a non-root container with no direct networking, a private per-session gateway, fixed resource limits, and independent browser and gateway deadlines.
Normal STUDIO-HOSTED-RECORDER-001 passes locally in 5.997 seconds, run 20261108T165640.285709000-e4e7e064e323488114590eb5; the complete harness takes 21.56 seconds.
The same News, review, one-draft save, reload, input and save recovery, help, focus, themes, narrow layout, and cancellation checks pass. Both browsers, gateways, volumes, and all three observed host processes are removed.
Reuse output/playwright/private-recording-worker. Targeted race checks pass for ownership, capacity, uncertain input, revoked authority, archive, idle, deadlines, TLS identities, API reconnect, and transfer exclusion. Actual PostgreSQL lease and recovery checks pass in 2.841 seconds.
Docker policy and independent-deadline checks pass in 6.616 seconds. Standalone daemon configuration, private readiness, and clean shutdown checks pass in 0.385 seconds.
The API selects --recording-worker-config or --recording-worker-config-env. The initial provider requires a dedicated Docker host and permits only the entered website origin; separate assets or sign-in origins remain a reviewed target extension.
The original authorized recording deployment completes in 0.10.0-dev from committed 7f1a023, build 852ce753cf8b. The current 0.10.1-dev patch retains that worker installation. The reviewed dedicated-host change set adds thirteen resources; the core change set modifies only ExecutionRole, Task, and Service. Both stacks and both existing services have completed healthy rollouts. API and UI build identities match.
The hosted browser records News and a visible story on an independently authored generic fixture, saves exactly one revision-1 draft, and cancels a second recording. No target test runs. Actual host inspection verifies the browser has no direct networking, a read-only root, dropped capabilities, resource bounds, and no Docker socket. Both browser and gateway pairs and their volumes are removed.
Twelve hosted deployment checks pass. Reuse output/aws-mmdev/recording-0100 and screenshots under output/playwright/aws-recording-0100. The fixture is removed, its verification project is archived, the owned application session closes, and the owned Edge automation browser stops.
The private t4g.medium host is i-001e74196a83b4693 in stack teststudioapi-recording, with one concurrent recording slot. Use deploy/aws/recording.yaml and scripts/deploy-aws-recording.py; preserve host.json and release.json checkpoints instead of starting duplicate deployments. API startup secrets use RecordingWorkerSecretArn. Rotate the dedicated transport CA and both leaf certificates before 2027-02-06. Keep the same host and worker identity for cleanup recovery.
Then complete hosted preparation and execution and verify the actual News journey with an ordinary tester browser.
Existing recording history, user project configuration, native schedules, and the port 8040 demo are preserved. No remote Git push occurs.
The acceptance count remains 50 verified of 57. Items 12, 13, 21, 25, 45, 49, 50, 51, and 57 are refined; broad worker and target acceptance remain open.

## Concurrent optional-logo delivery

Release 0.6.0-dev is deployed as build dca2ee3e4ddf from verified logo implementation 3d13a08. Both remote implementation and status-page-logo branches contain the implementation and final release notes. The local reliability delivery incorporates the upstream logo implementation and release notes. Its shared Angular model retains logo fields and centered rendering. That shared source is now included in deployed 0.8.0-dev. Keep its broader target acceptance separate from the height verification. Receipts and screenshots are output/aws-mmdev/status-logo-060 in this original checkout. The generic verification page is unpublished, owned sessions and tabs close, and no user page is edited.

## Concurrent height delivery

Release 0.8.0-dev is deployed as build 2320bb94582b from dc558de. Both remote delivery branches contain the height implementation and final notes. This release includes the committed shared reliability/library integration bf9ab8a. Preserve any newer local work in this checkout and incorporate upstream before another deployment. Library 0.2.0-dev and its packed artifact include logoHeight. Twenty-eight hosted checks and actual 192-pixel rendering pass. The requesting page is unchanged; the generic verification page is unpublished. Receipts and screenshots are output/aws-mmdev/status-height-080 here.

## Public Help removal release

Release 0.8.1-dev is deployed from 8c8a81a as build 3f0963838890. Both remote delivery branches include this release and its notes. The public status document and iframe omit Help, while management guidance remains. The browser workflow and seventeen hosted checks pass. Preserve separate local work and incorporate upstream before another deployment. Receipts and screenshot are output/aws-mmdev/status-no-help-081 in this checkout.

## Centered status-page release

Release 0.8.2-dev from d47c744 is deployed as build a3891c1ad13a and pushed to both remote delivery branches. Content uses a centered 72rem maximum with compact pills and full-viewport background. The local status workflow passes at 1920, 1280, and 390 pixels; eighteen hosted checks and equal-margin rendering on the requesting public page pass. Receipts are output/aws-mmdev/status-centered-082, with local browser evidence and the packed 0.2.1-dev library retained here. Preserve this checkout's separate reliability inspection work and incorporate upstream before another deployment. No user page configuration or port 8040 demo changes.

## Current delivery: individual reliability inspection

The user requests committing and deploying the inspection feature on 2026-11-08. Product 0.9.0-dev from 5468130, build 874f3a6eafce, now runs in MMDev. It combines the completed centered-page, logo-height, and public Help changes with Individual checks, paginated raw observations, and pill drill-down. API contract 0.43.0 and Angular library 0.3.0-dev describe the combined source.
The two focused browser workflows, package and native API checks pass. Fifteen hosted checks and actual public table rendering, 50-to-100 pagination, and exact-day drill-down pass. The requesting private browser sees the matching new version. The generic verification page is unpublished, its session closes, temporary browser tabs close, and owned local resources are removed. No new target test execution or user configuration change occurs. Receipts are output/aws-mmdev/reliability-090.
Open Project → Test catalog → History → Reliability history, then Individual checks. On published status pages use Individual checks or select a pill and Inspect period. Refresh an older tab to load the deployed UI. This supersedes the earlier local-only inspection checkpoint. Acceptance remains 50 verified of 57. Local commits are retained without a new push. Registry publication and broad cross-profile acceptance remain separate.

## Current product and release

The application is usable locally and through the hosted UI. Complete product acceptance remains unfinished.
The count is **50 verified of 57**. Open items are **41, 46, 51, 53, 54, 56, and 57**.
UI improvements refine existing accepted items. They do not close the remaining cross-profile requirements.

| Item | Last verified value |
| --- | --- |
| Hosted UI | https://teststudio.devfoodmarketmaker.com |
| Product version | 0.13.0-dev; footer displays 0.13.0.dev |
| API contract | 0.47.0 |
| Build identifier | 5d81af4f2f8d |
| Build creation | 2026-11-09T18:52:46.059Z |
| Implementation commit | db131a541496623193f8eb24b0b20cdb88a4593f |
| Source state at build | Committed source; modified is false |
| Service image digest | sha256:4908079399d8398218145ab995b7c2f9c4c1670911e906de4ea48d4b3eb19e7f |
| Task startup | 2026-11-09T18:55:59.529Z |
| UI publication | 2026-11-09T18:59:41Z |
| Live version metadata | GET /api/v1/version; public and no-store |

Later documentation and operator-tooling commits do not change the deployed implementation commit or its immutable build date.
The public API, UI bundle, and portable image labels match this release.
The local user demo is separate. Do not assume it runs the same release as the hosted installation.

## Active delivery: centered status-page content

The status component now centers all content in a maximum width of 72rem, with --ts-status-content-width available to hosts. Background color fills the public viewport; automatic margins bound wide layouts without stretching pills. Existing responsive behavior and logo proportions remain.
Normal STUDIO-STATUS-PAGES-001 passes in 7.131 seconds, run 20261108T145225.305888000-5d4d2bb3419d754f8564d43e, with cleaned execution. Checks cover centered widths at 1920, 1280, and 390 pixels, compact pills, no horizontal overflow, full-height background, and prior public/private workflows. The independent built consumer and SSR checks pass at library 0.2.1-dev.
Release 0.8.2-dev deploys before the inspection release and is preserved in the combined 0.9.0-dev delivery. Its centered-layout checks remain valid. Reuse passing evidence; preserve user page configuration and the separate demo.

## Completed delivery: remove public status Help

The public status component and route no longer render Help or configure a help topic. Standalone documents, embeds, loading, and error states share the minimal shell. Management Help remains available. The explicit public-document exception is recorded in UI guidance and acceptance 50.
The UI build and normal STUDIO-STATUS-PAGES-001 pass. Run 20261108T040919.382666000-2f47d1bdf79142a6b2ff8132 completes in 8.062 seconds with cleaned execution and passing public, iframe, dark, and private-help checks.
Release 0.8.1-dev from committed 8c8a81a is pushed and deployed as build 3f0963838890. The reviewed change set modifies only Service and Task. Seventeen hosted checks pass for API/UI identity, public component bytes, health, completed rollouts, and invalidation. The requesting public page visibly renders without Help. Temporary browser tabs close; no page configuration changes. Receipts are output/aws-mmdev/status-no-help-081.
Retain passing ignored self-test, service, UI, and dependency caches in this managed checkout for later changes. Owned test servers and browsers are stopped; temporary helpers and verification databases are removed after evidence is copied to the original checkout. Preserve the original checkout’s separate work and the user’s demo and page configuration.

## Completed delivery: configurable logo height

The user requests pixel height for optional status-page logos. Appearance adds Logo height (pixels), a live proportional preview, and blank automatic sizing. Values from 1 through 1024 persist through the existing draft and publication workflow. Narrow containers shrink the rendered image without distortion.
The shared Angular library report and component expose the same option. The built independent consumer and SSR checks pass at library 0.2.0-dev. Product 0.8.0-dev includes the committed shared reliability/library integration bf9ab8a and preserves its released logo support.
The focused SQLite check passes in 0.520 seconds and PostgreSQL with race detection in 3.046 seconds. Normal STUDIO-STATUS-PAGES-001 passes in 8.088 seconds, run 20261108T034657.315445000-1c34f428c409b3472bc67f77; its execution is cleaned. Reuse these local checks and screenshots.
Canonical OpenAPI 0.42.0 generates the matching field. Release 0.8.0-dev from committed dc558de, build 2320bb94582b, is pushed and deployed. The reviewed change set modifies only Service and Task. Twenty-eight hosted checks pass; the public browser measures the logo at exactly 192 by 192 pixels and centered. The requesting editor shows the height control with its existing logo and automatic value. No user page values or publication change. The generic verification page is unpublished, owned sessions and tabs close, and the helper is removed. Receipts are output/aws-mmdev/status-height-080. Library 0.2.0-dev is packed locally under output/angular-status; registry publication is separate.

## Completed delivery: optional status-page logos

The user requests an optional centered logo, either uploaded or referenced. Appearance supports both sources and removal. The API validates uploaded PNG/JPEG data and public HTTPS URLs without fetching them; uploaded UI files preserve transparency and become bounded PNGs. Published snapshots retain their logo independently of draft edits.
The UI build, focused SQLite cases, race-enabled PostgreSQL cases, generated client synchronization, and normal STUDIO-STATUS-PAGES-001 pass. The final browser run is 20261108T031201.518911000-baa5d1135a9b9ec9387d2f92, 7.171 seconds. Review retained screenshots rather than repeating these checks.
Release 0.6.0-dev and canonical OpenAPI 0.40.0 are committed and pushed. Build dca2ee3e4ddf from 3d13a08 is deployed to MMDev. Nine local AWS schema/package checks and thirty-five hosted checks pass. The actual public image decodes at 256 by 64 pixels and centers correctly; the requesting account sees Logo source in Appearance. Its expired application session resumes through its existing organization/Google SSO without credential entry. No page values or publication are changed.
Continue in this managed checkout on codex/status-page-logo. Both remote branches include the merged reliability/library source, logos, and configurable height. Preserve any work in the original checkout and incorporate upstream before another deployment. Hosted receipts and screenshots are output/aws-mmdev/status-logo-060. The owned verification page is unpublished, owned sessions and temporary tabs close, and the helper is removed.

## Latest delivered UI changes

The logo and copy-button releases use this managed checkout. The app retains it as this chat’s primary checkout and rejects archival. Evidence is retained in the original checkout; owned temporary build data and processes are cleaned. Keep logo work here on codex/status-page-logo. Preserve and update the separate shared implementation branch before continuing its reliability work.

Release 0.5.1-dev adds Copy Public URL and Copy Iframe embed code beside the sharing fields. Both announce success or explain manual copying when browser clipboard access is unavailable. The normal local browser workflow passes in 6.709 seconds, run 20261108T025119.540693000-f97987f5b8ae0d58e4383b5e; sixteen hosted identity, asset, health, rollout, and invalidation checks pass. The requesting account sees both buttons. Its page remains unchanged and the temporary tab closes. Receipts are output/aws-mmdev/status-copy-051.

Status-page details now have separate Edit and Review publication routes. Optional vanity names are unique across the installation and reserve on Save draft. Publication activates the saved name, while permanent generated URLs remain compatible. The deployed 0.5.0-dev release includes this workflow and shared guidance for pages, dialogs, and inline controls.

On 2026-11-07, HTTPS probe recent checks switch from checkmark tiles to compact colored vertical pills.
The patch is included in deployed 0.3.0-dev alongside scoped membership management. The release table above identifies the published artifact.
The UI build and STUDIO-PROBE-OVERVIEW-001 pass. Run 20261107T153825.676380000-f359987ef19bbbc8498e69d0 verifies 30 checks, keyboard links, light and dark themes, and a 390-pixel layout.
Evidence is complete, execution is cleaned, and owned processes exit. The total stays 50 verified of 57; items 34, 49, and 50 are refined.
Project and Application members support reviewed role changes and conversion recovery. Project Status pages now provides public, embeddable probe summaries with reviewed publication and time-based availability. Reuse the passing evidence below.

## Recent completed work to preserve

- HTTPS probes now has target-specific results, reviewed single and bulk actions, explicit adoption, multiple-address setup, and shared-component visual help.
- The user-selected FoodMarketMaker project contains nine private configured probes across three applications and stages. Initial normal hosted runs all pass.
- Nine five-minute schedules are enabled. All nine first automatic firings pass with complete evidence.
- Normal project roles grant the existing requesting user access. Installation privileges remain unchanged.
- Project creation uses /projects/new. The project list has a Create project action and no redundant count.
- Project tiles are square and use bounded widths. Optional private backgrounds support upload and browser-assisted website tab capture.
- Image capture requires a browser sharing prompt. It takes one frame, stops sharing, and requires explicit Save image.
- Capture is not automatic server rendering of an entered URL. Unsupported browsers retain upload. It starts no test run.
- The sidebar footer shows version, build identifier, and UTC creation date. Build expands source details. Mobile Navigation reveals the footer.
- VERSION and scripts/version.mjs synchronize release numbers. Standard builders stamp the UI and API with one generated metadata snapshot.
- Generic OIDC supports the selected organization issuer. Cognito owns Google federation; TestStudio contains no Google-specific sign-in path.
- The organization and bootstrap administrator providers are both enabled. This explains the two SSO buttons; do not remove backup access silently.
- Requested project creation and existing-project access were granted through normal authorization. Do not repeat grants or infer access from sign-in.

See [versioning](versioning.md), [project images](project-images.md), [first installation](first-installation.md), and [MMDev operation](aws-mmdev.md).

## MarketMaker deployment entry point

On 2026-11-07 the user selects /Users/johnbauer/Dev/MarketMaker/api for the deployment launcher.
Its deploy-teststudio-dev.py pins mm_dev, account 253455159142, region us-west-2, stack teststudioapi, and cluster marketmaker-testing-dev.
The launcher delegates to this checkout's maintained publish-aws-core.py. It supports a source override and works from another working directory.
Six focused launcher checks pass. Its actual prepare invocation confirms 0.2.0-dev / 6caa2b0ca508 is already deployed with no newer runtime changes.
The current stack is UPDATE_COMPLETE. Both teststudioapi and testing-server have completed healthy rollouts. No duplicate build or deployment starts.
The README in the API checkout documents status, prepare, review, execute, finish, and per-release receipt selection.
This operator convenience changes no application release and closes no acceptance criterion. Continue the default queue below when requested.

## Status-page access repair

After the 0.4.0-dev release, the requesting account’s FoodMarketMaker sidebar omits Status pages because its four compatibility roles lack the new statuspages permissions.
The exact signed-in internal account is confirmed from its account screen, without matching or merging by email.
Normal audited APIs grant only statuspages.read, statuspages.manage, and statuspages.publish in foodmarketmaker. The original four assignments remain unchanged. Status-page deletion, System access, and status-page permissions in aws-core remain denied.
The requesting browser now shows Status pages under Project overviews and its Create status page control. The tab is left on the requested project’s Status pages screen.
Private before/after receipts and visual evidence are output/aws-mmdev/status-page-access. Owned administrator sessions close and temporary helpers are removed.
This is a scoped configuration repair, with no application rebuild or redeployment. Release 0.4.0-dev remains unchanged. Future new-feature checks must include the requesting account’s compatibility-role capabilities, not only administrator verification.

## Completed request: focused management and vanity status URLs

The user requests clearer status-page workflows and duplicate-safe vanity URLs. Implementation uses separate create, edit, detail, and publication-review routes. Shared UI guidance follows Create project and distinguishes dedicated pages, short dialogs, and inline controls.
Vanity names are optional, normalized, globally reserved on save, and activated on publication. SQLite and PostgreSQL verify competing claims produce one winner. Existing generated URLs remain compatible. Renamed or deleted aliases cannot be claimed by another page.
The final changed browser workflow passes in 6.497 seconds, run 20261108T023727.088360000-03fb9e6133bcc031ea0886d4. It covers duplicate-name feedback, correction, narrow authoring screens, reviewed publication, rename snapshots, permanent-link compatibility, revision recovery, anonymous embedding, themes, and unpublishing. Release 0.5.0-dev is committed, pushed, and deployed as build a21537b24309. Twelve package checks and thirty-four hosted checks pass. The API and published UI match; CloudFront invalidation completes. Both services have healthy completed rollouts. Private receipts are output/aws-mmdev/status-vanity-050.
The requesting account’s actual browser verifies the saved detail page and dedicated editor with an optional vanity field. No existing page is edited or republished. The temporary tab closes, leaving both user tabs intact. The owned generic verification page is unpublished, no new probe run executes, the owned session closes, and the helper is removed. Preserve the current scoped grants and page configuration.

## Local delivery: shared reliability history for all checks

The approved [shared reliability design](trends.md#shared-reliability-capture) is implemented locally across native HTTP, Go, browser, and API result paths, including unknown terminal selections, atomic capture, safe replay, retained-history backfill, hourly and daily summaries, and bounded display buckets.
Time-based availability requires an explicit reporting policy and current definition/target matching. Ordinary tests retain outcome and timing history without implied uptime.
The [Angular library](../web/projects/status/README.md), @wtiger001/teststudio-angular 0.1.0-dev, supplies a stylable bar, page, and optional public client. TestStudio reuses it. An independently installed Angular 22.2.4 consumer verifies styling, cross-origin data, withdrawal, cleanup, and server rendering.
Product 0.7.0-dev is prepared locally. API contract 0.42.0 and database metadata version 2 cover this delivery. Compact reliability data has no automatic expiry; evidence retention remains separate. Installation export/import preserves all reliability tables and pending jobs.
The ledger records focused SQLite/PostgreSQL checks, the normal browser receipts, and dense-year and multi-year measurements. Acceptance remains 50 verified of 57.
Registry publication and target deployment remain pending. The upstream logo implementation is integrated locally and its separately verified hosted release remains intact.

## Local follow-up: individual reliability checks

The latest request adds Individual checks for any report period and Inspect period for each pill. The shared table reads compact observations directly, pages through older results without aggregation, and opens authorized saved runs privately. Public native and independent Angular pages expose only time, outcome, duration, and censoring, and preserve source authority, target matching, and withdrawal.
Focused SQLite/PostgreSQL and normal STUDIO-RELIABILITY-001 pass; final UI run 20261108T035846.711583000-448fee9fed024a3d2b99416e retains complete evidence and cleaned execution state. The built package consumer and SSR pass. The ledger records exact scope and limits. Acceptance remains 50 verified of 57. This follow-up is local and does not deploy or supersede the concurrent height release. Incorporate that upstream delivery before publishing a later combined release.

## Next implementation priorities

### Completed request: public status pages

The user approves public URLs with iframe embedding, private drafts followed by explicit Publish, and time-based availability on 2026-11-07.
Follow [the status-page plan](status-pages-plan.md) and [feature guide](status-pages.md).
The implementation adds project-owned pages, ordered public probe text, background color, shared range selection, recent pills, and availability donuts with explicit unknown coverage.
New statuspages permissions keep administration at Project scope. Public projections contain no target addresses, private identifiers, diagnostics, or evidence.
Focused SQLite and PostgreSQL checks cover duration calculations, complete retained history beyond 1000 runs, target changes, publication revisions, privacy, and revocation. UI and local AWS packaging compile and validate.
STUDIO-STATUS-PAGES-001 passes in 2.843 seconds through normal local execution, including actual cross-origin embedding, all time windows, privacy, recovery, and light/dark narrow layouts. Eight UI utility checks pass.
Versioned publication and target verification now pass. Reuse the retained local evidence under output/selftest/20261107T223233.325865000-9897ae3890872520b06d7718.
Version 0.4.0-dev is deployed from committed 3fc524f, build 8f7db1066f2e. All implementation and operator commits are pushed.
The twelve service-package checks and twenty-three hosted checks pass. Two independent generic pages verify private drafts, explicit publication, snapshot isolation, all ranges, unknown coverage, privacy, framing headers, and immediate unpublishing. Their single native HTTPS run passes in 140.412653 ms.
Both verification pages are unpublished, the owned session closes, and the temporary helper is removed. Public UI assets and API metadata match. CloudFront invalidation completes; the stack and both services have healthy completed rollouts. Receipts are output/aws-mmdev/status-pages-040.
Use publish-aws-core.py prepare --public-status for the scoped CloudFront document behavior and response policy, then review the concrete change set. Ordinary image releases retain the image-only path.
Preserve existing projects, probes, schedules, memberships, and the port 8040 demo.


### Completed request: scoped membership UI and MMDev release

The user keeps Project and Group, selects System, and requests planning for Owner, Contributor, Developer, Member, and Reader on 2026-11-07.
Member runs tests. Contributor sits directly below Owner, with scoped administration except resource deletion and adding, removing, or changing Owners. Guard indirect Owner changes through groups and custom policies too.
The [role plan](roles-plan.md) defines cumulative Project and Application roles, a separate System Admin, inheritance, shared-resource boundaries, owner safeguards, and reviewed migration.
Application-only roles require server authorization; existing application filters do not establish access isolation.
The user requests API implementation. Backend milestones 1 through 3 are implemented locally with focused SQLite and PostgreSQL evidence.
API contract 0.37.0 and the generated identity client cover scoped assignments, atomic standard-role replacement, capability and access explanations, owner status, exact user lookup, conversion previews, apply, and rollback previews.
Standard roles use native permission-service evaluation with server-resolved Application context. Full mixed runs and shared source mutation remain restricted to Project authority.
New ordinary project creators receive one Owner assignment. Existing role bundles and memberships stay unchanged until reviewed conversion.
The [access guide](access.md#standard-project-and-application-roles) documents routes, previous-role revision guards, Owner safeguards, and migration recovery.
The retained PostgreSQL selections pass 59 initial checks, 18 focused follow-up checks, and two final delegation/Owner guard checks. Matching SQLite and targeted race checks pass. Reuse these receipts under output/roles-api unless changed behavior requires another check.
The user then requests the necessary membership UI, commit, push of all branch commits, and MMDev deployment.
Project members and Application members now support reviewed standard-role creation and replacement, inherited access, Contributor restrictions, ownership status, conversion receipts, and reviewed rollback. System labels, exact application explanations, and contextual help are included.
The UI build, eleven focused utility checks, STUDIO-ROLES-001, STUDIO-PERMISSIONS-001, and new STUDIO-APPLICATION-ACCESS-001 pass locally. The ledger records timings and retained runs. All owned controllers and execution containers exit.
Version 0.3.0-dev is committed, pushed, and deployed through a reviewed Service-and-Task-only change set. Both services remain healthy and the stack is UPDATE_COMPLETE.
Fourteen hosted permission checks pass, including actual local-password sessions, application-only execution, group access, Contributor Owner rejection, current-session revocation, and rejection of an already-issued evidence URL. One generic native HTTPS run passes in 98.745277 ms. All four owned sessions close and three verification users are disabled.
The public index, primary assets, membership bundle, and API metadata match the committed artifact. CloudFront invalidation is complete. Receipts remain under output/aws-mmdev/roles-030.
Existing hosted memberships and the nine private schedules are preserved. No hosted legacy conversion is performed. Preserve the port 8040 demo.
The count stays 50 verified of 57. The default AWS worker queue remains available after this completed request.

### Completed request: HTTPS probes features and help

The [HTTPS probes guide](https-probes.md) documents the delivered workflow. The [six-milestone plan](https-probes-plan.md) records implementation and verification evidence.
Milestones 1 through 6 are complete for this request. Local workflows pass and release 0.2.0-dev is deployed from committed source 6d2444f with matching public UI and API assets.
The user selects a new FoodMarketMaker project. Its nine explicit probes have passing normal runs, complete evidence, and enabled five-minute schedules.
Private evidence download and the requesting user's project roles pass inspection. All nine first automatic schedule firings pass with complete evidence.
Return to the default AWS worker queue below. Do not repeat onboarding, initial runs, or local workflow tests.
Important receipts are output/aws-mmdev/https-probes-release.json and foodmarketmaker-probes.json. They contain run checkpoints and publication metadata, not credentials.
Use the tracked publish-aws-core.py and manage-https-probes.py helpers. The latter derives schedule execution location from the actual retained run.
Keep private domain configuration outside product defaults and self-tests. The acceptance count stays 50 verified of 57, with the same seven open items.

### Critical: complete the requested import and execution path

The MMDev core currently supports native HTTP checks. Separate AWS coded execution and source preparation are not installed.
Continue with one bounded worker milestone using [the staged worker package](aws-workers-package.md).
Inspect the deployed core and package inputs before proposing changes. Do not recreate the core or reuse another application's resources.
The staged sequence is prepare, verify private networking, connect, seal, and enable the authorized worker capabilities.
Keep actual target verification separate from local functional implementation checks. Preserve the existing core and other ECS services.

The existing application test repository also needs reviewed additional browser origins and a managed browser credential or fixture bridge.
These are shared product capabilities. Do not hardcode MarketMaker domains, sign-in providers, or credentials into TestStudio.
Current repository synchronization binds targetless suites. Environment-backed source bundles use the reviewed ZIP preparation path.
Read [test repository contracts](test-repositories.md), [browser execution](browser-execution.md), and [deployment readiness](deployment-readiness.md) before enabling these workflows.

### Important: keep contracts and acceptance evidence complete

The version and project-image endpoints are documented in their feature guides but are absent from contracts/openapi.yaml.
Reconcile those API families during the next contract documentation pass. The canonical contract already states that some families remain pending.
Do not describe its current coverage as complete or overwrite its separate contract version with the product version.

Core application publication and HTTPS onboarding now use tracked publish-aws-core.py and manage-https-probes.py tooling.
Replace other useful ad hoc helpers before relying on them for worker releases.
The ignored .data/aws-mmdev snippets remain convenience history. Use tracked templates and operator guides as the durable deployment contract.

| Priority | Acceptance item | Remaining scope |
| --- | --- | --- |
| 1 | 51 | Consistent execution status, cancellation, deadlines, artifacts, and recovery on remaining profiles. |
| 2 | 46 | Retention, quotas, pins, encryption, and identity behavior on remaining cloud evidence profiles. |
| 3 | 54 | Equivalent configured OIDC and project-permission scenarios on remaining profiles. |
| 4 | 41 | Common timing and timeout behavior across remaining runner and deployment profiles. |
| 5 | 56 | Generic coded suites, progress, history, reports, and trends on each required target. |
| 6 | 57 | Complete common scenarios across every required deployment profile. |
| 7 | 53 | Independent Azure installation and coded execution after the user ends its deferral. |

Azure remains deferred for implementation and verification. Preserve its staged packages without adding inspectors or account requirements.
Local Kubernetes already has retained execution, identity, failover, gateway, and recording evidence. Reuse it unless changed behavior requires another check.

### Nice to Have

Optional installer convenience, automatic updates, and unrelated dashboard polish follow required workflow gaps.
Explicit user feedback on usability takes priority over this default ordering. Keep each correction small and documented.

## Deployment boundaries and recovery locations

| Target | Required boundary |
| --- | --- |
| AWS core | Profile mm_dev; account 253455159142; region us-west-2; cluster marketmaker-testing-dev; service teststudioapi. |
| AWS resources | Stacks teststudioapi-foundation and teststudioapi; preserve existing testing-server and other workloads. |
| External Cognito | mm_prod only for the authorized dedicated TestStudio client in pool us-west-2_p1Epu0go8. Preserve existing clients and federation. |
| Local Kubernetes | Dedicated teststudio-k3s profile; explicit .data/local-k3s/kubeconfig.yaml; no inherited user context. |
| Local demo | Preserve port 8040, SQLite data, and retained evidence. Use the documented launcher lifecycle only when needed. |
| Azure | Deferred. No new implementation, deployment, or verification without a later user request. |

The last AWS update leaves both TestStudio stacks at UPDATE_COMPLETE and one healthy teststudioapi task with a completed rollout.
The existing testing-server retains one healthy task. These observations are recorded snapshots.
Verify account and current resource state before any subsequent mutation.

Tracked [MMDev instructions](aws-mmdev.md) describe credential references and stack parameters. They contain no credential values.
Local parameter references are in .data/aws-mmdev/core-parameters.json. Deployment receipts are under output/aws-mmdev.
Useful receipts include published-image.json, build-version-deployment.json, and build-version-publication.json.
The administrator credential remains in Secrets Manager at teststudioapi/initial-administrator. Never print it or place it in arguments.
Full CloudFront distribution configuration contains guarded origin headers. Query only needed public fields.
Temporary Docker login configuration must stay private and be removed. Preserve the user's Docker credentials and contexts.

## Separate application repository

The marketmaker-tests integration is committed locally as f233b55. Its working tree was clean at this review.
On this workstation it is at /Users/johnbauer/Dev/MarketMaker/marketmaker-tests. It is a separate repository with its own instructions.
Only local commits were authorized. Remote availability is not verified; do not push without an explicit request.

The native export contains 604 existing definitions: 505 Website and 99 FoodSearcher.
Local assembly and isolated discovery pass. This does not mean those application tests ran or passed.
The repository's npm run teststudio:bundle command produces a source ZIP and output/teststudio/bundle.json.
Read its docs/teststudio.md and TestStudio's [generic repository guide](test-repositories.md) before changing the adapter.
Do not copy its tests, credentials, browser state, or application configuration into this product.

## Targeted local verification and release

Choose the check that exercises the changed behavior. The commands in AGENTS.md are starting points, not a mandatory checklist.
Select browser IDs from tests/selftest/catalog.json and use the normal TestStudio execution path.
Do not rerun completed checks after documentation-only edits.

| Recent workflow | Retained local evidence |
| --- | --- |
| Dedicated project wizard | STUDIO-WIZARD-001; run 20261106T131801.535325000-755c948292cb022d7125d961; 7.533 seconds. |
| Square tiles, upload, real tab capture, and removal | STUDIO-PROJECTS-001; run 20261106T140014.789307000-4ac204a42ec7e6e01bf4e4e1; 1.688 seconds. |
| Shared UI/API version and footer | STUDIO-VERSION-001; run 20261106T141549.201293000-da61d7429e7ac5e9b3ab33b7; 0.532 seconds. |

Each recorded run has complete evidence and a cleaned execution ledger. Owned verification processes exit afterward.
Screenshots and summaries are under output/selftest/{runId}. These ignored files may be absent in a fresh checkout.
Project capture uses headed Chromium on an owned virtual display in its isolated test container. Other browser checks remain headless.

For changed application behavior, choose the version bump, update affected help and guides, verify locally, and commit the implementation.
Build the release from that committed source. A later documentation commit may record the deployed artifact without rebuilding it.
Publish matching UI and service artifacts. Verify build identity and health; do not rerun a broad functional suite on the hosted installation.
The [release guide](versioning.md) covers cache stability, canonical SemVer, four-part display, local stamped builds, and image labels.

No cloud or browser work is required merely to resume this plan. Start with the selected implementation task.
