# TestStudio implementation plan

Status: Implementation authorized. Track verified delivery in `implementation.md`.
The user authorizes targeted tests and frequent feature commits. Remote pushes remain separate operations.

TestStudio is an independent testing product for teams with multiple projects.
The repository name is `github.com/wtiger001/teststudio`. Private visibility remains the proposal.
The repository contains the Go server, Angular UI, contracts, generic runners, and deployment definitions.
The product name is TestStudio. The repository and command name is `teststudio`.
The UI hostname remains an open decision. The GitHub owner is `wtiger001`.
Consumer tests can remain in their own application repositories or separate suite repositories.
No second product repository is required initially.

### Product independence

Start with an empty project registry and fresh infrastructure configuration.
Copy no existing tests, fixtures, test catalogs, credentials, environments, schedules, reports, or history.
Include only newly written generic examples and product validation tests.
MarketMaker is not a required first project, dependency, deployment, or acceptance target.

Use the current system as a behavioral reference for generic workflows.
Any reused product implementation must have no application-specific dependencies.
Do not depend on MarketMaker modules, shared libraries, authentication, notification transports, or deployment scripts.
Do not inherit application identifiers, permission groups, account identifiers, AWS profiles, hostnames, or sibling checkout paths.
Do not add legacy record adapters, catalog aliases, redirects, compatibility wrappers, or a MarketMaker cutover workflow.
The existing MarketMaker deployment remains a separate system outside this plan.

Applications, suites, endpoints, stages, credentials, permissions, and notification policies are configuration.
Use provider-neutral domain models and service contracts.
Cloud resource identifiers belong only in provider configuration and adapters.

### Deployment targets

Support AWS, Kubernetes, Azure, and a local server as required deployment targets.
Each target runs the complete product, including UI, API, scheduling, results, and private artifacts.
Each target supports isolated suite execution through its execution adapter.
A deployment target hosts the service. An execution location hosts a test worker.
A tested environment identifies the application under test. Keep these three concepts separate.
A remotely hosted service can register a local worker without moving its own deployment.

| Target | Go service and UI | Isolated suite execution | Artifacts and secrets |
| --- | --- | --- | --- |
| AWS | ECS service. Private S3 UI origin behind CloudFront. Route API requests to ECS. | ECS tasks with configured resource limits and identities. | S3 artifacts and Secrets Manager references. |
| Kubernetes | Deployments, Services, and configured ingress or Gateway. Serve static UI through its own container or the Go service. | Kubernetes Jobs with configured service accounts and resource limits. | Configured object storage or persistent volumes. Secret references use the cluster integration. |
| Azure | Container Apps service. Serve UI from the service or a separate static distribution. | Container Apps Jobs started through the execution adapter. | Blob Storage artifacts and Key Vault references. |
| Local server | Native Go service or Docker Compose. Serve the built Angular UI locally. | Local container worker. Registered native execution is an explicit trusted option. | Local persistent directories and local secret bindings. |

Azure Container Apps is the proposed Azure baseline.
It fits the continuous Go API and scheduler without requiring a Functions-specific service architecture.
Use Container Apps Jobs for finite isolated executions.
Azure Functions is not required for the baseline. Evaluate it separately if a later event handler needs it.
Keep at least one scheduler-capable instance active when schedules are enabled.
A static distribution is optional outside the required AWS CloudFront profile.

Define ports for durable metadata, artifacts, secrets, identity, builds, and execution.
Keep provider SDK types outside domain models and shared contracts.
Local, Kubernetes, and Azure deployments require no AWS account, credentials, or services.
The local server supports durable runs, schedules, results, and reports without a cloud connection.
External Git imports and dependency downloads can still require network access.

Use PostgreSQL for hosted and multi-instance deployments.
Use SQLite as the default for development and a single-instance local server.
Local deployments can select PostgreSQL when they need the hosted database configuration.
Both databases support the same catalog, run, schedule, result, and evidence contracts.
Select managed or self-hosted PostgreSQL through deployment configuration.
Artifact interfaces support S3, Azure Blob Storage, and filesystem storage.
Keep private artifact authorization independent of CloudFront signing.
Use authorization checks before issuing a provider-specific download grant or serving a local artifact.
Serve generated HTML reports on a separate origin from the authenticated UI.

Use one scheduler implementation with durable claims and execution leases across all targets.
Replicas can recover abandoned work without starting duplicate logical runs.
Provider retries must reconcile an existing execution before starting another worker.
Application schedules do not require EventBridge, Kubernetes CronJobs, or Azure timer functions.
See [Durable schedules](schedules.md) for the implemented local scheduler, project interface, and atomic firing contract.
Configure concurrency by project, execution location, and workload.
Define environment exclusivity explicitly. Frequent health probes must not inherit a global browser-run lock.

### Required behavior

Provide these generic workflows:

- Manual execution, scheduled execution, cancellation, deadlines, and failed-test reruns.
- Durable run records, configurable execution limits, exclusive-run locks, and idempotent requests.
- In-process Go scheduling. Do not introduce EventBridge.
- Immutable suite releases and run snapshots.
- Suite publication without a testing API deployment.
- Progress polling every five seconds, provisional counts, and runner heartbeat reporting.
- Private artifacts, screenshots, result details, and access checks on every deployment target.
- Full-width result cards under collapsible categories.
- Application, environment, stage, category, test, and date filters.
- Stacked area charts for per-run result states, with visible gaps for missing history and explicit incomplete results.
- Most failed and most volatile test rankings.
- TestStudio-owned Playwright self-tests that run through its normal catalog, release, execution, timing, and evidence workflows.
- Durable completion notifications with retry handling.
- Production safety labels, request guards, and analytics suppression.

Safety labels remain `safe`, `unsafe`, and `needs-review`.
Safety labels do not require separate approval records or renewal steps.
Protected targets accept only tests labeled `safe`, including their dependencies.
Project-specific request guards continue to enforce the declared operating conditions.

### Domain model

Use the following ownership structure:

```text
Project
  Applications
    Environments
  Suites
    Catalog drafts and test revisions
    Releases
      Test catalog
      Runner configuration
  Test sources
  Builds
  Runs
    Attempts and steps
    Events and log streams
    Evidence manifests and artifacts
  Schedules
  Notification settings
```

A project groups related applications, suites, history, and settings.
An application has an arbitrary permanent identifier and a display name.
An environment belongs to an application and has a permanent identifier, display name, endpoints, stage, and protection setting.
Display names such as Production can repeat across applications.
A suite belongs to a project and declares the applications it supports.
A test source belongs to a project and supplies uploaded content, a registered repository, or UI-managed definitions.
One repository can supply multiple suites through separate module roots, test paths, and manifests.
A build records source preparation and artifact validation. It is separate from a test execution.
A release identifies its catalog, source revision, adapter, execution configuration, and immutable image or bundle.
A run freezes its project, suite release, test selection, environment, endpoints, and execution configuration.
Later configuration edits do not change that snapshot.

Separate test type from runner in the new contracts.
Type describes the check: browser, HTTPS health, API, unit, integration, accessibility, visual, or performance.
Runner describes its implementation: Playwright, Go, or native HTTP.
A Go API test therefore has type `api` and runner `go`.
An execution location is separate from both fields. It identifies an ECS, Kubernetes, Azure, or local worker.

Suites declare whether they require a target environment.
Pure unit suites require no web URL or API credentials.
Their runs appear under All and an explicit No target filter, rather than a production stage.
API and integration suites declare their required endpoint and credential bindings.

Test identity includes project, suite, and permanent test identifier.
Browser or framework variants remain separate result dimensions.
Two projects can use the same local test identifier without sharing history.
Test identifiers remain stable across releases when they represent the same check.

Stages include production, beta, development, local, and custom values.
Protection is independent of stage.
A development application that reads production data can require production-safe tests without sending production-stage notifications.
Environment records store explicit stages. Do not infer stages from their names.

### Users, groups, and permissions

Use `github.com/wtiger001/go-permissions` as the authorization engine.
Pin a reviewed library version or commit and verify its Go toolchain requirements before implementation.
Use `permissions.Service` for permission checks, role management, assignments, grants, and effective-permission reports.
Do not create a second authorization evaluator in TestStudio.
The library handles authorization. TestStudio handles identities, group membership, sessions, and OIDC sign-in.

#### Users and groups

Store permanent internal user identifiers independently of email addresses and identity providers.
An external identity uses the exact OIDC issuer and subject pair as its unique key.
A user can link multiple external identities through an authenticated, explicit account-linking workflow.
Require fresh authentication and proof of control of the identity being linked.
Do not link accounts automatically because their email addresses match.
Store display name, email verification state, lifecycle state, creation time, and last sign-in time.
Disabling a user blocks new requests and sessions while preserving ownership, run history, and audit records.

Use direct project membership instead of invitation messages, acceptance links, or invitation expiry.
Project owners with membership authority add existing users and choose permitted project roles.
Support pending project access for users who have not signed in.
A pending access record stores an allowed provider, verified email requirement, project, and permitted role.
Apply pending access only after that provider confirms the verified email during sign-in.
Do not use pending access to link existing internal accounts or grant installation administration.
Audit pending access creation, cancellation, activation, and the resulting role assignments.
Support configurable user provisioning at first OIDC sign-in.
Successful sign-in alone grants no project access.
Do not grant installation administration from an email domain or first-login order.

