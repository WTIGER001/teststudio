# ECS execution and preparation workers

## Current scope

The API and preparation commands now accept optional ECS worker settings.
They construct scoped SDK clients, verify the calling account, authenticate to ECR, and connect the existing durable lifecycle.
The supported runners include offline Go and Playwright, plus optional separate environment execution providers.
Discovery uses the offline gated provider with a separate preparation identity.
The SDK, command registration, source capability checks, and environment lifecycle pass targeted local protocol verification.
Dedicated environment deployment packaging now passes local contract checks. Actual AWS verification and hosted browser recording remain unfinished.
Native HTTP checks retain their existing controller execution.

This implementation has local protocol and native startup evidence only.
The [staged AWS worker package](aws-workers-package.md) now provisions private networking, roles, repositories, and separate preparation.
The core template accepts its generated settings and private target group. Defaults keep coded workers and new source work disabled.
DNS ownership verification, connection sealing, and a healthy private listener remain deployment prerequisites.
Do not treat local package checks as a verified deployed profile or close hosted acceptance from them.
No live AWS account or registry is used during current verification.

## Required settings

Start from [worker-settings.example.json](../deploy/aws/worker-settings.example.json).
Replace every example account, ARN, resource identifier, hostname, and repository namespace with the independent installation's resources.
The example contains no credentials and names no existing deployment.

The worker object identifies the region, account, cluster, image-pull execution role, independent deadline role, and Scheduler group.
It also selects the family prefix, architecture, platform version, subnets, security group, and private exchange origin.
The network object identifies both VPCs, required groups and endpoints, registry namespace, private load balancer, target group, and DNS firewall.
See [network admission](ecs-network-admission.md) and [task contracts](ecs-worker-contract.md) for the exact required state.
The loader derives networkProfile from these settings. Omit that derived field when writing a new configuration.
Both processes must use the same worker and network settings for ownership and recovery.

### Optional environment pool

Keep the top-level worker and network objects for the offline execution and discovery pool.
Add an environment object only when configuring the separate application execution pool.
That object contains worker, network, and gateway objects.
Its network object includes the [gateway network settings](ecs-gateway-network.md), and gateway selects the [trusted task contract](ecs-gateway.md).
All objects must be complete and refer to consistent resources. Partial configuration fails before SDK initialization.

The environment pool shares the offline pool's account, Region, architecture, cluster, exchange service, registry namespace, and controller backend.
It has a separate worker VPC, subnets, security groups, endpoints, and immutable network profile.
Its trusted gateway uses another VPC with narrow peering, separate private exchange delivery, and NAT.
The gateway task and network settings must select the same private subnets and transport group.
The immutable gateway image must belong to the same configured ECR namespace.
Certificate settings contain approved secret references and optional public trust, never private key values.
Omit derived networkProfile fields from new documents. The loader binds both pool identities.

The API and headless controller pass their current private exchange authority to the runtime factory.
Complete environment settings then construct the separate provider and register Go API and Playwright browser execution and recovery.
Offline execution and discovery retain their existing provider.
The preparation command supplies no environment execution authority. Its factory creates only the offline provider and shared registry client.
The SDK registry client and HTTP pools close once with the runtime.

These settings permit manually provisioned native resources after normal network and image admission.
The [worker template](aws-environment-package.md) generates combined settings after separate environment provisioning and activation.
Its defaults keep the environment pool disabled. Supply complete certificate references and an approved immutable image before activation.

Select exactly one input source:

- --ecs-workers-config reads a bounded private JSON file.
- --ecs-workers-env names an environment variable containing the same JSON document.

The JSON document accepts no credential values, endpoint overrides, or arbitrary runner commands.
Keep configuration files private. Never pass the JSON document itself as a command argument.
The artifact-only service role rejects ECS settings. ECS and Kubernetes settings are mutually exclusive on one process.

## Credentials and SDK scope

The optional factory uses the AWS SDK credential chain only after ECS settings are selected.
Prefer an ECS task role for each trusted controller or preparation process. The SDK refreshes workload credentials.
Environment credentials and selected shared profiles follow standard SDK behavior. Changing static credentials requires a process restart.
Worker SDK initialization occurs only when ECS settings are selected.

Startup calls regional STS GetCallerIdentity and checks both its account and ARN account against the configured account.
The initial provider supports only the standard aws partition.
A fresh client configuration fixes the region and excludes ambient service endpoint, logger, proxy, and retry overrides.
Control requests allow only the expected regional STS, ECS, EC2, ECR, ELBv2, Scheduler, Resolver, and Profiles hosts over HTTPS.
Redirects fail. Each request has a five-second HTTP bound and an 8 MiB response bound.
Runtime initialization has a fifteen-second context bound. Service clients disable automatic retries, including creation retries.
Failures return bounded product messages without raw credentials, signed URLs, response bodies, or SDK diagnostics.
Shutdown closes owned registry and SDK idle connections.

The factory verifies account scope at startup. It does not replace IAM authorization or the later resource ownership checks.
Retain the [durable lifecycle](ecs-worker-lifecycle.md) journal and independent deadline controls.

## Registry authority

Registry access is fixed to ACCOUNT.dkr.ecr.REGION.amazonaws.com and the configured repository prefix.
The ECR authorization request selects that account. Its sole response must identify that exact HTTPS registry.
Decoded credentials must use the AWS registry username and a nonempty bounded password.
TestStudio permits passwords up to 64 KiB and rejects larger authorization data. Token size is not assumed to be constant.

Authorization refreshes at least every five minutes, before provider expiry, and when the current SDK credential identity changes.
It never caches beyond workload credential expiry. Concurrent inspections share one authorization request.
An SDK credential retrieval failure rejects inspection even when a registry token remains cached.
A failed token refresh clears that cached registry authority. No stale-token fallback occurs.
Tokens stay in trusted process memory and never enter settings, persisted worker journals, launch overrides, or test input.

