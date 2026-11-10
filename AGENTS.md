# TestStudio agent instructions

## Start each session

1. Confirm this checkout with `pwd`, `git status --short`, and `git branch --show-current`. Preserve unrelated edits.
2. Read [continuation notes](docs/continue.md), the active sections at the start of [the plan](docs/plan.md), and the acceptance section in [the ledger](docs/implementation.md).
3. Read the affected feature guide and source before editing. Read [UI rules](docs/ui.md) for UI work and [versioning](docs/versioning.md) before a release.
4. Choose one concrete milestone. State its acceptance numbers and the smallest useful local verification.
5. Update the user when a milestone completes. Distinguish implemented behavior, local evidence, deployment checks, and unverified target acceptance.

Historical plan checkpoints are reference material. They do not override the active delivery sequence or later user instructions.
Use focused reads and searches. The long implementation ledger is not a current task queue.
This file applies to TestStudio. Read separate repository instructions before editing another checkout.

## Product boundaries

- Build the independent Go and Angular product in this repository. It must not require MarketMaker packages, configuration, credentials, sibling checkouts, or existing application tests.
- Write new product tests, fixtures, and examples. Normal installation starts with empty testing registries. Examples require explicit selection.
- The user authorizes adapting the separate marketmaker-tests repository. Keep its application-specific adapter and existing tests there.
- PostgreSQL is the hosted database. SQLite supports single-instance development and local-server operation.
- Use `github.com/wtiger001/go-permissions` through `permissions.Service`. TestStudio owns its internal identities and memberships.
- Parse resource context before authorization. Enforce permissions inside handlers or domain services. Recheck current authority for mutations.
- Keep local accounts and configurable OIDC generic. Upstream federation belongs to the issuer. Add no Google-specific application flow.
- Successful sign-in grants no project membership. Do not merge identities by email or broaden installation privileges to solve project access.

## Implementation and UI

- Use standalone Angular components, OnPush, signals, small services, and pure utilities. Keep components below 450 lines.
- Components handle rendering and local interaction. Services own APIs, state, workflows, and side effects. Pure utilities handle transformations and validation.
- Use shared components and semantic HTML. Preserve light, dark, and system themes, keyboard focus, accessible feedback, and narrow layouts.
- Follow docs/ui.md. Use dedicated routes for substantial creation, labeled steppers for related decisions, and named Advanced sections for uncommon controls.
- Keep project and application context explicit. Suites belong to exactly one application. Create an empty Default suite with each new application. Existing shared or unassigned suites require reviewed assignment. Environments belong to applications.
- Every current page needs a right-hand help drawer with Introduction, Concepts, and Tutorials. Help must preserve edits.
- Preserve review, unsaved changes, revision conflicts, explicit execution, and partial-save recovery. Do not replace them with parallel workflows.
- Record test durations and preserve frozen timing criteria. Timeouts remain a distinct result status; failure reruns include them. State trends use stacked area charts.
- Project images are optional and private. URL screenshots use browser-approved tab capture, not automatic server rendering. Stop sharing tracks and retain the upload fallback.
- Add no code comments unless the user requests them. Use short, active sentences in documentation.

## Implementation first; targeted verification

- Implement the requested behavior, then run the smallest relevant check. Use local targets for functional testing.
- Use TestStudio's normal Playwright execution for changed browser workflows. Select explicit IDs; do not run the full catalog by default.
- Reuse passing evidence, cached assets, binaries, and runtimes. Repeat a check only for changed behavior, a failure, or a concrete unresolved concern.
- Do not run broad test matrices, rebuild unrelated fixtures, or repeat setup to fill time. Targeted checks must exercise meaningful outcomes.
- Documentation-only work needs consistency and link checks. It needs no UI build, browser run, release bump, or deployment.
- Close and verify shutdown of owned browsers, automation processes, containers, and test servers after success or failure. Preserve user browsers and other workloads.
- Preserve the user's demo on port 8040, its database, and its history. A UI build alone does not replace the demo's private UI snapshot.

| Changed area | Starting command |
| --- | --- |
| Go package behavior | `go test ./internal/<package> -run '^TestRelevantName$' -count=1` |
| UI compilation | `npm --prefix web run build` |
| Focused browser workflow | `bash scripts/test-self.sh --test=STUDIO-ID` |
| Version utilities | `node --test scripts/lib/version.test.mjs` |
| Stamped local package without Docker | `node scripts/build-selftest.mjs --local-only` |
| Portable service package | `node scripts/build-service-image.mjs --arch=arm64` |

See docs/self-testing.md and tests/selftest/catalog.json before choosing browser IDs. Build commands are not deployment commands.

## Deployment authorization and limits

- AWS deployment is authorized in MMDev as teststudioapi. Use mm_dev, account 253455159142, region us-west-2, cluster marketmaker-testing-dev.
- Verify the selected account before mutations. Scope changes to TestStudio-owned resources. Preserve the existing testing-server and other services.
- Review a concrete CloudFormation change set before execution. Preserve in-progress deployments; inspect them instead of starting duplicates.
- Follow docs/aws-mmdev.md and the relevant worker package guide. Core health does not establish coded-worker or source-preparation readiness.
- mm_prod access is limited to the user-selected external Cognito pool us-west-2_p1Epu0go8 and its dedicated TestStudio OIDC client. Preserve other clients and upstream federation.
- Never print passwords, tokens, cookies, secret payloads, or guarded origin headers. Use secret references and private temporary credential files; remove owned files afterward.
- Kubernetes verification uses only the dedicated local k3s/k3d installation with its explicit private kubeconfig. Do not use inherited or remote cluster contexts.
- Azure implementation and verification remain deferred. Preserve staged code. Prepare an Azure deployment script only when requested. Do not resume historical Azure inspector work.
- Keep local implementation checks and target deployment checks separate. Never infer cross-profile completion from mocks or one healthy native HTTP run.

## Releases, documentation, and Git

- VERSION is the product release source. Follow docs/versioning.md and use scripts/version.mjs before publishing changed application behavior.
- Build release artifacts from committed implementation source. Use the shared generated metadata for UI, API, and image labels.
- Keep build creation, deployment, and server startup dates distinct. Do not hand-edit generated build metadata.
- API contract and dependency versions remain separate from the product version.
- Update affected help, feature guides, API contracts, and operator instructions with behavior changes. Follow the plan's workflow completion checklist.
- Track all 57 stable acceptance numbers in docs/implementation.md. Report verified totals and the item numbers advanced. Keep broad criteria open until their complete scope passes.
- Update docs/continue.md when priorities, release state, deployment state, known gaps, or resume instructions materially change.
- Commit verified features frequently; the user already authorizes these local commits. Do not push without an explicit request.
- Keep secrets, local databases, generated metadata, temporary helpers, and raw evidence out of Git. Important resume knowledge belongs in tracked documentation.