Groups have permanent identifiers, names, descriptions, ownership scope, lifecycle state, and membership provenance.
Support local groups and groups populated by explicit OIDC claim mappings.
A user can belong to multiple groups. A group can receive different role assignments in different authorized projects.
Use flat user-to-group membership initially. Nested directory groups require an explicit resolver with cycle and expansion limits.
Local membership edits and external synchronization remain distinguishable in the UI and audit history.
Externally managed membership is read-only in ordinary local membership editors.
Archiving a group removes its effective access without erasing past assignments or audit records.

Installation administrators manage installation-owned users and groups.
Project administrators manage only project-owned groups and project assignments they can delegate.
Project-owned groups cannot receive assignments outside their project.
Membership changes to shared installation groups require installation-level authority.

Implement the library's `IdentityProvider` interface through TestStudio's identity repositories.
Resolve `GetUserGroups`, `GetGroupMembers`, and `IsUserInGroup` from authoritative active membership records.
Implement `GetUserTeams` and `IsUserInTeam` for the pinned interface without inventing implicit project membership.
Initially return no team membership unless a team integration is explicitly configured.
Use user and group principals for initial role assignment workflows.
The permission library does not own or synchronize the user directory.

#### Permission scopes and evaluation

Use installation scope for user administration, provider configuration, global workers, and product settings.
Map project identifiers to the library's `Request.TeamID` for project checks.
Map a resource to a canonical object identifier containing project, resource type, and permanent resource identifier.
Load the resource's authoritative project before constructing an authorization request.
Do not trust a browser-supplied project identifier to establish resource ownership.
Object-scoped grants are explicit. Parent access does not imply descendant access without a documented grant.

Register stable permission identifiers in the `teststudio` namespace.
Register predefined role identifiers with the required `builtin.` prefix.
Project role grants use explicit project bindings rather than wildcard project scopes.
For reusable project roles, validate `?team` binding resolution through the pinned library's grant and assignment contracts.
Keep missing bindings and unauthorized wildcard scopes as errors.
Custom roles contain only permitted registered actions and validated installation, project, or object scopes.

| Area | Proposed permission identifiers |
| --- | --- |
| Catalog | `teststudio.catalog.{read,create,update,import,export,publish,archive,restore,delete}`. |
| Execution | `teststudio.runs.{read,create,cancel,execute-protected}`. |
| Evidence | `teststudio.evidence.{read,download,retain,delete}`. |
| Schedules | `teststudio.schedules.{read,create,update,delete}`. |
| Project configuration | Separate read and manage permissions for environments, sources, credential bindings, notifications, and project settings. |
| Access management | `teststudio.members.{read,manage}`, `teststudio.roles.{read,manage}`, and `teststudio.permissions.grant`. |
| Installation | Separate manage permissions for users, installation groups, OIDC providers, global workers, and system settings. |

Braces expand into separate permission identifiers. Use full identifiers in contracts and stored grants.
Credential-binding permissions expose references and configuration, not resolved secret values.
Protected execution requires both ordinary run creation and the protected-execution permission.
It also requires the suite safety policy. No role bypasses safety labels or request guards.

| Predefined role | Default responsibilities |
| --- | --- |
| `builtin.teststudio.viewer` | Read authorized catalogs, results, trends, and evidence. |
| `builtin.teststudio.operator` | View, start and cancel runs, and manage ordinary schedules. Protected execution requires a separate grant. |
| `builtin.teststudio.editor` | Edit catalog drafts and declarative definitions. Publication requires a separate grant. |
| `builtin.teststudio.publisher` | Publish and activate validated releases and manage registered test sources. |
| `builtin.teststudio.project-admin` | Manage one project's settings, members, groups, and permitted role assignments. |
| `builtin.teststudio.installation-admin` | Manage installation identity, providers, workers, and explicitly configured installation-wide grants. |

Built-in definitions remain code-managed. Custom roles, assignments, grants, and disabled-role overrides are persistent.
Do not give `builtin.public` or `builtin.authenticated` project permissions by default.
Use a dedicated internal administrator group for any configured synthetic administrator role.
Do not bind an arbitrary external group claim directly to synthetic administration.
Retain the library's explicit-deny precedence and expired-grant behavior.
Fail closed on permission-store or identity-resolution errors.
Use authentication middleware to identify callers, then check `HasPermission` inside handlers or domain services after parsing resource context.
Apply checks to lists, searches, pagination, counts, charts, exports, downloads, and mutations.
Filter unauthorized projects before computing aggregate results.
The UI displays server-computed capabilities; it does not infer authority from an OIDC group name.

Granting roles requires explicit delegation authority for the target project, role, and scope.
Project administrators cannot grant installation roles, widen bindings, or change shared groups to elevate their own access.
Validate role inheritance and scoped bindings before enabling custom inheritance workflows.
Permission checks remain authoritative even when the UI hides an action.

Refresh authorization after membership, assignment, grant, or user-state changes.
Invalidate local caches immediately and propagate changes across instances within a documented maximum of five seconds.
Verify library caches and product caches together. Do not rely on a new OIDC login to remove local access.
User disabling and session revocation take effect on the next authenticated request.
Existing short-lived artifact grants expire within their configured lifetime.
Schedules record an execution principal and recheck its authority before dispatch.
Revocation blocks future starts without rewriting past run snapshots.

#### Access management API and UI

Expose identity and access management through the same versioned API used by the UI.
Use `/api/v1/me` for the current profile and linked identities.
Expose effective capabilities and permission explanations for the current authorized project and resource context.
Restrict inspection of another user's effective access to authorized access managers.

| Route family | Purpose |
| --- | --- |
| `/api/v1/users` | List, create, inspect, disable, and restore users with installation authority. |
| `/api/v1/groups` | Manage permitted groups and their local memberships. Show synchronized membership separately. |
| `/api/v1/roles` | List predefined roles and manage authorized custom roles. |
| `/api/v1/projects/{projectId}/members` | Inspect project access and manage scoped member or group assignments. |
| `/api/v1/projects/{projectId}/role-assignments` | Assign and revoke permitted roles with validated project and object bindings. |
| `/api/v1/projects/{projectId}/permissions` | Inspect effective grants and manage permitted explicit grants or denies. |
| `/api/v1/identity-providers` | Configure OIDC providers and explicit external-group mappings with installation authority. |

Document single-assignment revocation and grant removal operations in OpenAPI.
Use revision checks and audit records for access changes, including actor, subject, scope, prior value, new value, and time.
Prevent an ordinary UI action from removing the last usable installation administrator.
Provide an operator-controlled recovery procedure independent of a failed provider or group mapping.

Add Users, Groups, Roles and Permissions, and Sign-in Providers to authorized Settings pages.
See [Identity administration](access.md) for implemented user, group, membership, profile, and session APIs, revision rules, and local recovery.
User details show linked identities, local and external groups, project assignments, effective permissions, and active sessions.
Group details show membership origin, affected projects, and role assignments.
Role editors show action descriptions, scope bindings, explicit denies, and an effective-access preview.
Show the reason for an access decision, including its role or group source, to authorized managers.
Return distinct unauthenticated and unauthorized responses without exposing another project's private data.

### OIDC sign-in and sessions

Support one or more explicitly configured OpenID Connect providers on every deployment profile.
Use provider discovery from an allowed issuer and support configurable scopes and claim mappings.
Configure each provider's issuer, client identifier, server-side credential reference when required, callback, and logout settings.
Providers can include enterprise or self-hosted installations. No provider or tenant is hardcoded.
A provider outage does not require changing catalog or permission records.

Use the authorization code flow with PKCE S256, state, and nonce validation.
The Go service handles the callback and validates issuer, audience, signature, expiry, nonce, and token-use requirements.
Use verified discovery metadata and signing keys with bounded refresh and key-rotation handling.
Validate the provider and callback binding before exchanging an authorization code.
Do not accept an ID token as a general API access token.
Keep provider secrets, refresh tokens, and access tokens on the server.
The browser receives an opaque session cookie rather than provider tokens in browser storage.

Use HttpOnly cookies, Secure on HTTPS deployments, and SameSite settings appropriate for the callback flow.
Protect cookie-authenticated mutations against cross-site request forgery.
Rotate the session after sign-in and privilege changes.
Store session expiry, revocation, authentication provider, and user status durably in the selected database.
Support idle and absolute expiry, current-session logout, and administrator session revocation.
Clear the local session even when provider logout fails. Provider logout is an explicit optional integration.
Keep callback redirects and post-login return routes within configured destinations.

Identity-provider sign-in establishes the internal user identity, not its project roles.
Apply configured external-group mappings only from validated claims for that issuer.
Use stable provider group identifiers when available and record mapping provenance.
An unmapped external group grants no access. External role strings are not executable permission names.
Distinguish complete membership claims from missing, partial, or provider-overflow group responses.
Use an explicit provider integration for additional membership retrieval when required.
Never preserve outdated elevated external membership as a silent fallback after an incomplete refresh.
Recheck authoritative external membership on a configured refresh interval and impose a maximum membership age.
If required membership cannot be refreshed, deny dependent access or require sign-in again.
Local group membership remains independent of provider-managed membership.

Expose `/auth/providers`, `/auth/login/{providerId}`, `/auth/callback/{providerId}`, `/auth/session`, and `/auth/logout`.
Publish only provider display information and enabled login choices to unauthenticated clients.
The sign-in screen shows configured providers and readable cancellation, failure, and expired-session states.
Preserve a permitted return route and show Access not granted when authentication succeeds without project access.
Use the same login and permission contracts for AWS, Kubernetes, Azure, and local hosted deployments.
Static UI hosting routes authentication and session API requests to the Go service without caching private responses.

