# AWS worker and preparation infrastructure

## Current package boundary

The independent AWS package now has two CloudFormation templates:

| Template | Responsibility |
| --- | --- |
| deploy/aws/service.yaml | Core API, UI, identity, evidence origins, and optional private worker settings and target registration. |
| deploy/aws/workers.yaml | Offline workers and preparation, plus optional separate environment workers, trusted gateways, and combined settings. |

The default core profile keeps coded workers disabled. The default worker stage creates no exchange endpoint, settings secret, or preparation service.
Local schemas, configuration previews, production admission fixtures, and image probes pass.
The original local package checks perform no AWS deployment, DNS mutation, registry publication, or live EFS access.
Keep acceptance 51, 53, 56, 57 and the related timing and storage criteria open for complete profile verification. Item 54 is retired; 52 and 55 retain their existing verified scope.
Environment-backed coded tests now have an [optional package](aws-environment-package.md) with separate provisioning and activation.
Hosted recording and preparation repository synchronization remain separate work.
EnableEnvironment and EnvironmentReady default to false. The existing offline resources remain unchanged when the option is enabled.
The expanded template requires a private packaging bucket through aws cloudformation deploy --s3-bucket.

The package targets the standard aws partition and a dedicated installation in the authorized MMDev account.
It imports no existing application's tests or infrastructure configuration.

## Verified MMDev prepare stage — 2026-11-14

teststudioapi-workers-bootstrap reaches CREATE_COMPLETE and teststudioapi-workers reaches UPDATE_COMPLETE at Stage=prepare.
The bootstrap contains two resources. The worker stack contains 46 resources and enables no preparation, new builds, environment workers, or test tasks.
The initial DNS association failure is corrected and the five retained resources are imported without deletion or replacement, as described in the recovery section below.
Ten helper tests, eleven package checks, the regional bootstrap schema check, and the race-enabled packaged ECS admission check pass. Four bootstrap and eight worker infrastructure checks pass against AWS.
Both existing ECS service deployments and the core inputs remain unchanged. Reuse output/aws-mmdev/workers-051, including its failed-attempt and restoration receipts.
Next, perform private DNS verification and disable worker-VPC DNS failure-open. Then review connect and ready change sets before API linkage or preparation enablement.
The initial deployment helper stops at prepare. Extend the tracked tooling for later stages and preserve this installation.
This advances item 51 without closing it. The current ledger has 50 verified, six open, 56 active criteria, and retired identifier 54.

## Inputs and existing resources

