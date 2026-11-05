# ECS trusted gateway integration

## Introduction

ECS environment-backed browser and coded API execution remains disabled.
The existing offline ECS lifecycle stays separate.
The shared transport now supports a verified task address and private certificate environment delivery.
These components pass targeted local checks. They do not prove native ECS dispatch or deployment acceptance.
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
Old offline images remain valid for offline work. Kubernetes certificate-file execution retains its existing contract.

### Remaining native integration

Implement an owned gateway task beside each uploaded worker task.
Bind the observed gateway identity and private interface before credential delivery.
Keep uploaded source outside the gateway's filesystem, execution role, credentials, and network identity.
Supply the private gateway certificate only to that trusted task.
Confirm readiness, inspected images, network policy, and independent deadlines before activation.
The hosted task health command must use --ready-remote instead of the image's local socket health command.
Track both tasks, definitions, and deadline resources through the durable dispatch journal and cleanup checkpoints.
Preserve original identities across lost acknowledgements and recovery. Never invent a passing observation or duration.

The existing worker VPC has a restrictive DNS firewall and no general target egress.
Preserve those worker restrictions when implementing the gateway network.
Gateway target resolution and delivery require a trusted path that does not broaden uploaded worker DNS or target access.
The configured worker network needs a narrowly verified route to its gateway transport.
Keep environment profiles rejected until the native network, workload, identity, deadline, and recovery paths are complete.

## Tutorials

### Reuse local evidence

Retain the passing Kubernetes records under output/local-k3s/gateway and its lifecycle directory.
The new shared transport checks need no image build, browser, Kubernetes redeployment, or cloud account.
They verify pinned transport with normal certificate validation, immutable address binding, and private certificate parsing.
Continue native ECS implementation before building the changed gateway and supervisor images.
Use targeted local SDK and policy fixtures for the new ECS behavior. Keep live deployment verification separate.