Configuration blobs may redirect only to the two regional AWS-owned ECR layer-bucket HTTPS origins listed in the network contract.
The OCI client strips authorization before following those signed redirects and verifies the retained digest and declared size.
Registry namespace, source provenance, supervisor, platform, and non-root volume checks remain mandatory.
Mutable tags and local Docker image identifiers cannot select hosted work.

The API role needs scoped registry reads. The separate preparation role needs scoped image publication as well.
Repositories for the trusted Go and Playwright runtimes must exist before preparation publishes content there.
Registry assembly does not create repositories or grant permissions.
The worker's platform execution role receives image-pull permissions only. It is distinct from the controller and preparation roles.
Test containers receive no AWS task role. A separate Scheduler role can stop only the permitted worker tasks.
Add the read actions in [network admission](ecs-network-admission.md#controller-read-permissions) to both trusted control roles.

## API configuration

After the required private resources exist, select the ECS location and private listener:

```sh
./teststudio \
  --database-env=TESTSTUDIO_DATABASE_URL \
  --execution-location=ecs \
  --ecs-workers-config=/private/ecs-workers.json \
  --worker-addr=0.0.0.0:8042 \
  --public-url=https://studio.example.test \
  --artifact-origin=https://evidence.example.test \
  --prepared-sources \
  --source-bundles \
  --source-bundle-runners=go,playwright
```

Retain the installation's identity, artifact storage, evidence listener, and bootstrap settings alongside these flags.
The database must use PostgreSQL. The private listener must use port 8042 behind the admitted TLS and PrivateLink path.
The exchange origin must differ from UI and artifact origins. Comparison includes normalized host case and default ports.
The factory does not expose port 8042 publicly or create its TLS ingress.
Local sign-in, local image inspection flags, and the local browser gateway cannot be combined with ECS worker settings.

The selected provider supplies both execution and discovery recovery drivers.
New prebuilt discovery requires --prepared-sources. Recovery remains configured when that flag is omitted.
Executable input stays gated until task ownership, the independent deadline, and actual task-interface admission all pass.
Native HTTP dispatch does not depend on this worker path.

## Separate preparation process

Use the same database, location, and settings with a separate trusted preparation identity:

```sh
./teststudio-prepare \
  --database-env=TESTSTUDIO_DATABASE_URL \
  --execution-location=ecs \
  --ecs-workers-config=/private/ecs-workers.json \
  --registry-go-runtime=123456789012.dkr.ecr.us-east-1.amazonaws.com/teststudio/go@sha256:REPLACE_WITH_DISTRIBUTION_DIGEST \
  --registry-playwright-runtime=123456789012.dkr.ecr.us-east-1.amazonaws.com/teststudio/playwright@sha256:REPLACE_WITH_DISTRIBUTION_DIGEST \
  --build-directory=/private/source-journals
```

The shown digest text is a placeholder, not an executable example value.
Build supervised runtimes with [the runtime producer](job-supervisor.md#build-a-supervised-source-runtime) and publish them through the installation's authorized release process.
The runtime must advertise both discovery supervision and the non-root task-volume capability.
Neither local source-runtime flag can be combined with hosted settings.
The API's source-bundle runner list must match the enabled preparation builders.

Mount a private durable journal directory. Keep its ownership and mode suitable for the service image's non-root user.
It stores private upload recovery state. It is not an artifact web root or worker scratch volume.
Preparation retains the existing source queue, review, publication, approval, cancellation, and checkpointed cleanup workflow.
See [hosted preparation](hosted-discovery.md) and [registry recovery](registry-preparation.md#ownership-and-recovery).

Omit both registry runtime flags to disable new source builds while retaining cleanup drivers.
Keep the same database, settings, IAM authority, and journal directory until pending cleanup finishes.
The API must also retain its provider configuration while any execution or discovery resources remain unresolved.
Do not change the network profile as a shortcut for draining old resources.

## Local evidence

Focused SDK and ECR race checks pass in output/ecs-sdk-wiring/final-sdk-contract.log.
They use explicit fixture credentials and intercepted HTTP transports. No credential chain or external endpoint is accessed.
The real SDK signs STS and ECR requests. OCI inspection reads a checksummed fixture image through a credential-free layer-bucket redirect.
Checks cover account mismatch, private provider errors, disabled retries, token rotation, cache expiry, and HTTP response limits.

Twelve native startup checks pass in output/ecs-sdk-wiring/startup/summary.json.
They reject unsupported location, mixed providers, local runners, SQLite, incorrect private ports, ambiguous origins, and invalid private JSON before database creation.
The preparation check without runtime flags reaches the PostgreSQL guard, preserving cleanup-only configuration.
The API and preparation binaries build from current sources. These checks start no browser, cloud worker, or uploaded test suite.
The existing demo and its data remain unchanged. No service image or trusted runtime rebuild is needed for this batch.

## Provider references

[SDK configuration](https://docs.aws.amazon.com/sdk-for-go/v2/developer-guide/configure-gosdk.html) describes region and credential selection.
[GetCallerIdentity](https://docs.aws.amazon.com/STS/latest/APIReference/API_GetCallerIdentity.html) defines the startup account observation.
[ECR authorization](https://docs.aws.amazon.com/AmazonECR/latest/APIReference/API_GetAuthorizationToken.html) describes delegated registry credentials and their expiry.
[AuthorizationData](https://docs.aws.amazon.com/AmazonECR/latest/APIReference/API_AuthorizationData.html) defines the registry endpoint and encoded credential response.
