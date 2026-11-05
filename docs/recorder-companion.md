# Desktop recording companion

## Introduction

The desktop companion connects a hosted TestStudio recording session to Playwright on the tester's computer.
It opens only its dedicated recording browser and Inspector. It does not execute uploaded test source.
The first command implementation targets macOS and Linux desktops.
The guided connection wizard and real Chromium capture now pass focused local verification.
Operators can now provide a standalone recorder archive. Testers need Node.js and the package; they do not need Go or a checkout.

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

Get the archive for your desktop from your trusted TestStudio operator. Extract it and open a terminal in its directory.
Install Node.js 22 or later, including npm. Run:

```sh
node setup-recorder.mjs
```

Setup installs the locked Playwright dependencies and Chromium. Linux desktops can need operator-installed browser system libraries.
Run the command from the extracted package directory. Keep its scripts and runners directories beside the binary.
The CLI version must match the product's locked Playwright version, currently 1.64.1.
The wizard includes First-time desktop setup and Advanced connection and recovery disclosures.
A one-click installer and automatic updates remain optional later improvements.

### Connect a hosted session

An operator enables --recording-companion on the API service without configuring the server's native desktop recorder.
The Kubernetes chart uses recordingCompanion: true. The AWS core stack uses RecordingCompanion=true.
Both options default to disabled. They require a service image built from the updated source and do not add a server-side browser.
In Create test, choose Playwright browser test and a destination suite.
Enter a title and start address. Use Advanced recording details only to change the generated identifier.
Select Continue to desktop connection. The wizard reserves a session and shows its owner-private pairing code.
The API routes are documented in [browser recording](browser-recording.md).

Start the command with the trusted server origin:

```sh
./teststudio-recorder --server https://studio.example.test
```

Paste the connection code at the prompt. Use the dedicated Playwright windows to record actions and assertions.
The wizard advances to Record when pairing succeeds. It clears the connection code at that transition.
The companion submits bounded source updates and checks control requests every two seconds.
An owner stop request closes the recorder and submits its final source. Cancel retains the server's cancellation intent.
Return to TestStudio to review generated code. Saving, preparation, publication, and execution remain separate actions.

### Recover an interrupted session

The command prints the recording identifier without printing its credentials.
Run the command with that identifier to recover its original server connection:

```sh
./teststudio-recorder --resume session-identifier
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

## Operator packaging

Build an archive from a trusted checkout:

```sh
node scripts/package-recorder.mjs
```

The default target matches the current desktop. Options select darwin or linux and arm64 or amd64.
Use --output to choose the parent output directory. Each build creates a fresh directory without overwriting another package.
Cross-compilation produces a package only. It does not prove native browser behavior on the selected target.

The archive contains the recorder binary, trusted recording helper, locked dependency declarations, setup script, README, and manifest.
The manifest records each distributed file checksum, compiler, source revision, and dirty source state.
The archive has a separate SHA-256 checksum. macOS metadata files do not enter the archive.
Distribute the archive and checksum through your trusted operator channel. No package is published automatically.

Verify the archive before distribution:

```sh
python3 scripts/verify-recorder-package.py PATH_TO_ARCHIVE
```

The verifier checks archive paths, file types, exact contents, checksums, helper and dependency bytes, executable permissions, and native command help.
Add --offline-setup on a matching desktop with cached npm dependencies and Chromium to check setup without downloads.
Temporary verification files are removed. The verifier writes a verification JSON beside the archive.
Other operating system and architecture packages require their own native setup and browser verification.
The native macOS ARM64 package passes these checks in output/recorder/package-pOD1pj.

## Current verification

Focused local checks cover connection restrictions, redirect refusal, private files, directory locking, ordered retries, lost acknowledgements, restart, revocation, and deadlines.
The command integration uses the real native process driver with an authored Node helper fixture and loopback HTTP server.
That check confirms private journaling, source submission, owner stop, credential-free output, helper termination, and lock release.
It does not open Chromium or prove the full wizard workflow. Full hosted recording acceptance remains open under item 57.

The separate TestHostedRecorderCompanionCompletesGuidedUIAndNativeCapture check now passes the complete guided workflow with real native Chromium.
TestStudio prepares, publishes, and executes that newly authored Playwright UI test through its normal local runtime and trusted gateway.
The test checks generated identifiers, code rotation, reload recovery, help, focus, both themes, and mobile overflow.
The native recorder captures the independent local target's initial navigation. Owner stop returns the source for review and draft saving.
No target test run starts. The check confirms that sixteen owned processes exit and all owned execution resources are removed.
Evidence is output/playwright/recorder-companion-help-fix. The check completes in about 27 seconds.
Standalone macOS ARM64 packaging and offline setup now pass. Actual cloud profile verification remains separate work. Acceptance 57 remains open.

The opt-in Kubernetes and AWS settings now pass focused local packaging checks.
Kubernetes validates three profiles, including separate controller and preparation roles, and rejects two malformed settings.
AWS validates four task profiles and rejects three malformed settings. HTTPS authorization forwarding and disabled caching remain required.
A native service subprocess check confirms default-disabled and explicitly enabled API modes. Enabled recording waits for pairing without opening a browser.
These settings have no new deployed-profile evidence. See [Kubernetes](kubernetes.md) and [AWS](aws.md) for operator configuration.
