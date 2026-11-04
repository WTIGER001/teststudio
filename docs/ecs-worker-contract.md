# ECS worker task and deadline contracts

## Current scope

The ECS worker package builds fixed AWS SDK task definitions, launch requests, deadline schedules, and recovery proofs.
It supports the existing offline Go and Playwright execution and discovery operations.
Pinned dependencies are ECS v1.103.1 and Scheduler v1.26.1. They retain the existing AWS core SDK version.

Optional [API and preparation flags](ecs-workers.md) now connect these contracts to the existing durable lifecycle.
The [durable provider lifecycle](ecs-worker-lifecycle.md) now implements dispatch, observations, and checkpointed cleanup with required network-admission hooks.
The [network checker and bounded settings parsing](ecs-network-admission.md) are implemented.
Scoped SDK initialization and command wiring pass local checks. The [staged AWS worker package](aws-workers-package.md) now supplies private infrastructure.
Deployment-wide execution and recovery acceptance remain open.
The default core service profile supports native HTTP execution. Coded workers require the optional staged package.
No cloud account, registry, or deployment is contacted by these contract checks.

## Explicit installation scope

The configuration identifies one AWS region, account, cluster ARN, registry-pull execution role, deadline role, and Scheduler group.
It also selects an explicit family prefix, architecture, platform version, subnet set, worker security group, and HTTPS exchange origin.
Roles and cluster references must belong to the configured account. Cluster references also match the selected region.
Execution and deadline roles must differ. Neither role becomes the test container's task role.
This initial contract uses the standard aws ARN partition and the pinned Fargate Linux platform 1.4.0.

The planner accepts one through sixteen unique subnet identifiers and exactly one worker security group.
Subnet ordering is canonical. Reordering equivalent subnets does not split the timing pool.
The execution profile binds the selected pool, architecture, platform, and fixed resource class.
The provider must still inspect actual subnet and security-group isolation before enabling dispatch.
Configuration validation alone proves no VPC routing, registry endpoint policy, or private ingress property.

## Immutable task definition

The task definition has one essential runner container and one immutable distribution image reference.
Its entrypoint is /runner/teststudio-job. Arguments select only the Go or Playwright runner and the configured private exchange.
Discovery adds the fixed --operation=discover argument and requires the preparation reference namespace.
Execution and discovery cannot reuse each other's profile or identity.

The task uses Fargate, awsvpc networking, Linux, and the configured ARM64 or x86_64 architecture.
Its fixed resource class is 2 vCPU and 4 GiB memory.
The container runs as 1000:1000 with a read-only root filesystem, dropped capabilities, and an init process.
Privileged mode, interactive input, terminal allocation, and container restart are disabled.
The container stop timeout is ten seconds.

The definition mounts two task-scoped volumes at /tmp and /output. It specifies no host path or persistent storage.
The explicit ephemeral allocation is 21 GiB, including image storage and both writable volumes.
This is the smallest explicit allocation supported by the ECS API. It is not a per-volume quota.
Fargate does not support tmpfs. The previous task contract incorrectly supplied tmpfs and an explicit 20 GiB allocation.
The corrected contract retains the 2 vCPU and 4 GiB memory class and changes the execution-pool identity to fargate-v2.

Supervised runtime overlays declare exactly /tmp and /output as image volumes, owned by 1000:1000.
Their directory modes are 1777 and 0700. The image declares teststudio.task-storage=task-volumes/v1.
Registry inspection checks that capability and both volume declarations. ECS rejects images without it before execution admission.
Source assembly preserves the capability and volume declarations in the prepared image.
Rebuild older supervised overlays and prepare their source again before selecting ECS.
Existing Kubernetes images without this optional capability retain their prior admission behavior.

The task has one container and shares no volume with another task or controller.
Fargate removes temporary data when the task stops. Result and evidence upload still precede normal resource cleanup.
These volumes do not provide the earlier proposed per-mount size or noexec controls.
The finite task allocation, memory limit, artifact limits, immutable runtime, private networking, and independent deadline remain separate controls.
Disk exhaustion can make an attempt incomplete. It cannot establish a passing test or complete evidence.

