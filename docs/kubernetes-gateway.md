# Kubernetes gateway lifecycle

## Current scope

The provider now implements an owned resource bundle for environment-backed Go and Playwright work.
It connects to the [private initialization authority](remote-gateway.md) and existing dispatch journal.
Targeted local HTTP protocol checks pass. The changed lifecycle checks also pass with the race detector.
Normal executor admission, source assembly, discovery admission, registry capability inspection, and opt-in Helm configuration are now implemented.
Targeted local checks cover the managed result lifecycle and reject unsupported runtime capabilities.
The dedicated local cluster still needs the changed images, installation upgrade, and actual browser and coded API scenarios.
This component does not prove actual Kubernetes execution or close deployment acceptance.
Azure remains deferred.

## Owned resources

| Resource | Purpose |
| --- | --- |
| Runner Pod | Runs the fixed supervisor and uploaded test code in the selected isolated runtime. |
| Gateway Pod | Runs the pinned trusted gateway with a separate service account, TLS material, and initialization ticket. |
| Headless Service | Resolves the execution's private TLS gateway after readiness. It exposes no public route. |
| Runner network policy | Adds only TCP port 8044 access to that execution's gateway. |
| Gateway network policy | Accepts TCP port 8044 from that execution's runner and permits trusted gateway delivery. |

The existing base worker policy retains its private exchange and DNS destinations.
The uploaded runner receives no direct target route from the new policy.
The gateway applies the canonical reviewed-origin, guarded destination, API policy, and credential rules before target delivery.
The two policies use complete execution labels. They cannot widen another execution's runner.
The gateway mounts no uploaded source, runner output volume, application session, database credential, or service account token.
Only its trusted configuration response contains a resolved application credential.

The gateway uses a read-only root filesystem, dropped capabilities, non-root identity, and the selected RuntimeClass.
Its CPU limit is one core. Its memory limit is 256 MiB.
Its Secret mounts supply the server certificate and optional approved exchange CA bundle.
The provider binds the actual verified gateway Pod UID before private initialization can return credentials.
The worker activation gate remains closed until the provider verifies readiness and the private exchange confirms configuration delivery.

## Dispatch and ownership

Network references use the separate go-kubernetes-environment-v1 or playwright-kubernetes-environment-v1 profile.
The gateway name derives from the same execution identifier as the runner name.
The journal freezes the worker specification, gateway image, resource specification hashes, and independent Pod deadlines.
It stores creation-attempt flags and observed resource UIDs.
It stores no raw runner ticket, initialization ticket, transport capability, or resolved application credential.

Each creation attempt is recorded before its API call.
A missing acknowledgement triggers observation of that same resource. It never repeats an uncertain creation request.
A later permitted Ensure call can finish unattempted bundle members while the original activation gate remains closed.
It retains the first active-deadline allowance instead of extending that allowance during retries.
Normal controller recovery revokes work and stops the existing bundle. It does not relaunch uploaded code.

Every resource must match its kind, namespace, full execution reference, specification hash, labels, deadline, and recorded UID.
Normal Kubernetes Service defaults are normalized before comparison.
Changes to resource identity or specification remain ownership failures.
Namespace policy inspection permits the base policy and only journal-verified execution policies.
An unrecognized additional policy invalidates admission.

## Observation and shutdown

The provider waits for the owned gateway's readiness before admission.
After activation, lost readiness or changed network constraints produce an infrastructure failure.
The shared lifecycle stops uploaded work and keeps that failure separate from an assertion result.
A passing report cannot override the failed infrastructure boundary.
Available private evidence remains recoverable. No test duration is inferred from provider state.

Stop attempts both owned Pods concurrently when one workload is absent or its observation is unavailable.
Each live owned Pod receives an active deadline of one second through UID and resource-version guards.
Both Pods also retain their original independent deadline.
Partial creation stays observable while the gateway remains alive.
Recovery can stop and remove that gateway without ever creating the missing runner.

Removal requires the existing result and evidence checkpoint.
It requires confirmed workload termination or authoritative absence before deleting bundle resources.
Each deletion uses both UID and resource-version preconditions and waits for confirmed absence.
The journal becomes removed only after the runner, gateway, Service, and both policies are absent.
A replaced or changed resource remains for inspection and keeps cleanup pending.

## Inventory and missing journals

Inventory checks runner Pods, gateway Pods, Services, execution policies, and durable journal rows.
It returns one execution reference for the complete or partial bundle.
It also detects retained resources after a partial cleanup.

If a dispatch journal row is missing, inventory requires an exact execution-ledger reference before reporting an orphan.
It does not reconstruct missing specification authority from resource annotations.
Such resources remain pending until the original journal is restored or an operator resolves ownership.
Normal controller restart and peer recovery retain the shared database journal.

## Verification and continuation

### Enable the profile

Apply gateway.example.yaml after the core, worker, and optional controller values.
Replace its gateway image with the installation-approved immutable registry digest.
The registry must contain the inspected TLS, API-policy, credential-injection, and private-configuration capabilities.
Environment source runtimes must advertise gateway transport, API-policy support when required, and the supervisor build revision.

workers.gateway.tlsSecret and workers.gateway.exchangeTrustSecret name Secrets in the worker namespace.
The server certificate must cover the generated gateway Service host under the worker namespace's .svc domain.
The optional workers.gateway.caSecret names the public gateway CA Secret in the application namespace.
It supplies ca.crt to the API and controller. Those processes receive no gateway server private key.
The chart creates the separate gateway service account. It creates no certificate material.

secretBindings.existingSecret names an application-namespace Secret containing bindings.json and its referenced credential files.
Use absolute file references under /etc/teststudio-secrets in that project-scoped binding catalog.
The API mounts that Secret. Execution controllers, preparation workers, and uploaded runners do not mount it.
The trusted standalone worker backend also accepts --secret-bindings when it provides the private exchange.
Catalog changes require a process restart. Credential file reads retain the existing rotation behavior.

Only the API and execution controller receive the additional Service and execution-policy write permissions.
Preparation retains discovery control and policy inspection without the gateway resource write role.
All resource permissions remain inside the dedicated worker namespace.

Timing keys include the inspected supervisor build revision and gateway image and resource identity.
Offline and environment profiles remain separate. Existing observations stay retained under their original keys.

The opt-in chart renders 24 valid resources in the local package check.
Missing TLS material, a shared runner account, disabled worker protection, and an unapproved image registry are rejected.
The rendered settings load through the actual Go parser and match the protected base network policy.
See output/kubernetes-gateway/package/summary.json and schema.log.

New local checks cover readiness before activation, full bundle shutdown, checkpoint enforcement, and guarded deletion.
They also cover a lost gateway creation acknowledgement, service restart, partial creation recovery, and foreign gateway rejection.
A stalled runner observation cannot consume the gateway's shutdown opportunity.
A changed policy invalidates the execution, stops both owned workloads, and remains retained through incomplete cleanup.
Selected existing offline provider checks still pass.
Shared lifecycle checks reject a failed boundary before activation and retain an explicit unknown result after a reported pass.
No image rebuild, cluster deployment, cloud connection, browser process, or new catalog run occurs for this component.

Normal execution and preparation now use the approved environment profiles through the supported service roles.
Build changed runtime inputs once and verify the complete browser and coded API scenarios on the dedicated local cluster.
Use the existing source queue, immutable publication, run API, private evidence, timing, and cleanup workflow.
Keep the retained offline runs and existing installation data.
Do not generate repeated runs for calibration.
Reuse the retained offline execution evidence. Keep AWS and Azure verification outside this local phase.
