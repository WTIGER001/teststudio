# Browser recording

The local recorder uses Playwright codegen to capture browser actions and assertions.
Open Create test, select Playwright browser test, and choose a suite.
Enter an HTTP or HTTPS start address without embedded credentials.
Enter a permanent test identifier and title. Select Start browser recording.

The recorder opens dedicated browser and Inspector windows on the computer running TestStudio.
Use those windows to record actions and add assertions.
Select Stop and keep recording to close the owned windows and retain generated source.
Closing the recording browser also ends the session. The page detects that change during polling.
Select Review recorded test to open an unsaved source workspace.
Saving that workspace creates a durable source draft.

## Review and execution

The source converter adds the permanent test tag and matching catalog selector.
It retains actions and assertions and uses the needs-review safety label.
Supported start-origin navigation receives a visible runtime target expression. The original code stays in private recording history.
Review code, assertions, timeouts, capture settings, and safety metadata before publication.
Raw generated code remains readable when conversion cannot identify one standard recorded test.

Recording, draft saving, publication, and execution remain separate operations.
Recorded drafts use the separate [local website execution profile](browser-execution.md).
Review and save the source, prepare it, publish a release, then select a registered environment for execution.
The offline starter retains its existing isolated profile. It cannot reach a website.
The current native recorder requires a local desktop.
Hosted recording requires the [desktop companion](recorder-companion.md). Its guided connection workflow remains pending.

## Hosted companion API milestone

The API now supports scoped desktop pairing behind the explicit --recording-companion option.
Keep this option disabled until the desktop client and guided authoring flow are installed and verified.
The desktop command now has focused local lifecycle and native helper evidence. The connection wizard remains unfinished.
This API milestone does not yet enable the full hosted recording workflow for testers.

A hosted session starts in pairing state. Its owner requests a five-minute connection token.
The desktop supplies a separate random device token before pairing. The server stores only credential hashes.
Pairing binds one desktop instance to the owner, project, suite, session, and original twenty-minute deadline.
An identical pairing retry recovers a lost acknowledgement. Another desktop cannot take over the session.
Rotating an unclaimed connection token invalidates the previous token. A paired session cannot issue another connection token.

The device can read control requests and submit ordered source snapshots for that session only.
Its credential cannot read catalog data, run tests, or create an application session.
Each request checks current source authority and project and suite archive state.
Stop and cancel requests survive server restart. An exact snapshot retry is idempotent.
Maintenance fails expired, disconnected, or revoked sessions while retaining previously captured private source.
The desktop must independently stop its owned browser when control requests fail or its deadline expires.
The client uses outbound HTTPS. No local HTTP listener or browser connection bridge is required.

## Ownership and recovery

Only the recording owner can read a session and its generated code.
Current suite source permissions also apply to each request.
Saving a source draft makes that draft available through normal suite source permissions.
The recording history shows the current actor's sessions for the selected suite.
Reopening a page resumes polling for an active session.
Repeated identical session identifiers do not open another recording browser.
Cancel recording closes the browser without offering automatic draft conversion.

One recording runs on the local computer at a time.
Sessions have a 20-minute limit. The maintenance worker also stops sessions after source authority is revoked.
The trusted helper stops its child when its server input closes or its own time limit expires.
Server shutdown closes owned recorder processes. Unrelated browser windows remain open.
Private source files allow recovery after a service restart.
Generated code has a 256 KiB limit. Previously captured source remains retained after a capture failure.

## Local setup

Install the locked runner dependencies and native Chromium browser:

```sh
npm ci --prefix runners/playwright
node runners/playwright/node_modules/playwright/cli.js install chromium
```

Use the server options below with explicit --local mode:

```sh
--recording-cli runners/playwright/node_modules/playwright/cli.js
--recording-helper scripts/record-browser.mjs
--recording-directory .data/recordings
```

The CLI must match the trusted Playwright version, currently 1.64.1.
The recorder uses fixed command arguments and the trusted helper. It does not execute uploaded source in the API.
Recording files stay in a private directory. Browser logs do not enter application logs or recording API responses.
The demo launcher configures this recorder and reuses the installed native browser.
Recording remains disabled on ordinary installations unless configured explicitly.

## API

Routes use /api/v1/projects/{project}/suites/{suite}/browser-recordings.

| Method | Suffix | Operation |
| --- | --- | --- |
| GET | /options | Read recorder availability and its time limit. |
| GET | none | Read the current actor's bounded recording history. |
| POST | none | Start a session with id, url, testId, and title. |
| GET | /{id} | Read current state and generated code. |
| POST | /{id}/pairing | Issue or rotate an unclaimed hosted connection token. |
| POST | /{id}/actions/stop | Close the recorder and retain generated source. |
| POST | /{id}/actions/cancel | Cancel the recording and close its browser. |
| GET | /{id}/source | Convert a finished recording into an unsaved source draft. |

Managed sessions protect all routes. Mutations require Origin and request-forgery protection.
Starting and stopping require current source management permission.
Recorded code never enters audit records.

Native companion routes use /api/v1/recording-companion/sessions/{id}.
POST /pair accepts pairingToken, instance, and deviceToken without browser cookies or an application session.
POST /control accepts instance and requires the device bearer token.
POST /snapshots also accepts a snapshot with sequence, state, optional code, and a bounded failure category.
Companion routes reject Origin, Cookie, and Sec-Fetch-Site headers. Responses issue no application cookie and disable caching.
Pairing tokens do not authorize control or source submission. Device tokens do not authorize ordinary API routes.
See the [API contract](../contracts/openapi.yaml) for request and response shapes.
