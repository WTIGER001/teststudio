# Test source publication

The Import and sync tests page accepts native HTTP catalogs in JSON or YAML.
Choose a project and an existing suite before uploading a file.
Use project settings to create a suite when required.

The same page prepares locally available immutable Go and Playwright images when the server enables that worker.
See [prepared suites](prepared-suites.md) for image labels, durable jobs, cancellation, recovery, and CLI publication.
The same page uploads [Go ZIP bundles](source-bundles.md) for a separate preparation worker.
The CLI can also assemble bundles before server inspection.
GitHub repository registration and offline Go and Playwright bundle assembly use the same reviewed publication flow.
[Recorded website execution](browser-execution.md) uses a separate environment-backed profile.

## Review and publication

Select Upload tests and load a UTF-8 file.
One upload accepts at most 30 test definitions and 1 MiB of source text.
The request body also has a 2 MiB limit.
The server stores the source text and computes its SHA-256 digest.
The digest covers the stored UTF-8 bytes, including whitespace.

Validation previews identifiers, titles, categories, types, safety labels, and definition changes.
Duplicate keys, aliases, merge keys, multiple documents, custom tags, and unsupported scalar values are rejected.
Documents also have depth and node limits.
The normal catalog validators check definitions and dependencies.

Review each change before selecting Publish source release.
The upload user must publish their own review.
Publication applies the reviewed changes and freezes a catalog release in one database transaction.
A failure rolls back all catalog, draft, release, and upload changes.
Source-bound import plans cannot be applied through the ordinary catalog import endpoint.

Activation is an explicit choice and starts unchecked.
Publication and activation do not start a test run.
A ready release proves executable definitions, not successful test outcomes.
Use Test runs to start tests explicitly.

Omitted tests remain in the suite.
Existing definitions change only through the reviewed import.
This direct upload workflow can replace editable metadata shown in the preview.
Prepared images use separate ownership for source facts and service-managed metadata.
Synchronization preserves titles, descriptions, categories, tags, owners, lifecycle state, safety policy, timeouts, timing, and capture settings.
Source-owned selectors, files, runners, dependencies, and bindings change through a reviewed preparation.
Missing definitions from that same prebuilt source become archived. Their revisions and existing release snapshots remain available.
A native upload does not supply a coded execution artifact.
Prebuilt preparation supplies the immutable image, discovered mapping, and image-reported provenance.
Prepared catalogs support up to 1000 tests. Declarative uploads retain their 30-test limit.

## Conflicts and retries

A preview records catalog and suite revisions.
Intervening changes reject publication with status 409.
Review the current catalog and create a new preview.
A failed upload or publication does not replace the active release.

Publication retries for the same upload return the original immutable release.
A retry cannot change the original activation choice.
Later activation and rollback use the normal release activation API.
Existing runs retain their original catalog snapshots.

## Source retention

Staging expires after 30 minutes.
Expired staging cannot publish or provide source content.
A bounded maintenance worker removes expired source text and import plans.
The worker retains an expired upload summary.

Published uploads retain their source content and catalog plan.
The release artifact records `sourceSnapshotId` and `sourceRevision`.
Catalog draft and publication checks verify the referenced source and digest.
The current implementation retains published snapshots indefinitely.
Configurable published-source retention remains pending.

## Authority

Source reads require `sources.read` and `catalog.read` for the destination suite.
Staging requires `sources.manage`, `catalog.import`, and the normal per-test catalog permissions.
Publication also requires `catalog.publish` and the normal draft permissions.
A publisher can also need the Catalog editor role to create or update definitions.
All publication checks run inside the shared authorization and metadata transaction.
Upload audit records contain identifiers and digests, not file contents.

## API

Routes use `/api/v1/projects/{project}/suites/{suite}` as their base.

| Method | Route | Result |
| --- | --- | --- |
| GET | `/source-preparation-options` | Read enabled image runners, bundle runners, and execution location. |
| POST | `/source-bundles` | Store a ZIP and queue its build with an `Idempotency-Key`. |
| GET | `/source-preparations/{id}/bundle` | Download exact authorized source ZIP bytes. |
| GET | `/source-preparations` | Read preparation history. |
| POST | `/source-preparations` | Queue an immutable image with an `Idempotency-Key`. |
| GET | `/source-preparations/{id}` | Read progress, diagnostics, and review identifiers. |
| POST | `/source-preparations/{id}/cancel` | Cancel unpublished work with `If-Match`. |
| GET | `/source-uploads` | Read a bounded upload history page. |
| POST | `/source-uploads` | Store a source file and prepare its review. |
| GET | `/source-uploads/{id}` | Read an upload summary and retained plan. |
| GET | `/source-uploads/{id}/content` | Read authorized retained source text and its digest. |
| POST | `/source-uploads/{id}/publish` | Publish the reviewed upload atomically. |

ZIP staging uses an application/zip body, with name and manifest query parameters.
The ZIP limit is 32 MiB compressed. The server validates archive limits and paths before storing it.
Native staging accepts `name`, `format`, and `content`.
Format must be `json` or `yaml`.
Publication requires a positive `If-Match` upload revision and an explicit `activate` boolean.
Mutations require the installation Origin and a current session request token.
The OpenAPI contract describes the implemented JSON operations and generated client types.

## YAML format

Use the catalog schema `teststudio.catalog/v1`.
Endpoint names refer to the selected environment configuration.
This definition reads only its configured health endpoint.

```yaml
schemaVersion: teststudio.catalog/v1
tests:
  - id: HEALTH-001
    title: Health endpoint responds
    description: Read the configured health endpoint
    category: Availability
    tags: [health]
    type: https
    runner: http
    state: enabled
    safety: safe
    rationale: Read only the registered health endpoint
    dependencies: []
    requiredBindings: []
    timeoutSeconds: 10
    check:
      endpoint: api
      path: /healthz
      method: GET
      statusCodes: [200]
```
