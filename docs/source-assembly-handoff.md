# Separate source assembly and discovery

## Tester workflow

Some hosted installations use separate workers for image assembly and test discovery.
The tester still uploads or synchronizes source, follows preparation, reviews discovered tests, and publishes explicitly.
Assembly alone creates no review, approval, active release, test run, or timing baseline.

After assembly, Import and sync tests displays Waiting for discovery.
The retained image, source digest, build timing, and assembly log remain available while the job waits.
The tester can cancel preparation or obtain the retained source through the existing authorized controls.
A discovery worker checks current access and retained source before starting discovery.
Review becomes available only after successful normal discovery.

## Optional operator mode

The existing preparation command adds --assembly-only, which defaults to false.
Use it with the existing ECS or Kubernetes registry settings and registry runtime flags.
Local image preparation is rejected before database or directory creation because its cleanup can remove the artifact.
Azure native settings and workload packaging remain unfinished. This flag does not enable Azure coded execution or an Azure preparation profile.

For an existing Kubernetes registry profile, the additional selection looks like this:

```sh
teststudio-prepare --assembly-only \
  --kubernetes-workers-config=/private/workers.json \
  --execution-location=kubernetes \
  --database-env=TESTSTUDIO_DATABASE \
  --build-directory=/private/source-journals \
  --registry-go-runtime=REGISTRY/NAMESPACE/go@sha256:DISTRIBUTION_DIGEST
```

Use the normal provider settings and immutable supervised runtime requirements from [hosted discovery](hosted-discovery.md).
Hosted settings still require PostgreSQL and reject mixed providers, locations, and local runtime flags.
The API keeps the normal hosted discovery inspector and --prepared-sources configuration.
It can consume assembled bundles without a registry writer or build journal directory.
Do not enable this mode until a compatible discovery worker is configured, or jobs remain queued until cancellation or expiry.

Assembly-only mode invokes no discovery inspector or execution-control operation.
Existing hosted client construction and startup credential checks still apply.
The flag does not remove permissions from an existing cloud role or complete provider identity packaging.
Keep registry publication authority on the trusted writer. Review deployment permissions separately.

## Durable handoff

The worker claims the existing build queue with an assembly-specific reference.
Source validation, matching provenance, immutable publication, and the normal build checkpoint remain required.
Builders must implement preparation.AssemblyBuilder and preserve published content when CompleteAssembly releases private process state.
The registry builder supports that contract. The local image builder does not.
An unsupported builder rejects the mode without claiming its queued job.

After a confirmed build, the worker releases its owned upload journal and rechecks the current job, lease, cancellation, source, permissions, and repository authority.
One authorization transaction saves assemblyComplete, preserves assemblyReference, clears the active worker reference and lease, and creates one discovery queue record.
The stored state becomes received with phase awaiting-discovery. An audit record identifies the image and source digest.
A repeated completed handoff returns without another queue record or publication.

The discovery worker claims that same source job with its own new execution reference and lease.
It skips image assembly, rechecks retained source integrity and expiry, and uses the existing isolated discovery protocol.
The assembly reference cannot authorize worker discovery requests.
A worker without that discovery inspector leaves the queued job available for a compatible worker.
Assembly-only workers also leave discovery jobs queued.

Build and discovery logs combine once in the final preparation result.
The existing review, conflict, source approval, publication, activation, and repository generation rules remain required.
No published image becomes an approved executable artifact merely because assembly succeeds.

## Recovery, cancellation, and expiry

A confirmed assembly checkpoint can survive interruption before handoff.
The recovery worker uses the original assembly reference and a recovery builder to finish the handoff without rebuilding or starting discovery.
A missing completed journal is safe because registry cleanup preserves immutable content and is idempotent.
Unconfirmed interrupted builds keep the existing cleanup and explicit retry behavior.
A failed or cancelled assembly needs owned cleanup before its recovery record clears.

After handoff, cancellation and expiry remove the discovery queue record.
Discovery rechecks source expiry even when the retention sweep has not run yet.
Current actor revocation blocks discovery. A missing or changed retained source also blocks dispatch.
Successful discovery cleanup uses only its discovery reference and needs no registry writer or assembly builder.
Stale assembly and discovery claims cannot checkpoint or finish another phase with the same worker label.

Old stored jobs omit the two new optional fields and retain the combined workflow.
Upgrade API and preparation workers together before enabling the optional mode.
The JSON record format needs no database migration. The OpenAPI response and generated client include the optional assembly fields.

## Verification scope

Local checks cover a complete handoff through hosted discovery, review, approval, and discovery cleanup without writer credentials on the API.
They also cover confirmed-checkpoint recovery, concurrent writers, cancellation during handoff, waiting cancellation, expiry before and after sweeping, and revoked access.
Registry protocol checks preserve both Go and Playwright artifacts after releasing journals.
The native command rejects unsupported local mode before creating state. API generation and the Angular production build pass.
UI changes add status text and help without new controls or layout. Existing keyboard, theme, and narrow-layout evidence remains applicable.

The fixtures use local databases, local TLS servers, fake registry credentials, and in-process provider doubles.
They start no Docker image, browser, cloud worker, real registry publication, or deployment.
Azure private workload placement, writer identity and grants, command composition, and native isolated discovery remain separate implementation work.
