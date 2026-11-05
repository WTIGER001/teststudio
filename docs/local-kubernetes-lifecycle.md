# Local Kubernetes lifecycle

## Introduction

New bounded Go checks exercise cancellation, test timeouts, and an abrupt controller interruption on the dedicated local cluster.
They use the normal source preparation, publication, run, evidence, and cleanup contracts.
They use no cloud account or external application. Azure work remains deferred.
The acceptance total is 50 of 57. These results advance the Kubernetes portions of 41, 51, 52, and 56.

## Concepts

### One source preparation

The independent sources live in tests/kubernetes/suites/lifecycle.
The targetless suite lifecycle-local belongs to project k3s-coded and application workers.
The checks wait inside isolated workers without external traffic, credentials, or permanent files.
Catalog timeouts bound each check. Each manual run selects one test and disables retries.
The preparation is bd554a324480fa7e3e6c6c1b4c2cb723c05ba30c1dfcc4c5e6731038b70c192c.
The published release is fb02607ae0cfb0e069d3297e7af84a18ae248fa02f488f64ac71ed57a77393a8.

### Cancellation

Run 20261105T154105.565982203-74b8b25732e0796505863913 receives cancellation after the selected test starts.
Its state becomes cancelled. Its only attempt records unknown, failure kind cancelled, and unavailable test duration.
The run has no fabricated final test result. Its evidence gap remains visible.
Its single execution reaches cleaned. The original worker Pod disappears.

### Test timeout

Run 20261105T154216.058915039-b5fb0aabb067070b35b7320b executes a check with a two-second catalog timeout.
The attempt and final result both record timed-out with failure kind timeout.
The observed duration is 2008 ms. The run counts one timeout and zero failed tests.
Evidence is complete. Its single execution reaches cleaned and its worker Pod disappears.

This proves actual runner timeout handling. The independent Kubernetes deadline has separate evidence below.

### Rerun timed-out tests

Run 20261105T154833.201607664-fc55cd1fb3946ebffd6803ef uses Rerun failures on the recorded timeout run.
Its frozen request links the parent run and selects only K3S-TIMEOUT-001.
It retains one timed-out result, complete evidence, and one execution identity.
The corrected adapter completes cleanup on its first attempt without a recovery delay.

### Controller interruption

Run 20261105T154326.524393658-22d54af5c51f1f3b6a3c62b7 starts its selected test before the controller receives SIGKILL.
The verifier identifies the exact controller Pod UID, container identity, namespace, and live process before signaling it.
It does not delete the worker or change the run lease manually.
Kubernetes restarts the controller container. The same Pod UID has an increased restart count.
The verifier follows the original run and execution identifiers throughout recovery.

The worker delivers its passing report after the private API returns.
The recovered observation retains its measured 20000 ms duration and complete evidence.
A private go-events.jsonl download passes size and SHA-256 checks after the subsequent service upgrade.
The execution records recovered=true and reaches cleaned. The original worker Pod disappears.
The logical run remains incomplete with unknown outcome and a visible interruption notice.
Recovery does not invent a completed run or create another execution.
This proves restart recovery for one controller. A separate [peer failover case](local-kubernetes-controllers.md) now passes with two controllers.

### Independent Kubernetes deadline

Run 20261105T155703.162689682-275ad94081254ddd2cb5b4a4 verifies the Pod deadline independently of application timers.
The verifier stops guest processes with runsc kill STOP after the selected check starts.
The guest supervisor reports state T. The runtime remains running and available to Kubernetes.
The verifier then stops the exact controller process shortly before the original absolute deadline.
Kubernetes reports DeadlineExceeded and a terminated worker container with exit code 137.
The Pod UID and its original activeDeadlineSeconds=15 remain unchanged.
The controller still reports state T when worker termination is confirmed.

The container starts at 15:57:04 UTC and terminates at 15:57:29 UTC.
The observed interval includes the configured ten-second termination grace and kubelet scheduling delay.
This proves independent termination with grace. It does not promise exact-second process death.
The verifier restores the controller. The original run records one timeout with unavailable test duration and an explicit evidence gap.
Its original execution reaches cleaned and the worker Pod disappears. No duration is inferred from provider elapsed time.
Evidence is output/local-k3s/deadline/summary.json with original Pod state, guest state, run, attempts, events, logs, and execution records.

Two preliminary observations remain retained rather than rewritten.
The first exposes Python 3.9 timestamp parsing before any process pause.
The next pauses the whole runtime, which prevents CRI from signaling that paused container.
Both original runs reach their normal timeout and cleanup. The verifier restores all paused processes.
The corrected check stops guest processes while keeping runtime controls available.
It submits a new case only after the prior run is terminal and its concrete fixture error is corrected.

### Asynchronous deletion

The actual cluster acknowledges Pod deletion before every read reports absence.
An immediate absence assertion previously recorded cleanup-failed and waited for the next recovery interval.
The adapter now waits for confirmed absence within the existing cleanup call deadline.
Every intermediate read must still match the original Pod ownership and specification.
A different Pod UID stops cleanup without deletion or a false removal checkpoint.
Two focused checks cover delayed deletion and a foreign replacement after deletion acknowledgement.
They pass without a browser or cluster rebuild.
The rebuilt service deploys through Helm revision 8 using the retained private values and dedicated kubeconfig.
The UI and trace assets reuse their cached builds. The actual timeout rerun verifies cleanup on this service.

### Evidence and limits

Evidence lives in output/local-k3s/lifecycle.
The references file retains preparation, release, idempotency keys, run identifiers, actions, and interruption identity.
Per-case files retain runs, attempts, events, logs, artifacts, execution journals, and sanitized provider status.
Recovery provider history preserves observed identity and state changes without worker tickets or credentials.
The private registry and local PostgreSQL remain available for later cases.
No host browser or automation daemon starts. The existing user demo remains on port 8040.

Controller peer failover now passes through a separate recorded case and closes acceptance 52.
Cloud deployment profiles remain open. Do not infer node or storage fault tolerance from these local cases.

## Tutorials

### Inspect or resume a recorded case

```sh
python3 scripts/verify-kubernetes-lifecycle.py --case=cancel
python3 scripts/verify-kubernetes-lifecycle.py --case=timeout
python3 scripts/verify-kubernetes-lifecycle.py --case=recovery
python3 scripts/verify-kubernetes-lifecycle.py --case=rerun-timeout
```

The verifier reuses the saved preparation and run identifiers.
Use python3 scripts/verify-kubernetes-deadline.py to inspect its saved successful deadline case.
The deadline fixture requires the single combined-controller installation and rejects separate-controller Pods.
A recorded interruption action does not execute again during a later inspection.
An armed interruption without a recorded outcome stops for inspection before another signal.
Saved restart proof remains valid after a later deliberate deployment replaces the controller Pod.
It waits for confirmed cleanup and retains diagnostics before checking the expected result.
An observation timeout is not permission to create a replacement run.
Read the saved run, execution journal, and provider state before continuing.
All Kubernetes operations use the owned node and its explicit local kubeconfig.

### Continue implementation

Independent deadlines now have actual termination evidence. Reuse those records rather than repeating the pause experiment.
The optional [separate controllers](kubernetes-controllers.md) now run with shared evidence storage and pass peer failover.
The second provider and private report access now pass. See [hosted identity](local-kubernetes-identity.md).
Audit remaining deployment-specific gaps before adding more checks.
Keep Azure implementation and live cloud verification deferred.

Related guides: [source execution](local-kubernetes-source-execution.md), [worker contract](kubernetes-workers.md), and [implementation ledger](implementation.md).
