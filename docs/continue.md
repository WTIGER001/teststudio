# Continue TestStudio implementation

Last reviewed: 2026-11-07. These notes describe the last verified state, not a live infrastructure inventory.
Read this guide before historical continuation commands elsewhere in the plan.

## Resume in this order

1. Read AGENTS.md and check the branch and working tree. The active branch is codex/implementation.
2. Read the active delivery sections at the start of [the plan](plan.md).
3. Read the Acceptance criteria section in [the ledger](implementation.md). It contains the authoritative 57 checkboxes.
4. Continue the latest user request first. Otherwise select one next milestone below and read its implementation before changing it.
5. Implement the milestone, run targeted local checks, and update the ledger and these notes with verified results.

Do not restart completed setup or broad acceptance matrices merely because this is a new session.
Missing ignored receipts require focused inspection or recovery. They do not prove the feature needs reimplementation.

## Current product and release

The application is usable locally and through the hosted UI. Complete product acceptance remains unfinished.
The count is **50 verified of 57**. Open items are **41, 46, 51, 53, 54, 56, and 57**.
UI improvements refine existing accepted items. They do not close the remaining cross-profile requirements.

| Item | Last verified value |
| --- | --- |
| Hosted UI | https://teststudio.devfoodmarketmaker.com |
| Product version | 0.2.0-dev; footer displays 0.2.0.dev |
| Build identifier | 6caa2b0ca508 |
| Build creation | 2026-11-06T16:02:07.885Z |
| Implementation commit | 6d2444fcb27a0d03af68c258e36da173adc38b77 |
| Source state at build | Committed source; modified is false |
| Service image digest | sha256:eb8ff64d74fcdc9eb06f2c5127bb8b202ee2029d39a0126dd9e0fa70e1651fdb |
| Live version metadata | GET /api/v1/version; public and no-store |

Later documentation and operator-tooling commits do not change the deployed implementation commit or its immutable build date.
The public API, UI bundle, and portable image labels match this release.
The local user demo is separate. Do not assume it runs the same release as the hosted installation.

## Latest local UI change

On 2026-11-07, HTTPS probe recent checks switch from checkmark tiles to compact colored vertical pills.
The local patch version is 0.2.1-dev. This change is not deployed; the hosted release table above remains authoritative for the last recorded publication.
The UI build and STUDIO-PROBE-OVERVIEW-001 pass. Run 20261107T153825.676380000-f359987ef19bbbc8498e69d0 verifies 30 checks, keyboard links, light and dark themes, and a 390-pixel layout.
Evidence is complete, execution is cleaned, and owned processes exit. The total stays 50 verified of 57; items 34, 49, and 50 are refined.
Use the committed patch source if publication is requested. Do not repeat the passing workflow merely to publish it.

## Recent completed work to preserve

- HTTPS probes now has target-specific results, reviewed single and bulk actions, explicit adoption, multiple-address setup, and shared-component visual help.
- The user-selected FoodMarketMaker project contains nine private configured probes across three applications and stages. Initial normal hosted runs all pass.
- Nine five-minute schedules are enabled. All nine first automatic firings pass with complete evidence.
- Normal project roles grant the existing requesting user access. Installation privileges remain unchanged.
- Project creation uses /projects/new. The project list has a Create project action and no redundant count.
- Project tiles are square and use bounded widths. Optional private backgrounds support upload and browser-assisted website tab capture.
- Image capture requires a browser sharing prompt. It takes one frame, stops sharing, and requires explicit Save image.
- Capture is not automatic server rendering of an entered URL. Unsupported browsers retain upload. It starts no test run.
- The sidebar footer shows version, build identifier, and UTC creation date. Build expands source details. Mobile Navigation reveals the footer.
- VERSION and scripts/version.mjs synchronize release numbers. Standard builders stamp the UI and API with one generated metadata snapshot.
- Generic OIDC supports the selected organization issuer. Cognito owns Google federation; TestStudio contains no Google-specific sign-in path.
- The organization and bootstrap administrator providers are both enabled. This explains the two SSO buttons; do not remove backup access silently.
- Requested project creation and existing-project access were granted through normal authorization. Do not repeat grants or infer access from sign-in.

