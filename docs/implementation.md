# Implementation verification

The full plan remains required scope. Check an item only after its implementation and verification pass.

## Acceptance criteria

- [x] A new project, application, environment, and suite can be registered without editing Go or TypeScript source.
- [ ] Users, groups, membership, custom roles, assignments, and effective permissions are manageable through authorized API and UI workflows.
- [x] `wtiger001/go-permissions` evaluates access on PostgreSQL and SQLite through the same service and identity contracts.
- [ ] User, group, role, object, and project grant checks cover explicit denies, expired grants, inheritance, and assignment revocation.
- [ ] Project administrators cannot grant installation privileges or affect another project's groups, assignments, results, or evidence.
- [x] Membership and grant revocation reach all instances within the documented limit. Disabled users and revoked sessions fail their next request.
- [ ] OIDC sign-in supports configured providers with code flow, PKCE, state, nonce, issuer and audience validation, key rotation, and logout.
- [ ] Duplicate email addresses from different issuer-subject pairs do not merge accounts or transfer access.
- [ ] Unmapped, missing, incomplete, or outdated external groups do not silently preserve elevated project access.
- [ ] Browser sessions keep provider tokens server-side. Expiry, revocation, callback rejection, and request-forgery protection pass focused checks.
- [ ] SQLite restarts preserve users, memberships, assignments, custom grants, disabled roles, and sessions without a PostgreSQL service.
- [ ] Two newly configured OIDC providers work with the same internal directory and project-scoped permissions across deployment profiles.
- [ ] Catalog creation, editing, validation, publication, import, export, bulk actions, archive, and restoration work through both API and UI.
- [ ] Concurrent catalog edits produce explicit conflicts. Source synchronization preserves service-managed metadata.
- [ ] Catalog changes do not change existing run snapshots. Disabled tests and invalid dependencies block new dispatch explicitly.
- [ ] Run details expose attempts, steps, context, logs, screenshots, traces, and configured attachments without requiring HTML reports.
- [ ] Available evidence survives service restarts. Duplicate events and upload retries do not create duplicate attachments or rerun tests.
- [ ] Capture gaps, failed uploads, expired files, and incomplete evidence remain visible beside the observed test outcome.
- [ ] Artifact access respects project permissions. Private trace viewing requires no public upload or third-party viewer.
- [ ] Retention, quotas, and evidence pins work across local, S3, and Azure artifact adapters.
- [ ] PostgreSQL and SQLite pass the same storage behavior checks, including claims, revision conflicts, and restart recovery.
- [ ] Local SQLite installation requires no database server. Export and import to PostgreSQL preserve identifiers and artifact references.
- [ ] Declarative uploads, complete source bundles, and registered GitHub repositories can produce versioned releases through the UI.
- [ ] One repository can provide multiple suites, including suites beside existing application source.
- [ ] Failed or duplicate preparation events preserve the active release and do not create unintended test runs.
- [ ] Runs retain their release across updates, activation changes, and rollbacks.
- [ ] Removed tests and archived suites retain their history and required artifacts.
- [ ] Newly written generic suites complete runs and produce progress, history, reports, and trends on each deployment target.
- [ ] Result-state trends use stacked area charts with consistent state colors, final logical counts, accessible values, and visible missing-history gaps.
- [x] Native HTTP attempts support comparable timing history, bounded adaptive timeouts, slowdown criteria, and API and UI estimates.
- [ ] Go and browser runners apply the same timing contract across execution and deployment profiles.
- [ ] HTTPS checks run manually and on schedules, with configurable assertions, TLS checks, response-time history, and normal result records.
- [ ] Adding an HTTPS check requires no browser or service deployment.
- [ ] Go suites preserve test identities, disable cached execution, and distinguish build failures from test assertion failures.
- [ ] Go unit suites run without artificial web targets or production credentials.
- [ ] API suites run through guarded direct clients and produce the same history and trends as browser tests.
- [ ] Package events, subtest aggregates, retries, and API steps do not inflate test counts.
- [x] TestStudio executes its own new Playwright suite through normal publication, execution, timing, result, and evidence contracts.
- [ ] The Go service and UI build from this repository without MarketMaker imports, packages, shared resources, or sibling checkouts.
- [ ] No existing tests, fixtures, catalog records, configuration, credentials, reports, or history are copied into the product.
- [ ] Installation starts with empty registries and can load only explicitly selected new examples.
- [ ] AWS ECS and CloudFront, Kubernetes, Azure Container Apps, and local-server profiles pass the common acceptance scenarios.
- [ ] Local, Kubernetes, and Azure installations operate without AWS services or credentials.
- [x] Local durable operation works without a cloud account.
- [ ] Provider adapters support execution status, cancellation, deadlines, artifacts, and recovery consistently.
- [ ] Hosted identity and private report access work through installation-specific configuration.
- [ ] Local and scheduled executions produce the same logical run and result records.
- [ ] Protected environments reject unsafe selections and unsafe dependencies.
- [ ] Manual and scheduled notifications use the same configured completion path.
- [ ] Replica failover creates no duplicate logical run. Notification retries retain their documented at-least-once behavior.
- [ ] Test-only publication requires no API or UI deployment.
- [ ] Provider credentials remain outside browser configuration and application logs.
- [x] Light, dark, and system modes pass browser workflow and keyboard checks.

## Feature log

Record feature commits and targeted validation here.

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
