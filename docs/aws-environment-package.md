# AWS environment execution package

## Introduction

The worker template now includes optional browser and coded API environment execution resources.
EnableEnvironment and EnvironmentReady both default to false.
Existing offline resources retain their properties when this option is enabled.
Local schema, configuration, permission, and native network contract checks pass.
These checks simulate AWS metadata. Actual AWS installation and execution verification remain separate work.
Azure remains deferred. The local demonstration and Kubernetes installation remain available.

## Concepts

### Installation boundaries

The optional package adds 59 resources to the existing worker stack.
It creates a separate environment worker VPC and a trusted gateway VPC.
The existing offline worker VPC and controller VPC remain separate.
The environment pool shares the original cluster, image-pull role, deadline role, exchange service, and source repositories.

Environment workers have two private subnets and their own registry endpoints, exchange endpoint, DNS firewall, and security groups.
Their routes add only the two gateway task subnet ranges through one same-account peering connection.
Their additional security rule permits only gateway TCP 8044.
The gateway has two private task subnets, two public NAT subnets, and one public NAT gateway in each availability zone.
Its separate exchange endpoint reaches the original private initialization service.
See [native network controls](ecs-gateway-network.md) for the exact admission requirements.

### Permissions and certificate delivery

GatewayRepository stores an immutable platform gateway image under RuntimeRepositoryPrefix/gateway.
The preparation role retains publication access only to the Go and Playwright source repositories.
Platform image publication uses the installation's separate deployment authority.

GatewayPullRole can pull only the gateway repository and read the two supplied certificate secret references.
GatewayControllerPolicy permits tagged gateway registration, launch, creation tags, and passing that pull role to ECS.
It attaches only to ControllerTaskRoleName. Preparation receives neither gateway launch authority nor gateway certificate access.
Original controller permissions provide owned cleanup, stop, inventory, and independent deadlines in the same cluster.
The gateway task has no task role and receives no uploaded source or output volume.

Store the PEM certificate chain and matching private key in separate Secrets Manager secrets.
Supply full same-account, same-Region secret ARNs without JSON-key or version selectors.
The certificate must cover GatewayHostname. Its DNS name needs no task address record because the supervisor uses the verified private address.
GatewayCertificateAuthority optionally contains base64-encoded public PEM trust, with a 4096-character CloudFormation parameter bound.
Leave it empty for a certificate that uses the runtime's standard trust roots.
GatewaySecretKmsKeyArn optionally supplies the certificate secrets' encryption key.
Its decryption grant requires Secrets Manager in the selected Region and one of the two secret encryption contexts.
Configuration stores references and public trust. It contains no private certificate value or application credential.

### Provisioning and activation

EnableEnvironment provisions the networks, repository, and gateway roles. It does not release environment execution settings.
EnvironmentReady activates the configuration after the trusted image and private endpoints are ready.
This separation permits repository creation and image publication before activation, including upgrades to sealed installations.
Activation requires EnableEnvironment=true, Stage=ready, and a nonempty immutable GatewayImageDigest.

Stage=ready releases one WorkerSettingsSecretArn containing the original worker and network objects plus the optional environment object.
Before environment activation, environment is null and the loader keeps the original offline profile.
After activation, that object identifies the new worker network, gateway network, approved image, and certificate references.
The [settings loader](ecs-workers.md#optional-environment-pool) validates both pools before SDK initialization.
Operators do not copy network identifiers into separate runtime documents.

The API and execution controller use combined settings and current exchange authority.
Preparation reads the same settings but constructs only offline discovery and registry access.
Source assembly and discovery require the verified-address capability for ECS environment artifacts.
Native network, image, task ownership, initialization, health, deadline, and current authority checks still apply before activation.

## Tutorials

### Prepare a new installation

Use the existing [staged worker procedure](aws-workers-package.md#staged-private-connection).
These deployment actions require the separately authorized AWS session. The local package checker performs none of them.
MMDev is the selected AWS account for later verification.

1. Provision the core installation and the separate certificate and key secrets.
2. Select private, non-overlapping ranges for the offline worker, environment worker, gateway, and controller VPCs.
3. Set EnableEnvironment=true and EnvironmentReady=false during Stage=prepare. Supply both certificate secret ARNs and the matching GatewayHostname.
4. Build current service, hosted runtime overlays, and gateway images for the selected architecture. Reuse unchanged runtime bases and UI assets.
5. Publish the trusted gateway to GatewayRepositoryUri and retain its distribution digest. Source runtime overlays must include the verified-address marker.
6. Complete the exchange hostname ownership verification. Confirm failure-open mode is DISABLED on both worker VPCs.
7. Set Stage=connect. The original service can automatically accept the three same-account consumer endpoints during this stage.
8. Confirm the offline, environment worker, and gateway exchange endpoints are Available with private DNS enabled.
9. Set Stage=ready, EnvironmentReady=true, and GatewayImageDigest. The service seals later connections and releases combined settings.
10. Update the core service to the current image and generated settings. A new ECS task must load the changed secret value.

Use scripts/build-service-image.mjs, scripts/build-job-runtime.mjs, and scripts/build-browser-gateway.mjs for current platform images.
The [worker image workflow](aws-workers-package.md) documents architecture selection and immutable distribution references.
Do not substitute Docker image identifiers for ECR distribution digests.

The expanded template exceeds the inline CloudFormation body limit.
Use a private installation packaging bucket with aws cloudformation deploy --s3-bucket.
See the [AWS template upload instructions](https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/using-cfn-cli-package.html).

### Add the option to a sealed installation

Keep the existing stack at Stage=ready and preserve its exchange acceptance requirement.
First update with EnableEnvironment=true, EnvironmentReady=false, certificate references, and the selected network ranges.
The repository and new networks can be created without publishing an active environment profile.
The two new exchange endpoints require explicit connection acceptance while the update proceeds.
Find their requests on the original endpoint service and confirm their VPCs, subnets, and account before accepting them.
Use accept-vpc-endpoint-connections for those two endpoint identifiers.
See [PrivateLink acceptance](https://docs.aws.amazon.com/vpc/latest/privatelink/configure-endpoint-service.html).

Confirm both new endpoints become Available and close failure-open mode on the environment worker VPC.
Publish the current trusted gateway image to the new repository.
Then update with EnvironmentReady=true and its GatewayImageDigest.
Roll the API or controller service so its new task reads the combined settings.
Secret changes alone do not replace existing service tasks.
The native guard keeps dispatch closed until all required observations match.

Drain environment work and finish resource cleanup before disabling this option or changing its network boundaries.
The gateway repository remains retained. Dispose of retained images through the installation's normal retention process.

### Verify the package locally

Run from the TestStudio repository:

```sh
.data/aws-tools/venv/bin/python scripts/verify-ecs-environment-package.py --output output/ecs-environment-package
TESTSTUDIO_ECS_ENVIRONMENT_PACKAGE_FIXTURE="$PWD/output/ecs-environment-package/environment-fixture.json" \
  go test -race ./adapters/execution/ecs -run '^TestECSEnvironmentPackageMatchesNativeNetworkAndTaskAdmission$' -count=1
```

The checker covers six profiles and six rejected configurations, including separate provisioning, activation, and certificate encryption.
It confirms unchanged offline resources, scoped permissions, complete settings, and disjoint network ranges.
Its resource-derived observations feed the production settings loader, network guard, and task interface verifier.
No AWS credentials, cloud connection, image build, browser, or catalog execution is required.
Retain actual ECS application, cancellation, deadline, evidence, and recovery verification for the separate deployment work.
