# Saved test selections

Saved selections belong to one suite and keep a permanent identifier.
They contain explicit test identifiers or a filter with `schemaVersion: 1`.
They do not publish a release or start tests when saved.

## Catalog interface

Select tests from one suite and choose **Use selected tests** under **Saved selections**.
Enter a name and permanent identifier, then select **Save selection**.
Explicit selections accept 1 through 1000 distinct identifiers.

For a reusable filter, choose a suite and **New saved selection**.
Choose **Versioned filter** under **Selection definition**.
Filters support search text, category, child categories, required tags, owner, application, type, runner, safety, and source.
Search includes the test identifier, title, description, category, and tags.
Category paths use slash-separated names. **Include child categories** includes deeper paths.
All supplied tags must match. Other supplied fields use exact values.
Source matches an executable source path or registered definition source identifier.
Empty filter fields match any value.
Filters match enabled definitions in a published release. Publish draft metadata before using it in a filter.

Select **Inspect** to edit a stored selection.
Edits require its current revision. Conflicts require **Reload saved selection** before another save.
**Archive saved selection** removes it from normal run choices and prevents new previews.
**Restore saved selection** keeps its identifier and makes it available for a new preview.
Archiving or editing a selection preserves existing run snapshots.

## Review and execute

Select **Preview saved selection** to inspect its current published release and resolved tests.
Dependencies appear in the resolved list. A missing or disabled dependency produces an explicit error.
An empty filter result cannot become a full-suite run.

Open **Test runs** and start a new run.
Choose the suite, environment, and saved selection.
Select **Preview saved selection** in the run form and review the resolved identifiers.
The form requires this review before estimating timing or starting the saved selection.

A preview lasts 15 minutes and belongs to the user who creates it.
It pins the release, saved selection definition and revision, requested identifiers, and resolved dependency closure.
A later active release does not replace the reviewed release.
A changed or archived saved selection requires a new preview.
Current test lifecycle, permissions, target safety, and execution admission still apply at run creation and dispatch.
Run creation asserts the saved definition revision in the same commit that stores the run.
The run keeps its saved definition, release, and test snapshots after later edits or archival.
Run details displays the saved selection name, identifier, definition kind, and revision.

Failure reruns use their parent's frozen release and failed, timed-out, or flaky identifiers.
They do not depend on the original preview or the current saved selection lifecycle.

## API

Use these routes below `/api/v1/projects/{project}/suites/{suite}`:

| Method | Route | Operation |
| --- | --- | --- |
| GET | `/saved-selections` | List readable selections with an opaque cursor. |
| POST | `/saved-selections` | Create an active saved selection. |
| GET | `/saved-selections/{id}` | Inspect its current definition and revision. |
| PUT | `/saved-selections/{id}` | Update with a positive `If-Match` revision. |
| POST | `/saved-selections/{id}/actions/archive` | Archive with `If-Match`. |
| POST | `/saved-selections/{id}/actions/restore` | Restore with `If-Match`. |
| POST | `/saved-selections/{id}/previews` | Resolve the active or explicitly requested release. |

Lists default to 30 records and accept limits from 1 through 100.
Use `archived=true` to include archived records. A filtered page can be empty while its cursor permits another page.
Explicit definitions denied by exact test read rules do not appear in lists.
Definition changes use current suite catalog permissions and an atomic assignment transaction.
Preview resolution and run creation recheck read access to selected tests and dependencies.
Mutations require normal session, Origin, and request-forgery protection.

Create a filter with this body:

```json
{
  "id": "local-smoke",
  "name": "Local smoke",
  "description": "Published local availability checks",
  "archived": false,
  "spec": {
    "schemaVersion": 1,
    "kind": "filter",
    "filter": { "category": "Availability", "includeCategoryDescendants": true, "tags": ["smoke"] }
  }
}
```

Post `{}` to its preview route to use the active release.
Supply `releaseId` in that body to inspect a specific retained release.
Pass the returned preview identifier to the normal run endpoint:

```json
{
  "suiteId": "health",
  "environmentId": "local",
  "savedSelectionPreviewId": "PREVIEW_ID",
  "executionLocation": "local",
  "timeoutSeconds": 60
}
```

Use an idempotency key for run creation.
Do not combine a preview with explicit `testIds` or a different release.
Repeated run requests keep the original stored run even after its preview expires or its saved definition changes.
OpenAPI records the saved selection contracts and run reference. The UI uses the generated client for selection operations.
