# Kubernetes execution workers

## Current scope

TestStudio now supplies a Kubernetes Pod adapter and explicit server configuration.
Both offline Go and Playwright execution use the shared hosted lifecycle and private worker exchange.
The Helm profile supplies the private listener route, worker namespace, service account, controller permissions, and worker network policy.
Local protocol and packaging checks pass. A dedicated [local k3s cluster](local-kubernetes.md) now has a Ready node.
Core application installation, OIDC sign-in, and a newly authored HTTPS probe now pass.
Actual local gVisor execution and selective worker network denial also pass.
Fresh Go and Playwright suites now pass actual preparation, publication, execution, timing, and private evidence checks.
See [local source execution](local-kubernetes-source-execution.md). Actual cancellation, test timeout, and interrupted-controller recovery now pass.
See [local lifecycle](local-kubernetes-lifecycle.md). Independent Pod deadline enforcement now passes while guest processes and the controller are stopped. Actual controller peer failover now passes and closes acceptance 52. See [local controllers](local-kubernetes-controllers.md).
Acceptance 51 remains open across providers. The total stays 50 verified items out of 57.

The separate preparation command now connects registry assembly, immutable publication, and hosted discovery to the existing source queue.
Native setup is documented in [Hosted preparation](hosted-discovery.md).
The optional [preparation package](kubernetes-preparation.md) adds its Helm deployment, private durable journals, separate service account, and registry writer references.
The catalog still requires a prepared and approved immutable artifact.
Source import does not automatically approve a new registry image.
Environment-backed browser and API artifacts require the explicitly configured trusted gateway profile.
The [gateway bundle component](kubernetes-gateway.md) now implements provider lifecycle controls and passes targeted local checks.
Normal admission, discovery, image approval, and installation wiring now pass targeted local checks. Actual network execution remains unverified.
Live AWS verification remains separate. Azure implementation and verification remain deferred.

See [Shared hosted execution](hosted-execution.md), [Registry inspection](registry-inspection.md), and [Private worker exchange](worker-exchange.md).

## Configure the installation

Start with the [core Kubernetes profile](kubernetes.md).
Use the changed service image and its registry distribution digest when preparing a deployment.
The local ARM64 package for this batch is recorded in output/kubernetes-workers/service-image.json.
A local image identifier is not a registry distribution digest.
Protocol and rendering checks contact no cluster. The separate [local isolation verification](local-kubernetes-isolation.md) uses the dedicated test cluster.

The worker profile is disabled by default.
Select deploy/kubernetes/workers.example.yaml alongside the core values when preparing the enabled configuration.
The example contains new independent names and references. It contains no credentials or existing application configuration.

The installation provides these dependencies:

- PostgreSQL and the configured private evidence store.
- A dedicated worker namespace and an explicitly selected RuntimeClass.
- A CNI that enforces Kubernetes NetworkPolicy.
- A dedicated worker-only private ingress controller, private DNS, and the worker TLS Secret.
- Approved supervised source runtimes and prepared images in the configured registry namespace.
- Registry reader references for the controller and image pull configuration for worker nodes.

Helm creates the dedicated worker namespace and its restricted Pod Security policy.
It creates no RuntimeClass, registry credential, database credential, or TLS Secret.
Create or synchronize the worker image-pull Secret in the worker namespace before coded dispatch.
The chart does not copy the API registry reader into that namespace.
Operators supply the existing RuntimeClass and private ingress implementation.
The selected private ingress controller routes only the worker exchange. It must not provide a route to other application targets.
The worker namespace cannot share the API namespace in this chart.
The private ingress class and host must differ from the public application route.

### Existing private TLS routing

workers.privateIngress.enabled defaults to true and creates the worker Ingress resource.
Set it to false when the installation supplies an existing private TLS proxy or gateway.
The chart still creates the worker Service, controller listener, permissions, and restrictive network policies.
The ingress class and TLS Secret inputs are then optional. The separate worker origin and exact peer selectors remain required.
Route that origin only to RELEASE-worker in the application namespace on Service port 80, targeting the private listener on port 8042.
Retain hostname and certificate validation. Configure the worker runtime to trust the exchange authority.
The existing route must expose no UI, API, artifact, or general target access through the worker's permitted peer.
Its actual Pod TLS port must match workers.privateIngress.port.

