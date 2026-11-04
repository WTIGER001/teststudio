# Recorded website execution

Use this local profile to run reviewed Playwright source against one registered application environment.
The offline starter keeps its separate execution profile.

## Tester workflow

1. Register the application and its environment from Applications.
2. Use the environment endpoint name `api`. Enter the website origin without a path, query, or embedded credentials.
3. Select the Local stage for a local HTTP target. Keep stage and protection as separate decisions.
4. Open Create test and choose Playwright browser test. Record navigation, actions, and assertions.
5. Stop the recorder and select Review recorded test.
6. Review the visible source, assertions, timeout, capture settings, and needs-review safety label. Save the draft.
7. Select Validate and prepare draft. Review the discovered catalog and publish its release. Activate it for the normal run wizard.
8. Open Test runs. Select the suite, registered environment, and desired tests. Review the selection and start the run.

Publication does not start execution. An inactive release remains available through explicit release selection in the run API.
Protected environments require protected execution authority and a safe selection, including dependencies.
Recording a flow does not establish that the flow is safe for application data.

## Target binding

New recording conversions replace supported start-origin `page.goto` literals with an explicit `TESTSTUDIO_BASE_URL` expression.
The source editor shows this change before saving or preparation. The original generated source remains in the private recording history.
The expression preserves the recorded path, query, and fragment. Execution supplies the selected environment origin.
Foreign-origin literals remain visible. The gateway rejects connections outside the selected endpoint.
Arbitrary uploaded code receives no automatic source rewrite.
Old saved drafts can retain literal localhost navigation. Convert the original recording again or update and review that source explicitly.

The generated manifest uses `buildProfile: playwright-environment` and `browser: { endpoint: api }`.
Complete source bundles can declare another valid endpoint identifier in `browser.endpoint`.
The prepared artifact retains this policy with its immutable source provenance. Publication rejects changes to that verified artifact.

## Network and request policy

The unprivileged browser worker keeps Docker `network=none`. It has no direct route to an application or the internet.
A separate trusted gateway connects to the registered endpoint through a private Unix socket on the owned execution volume.
A loopback relay connects Playwright's browser and request fixtures to that socket.
The gateway publishes no TCP port. It checks each HTTP proxy request and CONNECT destination against the selected origin and port.
DNS resolution checks every returned address before connection. Metadata and link-local destinations remain blocked.
Private addresses require the Local stage or the environment's explicit private-network setting.

This profile permits all request methods to that endpoint. It does not promise read-only requests or inspect encrypted request contents.
CONNECT tunnels serve browser HTTPS and Playwright's HTTP and HTTPS request clients.
Third-party assets, off-site redirects, and separate login origins fail unless they remain within the selected endpoint.
The profile has no additional-origin list, custom request policy, or credential-binding injection.
Recorder cookies and personal browser storage never become execution credentials automatically.
Explicit source can still contain credentials. Review source and configure private evidence access before publication.

Use Playwright's configured `page` and `request` fixtures for the first workflow.
Independent clients must use the supplied proxy. Clients that bypass it cannot reach external networks.
The runner blocks service workers and rejects suite `webServer` configuration in this profile.
Browser contexts start separately from the recording session.

## Local addressing and recovery

The recorder uses the TestStudio server desktop. The execution browser runs inside a container.
Container localhost refers to that container. Admission rejects localhost, loopback, and unspecified target addresses before creating the run.
On Docker Desktop, a host service can use `http://host.docker.internal:PORT` as its registered origin.
Bind the fixture to an interface reachable from Docker. Native Linux needs an explicit worker-reachable host address or operator-managed hostname mapping.
Execution-location restrictions still apply. This profile supports only the local location.

The gateway must become healthy before browser dispatch. Startup failures produce a configuration result with a bounded diagnostic.
Unreachable targets produce browser or request errors. Inspect the failed step and retained logs before changing the target or retrying.
Permissions, test state, dependency safety, environment policy, and worker ownership are checked again before dispatch.
Changing endpoint addresses, network flags, credentials, stage, or reachable locations requires a newly reviewed run.
Capture limits, timeouts, failure reruns, timing observations, and private evidence use the existing run contracts.
Owned gateway containers, worker containers, and output volumes use the durable execution cleanup and recovery path.

## Operator setup

Build the trusted gateway without application source:

```sh
node scripts/build-browser-gateway.mjs
bash scripts/build-playwright-runtime.sh
```

The gateway build prints its immutable image identifier and caches unchanged build inputs.
Enable source bundles and pass that identifier to the API:

```sh
bin/teststudio --local --source-bundles \
  --browser-gateway-image sha256:TRUSTED_GATEWAY_IMAGE
```

Configure the separate preparation worker with the trusted Playwright runtime as described in [Source bundles](source-bundles.md).
Omitting the gateway flag keeps new environment-backed execution disabled. Cleanup and recovery still recognize existing gateway references.
The demo launcher builds and enables the gateway. It preserves saved data when restarted.
Hosted browser execution and multi-origin or credential-bound profiles remain separate delivery work.

The runner uses Playwright's documented [proxy and base URL options](https://playwright.dev/docs/test-use-options).
