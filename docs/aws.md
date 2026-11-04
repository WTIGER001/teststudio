# AWS core service profile

The template packages TestStudio's core service on ECS Fargate with CloudFront.
It creates independent resources. It imports no MarketMaker configuration.
Use the authorized MMDev account for later AWS verification.
Current verification uses local schemas and package checks only. No AWS deployment occurs in this session.

The profile currently supports the catalog, identity, native HTTP execution, schedules, and private evidence service.
Hosted Go and Playwright execution, preparation workers, and hosted recording remain unfinished.
The [ECS task and deadline contracts](ecs-worker-contract.md) now provide immutable SDK drafts and recovery proofs for the worker adapter.
The [durable provider lifecycle](ecs-worker-lifecycle.md) now adds reserved dispatch, recovery, and checkpointed cleanup.
These components do not enable hosted coded execution. Network admission, settings parsing, and supported task-local storage now pass local checks.
Scoped SDK setup, command wiring, private routing infrastructure, and worker packaging remain required.
Do not enable local Docker runner flags inside this service.
Keep acceptance 57 and the related hosted acceptance items open.

## Required operator inputs

Use a dedicated stack name with at most 40 characters.
Provision the following installation resources before deployment:

| Input | Requirement |
| --- | --- |
| VPC | One installation VPC with DNS support. |
| PublicSubnets | At least two public subnets in separate availability zones for the load balancer. |
| PrivateSubnets | Private task subnets with database reachability and outbound access. Tasks receive no public addresses. |
| HostedZoneId | A public Route53 zone that owns all three installation hostnames. |
| UiHostname | The HTTPS UI and API host. |
| ArtifactHostname | A separate HTTPS host for authorized evidence and the private trace viewer. |
| OriginHostname | A third HTTPS host for CloudFront's load balancer connection. |
| ViewerCertificateArn | An issued certificate in us-east-1 covering the UI and artifact hosts. |
| OriginCertificateArn | An issued certificate in the stack region covering the origin host. |
| CloudFrontPrefixListId | The regional AWS-managed CloudFront origin-facing prefix list identifier. |
| ServiceRepositoryName | The installation's ECR repository in the stack account and region. Nested repository names are supported. |
| ServiceImageDigest | The pushed registry distribution digest, including sha256:. |
| Architecture | ARM64 or X86_64, matching the image. ARM64 is the current locally verified package. |
| DatabaseSecretArn | A Secrets Manager secret containing the complete PostgreSQL connection URL as its plain string value. |
| OidcConfigurationSecretArn | A Secrets Manager secret containing the initial provider JSON array as its plain string value. |
| BootstrapIssuer | One configured provider's exact issuer. |
| BootstrapSubject | The initial administrator's exact external subject. |

The template creates an ECS cluster, service, roles, log group, load balancer, security groups, distributions, buckets, and DNS records.
It creates no database, VPC, subnet, certificate, ECR repository, external identity provider, or upstream application secret.
Enable TLS verification for PostgreSQL. Use the database's supported root certificate configuration.
Grant the output TaskSecurityGroupId access to the database's TLS port through the database security group.
The API needs that access before ECS can become healthy.

Private tasks need access to ECR, its layer storage, Secrets Manager, CloudWatch Logs, S3, PostgreSQL, identity providers, and registered HTTP targets.
Provide NAT or suitable VPC endpoints and routing for these destinations.
A service security group permits outbound IPv4 traffic. This is core service connectivity, not an untrusted worker sandbox.
The CloudFront prefix list consumes security group quota. Confirm the VPC's available quota before deploying.
Certificate ownership, zone ownership, subnet routing, real identity, and network reachability require deployment verification.

## Identity and secrets

The configuration secret contains provider settings, not resolved client credentials.
Use one or more providers with unique permanent identifiers. The server accepts up to 100 initial providers.
Use this structure with installation-specific values:

```json
[
  {
    "id": "primary",
    "name": "Organization sign-in",
    "issuer": "https://identity.example.test",
    "clientId": "teststudio",
    "secretRef": "env:TESTSTUDIO_OIDC_CLIENT_ONE",
    "scopes": ["openid", "profile", "email"],
    "disabled": false,
    "autoProvision": true
  }
]
```

Register https://UI_HOST/auth/callback/PROVIDER_ID with each external provider.
Confirm the exact callback displayed in TestStudio before enabling that provider.
Do not infer a subject from an email address or a display name.
Initial provisioning does not grant new users project authority. Project owners add authorized users directly.

OidcClientOneSecretArn injects TESTSTUDIO_OIDC_CLIENT_ONE. OidcClientTwoSecretArn injects TESTSTUDIO_OIDC_CLIENT_TWO.
Each optional secret holds one plain client credential string. Public clients can omit these inputs and their secretRef fields.
Multiple providers can use the two mapped credential references where the identity configuration permits it.
Additional independent client credentials require extending the explicit task secret and approved-reference mappings.
LocalAccounts optionally enables managed password accounts alongside OIDC. It does not enable the loopback administrator shortcut.

The task execution role reads the selected startup secrets. The runtime task role cannot retrieve them through Secrets Manager.
Use SecretKmsKeyArn when all selected secrets use one customer-managed encryption key.
Its key policy must permit the execution role's decryption access.
Different customer-managed secret keys require extending the explicit key permissions.
Secret references and configuration cannot contain inline AWS credentials.

Secret values enter the protected task environment. They stay out of command arguments and template outputs.
Authorized infrastructure operators can inspect container environments and resolved origin settings. Restrict those administrative permissions.
Replace service tasks after rotating database or OIDC secrets. Existing tasks retain their injected values.
Initial provider records are inserted only when absent. Use TestStudio's provider administration to change existing provider settings.
Bootstrap grants apply once. Later restarts do not restore revoked administrator access.

