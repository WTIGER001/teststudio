# Create tests

Select Create test from the catalog.
The application opens /projects/{project}/tests/new.
Choose the test type before entering its definition.

| Type | Route suffix | Workflow |
| --- | --- | --- |
| Playwright browser test | /playwright | Record browser actions, then review and save generated source. |
| HTTPS probe | /https | Follow Name, Request, and Review. Save a draft or explicitly save and run. |
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

Enter a title and choose a suite. Select Continue.
The wizard generates a permanent identifier. Advanced details contains its editor and the category.
Choose a registered target environment. The picker shows its stage and protection.
Choose its endpoint alias, then enter the relative path and accepted status codes.
The wizard previews the resolved address. Other runs can use the same alias in another environment.
Open named Advanced sections for response assertions, method, TLS, timeout, or slowdown settings.
Select Continue. Invalid options reopen their section and receive focus.
Back and completed step buttons preserve entered values.
Review the request, target, safety label, and rationale.
GET and HEAD do not establish production safety. New probes require review by default.
Select Save draft to add the definition and return to the catalog.
Save and run requires publication and run permissions, plus a registered target.
Protected targets also require safe definitions and protected execution permission.
This action validates and publishes a suite snapshot without changing its active release.
It starts only the selected test and its dependencies through the normal local run API.
The action keeps successful save and publication receipts during a retry.
Publication and run creation reuse stable idempotency keys.
After a partial failure, entries stay locked. Retry run continues the action. Open saved draft returns to the catalog.
Reloading ends this in-page continuation. Saved resources remain in the catalog and history.
The wizard shares check parsing, timing rules, and saving with the catalog editor.
See http-checks.md for supported assertions.

## Go suites

Choose a destination suite. Upload its complete source ZIP and enter the suite manifest path.
Inspect preparation diagnostics and review the discovered catalog before publication.
Repository connections remain under Sources.
See source-bundles.md for archive limits and source context rules.
