# Local Kubernetes worker isolation

## Introduction

The dedicated local k3s cluster now runs an actual gVisor diagnostic Pod and enforces selective worker network policy.
This verifies worker prerequisites. Fresh Go and Playwright suites now also pass [actual source execution](local-kubernetes-source-execution.md).
Cancellation, deadlines, and recovery remain pending.
The verified acceptance total remains 48 of 57. Azure implementation remains deferred.

## Concepts

### Runtime installation

The ARM64 gVisor release is 20261005.0. Its official archive and SHA-512 sidecar pass checksum comparison before extraction.
Installation follows the official [gVisor package guide](https://gvisor.dev/docs/user_guide/install/).
The complete package includes runsc, containerd-shim-runsc-v1, and the accompanying gvisor-bin directory.
Those files reside together in /usr/local/bin inside k3d-teststudio-local-server-0.
No runtime is installed into the default Docker profile or onto the Mac host.

The selected runtime uses systrap. See [gVisor platforms](https://gvisor.dev/docs/user_guide/platforms/).
Containerd loads the owned version-three runtime drop-in from /var/lib/rancher/k3s/agent/etc/containerd/config-v3.toml.d/teststudio-runsc.toml.
Its handler is teststudio-runsc. Its shim configuration is /etc/containerd/teststudio-runsc.toml.
RuntimeClass teststudio-isolated selects that handler and Linux ARM64 nodes.
The existing generated containerd configuration and default runc handler remain intact.
The owned node container restarts once to load the additional runtime. Its existing persistent volumes and private API mapping remain available.

The first diagnostic exits zero and reports Linux version 4.19.0-gvisor, the gVisor startup log, and UID 1000.
This is runtime evidence, not an inference from a RuntimeClass name.
See [containerd configuration](https://gvisor.dev/docs/user_guide/containerd/configuration/) for the shim options.

### Private routing and policy

The isolation phase uses Helm revision 3 and .data/local-k3s/worker-values.json.
The current source-enabled installation uses revision 8 and .data/local-k3s/preparation-values.json.
The worker namespace is teststudio-workers. Its RuntimeClass, restricted service account, projected controller identity, and network policy are installed.
The chart uses workers.privateIngress.enabled=false with an externally supplied local TLS proxy.
The proxy runs in teststudio-private-routing, accepts the worker hostname, and exposes only the private execution path prefix.
Its fixed backend is the chart's teststudio-worker Service. It exposes no general application route.
Its own policy permits outbound traffic only to the controller's private listener and cluster DNS.
The TLS peer port is 8443. The internal Service exposes HTTPS port 443.

The diagnostic Pod uses the real worker namespace, worker label, service account, RuntimeClass, and installed network policy.
It has no Kubernetes API token and runs as UID 1000 with a read-only root filesystem and dropped capabilities.
It mounts only a public test CA for this diagnostic. Normal worker Pod specifications remain unchanged.

The check first confirms that four destination services accept TCP connections from the owned node.
The isolated Pod then reaches the permitted TLS exchange with certificate validation.
The exchange rejects an unowned execution, and the private proxy rejects an API route.
Connections to the API, artifact service, PostgreSQL, and local probe target are denied.
The denied services remain available in the separate node-side check.

The first permitted connection attempts receive connection refused during policy convergence.
Four bounded readiness retries then reach the private exchange. No destination policy is relaxed.
The production supervisor already has bounded input-acquisition retries.
The diagnostic treats refused connections as denial only alongside the successful destination baseline and permitted exchange checks.

### Evidence and cleanup

Runtime package metadata and the exact installed configuration remain in .data/local-k3s/gvisor.
Runtime execution evidence is output/local-k3s/gvisor/runtime-pod.json, runtime.log, and runtime-summary.json.
Network evidence is output/local-k3s/isolation/baseline.json, pod.json, diagnostic.log, summary.json, and cleanup.json.
The network diagnostic removes its temporary Pod, CA ConfigMap, and node-side helper after verification.
The initial completed runtime diagnostic Pod is also removed after its evidence is saved.
The RuntimeClass and private route remain available for actual coded worker integration.
No browser, AWS, or Azure execution occurs in these checks.

## Tutorials

### Run the focused routing checks

```sh
.data/kubernetes-tools/venv/bin/python scripts/verify-kubernetes.py \
  --routing-only --helm .data/kubernetes-tools/helm \
  --kubeconform .data/kubernetes-tools/kubeconform \
  --output output/local-k3s/routing-checks
```

This checks the base profile, managed private ingress, and external private routing.
Four invalid configurations remain rejected. It does not contact a cluster.

### Inspect the actual local isolation result

Read output/local-k3s/isolation/summary.json and cleanup.json before repeating the check.
The fixture source is tests/kubernetes/isolation. The private TLS proxy source is tests/kubernetes/private-ingress.
Both binaries use the existing scratch fixture Dockerfile with executable file permissions.
Their imported manifest references are retained in the local output metadata.
The network verifier accepts the immutable diagnostic image through --image:

```sh
python3 scripts/verify-kubernetes-isolation.py --image=IMMUTABLE_LOCAL_DIAGNOSTIC_IMAGE
```

The current diagnostic binary is .data/local-k3s/isolation-check.
The verifier stages that binary under a unique temporary node path for the positive connectivity baseline, then removes it.
It uses the dedicated Docker socket and owned node, never an inherited Kubernetes context.

### Recreate the source workflow

This setup now passes in the current installation. Reuse its [source execution records](local-kubernetes-source-execution.md) instead of recreating it during ordinary continuation.
Use these steps only for a fresh installation:

Create a local HTTPS OCI registry with separate controller reader and preparation writer credentials.
Publish the approved supervised Go and Playwright runtime images there, with trust for the private exchange authority.
Keep registry credentials outside worker input and execution containers.
Enable the existing preparation worker and its private journal claim through installation configuration.
Publish fresh independent source bundles through the normal API, then execute one Go test and one Playwright test.
Retain timing, result, logs, and browser evidence before testing cancellation, deadlines, and recovery.
The source execution guide records actual coded runs separately from these diagnostic prerequisites.

See [local cluster operation](local-kubernetes.md) and [worker configuration](kubernetes-workers.md).
