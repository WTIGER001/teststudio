# Hosted source preparation and discovery

## Current scope

The preparation command connects registry source assembly to the durable source queue and isolated provider discovery.
Kubernetes uses Pods. Optional [ECS settings](ecs-workers.md) now use the same queue with Fargate tasks and ECR authentication.
ECS now has a [staged infrastructure and preparation package](aws-workers-package.md). Deployed acceptance remains required.
It supports the offline Go and Playwright profiles. Local preparation retains its existing behavior.
The API accepts source uploads and presents the normal source review. It does not assemble uploaded source in its process.
The portable service image now includes /app/teststudio-prepare for a separate preparation process.

Native configuration, local TLS protocol checks, and package checks pass without registry or cluster access.
The optional [Helm preparation package](kubernetes-preparation.md) now supplies a separate deployment, durable journal claim, and scoped writer references.
Complete hosted source operation still requires deployment verification.
Environment-backed browser and API profiles require remote gateway delivery.
AWS dispatch is implemented with separate deployment verification pending.
Azure now has a [durable job provider](azure-worker-lifecycle.md) and [job ownership contract](azure-worker-contract.md).
Its concrete isolation controls, independent stop resources, registry authentication, and native preparation wiring remain open.

## Configure separate processes

Use the same PostgreSQL installation and execution-location=kubernetes in the API and preparation process.
Use the protected worker namespace, RuntimeClass, network policy, and private HTTPS origin from [Kubernetes workers](kubernetes-workers.md).
Both processes need their mounted Kubernetes API token and trust configuration.
The API registry identity needs read access. The separate preparation registry identity needs scoped pull,push access.
Inject each identity's values through its own protected TESTSTUDIO_REGISTRY_ environment references.
Never put registry credentials in test Pods, source bundles, public settings, or process arguments.

Enable source staging and discovery on the API alongside its existing sign-in, storage, and listener settings:

```sh
bin/teststudio \
  --database-env=TESTSTUDIO_DATABASE --require-postgres \
  --execution-location=kubernetes \
  --kubernetes-workers-config=/private/kubernetes-workers.json \
  --worker-addr=0.0.0.0:8042 \
  --public-url=https://studio.example.test \
  --artifact-origin=https://evidence.example.test \
  --source-bundles --prepared-sources
```

Run the separate preparation process with immutable supervised runtime distribution references:

```sh
bin/teststudio-prepare \
  --database-env=TESTSTUDIO_DATABASE --require-postgres \
  --execution-location=kubernetes \
  --kubernetes-workers-config=/private/kubernetes-workers.json \
  --build-directory=/private/source-journals \
  --registry-go-runtime=registry.example.test/teststudio/source@sha256:GO_DIGEST \
  --registry-playwright-runtime=registry.example.test/teststudio/browser@sha256:PLAYWRIGHT_DIGEST
```

Replace each digest placeholder with its complete 64-character digest.
Use /app/teststudio-prepare as the service image's command override. Its default entrypoint starts the API.
Pass --database-env explicitly for this command. The preparation command does not inherit the API entrypoint's flags.
The preparation process exposes no HTTP listener. Discovery Pods use the API's existing private worker exchange.
The registry and runtime repositories must satisfy the configured namespace restrictions.
Build and publish the trusted [supervised runtimes](job-supervisor.md#build-a-supervised-source-runtime) through the installation's authorized release process.
Local Docker image identifiers cannot replace registry distribution references.

The selected runtime requires teststudio.discovery-supervisor=worker-exchange/v1 in addition to its existing supervisor and runtime contracts.
An older runtime fails before source publication. Rebuild the thin supervisor overlay without rebuilding its trusted toolchain or browser base.
Hosted settings reject SQLite, another queue location, or local --go-runtime and --playwright-runtime flags.
Registry runtime flags without hosted settings also fail before storage initialization.

Mount a private durable directory at --build-directory. Its mode must exclude group and other access.
Keep that directory available to the workers that recover its jobs. Preserve ownership as the packaged user 65532.
It holds private registry upload journals, including scoped upload addresses. It does not act as the public artifact directory.
Use [registry recovery](registry-preparation.md#ownership-and-recovery) for interrupted or uncertain uploads.

Omit a runner's registry runtime flag to disable new builds for that runner while retaining its cleanup driver.
Set the API's --source-bundle-runners list to match the available builders. Its default is go,playwright.
Keep the worker running with the same database, provider settings, credentials, and journal directory until pending cleanup finishes.
The API retains discovery recovery even when new prebuilt discovery is disabled.
For repository synchronization, use the existing [repository settings](repository-sources.md) and enable the corresponding provider on the preparation process.

## Queue, review, and approval

1. A tester stages a supported ZIP or pinned repository snapshot through the existing source workflow.
2. The preparation worker claims the durable job and assembles source against the selected trusted runtime.
3. Registry publication saves an immutable distribution reference and retained source digest in the existing build checkpoint.
4. The worker creates a gated discovery exchange and one owned discovery worker.
5. The controller verifies provider creation before releasing input to that exact worker.
6. The supervisor invokes only the selected runner's fixed --discover operation with no test work item on standard input.
7. The exchange validates the discovery schema, suite, runner, source digest, and offline profile.
8. A successful provider termination and accepted discovery produce the existing source review.
9. The tester reviews and publishes the source. Publication records exact artifact approval. Activation remains a separate choice.

Discovery reports carry no measured test result or attachments. Discovery never creates a test run or timing baseline.
Raw discovery has an 8 MiB bound. Its scrubbed diagnostic log has a 128 KiB bound.
Normal execution retains its existing report and attachment limits.
Provider success, process exit, report validity, source review, artifact approval, publication, and activation remain separate decisions.
An accepted discovery report cannot hide provider failure.

## Cancellation and cleanup

Each authenticated discovery operation checks the current job lease, actor permissions, cancellation, project, suite, and repository authority.
Discovery always uses the provider activation gate. Workers cannot activate themselves.
The inspector revokes its ticket before returning its observation to the source service.
The source service saves the ready review or failure before permitting provider deletion.
The cleanup guard uses the exact preparation reference instead of a fabricated test-run checkpoint.

Ready reviews can show cleanup pending while their source remains available for review.
Cleanup stops owned work, confirms termination, deletes with provider ownership preconditions, and then removes the exchange and upload journals.
Recovery claims expired leases and performs cleanup only. It never repeats discovery or dispatches another worker.
An interrupted active preparation becomes failed after cleanup. The tester starts any replacement explicitly.
Uncertain provider termination or upload cleanup retains a pending record for another reconciliation attempt.
Published manifests and source blobs remain available for retained releases. Preparation cleanup does not delete them.

## Targeted local evidence

Evidence lives under output/hosted-discovery.
contract.log covers source queue integration, both discovery report formats, admission, ownership, provider lifecycle, and normal execution compatibility with race detection.
provider-outcome.log verifies that provider failure cannot produce a ready review.
deferred-recovery.log covers cleanup with new builds disabled and cancellation of a ready review with an existing cleanup lease.
exchange-compatibility.log retains the normal transport limits and checkpoint behavior.
runtime-package and startup record immutable package metadata, native startup guards, packaged preparation execution, and owned container cleanup.
The trace viewer, trusted Go base, and trusted Playwright base are reused. The Angular build includes updated source help.
No browser, external registry, cluster, AWS, or Azure verification runs.
