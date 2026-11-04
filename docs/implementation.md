# Implementation verification

The full plan remains required scope. Check an item only after its implementation and verification pass.

Implementation resumes at the user's request on 2026-11-03.
The acceptance count is 37 verified of 57.
Focus on implementation and targeted local checks. Cloud verification remains separate work.

## Acceptance criteria

These stable numbers match the priority list supplied to the user.
Report progress against these numbers. Keep broad criteria unchecked until their full scope is verified.

- [x] **01.** The Go service and UI build from this repository without MarketMaker imports, packages, shared resources, or sibling checkouts.
- [ ] **02.** No existing tests, fixtures, catalog records, configuration, credentials, reports, or history are copied into the product.
- [ ] **03.** Installation starts with empty registries and can load only explicitly selected new examples.
- [x] **04.** Local durable operation works without a cloud account.
- [x] **05.** A new project, application, environment, and suite can be registered without editing Go or TypeScript source.
- [ ] **06.** PostgreSQL and SQLite pass the same storage behavior checks, including claims, revision conflicts, and restart recovery.
- [x] **07.** SQLite restarts preserve users, memberships, assignments, custom grants, disabled roles, and sessions without a PostgreSQL service.
- [x] **08.** Local SQLite installation requires no database server. Export and import to PostgreSQL preserve identifiers and artifact references.
- [x] **09.** Local accounts, OIDC accounts, and mixed sign-in work through the same directory, permission checks, and managed sessions.
- [x] **10.** Users, groups, membership, custom roles, assignments, and effective permissions are manageable through authorized API and UI workflows.
- [x] **11.** `wtiger001/go-permissions` evaluates access on PostgreSQL and SQLite through the same service and identity contracts.
- [ ] **12.** User, group, role, object, and project grant checks cover explicit denies, expired grants, inheritance, and assignment revocation.
- [x] **13.** Project administrators cannot grant installation privileges or affect another project's groups, assignments, results, or evidence.
- [x] **14.** Membership and grant revocation reach all instances within the documented limit. Disabled users and revoked sessions fail their next request.
- [x] **15.** Local credential changes, resets, disabling, and recovery preserve audit records and the last usable administrator.
- [x] **16.** OIDC sign-in supports configured providers with code flow, PKCE, state, nonce, issuer and audience validation, key rotation, and logout.
- [x] **17.** Duplicate email addresses from different issuer-subject pairs do not merge accounts or transfer access.
- [x] **18.** Unmapped, missing, incomplete, or outdated external groups do not silently preserve elevated project access.
- [x] **19.** Browser sessions keep provider tokens server-side. Expiry, revocation, callback rejection, and request-forgery protection pass focused checks.
- [ ] **20.** Provider credentials remain outside browser configuration and application logs.
- [x] **21.** Catalog creation, editing, validation, publication, import, export, bulk actions, archive, and restoration work through both API and UI.
- [ ] **22.** Declarative uploads, complete source bundles, and registered GitHub repositories can produce versioned releases through the UI.
- [x] **23.** One repository can provide multiple suites, including suites beside existing application source.
- [x] **24.** Test-only publication requires no API or UI deployment.
- [x] **25.** Concurrent catalog edits produce explicit conflicts. Source synchronization preserves service-managed metadata.
- [x] **26.** Failed or duplicate preparation events preserve the active release and do not create unintended test runs.
- [x] **27.** Runs retain their release across updates, activation changes, and rollbacks.
- [ ] **28.** Catalog changes do not change existing run snapshots. Disabled tests and invalid dependencies block new dispatch explicitly.
- [x] **29.** Removed tests and archived suites retain their history and required artifacts.
- [x] **30.** TestStudio executes its own new Playwright suite through normal publication, execution, timing, result, and evidence contracts.
- [x] **31.** Go unit suites run without artificial web targets or production credentials.
- [x] **32.** Go suites preserve test identities, disable cached execution, and distinguish build failures from test assertion failures.
- [x] **33.** Adding an HTTPS check requires no browser or service deployment.
- [ ] **34.** HTTPS checks run manually and on schedules, with configurable assertions, TLS checks, response-time history, and normal result records.
- [ ] **35.** API suites run through guarded direct clients and produce the same history and trends as browser tests.
- [x] **36.** Package events, subtest aggregates, retries, and API steps do not inflate test counts.
- [ ] **37.** Local and scheduled executions produce the same logical run and result records.
- [ ] **38.** Protected environments reject unsafe selections and unsafe dependencies.
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
- [ ] **52.** Replica failover creates no duplicate logical run. Notification retries retain their documented at-least-once behavior.
- [ ] **53.** Local, Kubernetes, and Azure installations operate without AWS services or credentials.
- [ ] **54.** Two newly configured OIDC providers work with the same internal directory and project-scoped permissions across deployment profiles.
- [ ] **55.** Hosted identity and private report access work through installation-specific configuration.
- [ ] **56.** Newly written generic suites complete runs and produce progress, history, reports, and trends on each deployment target.
- [ ] **57.** AWS ECS and CloudFront, Kubernetes, Azure Container Apps, and local-server profiles pass the common acceptance scenarios.

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
