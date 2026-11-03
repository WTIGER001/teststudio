# Offline Go source bundles

The publisher CLI assembles validated Go ZIP bundles into local immutable images.
Server inspection and reviewed publication use the existing prepared source workflow.
Direct UI ZIP uploads, durable server build dispatch, GitHub sync, and Playwright bundle assembly remain pending.

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

## Trusted runtime

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

## Assemble and publish

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
Keep the original ZIP and result file while the source snapshot is needed.
The server currently retains discovered source metadata, rather than the uploaded ZIP itself.
Server-side ZIP staging and retention remain pending.

Remove an assembled image only after its releases and runs no longer need it:

```sh
bin/teststudio-publish cleanup --state-file PATH_TO_BUILD_REFERENCE
```

Cleanup checks the project, installation, and preparation labels before removing the exact image tag.
It does not remove another preparation's image or unrelated containers.
CLI build references can also recover owned image tags after interruption.
The private reference also identifies the temporary source context for cleanup after process interruption.
Cleanup rejects moved references, symbolic links, and non-private context directories.
Normal completion and cancellation remove temporary source contexts.

## Focused local validation

Archive and builder package tests cover validation, fixed build instructions, bounded logs, reference recovery, and ownership checks.
The Go runner has a targeted offline vendor check for discovery and normal execution.
A real local ZIP workflow verifies assembly, server inspection, reviewed publication, execution, and owned resource cleanup.
Reuse the trusted runtime when repeating that workflow after a relevant behavior change.
