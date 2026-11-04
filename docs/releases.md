# Retained suite releases

The Suite releases page lists immutable catalogs retained for a project suite.
Open it through project navigation or Browse retained releases on the catalog page.
Choose a suite, then inspect a release.
Search by release identifier, author, repository, or commit.

## Inspect and compare

Selected release shows its publication actor, time, test count, execution image, and recorded source provenance.
Expand Published test definitions to inspect permanent identifiers, types, safety labels, and lifecycle states.
When GitHub provenance exists, View recorded commit opens that exact commit.
It does not follow the current branch head.

Compare selected release against chooses a different retained release as the baseline.
The comparison lists added, removed, and changed definitions by permanent test identifier.
Changed fields include nested assertions, source selectors, safety settings, timing rules, and evidence policies.
Map key order alone does not create a definition change.
Separate notices identify changes to execution artifact identity, source provenance, suite timing policy, and suite capture policy.
Review recorded source and test results when artifact identity changes.

## Activate or roll back

Inspect a retained release and select Activate selected release.
This action selects it for future runs that follow the active suite release.
Selecting an older release rolls back the suite without rewriting current catalog definitions.
It does not execute tests, rebuild images, or create another release.
Current test-state, dependency, safety, and execution admission checks still apply to future dispatch.

Activation checks current suite-scoped publication authority inside the shared permission transaction.
It uses the displayed suite revision. Concurrent suite changes return an explicit conflict.
Refresh retained releases, review the new state, and select the intended release again.
The audit records the previous and selected release identifiers.

## Frozen runs and pins

Run creation freezes its selected release, definitions, artifact, and policies.
Publication, activation, and rollback do not rewrite existing run snapshots.
A future unpinned run uses the active release at creation time.
An explicit run release identifier selects that retained release even when another release is active.
A schedule release pin remains unchanged when the suite active release changes.
Schedule configuration controls whether future firings follow the active release or retain a specific pin.

## Archive and authority

Archived suites retain readable release history and comparisons.
Their activation controls remain unavailable until the suite is restored.
Catalog read permission protects release inspection.
The page uses the server's exact suite-scoped activation decision.
Read-only users can inspect releases without receiving activation authority.
Release identifiers from another project or suite cannot activate through this registry.

Open a published test title to inspect its [retained history](test-history.md), including archived suites.
The history page labels current definitions separately from original run snapshots.

## Local verification

STUDIO-RELEASES-001 exercises comparison, a stale activation conflict, refresh, activation, rollback, frozen run definitions, future selection, and release pins.
It also verifies archived-suite release history, contextual help, keyboard dismissal, dark mode, and mobile width.
Focused Go checks cover retained ordering, revision conflicts, cross-project rejection, archived history, and exact publication denies.
Pure comparison checks cover permanent identities, artifact changes, nested key ordering, and immutable commit links.
The common implementation remains subject to the separate deployment-profile acceptance checks.

## Review before publication

The catalog now compares a new draft with its active release before publication.
Publish release leaves activation unchanged. Publish and activate selects the new release for future unpinned runs.
See [Publication review](publication-review.md) for validation, schedule effects, stale conflicts, and lost-response recovery.
