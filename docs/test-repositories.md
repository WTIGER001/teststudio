# Test repository contract

## Introduction

Keep application tests in their own repository. TestStudio manages their catalog, releases, execution, timing, and private evidence.
Existing declared repositories use the manifest and catalog contract below. The Git onboarding workflow can generate these declarations for existing Playwright source and keep them in the database or commit them to Git. Discovery and preparation still validate the source before execution.
Use this contract for new repositories and for adapters that preserve existing tests.
See [source bundles](source-bundles.md) for packaging and [repository sources](repository-sources.md) for GitHub access.
The [Git test source workflow](repository-sources.md) supports onboarding without these declarations, untagged framework mappings, database or Git metadata ownership, and start-time synchronization. Live provider and hosted worker acceptance remain separate.

## Concepts

### Required declarations

| File or value | Requirement |
| --- | --- |
| Suite manifest | schemaVersion 1, permanent suiteId, runner, type, applications, moduleRoot, tests, catalog, and buildProfile. |
| Catalog | schemaVersion teststudio.catalog/v1 and one definition for each selected framework test. |
| Test identity | A permanent unique id. Renaming a test must preserve this value and its history. |
| Browser selector | One exact tag such as @UI-001. It must discover exactly the declared framework test in the selected project. |
| Source | A regular file path relative to the archive or repository root. It must match the discovered file. |
| Safety | safe, unsafe, or needs-review, with a rationale and operating conditions. Labels must assess setup, helpers, cleanup, and dependencies. |
| Application | A permanent applicationId in the catalog and the same application in the suite manifest and destination suite. |
| Timeout | timeoutSeconds from 1 through 3600. The platform records measured duration and applies its reviewed timing policy. |
| Dependencies | Permanent identifiers in the same suite. Preserve dependency safety and reject missing or cyclic references. |

The destination suite identifier must match suiteId. Register its application before preparing the source.
All manifest and catalog paths refer to the complete source root. Nested manifests do not change that root.
tests.config refers to the selected moduleRoot. Use exact project names when Playwright defines several projects.
Keep test bodies and assertions in source. The catalog supplies identity, selection, safety, lifecycle, and timing metadata.

### Dependencies and preparation

The current browser runtime uses exactly @playwright/test 1.64.1.
The selected moduleRoot needs package.json and a version 3 lockfile for @playwright/test, playwright, and playwright-core at that version.
Uploaded node_modules, workspace dependencies, package installers, and arbitrary build scripts do not run during preparation.
For additional JavaScript dependencies, use your repository's controlled build to produce self-contained source modules before upload.
Keep @playwright/test external so the trusted runtime owns the framework and browser version.
The exported module's package files must describe its actual remaining external dependency. Do not upload unresolved imports or undeclared runtime assets.
Go sources require go.mod and complete vendored dependencies or local replacement modules. Preparation does not download modules.

Source ZIP limits are 32 MiB compressed, 128 MiB expanded, 16 MiB per file, and 10000 entries.
Exclude secrets, environment files, browser storage, credentials, previous results, and generated private artifacts.
Archive validation rejects links, unsafe paths, duplicate paths, and case collisions.
Store original source references and build provenance beside generated bundles. Preserve the same permanent identifiers in generated catalogs.

### Discovery and execution

Discovery imports the configuration and lists tests. It does not run test bodies.
Module imports and configuration evaluation must not fetch a website, retrieve credentials, or require an active application session.
Avoid top-level setup that depends on a target address. A targetless discovery operation has no selected application environment.
Do not submit the old platform's execution payload or upload its result manifest from test bodies.
The trusted runner supplies execution selection, timeouts, capture, reporting, and evidence collection.
Use Playwright's normal fixtures, assertions, steps, and attachments. Do not replace the platform reporter or upload results directly to S3.
Saving and source preparation never start tests. A saved version becomes available after setup; execution remains explicit.
A successful discovery proves source compatibility. It does not prove that the application assertions pass.

### Targets, credentials, and privacy

