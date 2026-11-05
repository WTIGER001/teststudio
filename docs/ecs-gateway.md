# ECS trusted gateway integration

## Introduction

ECS environment-backed browser and coded API execution remains disabled.
The existing offline ECS lifecycle stays separate.
The shared transport supports a verified task address and private certificate environment delivery.
Native gateway task construction, ownership, deadline, recovery proof, and metadata bootstrap components now pass targeted local checks.
The gateway child provider now reuses the durable SDK lifecycle with a separate role journal and verified address binding.
These components do not yet provide the complete native ECS dispatch lifecycle or deployment acceptance.
Azure remains deferred, and this work makes no live cloud call.

## Concepts

### Task address and TLS identity

A Fargate gateway task has its own network interface.
The provider must inspect that task and interface before binding their address and immutable task identity.
Use the shared requireAddress setting to prepare initialization before the task address becomes known.
BindGatewayAddress freezes that address and instance together. Missing addresses prevent initialization and activation.
Retries cannot replace the bound address or create another workload after an uncertain launch.

The supervisor receives the private connection through gateway-connection after activation and input claim.
Its optional address chooses the TCP destination.
Its canonical HTTPS origin still chooses the port, certificate hostname, and TLS server name.
This avoids a dynamic DNS record for each task without weakening certificate verification.
The native provider must prove the address belongs to its observed task, configured account, subnet, network, and security group.
Loopback support belongs to local protocol fixtures and cannot satisfy native ECS network admission.
The uploaded child receives neither the connection nor its transport capability.

### Private server certificate

The gateway supports private PEM material in TESTSTUDIO_GATEWAY_TLS_CERTIFICATE and TESTSTUDIO_GATEWAY_TLS_KEY.
Both values must be present, nonempty, and within 64 KiB each.
The key must match the certificate. Errors contain no supplied material.
Mounted certificate files remain available when both variables are absent.

