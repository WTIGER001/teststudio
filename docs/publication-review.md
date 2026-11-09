# Save tests and retain versions

## Normal workflow

Save tests directly. Saving updates future runs automatically when the source is ready. Tests, suites, and sources need no separate publication or activation step.
Recordings and Playwright source appear immediately in the catalog. Background setup freezes saved source revisions and makes the current version available automatically. Getting ready means setup is incomplete; the source stays saved even without an installed worker.
Native HTTP definitions are ready after a valid Save. Source imports retain their review before Save tests. Successful ZIP, image, and repository setup saves the current version automatically.
Saving never starts a test run. Run and Save and run remain explicit execution choices under current run authority.

## History and schedules

Internal immutable versions preserve earlier definitions, artifacts, timing, and capture settings. Existing runs and pinned schedules retain their original versions. Unpinned schedules use the current version on their next eligible firing.
Use retained versions to inspect, compare, or deliberately restore an older version. This recovery action does not execute tests.

## Conflicts and recovery

Save rechecks current authority and the expected revision in the same transaction. A rejected save leaves existing definitions and versions unchanged. Preserve local work and inspect the current revision before retrying.
An interrupted creation confirmation reads the saved definition and exact saved-version link before retrying. Explicit run retries retain their request key and recover the same run.
Source setup rejects superseded saved revisions. Setup failure retains source, catalog entries, and the previous working version.

## API compatibility

The direct command is POST /api/v1/projects/{project}/suites/{suite}/tests/{test}/save with If-Match. It returns the saved record and savedVersionId when ready. Legacy catalog-drafts, publication-reviews, publish, and activate endpoints remain compatible for existing integrations. Their internal names do not add steps to normal UI workflows.
Lifecycle actions and catalog import apply use save=true for atomic current-version updates. Playwright source saves use saveAsTest=true and applicationId. Background preparation and immutable publication remain service implementation details.
Status-page definitions use enabled. Save with enabled=true updates the public snapshot atomically. Save with enabled=false withdraws public access. Legacy omitted enabled retains its API behavior. Current statuspages.publish authority is still required to expose or update public content.
