# Browser recording without tester installation

The user approves an in-page recording experience for testers who need simple instructions.
Testers use their existing browser. Operators install and maintain the recording runtime.
The desktop companion remains available where explicitly configured.

## Tester journey

1. Choose a suite and enter the test title and website address.
2. Select Start recording. Use the website shown inside TestStudio.
3. Select Check something appears, then select an expected story or other item.
4. Select Finish recording. Review the plain-language captured steps.
5. Select Save test draft. Advanced source editing remains available.
6. Prepare the draft, review its publication and target, then explicitly run it.

Saving does not publish or execute. Preserve review, revision conflicts, and partial-save recovery.
A future combined Save and run action must explain publication and execution and retain successful intermediate receipts.

## Milestones

| Milestone | Acceptance numbers | Smallest useful verification |
| --- | --- | --- |
| 1. In-page local proof | 21, 49, 50 | A real browser records a generic News click and a story visibility assertion. The tester reviews and saves without a source editor. Reload, help, focus, light, dark, narrow layouts, and cleanup pass through normal TestStudio execution. |
| 2. Isolated hosted recording worker | 12, 13, 45, 51 | An authenticated session reaches only its owned private worker. Current authority, archive, expiry, disconnect, capacity, target policy, shutdown, and recovery pass against the actual worker. |
| 3. Hosted preparation and execution | 22, 42, 51, 56 | The saved recording prepares, publishes, runs against its reviewed environment, and exposes private results and evidence on MMDev. |
| 4. Tester acceptance | 21, 49, 50, 56, 57 | A tester completes the selected News journey on the actual target from an ordinary browser with no installed companion or development tools. |

Milestone 1 has local evidence. Milestone 2 is implemented and verified against the actual isolated Docker worker locally, including private TLS, PostgreSQL leases, and the normal News authoring workflow.
Milestone 2 target deployment and verification remain open. Milestones 3 and 4 remain open.
The ledger stays 50 verified of 57. These milestones refine stable criteria and do not close their broad cross-profile scope.
Azure remains deferred.

## Local proof

The API relays bounded frame and input requests to a trusted Node helper over private process pipes.
The helper runs a headless Chromium browser with its sandbox enabled and uses the exact pinned Playwright recorder and code generator.
Code and plain-language steps feed the existing owner-private recording history and draft-saving service.
The recording identifier also identifies its draft. A reload reopens the retained version. A lost save confirmation recovers an identical saved document without another draft or revision.
Frames are private, uncached JPEG responses. They are not stored in the database.
Input uses a monotonic sequence. An exact retry does not repeat a click. Changed duplicates fail.
The page stops input after an uncertain response and requires a refreshed view before the tester continues.
Finish waits for queued input. Cancel and the twenty-minute limit close the owned browser.
Popup windows, uploads, downloads, drag gestures, and recording-browser storage transfer are outside this first proof.
Typing, paste, scrolling, basic keys, and visibility checks have bounded controls. Keyboard focus can leave the frame.
The frame is a visual view; full semantic access to the remote website remains a separate accessibility requirement.

The helper uses pinned Playwright internal recorder interfaces. A runtime upgrade must repeat the focused native and UI checks.
This preview supports one recording browser per API process and private process-local recovery.
API restart ends the browser. Persisted recording history retains previously saved code and steps.
The preview requires explicit loopback-only local mode. Do not expose it as a hosted browser service.

Operators can use a separate local verification installation:

```sh
bin/teststudio --local --recording-embedded-cli runners/playwright/node_modules/playwright/cli.js \
  --recording-directory PRIVATE_RECORDING_DIRECTORY
```

The operator supplies the normal local addresses, private database, artifact paths, UI, and trusted runtime.
Use a separate dynamic port for verification. Preserve the user's port 8040 demo and private UI snapshot.
Testers do not run this command or install any runtime.

## Hosted worker requirements

The implemented [private worker](recording-worker.md) provides a dedicated daemon, mutually authenticated TLS transport, durable capacity and cleanup leases, fenced restart recovery, isolated browser containers, and a separate origin-restricted gateway.
The API uses --recording-worker-config or --recording-worker-config-env. Existing in-page review and saving remain unchanged.
The initial provider requires a dedicated Docker host. It is not installed by the current AWS templates.

Keep recording outside the API service and outside other application workloads.
Use one isolated browser per leased recording session, bounded capacity, a fixed lifetime, and an idle or disconnected-owner policy.
Use authenticated private transport; never expose Chromium debugging, control sockets, or recording frames publicly.
Bind each worker to the actor, project, suite, session, and original deadline. Recheck current authority on view and input.
Persist lease and cleanup references so API or worker restarts cannot orphan a browser or replay actions.
Preserve captured code and steps after failure. Expired sessions cannot resume interactive control.

Resolve a reviewed target before launch. Restrict network access through a trusted gateway and prevent access to metadata, internal control services, and unapproved origins.
The local preview's unrestricted operator-host networking does not meet this hosted requirement.
Record approved additional origins explicitly when a real target needs separate assets or sign-in origins.
The implemented initial policy permits only the entered origin. Additional-origin configuration remains a target-specific extension; do not silently broaden the policy.
Keep recording credentials separate from the test's execution credentials.

Build the worker package from committed implementation source. Verify the selected MMDev account before mutations.
Inspect the existing deployment and review a concrete TestStudio-only change set before execution.
Core health does not prove recording, preparation, or execution readiness.

## Evidence

Normal STUDIO-EMBEDDED-RECORDER-001 runs through TestStudio's source preparation, publication, environment-backed execution, and private evidence contracts.
Its fixture records a fresh generic website. It creates no test against a user's website or new user run.
Normal STUDIO-HOSTED-RECORDER-001 repeats that journey through the actual isolated runtime and private TLS worker.
Run 20261108T165640.285709000-e4e7e064e323488114590eb5 passes in 5.997 seconds; the full focused harness takes 21.56 seconds.
Two recording browsers, two gateways, their volumes, and all three observed host processes are removed. Receipts are output/playwright/private-recording-worker.
See [self-testing](self-testing.md) for the focused harness and retained evidence.