See [versioning](versioning.md), [project images](project-images.md), [first installation](first-installation.md), and [MMDev operation](aws-mmdev.md).

## MarketMaker deployment entry point

On 2026-11-07 the user selects /Users/johnbauer/Dev/MarketMaker/api for the deployment launcher.
Its deploy-teststudio-dev.py pins mm_dev, account 253455159142, region us-west-2, stack teststudioapi, and cluster marketmaker-testing-dev.
The launcher delegates to this checkout's maintained publish-aws-core.py. It supports a source override and works from another working directory.
Six focused launcher checks pass. Its actual prepare invocation confirms 0.2.0-dev / 6caa2b0ca508 is already deployed with no newer runtime changes.
The current stack is UPDATE_COMPLETE. Both teststudioapi and testing-server have completed healthy rollouts. No duplicate build or deployment starts.
The README in the API checkout documents status, prepare, review, execute, finish, and per-release receipt selection.
This operator convenience changes no application release and closes no acceptance criterion. Continue the default queue below when requested.

## Next implementation priorities

### Latest request: scoped membership UI and MMDev release

The user keeps Project and Group, selects System, and requests planning for Owner, Contributor, Developer, Member, and Reader on 2026-11-07.
Member runs tests. Contributor sits directly below Owner, with scoped administration except resource deletion and adding, removing, or changing Owners. Guard indirect Owner changes through groups and custom policies too.
The [role plan](roles-plan.md) defines cumulative Project and Application roles, a separate System Admin, inheritance, shared-resource boundaries, owner safeguards, and reviewed migration.
Application-only roles require server authorization; existing application filters do not establish access isolation.
The user requests API implementation. Backend milestones 1 through 3 are implemented locally with focused SQLite and PostgreSQL evidence.
API contract 0.37.0 and the generated identity client cover scoped assignments, atomic standard-role replacement, capability and access explanations, owner status, exact user lookup, conversion previews, apply, and rollback previews.
Standard roles use native permission-service evaluation with server-resolved Application context. Full mixed runs and shared source mutation remain restricted to Project authority.
New ordinary project creators receive one Owner assignment. Existing role bundles and memberships stay unchanged until reviewed conversion.
The [access guide](access.md#standard-project-and-application-roles) documents routes, previous-role revision guards, Owner safeguards, and migration recovery.
The retained PostgreSQL selections pass 59 initial checks, 18 focused follow-up checks, and two final delegation/Owner guard checks. Matching SQLite and targeted race checks pass. Reuse these receipts under output/roles-api unless changed behavior requires another check.
The user then requests the necessary membership UI, commit, push of all branch commits, and MMDev deployment.
Project members and Application members now support reviewed standard-role creation and replacement, inherited access, Contributor restrictions, ownership status, conversion receipts, and reviewed rollback. System labels, exact application explanations, and contextual help are included.
The UI build, eleven focused utility checks, STUDIO-ROLES-001, STUDIO-PERMISSIONS-001, and new STUDIO-APPLICATION-ACCESS-001 pass locally. The ledger records timings and retained runs. All owned controllers and execution containers exit.
Version 0.3.0-dev is prepared for publication. Release/target milestone 5 remains active. Existing hosted memberships and the nine private schedules remain unchanged until explicit reviewed edits. Preserve the port 8040 demo.
The count stays 50 verified of 57. The default AWS worker queue remains available after this planning request.

### Completed request: HTTPS probes features and help

The [HTTPS probes guide](https-probes.md) documents the delivered workflow. The [six-milestone plan](https-probes-plan.md) records implementation and verification evidence.
Milestones 1 through 6 are complete for this request. Local workflows pass and release 0.2.0-dev is deployed from committed source 6d2444f with matching public UI and API assets.
The user selects a new FoodMarketMaker project. Its nine explicit probes have passing normal runs, complete evidence, and enabled five-minute schedules.
Private evidence download and the requesting user's project roles pass inspection. All nine first automatic schedule firings pass with complete evidence.
Return to the default AWS worker queue below. Do not repeat onboarding, initial runs, or local workflow tests.
Important receipts are output/aws-mmdev/https-probes-release.json and foodmarketmaker-probes.json. They contain run checkpoints and publication metadata, not credentials.
Use the tracked publish-aws-core.py and manage-https-probes.py helpers. The latter derives schedule execution location from the actual retained run.
Keep private domain configuration outside product defaults and self-tests. The acceptance count stays 50 verified of 57, with the same seven open items.

### Critical: complete the requested import and execution path

The MMDev core currently supports native HTTP checks. Separate AWS coded execution and source preparation are not installed.
Continue with one bounded worker milestone using [the staged worker package](aws-workers-package.md).
Inspect the deployed core and package inputs before proposing changes. Do not recreate the core or reuse another application's resources.
The staged sequence is prepare, verify private networking, connect, seal, and enable the authorized worker capabilities.
Keep actual target verification separate from local functional implementation checks. Preserve the existing core and other ECS services.

The existing application test repository also needs reviewed additional browser origins and a managed browser credential or fixture bridge.
These are shared product capabilities. Do not hardcode MarketMaker domains, sign-in providers, or credentials into TestStudio.
Current repository synchronization binds targetless suites. Environment-backed source bundles use the reviewed ZIP preparation path.
Read [test repository contracts](test-repositories.md), [browser execution](browser-execution.md), and [deployment readiness](deployment-readiness.md) before enabling these workflows.

### Important: keep contracts and acceptance evidence complete

The version and project-image endpoints are documented in their feature guides but are absent from contracts/openapi.yaml.
Reconcile those API families during the next contract documentation pass. The canonical contract already states that some families remain pending.
Do not describe its current coverage as complete or overwrite its separate contract version with the product version.

Core application publication and HTTPS onboarding now use tracked publish-aws-core.py and manage-https-probes.py tooling.
Replace other useful ad hoc helpers before relying on them for worker releases.
The ignored .data/aws-mmdev snippets remain convenience history. Use tracked templates and operator guides as the durable deployment contract.

| Priority | Acceptance item | Remaining scope |
| --- | --- | --- |
| 1 | 51 | Consistent execution status, cancellation, deadlines, artifacts, and recovery on remaining profiles. |
| 2 | 46 | Retention, quotas, pins, encryption, and identity behavior on remaining cloud evidence profiles. |
| 3 | 54 | Equivalent configured OIDC and project-permission scenarios on remaining profiles. |
| 4 | 41 | Common timing and timeout behavior across remaining runner and deployment profiles. |
| 5 | 56 | Generic coded suites, progress, history, reports, and trends on each required target. |
| 6 | 57 | Complete common scenarios across every required deployment profile. |
| 7 | 53 | Independent Azure installation and coded execution after the user ends its deferral. |

Azure remains deferred for implementation and verification. Preserve its staged packages without adding inspectors or account requirements.
Local Kubernetes already has retained execution, identity, failover, gateway, and recording evidence. Reuse it unless changed behavior requires another check.

### Nice to Have

Optional installer convenience, automatic updates, and unrelated dashboard polish follow required workflow gaps.
Explicit user feedback on usability takes priority over this default ordering. Keep each correction small and documented.

## Deployment boundaries and recovery locations

| Target | Required boundary |
| --- | --- |
| AWS core | Profile mm_dev; account 253455159142; region us-west-2; cluster marketmaker-testing-dev; service teststudioapi. |
| AWS resources | Stacks teststudioapi-foundation and teststudioapi; preserve existing testing-server and other workloads. |
| External Cognito | mm_prod only for the authorized dedicated TestStudio client in pool us-west-2_p1Epu0go8. Preserve existing clients and federation. |
| Local Kubernetes | Dedicated teststudio-k3s profile; explicit .data/local-k3s/kubeconfig.yaml; no inherited user context. |
| Local demo | Preserve port 8040, SQLite data, and retained evidence. Use the documented launcher lifecycle only when needed. |
| Azure | Deferred. No new implementation, deployment, or verification without a later user request. |

The last AWS update leaves both TestStudio stacks at UPDATE_COMPLETE and one healthy teststudioapi task with a completed rollout.
The existing testing-server retains one healthy task. These observations are recorded snapshots.
Verify account and current resource state before any subsequent mutation.

Tracked [MMDev instructions](aws-mmdev.md) describe credential references and stack parameters. They contain no credential values.
Local parameter references are in .data/aws-mmdev/core-parameters.json. Deployment receipts are under output/aws-mmdev.
Useful receipts include published-image.json, build-version-deployment.json, and build-version-publication.json.
The administrator credential remains in Secrets Manager at teststudioapi/initial-administrator. Never print it or place it in arguments.
Full CloudFront distribution configuration contains guarded origin headers. Query only needed public fields.
Temporary Docker login configuration must stay private and be removed. Preserve the user's Docker credentials and contexts.

## Separate application repository

The marketmaker-tests integration is committed locally as f233b55. Its working tree was clean at this review.
On this workstation it is at /Users/johnbauer/Dev/MarketMaker/marketmaker-tests. It is a separate repository with its own instructions.
Only local commits were authorized. Remote availability is not verified; do not push without an explicit request.

The native export contains 604 existing definitions: 505 Website and 99 FoodSearcher.
Local assembly and isolated discovery pass. This does not mean those application tests ran or passed.
The repository's npm run teststudio:bundle command produces a source ZIP and output/teststudio/bundle.json.
Read its docs/teststudio.md and TestStudio's [generic repository guide](test-repositories.md) before changing the adapter.
Do not copy its tests, credentials, browser state, or application configuration into this product.

## Targeted local verification and release

Choose the check that exercises the changed behavior. The commands in AGENTS.md are starting points, not a mandatory checklist.
Select browser IDs from tests/selftest/catalog.json and use the normal TestStudio execution path.
Do not rerun completed checks after documentation-only edits.

| Recent workflow | Retained local evidence |
| --- | --- |
| Dedicated project wizard | STUDIO-WIZARD-001; run 20261106T131801.535325000-755c948292cb022d7125d961; 7.533 seconds. |
| Square tiles, upload, real tab capture, and removal | STUDIO-PROJECTS-001; run 20261106T140014.789307000-4ac204a42ec7e6e01bf4e4e1; 1.688 seconds. |
| Shared UI/API version and footer | STUDIO-VERSION-001; run 20261106T141549.201293000-da61d7429e7ac5e9b3ab33b7; 0.532 seconds. |

Each recorded run has complete evidence and a cleaned execution ledger. Owned verification processes exit afterward.
Screenshots and summaries are under output/selftest/{runId}. These ignored files may be absent in a fresh checkout.
Project capture uses headed Chromium on an owned virtual display in its isolated test container. Other browser checks remain headless.

For changed application behavior, choose the version bump, update affected help and guides, verify locally, and commit the implementation.
Build the release from that committed source. A later documentation commit may record the deployed artifact without rebuilding it.
Publish matching UI and service artifacts. Verify build identity and health; do not rerun a broad functional suite on the hosted installation.
The [release guide](versioning.md) covers cache stability, canonical SemVer, four-part display, local stamped builds, and image labels.

No cloud or browser work is required merely to resume this plan. Start with the selected implementation task.
