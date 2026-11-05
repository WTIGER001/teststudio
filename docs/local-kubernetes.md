# Local Kubernetes integration

## Introduction

The dedicated local k3s cluster is running through k3d on this Mac.
Azure implementation is deferred at the user's request. A later deployment script and account setup remain separate work.
Kubernetes previously had protocol, rendering, schema, and container startup checks. The current phase adds actual local cluster verification.

## Concepts

### Verified infrastructure

| Component | Observed state |
| --- | --- |
| Colima profile | teststudio-k3s, separate from the default profile. |
| VM resources | Four CPUs, 6 GiB memory, 20 GiB disk. |
| k3d | v5.9.0, official Darwin ARM64 binary with verified release checksum. |
| k3s | v1.35.5+k3s1, one ARM64 server node. |
| Node | k3d-teststudio-local-server-0 reports Ready=True. |
| System services | CoreDNS, local-path provisioner, metrics server, and Traefik report Running. |
| API | https://127.0.0.1:32768 through the dedicated Docker VM. |
| Private kubeconfig | .data/local-k3s/kubeconfig.yaml, mode 0600. |
| Host client | Helm connects through that kubeconfig using its Secret driver. |
| Default Docker context | Remains default. |
| Existing demo | Remains on port 8040. |

The selected cluster API uses loopback only. Its ingress maps to loopback port 18443.
No user kubeconfig is merged or selected. Every cluster command must use the private kubeconfig or dedicated Docker socket.
The official [k3d release](https://github.com/k3d-io/k3d/releases/tag/v5.9.0) supplies the verified tool and its release checksums.

TestStudio is not installed in this cluster yet.
No application, coded worker, cancellation, deadline, artifact, or recovery scenario is verified on the cluster yet.
The 57-item ledger remains at 48 verified and nine open items.

### Evidence and ownership

Local ownership and tool metadata are in .data/local-k3s/ownership.json and tool.json.
Cluster evidence is in output/local-k3s/cluster-summary.json, nodes.json, system-pods.json, helm-connectivity.json, and cluster-create.log.
The private kubeconfig contains credentials. Do not commit, print, or copy it into application settings.
The cluster and VM remain available for the next installation step.

The initial k3d export reports port zero after creation with an ephemeral API port.
The verified private kubeconfig uses the actual Docker port mapping, observed from the owned server load balancer.
After restarting or regenerating configuration, confirm that mapping before using a new export.

## Tutorials

### Inspect the owned cluster

Use the dedicated socket for Docker-based k3d operations:

```sh
DOCKER_HOST="unix://${HOME}/.colima/teststudio-k3s/docker.sock" \
  .data/local-k3s/bin/k3d cluster list
```

Use the private kubeconfig for host clients:

```sh
HELM_DRIVER=secret .data/kubernetes-tools/helm list \
  --kubeconfig=.data/local-k3s/kubeconfig.yaml \
  --kube-context=k3d-teststudio-local --namespace=default
```

The next application step uses the existing [Helm chart](kubernetes.md), a fresh local PostgreSQL instance, protected settings, and private evidence storage.
Then run one newly authored HTTPS probe through the installed application.
Isolated Go and Playwright workers require an enforced RuntimeClass and network policy before execution.
Check cancellation, deadlines, artifacts, and restart recovery through those actual workers after basic installation works.

### Stop local resources

Stop this cluster and VM when the local integration work no longer needs them:

```sh
DOCKER_HOST="unix://${HOME}/.colima/teststudio-k3s/docker.sock" \
  .data/local-k3s/bin/k3d cluster stop teststudio-local
colima stop teststudio-k3s
```

These commands preserve the cluster's local data. They do not stop the default Colima profile.
Deleting the cluster removes its test workloads and local persistent data. Keep deletion separate from normal stop and resume.

Related guides: [Kubernetes service](kubernetes.md), [workers](kubernetes-workers.md), and [implementation ledger](implementation.md).
