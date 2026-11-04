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
Prepare the saved draft, review its catalog, and publish an active release.
Use Test runs to select a registered environment and review its endpoint before execution.
See [Browser recording](browser-recording.md) and [Website execution](browser-execution.md) for setup, privacy, recovery, and limits.

## HTTPS probes

Enter a title and choose a suite. Select Continue.
The wizard generates a permanent identifier. Advanced details contains its editor and the category.
Paste the complete website URL and enter accepted status codes.
HTTPS uses port 443 automatically. An explicit port is optional.
The wizard derives the endpoint origin, path, and query from the address.
A URL entered as the title also fills the address when it is valid.
If one registered target matches, the wizard reuses its endpoint and existing protection.
If several targets match, choose one explicitly. Disabled targets remain visible and cannot run.
A new address lists its target configuration on Review before saving.
The target belongs to the suite's linked application, or an application chosen under Advanced target options.
If no applications are available, Review lists a new application named from the hostname.
Saving new target configuration requires environment management permission.
Saving a new application also requires project management permission.
Advanced target options contains the application choice, stage, and protection.
Choose Local explicitly for a new plain HTTP or private-network target.
Save draft stores the reviewed configuration and test without contacting the website.
Save and run also starts the selected check locally.
Use a registered environment remains available for endpoint aliases and relative paths.
Other runs can use the same alias in another environment.
Open named Advanced sections for response assertions, method, TLS, timeout, or slowdown settings.
Select Continue. Invalid options reopen their section and receive focus.
Back and completed step buttons preserve entered values.
Review the request, target, safety label, and rationale.
GET and HEAD do not establish production safety. New probes require review by default.
Select Save draft to add the definition and return to the catalog.
Save and run requires publication and run permissions. It saves any reviewed new target first.
Protected targets also require safe definitions and protected execution permission.
This action validates and publishes a suite snapshot without changing its active release.
It starts only the selected test and its dependencies through the normal local run API.
The action keeps successful application, target, test, and publication receipts during a retry.
Publication and run creation reuse stable idempotency keys.
After a partial failure, entries stay locked. Retry run continues the action. Open saved draft returns to the catalog.
Reloading ends this in-page continuation. Saved resources remain in the catalog and history.
The wizard shares check parsing, timing rules, and saving with the catalog editor.
See http-checks.md for supported assertions.

## Go suites

Choose a destination suite. Upload its complete source ZIP and enter the suite manifest path.
Inspect preparation diagnostics and review the discovered catalog before publication.
Repository connections remain under Import and sync tests.
See source-bundles.md for archive limits and source context rules.