Use a current [service image](service-image.md) distribution digest and the same service architecture in both templates.
WorkerArchitecture independently selects the Go and Playwright runtime architecture.
Publish matching [supervised runtimes](job-supervisor.md#build-a-supervised-source-runtime) only in the later authorized deployment session.
The worker stack creates retained, immutable Go and Playwright repositories under RuntimeRepositoryPrefix.
It creates no service-image repository and never builds or pushes an image.

Supply these existing controller resources to the worker stack:

| Worker input | Source |
| --- | --- |
| ControllerVpcId | The core installation's VpcId. |
| ControllerSubnetOne, ControllerSubnetTwo | Two existing private controller subnets in different availability zones. |
| AvailabilityZoneOne, AvailabilityZoneTwo | The matching zones for those controller subnets. Worker endpoint subnets use the same zones. |
| ControllerClusterArn | The core stack's ClusterArn output. |
| ControllerTaskRoleName | The core stack's TaskRoleName output. |
| ControllerSecurityGroupId | The core stack's TaskSecurityGroupId output. |
| DatabaseSecurityGroupId, DatabasePort | The existing PostgreSQL group and port. The worker stack adds preparation-only ingress. |
| DatabaseSecretArn, SecretKmsKeyArn | The existing database reference and its optional customer encryption key. |
| ExchangeHostname, ExchangeCertificateArn | A separate private exchange hostname and its valid regional TLS certificate. |
| S3PrefixListId | The region's AWS-managed com.amazonaws.REGION.s3 prefix list. Admission verifies the selected list. |

Controller subnets must support the core database, ECR, regional control APIs, Secrets Manager, and log delivery.
Preparation also uses private EFS mount targets there. Registered native HTTP checks retain their existing controller network access.
The worker stack creates a separate IPv4 VPC with its own DHCP settings and two worker subnets.
Use valid IPv4 networks from /16 through /28. Both worker subnet ranges must fit their VPC and must not overlap each other.
WorkerFamilyPrefix must be installation-specific. It determines the permitted task-definition families and dedicated Scheduler group.
Keep stack names within the documented core limit of 40 characters.
Review both change sets before execution. Deployment requires authority to create IAM resources and modify the specified controller and database groups.

## Staged private connection

Use the stages in order. Existing tasks and cleanup must drain before changing placement or deleting these resources.
The staging mechanism establishes the private connection before requiring acceptance for future connections.
No worker receives executable input during an unsealed stage, even if a caller supplies settings manually.
Runtime admission independently requires acceptance protection, verified private DNS, and fail-closed DNS filtering.

### Initial deployment tooling

Use `scripts/deploy-aws-workers.py` for the initial bootstrap and prepare stages.
It verifies the selected account, TestStudio core ownership, disabled workers, stable ECS services, private controller subnets, and the regional S3 prefix list.
Each phase uses separate prepare, review, execute, status, and verify actions. It refuses to replace an existing deployment or execute an unreviewed change set.
Review freezes the core inputs, service deployments, template digest, parameters, and exact resource additions. Changed inputs require inspection before proceeding.
The helper reads only secret references. It never retrieves database credentials or enables a worker.

The bootstrap template creates a private encrypted package bucket and a regional certificate for a dedicated worker hostname.
Certificate validation adds its DNS ownership record in the selected public zone. It creates no public worker address.
The bucket remains private and retained. The larger worker template is uploaded there under its source digest.
The initial worker phase creates the offline prepare-stage resources. Preparation, new builds, and environment workers remain disabled.
The helper does not implement later connect, ready, or execution enablement stages.

For the authorized MMDev installation, run:

```sh
.data/aws-tools/venv/bin/python scripts/deploy-aws-workers.py prepare \
  --phase bootstrap --profile mm_dev --region us-west-2 --account 253455159142 \
  --core-stack teststudioapi --stack teststudioapi-workers \
  --output output/aws-mmdev/workers-051 \
  --hostname teststudioapi-workers.devfoodmarketmaker.com --zone-id Z05880181BYZPNPNOO9NI
```

Use review with the same phase, account, core, stack, and output arguments after the change set finishes creating.
Inspect its concrete additions before execute. Use status to inspect certificate validation and stack progress; preserve an in-progress deployment.
After bootstrap reaches CREATE_COMPLETE, run verify and repeat prepare, review, execute, status, and verify with `--phase workers`.
The hostname and zone arguments are needed only for bootstrap prepare.
The worker phase reuses the issued certificate and deployed service digest without rebuilding or rolling the core.
Verification inspects actual resources, disabled capabilities, routing, egress, private TLS, journal access, registry settings, and policy attachments.
It also confirms that core inputs and both ECS service deployments match their pre-deployment state.
Keep the private JSON receipts outside Git. They record deployment inputs and review evidence for later stages.

The targeted helper checks run without cloud access:

```sh
.data/aws-tools/venv/bin/python -m unittest discover -s scripts -p test_deploy_aws_workers.py -v
.data/aws-tools/venv/bin/cfn-lint -t deploy/aws/worker-bootstrap.yaml -r us-west-2
```

### Recover an initial rollback without deleting retained data

The initial MMDev attempt exposes a package defect: association priority 100 passes the pinned local schema but fails the live CloudFormation resource handler.
Both offline and environment associations now use 101. Their internal allow and block rule priorities remain 100 and 200.
The package checker separately enforces the association's exclusive 100–9900 range, as documented in [the CloudFormation reference](https://docs.aws.amazon.com/AWSCloudFormation/latest/TemplateReference/aws-resource-route53resolver-firewallrulegroupassociation.html#cfn-route53resolver-firewallrulegroupassociation-priority).

Use `scripts/restore-aws-workers.py` only for an owned initial prepare-stage rollback with preparation, builds, and environment workers disabled.
Its capture action requires ROLLBACK_COMPLETE, unchanged core inputs and service deployments, the exact retained inventory, and the original reviewed retain policies.
Inspect that inventory before retire. Retire removes only the failed stack record and preserves both runtime repositories, the journal filesystem and access point, and the preparation log group.
It refuses additional or still-active resources. It never deletes retained data.
After the failed stack record disappears, prepare creates a concrete import change set containing exactly those five resources.
Run review, inspect the import actions, and then execute. Use status until IMPORT_COMPLETE confirms the original physical identifiers.
The failed deployment receipt is archived as workers-failed.json. The restoration receipt remains restore.json.
CloudFormation imports cannot add stack tags. During import, ownership is bound to the captured stack receipt and exact retained physical identifiers.
The following normal worker update adds the TestStudio stack tag; existing resource tags are preserved during import.
The normal deployment helper can then prepare a fresh update that adds only the missing resources to the imported stack.
It permits only additions and ownership-tag updates on the five captured imported resources. Tag updates must keep the same physical identifiers, require no replacement, and change no resource properties.
It rejects other modifications or removals. Repeat review, execute, status, and verify for the corrected worker stage.

Every restoration action uses the selected profile, region, account, core stack, and private output directory:

```sh
.data/aws-tools/venv/bin/python scripts/restore-aws-workers.py capture \
  --profile mm_dev --region us-west-2 --account 253455159142 \
  --core-stack teststudioapi --output output/aws-mmdev/workers-051
```

Use the same arguments for retire, prepare, review, execute, and status. Preserve an in-progress rollback or import.

### 1. Create the core profile without coded workers

Deploy the core template with WorkerSettingsSecretArn and WorkerTargetGroupArn empty.
Keep WorkerNewSources=false. Retain its identity, database, and artifact configuration.
The core now uses execution location ecs for both native HTTP work and optional coded workers.
Do not move an active installation from its earlier aws-ecs location without first draining its old queue and schedules.
Existing retained history remains unchanged. Deployment upgrade acceptance remains unverified.

### 2. Create the worker stack with Stage=prepare

Keep EnablePreparation=false and EnableNewBuilds=false.
The stack provisions private networking, registry endpoints, the internal TLS load balancer, exact permissions, repositories, and private journals.
It attaches bounded control authority to the existing API role and a separate preparation role.
It creates no running untrusted task. Those tasks remain controller-owned per-attempt resources.

The endpoint service initially permits only the current account and does not require acceptance.
No consumer endpoint or worker configuration is released in this stage.
The internal load balancer has no direct ingress rule. It accepts only the later PrivateLink path.

### 3. Verify private DNS and close DNS failure mode

Private DNS ownership and firewall failure mode are explicit operator steps for the staged resources.
The pinned regional CloudFormation schema does not provide the DNS configuration fields used by these operations.
The template does not invent an unsupported resource or substitute a broad DNS allow rule.
The runtime checks their actual AWS state before dispatch.

The following commands are instructions for the separate deployment session. They are not executed by the local package checker.
Use the account and region selected for this independent installation. Substitute the worker stack's output identifiers:

```sh
aws ec2 modify-vpc-endpoint-service-configuration \
  --service-id SERVICE_ID \
  --private-dns-name EXCHANGE_HOSTNAME

aws ec2 describe-vpc-endpoint-service-configurations \
  --service-ids SERVICE_ID \
  --query 'ServiceConfigurations[0].PrivateDnsNameConfiguration'
```

Publish the returned TXT name and value in the hostname's public authoritative DNS zone.
This ownership record does not publish an address for the private worker listener.
Then start verification and inspect its state:

```sh
aws ec2 start-vpc-endpoint-service-private-dns-verification --service-id SERVICE_ID
aws ec2 describe-vpc-endpoint-service-configurations \
  --service-ids SERVICE_ID \
  --query 'ServiceConfigurations[0].PrivateDnsNameConfiguration.State'

aws route53resolver update-firewall-config \
  --resource-id WORKER_VPC_ID \
  --firewall-fail-open DISABLED

aws route53resolver get-firewall-config --resource-id WORKER_VPC_ID
```

Wait for verified private DNS and confirm DISABLED failure-open state.
The allowlist contains only the exchange hostname, two ECR names, and two AWS-owned image-layer bucket names.
It blocks other names with NXDOMAIN. It applies only to the worker VPC.
No Route 53 Profile association is permitted there.

### 4. Connect, then seal

Update the worker stack to Stage=connect. Keep preparation and new builds disabled.
This creates its IPv4 exchange endpoint with private DNS while same-account connections can be accepted automatically.
Confirm that the endpoint reaches Available and uses the expected service, subnets, and group.
Do not skip directly from prepare to ready. A new endpoint requiring manual acceptance can stall stack creation.

Update the same stack to Stage=ready.
The service now requires acceptance for any later connection request.
The stack creates WorkerSettingsSecretArn containing references to these exact resources. It contains no provider or worker credentials.
Both API and preparation consume this one generated document. Operators do not need to copy individual endpoint identifiers into JSON.

### 5. Connect the API and enable preparation

Update the core stack with the worker stack's WorkerSettingsSecretArn and ExchangeTargetGroupArn outputs.
The core template adds only private port 8042 and its target registration, plus the settings environment reference.
The two public API and artifact target groups retain their existing ports and routing.
Worker settings use startup-only Secrets Manager access. Runtime evidence and worker-control permissions remain separate.
The API's task interface retains its sole controller security group.

Confirm that the core is ready and the private target group reports healthy listeners before enabling preparation.
The private target uses HTTP /healthz with expected status 404. The private listener intentionally exposes no health route.
This verifies listener reachability; public database readiness remains /healthz on port 8040.

Update the worker stack with EnablePreparation=true. Leave EnableNewBuilds=false for initial recovery-only startup.
The preparation task has no public address, listener, or load balancer registration.
It shares PostgreSQL and private EFS journals with replacements. It does not share a filesystem or IAM role with test tasks.
Journal mounts use TLS, IAM authorization, an exact access point, and user 65532 with mode 0700.
The EFS policy denies plaintext mounts, root access, and mounts without an access point.
Only the preparation role has the access-point-specific mount and write authority.

To enable new source preparation, supply the published GoRuntimeDigest or PlaywrightRuntimeDigest and set EnableNewBuilds=true.
Then set WorkerNewSources=true in the core stack and match WorkerSourceRunners to the selected runtimes.
A Go-only preparation service requires WorkerSourceRunners=go. Both runtimes use go,playwright.
The API enables source staging and prebuilt discovery. Existing review, approval, publication, and activation remain explicit.

## Permission and storage boundaries

The API and preparation controller policy can inspect the admitted network and manage only the selected worker cluster, families, and deadline group.
Where AWS requires wildcard reads, the policy restricts the requested region and the implementation checks exact ownership.
RunTask and stop authority bind the worker cluster. PassRole names only the image-pull and deadline roles with their respective services.
Tag creation requires the product's worker marker and only the corresponding creation action.
Stop authority does not depend on tags that StopTask removes.

The platform execution role can pull within the configured runtime namespace. It has no artifact, secret, or journal permissions.
The independent Scheduler role can only stop tasks in that worker cluster.
The API has registry reads. Only the separate preparation role can publish to the two owned runtime repositories.
Untrusted test containers have no AWS task role, persistent mount, registry password, or controller credentials.

The worker group permits no ingress and only HTTPS to the exchange group, registry group, and regional S3 prefix list.
Endpoint groups permit only worker HTTPS ingress and a bounded HTTPS peer egress rule back to that worker group.
The worker group has no ingress, so this peer rule creates no reachable worker listener.
Its explicit declaration suppresses CloudFormation's default allow-all outbound rule without a custom mutation function.
Admission also accepts an endpoint group with no egress rules. It rejects broader or different destinations.
The internal load balancer permits egress only to the controller's worker port.

The private journal filesystem, access point, runtime repositories, settings secret, and log groups retain their data on stack deletion or replacement.
Automatic EFS backup is enabled. The application still controls run evidence retention in its separate object store.
Restore the database, private journals, and retained worker settings together when recovering an installation.
Keep unresolved resource receipts and upload journals. Do not manually remove uncertain worker tasks as a substitute for checkpointed recovery.

## Drain and cleanup-only operation

First set core WorkerNewSources=false to stop new source staging and discovery.
Then set worker EnableNewBuilds=false while retaining EnablePreparation=true and Stage=ready.
This removes new-runtime flags but retains the preparation process, private journals, registry authority, and cleanup drivers.
Disable project schedules and stop new runs through the existing authorized controls when draining execution.
Wait for active attempts, source claims, worker exchanges, and ECS dispatch cleanup to finish.
Do not clear core worker references, change the worker network profile, reduce preparation to zero, or delete the stack before that checkpoint.
Task-definition deletion can remain pending for an hour after termination. The saved result remains available during that delay.
API backend rotation now permits known startup and draining transitions while at least one owned target stays healthy.
The guard checks every existing target interface, including retiring interfaces. Only confirmed deregistration permits an absent interface.
Cross-zone routing and health thresholds use the fixed [network admission contract](ecs-network-admission.md).
Live rollout availability and replica recovery still require deployment verification.
Before upgrading older worker stacks, drain active work and apply the updated target attributes and controller read policy.
Then deploy a freshly built service image. Older cached service images do not include the new admission behavior.
Future compatible API rotations can retain admission while their owned healthy backend remains available.
After a complete drain, disable preparation and worker references before removing worker infrastructure.
Retained storage needs its own reviewed disposal decision.

## Local evidence and limits

Run the targeted checks from the repository root:

```sh
.data/aws-tools/venv/bin/python scripts/verify-aws-workers.py --output output/aws-workers-package/check
TESTSTUDIO_AWS_PACKAGE_FIXTURE="$PWD/output/aws-workers-package/check/admission-fixture.json" \
  go test -race ./adapters/execution/ecs -run 'TestPackagedECSNetwork|TestECSNetwork' -count=1
```

The baseline checker covers 50 ready-stage worker resources, 33 core resources, nine stage or service configurations, and eleven rejected configurations.
The [environment checker](aws-environment-package.md#verify-the-package-locally) covers the additional 59 optional resources and native network contract.
Both templates pass pinned local schemas in two commercial regions. IAM policy size and scoped relationships are checked locally.
The generated fixture feeds actual policies, groups, endpoints, DNS rules, and settings into production admission code.
It simulates AWS metadata after the documented DNS setup. It proves contract alignment, not deployed isolation or policy enforcement.

The changed ARM64 service image passes three local probes in output/aws-workers-package/image.
Its packaged API and preparation flags reject SQLite before provider access.
A real local volume permits non-root preparation to initialize a private child under a read-only root filesystem.
The stopped SQLite copy uses immutable read mode. Its first probe used the wrong read mode for a copied WAL database; the checker corrects that artifact read.
Owned probe containers and volumes are removed after both failed and successful checks.
The UI, trace viewer, and trusted runtime bases are reused. The demonstration remains available.

The updated worker template also passes local rollout-attribute and read-authority checks in output/ecs-rollout/package.
Its generated settings and target attributes pass the production admission guard with simulated SDK responses.
Use an absolute fixture path because Go executes each test from its package directory:

```sh
TESTSTUDIO_AWS_PACKAGE_FIXTURE="$PWD/output/ecs-rollout/package/admission-fixture.json" \
go test -race ./adapters/execution/ecs -run '^TestPackagedECSNetworkMatchesAdmissionAfterDocumentedDNSSetup$' -count=1
```
The rollout batch changes no UI or source runtime. It requires no image rebuild for local contract verification.
A package test first uses a relative fixture path from the wrong working directory. The corrected absolute path passes.

Live worker IAM enforcement, PrivateLink transport and DNS ownership, mounted EFS access, registry publication, Fargate execution, cancellation, deadlines, replica recovery, and upgrades remain unverified.
Perform those checks only in the separate authorized MMDev deployment session.

## Primary references

[PrivateLink DNS ownership](https://docs.aws.amazon.com/vpc/latest/privatelink/manage-dns-names.html) describes the TXT record and verification state.
[Endpoint lifecycle](https://docs.aws.amazon.com/vpc/latest/privatelink/concepts.html) describes connection acceptance and availability.
[Security group defaults](https://docs.aws.amazon.com/AWSCloudFormation/latest/TemplateReference/aws-resource-ec2-securitygroup.html) explains automatic outbound rules.
[ECS authorization](https://docs.aws.amazon.com/service-authorization/latest/reference/list_ecs.html) defines supported resources and condition keys.
[EFS authorization](https://docs.aws.amazon.com/efs/latest/ug/iam-access-control-nfs-efs.html) describes client permissions and access-point conditions.
