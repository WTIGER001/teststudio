# Catalog management

Each suite owns permanent test identifiers, revision history, drafts, and immutable releases.
Catalog edits affect future publications. Existing releases and run snapshots keep their original definitions.
Use explicit lifecycle actions to enable, disable, archive, or restore a test.
Duplication creates a new identifier in draft state and starts independent history.

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
| `/tests/{id}/actions/duplicate` | POST | Create a new draft identity with `If-Match` and a new `id`. |

Each import or bulk request accepts 1 through 30 tests within the 2 MiB request limit.
The limit keeps definitions, history, audit records, and guards in one bounded transaction.
Imports update named definitions and preserve omitted tests.
Existing lifecycle states require explicit actions. Imports do not change those states.
Coded tests require source preparation and cannot enter through declarative import.
Use Import and sync tests for reviewed JSON or YAML definitions, complete source ZIP uploads, and registered GitHub source preparation.

## Preview and apply

Previews show creates, updates, unchanged definitions, before values, expected revisions, and field errors.
An import validates identifiers, native definitions, application scope, and the combined catalog dependencies.
Plans expire after 30 minutes and belong to the actor who creates them.
Apply checks current permissions, suite revision, catalog revision, and each test revision.
A changed catalog returns a conflict. Create a new preview before retrying.
The transaction applies all changes or none. Repeating an applied plan does not add revisions.
Publication remains a separate, explicit action.

## Interface workflows

Create test opens a dedicated route and offers Playwright browser tests, HTTPS probes, and Go suites.
Each type opens its focused creation workflow. See create-tests.md.


Search, suite, application, and test type stay visible above the catalog.
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
The browser filters the authorized working catalog. Published release selections remain separate.

Selections survive filtering and view changes. The bulk bar shows how many selected tests are currently hidden.
Select visible tests replaces the selection with the current filtered list. Clear selection removes every selected test.
Review the complete selection before applying a bulk action.
Edit and applicable Enable, Disable, or Restore actions stay visible on each row.
More contains History, Duplicate, and Archive. Normal removal uses Archive and retains history.
Changing lifecycle does not publish a release or rewrite an existing run.
Bulk actions preview one suite at a time. Review the proposed states before applying changes.
The import panel loads a JSON file or accepts pasted definitions.
Export fills the document field and provides a local JSON download.
Validation errors identify the test and field. Invalid plans cannot apply.
History shows retained definitions, actor, timestamp, and revision number.
Duplicate asks for a new permanent identifier and creates a draft entry.
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
Select an explicit rationale when selecting a safety label. A changed label requires current publishing permission.

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
Metadata edits do not publish a release, start a run, or rewrite retained results.
After a conflict, reload the catalog and create a new preview.
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
