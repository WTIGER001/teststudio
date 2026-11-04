# TestStudio

TestStudio is an independent test catalog, execution, and evidence platform.
The implementation follows [the product plan](docs/plan.md).
The [acceptance ledger](docs/implementation.md) records verified work and pending requirements.
The [UI contract](docs/ui.md) defines tester workflows, shared interaction patterns, and documentation requirements.
The [help guide](docs/help.md) explains contextual guide maintenance.
Use the [workflow completion checklist](docs/plan.md#workflow-completion-checklist) for each behavior change.

## Local demonstration

Start a separate demonstration with new generic examples:

```sh
node scripts/demo.mjs start
```

The command builds the application and starts a local server, source worker, and controlled target.
It requires Go, Node, npm, and Docker. The browser example runs in a local container.
Unchanged builds and the trusted browser runtime are reused.
The terminal prints the application and target addresses when setup completes.
Select Open local workspace to explore the prepared project.
Use Ctrl+C or `node scripts/demo.mjs stop` to stop its processes.
History and evidence stay in `.data/demo` for the next start.
The [demonstration guide](docs/demo.md) provides a short walkthrough and explains the intentional failures.
Normal local development still starts with an empty project registry.

## Guide index

| Task or audience | Guides |
| --- | --- |
| Start and organize testing | [Demo](docs/demo.md), [Applications](docs/applications.md), [Suites](docs/suites.md). |
| Create tests | [Create test](docs/create-tests.md), [HTTPS checks](docs/http-checks.md), [Browser recording](docs/browser-recording.md), [Website execution](docs/browser-execution.md), [Playwright editor](docs/playwright-editor.md), [Go testing](docs/go-testing.md). |
| Author request-based suites | [Coded API policies](docs/api-testing.md). The policy foundation and API starter are implemented; dedicated authoring and full worker acceptance remain open. |
| Manage test definitions | [Catalog](docs/catalog.md), [Saved selections](docs/saved-selections.md), [Releases](docs/releases.md). |
| Inspect runs | [Results](docs/results.md), [Test history](docs/test-history.md), [Evidence](docs/evidence.md), [Timing](docs/timing.md), [Trends](docs/trends.md). |
| Automate testing | [Schedules](docs/schedules.md), [Notifications](docs/notifications.md). |
| Import and synchronize | [Native uploads](docs/sources.md), [Source bundles](docs/source-bundles.md), [Prepared suites](docs/prepared-suites.md), [Repositories](docs/repository-sources.md), [CLI sign-in](docs/cli-sign-in.md). |
| Administer and operate | [Access](docs/access.md), [Artifact storage](docs/artifact-storage.md), [Backup and database transfer](docs/data-transfer.md), [Provider profiles](docs/providers.md), [Notification transports](docs/notification-transports.md). Current installation commands appear below. |
| Use the API | [OpenAPI contract](contracts/openapi.yaml), [Source webhook contract](contracts/source-webhooks.openapi.yaml), and the affected feature guide. |
| Contribute | [Plan](docs/plan.md), [Acceptance ledger](docs/implementation.md), [UI contract](docs/ui.md), [Help maintenance](docs/help.md), [Self-testing](docs/self-testing.md). |

## Local development

Requirements: Go 1.26.2 or later and Node 22.22.3, Node 24.15, or Node 26 or later.

Build the Angular UI:

```sh
npm --prefix web ci
npm --prefix web run build
```

Start the local server:

```sh
go run ./cmd/teststudio --local
```

Open http://127.0.0.1:8040 and select Open local workspace.
The explicit local mode binds to loopback and provides local installation administrator access.
The server starts with an empty project registry.
Data stays under `.data/` by default.
The [first installation guide](docs/first-installation.md) explains empty startup and selecting optional HTTP examples without Docker.

Create a project, application, suite, and environment through the UI.
Add native HTTPS checks, save catalog changes, validate a draft, and publish a release.
Use Import and export to transfer native JSON definitions through a reviewed preview.
Use table selection for bulk lifecycle actions. Duplicate and History appear on each test row.
The [catalog guide](docs/catalog.md) describes schemas, revision checks, and request limits.
The [saved selection guide](docs/saved-selections.md) describes reusable identifiers, versioned filters, previewed dependencies, and frozen run selections.
The [source upload guide](docs/sources.md) describes reviewed JSON and YAML publication, source digests, and retained snapshots.
The [CLI sign-in guide](docs/cli-sign-in.md) describes browser confirmation for local and OIDC accounts and private managed sessions.
The [release guide](docs/releases.md) describes retained catalogs, comparisons, activation, rollback, and frozen run selections.
The [suite guide](docs/suites.md) describes settings, archive, restoration, affected schedules, and retained history.
The [prepared suite guide](docs/prepared-suites.md) describes manifests, isolated discovery, and reviewed image publication.
The [repository source guide](docs/repository-sources.md) describes GitHub registration, private App access, manual sync, and shared suite snapshots.
The [source bundle guide](docs/source-bundles.md) describes Go and Playwright ZIP uploads, trusted runtimes, separate build workers, CLI assembly, and retained snapshots.
Native HTTP runs use durable local workers and retained attempt records.
Private HTTPS targets can use project-scoped, exact-origin CA bundles through `--http-trust-bundles`. See [HTTP trust](docs/http-checks.md#private-certificate-authorities).
The API provides run creation, progress, cancellation, failure reruns, logs, and private evidence access.
The artifact listener uses `http://127.0.0.1:8041` by default.
Keep the artifact directory outside the public UI directory.
Use `--artifact-origin`, `--artifact-addr`, and `--artifact-directory` to change these settings.
Use `--artifact-store-config` for operator-selected S3 or Azure Blob storage. See [Artifact storage](docs/artifact-storage.md).
Select Test runs to start a run, inspect progress, cancel execution, or rerun failed tests.
The detail page shows attempts, filtered logs, events, and private artifact previews or downloads.
The [private trace guide](docs/trace-viewer.md) describes installation-hosted action and snapshot inspection, expiry, and local download fallback.
Other plan areas remain tracked in the acceptance ledger.

## TestStudio self-tests

Run the new Playwright suite through TestStudio itself:

```sh
bash scripts/test-self.sh
```

The command needs Docker, Go, Node, and npm.
It builds an immutable local image and starts disposable TestStudio instances.
Results, steps, screenshots, traces, and logs use the normal run and private evidence contracts.
See [Self-testing](docs/self-testing.md) for registration, retained history, and intentional failure checks.

## Authentication

Hosted installations configure OIDC providers and use server-managed browser sessions.
Use `--oidc-config` with a JSON array of provider definitions for initial registration.
Existing provider records remain unchanged when the server restarts.

Set the initial administrator through explicit issuer and subject identifiers:

```sh
go run ./cmd/teststudio --public-url https://tests.example.com --addr 127.0.0.1:8040 \
  --artifact-origin https://evidence.example.com --artifact-addr 127.0.0.1:8041 \
  --oidc-config config/oidc.example.json \
  --bootstrap-issuer https://identity.example.com --bootstrap-subject SUBJECT
```

Register the callback `/auth/callback/PROVIDER_ID` at the identity provider.
Provider secret references use `env:VARIABLE_NAME` for local configuration.
Use `--oidc-secret-refs` to approve references for API and UI selection.
References declared in the operator-supplied initial provider file are also approved.
Manage configuration, provider-owned groups, and exact mappings through Identity providers.
The [provider guide](docs/providers.md) describes setup, authority, lifecycle changes, and local verification.
The browser receives no provider tokens or client secret.

## Databases

SQLite is the default for a single local service.
Use `--database postgres://USER:PASSWORD@HOST/DATABASE` for PostgreSQL.
Use a protected configuration mechanism for production database credentials.
The permission engine uses a separate `teststudio_permissions` schema in PostgreSQL.
Metadata and permission migrations serialize concurrent startup within the database.

## Local PostgreSQL checks

Run the shared functional checks against a disposable local PostgreSQL container:

```sh
bash scripts/test-postgres.sh
```

The script uses a loopback port, temporary data, generated credentials, and isolated databases.
It removes its own container after success, failure, or interruption.
Pass package names to run a smaller check:

```sh
bash scripts/test-postgres.sh ./internal/storage ./internal/authorization
```

The same feature tests use SQLite when `TESTSTUDIO_TEST_DATABASE` is unset.
Database contract checks reject remote PostgreSQL addresses.
Initial functional checks require no cloud account.

Memberships, assignments, direct grants, user status, and sessions use current database records.
Built-in role overrides invalidate the local engine immediately and reach another engine within five seconds.

## Validation

Run tests for the changed feature:

```sh
go test ./internal/storage -count=1
go test ./adapters/permissions/sqlite -count=1
go test ./internal/catalog -count=1
go test ./adapters/auth/oidc ./internal/sessions -count=1
go test ./internal/httpapi -count=1
```

Build the server with `go build -o bin/teststudio ./cmd/teststudio`.
The user authorizes frequent feature commits. Remote pushes require a separate request.

## Local Go source suites

Run the new isolated Go suites through normal TestStudio publication, execution, timing, and evidence workflows:

```sh
bash scripts/test-go-self.sh
```

See [Local Go suites](docs/go-testing.md) for the manifest, selectors, runtime profiles, diagnostics, and recovery contract.

## Repeating runs

Manage interval and cron schedules from the project Schedules page.
Preview future times, select published tests, and inspect firing history and normal run evidence.
The local server processes schedules by default.
See [Durable schedules](docs/schedules.md) for permissions, release pins, overlap, downtime, and replica behavior.

## Completion notifications

Project channels use scoped secret references and one durable delivery path for manual and scheduled runs.
Open the project Notifications page to configure channels and inspect delivery attempts.
Run details show delivery history for that run. Delivery retries preserve test outcomes.
Run `bash scripts/test-notifications.sh` for a fresh local receiver and built-service verification.
Run `bash scripts/test-self.sh --test=STUDIO-NOTIFY-001` for the focused browser workflow through TestStudio.
See [Durable completion notifications](docs/notifications.md) for configuration, retries, permissions, and API routes.

## Identity administration

User, group, local membership, profile, and session administration APIs enforce stored ownership and current permissions.
Disabling invalidates earlier sessions through later restoration. Ordinary lifecycle actions protect the last usable installation administrator.
Run `bash scripts/test-access.sh` to verify persistence, restart, and explicit loopback recovery with a fresh local installation.
See [Identity administration](docs/access.md) and the [identity OpenAPI contract](contracts/openapi.yaml).
Open Users, Installation groups, or the selected workspace's Project groups to manage authorized identity records.
Run `bash scripts/test-self.sh --test=STUDIO-ACCESS-001` for the focused local browser workflow.
Open Installation access or Project access to manage predefined scoped role assignments.
Enable managed password sign-in with `--local-accounts`, alongside configured OIDC providers if required.
Use User details for authorized credential provisioning and reset. Use Your account for personal password changes.
Run `bash scripts/test-self.sh --test=STUDIO-LOCAL-001` for the managed local account browser workflow.
Project owners add members directly. Identity linking and pending project access remain pending.
Custom role APIs support exact scoped rules, inheritance, disabled state, and managed assignment.
The permission catalog exposes registered descriptions and current grantability. See [Identity administration](docs/access.md).
Open Project roles or Installation roles to edit custom policies, inspect inheritance, and preview proposed access without saving changes.
Run `bash scripts/test-self.sh --test=STUDIO-POLICY-001` for the guarded custom-policy browser workflow.
Installation roles also manages installation-wide predefined role disabling and restoration while retaining assignments and code-managed definitions.
Run `bash scripts/test-self.sh --test=STUDIO-BUILTIN-001` for the predefined role browser workflow.
Open Project permissions or Installation permissions to manage direct user and group allow and deny rules, precise expiry, and reason-based removal.
Run `bash scripts/test-self.sh --test=STUDIO-GRANTS-001` for the direct grant browser workflow.
