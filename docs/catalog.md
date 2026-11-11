# Catalog management

Each application owns its catalog. Every suite has one application owner and owns permanent test identifiers, revision history, and immutable saved versions.
Save updates future runs automatically. Existing versions and run snapshots keep their original definitions. Source tests appear immediately while setup completes in the background. Saving never starts a run.
Current lifecycle and safety restrictions still apply before dispatch and each attempt.
Selected dependencies must remain enabled. Protected targets require safe saved and current labels throughout the dependency closure.
See [Dispatch safety](results.md#dispatch-safety-and-frozen-runs) for concurrent edits, blocked runs, and retained attempt observations.
Use explicit lifecycle actions to enable, disable, archive, or restore a test.
Duplication creates a new disabled identifier and starts independent history.

Open a test title or More > History to inspect its [retained history](test-history.md).
Revisions under More opens the compact definition history beside its catalog row.

## Save API

`POST /api/v1/projects/{project}/suites/{suite}/tests/{test}/save` saves a definition and its current execution version atomically. Use `If-Match: 0` for creation or the current revision for edits. The response includes `savedVersionId` when that exact definition is ready. GET on the test recovers this confirmation after an interrupted response.
Lifecycle actions and import-plan apply accept `save=true` to update the current version in the same transaction. Legacy mutation and publication endpoints remain available for existing API clients; normal UI workflows use direct saving. Existing save authority still applies.

## JSON transfer

The export route returns a document with `schemaVersion: "teststudio.catalog/v1"` and a `tests` array.
The document contains definitions and catalog metadata. It does not read credentials or provider configuration.
Origin project and suite identifiers provide provenance. They do not select the import destination.

Use these routes below `/api/v1/projects/{projectId}/suites/{suiteId}`:

| Route | Method | Operation |
| --- | --- | --- |
| `/catalog-exports` | GET | Read a versioned definition document. |
| `/catalog-imports` | POST | Validate native HTTP definitions and store an import preview. |
| `/catalog-imports/{id}` | GET | Read the current actor's preview. |
| `/catalog-imports/{id}/apply` | POST | Apply the preview with its `If-Match` revision. |
| `/tests/bulk-preview` | POST | Preview explicit lifecycle actions with test revisions. |
| `/tests/{id}/actions/duplicate` | POST | Create a new disabled identity with `If-Match` and a new `id`. |

Each import or bulk request accepts 1 through 30 tests within the 2 MiB request limit.
The limit keeps definitions, history, audit records, and guards in one bounded transaction.
Imports update named definitions and preserve omitted tests.
Existing lifecycle states require explicit actions. Imports do not change those states.
Coded tests require source preparation and cannot enter through declarative import.
Use Import and sync tests for reviewed JSON or YAML definitions, complete source ZIP uploads, and registered GitHub or self-managed GitLab source preparation.
Git-managed test editing opens the repository metadata review. Read-only connections store reviewed JSON in TestStudio. Read/write connections commit metadata to a review branch; merging or explicitly selecting that branch makes committed changes available to preparation. Pending edits never change an executable version.

## Preview and apply

Previews show creates, updates, unchanged definitions, before values, expected revisions, and field errors.
An import validates identifiers, native definitions, application scope, and the combined catalog dependencies.
Plans expire after 30 minutes and belong to the actor who creates them.
Apply checks current permissions, suite revision, catalog revision, and each test revision.
A changed catalog returns a conflict. Create a new preview before retrying.
The transaction applies all changes or none. Repeating an applied plan does not add revisions.
Save applies the reviewed changes and makes the current version available in one transaction. There is no separate publication step.
Existing immutable history, validation, safety, schedule pins, and revision conflicts remain. See [Saved versions](publication-review.md).

## Interface workflows

Create test opens a dedicated route and offers Playwright browser tests, HTTPS probes, and Go suites.
Each type opens its focused creation workflow. See create-tests.md.


Choose an application before opening the catalog. Search, suite, and test type stay visible above the catalog.
Table shows a flat list. By category adds a hierarchical category navigator beside the same test list.
Category paths use `/` between levels. Spaces beside separators do not change the path.
Expand or collapse a branch with its native disclosure control. Select View all in a parent to include its children.
Category counts follow search, lifecycle, suite, application, type, and the other metadata filters.
The selected category does not hide sibling navigation entries. Clear category returns to the broader filtered list.

Sort tests offers names in either direction, identifiers, category paths, and recently updated records.
Stable suite and test identifiers resolve equal values. Sorting changes the display, not execution order.
Advanced filters contains lifecycle state, exact category scope, tag, owner, runner, safety label, and source.
Include child categories is on by default. Turn it off to match only the exact category.
A source filter matches either the executable source or its registered definition source identifier.
Filters combine. An empty result explains how to recover. Clear filters restores the active catalog across suites.
Application options use readable registered names when the user can read application records.
The UI uses the shared server list contract. Historical version selections remain separate.
Load more tests appends the next page. Counts and category branches cover every authorized match.
See [Catalog lists](catalog-list.md) for query fields, cursor scope, permissions, and recovery.

Selections survive filtering and view changes. The bulk bar shows how many selected tests are currently hidden.
Select visible tests replaces the selection with the current filtered list. Clear selection removes every selected test.
Review the complete selection before applying a bulk action.
Edit and applicable Enable, Disable, or Restore actions stay visible on each row.
More contains History, Duplicate, and Archive. Normal removal uses Archive and retains history.
Lifecycle saves update future runs automatically and preserve existing runs.
Bulk actions preview one suite at a time. Review the proposed states before applying changes.
The import panel loads a JSON file or accepts pasted definitions. File reading disables transfer actions until the selected content is ready.
Export fills the document field and provides a local JSON download. It covers the complete suite, including unloaded list pages.
Export requires current read and export permission for every included definition. A denied definition rejects the full export.
An export access rejection clears the old download link and document. Transfer limits still apply when importing an exported document.
Validation errors identify the test and field. Invalid plans cannot apply.
History shows retained definitions, actor, timestamp, and revision number.
Duplicate asks for a new permanent identifier and creates a disabled entry.
Opening a preview or history panel moves keyboard focus to its heading.
Saved selections support explicit identifiers and versioned filters through the same catalog permissions.
The catalog manages their definitions, previews, archive, and restoration.
The run form previews a saved selection before using its frozen release and resolved identifiers.
See [Saved test selections](saved-selections.md).

## Bulk metadata

Select 1 through 30 tests from one suite. Choose **Edit metadata** under **Bulk action**.
Select each field to replace. Available fields include title, description, category, tags, owner, safety label, rationale, and operating conditions.
Unselected fields keep their values. Empty optional text clears the selected field. An empty tag list removes all tags.
The UI accepts comma-separated tags. Tags require distinct values and remain limited to 30 entries of 64 bytes each.
Titles, categories, and owners accept up to 200 bytes. Descriptions, rationales, and conditions accept up to 4096 bytes.
Titles, categories, and selected rationales require nonempty text.
Select an explicit rationale when selecting a safety label. A changed label requires current save permission.

Select **Preview bulk changes** and inspect the visible before and after values.
The preview retains the full definitions for optional inspection.
Apply uses the same revision-controlled plan operation as imports and lifecycle actions.
Permission checks and mutations share one assignment transaction. A rejected change cannot partially apply.
Concurrent replays of an applied plan return its result without adding test revisions.
Plans recheck current exact read and editing authority. A stored preview does not retain revoked access.

The existing `POST /tests/bulk-preview` route accepts this request:

```json
{
  "action": "metadata",
  "tests": [{ "id": "HEALTH-001", "revision": 3 }],
  "metadata": {
    "category": "Availability / Local",
    "tags": ["reviewed", "local"],
    "owner": "Example team"
  }
}
```

Omitted or null metadata fields remain unchanged. Use an empty string or array for an explicit supported clear.
Metadata cannot include executable fields, selectors, source identities, dependencies, runner settings, or lifecycle state.
Prepared source facts stay unchanged. Future source synchronization preserves the service-managed metadata.
Metadata saves update future runs automatically. They never start a run or rewrite retained results.
After a conflict, select Reload catalog and create a new preview. Reload preserves the import document.
The API request, plan read, and apply operations are recorded in OpenAPI and the generated TypeScript client.

## Native HTTP editor

The editor supports GET, HEAD, accepted statuses, response text, exact headers, JSON fields, response-time limits, and certificate lifetime thresholds.
Field errors block saving. HEAD body assertions remain visible until the user corrects them.
Metadata-only edits preserve all assertion settings.
Validation and publication use the same HTTP definition checks as execution.
Invalid new drafts cannot replace the active release.
Opening or submitting another edit clears the previous success message.
See [Native HTTP checks](http-checks.md) for definition fields, limits, diagnostics, and local verification.

## Go definitions

Go definitions use one explicit package and test path per permanent identifier.
Register a parent aggregate or its independent child tests. Overlapping selectors block publication.
The immutable image manifest and discovered selector map must match each published definition.
See [Local Go suites](go-testing.md) for the source manifest and runtime contract.

## Prepared source ownership

Prepared catalogs record a stable definition source identifier on each managed test.
Direct edits preserve source facts. A new reviewed preparation updates those facts.
Synchronization retains service-managed metadata and explicit lifecycle choices.
Omitted tests owned by that source become archived. Unrelated definitions remain in the suite.
History, existing releases, and run snapshots retain removed definitions.
Preparation can review up to 1000 definitions within one atomic publication.
See [prepared suites](prepared-suites.md) for the local image workflow and current limits.

## Timing policy

Project and suite timing defaults flow into each new catalog draft.
Each test can override these defaults through the API or catalog editor.
Timing rules and timeout ceilings enter immutable releases with the test definition.
Publication activates changes for future runs. Existing runs keep their captured timing criteria.
See [Test timing](timing.md) for baseline scope, adaptive timeouts, estimates, and reset contracts.

## Native source releases

Use Import and sync tests to stage JSON or YAML files and publish their reviewed catalog changes atomically.
The [source upload guide](sources.md) describes provenance, retention, permissions, and publication retries.

## Catalog API coverage

OpenAPI 0.28 and the generated client describe lists, test creation and updates, revision history, lifecycle actions, duplication, JSON transfer, drafts, validation, and retained release reads.
Reviewed publication and activation retain their documented operations.
Catalog changes still require current context-specific permissions, Origin, CSRF protection, and mutation revisions where applicable.

## Environment requirements

Tests can declare `requiredParameters` using names from their application's configuration contract.
Existing `requiredBindings` names declare secret references for reviewed API execution.
HTTP probes require their named endpoint automatically. Environment browser profiles require their prepared endpoint.
Required configuration includes the selected dependency closure. Optional requirements of unrelated tests do not block a run.
HTTP paths, required text, and forbidden text support `{{parameterName}}` references declared in `requiredParameters`.
Path substitutions are escaped as path data; query substitutions are escaped as query data. They cannot choose another origin.
Run snapshots retain parameter values and secret reference identifiers. They retain no resolved secret bytes.
See [Environment configuration](environment-configuration.md).
