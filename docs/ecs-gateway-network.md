# ECS gateway network controls

## Introduction

The native ECS network guard now supports a separate trusted gateway VPC.
Targeted local AWS protocol checks pass. This does not prove a deployed AWS network.
Production executor selection and deployment resources remain unfinished.
The current SDK factory rejects gateway network settings before it makes an AWS request.
Azure remains deferred. Retain the working local Kubernetes installation and its existing evidence.

## Concepts

### Separate network identities

Use three distinct VPCs for the environment execution profile: the controller VPC, worker VPC, and gateway VPC.
Use canonical private IPv4 ranges for the worker and gateway networks.
The gateway VPC has its own private task subnets, public NAT subnets, and security groups.
The worker VPC retains its restrictive DNS firewall and private exchange and registry endpoints.
Its additional routes reach only the declared private gateway subnet ranges through one owned peering connection.
Worker egress adds only TCP 8044 to the gateway security group. Worker inbound access remains absent.

The peer uses the same account and Region. Both VPC identities and the active connection must match.
AWS supports security group references across a same-Region peer.
See [peer security groups](https://docs.aws.amazon.com/vpc/latest/peering/vpc-peering-security-groups.html).
The guard validates peer identifiers and any returned peer metadata in these rules.
An offline pool has no gateway settings and still rejects peer routes or gateway egress.
Provision the environment pool separately. Do not add these routes to the retained offline installation.

### Trusted gateway traffic

Only the worker security group can initiate gateway transport on TCP 8044.
The trusted gateway permits outbound IPv4 TCP on ports 1 through 65535 for reviewed application targets and platform HTTPS dependencies.
The existing gateway engine still enforces the frozen origin, request rules, destination policy, credential boundary, and execution capability.
This outbound rule applies only to the trusted gateway. Uploaded code retains no direct application path.

Gateway task subnets have no public addresses or direct internet gateway route.
Their default routes use only the declared available public NAT gateways.
Return routes reach only the declared worker subnet ranges through the same peer.
The guard verifies NAT subnet ownership and the exact public subnet routes and owned internet gateway attachment.
EC2 reports the internet gateway attachment as available, even though the SDK enum does not declare that value.
See the [EC2 response contract](https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_DescribeInternetGateways.html).

The gateway uses a separate interface endpoint for the original private exchange service.
That endpoint belongs to the gateway VPC and uses its private task subnets and dedicated endpoint security group.
It must be available and have private DNS enabled.
The shared endpoint service, private hostname, TLS listener, and controller target checks still apply.
Private DNS resolves the service name within each consumer VPC.
See [PrivateLink DNS names](https://docs.aws.amazon.com/vpc/latest/privatelink/manage-dns-names.html).

### Frozen configuration and task addresses

NetworkSettings accepts an optional gateway object with these fields:

| Field | Required value |
| --- | --- |
| vpc | A separate owned gateway VPC identifier. |
| subnets | One through sixteen private gateway task subnet identifiers. |
| securityGroup | The gateway transport and target-egress group. |
| peeringConnection | The owned active worker-to-gateway peer. |
| exchangeEndpoint | The gateway VPC's interface endpoint for the original private exchange service. |
| exchangeGroup | The dedicated endpoint security group. |
| natGateways | The public NAT gateways used by the private task routes. |

The task GatewayConfig must select the same gateway subnets and security group.
Configuration rejects reused groups, endpoint identifiers, subnets, or VPC identities.
Gateway subnet and NAT order is canonical. The complete network settings enter the immutable execution profile.
Changing or removing the gateway configuration cannot retain the previous network identity.
The provider copies these settings before use.

The shared native interface verifier checks the ECS attachment against EC2's actual interface.
It requires the configured account, VPC, subnet, security group, IPv4 address, source check, and running interface state.
Public addresses, IPv6, extra groups, delegated prefixes, mismatched addresses, and another attachment fail.
Only this verified address can enter the private gateway initialization binding.
Pool observations retain the existing two-second cache. Task interface checks remain separate on each observation.

## Tutorials

### Continue installation implementation

Connect the environment provider and this guard through explicit SDK settings and native factory selection.
Add dedicated environment worker and gateway network resources without changing the existing offline pool.
Package peering, narrow routes, security groups, NAT, the private exchange endpoint, and certificate delivery together.
Use separate pull roles and retain the reviewed runtime and gateway image capabilities.
Wire source preparation and discovery to their intended profiles. Discovery receives no application target or credential.
The worker package now includes the three additional read operations for peer, NAT, and internet gateway observations.
This permission addition does not create or enable an environment network.

### Reuse targeted local verification

The authored protocol fixture exercises one complete accepted network and the owned task interface.
Changed routes, direct worker egress, foreign or inactive peers, gateway ingress, NAT, exchange DNS, and worker DNS policy fail admission.
Native errors remain sanitized. Expired pool observations detect drift.
An incomplete production selection fails before an AWS request.
The changed checks and selected existing offline network checks pass with the race detector.
A fresh local preview in output/ecs-gateway-network/admission-fixture.json matches the current checked-in worker template.
Its existing package-to-admission check passes. The preview also confirms the three new read permissions retain the configured Region condition.
No image build, browser, Kubernetes redeployment, live cloud call, or catalog test run occurs.
Keep complete deployment acceptance open and use separately authorized AWS verification later.
