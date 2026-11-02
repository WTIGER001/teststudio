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
YAML and source archive upload remain pending.

## Preview and apply

Previews show creates, updates, unchanged definitions, before values, expected revisions, and field errors.
An import validates identifiers, native definitions, application scope, and the combined catalog dependencies.
Plans expire after 30 minutes and belong to the actor who creates them.
Apply checks current permissions, suite revision, catalog revision, and each test revision.
A changed catalog returns a conflict. Create a new preview before retrying.
The transaction applies all changes or none. Repeating an applied plan does not add revisions.
Publication remains a separate, explicit action.

## Interface workflows

The catalog table supports suite and lifecycle filters and explicit checkbox selection.
It also filters by test type. Type remains separate from the execution runner.
Bulk actions preview one suite at a time. Review the proposed states before applying changes.
The import panel loads a JSON file or accepts pasted definitions.
Export fills the document field and provides a local JSON download.
Validation errors identify the test and field. Invalid plans cannot apply.
History shows retained definitions, actor, timestamp, and revision number.
Duplicate asks for a new permanent identifier and creates a draft entry.
Opening a preview or history panel moves keyboard focus to its heading.

## Native HTTP editor

The editor supports GET, HEAD, accepted statuses, response text, exact headers, JSON fields, response-time limits, and certificate lifetime thresholds.
Field errors block saving. HEAD body assertions remain visible until the user corrects them.
Metadata-only edits preserve all assertion settings.
Validation and publication use the same HTTP definition checks as execution.
Invalid new drafts cannot replace the active release.
Opening or submitting another edit clears the previous success message.
See [Native HTTP checks](http-checks.md) for definition fields, limits, diagnostics, and local verification.

## Timing policy

Project and suite timing defaults flow into each new catalog draft.
Each test can override these defaults through the API or catalog editor.
Timing rules and timeout ceilings enter immutable releases with the test definition.
Publication activates changes for future runs. Existing runs keep their captured timing criteria.
See [Test timing](timing.md) for baseline scope, adaptive timeouts, estimates, and reset contracts.