Provision the initial administrator through an operator-controlled bootstrap bound to an exact issuer and subject.
An explicit loopback-only development mode can create a local principal without an external provider.
That mode still uses the permission engine and configured local groups. It never silently activates for remote clients.
Shared or remotely accessible local servers use configured authentication.
Automation clients use revocable, expiring project-scoped credentials. Workers retain separately scoped run-ingestion credentials.
Neither credential type reuses a browser cookie or receives installation administration by default.

### Database and durable storage

PostgreSQL stores catalog revisions, releases, run snapshots, attempts, steps, schedule claims, artifact metadata, and audit events.
It also stores users, linked identities, memberships, sessions, custom roles, assignments, grants, and provider configuration references.
SQLite stores the same logical records for development and single-instance local operation.
Keep database access behind focused repositories with shared domain rules.
Use versioned migrations for each database dialect.
Use the library's PostgreSQL permission store within a product-owned schema and a controlled migration lifecycle.
Implement a SQLite adapter for the pinned `permissions.PermissionStore` interface.
The reviewed library provides PostgreSQL storage; this plan does not assume a built-in SQLite backend.
Both stores use the same `permissions.Service` evaluator and TestStudio identity adapter.
Persist role inheritance, assignment bindings, grants, expiry, and disabled built-in overrides on both databases.
SQLite mode does not fall back to permissive checks or lose assignments on restart.
Add shared store checks for explicit denies, expiration, revocation, scope binding, and built-in role overrides.
Do not require PostgreSQL extensions for the core product.

Use SQLite WAL mode on a local persistent disk, with bounded transactions and a configured busy timeout.
Support one local service instance. Workers communicate through the service API rather than opening its database file.
Use PostgreSQL for multiple API or scheduler instances.
Do not place a SQLite WAL database on a shared network filesystem.
Verify revision conflicts, idempotency, execution claims, lease recovery, and retention behavior against both databases.
Use atomic conditional updates for claims. Keep provider-specific locking outside domain services.

Store screenshots, traces, videos, reports, attachments, and log segment bodies in configured artifact storage.
Store their ownership, content type, size, checksum, status, capture policy, and storage reference in the database.
Logs use immutable ordered segments with a searchable metadata index and cursor-based reads.
Do not put large binaries or unrestricted log bodies in relational rows.
The local profile stores artifacts under its persistent data directory.
PostgreSQL operation does not require object storage when a filesystem adapter meets the deployment requirements.

Back up database state and artifact manifests with documented consistency rules.
Test restoration with artifact checksums and missing-file detection.
Provide an explicit export and import path from SQLite to PostgreSQL when the installation grows.
Preserve identifiers, revisions, run snapshots, audit history, and artifact references during that transfer.
Changing the database connection alone does not move existing data.

### Catalog management through the API and UI

The service owns the test catalog and exposes all catalog workflows through its versioned API.
The Angular UI uses that same API. CLI publishers and integrations use the same contracts and permissions.
A project has a searchable catalog across its suites.
A suite owns permanent test identities, catalog drafts, published revisions, and release membership.
Use the project, suite, and test identifier together for identity.

Each test records these fields:

- Permanent identifier, display title, description, application, suite, category, tags, and optional owner.
- Test type, runner, source origin, source revision, and executable selector or declarative definition.
- Required endpoint and credential bindings, dependencies, timeout, and registered execution profile.
- Safety label, rationale, operating conditions, lifecycle state, and capture policy.
- Revision, creation and update timestamps, actor, and optional external reference links.

Treat latest outcomes, failure rates, and artifact counts as derived observations, not editable catalog metadata.
Keep catalog revision separate from the latest execution result.

#### Editing and publication

Create and edit catalog entries in a draft.
Declarative HTTPS and API checks use structured editors and API definitions.
Code-backed tests reference discovered selectors from a pinned source snapshot or immutable artifact.
Creating a code-backed catalog entry does not create executable test code.
Use the source workflow to add or change its implementation.

Keep discovered source facts separate from service-managed metadata.
Record the authority for each editable field. Show source-owned fields as read-only with a source link.
Store service-managed titles, categories, tags, owners, and descriptions as versioned metadata.
A source synchronization previews additions, removals, and changes before applying the configured publication policy.
Do not silently overwrite service-managed metadata during discovery.
A safety change requires a rationale and publishing permission. It does not add a separate approval workflow.
Reassess safety when source, helpers, dependencies, or request policy changes.

Validate identifiers, selector mappings, definitions, dependency existence, dependency cycles, required bindings, and permitted execution profiles.
Code-backed publication verifies that referenced tests exist in the prepared artifact.
Return field-specific errors and a release comparison before publication.
Use revision tokens and If-Match for edits and bulk mutations.
Reject stale edits with a conflict response rather than losing another user's changes.

Publishing freezes the catalog revision and its execution artifact in an immutable release.
Activation selects a ready release. Failed validation leaves the active release unchanged.
Metadata edits do not silently change a published release or a historical run.
New runs record the exact release, catalog revision, selected identifiers, definitions, and effective capture policy.

Support draft, enabled, disabled, and archived states.
Archive removes a test from normal selection while preserving its identity, revisions, history, and evidence.
Restore preserves the identifier. Duplicating a test creates a new identifier and independent history.
Disabled or archived tests cannot start new attempts, including through an old release or queued request.
Recheck admission immediately before dispatch. Report blocked selections and affected schedules explicitly.
An already started attempt follows the cancellation policy; a catalog edit does not silently terminate it.
Permanent deletion is a separate administrator action and cannot remove records required by retained runs or releases.

Saved selections can contain explicit identifiers or a versioned filter expression.
Preview the resolved selection and freeze its identifiers when creating a run.
Show removed or disabled dependencies as configuration errors instead of silently omitting them.
Bulk edits preview affected records and use atomic changes within the documented request limit.
Return explicit validation failures. Do not report partially applied edits as complete success.
Record actor, revision, timestamp, and changes for create, edit, import, publish, disable, archive, and restore operations.

#### Catalog API

Use `/api/v1/projects/{projectId}` as the project route prefix.
Use `/suites/{suiteId}` below that prefix for suite-owned operations.
The exact schemas and operation identifiers belong in OpenAPI.

| Route below the project prefix | Operations |
| --- | --- |
| `/tests` | Search and paginate the authorized project catalog across suites. |
| `/suites/{suiteId}/tests` | List or create suite test identities and draft entries. |
| `/suites/{suiteId}/tests/{testId}` | Read details, patch an editable draft with If-Match, and inspect history links. |
| `/suites/{suiteId}/tests/{testId}/revisions` | Read retained revisions and compare definitions or metadata. |
| `/suites/{suiteId}/tests/{testId}/actions/{action}` | Disable, enable, archive, restore, or duplicate with revision checks. |
| `/suites/{suiteId}/catalog-drafts` | Create, inspect, and revise a catalog draft. |
| `/suites/{suiteId}/catalog-drafts/{draftId}/validate` | Validate the complete draft and preview resolved execution bindings. |
| `/suites/{suiteId}/catalog-drafts/{draftId}/publish` | Create an immutable release with an idempotency key. |
| `/suites/{suiteId}/catalog-imports` | Validate and preview an import, then apply an explicit import plan. |
| `/suites/{suiteId}/catalog-exports` | Export a versioned catalog definition and metadata without secret values. |

Expose bulk edits and saved selections through documented operations on the same API.
Use deterministic sorting and opaque cursors for catalog lists.
Filters include text, suite, application, category, tag, type, runner, lifecycle state, safety label, and source.
Text search includes identifier, title, and description.
Allow optional latest-outcome filters without treating tests with no runs as passing.
Separate viewing, editing, publishing, running, source management, and destructive administration permissions.
Apply project authorization to every detail, revision, import, export, and history operation.

#### Catalog UI

Provide table and category-tree views with search, filters, sorting, saved selections, and bulk actions.
Show draft changes and the active release separately.
Test detail shows metadata, definition or source, revisions, dependencies, safety, latest result, history, and evidence links.
Provide Create test, Edit, Duplicate, Disable, Archive, Restore, Import, Export, Validate, and Publish actions according to permissions.
Show field validation beside its control and stale-edit conflicts with reload or review choices.
The publication screen compares additions, edits, removals, safety changes, and affected schedules.
Keep code source editing in the registered source workflow. The catalog UI manages metadata and declarative definitions.

### Repository structure

Keep the generic service and its dedicated UI in one repository.
Consumer test suites remain outside the product except for newly written generic examples.

