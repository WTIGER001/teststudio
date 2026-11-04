# Portable service package

The service image contains the Go API, Angular UI, private trace viewer, and bounded health probe.
It uses the same product code and configuration contracts as the local server.
The image supplies a common package for local containers and future AWS, Kubernetes, and Azure profiles.
Hosted execution adapters and deployment acceptance remain open under the implementation ledger.

## Build locally

Install the repository's locked web dependencies with npm ci --prefix web.
Go 1.26.2 or later, Node, npm, and Docker are required for the build.
Build the Docker host architecture by default:

```sh
node scripts/build-service-image.mjs
```

Select the required Linux architecture explicitly:

```sh
node scripts/build-service-image.mjs --arch=amd64
node scripts/build-service-image.mjs --arch=arm64
```

The command returns JSON with the immutable local image identifier, architecture, and source key.
It tags the image as teststudio/service:ARCH-local. It performs no registry push or deployment.
The local image identifier differs from a registry distribution digest. Use a registry image digest when preparing hosted manifests.
The current functional package verification uses arm64. Other architecture execution still requires its own verification.

The builder reuses the canonical UI and trace-viewer assets through --assets-only.
It builds only the service and health binaries needed by this package.
Go workspace mode is off. Module metadata is read-only, and binaries use CGO_ENABLED=0.
Source inputs, effective compiler settings, binary hashes, asset hashes, Dockerfile content, and image metadata control reuse.
The container build has no network access and uses an immutable certificate-source image.
Its temporary context contains only the selected binaries and static assets. The context is removed after assembly.
An unchanged image build reuses existing assets, binaries, and the verified image identifier.

## Runtime configuration

The image runs as user and group 65532. It supports a read-only root filesystem.
Mount writable storage at /data for local SQLite and filesystem artifacts. Mount private temporary storage at /tmp.
The default database environment variable is TESTSTUDIO_DATABASE. Its default value is /data/teststudio.db.
Hosted or multi-instance operation requires the installation's PostgreSQL URL in that variable.
Supply secret values through the deployment's protected environment or secret integration.

The API and preparation command also accept this option outside a container:

```sh
bin/teststudio --database-env TESTSTUDIO_DATABASE
bin/teststudio-prepare --database-env TESTSTUDIO_DATABASE
```

An explicitly supplied --database conflicts with --database-env. The ordinary local file default remains compatible.
A missing or blank selected variable fails before startup. Invalid environment names also fail explicitly.
Database initialization errors from this environment path exclude private connection values and upstream diagnostics.
The connection value never becomes a process argument.
Hosted API and preparation processes can add --require-postgres to reject a local SQLite fallback before opening storage.

The packaged entrypoint includes these settings:

| Setting | Packaged default |
| --- | --- |
| API listener | 0.0.0.0:8040 |
| Artifact listener | 0.0.0.0:8041 |
| Database reference | TESTSTUDIO_DATABASE environment variable |
| Filesystem artifacts | /data/artifacts |
| Angular assets | /app/ui |
| Private trace viewer | /app/trace-viewer |
| Health command | /app/teststudio-health |

Additional command arguments use the normal service flags.
Set --public-url and --artifact-origin to the installation's actual origins.
Use separate API/UI and artifact origins. Configure ingress TLS for hosted sessions.
Configure OIDC providers, approved secret references, and the exact initial administrator through the existing identity setup.
The /auth/local shortcut remains loopback-only. The existing host demonstration remains the convenient interactive local setup.
See [Identity providers](providers.md), [Access](access.md), and [Artifact storage](artifact-storage.md) for installation-specific settings.

## Local container example

Create a dedicated volume and substitute the built image identifier for IMAGE_ID:

```sh
docker volume create teststudio-local-data
docker run --rm --name teststudio-local-service \
  --read-only --cap-drop=ALL --security-opt=no-new-privileges \
  --tmpfs /tmp:rw,nosuid,nodev,size=67108864,uid=65532,gid=65532,mode=1777 \
  --mount type=volume,source=teststudio-local-data,target=/data \
  --publish 127.0.0.1:18040:8040 --publish 127.0.0.1:18041:8041 \
  IMAGE_ID --public-url=http://127.0.0.1:18040 \
  --artifact-origin=http://127.0.0.1:18041
```

Add the installation's identity and artifact settings before interactive use.
Use --env-file with a private operator-owned file when supplying a local PostgreSQL connection.
The image includes CA certificates and built-in timezone data for named schedule timezones.
The health command checks the bounded TestStudio ready response with a two-second deadline and rejects redirects.
/healthz checks database readiness. /livez checks process liveness independently of database availability.
Override its --url or --timeout only when the installation changes the API listener.
Provider orchestrators must configure their readiness and startup probes according to their own lifecycle contract.

## Verify this package

The focused local verifier starts only owned containers and a fresh private volume:

```sh
python3 scripts/verify-service-image.py --image IMAGE_ID \
  --output output/service-package/local-check
```

It checks non-root startup, root filesystem protection, private writable storage, the in-image health probe, and origin separation.
It recreates the container against the same volume and verifies retained SQLite migration state and empty testing registries.
A new invalid-connection canary verifies that startup errors and arguments exclude the selected database value.
The verifier closes its containers and removes its volume after success or failure.
It reads cold SQLite copies in immutable mode. It starts no browser and makes no cloud request.

The service supports native HTTP checks, scheduling, notifications, and the selected artifact adapter through their existing contracts.
Browser and coded suites require their separate execution workers. Keep Docker access on dedicated local workers.
Hosted dispatch, preparation, cancellation, deadlines, recovery, and provider-specific private access remain required work.
The package verification does not complete AWS, Kubernetes, Azure, or cross-profile acceptance.

The [Kubernetes service profile](kubernetes.md) supplies Helm configuration, private Secret references, separate origins, storage, and probes for this image.
