# Desktop recording companion

## Introduction

The desktop companion connects a hosted TestStudio recording session to Playwright on the tester's computer.
It opens only its dedicated recording browser and Inspector. It does not execute uploaded test source.
The first command implementation targets macOS and Linux desktops.
The guided connection wizard remains unfinished. Keep hosted recording disabled in ordinary installations until that workflow passes verification.

## Concepts

A connection code combines the recording session identifier and a five-minute pairing credential.
Paste the code into the companion prompt. Do not put it in command arguments, URLs, source files, or shell history.
Pairing binds one desktop instance. The companion creates and saves its separate device credential before contacting TestStudio.
The server stores hashes. The private desktop journal stores the credentials needed to recover a lost acknowledgement.
Only outbound HTTPS connects the desktop and server. Plain HTTP is allowed only for a numeric loopback address during local verification.
The client sends no browser cookies, follows no redirects, and issues no local HTTP listener.

The device credential permits control and source submission for one recording only.
It grants no catalog, execution, project administration, or ordinary application session access.
Current owner permissions and the original twenty-minute deadline remain required.
The companion stops its owned recorder after transport failure, authority withdrawal, cancellation, or deadline.
Captured source stays in the private desktop directory when delivery fails.

The default journal directory is TestStudio/recordings beneath the operating system's user cache directory.
Connection files use mode 0600. Directories use mode 0700.
An operating-system lock permits one active companion per directory and releases when its owner exits.
Captured source and recorder diagnostics remain under captures/{session}. Diagnostics do not enter API responses or ordinary command output.
This initial command does not automatically purge desktop recordings. Remove completed private session files when local retention policy permits.

## Tutorials

### Prepare the desktop

Use a trusted TestStudio checkout. Build the command and install the locked recording dependencies:

```sh
go build -o .data/bin/teststudio-recorder ./cmd/teststudio-recorder
npm ci --prefix runners/playwright
node runners/playwright/node_modules/playwright/cli.js install chromium
```

Run the command from the checkout directory. The defaults use its trusted CLI and recording helper.
The CLI version must match the product's locked Playwright version, currently 1.64.1.
Separate distribution and first-time installation guidance remain part of the guided workflow work.

### Connect a hosted session

An operator enables --recording-companion on the API service without configuring the server's native desktop recorder.
The managed authoring API reserves a session and issues its owner-private pairing code.
The normal wizard does not yet expose this step. The API routes are documented in [browser recording](browser-recording.md).

Start the command with the trusted server origin:

```sh
.data/bin/teststudio-recorder --server https://studio.example.test
```

Paste the connection code at the prompt. Use the dedicated Playwright windows to record actions and assertions.
The companion submits bounded source updates and checks control requests every two seconds.
An owner stop request closes the recorder and submits its final source. Cancel retains the server's cancellation intent.
Return to TestStudio to review generated code. Saving, preparation, publication, and execution remain separate actions.

### Recover an interrupted session

The command prints the recording identifier without printing its credentials.
Run the command with that identifier to recover its original server connection:

```sh
.data/bin/teststudio-recorder --resume session-identifier
```

Resume uses the original private journal. An optional --server must match its original server exactly.
An uncertain browser launch is never replayed. Recovery waits for the old trusted helper to close before reading retained source.
Exact pending snapshots retry with their original sequence. A lost final acknowledgement does not replace terminal source.
Expired or revoked authority prevents further delivery. The private local source remains available for manual review.
Start a new recording session when another browser recording is needed.

Ctrl+C stops the owned recorder and retains local source. Resume can deliver its retained failure snapshot while authority remains valid.
The command preserves unrelated browser windows.

### Advanced settings

Use --directory to select another private recording directory.
Use --playwright-cli and --helper only for operator-approved installed files.
The companion never takes executable paths from a pairing response or uploaded source.

## Current verification

Focused local checks cover connection restrictions, redirect refusal, private files, directory locking, ordered retries, lost acknowledgements, restart, revocation, and deadlines.
The command integration uses the real native process driver with an authored Node helper fixture and loopback HTTP server.
That check confirms private journaling, source submission, owner stop, credential-free output, helper termination, and lock release.
It does not open Chromium or prove the full wizard workflow. Full hosted recording acceptance remains open under item 57.
