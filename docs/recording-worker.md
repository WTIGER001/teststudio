# Private recording worker

The recording worker runs the website browser outside the API service. Testers use the existing in-page recorder and install nothing.
The implementation has local Docker, PostgreSQL, private TLS, and normal TestStudio browser evidence. MMDev runs it with a dedicated private host and one browser slot. The current API and UI release is 0.10.2-dev, build c9d8e9d2d4b2. The worker uses the same private host with a reviewed origin policy and frozen network references.
See [the delivery plan](hosted-recording-plan.md) for the separate hosted execution and tester acceptance milestones.

## Installation boundary

Run teststudio-recording-worker on a dedicated Linux Docker host. Keep this host separate from the API, database, preparation workers, execution workers, and other applications.
The trusted worker controls its Docker daemon. Never mount that daemon into the API, recording browser, or gateway.
The current provider uses Docker containers on this host. It does not launch ECS Fargate tasks or Azure Sandboxes.

Allow private API-to-worker HTTPS and worker-to-PostgreSQL access. Permit gateway egress to approved public website addresses.
Do not publish the recording worker through the public load balancer or CloudFront. Testers access only the ordinary managed-session API.
The API and worker share the installation database and permissions. Hosted operation requires PostgreSQL.
Use one stable worker identity and one private HTTPS endpoint per installation. Multiple API replicas can use that endpoint.
Keep the same Docker host, private transport directory, and stable identity through worker restarts so cleanup can find reserved resources.
Moving a worker to another host requires draining and verified cleanup first.

## Build the runtime

Use committed implementation source and a cached trusted Playwright 1.64.1 Linux runtime for the worker host architecture.
The recording image adds only the fixed recording helper and private gateway relay. It contains no application test bundle.

```sh
node scripts/build-recording-runtime.mjs --image=sha256:CACHED_RUNTIME_DIGEST
node scripts/build-browser-gateway.mjs
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath \
  -o bin/teststudio-recording-worker ./cmd/teststudio-recording-worker
```

Select amd64 instead for an amd64 host. Transfer the selected immutable images, binary, and generated .data/recording-worker/seccomp.json through the installation's private artifact process.
Use image digests on the host; mutable tags reject. Registry publication and installation are separate operator actions.
The builder verifies the pinned upstream Playwright policy checksum and adds only chroot for Chromium's namespace sandbox with all host capabilities dropped.
The worker verifies the derived policy checksum before accepting sessions. Keep Chromium sandboxing enabled.

## Private identities and configuration

Issue a dedicated worker server certificate with its private DNS name in its SAN and server authentication usage.
Issue a dedicated API client certificate with teststudio-api.recording.internal in its DNS SAN and client authentication usage.
Keep their private keys and CA references in protected operator storage. Ordinary application session cookies and bearer tokens cannot authenticate this transport.
The worker requires TLS 1.3, a verified client certificate, and the configured API client identity.

Start the worker with the shared database connection supplied through a protected environment variable:

```sh
bin/teststudio-recording-worker \
  --addr=PRIVATE_WORKER_IP:8045 --worker-id=recording \
  --database-env=TESTSTUDIO_DATABASE \
  --tls-certificate=/run/teststudio/worker.crt \
  --tls-key=/run/teststudio/worker.key \
  --client-ca=/run/teststudio/api-client-ca.crt \
  --runtime-image=REGISTRY/recording@sha256:RUNTIME_DIGEST \
  --gateway-image=REGISTRY/gateway@sha256:GATEWAY_DIGEST \
  --seccomp=/etc/teststudio/recording-seccomp.json \
  --directory=/var/lib/teststudio-recording --capacity=2 --idle-limit=2m
```

The directory must be private, with mode 0700. Certificate private keys and API configuration files must have restricted operator access.
Provide a private settings file to the API:

```json
{
  "origin": "https://recording.private.example:8045",
  "serverCAFile": "/run/teststudio/recording-server-ca.crt",
  "clientCertificateFile": "/run/teststudio/recording-api.crt",
  "clientKeyFile": "/run/teststudio/recording-api.key"
}
```

Add --recording-worker-config=/run/teststudio/recording-worker.json to the API's existing installation arguments.
The alternative --recording-worker-config-env accepts the name of an environment variable containing the same JSON settings.
For secret-injected certificate material, replace each file field with serverCAEnvironment, clientCertificateEnvironment, or clientKeyEnvironment and the corresponding variable name.
Each certificate input accepts exactly one file or environment reference. Never put certificate or database secret values in command arguments.
Only the API role accepts this backend. It is exclusive with the companion and native recorder options.
Do not enable --local on a hosted worker. That option permits private fixture targets and restricts its listener to numeric loopback for local verification.

