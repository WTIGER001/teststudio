# Suite settings and lifecycle

A suite owns permanent test identities and retained versions within one application. A new application includes an empty Default suite.
Its identifier stays fixed. Display settings can change without editing test definitions.

## Interface

Open **Project settings**, choose **Suites**, and select **Inspect suite**.
The panel shows the permanent identifier, current version, lifecycle, and current management authority.
Edit the name, description, or target requirement. Existing shared or unassigned suites also need one reviewed application owner. Select **Save suite settings**.
The update preserves capture policies, timing policies, and the current version.
Saving suite settings needs no publication step and never starts tests. Tests saved in this suite become current automatically when ready.

Read **Affected schedules** before archiving a suite.
The list includes only schedules you can read. It preserves pinned release identifiers.
Save or reload unsaved settings before changing the lifecycle.
Select **Archive suite** to block new run dispatch and new attempts.
Existing run snapshots, results, artifacts, releases, and schedule pins remain retained.
Already executing attempts use the normal completion and evidence flow. Archive does not cancel them.
An enabled schedule remains enabled. Its dispatch checks the archived suite and records the normal blocked firing.
Select **Restore suite** to reuse its identifier and current version.
Restoration does not start a run. An enabled schedule can dispatch at its next eligible firing.

Conflicts require **Reload suite settings** before another save or lifecycle change.
Reload replaces the local form with the current stored values.
Contextual help explains these operations on the Project settings page.

## API

Use these routes below `/api/v1/projects/{project}/suites/{suite}`:

| Method | Route | Result |
| --- | --- | --- |
| GET | `/lifecycle` | Read settings, revision, authority, and readable affected schedules. |
| PATCH | `/settings` | Replace editable display settings and target requirements. |
| POST | `/actions/archive` | Archive the suite and preserve retained history. |
| POST | `/actions/restore` | Restore the same suite and current version. |

Settings accept `name`, `description`, `applications`, and `requiresTarget`.
Applications contains exactly one registered identifier. The owner is permanent after assignment. Existing shared or unassigned suites require review; incompatible current tests, probes, and schedules block assignment. Names and descriptions have bounded lengths.
Mutations require Origin, session protection, and an explicit `If-Match` revision.
Read and management permissions apply to the exact suite object.
Permission checks, mutation, audit, and response construction share one transaction.
A denied read or management check cannot commit a settings or lifecycle change.
Suite listings omit suites denied by an exact read rule.
The existing suite PUT route also applies exact suite read and management authority.

## Local verification

Run the focused API check:

```sh
go test ./internal/httpapi -run '^TestSuiteLifecycle' -count=1
```

Run only the affected Playwright workflow through TestStudio:

```sh
bash scripts/test-self.sh --test=STUDIO-SUITES-001
```

The checks cover revision conflicts, exact denials, schedule visibility, archive, restore, retained run evidence, and release pins.

## Tester navigation

Choose Suites in the Testing sidebar section.
Suite cards provide View tests, Create test, and Suite settings actions.
View tests filters the application catalog. Create test carries the selected suite through the type chooser.
New suite opens the shared Name, Options, and Review registration wizard.
Settings reuse the existing suite manager and lifecycle controls.
Unsaved creation or settings changes require a choice before route navigation.
