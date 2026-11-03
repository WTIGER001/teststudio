# Create tests

Select Create test from the catalog.
The application opens /projects/{project}/tests/new.
Choose the test type before entering its definition.

| Type | Route suffix | Workflow |
| --- | --- | --- |
| Playwright browser test | /playwright | Record browser actions, then review and save generated source. |
| HTTPS probe | /https | Enter request settings and assertions in the native form. |
| Go suite | /go | Upload a complete source bundle and review discovered tests. |

The creation page has its own contextual help.
Back to catalog returns to the catalog. Choose another test type returns to the chooser.
Unsaved test definitions and source edits require an explicit choice before leaving.
Empty installations show a suite setup link after selecting a type.

## Playwright browser tests

Choose a destination suite. Enter the recording address, permanent test identifier, and title.
Select Start browser recording.
The trusted local recorder opens a browser and Playwright Inspector on the server desktop.
Record actions and assertions. Return to TestStudio and select Stop and keep recording.
Select Review recorded test. Review the generated code and catalog metadata, then save the draft.
Recorded tests start with the needs-review safety label.
Recording does not publish or execute a test.
See browser-recording.md for local setup, privacy, recovery, and current execution limits.

## HTTPS probes

Choose a suite and permanent identifier. Enter the title, category, timeout, safety label, and rationale.
Configure the request and expected response with the native check form.
Select Save draft changes. The catalog opens and shows the new definition.
Validate and publish a suite release before starting a run.
Catalog edits and new probe creation use the same form, validation, and save service.
See http-checks.md for supported assertions.

## Go suites

Choose a destination suite. Upload its complete source ZIP and enter the suite manifest path.
Inspect preparation diagnostics and review the discovered catalog before publication.
Repository connections remain under Sources.
See source-bundles.md for archive limits and source context rules.
