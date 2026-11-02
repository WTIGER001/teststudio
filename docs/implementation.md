# Implementation verification

The full plan remains required scope. Check an item only after its implementation and verification pass.

## Acceptance criteria

- [ ] A new project, application, environment, and suite can be registered without editing Go or TypeScript source.
- [ ] Users, groups, membership, custom roles, assignments, and effective permissions are manageable through authorized API and UI workflows.
- [ ] `wtiger001/go-permissions` evaluates access on PostgreSQL and SQLite through the same service and identity contracts.
- [ ] User, group, role, object, and project grant checks cover explicit denies, expired grants, inheritance, and assignment revocation.
- [ ] Project administrators cannot grant installation privileges or affect another project's groups, assignments, results, or evidence.
- [ ] Membership and grant revocation reach all instances within the documented limit. Disabled users and revoked sessions fail their next request.
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
- [ ] HTTPS checks run manually and on schedules, with configurable assertions, TLS checks, response-time history, and normal result records.
- [ ] Adding an HTTPS check requires no browser or service deployment.
- [ ] Go suites preserve test identities, disable cached execution, and distinguish build failures from test assertion failures.
- [ ] Go unit suites run without artificial web targets or production credentials.
- [ ] API suites run through guarded direct clients and produce the same history and trends as browser tests.
- [ ] Package events, subtest aggregates, retries, and API steps do not inflate test counts.
- [ ] The Go service and UI build from this repository without MarketMaker imports, packages, shared resources, or sibling checkouts.
- [ ] No existing tests, fixtures, catalog records, configuration, credentials, reports, or history are copied into the product.
- [ ] Installation starts with empty registries and can load only explicitly selected new examples.
- [ ] AWS ECS and CloudFront, Kubernetes, Azure Container Apps, and local-server profiles pass the common acceptance scenarios.
- [ ] Local, Kubernetes, and Azure installations operate without AWS services or credentials.
- [ ] Local durable operation works without a cloud account.
- [ ] Provider adapters support execution status, cancellation, deadlines, artifacts, and recovery consistently.
- [ ] Hosted identity and private report access work through installation-specific configuration.
- [ ] Local and scheduled executions produce the same logical run and result records.
- [ ] Protected environments reject unsafe selections and unsafe dependencies.
- [ ] Manual and scheduled notifications use the same configured completion path.
- [ ] Replica failover creates no duplicate logical run. Notification retries retain their documented at-least-once behavior.
- [ ] Test-only publication requires no API or UI deployment.
- [ ] Provider credentials remain outside browser configuration and application logs.
- [ ] Light, dark, and system modes pass browser workflow and keyboard checks.

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