Definitions carry no worker ticket, AWS task role, secrets, environment file, sidecar, log driver, or arbitrary command override.
The platform execution role remains separate from container credentials.
AWS documents unsupported parameters in [Fargate differences](https://docs.aws.amazon.com/AmazonECS/latest/developerguide/fargate-tasks-services.html).
[EphemeralStorage](https://docs.aws.amazon.com/AmazonECS/latest/APIReference/API_EphemeralStorage.html) defines the supported explicit size range.
[Bind mounts](https://docs.aws.amazon.com/AmazonECS/latest/developerguide/bind-mounts.html) describes task lifetime and image-derived directory ownership.

Ownership tags bind the complete execution reference hash, installation, project, configuration scope, operation, deadline, and specification hash.
Definition verification checks the exact family, account, region, ARN revision, active status, ownership tags, and immutable settings.
Service-generated registration metadata is excluded from content comparison. Absent and empty default values normalize consistently.
Unexpected identity, credentials, commands, containers, capabilities, restart policy, or resource changes fail ownership verification.
These content hashes detect changed inputs. They do not replace current run or source authorization.

## Launch request

The launch request selects the exact returned task-definition ARN, one task, the fixed platform, configured subnets, and worker security group.
It disables public addresses, ECS Exec, and managed tags. It does not select Spot or an implicit capacity strategy.
Its only container override supplies the scoped worker execution identifier and ticket.
The supervisor creates its own instance identifier. No application session, registry credential, or provider credential enters that environment.

The launch client token derives from the complete frozen plan, including a ticket hash.
The definition family and startedBy value derive from ownership. The same frozen plan retains its identity across exact retries.
Plan formatting excludes the raw ticket. Raw SDK launch requests still contain that ticket and must never enter public diagnostics.

RunTask idempotency has a finite retention window. It cannot replace durable creation reservations or cleanup-only recovery.
AWS documents the bound as the smaller of 24 hours and the resource lifetime plus one hour.
See [ECS idempotency](https://docs.aws.amazon.com/AmazonECS/latest/developerguide/ECS_Idempotency.html).
The future provider must not retry an uncertain launch after that window or recreate a previously stopped task.

## Independent deadline

Each launched task requires a separate one-time Scheduler stop action before controller activation releases worker input.
The schedule targets ecs:stopTask through the universal AWS SDK target and names the exact task and cluster ARNs.
The planner restricts task ARNs to the configured account, region, and cluster.
The provider must bind the selected ARN to a verified creation receipt before requesting its schedule.
Its JSON input follows the ECS StopTask field names, including lowercase cluster, task, and reason.
The deadline role is explicit and separate from the image-pull execution role.
See [universal targets](https://docs.aws.amazon.com/scheduler/latest/UserGuide/managing-targets-universal.html) and [StopTask](https://docs.aws.amazon.com/AmazonECS/latest/APIReference/API_StopTask.html).

The schedule uses UTC, rounds fractional deadlines upward to the next second, and disables the flexible window.
It stays enabled and retains its resource after invocation so recovery can inspect it.
Retries have a sixty-second event age and at most two retry attempts.
Deadline verification binds the complete schedule identity, role, exact stop target, expression, state, and retry settings.

Scheduler has sixty-second invocation precision, even without a flexible window.
The backup stop therefore provides a platform cleanup mechanism, not subsecond test-timeout precision.
The runner and worker exchange retain their existing exact test and ticket deadlines.
See [Scheduler precision](https://docs.aws.amazon.com/scheduler/latest/UserGuide/schedule-types.html).
Do not close deployment timing acceptance until this complete behavior is verified on the target profile.

## Recovery proof

The serializable proof retains the execution reference, image, runner, operation, deadline, ticket hash, and plan fingerprint.
It contains no raw ticket. It reconstructs definition and deadline verification without restoring launch authority.
A restored plan cannot generate a launch request.
Changing the claimant, execution pool, configuration, or retained content invalidates that proof.
The provider must obtain proofs from authorized durable records and match them to the current ledger or source checkpoint.

The next delivery must persist the exact task and task-definition ARNs before stop or cleanup.
StopTask removes task tags, so later ownership checks cannot require their continued presence.
Saved provider receipts, the exact task-definition identity, and verified terminal observations must survive that transition.
Missing tasks without a saved terminal observation remain unresolved. Resource absence alone must not become successful cleanup.

## Targeted local evidence and continuation

output/ecs-worker-contract/contract.log records focused race checks for both runners and both operations.
Cases cover immutable drafts, mutation rejection, independent stop targets, proof restoration, private ticket exclusion, and timing-pool identity.
A new local TLS fixture receives the production SDK's signed StopTask request and compares its body with the deadline target input.
Credentials come only from that owned fixture's explicit callback. No AWS credential discovery or external endpoint is used.
The fixture shuts down after the check. No browser, service image, UI, or source-runtime build is required.
The API, preparation command, and supervisor retain successful compilation checks in compile.log.

The durable ECS state machine now reserves registration and launch, recovers lost acknowledgements, and retains terminal observations and cleanup state.
The shared driver waits for explicit provider admission before opening the exchange gate.
Supported task-local storage and non-root overlay volumes now pass focused local checks.
Evidence is output/ecs-network-admission/task-storage-contract.log and output/ecs-task-storage.
The package probe verifies both runner overlays with real local volume initialization, writable scratch paths, and rejected runtime writes.
It starts no browser or source suite. All owned containers and anonymous volumes exit through removal.
Scoped SDK initialization, ECR authentication, and command wiring now pass local checks.
Private HTTPS routing infrastructure, controller roles, and separate preparation now have a locally checked CloudFormation package.
Next verify rollout admission behavior locally, then complete target-profile verification in its separate authorized session.
Keep the AWS provider disabled until those lifecycle and routing contracts are complete.