ECS can inject Secrets Manager values through container secret references using the task execution role.
Values enter at task startup. Rotation requires a new task.
See the official [ECS Secrets Manager environment guide](https://docs.aws.amazon.com/AmazonECS/latest/developerguide/secrets-envvar-secrets-manager.html).
The gateway execution role must remain separate from the uploaded worker execution role.
The worker must receive no permission or secret reference for the gateway key.
Task definitions and durable journals contain only approved secret references, never certificate key values.

### Capability admission

Address-capable source images declare teststudio.job-gateway-address=verified-ip/v1.
The runtime builder validates this marker, and registry inspection exposes it.
Gateway images with private certificate environment support declare teststudio.gateway-certificate=secret-env/v1.
The gateway registry inspector exposes this separate optional capability.
The native ECS gateway provider must require both capabilities before enabling environment execution.
It must also require teststudio.gateway-instance=ecs-task/v1 on the gateway image.
Registry inspection exposes this separate metadata identity capability.
Old offline images remain valid for offline work. Kubernetes certificate-file execution retains its existing contract.

### Implemented gateway task contract

NewGatewayPlan creates one fixed trusted gateway task for the environment execution reference.
It reuses the existing task definition, launch, ownership, deadline, and recovery-proof verifiers.
The gateway has its own deterministic family, ownership tags, task name, and schedule name.
It uses a separate execution role, subnet set, and security group.
The constructor rejects shared worker subnets, worker security groups, and worker or deadline execution roles.
Its fixed resources are 512 CPU units and 1024 MiB of memory.

The container runs /gateway as user 1000:1000 with a read-only filesystem and dropped capabilities.
It has no task role, uploaded source mount, output volume, interactive shell, restart policy, or execute-command access.
Its only declared port is TCP 8044. Its health command uses --ready-remote.
Unknown health keeps admission pending. Unhealthy status marks the infrastructure boundary failed.

Two Secrets Manager references supply the server certificate and key.
References must be distinct full secret ARNs from the configured account and region.
This first task contract accepts separate secrets, without JSON-key or version selectors.
The task definition contains no raw initialization ticket or application credential.
Launch overrides contain only the execution identifier and separate gateway initialization ticket.
The uploaded worker ticket cannot serve as that gateway ticket.

The gateway schedule retains the original execution deadline and the scoped StopTask target.
It uses the separate gateway resource name, preventing collision with the worker deadline.
The durable proof includes the frozen gateway configuration and ticket hash.
Restoring that proof can verify the original definition and schedule but cannot launch another task.
Changed certificate references, task identities, commands, resources, or launch authority fail ownership checks.

### Metadata bootstrap

The gateway supports --remote-ecs-exchange=HTTPS_ORIGIN.
It reads ECS_CONTAINER_METADATA_URI_V4 and requests only its /task endpoint.
Fargate injects that metadata variable. See the official [Fargate metadata guide](https://docs.aws.amazon.com/AmazonECS/latest/developerguide/task-metadata-endpoint-v4-fargate.html).
The helper accepts only the literal link-local metadata host and canonical version-four container path.
It permits no credentials, query, alternate host, encoded path, port override, or redirect.
The read has a three-second limit and a 64 KiB response limit.
No initialization ticket or certificate value enters that request.
Additional metadata fields remain compatible with the platform contract.

The helper extracts the task identifier from the canonical task ARN.
It uses that identifier for the existing private initialization request.
An explicitly supplied gateway instance variable is rejected in this mode.
The native controller must independently verify the complete observed task ARN, cluster, account, region, and network before binding the same identifier.
Task metadata alone does not prove controller-side ownership.

### Durable gateway child lifecycle

NewGateway uses the existing task registration, launch, observation, shutdown, and guarded removal state machine.
Its ecs-gateway-dispatches journal remains separate from the uploaded worker journal.
Both roles retain the original execution reference and private exchange authority.
The frozen gateway configuration includes the public certificate authority. It contains no certificate key or raw initialization ticket.
Configuration changes cannot acquire an existing gateway row.

Before native creation, image inspection requires the remote transport, private certificate, and ECS task identity capabilities.
The configured network guard must admit the separate gateway network.
Observation verifies the task definition, launch authority, independent deadline, native task network, and reported private IPv4 address.
Only then does it bind the immutable task identifier and address to the private exchange.
Unknown health still allows this binding so the task can fetch initialization. It keeps worker admission pending.
Withdrawn run authority prevents binding. Replacing an existing address fails ownership verification.

Lost registration, launch, and deadline acknowledgements recover through the original SDK receipts without another creation attempt.
Gateway removal requires ticket revocation, confirmed termination, saved results, and complete or abandoned evidence.
Local checks use the real private exchange with authored AWS protocol fixtures.
They cover initialization order, unsupported capabilities, address mismatch, authority withdrawal, restart recovery, and checkpointed cleanup.
Selected existing offline worker lifecycle checks still pass.

This child provider has no normal executor registration. It cannot enable environment execution by itself.
The concrete native gateway network guard and parent two-task orchestration remain required.

### Remaining native integration

Wire the implemented task contract into an owned gateway lifecycle beside each uploaded worker task.
Bind the observed gateway identity and private interface before credential delivery.
Keep uploaded source outside the gateway's filesystem, execution role, credentials, and network identity.
Supply the private gateway certificate only to that trusted task.
Confirm readiness, inspected images, network policy, and independent deadlines before activation.
The hosted task health command must use --ready-remote instead of the image's local socket health command.
Track both tasks, definitions, and deadline resources through the durable dispatch journal and cleanup checkpoints.
Preserve original identities across lost acknowledgements and recovery. Never invent a passing observation or duration.
Use the implemented gateway child provider beside the worker lifecycle.
Gateway and worker rows share execution authority while retaining distinct task, definition, and schedule identities.
Stop both workloads even when one observation fails. Cleanup requires the existing result and evidence checkpoint.

Create the worker only after the gateway's identity, initialization, readiness, network, and deadline are verified.
The current supervisor retries initial input for a short bounded period.
Do not let a cold gateway exhaust that period before the worker can receive activated input.
Continue only unattempted dispatch steps under current run authority. Recovery must never launch uploaded code again.

The existing worker VPC has a restrictive DNS firewall and no general target egress.
Preserve those worker restrictions when implementing the gateway network.
Gateway target resolution and delivery require a trusted path that does not broaden uploaded worker DNS or target access.
The configured worker network needs a narrowly verified route to its gateway transport.
Keep environment profiles rejected until the native network, workload, identity, deadline, and recovery paths are complete.

## Tutorials

### Reuse local evidence

Retain the passing Kubernetes records under output/local-k3s/gateway and its lifecycle directory.
The new transport, task-contract, and metadata checks need no image build, browser, Kubernetes redeployment, or cloud account.
They verify pinned transport with normal certificate validation, immutable address binding, and private certificate parsing.
Continue native ECS implementation before building the changed gateway and supervisor images.
Use targeted local SDK and policy fixtures for the new ECS behavior. Keep live deployment verification separate.
