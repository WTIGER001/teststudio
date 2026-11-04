# Playwright editor

Open Import and sync tests and choose a destination suite. Select New Playwright draft.
Drafts require an active suite. The offline starter requires a targetless suite. Recorded website drafts support suites with registered targets.
Each draft contains the complete source workspace for one suite.

## Open retained source

Inspect a Playwright source preparation and select Edit a copy in Playwright.
The editor opens an unsaved copy of the retained source context.
It preserves nested manifest paths, package declarations, helper files, and executable flags.
Save the copy before preparing it. The original bundle, published releases, and existing runs remain unchanged.
The application does not change a registered repository through this copy.
Binary files and workspaces above the editor limits produce an explicit error.
Continue using the complete ZIP or repository workflow for those sources.
Runtime provenance is generated again when the copied source is prepared.

## Write and save

The starter includes a TypeScript test, catalog.json, and playwright.config.ts.
The code editor provides syntax colors, line numbers, bracket matching, undo, and search.
Use Ctrl+F or Command+F to search. Tab moves keyboard focus out of the editor.
Select a file from File. Add source files or helpers with Add a source or helper file.
Keep each permanent test tag aligned with its catalog selector.
Set test titles, identifiers, timeouts, and evidence policies in catalog.json.
Keep all suite tests in the workspace and its catalog.

Select Save Playwright draft to store the current source in the installation database.
Saving permits unfinished code and invalid catalog text. It does not execute, prepare, publish, or run tests.
Drafts persist across page reloads and service restarts. Source read permission controls access.
Source management permission controls edits. Current permissions apply inside the save transaction.
Save or discard local changes before selecting another draft or suite.
Page navigation and browser reload warn about unsaved changes.

Updates require the saved revision. Concurrent edits return a conflict and preserve the local buffer.
Copy useful local edits before discarding them. Reopen the saved draft to load the latest revision.

## Prepare and publish

Select Validate and prepare draft after saving.
New starter drafts use a generated suite manifest and exact package declarations for the trusted Playwright runtime.
Imported workspaces retain their declared manifest and package files.
The current runtime uses Playwright 1.64.1. Imported package declarations must still match the offline runtime profile.
The API validates the workspace structure and snapshots its exact saved revision.
A separate worker performs normal bundle assembly and browser discovery.
Syntax errors, import errors, missing tests, and tag mismatches appear in preparation diagnostics.
Fix the draft, save a new revision, and prepare it again.

Suite preparation shows the draft identifier and saved revision.
Later draft edits do not change the retained ZIP, preparation, published release, or existing runs.
An identical request key reuses its preparation. A changed revision requires a new request key.
When preparation is ready, select Review prepared source.
Review the catalog and publish its release. Activation remains optional.
Start execution separately from Test runs.

## Current limits

The starter uses the existing offline, isolated browser profile. It has no external network access.
The starter renders a local HTML fixture inside Chromium and attaches a screenshot.
[Browser recording](browser-recording.md) captures website actions and creates reviewed source drafts.
Recorded drafts use the separate [environment-backed local profile](browser-execution.md).
That profile binds reviewed navigation to one registered endpoint and keeps direct network access disabled.
The editor supports 50 text files, 256 KiB per file, and 1 MiB of source text per draft.
Paths must remain relative and unique. New starter drafts keep generated package and manifest paths reserved.
ZIP uploads and GitHub sources remain available for larger source contexts and normal repository development.

## API

All routes use /api/v1/projects/{project}/suites/{suite}/playwright-drafts.

| Method | Suffix | Operation |
| --- | --- | --- |
| GET | /starter | Read the independent starter workspace. |
| GET | none | Read a bounded draft page with cursor and limit. |
| GET | /{id} | Read the saved source and revision. |
| PUT | /{id} | Save a draft with an explicit If-Match revision. Use 0 to create. |
| POST | /{id}/prepare | Snapshot an exact positive revision with an Idempotency-Key and queue preparation. |

Mutations require managed sessions, Origin, and request-forgery protection.
GET /starter?type=api reads a complete request-restricted API workspace. Create test offers a dedicated API wizard and Save API draft action. The source page keeps its existing browser starter button.
See [Coded API policies](api-testing.md) for source fields, transport, safety, and remaining authoring work.
Preparation rechecks source authority, catalog import authority, suite state, and the saved revision in one transaction.
It uses the same isolated source preparation and immutable release contracts as ZIP uploads.

GET /api/v1/projects/{project}/suites/{suite}/source-preparations/{id}/editor-source opens a bounded text copy of retained Playwright source.
Current source and repository read checks still protect that copy.
