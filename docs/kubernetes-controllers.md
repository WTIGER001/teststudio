# Separate Kubernetes execution controllers

## Introduction

The Helm chart can run execution controllers separately from the API, sign-in, and private worker exchange.
Set controller.enabled=true to select the existing headless controller role.
The API then receives --execution-worker=false. Its other configured services remain enabled.
This package supports independent replica counts and restarts without making each controller an API replica.
The default remains the combined single-instance profile.

Three targeted local profiles and four rejected configurations pass Helm rendering and pinned schema checks.
Actual separate-controller deployment and peer failover now pass on the dedicated local cluster through Helm revision 9.
See [local controller failover](local-kubernetes-controllers.md) for preserved evidence and the acceptance 52 audit.
The acceptance count is 49 of 57. Azure implementation remains deferred.

## Concepts

### Process and configuration

Controllers use the same immutable service image and the existing --serve-role=controller command.
The command overrides the image entrypoint and receives only its supported controller flags.
Its configuration contains the shared artifact adapter and Kubernetes worker profile.
The controller receives database and registry reader references. It receives no OIDC client secret or registry writer reference.
Optional project-scoped HTTP trust mounts match the API's trust configuration.

Each controller runs as user 65532 with a read-only root filesystem, dropped capabilities, and private temporary storage.
It uses a separate ServiceAccount with the worker profile's bounded dispatch permissions and rotating projected Kubernetes token.
controller.serviceAccount.annotations and controller.podLabels support installation workload identity.
Protected selector labels cannot be replaced.
A configuration checksum restarts controllers when their artifact or worker configuration changes.

The health listener exposes only the existing controller health routes on port 8043.
The package creates no controller Service or Ingress. Its NetworkPolicy denies ordinary inbound traffic.
Kubelet health probes remain available. The API continues serving private worker input, progress, reports, and evidence.
Controllers have no preparation journal mount and start no preparation or discovery loop.
See [controller responsibilities](execution-controller.md).

### Storage and replicas

controller.replicas controls the separate deployment. It defaults to one.
Multiple controllers create a PodDisruptionBudget with maxUnavailable=1 and use rolling updates.
The shared database remains PostgreSQL.

Filesystem artifacts require an actual shared ReadWriteMany claim whenever separate controllers are enabled.
This requirement applies even with one controller because the API also uses that evidence store.
The chart rejects ReadWriteOnce configuration. It does not convert a claim or copy existing evidence automatically.
Provision shared storage, preserve existing bytes and references, and verify storage-driver behavior before switching an installation.
An object-store installation uses its existing adapter and workload identity instead of a shared filesystem claim.
Preparation journals remain a separate private claim with independent lifecycle requirements.

Concurrent run claims and recovery use the existing durable run and execution journals.
A successful claim emits Execution run claimed with project, run, and worker identifiers through the structured service logger.
These fields let operators identify the owning controller without exposing tokens, credentials, test bodies, or target URLs.
A disruption budget does not prove replica failover or prevent an abrupt process failure.

## Tutorials

### Select the profile

Merge deploy/kubernetes/controller.example.yaml into the installation's complete private values.
Keep the worker profile enabled and replace the example shared claim with the actual installation claim.
For object storage, retain the installation artifact adapter and supply its controller workload identity.
Then upgrade the chart using the dedicated installation kubeconfig and context.

Keep the API's private worker exchange route available across controller changes.
If a private registry restricts client peers, allow the controller selector with reader access only.
Preserve separate preparation writer credentials and worker network restrictions.

### Run the targeted package check

```sh
.data/kubernetes-tools/venv/bin/python scripts/verify-kubernetes.py \
  --controller-only \
  --helm=.data/kubernetes-tools/helm \
  --kubeconform=.data/kubernetes-tools/kubeconform \
  --output=output/kubernetes-controller
```

This check renders the unchanged default, shared filesystem controllers, and object-store controllers.
It verifies command roles, configuration separation, service accounts, secret references, probes, selectors, storage, budgets, and absent public routes.
It rejects missing workers, ReadWriteOnce evidence, selector overrides, and zero replicas.
It contacts no Kubernetes cluster or cloud account.

### Recreate local failover verification

The current local installation already passes this workflow. Reuse its saved records during ordinary continuation.
Preserve the installation's artifact claim and retained run identifiers before changing storage.
Prepare local shared storage and copy only this installation's evidence during a controlled writer stop.
Keep the previous claim for rollback. Do not rerun tests to recreate their history.
Deploy two controllers with the API execution worker disabled.
Use a new bounded case and retain the successful claim's controller identity.
Interrupt that exact controller while its peer remains available.
Verify the original run, single execution, retained observation, private evidence, and cleanup through the surviving deployment.
Treat co-located replica verification separately from node or storage failover.

Related guides: [Kubernetes installation](kubernetes.md), [local lifecycle](local-kubernetes-lifecycle.md), and [worker contract](kubernetes-workers.md).
