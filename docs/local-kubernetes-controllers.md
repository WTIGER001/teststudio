# Local Kubernetes controller failover

## Introduction

The dedicated local installation now runs one API and two separate execution controllers through Helm revision 9.
The API keeps sign-in, catalog workflows, and private worker transport. Its execution worker is disabled.
The original artifact bytes remain available through a local shared NFS claim.
A bounded actual run proves recovery through the surviving controller without duplicate runs or executions.
Acceptance 52 is complete. The total is 49 verified of 57, with eight open items.
Azure implementation and live cloud verification remain deferred.

## Concepts

### Shared evidence

The local fixture runs the Kubernetes SIGs Ganesha image with verified Linux ARM64 metadata and an immutable digest.
Its upstream image is ghcr.io/kubernetes-sigs/nfs-ganesha@sha256:7595e489709c854d0f9c5c9eb6653f57e9958afba35b5e59379d7d4a3016df5b.
See the official [image package](https://github.com/kubernetes-sigs/nfs-ganesha-server-and-external-provisioner/pkgs/container/nfs-ganesha) and [core configuration](https://github.com/nfs-ganesha/nfs-ganesha/blob/V6.5/src/doc/man/ganesha-core-config.rst).
The selected manifest's architecture is checked before import. A digest for another architecture is not accepted.

The server mounts the original teststudio-artifacts claim and exports its existing directory as /evidence.
The new teststudio-shared-evidence claim binds the retained teststudio-local-shared-evidence NFS volume with ReadWriteMany access.
The API and both controllers use that same claim at /data. Artifact paths and database references stay unchanged.
No test reruns, catalog imports, evidence copies, or history reconstruction occur during this storage switch.
A private download verifies an existing browser screenshot after the switch with its retained size and checksum.

The fixture uses NFSv4.1 over TCP 2049 inside the dedicated cluster.
Exports admit only the owned node address and Pod network. Network policy restricts ingress and denies outbound traffic.
The server has no Kubernetes token, database credential, registry writer credential, host network, or public route.
It receives DAC_READ_SEARCH and SYS_RESOURCE capabilities for its filesystem server duties.
Its normal I/O flusher setting remains enabled. Writable runtime directories are separate temporary mounts.
The supervisor starts rpcbind inside the same Pod. Its other RPC ports are not exposed through the Service or ingress policy.
This is a local storage fixture. It does not establish production storage availability, node failover, or recovery after server loss.

### Controlled cutover

The owned API drains before the Helm upgrade. PostgreSQL, preparation journals, and the original artifact claim remain intact.
Private rollback values are .data/local-k3s/before-controller-values.json.
The active installation values are .data/local-k3s/controller-values.json with mode 0600.
The registry fixture adds only the controller peer to its reader-side network policy.
Preparation retains its separate writer identity and journal claim.

The setup script checks the actual installed profile before repeating changes.
An already deployed matching profile returns without mutations.
An existing pending controller deployment requires observation of that same deployment rather than another upgrade.
Source image imports use the explicit ARM64 platform, inspect the resulting reference, and remove their temporary archives.
The UI and private trace viewer reuse cached builds.

### Actual peer failover

Run 20261105T161905.063920611-46e31390481d4b43ad92cdba selects the newly authored K3S-RECOVERY-001 check.
Execution 8fd66f4eb0dc4065fc11bde9e31e1fe3ec76fb6e2432008ba83b1935e3b603de remains its only execution identity.
The claim log identifies exactly one owning controller and its worker identifier.
The verifier checks the exact Pod UID, container identity, namespace, and live process before signaling that controller.

The owner stops before graceful cleanup. Deletion costs select that exact Pod while the pool temporarily scales to one.
The verifier confirms the intended deletion target before SIGKILL.
This prevents a replacement controller from participating in the recovery observation.
See the [ReplicaSet deletion preference](https://kubernetes.io/docs/concepts/workloads/controllers/replicaset/#pod-deletion-cost).
The original peer remains Ready with the same Pod UID, container identity, process, and restart count through cleanup.

That peer recovers the passing report, its measured 20000 ms observation, and complete private evidence.
A private go-events.jsonl download passes size and SHA-256 checks.
The interrupted logical run remains incomplete with unknown outcome. Recovery does not manufacture a completed run.
The original worker disappears and its execution reaches cleaned.
The suite contains exactly the eight saved lifecycle observations, including two retained deadline-fixture errors and this failover case.
There is one attempt for the failover case and no duplicate logical run.
Both controller replicas are restored. Kubernetes and CRI report no remaining test worker.
This verifies co-located controller failure on one local node. It does not imply node or storage fault tolerance.

### Acceptance 52 audit

The actual peer case proves the criterion's replica-failover requirement without duplicate logical runs.
The focused TestNotificationRestartRetriesUncheckpointedAcceptanceWithTheSameIdentity check proves its notification requirement.
That check uses an actual local HTTP receiver and a second database connection after acceptance loses its checkpoint.
It verifies two sends with the same delivery and run identities, retained unknown acceptance, and exactly one test execution.
The documented at-least-once behavior remains explicit. A message can repeat without repeating a test.
The focused check passes in 0.05 seconds. Its log is notification-recovery-check.log beside the actual cluster evidence.
Cross-provider deployment scenarios remain under the other open acceptance items.

## Tutorials

### Inspect the saved result

The main records are output/local-k3s/controllers/summary.json, installation.json, and failover-reference.json.
Adjacent files retain the claim, original owner, surviving peer, run, attempts, execution journal, logs, artifacts, and private download.
The previous browser evidence identifier and checksum are retained in the reference file.
The last Helm upgrade log records revision 9.
The original artifact claim remains the backing store. Keep it while any exported evidence is required.

### Resume or recreate the fixture

```sh
python3 scripts/setup-kubernetes-controllers.py
python3 scripts/verify-kubernetes-failover.py
```

The setup command reuses the verified source and NFS image metadata under output/local-k3s.
Run it only on this owned installation. It does not select an inherited cluster or copy user credentials.
The verifier reuses its saved run and never repeats a completed interruption.
An armed interruption without a recorded outcome requires inspection before another signal.
Its cleanup restores controller capacity, including after a failed assertion.
A pool restored before peer-only recovery is proved cannot later pass as that observation.

### Continue the plan

Keep the existing runs and infrastructure for the next local milestone.
Verify a second installation-specific OIDC provider and project-scoped directory behavior on this Kubernetes profile.
Do not repeat the completed source, lifecycle, deadline, and failover cases to generate more history.
Keep provider storage acceptance, remaining cross-profile identity and timing, and cloud deployment criteria open.

Related guides: [controller package](kubernetes-controllers.md), [local lifecycle](local-kubernetes-lifecycle.md), and [acceptance ledger](implementation.md).
