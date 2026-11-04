# ECS worker network admission

## Implementation boundary

The ECS adapter now includes SDK-backed network admission and bounded private settings parsing.
Its typed clients read EC2, ELBv2, Route 53 Resolver, and Route 53 Profiles state.
They perform no network mutation. Local verification supplies authored protocol fixtures instead of live clients.

The API and preparation commands do not enable this adapter yet.
SDK construction, ECR authentication, command wiring, and worker infrastructure remain required.
The current AWS service profile continues to execute native HTTP checks only.
See [worker lifecycle](ecs-worker-lifecycle.md) for dispatch and checkpoint recovery.

## Separate worker network

Workers require a dedicated IPv4 VPC, separate from the controller VPC.
DNS Firewall applies at VPC scope. Applying this allowlist to controllers would block ordinary identity, notification, and target traffic.
The worker VPC has no internet gateway route, NAT route, peering route, or IPv6 path.
Effective worker subnet route tables permit only local VPC routes and the regional S3 gateway endpoint.
Worker subnets disable public addressing, automatic IPv6 assignment, and DNS64.
VPC DNS support and hostnames remain enabled. DHCP uses AmazonProvidedDNS only.

The worker security group has no ingress. Its egress permits only TCP 443 to:

- The private exchange endpoint security group.
- The ECR endpoint security group.
- The AWS-managed regional S3 prefix list.

Both interface endpoint groups accept TCP 443 only from the worker group and have no egress rules.
Remove default allow-all egress before admission. CloudFormation adds that default when initial group creation supplies no egress rules.
Infrastructure packaging must account for this behavior. An empty configuration array alone does not establish the required state.

ECR API and Docker endpoints use private DNS, IPv4, the expected security group, and every configured worker subnet.
Their policies permit the configured platform execution role to pull only the configured repository prefix and its descendants.
The separate authorization statement permits that same role to request registry authorization.
The S3 gateway policy permits GetObject only from the regional AWS-owned ECR layer bucket.
It grants no write, listing, or arbitrary bucket access.
Policy comparison rejects duplicate JSON keys and unexpected statements or permissions.
Equivalent policy documents must retain the specified statement, action, and resource ordering.

## Private exchange path

The exchange hostname resolves through an accepted IPv4 PrivateLink endpoint in the worker VPC.
The endpoint service requires connection acceptance, verified private DNS, and the configured account's root principal as its sole allowed principal.
It references exactly one internal Network Load Balancer in the controller VPC.
The load balancer has one TLS 443 listener, the fixed TLS policy, and a certificate from the configured account and region.
Its sole action forwards to the configured TCP 8042 IP target group.

The load balancer group has no ingress and permits egress only to the controller group on TCP 8042.
PrivateLink inbound group enforcement is disabled. Direct connections still have no permitted ingress.
The controller group permits worker-port ingress only from that load balancer group.
Other controller API rules remain independent.
The target group's HTTP health check uses its traffic port, /healthz, and status 404.
The private listener intentionally exposes no health route. A healthy 404 confirms that isolated listener accepts HTTP.
This check does not replace controller database readiness or exchange authorization.

Admission verifies one through 32 healthy backend targets and rejects duplicate targets.
Each target must match a private interface in the configured controller VPC, owned by the configured account, with only the controller group.
Neither the public API listener nor the artifact listener becomes a worker endpoint.

## DNS policy

The VPC has exactly one completed, mutation-protected firewall association and no Route 53 Profile association.
Profiles can introduce inherited firewall settings. This initial worker profile rejects all such overlays.
The firewall group and both domain lists must belong to the configured account and region.
Firewall failure mode is disabled, so a firewall error blocks resolution.

The first rule allows these exact names at priority 100:

- The private exchange hostname.
- ACCOUNT.dkr.ecr.REGION.amazonaws.com.
- api.ecr.REGION.amazonaws.com.
- prod-REGION-starport-layer-bucket.s3.REGION.amazonaws.com.
- prod-REGION-starport-layer-bucket.s3.amazonaws.com.

The second rule blocks all other names with NXDOMAIN at priority 200.
Neither rule has query-type exceptions, advanced-rule alternatives, or override domains.
Both use TRUST_REDIRECTION_DOMAIN. Exact initial queries remain restricted while trusted private and AWS aliases can resolve.
No arbitrary original hostname receives that exception.

