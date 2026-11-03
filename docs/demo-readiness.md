# Local demonstration delivery

The first delivery presents a usable local TestStudio application.
The complete product plan and 57 acceptance items remain the later delivery scope.
The current acceptance ledger has 26 verified items. It measures full product requirements rather than the size of this demonstration.

## Existing application

The service and UI build independently from MarketMaker and sibling checkouts.
The local server uses SQLite and starts without cloud credentials.
The current README starts the application with a UI build and go run command.
Local sign-in, project registration, catalog editing, publication, and native HTTP execution are implemented.
Run details, logs, evidence, timing, stacked trends, failure reruns, help, and themes are implemented.
Go and Playwright source preparation, normal execution, and retained artifacts pass local checks.
Browser-confirmed CLI sign-in and retained release comparison and rollback are also implemented.

## Immediate checklist

- [ ] Provide one local start and stop command with an explicit, separate demo data directory.
- [ ] Provide an explicit demo setup action with new generic data and a local target fixture.
- [ ] Include pass, failure, and timeout scenarios, plus a browser example with screenshot evidence.
- [ ] Document a short walkthrough: catalog, publish, run, result, evidence, rerun, timing, and trends.
- [ ] Run one targeted local demonstration check from a clean demo installation.

Reuse trusted runtime images and unchanged build outputs.
Run only the demonstration workflow and checks required by changes to its setup.
Keep installation empty by default. Load demo data only through its explicit setup action.
Use no MarketMaker tests, fixtures, credentials, endpoints, or history.
Clean up only processes and resources created by the demonstration check.

## Later delivery

AWS, Kubernetes, and Azure deployment verification follows the local demonstration.
Additional dependency preparation profiles, database transfer, broader recovery checks, and remaining authorization and evidence requirements remain on the full ledger.
They do not gate this initial local demonstration.

## Current worktree

Suite settings, archive, restoration, affected schedule visibility, and exact suite management checks are in progress.
The isolated server and UI builds pass with these changes.
Their targeted functional checks remain pending. Preserve the edits and avoid adding more scope to them before the demonstration.