GET /healthz requires the same private TLS identity and checks current worker ownership. A healthy response alone does not prove browser readiness.
Configuration or certificate changes require a controlled restart. Drain active recordings before rotating worker identities or runtime images.

## Session behavior and recovery

The worker reserves capacity and durable resource identities before launching a browser. Reservations bind the original actor, project, suite, session, website, deadline, image digests, and sandbox policy.
Each recording owns one sandboxed non-root browser container, one gateway container, and one private volume.
The browser has no network interface beyond loopback, a read-only root, no host capabilities, and fixed CPU, memory, process, and temporary-storage limits.
Only its Unix-socket gateway can reach the selected website origin. Public mode rejects metadata, private addresses, and unapproved origins.
The worker sends no database, API, or TLS credentials into these containers.

The default policy permits only the entered website origin. Operators can approve exact additional origins for a selected starting origin through --origin-policy=PRIVATE_JSON_FILE.
The file maps each starting origin to an array of approved resource origins:

```json
{
  "https://website.example": ["https://api.example", "https://assets.example"]
}
```

Use canonical HTTP or HTTPS origins without paths, credentials, or wildcards. Each starting origin permits at most sixteen distinct additions; the file permits at most sixty-four starting origins and 64 KiB.
Validate the file before installation with teststudio-recording-worker --validate-origin-policy --origin-policy=PRIVATE_JSON_FILE. Validation opens no database or browser.
The worker freezes the complete network target in its durable reservation. Its resource fingerprint includes the network-policy hash. Changing a policy never silently expands an existing browser's access.
Additional origins still use public-address validation and cannot reach metadata or private networks. Credential-bearing API request policies cannot add origins.
Approve only observed dependencies needed for the selected website. A script or startup API on another origin can otherwise leave an application on its loading screen even when its main Angular scripts load successfully.
Restart the worker through a controlled drain to change its policy. Captured source remains in history; interactive browser control requires a fresh session after restart.
Recording approvals do not change the separate execution profile or automatically grant a saved test the same network access.
There is no execution storage-state transfer, popup, upload, download, or drag recording in this milestone.
The visual frame also does not provide full semantic access to the remote website.

View and input recheck current source authority and archive state. Frames are private, uncached responses and are never persisted in the database.
Input reservations prevent uncertain actions from replaying. Exact retries retrieve a view; changed duplicates and sequence gaps reject.
API restart can reconnect to the same worker without launching another browser. Worker restart fences the old process, removes reserved resources, and preserves the latest captured code and steps; it does not resume interactive control.
The default capacity is two, with an operator range of one through sixteen. Cleanup failures continue to consume a slot until removal is verified.

The original recording lifetime is twenty minutes. Browser and gateway independently stop at that original deadline.
The default disconnected-owner limit is two minutes since the last successful view or input. Status reads do not extend it.
Worker maintenance runs every five seconds and stops expired, idle, revoked, or archived recordings. Ownership renews on a ninety-second lease.
Graceful shutdown captures available source and removes only resources whose names, labels, images, and binding match the saved reservation.
Allow at least ninety seconds for service shutdown and recovery. Inspect pending cleanup before replacing the worker host.
After an abrupt host failure, containers can remain stopped until the same host and stable worker identity recover cleanup.
Installation export requires stopped workers and completed cleanup; private runtime ownership never transfers to another installation.

## Verification and deployment

See [self-testing](self-testing.md#private-recording-worker) for the exact focused harness and retained local evidence.
The generic News journey passes through TestStudio's normal authoring-test execution and saves one revision-1 draft.
This does not establish execution of that saved target test on MMDev.
Before AWS mutations, follow [MMDev deployment](aws-mmdev.md), verify the selected account, and review a concrete TestStudio-only CloudFormation change set.
The dedicated AWS installation uses deploy/aws/recording.yaml and scripts/deploy-aws-recording.py; see the private recording section in the MMDev guide. The core and execution-worker templates remain separate.
MMDev verifies News recording, a visible-story assertion, review, one saved revision-1 draft, second-session cancellation, and complete container and volume cleanup through the hosted UI. Actual host inspection confirms the fixed browser isolation profile. Twelve hosted deployment checks pass. Receipts are output/aws-mmdev/recording-0100 and screenshots are output/playwright/aws-recording-0100.
The private host and API use dedicated transport certificates expiring on 2027-02-06. Rotate the CA and both leaf certificates through a controlled drain before that date.
Keep hosted preparation, execution, and actual tester acceptance separately open.

Release 0.10.2-dev verifies a real selected application whose startup API is hosted separately. Its homepage and News stories render inside the hosted recorder, and the captured News click and story visibility check save as one revision-1 draft. Ten hosted checks pass. Receipts are output/aws-mmdev/recording-origins-0102 and screenshots are output/playwright/recording-site-loading. No target test executes.