```text
teststudio/
  cmd/
    teststudio/
  internal/
    domain/
    identity/
    authorization/
    sessions/
    projects/
    applications/
    environments/
    suites/
    catalog/
    evidence/
    sources/
    builds/
    healthchecks/
    runs/
    schedules/
    results/
    analytics/
    notifications/
    httpapi/
  adapters/
    auth/oidc/
    permissions/postgres/
    permissions/sqlite/
    sources/github/
    builds/
    execution/ecs/
    execution/kubernetes/
    execution/azure/
    execution/local/
    storage/postgres/
    storage/sqlite/
    artifacts/s3/
    artifacts/azure/
    artifacts/filesystem/
    secrets/aws/
    secrets/azure/
    secrets/kubernetes/
    secrets/local/
    notifications/discord/
  web/
    src/app/
      shell/
      core/auth/
      core/api/
      features/projects/
      features/applications/
      features/environments/
      features/suites/
      features/catalog/
      features/sources/
      features/runs/
      features/schedules/
      features/trends/
      features/settings/
      features/users/
      features/groups/
      features/permissions/
  contracts/
    openapi.yaml
    catalog.schema.json
    execution.schema.json
    progress.schema.json
    results.schema.json
    https-check.schema.json
    api-check.schema.json
    go-suite.schema.json
    suite.schema.json
    source.schema.json
    catalog-draft.schema.json
    run-event.schema.json
    artifact.schema.json
    evidence.schema.json
    identity.schema.json
    authorization.schema.json
    oidc-provider.schema.json
  packages/
    runner-sdk/
      typescript/
      go/
  runners/
    playwright/
    go/
    http/
  examples/
    sample-app/
    sample-suite/
  infrastructure/
    aws/
    kubernetes/
    azure/
    local/
  scripts/
    publish-suite/
    deploy/
    validate-deployment/
  docs/
```

Use standalone Angular components, small services, and pure utilities.
Keep product UI primitives and notification implementations inside this repository.
Do not create a shared library that requires changes to an unrelated application.

### Execution and publication contracts

Define versioned catalog, execution, progress, result, and artifact contracts before implementing adapters.
The service accepts project, suite release, optional environment, selection, and execution location identifiers.
Suites declare required bindings. Targetless suites omit the environment instead of using a placeholder.
Provider-specific task details belong to registered execution configuration.
The server does not accept arbitrary shell commands from the browser.

Plan adapters for Playwright browser tests, native HTTPS checks, Go suites, and REST API suites.
Their adapters send the same progress and final result records used by other adapters.
Reserve further adapter identifiers for accessibility, visual, and performance suites.
These additional adapters do not require empty implementations in the first release.
Generated HTML reports remain optional artifacts. Stored results remain the source for trends.

Use immutable container publication for coded suites initially.
Publish portable OCI images through configured registries. Record supported operating systems and processor architectures.
Register and activate releases per project and suite.
Validate release ownership, image digests, permitted execution configuration, and catalog references.
The server executes published releases. It does not clone Git repositories during test execution.
Record source repository and revision metadata for review and traceability.

Container releases require an immutable image reference.
Native HTTPS releases reference versioned check definitions and require no browser image or provider-specific task definition.

A later publication improvement can separate test bundles from pinned browser runtime images.
That work needs dependency verification and digest checks before activation.
It is not required for the initial product release.

### Run details and evidence

A run captures execution context and diagnostic evidence independently of optional HTML reports.
Expose the same details through authenticated API responses and the dedicated UI.
Results remain readable when report generation or an individual artifact upload fails.

Record project, suite, release, catalog revision, source commit, selected tests, trigger, actor, and parent rerun.
Record the resolved environment, endpoint snapshot, execution location, image digest, and toolchain or browser versions.
Record capture policy, worker identity, operating system, architecture, start and end times, exit status, and cancellation reason.
Store credential references and binding names without storing resolved secret values in the run snapshot.
Distinguish queue, preparation, execution, reporting, and artifact upload durations.
Use timestamps from the service and worker with clear clock origin. Do not imply a precise global ordering across workers.

#### Predictive timing and regression criteria

Time every test attempt and retain total execution time across retries.
Keep functional outcomes separate from duration regression assessments.
Compare only matching tests, environments, execution locations, runtime profiles, definitions, and target configurations.
Use clean first-attempt passes to estimate typical durations and guide bounded adaptive timeouts.
Keep slow, failed, retried, and interrupted observations outside the baseline.
Require enough comparable observations before enforcing slowdown rules.
Default to a failure at twice the median, with an absolute noise guard and a historical tail guard.
Support configurable warning, failure, and disabled modes, sample windows, history age, and timeout controls.
Freeze predictions and thresholds when each run enters the queue.
Show pre-run estimates, baseline readiness, attempt durations, slowdown ratios, and timeout budgets through the API and UI.
Make incomplete predictions explicit. Keep queue, preparation, and capture time separate.
Support audited baseline resets with revisions and generation guards for queued work.
See [Test timing](timing.md) for the initial native HTTP policy and contracts.

#### Attempts, steps, and capture

Each test result has its own attempts, assertions, steps, timing, errors, and artifact links.
Timeouts use a separate `timed-out` status and `timedOut` count across runners, API responses, and the UI.
Rerun failures includes failed, timed-out, and flaky tests. Timeout evidence retains unsuccessful-attempt capture and retention rules.
The run deadline preserves a known active-test timeout when the worker can save its result.
See [Test results](results.md) for status, aggregation, compatibility, and rerun behavior.
An explicit retry creates a separate attempt. A passing retry remains flaky.
Keep setup, teardown, worker, and build diagnostics identifiable even when no test starts.
Run-level output remains run-level when the adapter cannot attribute it reliably to a test.

| Evidence | Required behavior |
| --- | --- |
| Structured events | Record lifecycle transitions, worker heartbeats, step boundaries, retries, cancellation, and infrastructure failures. |
| Process logs | Capture timestamped stdout and stderr, severity when available, worker, stream, attempt, and ordered sequence. |
| Browser diagnostics | Capture console errors, page errors, failed requests, and bounded redacted request metadata. |
| Screenshots | Capture failed browser attempts and explicit test attachments. Allow configured checkpoint images and masking. |
| Traces | Retain Playwright traces for failed attempts by default. Collect them from the first attempt, even when retries are disabled. |
| Video | Support opt-in recording with always or failure-only retention. Keep recording disabled by default. |
| Assertions and API details | Capture expected and actual values, stack traces, bounded redacted request and response excerpts, and assertion timing. |
| Other attachments | Support typed attachments, optional coverage files, generated reports, and adapter-specific diagnostics. |

Default to logs for every run, screenshots on browser failure, retained failure traces, and no video.
Allow project defaults and suite overrides with optional test-specific capture settings.
Explicit run overrides require permission and remain within project capture and size limits.
Freeze the effective policy in the run snapshot.
Where supported, capture on assertion failure before teardown removes useful state.
Record unavailable evidence explicitly when a browser crashes or a worker terminates before capture.

Do not record authorization headers, cookies, secret values, or complete response bodies by default.
Redact configured sensitive fields in logs and structured diagnostics before publication.
Use configured screenshot masks and capture exclusions where binary evidence can contain sensitive application data.
Apply capture restrictions before collecting an attachment. Do not assume text redaction sanitizes an image, video, or trace.

#### Collection and publication protocol

Workers send structured events, log segments, and artifact manifests through versioned ingestion contracts.
Use credentials scoped to the assigned run, worker, permitted operations, and expiry.
A worker cannot attach evidence to another project, run, or attempt.
Uploads use the artifact adapter or a short-lived scoped upload grant.
Completion verifies expected ownership, size, content type, and checksum before making an artifact available.
Publish no arbitrary worker-supplied storage path or external URL as a trusted artifact reference.

Identify events and log segments by worker session, stream, and sequence for idempotent ingestion.
Preserve ordering within a stream. Expose missing sequence ranges and truncated output explicitly.
Send logs periodically during execution rather than only when a run finishes.
Use five-second flush and UI polling intervals initially, with configurable bounded batches.
The log API supports cursor-based reads, filtering by attempt and stream, and resumable downloads.
Keep acknowledged segments durable across service restarts.
A future streaming transport can reuse the event and cursor contracts.

Flush final events and upload available artifacts on success, failure, timeout, and cancellation when the worker can still execute cleanup.
Reconcile available evidence after unexpected worker loss.
Use separate states for test outcome, run execution, and evidence completeness.
Artifact failures do not convert an observed assertion failure or pass into a different assertion result.
Missing required result records keep execution incomplete. Missing optional attachments mark evidence incomplete.
Show missing or failed captures even when the test result is available.
Artifact upload retries do not rerun tests.

An artifact manifest records identifier, kind, filename, content type, bytes, checksum, capture time, attempt, step, and storage reference.
States include pending, available, upload-failed, missing, expired, and deleted.
Only available artifacts receive view or download grants.
The manifest also records capture omissions, redaction policy version, and size or retention limits reached.

#### Evidence API and UI

Use the project route prefix and project-authorized run routes.
Expose `/runs/{runId}` for run details.
Expose attempts, events, logs, and artifacts at `/runs/{runId}/{collection}`.
Name the collections `attempts`, `events`, `logs`, and `artifacts` in OpenAPI.
Use `/runs/{runId}/artifacts/{artifactId}/access` to obtain a short-lived view or download grant.
Document worker ingestion separately from user-facing read APIs.
The UI and external API clients have access to the same result and evidence metadata.

Run detail provides Summary, Tests, Timeline, Logs, and Artifacts views.
Test cards link directly to their attempts, failure screenshots, trace, related logs, and attachments.
Logs support live following, pause, text search, severity filters, stream filters, and download.
Provide screenshot thumbnails, image previews, video playback, and attachment metadata.
Use an installation-hosted trace viewer on the isolated artifact origin, with an authenticated download fallback.
Trace viewing must not require making evidence public or sending it to an external service.
Filter large artifact lists by test, attempt, kind, and state.
Show capture progress, failed uploads, missing evidence, and expired artifacts as distinct states.

#### Retention and access

