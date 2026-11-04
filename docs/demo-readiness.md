# Local demonstration delivery

The first delivery presents a usable local TestStudio application.
The complete product plan and 57 acceptance items remain the later delivery scope.
The current acceptance ledger has 28 verified items. It measures full product requirements rather than the size of this demonstration.

## Existing application

The service and UI build independently from MarketMaker and sibling checkouts.
The local server uses SQLite and starts without cloud credentials.
The current README starts the application with a UI build and go run command.
Local sign-in, project registration, catalog editing, publication, and native HTTP execution are implemented.
Run details, logs, evidence, timing, stacked trends, failure reruns, help, and themes are implemented.
Go and Playwright source preparation, normal execution, and retained artifacts pass local checks.
Browser-confirmed CLI sign-in and retained release comparison and rollback are also implemented.

## Immediate checklist

- [x] Provide one local start and stop command with an explicit, separate demo data directory.
- [x] Provide an explicit demo setup action with new generic data and a local target fixture.
- [x] Include pass, failure, and timeout scenarios, plus a browser example with screenshot evidence.
- [x] Document a short walkthrough: catalog, publish, run, result, evidence, rerun, timing, and trends.
- [x] Run one targeted local demonstration check from a clean demo installation.

Run `node scripts/demo.mjs start` from the repository directory.
Follow [the demonstration guide](demo.md).
Use Ctrl+C or `node scripts/demo.mjs stop` to stop the demonstration.
The verified walkthrough installation remains in `.data/demo-review` with its recorded history and catalog edit.
Use `--data .data/demo-review` to resume that installation.

The fresh setup produces three HTTP runs and one passing browser run through normal API contracts.
The UI walkthrough edits and publishes a catalog, starts a run, previews its screenshot, and reruns failures after target recovery.
The rerun contains one pass and one timeout. It excludes the already passing healthy check.
The healthy check has three comparable timing observations before the manual run.
The stacked area chart displays the resulting states. The application browser console has no errors or warnings.
A restart preserves all six runs, the catalog edit, and available screenshot metadata without duplicate setup runs.
An occupied saved API port causes startup to stop before server startup or seeding.
Evidence is `output/playwright/demo-checkpoint.json`, `demo-port-conflict.json`, the workflow snapshots, and the screenshot captures.
The owned browser session and demo processes stop after verification.

Reuse trusted runtime images and unchanged build outputs.
Run only the demonstration workflow and checks required by changes to its setup.
Keep installation empty by default. Load demo data only through its explicit setup action.
Use no MarketMaker tests, fixtures, credentials, endpoints, or history.
Clean up only processes and resources created by the demonstration check.

## Later delivery

AWS, Kubernetes, and Azure deployment verification follows the local demonstration.
Additional dependency preparation profiles, database transfer, broader recovery checks, and remaining authorization and evidence requirements remain on the full ledger.
They do not gate this initial local demonstration.

## Follow-up implementation

Suite settings, archive, restoration, affected schedule visibility, and exact suite authority checks now pass focused local verification.
Project settings exposes these controls. See [Suite settings](suites.md).
This follow-up does not change the demonstration setup or add a demonstration gate.

## Next usable workflow

The completed demonstration checklist remains complete. It does not prove all ordinary-user or hosted workflows.
Project creator access now passes ordinary-user setup and scoped-boundary checks.
The first environment-backed local recorded website flow now passes preparation, publication, execution, and private evidence collection.
The demo enables the trusted single-origin gateway. The offline starter keeps its separate isolated profile.
Tester investigation and failure recovery now pass scoped evidence, accessible mobile presentation, and rerun checks without duplicate children.
Acceptance 44 is complete. Private trace viewing and remaining catalog workflows are next.
Follow [the active delivery plan](plan.md#active-delivery-plan-after-the-tester-walkthrough) for bounded batches and required evidence.
Keep the running demonstration available while these workflows are implemented.