The dedicated local cluster uses this option with a small test-only TLS proxy in a separate restricted namespace.
It exposes only the worker path prefix and exact worker host. Its outbound policy permits only the controller and cluster DNS.
This proxy is a verification fixture, not a production ingress implementation.

The private ingress peer port names the actual ingress Pod TLS port.
Its default is 443. Set it to the ingress container's port when service translation uses another target port.
Workers can reach only that peer and the selected DNS Pods through the supplied network policy.
Policy programming can briefly reject the permitted endpoint after Pod creation. Keep bounded input retries without relaxing the destination policy.
The dedicated namespace contains exactly the supplied worker policy.
Additional namespace network policies require review because Kubernetes combines their permissions.
The adapter rejects an unrecognized policy instead of assuming the intended restriction remains effective.
The staged gateway component additionally permits only journal-verified execution policies.
The currently installed offline profile keeps its existing single policy.

The namespace and worker network policy have Helm keep annotations.
Drain active work before uninstalling. Retained namespace and policy resources require explicit operator cleanup afterward.
This prevents uninstall from removing the worker restriction while retained Pods still exist.

## Server settings

The server accepts --kubernetes-workers-config=FILE or --kubernetes-workers-env=VARIABLE.
The loader accepts one bounded JSON object with worker and registry settings.
It rejects unsupported fields and unsafe credential reference names.
The Helm ConfigMap renders that object as kubernetes-workers.json.
The private listener uses --worker-addr=0.0.0.0:8042 behind the private TLS ingress.

Kubernetes execution requires execution-location=kubernetes and PostgreSQL.
It cannot combine local administrator mode, local coded image lists, or local gateway images with this profile.
Run creation checks that the configured coded driver matches the selected runner and execution location.
A matching Kubernetes driver is admitted. A local driver cannot execute through a hosted location.
The --prepared-sources flag selects hosted discovery when Kubernetes settings are present. The --source-bundles flag enables source staging.
The artifact-only service role cannot enable hosted execution or its private listener.
The normal API and artifact origins remain separate from the worker origin.

The controller service account receives only these worker-control permissions:

| Scope | Permission |
| --- | --- |
| Worker namespace | Create, read, list, patch, and delete Pods. |
| Worker namespace | Read the selected runner service account and list network policies. |
| Cluster | Read only the selected worker namespace and RuntimeClass. |

The profile grants no Secret reads, Pod exec, Pod logs, or workload permissions in the API namespace.
The controller uses a projected, rotating API token and mounted cluster CA certificate.
The adapter rereads that token for each request. It does not use a developer kubeconfig or cloud credential discovery.
The controller registry reader uses dedicated TESTSTUDIO_REGISTRY_ environment references.
Registry credentials never enter worker Pod settings.
Optional registry CA references mount separately from the API trust bundle.

The worker service account has no automatic API token mount and no workload identity annotation.
Worker Pod specifications include only scoped job identity and fixed execution settings.
Admission mutations that add credentials, extra containers, or mounts invalidate the expected specification.

## Dispatch and admission gate

The run service first creates the durable execution reference.
The shared lifecycle freezes its input with controller activation still pending.
The Kubernetes adapter reserves one database dispatch record before contacting the creation API.
Only the winning reservation can create the Pod. Another controller discovers the same reference.
The record retains a token fingerprint and specification hash, not the worker ticket or API token.
Removed records remain as replay tombstones.

The adapter checks the namespace restriction, runner service account, RuntimeClass, and exact network policy before creation.
It uses one deterministically named Pod with restartPolicy=Never.
The profile creates no replacement Job controller.
The Pod has one fixed supervisor entrypoint, non-root execution, a read-only root filesystem, dropped capabilities, and bounded resources.
Temporary and output storage use owned emptyDir volumes.
The Pod UID becomes the worker instance identifier.

