# Versions and build information

## Introduction

The sidebar footer shows the application version, build identifier, and UTC build date on every page.
Expand Build to inspect the semantic version, source commit, source fingerprint, and source state.
On a narrow screen, select Navigation to reveal the footer.
The date identifies creation of the build. It does not report deployment time or server startup time.

## Version policy

VERSION is the source of truth for the product release. The first tracked development release is 0.1.0-dev.
The footer displays Major.Minor.Patch.Tag, such as 0.1.0.dev or 1.0.0.stable.
Package metadata uses the canonical [Semantic Versioning](https://semver.org/) spelling: 0.1.0-dev or 1.0.0.
Tags use lowercase letters, digits, and hyphens. Common tags are dev, alpha, beta, and rc1.
A stable release has no canonical prerelease suffix. The footer displays its tag as stable.

Increase Major for incompatible public API changes. Increase Minor for compatible features. Increase Patch for compatible fixes.
Major changes reset Minor and Patch. Minor changes reset Patch.
Version zero identifies initial development. Public API stability starts with version 1.0.0.
Published release contents stay immutable. Change the release version before publishing changed application behavior.
Build identifiers distinguish source snapshots and build creation. They do not replace release version increments.
Dependency and generated contract versions remain independent of the product version.

## Release commands

Run one of these commands from the repository root:

```sh
node scripts/version.mjs patch
node scripts/version.mjs minor
node scripts/version.mjs major
node scripts/version.mjs tag beta
node scripts/version.mjs tag rc1
node scripts/version.mjs tag stable
```

Numeric bumps retain the current tag. Tag changes retain the three numeric components.
The helper synchronizes VERSION, web/package.json, and the root package version in web/package-lock.json.
Commit these files with the release changes. The helper does not create Git tags or push changes.

## Build metadata

The shared generator records the release, creation date, source commit, source fingerprint, and working-source flag.
It excludes private data and generated build output. Includes working changes means the artifact contains uncommitted implementation changes.
The source fingerprint identifies those contents. The displayed source commit alone does not identify pending changes.
An unchanged source snapshot reuses its cached creation date and build identifier. Restarting or redeploying that artifact changes neither value.
Generated UI metadata stays out of Git. The UI build imports the generated snapshot into its JavaScript bundle.
Standard server builds embed the same snapshot through Go linker flags.
The portable service image also includes version, creation date, commit, and build identifier labels.

```sh
node scripts/build-info.mjs
node scripts/build-info.mjs --refresh
node scripts/build-service-image.mjs --arch=arm64
```

Use --refresh only to create a distinct build from unchanged source. It generates a new identifier and creation date.
Build the complete UI and server package afterward so both use that new snapshot.
The self-test builder, demonstration builder, UI lifecycle commands, and service builder generate metadata automatically.
A raw Go build without the linker manifest reports development with an unversioned tag and no invented build date.

## Version API

GET /api/v1/version returns public build metadata and uses no-store caching. It requires no account or project access.
Fields include version, displayVersion, major, minor, patch, tag, buildId, builtAt, commit, sourceHash, and modified.
The endpoint contains no users, session data, secrets, or project configuration.
The footer identifies the UI bundle currently loaded. The API identifies the running server artifact.
During a rolling deployment or in an old browser tab, these values can differ. Refresh after deployment to load the matching UI.

## Targeted verification

Version utility checks cover numeric increments, tags, invalid versions, and linker encoding.
Go checks cover embedded metadata, unversioned fallback, and the public API response.
STUDIO-VERSION-001 compares the actual packaged UI and API through normal local TestStudio execution.
It verifies the footer, UTC date, source details, reload persistence, and narrow layout.
