# Review and publish a catalog

## Review changes

Save test edits in the catalog. Select the suite under Publish a release, then select Validate and preview.
The preview compares that draft with the suite's current active release.
It lists added, changed, and removed definitions. Unchanged definitions remain in the summary count.
Changed fields identify metadata, assertions, dependencies, lifecycle, source, capture, and timing edits.
Safety warnings identify label, rationale, or operating-condition changes.
Expand Before and after definition to inspect complete changed definitions.
Advanced review details shows the prior and proposed release source, timing, and capture settings.
The first release has no active comparison. The preview identifies this condition explicitly.

Validation checks configuration. It does not execute tests or prove they pass.
Resolve all listed issues before publication. An outdated working draft requires a new draft and review.

## Choose the publication effect

Publish release creates an immutable release and leaves the active release unchanged.
Publish and activate creates the release and selects it for future unpinned runs.
Existing runs retain their original release, definitions, result, and evidence snapshots.
Starting a run remains a separate action.

Schedules you can inspect lists readable schedules for this suite.
Schedules without a release pin follow the active release on future execution.
Pinned schedules keep their selected release. Disabled schedules do not run until enabled.
Hidden schedules do not appear in the list. Publishing requires no new schedule permission.

## Conflicts and recovery

An unapplied review expires after 30 minutes and belongs to the user who creates it.
An applied review still supports same-key result recovery after that deadline, subject to current permissions.
Publication rechecks current read, source, and publishing authority.
It also checks the reviewed project, suite, draft, working catalog, active release, and schedule configuration.
An intervening change rejects publication before any release or activation commits.
Select Validate and preview again. Review the new changes before submitting another publication.
Routine schedule progress does not invalidate a review. Schedule configuration changes do.

If the publication response is unavailable, retry the same action on the same page.
The page retains its request key and effect. It disables the other effect and a replacement preview while the result is uncertain.
A committed publication returns the original release on retry. It does not create another release or test run.
Reloading loses this in-page continuation. Inspect Suite releases before publishing again after reload.
A rejected access request hides the review. Refresh access before preparing another review.

## API contract

Use these routes below `/api/v1/projects/{project}/suites/{suite}`:

| Method | Route | Result |
| --- | --- | --- |
| POST | `/catalog-drafts/{id}/publication-reviews` | Create an actor-owned review of this draft revision. |
| GET | `/publication-reviews/{id}` | Read the actor's review with current permissions. |
| POST | `/catalog-drafts/{id}/publish` | Publish the reviewed draft with explicit activation behavior. |

Review creation requires `If-Match` with the draft revision.
Publication requires that revision, `Idempotency-Key`, and `{ "reviewId": "REVIEW_ID", "activate": true }`.
Set activate to false for publication without activation.
Use the same review, revision, key, and activation value for retries.
The server rejects an applied review reused with another key or effect.
Expired unapplied reviews and changed or mismatched review inputs return a conflict.
Origin and CSRF requirements apply to both mutations.
The original publication request without reviewId remains available for existing trusted publication clients.
Prepared-source publication retains its own reviewed source workflow.

The review response includes changes, validation issues, settings before and after, readable schedule effects, and expiry.
Readers cannot retrieve another user's review through its identifier.
Exact test read denials protect both preview creation and later reads or publication.
Review reads refresh the readable schedule list. Publication also detects changed hidden schedule configuration without exposing its details.
OpenAPI 0.27 and the generated client describe the reviewed publication operations.
