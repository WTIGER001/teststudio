# MMDev deployment

## Introduction

The user authorizes deployment to the MMDev ECS cluster on 2026-11-05. The new service name is teststudioapi.
Use AWS profile mm_dev, account 253455159142, and region us-west-2.
The existing cluster is marketmaker-testing-dev. Preserve its testing-server service.
Azure implementation and verification remain deferred.

## Concepts

### Installation boundaries

The core service uses the existing cluster through ExistingClusterArn and the exact ServiceName parameter.
CloudFormation does not import or modify that cluster or its existing service.
The foundation stack creates dedicated TestStudio resources. It copies no existing application configuration, secrets, tests, or data.
The VPC has two public load-balancer subnets and two private task and database subnets.
One NAT gateway provides development egress. This is a single-zone egress dependency.
PostgreSQL uses one private encrypted db.t4g.micro instance, seven-day backups, and snapshot retention during replacement or deletion.
This development profile does not establish high availability.
The core stack admits its task security group to the database on TCP 5432.
Database TLS verifies both the certificate chain and hostname using the reviewed public RDS CA bundle.

The dedicated Cognito pool permits operator-created users only. It supplies a public OIDC client with authorization-code flow.
The initial administrator is studioadmin. Its generated password stays in Secrets Manager at teststudioapi/initial-administrator.
No password enters source control, command arguments, deployment receipts, or chat output.
The runtime task receives the database URL and OIDC configuration through selected startup secrets.
It receives no permission to read the administrator credential.
The UI and evidence use separate CloudFront origins and hostnames. The load balancer admits only guarded CloudFront traffic.

### Current progress

The initial installation completes both stacks. The hostname update now leaves both stacks at UPDATE_COMPLETE.
Both CloudFront distributions report Deployed. The canonical UI address is https://teststudio.devfoodmarketmaker.com.
The ECS service name remains teststudioapi.
The reviewed foundation change set adds 29 resources. The reviewed core change set adds 31 resources.
Neither change set modifies existing resources. The core change set creates no cluster.
ECS reports one desired and one running teststudioapi task with a completed rollout.
Both API and artifact target groups report healthy. The existing testing-server remains at one running task.
The ARM64 image is sha256:8e9af90c8cbbc397593946e2b266446dd062adc3dcb37a60eaed36ec289e371f.
It is published in teststudioapi/service. The versioned release is 0.1.0-dev, displayed as 0.1.0.dev. Build e9dc1c0ca5e9 uses committed source ca6968f.
Its UTC creation date is 2026-11-06T14:18:59.251Z. The API, UI, and image labels share that immutable identity.
The unchanged trace viewer is reused. Native image storage uses the existing private database. The image retains the reviewed public RDS trust bundle.

Actual Cognito sign-in, catalog publication, and one native HTTPS probe now pass through the public installation.
The original run is 20261106T014449.692767879-e0841a5b4b501c410e0db21d in project aws-core.
It passes in 167.768248 ms, including a 16.519076 ms TLS handshake.
One attempt, four events, four log segments, and four artifacts are retained.
A private evidence download matches its size and SHA-256 checksum. The owned application session signs out afterward.
The first setup attempt omits the publication idempotency key. It creates no test run.
The corrected check reuses that draft and creates one measured run. Future invocations reuse the saved run reference.
The public index and main JavaScript match the built UI bytes. Anonymous project access returns 401.
The evidence bucket reports AES256 encryption and all four public-access blocks enabled.
No browser, coded worker, or unrelated service is started or modified by these checks.

Complete AWS acceptance remains open. Isolated coded workers, preparation, two-provider permission parity, lifecycle, and storage-policy scenarios require further work.
The full cross-profile ledger remains 50 verified of 57. This milestone advances 41, 46, 54, 56, and 57 without closing them.

## Tutorials

### Reproduce the deployment inputs

