# Source bundles

The Import and sync tests page stores complete Go or Playwright ZIP bundles and queues durable preparation jobs.
A separate worker assembles the source and invokes isolated discovery.
Review and publication use the common immutable release workflow.
The CLI can also assemble bundles outside the API.
Registered [GitHub sources](repository-sources.md) use the same preparation workflow.
The offline Playwright profile uses dependencies already installed in a trusted browser runtime.

The in-app [Playwright editor](playwright-editor.md) generates these bundles from saved source drafts.

## Archive context

Include the complete suite context, manifest, catalog, tests, and required dependencies.
A manifest can live at a selected nested path. Its declared paths refer to the ZIP root.
This permits multiple suite manifests and shared files within one repository snapshot.
The selected suite identifier must match the destination suite.

The compressed archive limit is 32 MiB. Expanded files cannot exceed 128 MiB in total.
Each regular file has a 16 MiB limit. An archive can contain up to 10000 entries.
Validation rejects absolute paths, parent traversal, duplicate paths, case collisions, links, special entries, and file-directory conflicts.
CRC errors and missing declared files reject the bundle before assembly.
The source digest covers the exact ZIP bytes.

Go contexts require go.mod under moduleRoot and each declared test source file.
Include vendor dependencies with vendor/modules.txt when external modules are needed.
Alternatively, include complete local replacement modules in the archive.
Discovery and execution disable module downloads and toolchain downloads.
Assembly does not install dependencies or execute uploaded build scripts.

## Trusted Go runtime

Prepare the pinned Go base image in the local Docker daemon before building the runtime.
The runtime script does not pull images automatically.
Build the generic runtime once:

```sh
bash scripts/build-go-runtime.sh
```

Use its printed immutable image identifier for runtime-image.
The trusted runtime contains the toolchain and TestStudio runner, with an empty suite directory.
Its labels identify the source-base contract and runtime profile.
The assembler rejects runtime images with ONBUILD instructions.
The script sends only the compiled runner directory as build context.

## Trusted Playwright runtime

Prepare the pinned Playwright base image in the local Docker daemon.
Build the trusted runtime once:

```sh
bash scripts/build-playwright-runtime.sh
```

The script sends only the runner declarations and implementation as build context.
It reuses cached dependency layers. It does not include TestStudio self-tests, the UI, or controller files.
The image contains the browser toolchain and pinned Playwright packages.
Suite images add only validated source files and provenance. Assembly runs no uploaded scripts or dependency installer.

Use runner playwright and buildProfile playwright-node in the suite manifest.
The moduleRoot contains package.json, package-lock.json, the selected configuration, tests, and helpers.
Declare the exact @playwright/test version from the trusted runtime as a dependency or development dependency.
The current runtime uses version 1.64.1.
The version 3 lockfile contains its root entry and @playwright/test, playwright, and playwright-core at that same version.
Floating versions, additional packages, workspace dependencies, and uploaded node_modules are rejected by this offline profile.
Include application source and local helpers in the archive when tests import them.
The Playwright runner resolves its trusted packages from the runtime, including when moduleRoot is nested.
Online preparation and additional dependency profiles remain separate required work.

Start the worker with playwright-runtime instead of, or beside, go-runtime:

```sh
bin/teststudio-prepare --database .data/teststudio.db \
  --playwright-runtime sha256:TRUSTED_BROWSER_RUNTIME_IMAGE \
  --execution-location local --build-directory .data/source-builds
```

Configure each runtime needed at the location. Workers leave jobs for other runtimes queued.
They scan subsequent queue pages to avoid starving an available runner behind pending work for another runtime.
Recovery keeps both ownership profiles available when new builds are disabled.
Offline Go and browser profiles require a suite without a target environment.
The [environment-backed browser profile](browser-execution.md) supports registered targets with the same offline dependency preparation.
Tests can start isolated application fixtures or render source directly in their own worker.

## Server worker and interface

Enable Go and Playwright source bundle staging on the API:

```sh
bin/teststudio --local --source-bundles
```

Build and start the separate preparation worker:

```sh
go build -o bin/teststudio-prepare ./cmd/teststudio-prepare
bin/teststudio-prepare --database .data/teststudio.db \
  --execution-location local --go-runtime sha256:TRUSTED_RUNTIME_IMAGE \
  --build-directory .data/source-builds
```