Environment-backed browser suites use playwright-environment and browser.endpoint to name one registered endpoint.
Use TESTSTUDIO_BASE_URL or Playwright baseURL for the selected origin. Keep concrete installation addresses outside generic test code.
The current browser profile permits one exact origin. Separate APIs, identity providers, map services, and asset hosts require additional profile support.
An environment's endpoint map alone does not authorize browser egress to every entry. Do not bypass the gateway to obtain additional access.
The current profile has no general browser credential injection. API request suites have separate reviewed authentication policies.
Tests receive no AWS task role or installation credentials. Do not retrieve application secrets through ambient cloud credentials.
Authenticated browser adapters need a supported managed credential bridge before execution. A catalog label cannot supply that bridge.
Keep passwords, cookies, bearer tokens, and private response bodies out of assertions, attachment names, logs, and source archives.
Review screenshot, trace, and video policies for authenticated workflows before enabling capture.
Retain application-specific safety guards. Protected environments also require safe selections and current platform permission.

### Operator prerequisites

The installation needs an approved runtime, isolated discovery and execution workers, and source preparation before coded bundles become usable.
The initial MMDev core deployment supports native HTTPS probes. Its separate coded worker and preparation installation remains pending.
Git bindings support the declared offline profiles. Environment-backed bindings select their named endpoint and require an installed execution gateway.
Private GitHub sources require an approved GitHub App reference with contents read access to the exact repository.
The browser selects that reference. It never receives the App key or an installation token.
Repository synchronization reads committed source. It does not run the repository's build script.
If a repository needs generated modules, publish those files at the declared source paths before synchronizing that commit.

## Tutorials

### Author a small browser suite

Use this layout at the source root:

```text
teststudio-suite.json
catalog.json
browser/package.json
browser/package-lock.json
browser/playwright.config.ts
browser/checks/navigation.spec.ts
```

Declare the suite:

```json
{
  "schemaVersion": 1,
  "suiteId": "website",
  "type": "browser",
  "runner": "playwright",
  "applications": ["website"],
  "moduleRoot": "browser",
  "tests": { "config": "playwright.config.ts", "project": "chromium" },
  "catalog": "catalog.json",
  "buildProfile": "playwright-environment",
  "browser": { "endpoint": "website" }
}
```

Declare the catalog entry:

```json
{
  "schemaVersion": "teststudio.catalog/v1",
  "tests": [{
    "id": "UI-001",
    "title": "Open the registered home page",
    "applicationId": "website",
    "category": "Navigation",
    "type": "browser",
    "runner": "playwright",
    "state": "enabled",
    "safety": "needs-review",
    "rationale": "Review application requests and side effects before approving this check.",
    "conditions": "One registered website origin and an anonymous browser session.",
    "dependencies": [],
    "requiredBindings": [],
    "timeoutSeconds": 30,
    "source": "browser/checks/navigation.spec.ts",
    "selector": "@UI-001"
  }]
}
```

The test source uses the same permanent selector:

```ts
import { test, expect } from '@playwright/test'

test('Open the registered home page', { tag: '@UI-001' }, async ({ page }) => {
  await page.goto('/')
  await expect(page.getByRole('main')).toBeVisible()
})
```

Configure one Chromium project with testDir ./checks and baseURL from TESTSTUDIO_BASE_URL.
Include the exact package and lockfile described in [source bundles](source-bundles.md#trusted-playwright-runtime).
The gateway supplies the selected target during execution. Discovery needs no real target or browser session.

### Load a source bundle

1. Confirm that source preparation and the required execution profile are available in the installation.
2. Create the project, application, and destination suite. Environment-backed suites require a target environment.
3. Register the environment with the endpoint identifier declared in browser.endpoint.
4. Open Import and sync tests. Select Source ZIP and the manifest path relative to the complete archive root.
5. Select Upload and prepare bundle. Inspect diagnostics and the discovered test count.
6. Review the source and catalog. Save after the requirements and safety labels are correct. Read/write Git metadata uses its separate commit and merge workflow.
7. Start a separate run against the reviewed environment. Inspect timing, logs, and evidence.

If preparation controls are unavailable, the operator must install or enable the source worker first.
Do not substitute a catalog-only upload for the missing executable source and discovery step.

### Validate a repository adapter

Preserve every original identifier, safety recommendation, condition, application, and dependency in the exported catalog.
Check missing selectors, duplicate declarations, foreign applications, unsupported dependencies, and credential handling before publication.
Assemble with teststudio-publish build-bundle and inspect the image with teststudio-publish inspect.
Use their private state-directory and cleanup commands. Reuse approved runtime images and retain discovery receipts.
Check that every expected definition discovers once. Do not run a full remote suite merely to validate packaging.
Document any remaining network or credential requirement explicitly. A preparation receipt must not claim successful application execution.
