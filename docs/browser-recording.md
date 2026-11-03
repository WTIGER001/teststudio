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
It retains generated browser steps and uses the needs-review safety label.
Review code, assertions, timeouts, capture settings, and safety metadata before publication.
Raw generated code remains readable when conversion cannot identify one standard recorded test.

Recording, draft saving, publication, and execution remain separate operations.
The current isolated browser execution profile cannot reach the recorded website.
Recorded drafts retain their start address and block preparation through that profile.
Environment-backed browser execution remains required work.
The current native recorder requires a local desktop.
Hosted recording requires a local companion or streamed browser, which remains pending.

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
| POST | /{id}/actions/stop | Close the recorder and retain generated source. |
| POST | /{id}/actions/cancel | Cancel the recording and close its browser. |
| GET | /{id}/source | Convert a finished recording into an unsaved source draft. |

Managed sessions protect all routes. Mutations require Origin and request-forgery protection.
Starting and stopping require current source management permission.
Recorded code never enters audit records.
