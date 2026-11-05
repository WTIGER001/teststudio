# Local Kubernetes application networking

## Introduction

This workflow exercises environment-backed browser and coded API suites on the dedicated local Kubernetes installation.
It preserves the existing offline runs, identity providers, evidence storage, and two-controller configuration.
Azure and live AWS verification remain separate work.

The actual local installation now passes both scenarios at Helm revision 14.
The API run takes 20 ms and retains seven verified artifacts.
The browser run takes 263 ms and retains eight verified artifacts, including screenshots and a trace.
Both runs retain complete evidence and cleaned execution records.
The worker namespace contains only its base network policy after cleanup.
No worker or gateway container remains running.

The API run identifier is 20261105T190135.888651628-f45615b1da24f0d003db1922.
The browser run identifier is 20261105T190237.146588449-543e82e7b812fb2d345278ba.
The twelve previous offline run identifiers remain unchanged.

## Concepts

The independent target lives in tests/kubernetes/environment.
Its browser page loads availability through a button and a same-origin request.
Its private API requires HTTPS and a fixture credential.
The private response reflects that credential so the API suite can check gateway redaction.
The uploaded suite contains no credential value.

The browser scenario uses an explicitly permitted local HTTP origin.
The API scenario uses HTTPS with normal certificate validation and a private fixture certificate authority.
These scenarios do not establish Chromium trust for private HTTPS certificates.
Both scenarios use the authenticated TLS transport between their supervisor and trusted gateway.

The Go suite also checks that unreviewed methods and paths return forbidden responses.
It checks that the uploaded worker cannot connect directly to the target.
The verifier checks measured duration, complete evidence, private download checksums, credential absence, and execution cleanup.
Existing offline history must retain its original twelve run identifiers.

Catalog discovery remains offline even when its destination suite requires an environment.
Normal publication still validates the discovered environment profile and catalog.
Test execution requires the separate reviewed target and gateway admission.
Typed Kubernetes list responses can omit item kind and API version.
The provider restores those fields from the resource endpoint while preserving strict object ownership checks.
A recorded worker failure keeps its original outcome when cleanup stops the gateway.
The upgraded controller cleans the retained failed execution without dispatching another worker.

## Tutorials

### Prepare changed images

Use the existing service, gateway, and supervisor build outputs for the selected source revision.
Run this command from the repository root:

```sh
python3 scripts/prepare-kubernetes-gateway-images.py
```

The script reuses unchanged images and adds the public local certificate authority to the runtime overlays.
It publishes immutable registry references through a temporary private publisher directory.
It removes temporary credentials and image archives after publication.
Its durable image checkpoints live in output/local-k3s/gateway/images.json.

### Upgrade the owned installation

```sh
python3 scripts/setup-kubernetes-gateway.py
```

The script uses only the explicit owned cluster and private kubeconfig.
It saves the previous Helm values before enabling the optional gateway profile.
It preserves the shared evidence claim and two execution controllers.
The four-core local profile reserves 250 millicores for preparation so one two-core worker and its gateway can fit.
Worker CPU limits and timing profiles remain unchanged.
Fixture credentials and certificate keys remain under .data/local-k3s/gateway and Kubernetes Secrets.
Only the API mounts the application credential catalog.
The controller mounts the public gateway certificate authority.
The preparation process receives no gateway initialization authority.
The private exchange ingress policy permits both isolated workers and trusted gateways from the owned worker namespace.

The installation checkpoint records each Helm revision before waiting for deployment.
If observation expires, inspect the same revision before continuing.
Do not create another deployment solely because a wait expires.

### Publish and execute both suites

```sh
python3 scripts/verify-kubernetes-gateway.py
```

The script uses normal OIDC sign-in and product APIs.
It creates a separate k3s-network project and gateway-fixture application.
It saves preparation and run identifiers before polling them.
It never replaces an active or uncertain operation automatically.
It signs out its owned sessions when finished.
It starts no host browser and leaves the user's local demonstration unchanged.

After a concrete implementation fix, a failed preparation can receive one corrected submission:

```sh
python3 scripts/verify-kubernetes-gateway.py --runner=go --retry-preparation-after-fix
```

The previous preparation must be failed and fully cleaned.
Its diagnostics and identifier remain retained beside the replacement checkpoint.
This option does not rerun a completed test.

After a concrete execution fix, retain the unsuccessful run and submit one corrected execution:

```sh
python3 scripts/verify-kubernetes-gateway.py --retry-run-after-fix
```

Both previous executions must be terminal and cleaned.
The script preserves their results, artifacts, logs, and execution records before replacing its continuation pointers.

### Inspect retained results

Read output/local-k3s/gateway/references.json for the original operation identifiers.
Read each runner's summary, run, preparation, artifact, log, and execution files in that directory.
Private download grants stay in memory and are not written into those summaries.
The stable acceptance count changes only after a complete acceptance requirement passes.
These scenarios advance 41, 51, 56, and 57. The verified count remains 50 of 57.
### Verify gateway lifecycle boundaries

```sh
python3 scripts/verify-kubernetes-gateway-lifecycle.py --case=cancel
python3 scripts/verify-kubernetes-gateway-lifecycle.py --case=recovery
```

Both actual cases now pass on Helm revision 14 without rebuilding any image.
One new lifecycle suite uses the existing authenticated target and runtime.
The selected test writes a private output marker only after its gateway request succeeds.
The verifier checks that marker inside the owned worker before the requested action.
It confirms the worker UID, gateway UID, original execution identifier, and controller identity.
It saves each operation identifier before polling or interrupting a process.
Reusing the command observes the existing run and never dispatches a replacement.

Cancellation run 20261105T190905.111792480-6395b63f6393d2b50ee25bda reaches cancelled and cleaned.
Its interrupted attempt remains unknown, with unavailable duration and incomplete evidence.
Five private diagnostic artifacts pass checksum and credential-absence checks.

Recovery run 20261105T191000.981246578-8b2e6eef59d96658294be7ea receives one verified controller SIGKILL after its authenticated request.
Its original execution retains a passing observation measured at 12010 ms.
The run remains incomplete with unknown outcome. Its execution evidence is complete, and cleanup reaches cleaned.
Five private diagnostic artifacts pass verification. The same controller Pod restarts, and both controllers return Ready.

Both cases keep one execution each. Their worker Pods, gateway Pods, Services, and execution policies are absent.
CRI reports no running worker or gateway container. All twelve previous offline runs remain unchanged.
The two checks use one preparation and pass once each. They start no host browser or cloud test.
Retained records live in output/local-k3s/gateway/lifecycle.
The next implementation step is the ECS trusted gateway integration with targeted local checks.