The foundation template is deploy/aws/foundation.yaml. Review its dedicated development resources before creating a new stack.
The core template is deploy/aws/service.yaml. Supply ExistingClusterArn to reuse a cluster without taking ownership of it.
The deployed core parameters remain .data/aws-mmdev/core-parameters.json. They contain references and public identifiers, not credential values.
Use foundation stack outputs for VPC, subnets, database access, registry, origin certificate, and identity configuration references.
Use the existing us-east-1 viewer certificate only when its names cover both public hosts.
Use the registry distribution digest from output/aws-mmdev/published-image.json, not an assumed local image identifier.
Publish the matching web/dist/web/browser assets into the core UiBucketName output. Keep index.html uncached.

Create the initial identity after the foundation completes:

```sh
python3 scripts/bootstrap-aws-administrator.py --profile mm_dev --region us-west-2 \
  --account 253455159142 --stack teststudioapi-foundation --output output/aws-mmdev/administrator.json
```

The helper verifies account and stack ownership. It creates only the dedicated studioadmin user when that user is absent.
It uses protected temporary request files and removes them after each operation.
An existing confirmed user's password remains unchanged. An existing unconfirmed user requires explicit operator recovery.
Use its exact issuer and subject receipt for BootstrapIssuer and BootstrapSubject. Never derive the subject from a username or email.

Create and review a core change set with the complete parameter file:

```sh
aws cloudformation create-change-set --profile mm_dev --stack-name teststudioapi \
  --change-set-name REVIEWED_CHANGE_NAME --change-set-type UPDATE \
  --template-body file://deploy/aws/service.yaml --parameters file://.data/aws-mmdev/core-parameters.json \
  --capabilities CAPABILITY_IAM
```

Use CREATE only for an absent stack. Review changes before execution. Preserve an in-progress stack and inspect its status before retrying.
The initial reviewed change sets remain recorded as initial-teststudioapi-foundation and initial-teststudioapi-service.
Retained buckets, ECR images, identity, logs, and database snapshots can survive stack removal. Removal requires a separate explicit request.

### Inspect deployment progress

```sh
aws sts get-caller-identity --profile mm_dev
aws cloudformation describe-stacks --profile mm_dev --stack-name teststudioapi-foundation \
  --query 'Stacks[].{Status:StackStatus,Outputs:Outputs}'
aws ecs describe-services --profile mm_dev --cluster marketmaker-testing-dev \
  --services teststudioapi testing-server \
  --query 'services[].{Name:serviceName,Status:status,Desired:desiredCount,Running:runningCount}'
```

Review status and existing resources before continuing an interrupted operation. Do not recreate a pending stack.
The hostname update changes only the Cognito callback, core public URL, CloudFront alias, and UI DNS record.
The existing service image, UI assets, database, and evidence bucket are retained.
Normal sign-in and private access to the original run pass on the new host. The owned probe environment now uses that host.
No additional test run starts for this hostname update.

The local receipt directory is output/aws-mmdev. Foundation changes and published image metadata contain no credential values.
Initial private request files use .data/aws-mmdev. The bootstrap helper uses protected operating-system temporary files.
Both remove request files immediately after their operation.
ECR authentication uses a temporary private Docker configuration, removed after publication.

### Sign in after deployment

Open https://teststudio.devfoodmarketmaker.com and choose TestStudio sign-in.
Retrieve the initial credential through the MMDev Secrets Manager console. Select teststudioapi/initial-administrator.
Use username studioadmin and the stored password. Rotate the password through authorized Cognito administration after handoff.
The evidence origin is https://teststudio-evidence.devfoodmarketmaker.com.
Native HTTPS probes are available in the core profile. Coded workers require the separate worker installation.
The desktop companion can record and save a draft independently of coded execution.
A healthy core deployment does not prove all hosted execution, storage, timing, identity, or recovery criteria.
Keep all seven open acceptance items unchanged until their complete requirements pass.

