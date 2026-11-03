# Playwright editor

Open Sources and choose a destination suite. Select New Playwright draft.
The suite must remain active and must not require a target environment.
Each draft contains the complete source workspace for one suite.

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
TestStudio generates the suite manifest and exact package declarations for the trusted Playwright runtime.
The current runtime uses Playwright 1.64.1. Users do not upload package files through this editor.
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

The editor uses the existing offline, isolated browser profile.
It has no network access to external websites or configured application environments.
The starter renders a local HTML fixture inside Chromium and attaches a screenshot.
Environment-backed website execution and browser recording remain separate work.
The editor supports 50 text files, 256 KiB per file, and 1 MiB of source text per draft.
Paths must remain relative, unique, and outside generated package and manifest files.
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
Preparation rechecks source authority, catalog import authority, suite state, and the saved revision in one transaction.
It uses the same isolated source preparation and immutable release contracts as ZIP uploads.
