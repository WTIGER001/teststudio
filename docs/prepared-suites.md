# Prepared suite inspection

TestStudio has a versioned manifest and a local isolated artifact inspector.
The publisher CLI currently provides `inspect` and `cleanup`.
Server-side prebuilt registration, preparation queues, publication, and the related interface remain pending.
Inspection does not create a TestStudio release or test run.

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
Future server publication must apply the installation's artifact policy and project authority checks.

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
The API does not invoke this inspector yet.

## CLI

Build the CLI locally:

```sh
go build -o bin/teststudio-publish ./cmd/teststudio-publish
```

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

Run `bash scripts/test-discovery.sh`.
The script builds new independent Go and Playwright fixture images.
It verifies discovery, source lines, dynamic child disclosure, exact project selection, isolation, cancellation, CLI privacy, and scoped cleanup.
It also executes each inspected fixture through the existing managed runner.
Private CLI results remain under the printed `.data/discovery-verification.*` directory.
No cloud account or remote application is required.