## Routing and evidence

The UI distribution reads the private UI bucket through Origin Access Control.
Only that distribution receives bucket read permission. The bucket has no website endpoint or public access.
The UI-only function rewrites extensionless application routes to index.html.
API, authentication, health, and liveness paths route to ECS without that rewrite.
There is no global error-to-HTML fallback.
Both distributions set error caching durations to zero without replacing error status codes or response bodies.

API and authentication behaviors disable caching and forward cookies, query strings, and viewer headers except Host.
CloudFront supplies the origin host for TLS and load balancer routing.
The UI response policy retains the application's content restrictions and permits the separate artifact host.
It does not override private evidence or trace-viewer response policies.

The artifact distribution forwards every request to the artifact listener on port 8041.
It disables response caching and forwards access-grant queries to TestStudio.
Evidence access still requires the application's current session, permissions, grant validity, and retained artifact manifest.
CloudFront receives no direct read permission for the evidence bucket.
The task role receives object read, write, and delete access only under the new bucket's teststudio prefix.
Bucket listing applies only to this dedicated evidence bucket.
It lets missing objects return S3's missing-object status instead of an ambiguous access denial.

Both distributions use HTTPS to the load balancer.
The origin security group accepts HTTPS only from the selected CloudFront prefix list.
A generated secret header also gates both load balancer routes. Unmatched requests receive 403.
Treat the resolved header value as a credential. Coordinate its replacement in the distributions and listener rules.
Do not rotate that secret independently of the referencing infrastructure.

ECS runs the service as 65532:65532 with no Linux capabilities and a read-only root filesystem.
The image declares /tmp as a volume with non-root ownership. ECS mounts a task-scoped temporary volume at that exact path.
PostgreSQL and S3 retain product state. No task-local SQLite or persistent artifact directory is selected.
The task health command checks /livez. Load balancer readiness checks /healthz on port 8040 for both target groups.
The service uses a deployment circuit breaker and a 90-second health grace period.
DesiredCount defaults to one. Replicas use shared PostgreSQL and S3, but deployment failover acceptance remains open.

The buckets and log group have retained deletion policies.
The template installs no bucket expiration rule. Application retention and pins control evidence deletion.
Logs retain 30 days. CloudFront and load balancer access logs are not enabled by this initial package.
Protect access-grant URLs from external access logs when adding logging infrastructure.
Back up PostgreSQL and follow the installation's evidence backup policy before destructive infrastructure changes.

## Prepare a later deployment

Build the required architecture with the [service image producer](service-image.md).
Push it to the dedicated ECR repository only when that action is authorized.
Use the registry distribution digest, not the local Docker image identifier.
Publish web/dist/web/browser into the output UiBucketName after the stack exists.
Upload the same built UI version that accompanies the service image.
Refresh the UI distribution after publishing a replacement. Its default cache lifetime is 60 seconds.

Use deploy/aws/service.yaml with a reviewed CloudFormation parameter file and CAPABILITY_IAM.
Do not put secret values in that file. Supply only the resource references described above.
Review a change set before executing it in the later deployment session.
After deployment, verify DNS, TLS, identity callbacks, database readiness, origin restrictions, and private evidence access.
Then complete the common acceptance scenarios and the missing hosted worker contracts.
Do not treat successful infrastructure creation as full product acceptance.

## Local packaging checks

Install the pinned local checker in a private environment:

```sh
python3 -m venv .data/aws-tools/venv
.data/aws-tools/venv/bin/pip install -r scripts/aws-check-requirements.txt
.data/aws-tools/venv/bin/python scripts/verify-aws-package.py --output output/aws-package/check
```

The command runs cfn-lint schemas for two regions and checks routing, caching, references, permissions, volumes, and health contracts.
It evaluates the UI routing function locally with Node. It performs no AWS service call or deployment.
Evidence is output/aws-package/check/summary.json with schema and routing logs.
The service image passes 12 focused local checks in output/aws-package/service/summary.json.
All owned verification containers and volumes are removed. Existing demonstration data and browser sessions remain intact.

## Provider references

[CloudFront S3 access](https://docs.aws.amazon.com/AmazonCloudFront/latest/DeveloperGuide/private-content-restricting-access-to-s3.html) describes private origin signing and bucket permissions.
[CloudFront load balancer access](https://docs.aws.amazon.com/AmazonCloudFront/latest/DeveloperGuide/restrict-access-to-load-balancer.html) describes HTTPS, prefix lists, and secret origin headers.
[ECS bind mounts](https://docs.aws.amazon.com/AmazonECS/latest/developerguide/bind-mounts.html) describes image volume ownership and Fargate temporary storage.

[CloudFront error caching](https://docs.aws.amazon.com/AmazonCloudFront/latest/DeveloperGuide/custom-error-pages-expiration.html) describes the separate error response cache controls.

[S3 GetObject permissions](https://docs.aws.amazon.com/AmazonS3/latest/API/API_GetObject.html) describes the listing permission needed to distinguish missing objects.

## Worker network admission

The optional ECS adapter now has [SDK-backed network admission](ecs-network-admission.md) and bounded private settings parsing.
Workers require a dedicated VPC with exact registry endpoints, DNS filtering, and a PrivateLink exchange path to the controller VPC.
The core service template does not provision or enable these worker resources yet.
SDK initialization, command wiring, private routing infrastructure, and worker packaging remain required.
Local protocol checks do not establish deployed network isolation or hosted execution acceptance.