## Actual task admission

Pool validation does not establish task isolation by itself.
Before the exchange releases executable input, the provider verifies the task's attached network interface against EC2 state.
It requires the configured account, worker VPC, selected subnet, matching private IPv4 address, and sole worker security group.
It rejects public addresses, extra groups, IPv6, delegated prefixes, alternate subnet associations, and disabled source checks.
The task container's attachment and address must agree with that same interface.
An interface still awaiting attachment remains retryable. Changed ownership blocks admission.

Successful pool snapshots cache for two seconds and serialize concurrent refreshes.
A pool refresh and a complete task check each have a five-second context bound. SDK calls disable automatic retries.
Incomplete reads and unrecognized errors produce bounded unavailable responses without provider diagnostics.
Later checks refresh expired snapshots. This polling detects drift; it does not make separate AWS reads atomic.
The lifecycle revokes the exchange and stops the task when admission is lost.

## Settings and timing identity

Settings contain worker and network objects. Each resource reference is explicit and installation-specific.
LoadSettings accepts one bounded JSON file or one named environment reference through the shared runtime loader.
It rejects unknown fields, multiple sources, unsupported origins, mismatched account or region references, and shared controller and worker VPCs.
The exchange must use an exact DNS hostname and HTTPS port 443.
The loader binds a digest of the complete network settings into the worker configuration.
That digest participates in the execution profile, resource ownership, recovery proof, and timing-pool identity.
Changing the isolation profile cannot silently reuse the previous pool identity.
No provider credentials, registry tokens, or worker tickets belong in this settings document.

## Controller read permissions

SDK wiring requires these read actions in the selected region and account.
These permissions belong to trusted controllers, never to the test container.

| Service | Actions |
| --- | --- |
| EC2 | DescribeVpcs, DescribeVpcAttribute, DescribeDhcpOptions, DescribeSubnets, DescribeSecurityGroups, DescribeRouteTables, DescribeVpcEndpoints, DescribePrefixLists, DescribeNetworkInterfaces, DescribeVpcEndpointServiceConfigurations, DescribeVpcEndpointServicePermissions. |
| ELBv2 | DescribeLoadBalancers, DescribeListeners, DescribeTargetGroups, DescribeTargetHealth. |
| Route 53 Resolver | GetFirewallConfig, ListFirewallRuleGroupAssociations, GetFirewallRuleGroup, ListFirewallRules, GetFirewallDomainList, ListFirewallDomains. |
| Route 53 Profiles | ListProfileAssociations. |

Missing read authority blocks dispatch. Infrastructure packaging must retain these permissions during cleanup and controller replacement.

## Local verification

Run the focused package checks from the repository root:

```sh
go test -race ./adapters/execution/ecs -count=1
```

Evidence is output/ecs-network-admission/final-contract.log.
Checks cover valid pool and task admission, concurrent cache reuse, expired snapshots, task drift, private errors, and invalid settings.
Mutation cases reject internet routes, wider egress, changed endpoint policies, public exchange paths, DNS overlays, and firewall fail-open state.
No AWS credentials, accounts, deployments, or live registry endpoints are used.
The running demo, UI, and source runtimes remain unchanged.

## Primary service references

[ECR VPC endpoints](https://docs.aws.amazon.com/AmazonECR/latest/userguide/vpc-endpoints.html) defines the API, Docker, and layer-bucket paths.
[Security group rules](https://docs.aws.amazon.com/vpc/latest/userguide/security-group-rules.html) explains the resolver exception to security group filtering.
[Network Load Balancer groups](https://docs.aws.amazon.com/elasticloadbalancing/latest/network/load-balancer-security-groups.html) describes target rules and PrivateLink enforcement.
[CloudFormation security groups](https://docs.aws.amazon.com/AWSCloudFormation/latest/TemplateReference/aws-resource-ec2-securitygroup.html) describes default outbound rules.
[DNS firewall rules](https://docs.aws.amazon.com/Route53/latest/APIReference/API_route53resolver_FirewallRule.html) defines redirection behavior.
[Route 53 Profiles](https://docs.aws.amazon.com/Route53/latest/DeveloperGuide/profiles.html) describes inherited VPC settings.
