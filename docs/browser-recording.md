# Browser recording

## In-page recording

The user selects browser recording without tester installation as the next delivery.
The [hosted recording plan](hosted-recording-plan.md) separates the local proof, isolated hosted worker, preparation and execution, and actual tester acceptance.
The recording browser appears inside TestStudio. Testers install no companion or development tools.
Select Start recording, click News on a generic website, select Check something appears, and select a story.
Finish recording opens a plain-language step review. Save test draft reuses the existing source draft service.
Advanced source editing remains available. Saving does not prepare, publish, or execute.
Help and page reload preserve the live session. Refresh browser view recovers an uncertain input without repeating it.
Finish waits for queued input. Cancel closes the owned browser.
Operators can select the separate [private recording worker](recording-worker.md) or the loopback-only local preview.
Websites can load scripts, images, or startup APIs from other addresses. Operators approve those exact resource origins in the private worker's per-website policy. An unavailable startup dependency can leave the embedded website on its loading screen.
Development and production starting addresses need separate approvals when they use different origins or startup APIs. After an operator changes the policy and restarts the worker, start a new recording. Refresh browser view does not expand an existing recording's approved origins.
The private worker is enabled in MMDev release 0.10.0-dev. Hosted News capture, story assertion, review, saving, cancellation, and owned cleanup pass.
Hosted browser test execution and actual application acceptance remain separate work.

## Select an item and add a check

Click links and buttons, type in fields, and change checkboxes in the website preview to record normal actions. Select Add an element check to inspect an item without following its link or changing its value. The pointer changes to a crosshair. Interact with website returns to normal recording.
For a News list, select a headline or story card. Review the outlined group, choose At least this many items, leave Expected item count at 1, and select Add check. This checks that stories exist without requiring one particular headline. Special labels are not required. Semantic articles and lists help identify groups; ordinary repeated cards are also recognized. Review the group because unrelated page structures can look similar.
Count what? switches between the selected item and its similar items. Matching counts include displayed items on the rest of the page, including cards below the current view. Hidden items are excluded. Two visible cards do not imply exactly two total stories. Outlines show up to thirty matching items in the current viewport.
Use the check type menu for visibility, minimum, maximum or exact counts, contained or exact text, field values, control enablement, and checkbox or radio states. Field and control checks appear only for compatible elements. Password values are not offered. A mismatched current expectation stays in review so it can be corrected before capture. Check page address and Check page title need no element selection.
These checks generate ordinary Playwright assertions. Labels and accessible names make individual controls easier to identify; testers do not need to write selectors. Inspect the captured steps before saving. Hidden-item selection, arbitrary attributes, screenshots as assertions, hover, drag, and popup recording remain outside this milestone.

## Existing desktop recording

The local recorder uses Playwright codegen to capture browser actions and assertions.
Open Create test, select Playwright browser test, and choose a suite.
Enter a title and an HTTP or HTTPS start address without embedded credentials.
The wizard generates a permanent test identifier. Change it under Advanced recording details when needed.
Local desktop installations use Configure, Record, and Review steps. Select Start browser recording.
Hosted installations add Connect desktop. Select Continue to desktop connection.

The native recorder opens dedicated browser and Inspector windows on the computer running TestStudio.
The desktop companion opens them on the recording computer after pairing.
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
Installations configured for desktop recording use the [desktop companion](recorder-companion.md) through the guided connection step.
Installations configured for the private worker use the in-page browser and omit desktop connection.
The complete connection, native capture, stop, review, and draft workflow now passes one focused local TestStudio browser run.
Standalone macOS ARM64 packaging and offline setup also pass. Live deployment verification remains separate.

## Hosted companion API milestone

The API now supports scoped desktop pairing behind the explicit --recording-companion option.
The option remains disabled by default. Enable it only where testers have the trusted desktop command and dependencies.
The desktop command, guided wizard, and real Chromium capture now have focused local evidence.
The wizard shows the desktop command and a temporary connection code. Its First-time desktop setup disclosure describes the operator-provided archive and setup script.
Get new connection code rotates an unclaimed code. The page clears the code after pairing and hides it after expiry.
Reload restores the active recording status. It does not restore a connection code.
The final review opens the existing source editor. Saving does not prepare, publish, activate, or execute the test.

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