Provider references: [ECS service cluster selection](https://docs.aws.amazon.com/AWSCloudFormation/latest/TemplateReference/aws-resource-ecs-service.html), [RDS PostgreSQL TLS verification](https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/PostgreSQL.Concepts.General.SSL.html).

### Organization OIDC sign-in

The user selects existing Cognito pool us-west-2_p1Epu0go8 for organization sign-in.
That external issuer is in account 507334592829. TestStudio's service and database remain in MMDev account 253455159142.
Stack teststudio-oidc-login creates one dedicated public app client in the selected pool.
The generic template is deploy/aws/cognito-client.yaml. ClientName, UserPoolId, IdentityProviders, UiHostname, and ProviderId are parameters.
It creates no pool, identity-provider secret, user, application role, or modification to another client.
Cognito owns the configured upstream methods. TestStudio's application code and service image remain unchanged.

| TestStudio setting | Configured value |
| --- | --- |
| Provider identifier | organization |
| Display name | Organization sign-in |
| Issuer | https://cognito-idp.us-west-2.amazonaws.com/us-west-2_p1Epu0go8 |
| Client identifier | hn7kqpel11og80p4ho47tsb9i |
| Client secret | None. This client uses authorization code with PKCE. |
| Scopes | openid profile email |
| Callback | https://teststudio.devfoodmarketmaker.com/auth/callback/organization |
| Profile creation | Enabled. New profiles receive no automatic project or installation roles. |

The existing TestStudio sign-in remains available for the bootstrap administrator.
While signed in as that administrator, open Your account and Link another sign-in.
Select Organization sign-in, enter a reason, verify the current account, and verify the new identity through Cognito.
Review the issuer and subject, then select Confirm identity link. Later organization sign-in uses the same account and administrator roles.
Starting a separate sign-in before linking can create another account. It does not inherit the bootstrap administrator's roles.

The managed provider is registered through the protected provider API. No service restart or issuer-specific application code is required.
Public discovery and the actual application redirect verify the issuer, client, callback, code flow, PKCE, state, and nonce.
The redirect contains no upstream-provider hint. Cognito returns its own sign-in page with its configured choices.
An interactive organization login and account linking still require the user. These checks do not claim a completed human authentication.
The unused initial provider-specific staging client is removed. The three original clients in the selected pool remain available.
Receipts are output/aws-mmdev/organization-provider.json, oidc-client.json, and organization-signin-check.json.
See [Identity providers](providers.md) for generic OIDC administration and account-linking behavior.

### Verify the first core workflow

```sh
python3 scripts/verify-aws-core.py --profile mm_dev --region us-west-2 \
  --account 253455159142 --stack teststudioapi \
  --credential-secret teststudioapi/initial-administrator --output output/aws-mmdev/core
```

The command requires the core stack to report complete. It checks the account and TestStudio ownership before application mutations.
It uses actual Cognito sign-in and one newly authored HTTPS probe against this installation's health endpoint.
A saved run reference and stable idempotency key reuse the original run after interruption.
The check verifies measured execution, retained attempts, events, logs, and a private evidence download with its checksum.
It signs out its owned application session. It starts no browser and executes no coded worker.
It does not establish complete AWS or cross-provider acceptance.

### Publish a reviewed application release

Use the tracked scripts/publish-aws-core.py helper instead of workstation-only release snippets.
Build the portable service from committed source and retain its JSON output in a private ignored image receipt.
Run prepare with profile, region, account, stack, image-receipt, and receipt arguments.
It verifies the account and stack ownership, uses a temporary Docker login configuration, publishes the selected image, and creates an image-only change set.
Review the concrete change set with its review action before execute. The review permits only the owned TaskDefinition and Service changes.
Run finish to inspect an in-progress rollout without starting another. It publishes the matching UI only after the selected service digest is healthy and its API metadata matches.
The receipt records immutable build creation separately from UI publication. It contains no credentials or guarded origin headers.

scripts/manage-https-probes.py uses the ordinary hosted sign-in and project APIs with explicit private configuration.
Its inspect, configure, run, enable, and check actions keep configuration, execution, schedule enablement, and observation inspection separate.
It reuses checkpointed run identities and revokes its owned application session on exit.
See [HTTPS probes](https-probes.md) for user workflows and current overview limits.
