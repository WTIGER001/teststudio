# Registry source assembly

## Current scope

The registry source builder implements the existing ArtifactBuilder contract.
It assembles validated offline Go and Playwright snapshots outside the API.
It publishes an immutable OCI image through the registry HTTPS protocol.
The [trusted ACR adapter](azure-registry-authentication.md) now supplies separate publication token scope for this same upload and journal flow.
The [Azure preparation path observer](azure-preparation-registry.md) adds selected endpoint, subnet, NIC, security, and DNS checks.
The [Azure publisher library](azure-preparation-writer.md) composes that path with a selected managed identity and registry-scoped push grant.
The [native Azure assembly worker](azure-native-assembly.md) can now publish and queue discovery. Complete Azure source preparation still requires admitted isolated discovery.
It needs no Docker daemon, local image cache, cloud SDK, shell, or uploaded Dockerfile execution.

The preparation worker enables this adapter through explicit hosted settings and registry runtime flags.
It connects the durable build queue to hosted discovery and the existing source review and approval workflow.
See [hosted preparation](hosted-discovery.md) for native setup, cleanup ordering, and remaining deployment packaging.
Do not use the local image inspector for registry references.
Published registry content alone is not an approved TestStudio artifact or executable catalog release.
The normal discovery, source review, approval, and publication steps remain required before dispatch.

See [Registry inspection](registry-inspection.md), [Prepared suites](prepared-suites.md), and [Kubernetes workers](kubernetes-workers.md).

The builder now implements AssemblyBuilder.CompleteAssembly for [separate discovery handoff](source-assembly-handoff.md).
It releases the exact owned upload journal while preserving immutable source content. Repeated completion is safe.
This capability lets the source service separate registry writer credentials from its discovery worker.

## Trusted inputs

Construct the builder with an OCI client, an immutable supervised runtime reference, and a private recovery directory.
The client restricts the registry origin, repository namespace, architecture, token endpoint, and credential source.
The runtime reference must identify an existing distribution digest within that namespace.
The base and assembled artifact use the same repository. No layer download or cross-repository copy is required.

The selected runtime must pass the existing platform, digest, fixed entrypoint, non-root user, and supervisor checks.
It must also advertise the hosted discovery supervisor contract. Older supervised runtimes fail before publication.
A base with project, installation, preparation, or source ownership labels is rejected.
Use a trusted source runtime, not another project's prepared source image.
ONBUILD instructions and unsupported runtime contracts remain invalid.

The builder accepts a previously validated complete source snapshot and matching provenance.
Its preparation reference identifies the project, installation, and job.
The installation identifier must use the preparation namespace.
Only go-module and playwright-node offline profiles are supported.
Registered targets, credential bindings, environment profiles, and dependency downloads remain unsupported here.

Playwright packages must match the trusted runtime's exact version and lockfile.
Local and registry assembly use the same dependency validator.
Uploaded packages cannot replace the trusted Playwright installation or add unprepared dependencies.

## Deterministic image construction

The builder creates one gzip-compressed source layer.
Source files remain beneath /suite and retain their executable flag.
Files and directories use owner 1000 and private permissions.
Entry ordering and timestamps remain deterministic.
The builder records both the compressed blob digest and the uncompressed layer digest.

Regular files and directories are the only uploaded entry types.
Path traversal, conflicting paths, reserved metadata conflicts, and uploaded OCI whiteout names are rejected.
The layer includes a controlled opaque marker that removes inherited /suite content.
Source sizes retain the bundle limits. Generated directory entries have a separate bound.
The canonical manifest and retained provenance replace uploaded copies of those metadata files.
Uploaded Dockerfiles remain ordinary source files. They never control image assembly.

The image retains the base configuration, environment, entrypoint, platform, and supervisor labels.
It appends the new filesystem digest and ownership labels.
The manifest references existing base layers and the new source layer.
The output uses a distribution digest. It creates no mutable image tag.
The same preparation reference, runtime, and source produce the same artifact identity.

## Registry publication

Registry credentials stay in the trusted preparation process.
Each operation reads current credentials. Recovery can use rotated credentials.
Write authentication requests the exact repository pull,push scope.
Unapproved token endpoints, repositories, or additional permissions are rejected.

The uploader checks existing blob digests before transferring content.
New blobs use the standard POST and PUT upload sequence.
The upload location must remain in the same registry and repository upload namespace.
Private query parameters remain intact. Redirects do not receive registry credentials.
External upload locations are unsupported by this profile and fail explicitly.

Acknowledged writes require matching digest observations.
After an uncertain blob acknowledgement, the builder checks the exact content address.
It closes the recorded upload session before clearing its journal.
After an uncertain manifest acknowledgement, immutable inspection can confirm the published artifact.
Inspection must match the project, preparation, source digest, and image configuration.

Each build has a two-minute context bound. Individual upload requests have a one-minute bound.
Final metadata confirmation can use a separate fifteen-second bound after cancellation.
Publication never starts test code or grants source approval.
The preparation service must still check cancellation and its lease before accepting a build checkpoint.

## Ownership and recovery

The recovery directory must be private and cannot be a symbolic link.
Each preparation reserves its own directory before network work starts.
An existing reference requires recovery before reuse.
The journal stores the full ownership reference, runtime reference, and pending upload state.
Private upload addresses stay in a mode-0600 journal. They never enter public job diagnostics.
Use durable private storage shared by the preparation workers that can recover these references.

Before creating a session, the builder saves an opening marker.
It saves the returned session location before sending source bytes.
Known upload sessions can be cancelled after process restart.
A valid interrupted location checkpoint also supports cleanup after an interrupted journal rename.
Cleanup rejects another owner, an unsafe directory, or an upload address outside the configured repository.

An opening request can lose its response after the registry creates a session.
Without a returned session address, the builder cannot prove cleanup.
It retains the journal and returns cleanup-incomplete. It does not repeat that preparation automatically.
Operators must reconcile the registry upload before clearing that private recovery record.
Registry expiry alone is not treated as observed termination.

Cleanup removes owned upload sessions and private journal state.
It does not delete manifests, source blobs, base layers, or published artifacts.
Registry lifecycle policies must preserve artifacts required by retained releases.
Unreferenced completed blobs can require the registry's own garbage collection.

## Local verification

The focused checks use new owned TLS registry fixtures. They contact no external registry or provider.
They cover both runners, deterministic source layers, dependency restrictions, source ownership, and preserved runtime configuration.
They also cover scoped tokens, private upload queries, deduplication, lost acknowledgements, cancellation, credential rotation, and restart cleanup.
Unknown creation remains uncertain. Foreign upload addresses and private recovery-directory links are rejected.
The existing OCI inspection and local assembly checks run only where their shared code changes.

Evidence is output/registry-preparation/contract.log. Final scope and durability checks pass in final-contract.log and durable-journal.log.
The API, preparation-worker, and publisher packages also pass compilation.
No browser, UI build, runtime image rebuild, registry push, cluster operation, or cloud verification is required for these fixtures.

The publication path follows the [OCI Distribution 1.1.1 upload contract](https://github.com/opencontainers/distribution-spec/blob/v1.1.1/spec.md#pushing-blobs).
Source layers follow the [OCI image layer format](https://github.com/opencontainers/image-spec/blob/v1.1.1/layer.md).