The returned Pod must match the complete reference, immutable image, specification, and deadline.
The driver checks provider identity again before activating input delivery.
Pending activation returns a retryable transport response. No uploaded code receives the work packet before activation.
The existing supervisor retries that response within its bounded input acquisition policy.
The activation flag stays in controller metadata and never enters worker JSON.

A rejected or uncertain creation never opens the gate.
Recovery revokes the pending ticket and stops the discovered Pod without activating it or launching another Pod.
An uncertain creation therefore remains an unmeasured unknown observation when no test ran.
An already accepted private report can recover after a later result checkpoint failure.
The local fixture proves both paths without repeating execution.

## Timing, cancellation, and recovery

Runtime timing identity includes the supervised runtime, architecture, cluster scope, RuntimeClass, node selection, and fixed resource profile.
Changing the execution pool starts a separate baseline.
Provider elapsed time never replaces observed test duration.

The supervisor applies the frozen absolute deadline.
The Pod also receives activeDeadlineSeconds as an independent runtime bound.
Cancellation reduces that Pod deadline to one second using UID and resource-version guards.
The controller waits for terminal state before resource deletion.
A still-live or uncertain Pod cannot pass cleanup as a completed worker.

The client requests a four-second API deadline and uses a five-second HTTP bound.
An unacknowledged creation has a thirty-second discovery guard before initial absence is accepted.
A previously observed live Pod that disappears is not proof of process termination.
That case remains pending for recovery or operator investigation.
A failed phase without a terminated container status also remains pending. A phase alone cannot prove a disconnected worker stopped.
The adapter never force-deletes an unconfirmed live worker to make cleanup appear successful.

A confirmed terminal state is retained in the dispatch journal.
Final deletion uses the exact Pod UID and resource version.
The adapter waits for confirmed absence within the cleanup call deadline after deletion acknowledgement.
Intermediate reads retain ownership checks. A foreign replacement never passes cleanup or receives another deletion request.
The normal result and evidence checkpoint must permit cleanup before deletion.
Private staging and quota reservations clear only through the existing exchange cleanup contract.
The worker ledger records cleanup completion afterward.

Inventory combines namespace Pods and outstanding database dispatch records.
It validates installation, profile, reference, resource name, and ownership.
Recovery can adopt a matching orphan only when its durable execution ledger exists.
Configuration must preserve the same API origin and worker namespace while outstanding work recovers.
Unexpected ownership, altered specifications, and uncertain node termination remain explicit errors.

## Local verification and continuation

The local TLS fixture exercises the real Pod adapter, shared lifecycle, private exchange, SQLite ledger, and private filesystem evidence.
It uses report-contract observations for Go and Playwright. It starts no real cluster workload or browser.
Focused race checks cover normal outcomes, deadlines, shared creation, lost acknowledgement, checkpoint recovery, ownership rejection, and blocked admission mutations.
Mounted CA and token checks prove controller token rotation.
The rendered Helm settings must load through the real settings parser and match the adapter's exact network policy.

Targeted chart checks render only the default and worker-enabled profiles.
They reject four invalid worker configurations and validate the generated worker Pod against the pinned schema.
Five native and packaged startup probes reject SQLite, artifact-only execution, or the wrong execution location before dispatch.
Packaged probes use no network and remove their owned containers.

Evidence is output/kubernetes-workers/provider.log, package/summary.json, startup/summary.json, and startup/cleanup.json.
The service image rebuild reuses the UI, private viewer, and unchanged health binary.
No source runner image rebuild, browser suite, registry push, cloud operation, or cluster test occurs.
The local demonstration retains its database and evidence.

The separate preparation package now preserves the admission gate and durable cleanup ordering through explicit Helm settings.
Then continue AWS and Azure adapters through the same lifecycle contract.
Actual deployment, CNI isolation, RuntimeClass behavior, OIDC, and multi-replica acceptance remain separate verification work.

## Platform references

Pod lifecycle behavior follows the [Kubernetes Pod documentation](https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/).
Deletion uses [Kubernetes API preconditions](https://kubernetes.io/docs/reference/using-api/api-concepts/).
The request bound uses the API server's [request deadline filter](https://github.com/kubernetes/apiserver/blob/v0.34.1/pkg/endpoints/filters/request_deadline.go).
