# Hosted registry inspection

## Scope

The OCI registry adapter reads immutable prepared-image metadata for hosted execution.
It implements the image-inspection method required by the shared provider contract.
It uses no Docker daemon, local image cache, AWS client, or Azure client.
It reads manifests and image configuration. It downloads no test source or filesystem layer.

Keep prepared-artifact approval enabled alongside metadata checks.
The resulting image identity feeds the [hosted execution lifecycle](hosted-execution.md).
Provider dispatch uses the Kubernetes adapter. Remote assembly now has a separate [registry source builder](registry-preparation.md).
The preparation worker now connects this adapter to [hosted discovery](hosted-discovery.md) through explicit native configuration.
No provider is enabled by this adapter alone.

## Installation inputs

Create the client with these operator-controlled settings:

| Setting | Purpose |
| --- | --- |
| Origin | One HTTPS registry origin without a path, query, fragment, or user information. |
| RepositoryPrefix | One explicit repository namespace. The exact repository and its child paths are eligible. |
| Architecture | The required Linux amd64 or arm64 platform. |
| Credentials | An optional callback that reads current private Basic or Bearer credentials for each inspection. |
| TokenURL | An optional exact HTTPS token endpoint for registry Bearer challenges. |
| TokenService | The exact expected service value for that endpoint. |
| BlobRedirectOrigins | Optional explicit HTTPS origins for signed configuration-blob redirects. |
| Transport | An optional trusted transport for installation certificates or local protocol fixtures. |

The default transport uses system TLS trust and no environment proxy.
The whole inspection has a fifteen-second bound. Each HTTP request has a ten-second bound.
The shared lifecycle can impose its shorter control deadline.
Close the client to release idle connections.

References include the configured registry host, allowed repository path, and sha256 distribution digest.
Mutable tags and local image identifiers cannot select remote work.
A neighboring repository prefix does not match the configured namespace.
References cannot select another registry or traverse a repository path.

## Authentication and redirects

Credentials remain in the trusted controller or preparation process.
They never become worker input or provider job configuration.
Credential values do not serialize or appear in diagnostic strings.
Errors use fixed messages instead of response bodies, signed URLs, or credential-bearing requests.

A credential callback runs once for each inspection and can return rotated credentials.
Basic credentials and pre-issued Bearer credentials are supported.
An inspection retains its scoped challenge token only for that inspection.
It does not share a token between repository inspections.

A Bearer challenge must match the configured token endpoint and service.
Its scope must be absent or exactly repository:NAME:pull for the selected repository.
The token request always uses that exact pull scope.
An unapproved token endpoint receives no request or credentials.
Token responses have a 128 KiB limit.

Manifest requests and token exchanges reject redirects.
Configuration blobs can follow up to three redirects to explicitly configured HTTPS origins.
Redirected requests contain no registry Authorization header, including redirects within the registry origin.
Signed query values remain private. Redirected bytes still pass exact size and digest verification.

## Immutable metadata checks

Every fetched manifest and configuration blob must match its requested sha256 digest.
An optional Docker-Content-Digest response header must agree with those bytes.
Descriptor sizes must match fetched child-manifest and configuration sizes.
Each metadata response has a 2 MiB bound.

The adapter accepts OCI manifests and Docker schema-two manifests.
It also accepts their image-index and manifest-list forms.
An index must identify exactly one eligible configured platform.
An ambiguous platform selection fails explicitly.
The selected index entry and actual image configuration must agree with the installation platform.
Linux arm64 accepts the default variant or v8. Unsupported variants and required platform features fail explicitly.
The selected child must be a direct image manifest.

Configuration must preserve the prepared source runtime contract:

- Non-root user 1000:1000 and working directory /suite.
- Original fixed Go or Playwright runner entrypoint.
- No ONBUILD instructions.
- The worker-exchange/v1 supervisor label and immutable supervised base label.
- The source-base/v1 runtime and 4 MiB work-item labels.
- Valid project, installation, preparation, and source-digest labels.
- A valid runtime identity, platform, root filesystem type, and layer digest list.

An optional teststudio.task-storage=task-volumes/v1 capability requires exactly two empty volume declarations: /tmp and /output.
Unknown storage capabilities, missing paths, extra paths, and invalid volume declarations fail inspection.
The result carries this capability to providers. ECS requires it for non-root task-local storage.
Legacy images without it retain existing provider-independent inspection behavior. See [ECS worker storage](ecs-worker-contract.md).

The adapter checks bounded layer descriptors without fetching their content.
Foreign layer URLs, unsupported layer types, invalid digests, and oversized layer declarations fail explicitly.
Container runtimes remain responsible for layer download and content verification during the approved image pull.

The result retains the original selected distribution reference, resolved platform manifest, configuration digest, runtime identity, and preparation ownership.
Providers must schedule the inspected architecture when dispatching an index reference.
The shared lifecycle compares inspected project ownership with the approved artifact.
When retained provenance declares a source digest, the inspected source digest must match.
The existing source approval binds that immutable artifact to its prepared project source.

## Local evidence

Focused race-enabled checks use owned local TLS servers.
Both runner identities pass with single-image and platform-index references.
The fixture checks scoped token reuse, later credential rotation, exact metadata requests, and private redirect handling.
It also checks invalid references, digest mismatch, unapproved token realms, oversize metadata, ambiguous indexes, descriptor size mismatch, and unsupported source runtimes.
No browser, source-runtime rebuild, Docker pull, provider API, or cloud account is used.
Evidence is output/registry-inspection/contract.log.

## Protocol references

The adapter follows the [CNCF Distribution HTTP API](https://distribution.github.io/distribution/spec/api/).
Image parsing follows the OCI 1.1.1 [manifest](https://github.com/opencontainers/image-spec/blob/v1.1.1/manifest.md), [index](https://github.com/opencontainers/image-spec/blob/v1.1.1/image-index.md), and [configuration](https://github.com/opencontainers/image-spec/blob/v1.1.1/config.md) specifications.