The API does not assemble images or install dependencies.
The worker shares the metadata database and execution location with the API.
SQLite supports local processes that share the same database file.
Each local location uses the same Docker daemon and private build directory across its workers.
PostgreSQL can also provide the shared metadata database.
Omit both runtime flags to recover recorded resources without dispatching new builds.

Open Import and sync tests and choose a suite without a target environment.
Select a complete ZIP under Source ZIP and enter its manifest path relative to the ZIP root.
Select Upload and prepare bundle. Repeating a request with the same key returns its original job.
The worker records ownership before assembly, then records the image before discovery.
Builds and discovery do not execute test bodies.

Preparation details show the source digest, image, runtime, build timestamps, diagnostics, and state.
Use Get original source ZIP to fetch the exact retained archive.
Use Review prepared source to inspect catalog changes and publish with optional activation.
Publication creates no run. Start tests through a separate normal run action.
Cancel stops active work. Withdraw review cancels unpublished ready work.
Failed builds preserve the active release and retain diagnostics.

## CLI assembly and publication

Build the publisher locally and assemble a complete bundle:

```sh
go build -o bin/teststudio-publish ./cmd/teststudio-publish
bin/teststudio-publish build-bundle \
  --archive source.zip --manifest suites/arithmetic.json \
  --runtime-image sha256:TRUSTED_RUNTIME_IMAGE \
  --project example --suite arithmetic --output build.json
```

Assembly uses a fixed Dockerfile with a pinned base, project labels, source copy, and working directory.
Uploaded Dockerfiles and ignore files stay inside the source directory and cannot replace those instructions.
The command has a two-minute deadline and bounded retained diagnostics.
It writes a new private result file and records the stored archive digest as source provenance.
The assembled image is not yet inspected, published, activated, or tested.

Authenticate with the publisher CLI, then submit the assembled image:

```sh
bin/teststudio-publish login --server http://127.0.0.1:8040 --local
bin/teststudio-publish prepare --project example --suite arithmetic \
  --image sha256:ASSEMBLED_IMAGE --runner go --key bundle-version-one
bin/teststudio-publish status --project example --suite arithmetic \
  --job PREPARATION_ID --review
bin/teststudio-publish publish --project example --suite arithmetic \
  --job PREPARATION_ID --activate
```

The server requires prepared-sources and a targetless destination suite.
Inspection compiles declared packages without executing test bodies.
Publication and activation do not start tests. Start a normal run explicitly afterward.
See [prepared suites](prepared-suites.md) for managed sessions, cancellation, review, and server recovery.

## Retention and recovery

The CLI saves a private build ownership reference before invoking Docker.
It retains that reference after success because the image is a required execution artifact.
For CLI-only assembly, keep the original ZIP and result file while the source snapshot is needed.
Server uploads retain exact ZIP bytes with a project-bound digest.
Unpublished staging expires after 30 minutes. A ready review receives its own 30-minute staging window.
The maintenance worker removes expired ZIP bytes and requests cleanup of unused images.
Expiry keeps preparation summaries, logs, original failure reasons, and completion timestamps.
Published ZIP snapshots remain immutable and retained with their releases.
Configured published-source retention and image garbage collection remain pending.

Remove an assembled image only after its releases and runs no longer need it:

```sh
bin/teststudio-publish cleanup --state-file PATH_TO_BUILD_REFERENCE
```

Cleanup checks the project, installation, and preparation labels before removing the exact image tag.
It does not remove another preparation's image or unrelated containers.
CLI cleanup and worker recovery remove owned image tags and private temporary contexts after interruption.
Recovery does not repeat builds, discovery, or tests.
A separate worker must remain available to recover bundle resources when API staging is disabled.
Private CLI references and durable worker references identify the temporary source context after process interruption.
Cleanup rejects moved references, symbolic links, and non-private context directories.
Normal completion and cancellation remove temporary source contexts.

## Focused local validation

Archive and builder package tests cover validation, fixed build instructions, bounded logs, reference recovery, and ownership checks.
The Go runner has a targeted offline vendor check for discovery and normal execution.
A real local ZIP workflow verifies assembly, server inspection, reviewed publication, execution, and owned resource cleanup.
Reuse the trusted runtime when repeating that workflow after a relevant behavior change.