Set project storage quotas, per-file and per-run limits, and separate retention periods for logs, images, traces, videos, and reports.
Allow longer retention for failed or flaky attempts and explicit retention pins for investigated runs.
Keep compact results and trends after large evidence files expire.
The UI and API show expiry times and retained metadata rather than broken download links.
Release pinning preserves execution artifacts; evidence retention pins preserve captured run artifacts.
Do not confuse these independent retention policies.

Authorize every artifact access request against its project and run.
Keep generated HTML and active attachment formats on an isolated origin.
Record access grants and administrative deletion actions in the audit history.
Retention cleanup respects pins and uses retryable storage deletion with explicit database state.
Report storage usage and quota failures without hiding the test outcome.

### TestStudio self-testing

Maintain a new Playwright suite in this repository. TestStudio runs this suite through its normal APIs and workers.
Use permanent test identifiers and immutable releases. Preserve comparable timing history when the same installation runs a newer release.
Cover catalog workflows, execution, evidence, timing controls, themes, and narrow layouts.
Extend the suite as new product areas become available.
Use separate intentional failures, timeouts, and cancellation checks to verify the runner contract.
A failed worker or missing report must never become a passing test.

Start with an explicitly enabled local container profile and disposable TestStudio instances.
Each test owns its database and target process. Do not reuse MarketMaker tests, resources, or data.
Keep self-test registration explicit. A normal installation starts with empty registries.
The first profile accepts targetless images with internal fixtures and no external network access.
Cloud and external-target profiles remain separate required work.
Keep stored attempts, steps, screenshots, traces, logs, and capture gaps visible through the standard result and evidence views.
See [Self-testing](self-testing.md) for local execution and retained verification output.

### Uploads, Git sources, and test management

Support uploads and registered Git repositories through the same release lifecycle.
Use Git for routine code maintenance and uploads for direct imports or source bundles.
Tests can live in an application repository or a separate test repository.
The service does not require every application to create a dedicated test repository.
Provide CLI publication as another way to register a release.

The common flow is:

```text
Upload files or sync a registered repository
  -> store an immutable source snapshot
  -> validate definitions and prepare the execution artifact
  -> discover and validate the test catalog
  -> create a ready suite release
  -> activate or pin the release according to suite policy
  -> execute manual and scheduled runs
```

Ready means the release is executable and its contracts are valid.
It does not mean that its tests passed against every environment.
Source synchronization and build validation do not silently execute production tests.
An optional run-after-publication setting must name its environment and selection explicitly.

#### Upload options

| Upload | Contents | Processing |
| --- | --- | --- |
| Declarative definitions | HTTPS checks or supported REST definitions in JSON or YAML. | Validate the schema and create a versioned native release. No container build is needed. |
| Source bundle | A ZIP containing the suite manifest, tests, helpers, policy, and dependency files. | Prepare an isolated build and discover its catalog. |
| Prebuilt release | A permitted immutable image or bundle reference plus catalog and provenance metadata. | Verify ownership, digests, runner compatibility, and contracts before registration. |

The upload wizard selects the destination project and suite.
It previews discovered identifiers, categories, types, safety labels, and validation errors.
New suites can be created during import.
Updating an existing suite creates a release rather than overwriting its current files.

A source bundle must include the dependencies needed to interpret its tests.
Playwright bundles include their package manifest, lockfile, fixtures, and required helpers.
Go bundles include the required module context, dependency metadata, and test catalog mapping.
A single source file can use an existing registered runtime only when its required context is explicit.
Do not assume an uploaded `.spec.ts` or `_test.go` file is self-contained.

Upload through a short-lived, project-scoped staging location.
The service computes and records the stored source digest.
Validate archive paths, entry types, file counts, and size limits before extraction.
Reject entries that escape the suite context.
Failed or abandoned staging uploads expire according to configured retention.
Published source snapshots remain immutable while retained releases reference them.

#### Registered Git repositories

Start with GitHub, including private repositories.
Keep the source provider interface independent of GitHub for later Git provider support.
Prefer a GitHub App with access limited to selected repositories and required read permissions.
Store connection credentials as server-side secret references.
Do not place personal access tokens in repository URLs, browser configuration, or build logs.

Repository registration records:

- Project and source display name.
- Provider connection and repository identity.
- Watched branch or tag, or an explicitly pinned commit.
- Suite paths, module roots, manifest locations, and permitted build profiles.
- Manual sync, push-triggered sync, or periodic sync policy.
- Automatic activation policy and any environment or schedule release pins.

Manual Sync now resolves the requested reference to a commit and stores that exact source snapshot.
Push synchronization validates the webhook signature and repository identity before queuing work.
Deduplicate preparation by source, resolved commit, manifest, and build profile.
Duplicate webhook deliveries must not produce duplicate builds or test executions.
Periodic synchronization uses the service scheduler when webhooks are unavailable.
GitHub Actions is not required.

Always execute a published release, not a branch checkout fetched during each run.
Record both the human-readable reference and resolved commit with the release.
Branch updates, tag changes, force pushes, and repository outages do not alter existing run snapshots.
A monorepo can register multiple suites without copying them into another repository.
Changes to shared fixtures, dependencies, or manifests must trigger preparation for affected suites.

#### Suite manifest

Use a small, versioned manifest with runner-specific configuration.
It identifies suite roots, catalog metadata, execution profiles, required bindings, and request policy.
Keep permanent test identifiers in source metadata or an explicit mapping file.
The service validates source declarations against registered project permissions and execution profiles.

An example proposed Go API suite manifest is:

```yaml
schemaVersion: 1
suiteId: api-regression
type: api
runner: go
applications: [service-api]
moduleRoot: .
tests:
  packages: [./tests/api]
catalog: testing/catalog.yaml
requestPolicy: testing/read-policy.yaml
buildProfile: go-module
```

Manifest validation is not a new production approval workflow.
Safety labels remain the production execution policy.

#### Builds and release activation

Persist source preparation and build jobs with progress, timestamps, logs, cancellation, and failure reasons.
The API and scheduler enqueue work. They do not run dependency installation or container builds in their own process.
Use isolated builders with the registered toolchain and bounded resource limits.
Build workers do not receive production test credentials or report-signing keys.
Repository access, private dependency access, and execution credentials remain separate bindings.

Select one supported build backend before delivering automatic source-bundle and Git preparation.
The product CLI publisher can register externally prepared releases initially.
Build backends can use local isolated workers, Kubernetes build Jobs, or configured cloud build services.
An AWS managed builder remains an AWS option. It is not a required service for other deployment targets.
The build backend remains a separate interface rather than a dependency of test scheduling.
Do not assume that the testing API container can run a Docker daemon.

Use observable preparation stages:

```text
received -> validating -> preparing -> discovering -> ready
                                      -> failed
                                      -> cancelled
```

A failed new build keeps the previous active release available.
Show that source is newer than the active release when preparation fails.
Do not report a preparation failure as a failed production health probe that never executed.
Older builds that finish late must not overwrite a newer active release.
Use activation revisions or source generations to prevent that race.

Support automatic activation of validated releases as suite policy.
Optional Activate, Pin, and Roll back actions select an existing immutable release.
These controls do not require per-test approval files or repeated confirmation for tests labeled safe.
Runs freeze their release when created.
Schedules either use the latest active release at firing time or an explicit release pin.
Changing the active release never changes tests already running.

#### Management UI and retention

The Sources area lists connection state, repository or upload origin, reference, last sync, and latest source version.
It provides Upload, Register repository, Sync now, source settings, and connection diagnostics.
The Suites area shows manifests, catalogs, build history, active release, pinned releases, and source links.
Build details expose progress and readable validation or compiler failures.
Git source links use the recorded commit rather than the current branch head.

Release comparisons show added, removed, and changed catalog entries and safety labels.
Source file changes remain inspectable without pretending that metadata alone proves behavior equivalence.
Stable identifiers preserve history when a test is updated or renamed.
Removed tests remain visible in historical runs.
Schedules that refer to removed tests show a clear configuration problem rather than silently dropping those checks.

Disabling a source stops synchronization. It does not erase retained releases or history.
Disabling a suite prevents new executions and identifies affected schedules.
Repository outages do not prevent execution of retained active releases.
Archiving a suite keeps its run history and reports accessible.
Retirement and restoration preserve permanent identifiers.
Preserve source snapshots and execution artifacts required by retained runs, release pins, or the rollback window.
Captured evidence follows the separate evidence retention and pinning policy.
Source and build retention are separate from test result and report retention.

Project permissions distinguish viewing, running, publishing, managing sources, and changing activation policy.
Record the actor and configuration version for upload, sync, activation, pin, rollback, and archive actions.

#### Source-management delivery sequence

1. Define the suite manifest, source snapshot, build, release, and activation contracts.
2. Support declarative uploads and product CLI or prebuilt release registration.
3. Add source archive uploads and GitHub registration with manual sync through one isolated build backend.
4. Add signed push synchronization, periodic sync, release comparisons, and activation policies.
5. Verify updates, failed builds, late build completion, rollback, retirement, and retained history.

No step requires a testing API deployment for each new suite release.

### HTTPS health-check test type

Add `https` as a first-class test type alongside `playwright` and `api`.
These tests check endpoint availability, response expectations, response time, and TLS health.
Each check has a permanent test identifier, project, application, category, safety label, and environment binding.
The UI creates and edits declarative check definitions without a test repository or application deployment.
Each edit creates a definition version. Existing runs keep their original version and resolved target snapshot.
Checks can also be imported through the publication API.

