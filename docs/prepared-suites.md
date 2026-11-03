# Prepared suite inspection

TestStudio has a versioned manifest and a local isolated artifact inspector.
The Sources page, API, and publisher CLI enqueue inspection jobs and publish reviewed catalogs.
The current worker supports locally available immutable Go and Playwright images.
Inspection does not start tests. Publication and activation remain explicit actions.
The Sources page also uploads offline Go or Playwright ZIP bundles for a separate preparation worker.
The CLI can assemble those bundles outside the API.
Registered GitHub sources use the same durable preparation and reviewed publication workflow.
See [repository sources](repository-sources.md) and [source bundles](source-bundles.md) for supported offline profiles.
See [source bundles](source-bundles.md) for worker setup, staging, and retention.

## Manifest

Put `teststudio-suite.json` at the image's `/suite` root.
Use schema version 1 and a permanent suite identifier.
The [manifest schema](../contracts/suite.schema.json) describes the canonical JSON shape.
The Go validator also checks paths, references, runner compatibility, and catalog relationships.

```json
{
  "schemaVersion": 1,
  "suiteId": "arithmetic",
  "type": "unit",
  "runner": "go",
  "applications": [],
  "moduleRoot": "module",
  "tests": { "packages": ["./checks"] },
  "catalog": "testing/catalog.json",
  "buildProfile": "go-module"
}
```

Module and catalog paths refer to the source context under `/suite`.
Paths cannot escape that context.
Catalog files use the existing `teststudio.catalog/v1` schema in JSON or YAML.
Every coded entry declares its runner, permanent identifier, source path, and selector.
Applications and required bindings must also appear in the manifest.

Go manifests list 1 through 100 explicit packages relative to `moduleRoot`.
Recursive packages and parent traversal are invalid.
A Go source path includes its module path and `_test.go` file name.

Playwright manifests use build profile `playwright-node`.
Their `tests` object can declare `config` and one exact `project` identifier.
The configuration path is relative to `moduleRoot` and defaults to `playwright.config.ts`.
One permanent tag must select exactly one framework test after project selection.
Multiple project instances or repeated cases cannot share one catalog selector.

The current inspector supports `go-module` and `playwright-node`.
Request-policy declarations and execution bindings require future compatible profiles.
The current local inspector rejects those declarations.

## Provenance

Put `teststudio-origin.json` beside the manifest.
The [origin schema](../contracts/source-origin.schema.json) describes its canonical form.
Git provenance records a credential-free HTTPS repository URL, the requested reference, and the exact resolved commit.
A commit contains 40 or 64 lowercase hexadecimal characters.
Local bundle provenance records a SHA-256 source snapshot digest.
Git provenance can also include that digest.

The inspector reads provenance from the immutable image.
It does not fetch the repository or independently attest the build's claimed source origin.
Server preparation requires the image label `teststudio.project` to match the destination project.
Publication records a project-bound artifact approval.
The execution worker accepts that exact published artifact without changing its startup image list.

## Discovery

Both prepared runtimes implement `--discover` and commit `/output/discovery.json` only after successful discovery.
Go discovery compiles the declared packages with local dependencies and module downloads disabled.
It records actual runnable root declarations and their source lines.
It never starts the compiled test executable, package initialization, or `TestMain`.
Helper functions, `TestMain`, and examples without expected output do not become executable catalog tests.

Dynamic Go child paths retain `parentOnly: true`.
Discovery verifies their parent's declaration and compilation.
The selected child path remains subject to normal execution checks.
An absent child cannot produce a passing result.

