# Retained test history

Open a test title in Test catalog, a published release, or a run result.
Test history follows its permanent project, suite, and test identity.
It shows the current definition, saved runs, and optional saved revisions.
This page starts no tests and changes no releases.

## Investigate an older run

1. Open the test title or its History action under More.
2. Review the lifecycle notice and saved run titles.
3. Select Inspect run and evidence beside the required run.
4. Inspect the selected test's attempts, logs, and private artifacts.
5. Select the result title to return to history. Return to run preserves that investigation context.

Each run keeps its original test title, release, target, final result, attempt, and timing.
Later catalog changes do not rewrite these snapshots.
A missing final observation appears as not-recorded. It does not indicate success or zero duration.
Test trends opens the existing stacked area view for this identity.
Advanced definition and Advanced revision history expose technical details only when needed.
Help provides Introduction, Concepts, and Tutorials for this page.

## Archive, source removal, and retention

Archiving a test or suite preserves saved definitions, releases, runs, and retained artifacts.
Source synchronization archives omitted tests. It does not delete their history.
If a working definition is absent, history displays its last saved revision and a removed notice.
This fallback needs a retained revision. It cannot reconstruct deliberately deleted history.
Restore an archived suite before new execution. History does not restore or enable it.

Evidence follows its existing retention, pin, quota, and authorization rules.
Archive does not extend retention. A pin preserves available files; it cannot restore deleted bytes.
Opening an older run uses the same private evidence grants as any other run.
See [Evidence](evidence.md) and [Results](results.md) for capture gaps and recovery.

## API and authority

Routes use this prefix:
`/api/v1/projects/{project}/suites/{suite}/tests/{test}`.

| Method and suffix | Result |
| --- | --- |
| GET `/retained` | Current definition or last saved revision, lifecycle, and suite archive context. |
| GET `/run-history` | Saved runs that contain this test identity, newest first. |
| GET `/revisions` | Existing saved definition revisions. |

Retained context requires current catalog.read authority for the suite and exact test.
Run history also requires project runs.read. Each row requires current authority for its exact run.
Evidence authority remains separate. A readable definition does not grant artifact access.
Refresh history hides rejected data and offers Refresh access after a permission change.

Run history defaults to 20 matches and accepts limit values from 1 through 100.
Each request scans at most 500 project runs. A partial scan may return no matches and a continuation cursor.
Select Load older runs to continue. An exhausted scan shows no continuation.
Cursors expire after 15 minutes and bind the actor, project, suite, test, and page limit.
New runs do not rewrite older-page positions. Refresh history starts with the newest records.
See [OpenAPI](../contracts/openapi.yaml) for typed fields and responses.

## Local verification

STUDIO-HISTORY-001 runs through TestStudio's normal publication, execution, and evidence workflow.
It executes two native HTTP runs and captures fresh diagnostic media in a separate local browser.
It attaches that media through the evidence service. The HTTP runner does not claim browser capture.
The workflow checks archived suites, removed working definitions, saved titles, pinned evidence, private screenshots, and trace bytes.
It also checks current access denial, contextual help, keyboard return focus, themes, and narrow layout.
Focused Go checks cover bounded scans, cursors, missing observations, exact permissions, and immutable snapshots.
The fixture removal operates only inside the disposable self-test database. Production exposes no working-definition deletion route.

Run history accepts an optional application query. It uses frozen run context and binds pagination cursors to that application.
Application catalog links retain the same scope through history, trends, and evidence navigation.
The retained definition and revision history still describe the permanent suite test identity.