The first implementation supports these settings:

| Setting | Planned behavior |
| --- | --- |
| Target | Select a registered environment endpoint and relative path, or an explicitly registered project URL. |
| Method | GET by default. HEAD is an explicit alternative. Mutating methods belong to API suites. |
| Status | Configure accepted status codes. The default is 200. |
| Redirects | Disabled by default, so a login redirect cannot hide a failed endpoint. Optional redirects have a bounded limit and permitted destinations. |
| Response assertions | Optional required or forbidden text, selected headers, and JSON field expectations. No arbitrary JavaScript expressions. |
| Time limits | Configure an overall timeout and optional maximum response time. Use a proposed 10-second timeout initially. |
| TLS | Validate trust, hostname, and certificate dates. Optionally require a minimum remaining certificate lifetime. |
| Credentials | Optional secret references for authentication headers or basic authentication. The browser does not receive secret values. |
| Response capture | Store a bounded, redacted diagnostic excerpt when needed. Do not store complete response bodies by default. |

HEAD checks cannot require a response body.
Do not silently replace HEAD with GET when a server rejects HEAD.
Redirect destinations must pass target validation at every hop.
Do not forward credentials to a different origin or downgrade an HTTPS request to HTTP.
Explicitly configured private endpoints can use a project trust bundle. Certificate verification remains enabled.
Plain HTTP is an explicit local-development option, rather than the production default.

Example definition for the proposed contract:

```yaml
id: HTTPS-0001
type: https
application: service-api
category: health
target:
  endpoint: api
  path: /health
request:
  method: GET
  timeoutMs: 10000
  followRedirects: false
expect:
  statusCodes: [200]
  maxResponseTimeMs: 1500
  json:
    - path: status
      equals: up
  tls:
    minCertificateValidityDays: 14
recommendation: safe
rationale: Reads the registered health endpoint without changing application data.
```

Use a lightweight Go HTTP executor with a bounded worker pool in the service deployment.
Do not launch a browser or isolated container job for every health probe.
Keep execution outside HTTP request handlers and the scheduler loop.
Use durable execution claims, deadlines, cancellation, and recovery, as for other run types.
The same run creation path handles manual and scheduled execution.
An HTTP response is successful only when all configured assertions pass.
Capture every retry attempt. A passing retry remains flaky rather than a clean pass.

The executor checks only registered targets from its configured network location.
Target and redirect validation prevent access to cloud metadata services and unregistered destinations.
Private network probes require an explicitly permitted execution location.
Localhost probes require a worker that can reach the developer's application.
The same production-safety labels apply. GET and HEAD do not automatically establish that an endpoint has no side effects.

Record the following information:

- Resolved target, check version, execution location, timestamp, and attempt.
- HTTP status, final permitted URL, redirect count, and assertion failures.
- Request duration and optional DNS, connection, TLS, and time-to-first-byte timings.
- Certificate expiry date and remaining lifetime.
- Distinct failure reasons for DNS, connection, timeout, TLS, HTTP status, and content assertions.

Request duration measures probe execution, rather than queue delay or browser startup.
Run elapsed time remains a separate value.
Record unavailable timing phases as unavailable rather than zero.
Success rates describe observed checks. Missing executions do not imply successful checks or measured uptime.

The dedicated UI provides an HTTPS check editor and a test-type filter.
HTTPS cards appear under the same collapsible categories as other test results.
Cards show status, response time, TLS details, and readable failures.
History and trends support the existing project, application, stage, environment, category, and test filters.
Add response-time charts alongside pass and failure history for this type.
Include HTTPS checks in failure and volatility rankings.
An HTML report is optional. Native run details must work without an Allure report.

Use the in-process scheduler for frequent probes as well as nightly checks.
Apply configured concurrency and request limits to prevent a slow endpoint from blocking other projects.
Keep notification policy configurable per suite.
HTTPS suites can select every completion, failures and recovery, or a summary policy.
Any recovery or suppression state must survive service restarts.

Add this type during contract and registry work, before the deployment acceptance milestone.
Verify successful responses, unexpected status codes, timeouts, redirects, invalid TLS, approaching certificate expiry, and body assertion failures.
Verify that manual and scheduled checks produce the same run, result, trend, and notification records.
Adding or editing a health check requires no API deployment, browser image build, or service rollout.
See [Native HTTP checks](http-checks.md) for the implemented editor, assertion limits, publication rules, and retained diagnostics.

### Go test suites

Register Go suites with runner `go` and an appropriate type such as `unit`, `integration`, or `api`.
Keep existing Go tests beside their application source when appropriate.
Publish a registered test artifact from a pinned source revision and toolchain.
The service runs that artifact in an isolated execution worker, not inside the API process.
The publisher can include source and dependencies in a runner image or publish registered test binaries with the required adapter.

Suite metadata defines permitted packages, test selectors, categories, stable test identifiers, and supported execution profiles.
The UI selects catalog entries. It does not submit arbitrary shell commands or Go flags.
An example registered execution is:

```sh
go test -json -count=1 -timeout=5m -run '^TestExampleEndpoint$' ./tests/api
```

Disable Go result caching for scheduled and on-demand executions.
Parse structured events by package and test name, including interleaved parallel execution.
Map run, pass, fail, skip, output, and elapsed-time events into the common result protocol.
Store bounded test logs as diagnostics and retain the process exit status.
Record build failures, panics, timeouts, and worker termination as explicit run problems.
A package-level pass is not an additional passed test.
No selected tests executed must not appear as a successful run.

Map permanent test identifiers to package and function selectors in published metadata.
A function rename can update the selector while preserving the permanent identifier and its history.
Show dynamic subtests as children of their registered parent test by default.
Independently registered subtests receive their own identifiers.
Do not count parent aggregates and their registered children twice.
Explicit retries use separate attempts. A passing retry remains flaky.

Pure unit suites do not need target endpoints or application credentials.
Integration suites bind explicitly to their environment, services, data stores, and permitted operations.
Choose package allowlists rather than assuming `go test ./...` is safe for every repository.
Consumer repositories can contain tests that access live cloud resources.
Production classification must assess those operations before execution.
Optional race and coverage profiles use registered runner images and resource limits.
Coverage is an optional artifact, rather than a replacement for individual test outcomes.

The UI groups Go results by category or package and shows test names, subtests, attempts, duration, and failures.
Use the normal run cards, trends, rankings, schedules, and notifications.
Go tests require no Playwright or Allure dependency to produce usable results.

See [Local Go suites](go-testing.md) for the implemented source worker, manifest, timing, diagnostics, and verification contract.

### REST API test suites

Use type `api` for API behavior checks, independently of implementation language.
Support these implementation choices:

| Mode | Runner | Use |
| --- | --- | --- |
| Declarative requests | Native HTTP | Status, headers, content, JSON fields, schemas, and bounded request sequences. |
| Coded TypeScript | Playwright API request context | Reuse TypeScript fixtures and reporters for API workflows without launching a browser. |
| Existing Go integration tests | Go | Reuse application tests and Go clients for API contracts and workflows. |

Start coded API support with Playwright and the Go adapter.
Add the declarative editor for bounded cases after the shared HTTP execution and definition contracts are stable.
HTTPS checks remain the simpler availability and TLS check type.
API suites cover endpoint behavior, validation, authorization, pagination, and related request sequences.

Example read-only checks include:

- Search returns expected status, structure, and matching entries.
- A detail request returns the identifier and fields requested by the caller.
- An unknown resource returns the expected not-found response.
- Missing credentials produce the expected authorization response.
- Pagination follows the API contract without duplicate or omitted fixture records.

Resolve API base URLs and credential references from the selected environment.
Request sequences can extract values and pass them to later steps through bounded declarative bindings.
Do not accept arbitrary scripts in UI-managed request definitions.
Redact authorization headers, cookies, credentials, and sensitive response fields from diagnostics.

Browser route interception does not protect direct API clients.
Playwright API fixtures and Go HTTP clients need their own guarded transport or request wrapper.
Each request must satisfy the suite's target, method, path, payload, and redirect policy before transmission.
Production execution requires the `safe` label and reviewed read-only operations, including any read-only POST endpoints.
General Go code is trusted published code. A framework wrapper alone is not a sandbox for arbitrary code.
Restrict worker credentials and execution network access to the registered suite requirements.

Create, update, and delete workflows target permitted non-production environments.
Use dedicated test data and unique resource identifiers.
Cleanup runs after assertion failures where the worker remains available.
Record cleanup failures and remaining resource references explicitly.
Tests that change application data remain ineligible for protected production targets.

Each API test produces a permanent test identifier, category, final outcome, attempts, elapsed time, and readable assertion details.
Request-level steps can appear within its card without counting each step as another independent test.
Manual and scheduled execution use the same run, progress, result, artifact, trend, and notification contracts.
Suite publication does not require a testing service deployment.

### Adapter delivery milestones

1. Freeze the shared result contracts, test type and runner fields, stable identifiers, and environment capabilities.
2. Implement the generic Playwright adapter and native HTTPS checks with newly written example suites.
3. Add the Go event adapter and coded API suites, including direct-client request policies.
4. Add the declarative REST editor for bounded request sequences.
5. Verify browser, HTTPS, Go unit, and API suites in two newly configured generic projects across all deployment profiles.

### Local execution

Remote cloud and Kubernetes workers cannot reach a developer's localhost through a localhost URL.
The UI must show which execution target has network access to the selected environment.
Reject an incompatible execution target before creating an executable run.