Playwright discovery uses [`--list`](https://playwright.dev/docs/test-cli) and a dedicated reporter.
The reporter uses the [suite API](https://playwright.dev/docs/api/class-suite) to retain source locations, project identity, and inherited tags.
Configuration and test module initialization can execute during Playwright loading.
Normal test dispatch, fixtures, hooks, global setup, the configured web server, and browser launch do not run during verified discovery.

The inspector checks the reported schema, complete declared catalog mapping, source files, selectors, runtime label, and image digest.
Reports have byte, depth, node, and test count limits.
Duplicate keys and unknown fields are rejected.
Only catalog entries are managed; other framework cases do not become implicit TestStudio tests.

## Local isolation

The local backend uses the existing managed container transport.
It has a separate preparation installation namespace and an explicit resource reference.
Inspection uses an approved, locally available immutable image.
It never pulls an image automatically.

The container runs as user 1000 with no network, host mounts, extra capabilities, or controller credentials.
Its root filesystem is read-only.
Limits allow two CPUs, 2 GiB of memory, 256 processes, and a two-minute inspection deadline.
Private temporary storage permits Go compilation.
One output volume retains the bounded discovery report until collection.
Unexpected output files prevent successful inspection.

Cleanup checks resource ownership before removal.
Cancellation removes the owned container and volume.
An existing resource reference requires explicit recovery before reuse.
The inspection interface stays separate from test scheduling and future source builds.
The API stores a durable job. A separate worker invokes inspection after the request completes.
Each job retains its resource reference before container creation.
Current source and catalog permissions are checked before dispatch and during preparation.

## Server and interface

Enable local preparation with the server flag `--prepared-sources`.
Docker must contain each immutable image before preparation starts.
Each image needs the expected runtime label and its destination `teststudio.project` label.
The current local profiles require a suite without a target environment.
The worker does not pull images or install source dependencies.

Open Sources, select the destination suite, and enter the image digest and runner.
Select Prepare image, inspect progress and retained logs, then open Review prepared source.
Review catalog changes and validation issues before publication.
Activation starts unchecked. Publication does not run tests.

Image jobs move from `received` to `discovering`, then `ready`, `failed`, or `cancelled`.
Bundle jobs add a `preparing` stage before discovery. Expired unpublished bundles cannot dispatch or publish.
Publication changes a ready job to `published` and records its release identifier.
Cancel can stop queued or running work and withdraw an unpublished ready review.
A published job cannot be cancelled.

Preparation records bounded diagnostics and observed runtime metadata.
Cleanup failures remain visible and retain their durable resource reference.
After a worker lease expires, recovery removes owned resources without repeating discovery or starting tests.
Recovery remains enabled when new preparation is disabled.

## CLI

Build the CLI locally:

```sh
go build -o bin/teststudio-publish ./cmd/teststudio-publish
```

Authenticate to an enabled loopback installation:

```sh
bin/teststudio-publish login --server http://127.0.0.1:8040 --local
```

Managed local accounts use `--user USER --password-env VARIABLE` instead of `--local`.
The CLI stores a private managed session file. Use `--session-file` to choose its location.
Remote connections require HTTPS. CLI OIDC browser sign-in remains pending.
Use `logout` to revoke the session and remove its local file.

Prepare and review a project-labelled image:

```sh
bin/teststudio-publish prepare --project example --suite arithmetic \
  --image sha256:YOUR_IMMUTABLE_IMAGE_DIGEST --runner go --key release-one
bin/teststudio-publish status --project example --suite arithmetic \
  --job PREPARATION_ID --review --logs
bin/teststudio-publish publish --project example --suite arithmetic \
  --job PREPARATION_ID --activate
```

Reuse the preparation key after a request failure. The same input returns the original job.
A changed image or runner requires a new key.
Omit `--activate` to publish without changing the active release.
Use `cancel` with the same project, suite, and job flags to cancel unpublished work.
The CLI does not start tests after publication.

### Standalone inspection

Inspect an externally prepared image already present in the local Docker daemon:

```sh
bin/teststudio-publish inspect \
  --image sha256:YOUR_IMMUTABLE_IMAGE_DIGEST \
  --runner go \
  --project example \
  --suite arithmetic \
  --output inspection.json
```

The output path must be new.
The CLI writes a private file with schema `teststudio.inspection/v1` and status `inspected` or `failed`.
An inspected artifact is not a published or activated release.
The file includes bounded diagnostics, observed source locations, and image-provided provenance.
Existing files and symbolic links are never replaced.

Before starting a worker, the CLI saves a private cleanup reference in the user cache.
Use `--state-directory` to choose another private directory.
Successful cleanup removes that reference.
If cleanup fails or the process is killed, recover its exact saved reference:

```sh
bin/teststudio-publish cleanup --state-file PATH_TO_SAVED_REFERENCE
```

The cleanup command validates the preparation namespace and resource ownership.
It does not remove unrelated containers or browser sessions.

## Local verification

The inspection foundation has a dedicated `bash scripts/test-discovery.sh` check.
Run it only when inspection changes require fresh fixture coverage.
The script builds new independent Go and Playwright fixture images.
It verifies discovery, source lines, dynamic child disclosure, exact project selection, isolation, cancellation, CLI privacy, and scoped cleanup.
It also executes each inspected fixture through the existing managed runner.
Private CLI results remain under the printed `.data/discovery-verification.*` directory.
No cloud account or remote application is required.

Targeted source, API, and CLI tests cover queued publication, retries, metadata preservation, cancellation, and interrupted-job cleanup.
Use these focused checks after publication changes. Reuse cached images for local end-to-end checks when available.
AWS, Kubernetes, and Azure verification remain separate work.