Provide local execution through a local service or worker with access to the application.
Local, ECS, Kubernetes, and Azure execution use the same run registration, progress, result, and artifact contracts.
Local CLI executions must appear in service history when registered with the service.
Document container-to-host networking when the local executor uses Docker.
Keep the request guard enabled for local targets that read production APIs.

### Dedicated UI

The UI has its own application shell, authentication, design tokens, and runtime configuration.
It builds from this repository alone.
Use generic interaction patterns without importing another application's controls or styles.

#### Recommended stack

Use a standalone Angular application under `web/` in the service repository.
Use Angular 22 as the proposed target.
Verify the test tree, result cards, forms, and charts against selected package versions before finalizing dependencies.
No other application requires an upgrade or a shared library build.

Use these building blocks:

| Layer | Proposed choice |
| --- | --- |
| Components and state | Standalone Angular components, signals, focused services, and pure utilities. |
| Accessible interactions | Native controls for simple cases, Angular ARIA for headless widget patterns, and CDK for dialogs, overlays, and accessibility utilities. |
| Styling | App-owned CSS and shared design tokens for spacing, typography, borders, focus, and status colors. |
| Charts | ECharts through a compatible Angular integration. |
| API client | Typed client generated from the service OpenAPI contract, with a small authentication adapter. |
| Authentication | OIDC authorization code sign-in through Go-managed sessions. Authorization uses `wtiger001/go-permissions`. |

Create a small set of shared UI primitives for buttons, form fields, selectors, dialogs, cards, status badges, and empty states.
Reuse these primitives across feature pages.
Keep test execution and source management out of visual components.
Keep styles scoped to app-owned classes so document elements cannot introduce unrelated width or alignment constraints.

#### Layout and visual direction

Use a compact operational dashboard with a persistent desktop sidebar and fluid main workspace.
Place the project switcher at the top of the sidebar.
Show application and environment filters in the page toolbar when relevant.
Keep Run tests as the primary execution action.
Make the chosen project, target, and protected state visible before execution.

The main sidebar contains Overview, Tests, Runs, Schedules, Trends, Sources, and Settings.
Tests groups suite catalogs, native check editing, and release details.
Project settings contain applications, environments, credentials references, and notification settings.
Global settings contain identity integration and worker configuration when the user has access.
The page inventory below does not require a separate sidebar item for every page.

Use neutral light backgrounds, white cards, dark text, subtle borders, and one navigation accent.
Reserve green for success, red for failures, amber for flaky or incomplete results, and neutral colors for pending states.
Use blue for timeout status. Keep timeout counts separate from assertion failures.
Always show status text as well as color.
Use readable body text, compact metadata, and consistent control heights.
Keep desktop content gutters small and cards at the available content width.
Collapse the sidebar on narrow screens and wrap filters and card metadata.
Honor reduced-motion settings and visible keyboard focus.

#### Primary screens

The Overview page prioritizes running work, recent failures, health-check status, and upcoming schedules.
It links directly to the relevant run or test history.
Use tables for scanning many runs or sources, and cards for detailed test results.

The Tests page provides the catalog table and tree, editing workflows, validation, publication, and revision history.
It includes test-type and runner filters, safety labels, and suite or application context.
Run selection supports categories, individual tests, and explicit dependencies.
Native HTTPS checks and REST definitions use structured editors.
Go and Playwright tests display their source and release metadata.

The Sources page provides Upload tests and Connect repository actions.
Source detail shows sync state, the resolved revision, build progress, failures, catalog changes, and active releases.
Keep source preparation status distinct from the outcome of a test execution.

Run details retain the current progress indicator, rotating activity icon, category summaries, collapsible categories, and full-width test cards.
Keep attempts and error output readable across the card width.
Provide the timeline, live logs, screenshots, traces, and artifact states described in the evidence contract.
Test and category links open their history while retaining context.

Trends use stacked area charts for result states. Retain failure and volatility rankings.
Stack passed, failed, timed-out, flaky, skipped, and unknown or incomplete counts within the same chart.
Use a stable stack order and consistent state colors across overall, category, and test views.
The default vertical axis shows counts. Each run contributes one final logical result per selected test.
Keep retries in attempt history. They do not add extra counts to the stacked areas.
Show each state's count and the total in the tooltip and an accessible data table.
Missing history creates visible gaps. Do not interpolate missing observations or count incomplete results as passed.
Project and environment scope, period, sample counts, and partial-history notices remain visible.
HTTPS check views add response-time history and certificate information.
See [Result trends](trends.md) for paging, logical counts, gaps, rankings, and the current project trend API.

Use separate routes for complex editing, including sources, environments, native checks, and schedules.
Use dialogs for short actions such as rename or confirmation.
Persist shareable filters in URLs.
Keep loading, empty, unavailable, and permission-denied states distinct.
An unavailable API must not appear as an empty healthy project.

#### Page help

Provide Help in the shared header on every page, including sign-in.
Open a drawer that slides in from the right and shows help for the current page.
Each guide contains Introduction, Concepts, and Tutorials with numbered task steps.
Keep guide content aligned with available controls and behavior.
Support keyboard navigation, Escape dismissal, focus restoration, and independent content scrolling.
Preserve unsaved page edits when opening and closing help.
Use shared light and dark theme tokens. Respect reduced motion and narrow viewports.
Verify the workflow through TestStudio's own Playwright suite.

#### Hosting and delivery approach

Build the UI as static assets with independent versioned releases.
Support the four deployment profiles defined above.
Cloud profiles can deploy the UI separately from the Go service.
Local and container distributions can serve the same UI build through the Go service.
Route authenticated API traffic through a configured origin or development proxy.
Do not cache authenticated API responses in a CDN.
Authorize private artifact access through the selected storage provider.
Keep generated reports separate from the authenticated UI origin on every profile.
Keep service addresses and identity settings in runtime configuration.
UI-only deployments do not require an API rollout when the contract remains compatible.

Review wireframes for Overview, Source import and build status, Run details, and Test history.
Validate the tree, cards, charts, and login flow with newly written product fixtures.
Test the complete workflows independently of any existing application or deployment.

Proposed navigation:

| Area | Purpose |
| --- | --- |
| Overview | Cross-project health, recent runs, and failures. |
| Projects | Project registration, applications, and project settings. |
| Run tests | Project, application, suite, environment, test tree, and execution target selection. |
| Runs | All authorized runs, newest first, with project and environment filters. |
| Run details | Progress, cards grouped by category, attempts, errors, cancellation, reruns, and reports. |
| Trends | Overall, category, and test stacked area charts with failure and volatility rankings. |
| Schedules | Project schedules, target selection, enabled state, and next execution. |
| Environments | Application targets, named endpoints, stages, protection, and credential references. |
| Sources | Uploads, repository registration, connection status, synchronization, and source versions. |
| Suites | Published releases, build history, active and pinned versions, catalog changes, safety labels, and source metadata. |
| Settings | Users, groups, roles, permissions, OIDC providers, notifications, and worker connections. |

The project switcher supports All projects for overview and history.
Execution requires one explicit project and an eligible suite.
Require an environment only when the suite declares target bindings. Targetless unit suites use No target.
Within a project, keep All, All Production, All Beta, All Dev, and individual environment filters.
Cross-project views keep project identity visible in chart points, links, and rankings.
Volatility compares outcomes within the same environment and test configuration.
Incomplete runs and skipped tests do not become successful observations.
Partial-history notices and sample counts remain visible.

Use the OIDC sign-in, session, user, group, and permission contracts defined above.
No identity provider, tenant, pool, account, or group is inherited from another product.
The API resolves an internal user and evaluates project and resource permissions through `permissions.Service`.
Project checks cover catalogs, runs, environments, evidence, schedules, publication, and access management.
The browser receives no cloud keys, provider tokens, or client secret.

### Notifications

Own notification transport implementations inside the product repository.
Providers use a common delivery contract and configurable project settings.
Notification delivery uses durable retries and at-least-once semantics.
A crash after remote acceptance can produce a retry duplicate.
Run links use the configured product UI origin and project routes.
Store webhook and provider credentials through the selected secret adapter.
Store only secret references in project settings.
No notification integration requires a MarketMaker library or compatibility wrapper.
See [Durable completion notifications](notifications.md) for the outbox, scoped API, channel editor, delivery history, and local verification contract.

### Delivery sequence

1. **Define independent contracts.** Specify projects, identities, memberships, scoped permissions, OIDC sessions, suites, targets, provider ports, and execution claims.
2. **Build the local product.** Deliver SQLite storage, the permissions adapter, user and group management, OIDC sign-in, catalog management, artifacts, and worker packaging.
3. **Add generic test capabilities.** Implement browser, HTTPS, Go, and API execution with evidence capture and newly written examples.
4. **Add deployment profiles.** Implement PostgreSQL, AWS ECS and CloudFront, Kubernetes, and Azure Container Apps with their execution adapters.
5. **Add source management.** Deliver uploads, GitHub connections, isolated preparation, immutable releases, and activation controls.
6. **Validate independent projects.** Register two generic projects through configuration. Verify isolated identities, data, permissions, and history.
7. **Validate deployment parity.** Run the same product acceptance scenarios on local, AWS, Kubernetes, and Azure deployments.
8. **Document operations.** Verify installation, upgrades, backups, restoration, worker recovery, and provider configuration.

These steps describe implementation order. All four deployment profiles remain required product scope.
Use fresh product-owned resources or resources explicitly configured for this independent installation.
There is no existing-test import, historical data migration, scheduler cutover, or Admin navigation replacement.
Upgrade and rollback guarantees apply to this product's own versioned contracts and records.
Namespace metadata, artifact keys, idempotency, locks, schedules, and notifications by project and suite where applicable.
Verify schedule ownership, execution recovery, and notification retries when replicas restart or deployment revisions overlap.

### Completion criteria

- A new project, application, environment, and suite can be registered without editing Go or TypeScript source.
- Users, groups, membership, custom roles, assignments, and effective permissions are manageable through authorized API and UI workflows.
- `wtiger001/go-permissions` evaluates access on PostgreSQL and SQLite through the same service and identity contracts.
- User, group, role, object, and project grant checks cover explicit denies, expired grants, inheritance, and assignment revocation.
- Project administrators cannot grant installation privileges or affect another project's groups, assignments, results, or evidence.
- Membership and grant revocation reach all instances within the documented limit. Disabled users and revoked sessions fail their next request.
- OIDC sign-in supports configured providers with code flow, PKCE, state, nonce, issuer and audience validation, key rotation, and logout.
- Duplicate email addresses from different issuer-subject pairs do not merge accounts or transfer access.
- Unmapped, missing, incomplete, or outdated external groups do not silently preserve elevated project access.
- Browser sessions keep provider tokens server-side. Expiry, revocation, callback rejection, and request-forgery protection pass focused checks.
- SQLite restarts preserve users, memberships, assignments, custom grants, disabled roles, and sessions without a PostgreSQL service.
- Two newly configured OIDC providers work with the same internal directory and project-scoped permissions across deployment profiles.
- Catalog creation, editing, validation, publication, import, export, bulk actions, archive, and restoration work through both API and UI.
- Concurrent catalog edits produce explicit conflicts. Source synchronization preserves service-managed metadata.
- Catalog changes do not change existing run snapshots. Disabled tests and invalid dependencies block new dispatch explicitly.
- Run details expose attempts, steps, context, logs, screenshots, traces, and configured attachments without requiring HTML reports.
- Available evidence survives service restarts. Duplicate events and upload retries do not create duplicate attachments or rerun tests.
- Capture gaps, failed uploads, expired files, and incomplete evidence remain visible beside the observed test outcome.
- Artifact access respects project permissions. Private trace viewing requires no public upload or third-party viewer.
- Retention, quotas, and evidence pins work across local, S3, and Azure artifact adapters.
- PostgreSQL and SQLite pass the same storage behavior checks, including claims, revision conflicts, and restart recovery.
- Local SQLite installation requires no database server. Export and import to PostgreSQL preserve identifiers and artifact references.
- Declarative uploads, complete source bundles, and registered GitHub repositories can produce versioned releases through the UI.
- One repository can provide multiple suites, including suites beside existing application source.
- Failed or duplicate preparation events preserve the active release and do not create unintended test runs.
- Runs retain their release across updates, activation changes, and rollbacks.
- Removed tests and archived suites retain their history and required artifacts.
- Newly written generic suites complete runs and produce progress, history, reports, and trends on each deployment target.
- Result-state trends use stacked area charts with consistent state colors, final logical counts, accessible values, and visible missing-history gaps.
- HTTPS checks run manually and on schedules, with configurable assertions, TLS checks, response-time history, and normal result records.
- Adding an HTTPS check requires no browser or service deployment.
- Go suites preserve test identities, disable cached execution, and distinguish build failures from test assertion failures.
- Go unit suites run without artificial web targets or production credentials.
- API suites run through guarded direct clients and produce the same history and trends as browser tests.
- Package events, subtest aggregates, retries, and API steps do not inflate test counts.
- TestStudio executes its own new Playwright suite through normal publication, execution, timing, result, and evidence contracts.
- The Go service and UI build from this repository without MarketMaker imports, packages, shared resources, or sibling checkouts.
- No existing tests, fixtures, catalog records, configuration, credentials, reports, or history are copied into the product.
- Installation starts with empty registries and can load only explicitly selected new examples.
- AWS ECS and CloudFront, Kubernetes, Azure Container Apps, and local-server profiles pass the common acceptance scenarios.
- Local, Kubernetes, and Azure installations operate without AWS services or credentials.
- Local durable operation works without a cloud account.
- Provider adapters support execution status, cancellation, deadlines, artifacts, and recovery consistently.
- Hosted identity and private report access work through installation-specific configuration.
- Local and scheduled executions produce the same logical run and result records.
- Protected environments reject unsafe selections and unsafe dependencies.
- Manual and scheduled notifications use the same configured completion path.
- Replica failover creates no duplicate logical run. Notification retries retain their documented at-least-once behavior.
- Test-only publication requires no API or UI deployment.
- Provider credentials remain outside browser configuration and application logs.

### Decisions before implementation

- UI hostname for TestStudio. The repository name is `wtiger001/teststudio`.
- Repository visibility. Private remains the proposal.
- Database driver and migration tooling. PostgreSQL is the hosted default; SQLite is the single-instance local default.
- Initial OIDC providers, external group mappings, session limits, and external-membership refresh intervals.
- Pinned go-permissions version, Go toolchain, SQLite store implementation, and permission-cache invalidation strategy.
- Native binary, Docker Compose, and local worker packaging priorities.
- Azure static UI hosting when a separate distribution is needed. Container Apps remains the proposed compute baseline.
- Kubernetes ingress, artifact storage, secret integration, and supported runtime requirements.
- Final predefined-role action matrix, delegation rules, catalog field ownership, and bulk-operation limits.
- Default health-check frequency, response and certificate thresholds, concurrency, and notification policy.
- Initial build backend, upload limits, GitHub connection ownership, and activation policy.
- Evidence quotas, retention durations, redaction defaults, capture overrides, and trace-viewer packaging.
- UI wireframes, compatible Angular and chart package versions, and deployment release compatibility.

### Protocol references

- [Playwright reporter interface](https://playwright.dev/docs/api/class-reporter) supports custom result and progress reporting for the first adapter.
- [Go HTTP client](https://pkg.go.dev/net/http#Client) documents client deadlines and redirect policy for native probes.
- [Go TLS connection state](https://pkg.go.dev/crypto/tls#ConnectionState) exposes verified connection and certificate information.
- [Go test event format](https://go.dev/cmd/test2json/) describes structured package and test results.
- [Go test flags](https://pkg.go.dev/cmd/go/internal/test) documents uncached execution and test selection.
- [Playwright API testing](https://playwright.dev/docs/api-testing) describes request fixtures for coded REST suites.
- [GitHub App installation tokens](https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/generating-an-installation-access-token-for-a-github-app) support repository-scoped source access.
- [GitHub webhook validation](https://docs.github.com/en/webhooks/using-webhooks/validating-webhook-deliveries) describes signed push delivery validation.
- [ECS standalone tasks](https://docs.aws.amazon.com/AmazonECS/latest/developerguide/standalone-tasks.html) describe isolated AWS executions.
- [Kubernetes Jobs](https://kubernetes.io/docs/concepts/workloads/controllers/job/) describe finite container workloads.
- [Azure Container Apps and Jobs](https://learn.microsoft.com/en-us/azure/container-apps/jobs) describe continuous services and finite executions.
- [Angular ARIA](https://angular.dev/guide/aria/overview) describes headless accessible interaction patterns for the proposed UI.
- [SQLite WAL](https://www.sqlite.org/wal.html) describes local filesystem and concurrency requirements.
- [Playwright recording options](https://playwright.dev/docs/test-use-options#recording-options) describe screenshot, trace, and video capture modes.
- [Playwright Trace Viewer](https://playwright.dev/docs/trace-viewer) describes inspection of captured browser execution details.
- [go-permissions](https://github.com/wtiger001/go-permissions) defines the selected authorization library and integration guides.
- [Permission and identity interfaces](https://github.com/wtiger001/go-permissions/blob/main/store.go) define the adapter contracts to verify when pinning a version.
- [OpenID Connect Core](https://openid.net/specs/openid-connect-core-1_0.html) defines sign-in claims and token validation.
- [OpenID Connect Discovery](https://openid.net/specs/openid-connect-discovery-1_0.html) defines provider metadata discovery.
- [OAuth security practices](https://www.rfc-editor.org/rfc/rfc9700.html) define current authorization-code and redirect protections.

## Light and dark interface

Support light, dark, and system theme preferences throughout the application.
Persist the explicit preference locally and apply it before the first application render.
Use shared semantic color tokens for surfaces, text, borders, focus, charts, and status.
Use native HTML landmarks, headings, labels, buttons, forms, tables, details, and dialogs where appropriate.
Use Angular ARIA for complex interaction patterns and CDK for accessibility utilities.
Support keyboard navigation, visible focus, reduced motion, and narrow viewports in both themes.
Every status includes text or an icon as well as color.
Verify complete catalog and run workflows in both themes.

## Local-first verification

Run initial functional testing locally.
Use SQLite and a local PostgreSQL container for persistence and authorization checks.
Use local application fixtures, browser sessions, workers, and artifact storage for complete workflows.
Use provider contract tests and local container or Kubernetes checks for deployment adapters.
Run targeted tests after changes. Add broader checks only when feature integration requires them.
Use the MMDev account for a small AWS deployment smoke check after local workflows are stable.
Keep account and profile values in installation configuration.
Cloud smoke tests supplement local functional coverage. They do not replace it.
