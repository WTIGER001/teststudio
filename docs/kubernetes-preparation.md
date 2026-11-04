# Kubernetes source preparation package

## Scope

The optional Helm preparation workload processes offline Go and Playwright source bundles through registry assembly and isolated discovery.
It uses the same immutable service image as the API and overrides its command with /app/teststudio-prepare.
The API presents the existing source review, publication, and activation workflow.
During preparation, source discovery runs only in isolated Pods. The trusted preparation process assembles source layers without Docker.

This package passes local manifest, configuration, and startup checks. It has no verified cluster deployment yet.
The current chart enables ZIP source staging and prebuilt image discovery. It does not configure GitHub repository synchronization.
Native [repository synchronization](repository-sources.md) remains available through explicit separate-process configuration.
Environment-backed browser and API artifacts require remote gateway delivery. Those hosted profiles remain unavailable.

## Installation inputs

Configure the [core service](kubernetes.md) and [protected worker profile](kubernetes-workers.md) first.
Supply an immutable service image that includes the preparation command and the source-bundle-runners API option.
Use trusted supervised Go and Playwright runtime distribution references from the selected registry namespace.
The runtime must advertise teststudio.discovery-supervisor=worker-exchange/v1. Local Docker image identifiers are insufficient.

Keep the API's registry identity read-only. Create a separate existing Secret for the preparation registry writer.
Grant that writer only the registry permissions required to inspect bases, publish source, and cancel its owned uploads.
The chart references Secret keys; it creates or embeds no credential values.
Writer and API registry reader references cannot use the same Secret object.
Configure Basic username/password references or one Bearer reference. Mixed authentication and unused writer references are rejected.
The writer uses the same registry origin, repository namespace, token endpoint, and trust settings as the reader.

Supply durable journal storage separately from artifact storage.
The claim must support filesystem ownership by user and group 65532 and preserve private directory permissions.
The Pod mounts it at /journals. The preparation command creates /journals/private with mode 0700.
Do not precreate that child with group or other access. Startup rejects an unsafe directory or symbolic link.
The claim holds private registry upload addresses and ownership records. Back it up and restrict storage access accordingly.
It must remain available to replacement preparation Pods that reconcile interrupted jobs.

## Review the configuration

Start with these new independent examples and replace their installation references and digest placeholders:

```sh
helm template teststudio deploy/kubernetes --namespace teststudio \
  --values deploy/kubernetes/values.example.yaml \
  --values deploy/kubernetes/workers.example.yaml \
  --values deploy/kubernetes/preparation.example.yaml
```

The command renders resources locally. Deployment remains a separate authorized operation.

| Value | Behavior |
| --- | --- |
| preparation.enabled | Creates the separate workload, service account, configuration, ingress-deny policy, and journal claim reference. Default false. |
| preparation.acceptNewWork | Enables API source staging and the configured builders. Default true when the workload is enabled. |
| preparation.goRuntime | Immutable supervised Go runtime distribution reference. Empty disables new Go builds. |
| preparation.playwrightRuntime | Immutable supervised Playwright runtime distribution reference. Empty disables new Playwright builds. |
| preparation.registry | Writer environment names and existing Secret key references. No raw credentials are accepted. |
| preparation.persistence | Existing private claim or a retained claim created by the chart, with storage class, size, and access mode. |
| preparation.replicas | Preparation workers sharing the same PostgreSQL installation and journal storage. Default 1. |
| preparation.resources | Separate CPU, memory, and temporary-storage requests and limits. |

At least one runtime is required when new work is enabled.
The API advertises only the source builders selected by those runtime flags.
Go-only installations offer Go ZIP staging. Prebuilt discovery remains separate from source assembly and can inspect either supported runner.
Native installations use --source-bundle-runners=go, --source-bundle-runners=playwright, or the default go,playwright on the API.
Keep native API advertising aligned with the builders available at that execution location.

The default single-replica journal claim uses ReadWriteOnce and a Recreate rollout.
Multiple preparation replicas require a shared ReadWriteMany claim. They use rolling updates and a disruption budget.
An existing claim's actual access mode and mount behavior require deployment verification.
The chart keeps newly created journal claims after release removal. Supply their names through existingClaim when reusing them.
Journal and filesystem artifact claims must remain separate, including when either uses an existing claim.

## Identity and process boundaries

The preparation workload has a distinct service account with no automatic token mount or inherited API workload identity annotations.
Its explicitly projected token uses that account's identity and rotates through the same mounted-token contract as the API.
Worker RoleBindings grant it only the existing bounded worker-control permissions.
It receives no Secret read, Pod exec, or Pod log permission.

The process receives only the shared database reference and its registry writer references through environment variables.
Its configuration contains worker settings and registry reference names. It contains no OIDC settings or artifact-store credentials.
The workload mounts no artifact claim and exposes no Service, Ingress, or container port.
Its ingress NetworkPolicy denies all incoming traffic. Outgoing database, registry, token endpoint, and Kubernetes API access remain necessary.
The existing chart does not restrict trusted controller egress. Apply installation-specific egress policy where required.
Discovery Pods retain their separate restricted service account and exact worker egress policy.

The preparation container uses a read-only root filesystem, user 65532, dropped capabilities, and bounded temporary memory storage.
It has no HTTP health endpoint. API health probes do not apply to this background process.
The controller restarts an exited process. Queue leases and cleanup records provide durable recovery.
Process startup alone does not prove database connectivity, registry permissions, or successful source preparation.
Use worker diagnostics and preparation job progress to assess operation after deployment.

## Drain and recovery

Set preparation.acceptNewWork=false while keeping preparation.enabled=true.
After the API and preparation Pods roll out, new source staging and builds are disabled.
Prebuilt discovery also stops accepting new API requests. Approved test execution and existing source reviews remain available.
Already claimed jobs can finish or record interruption during rollout. Existing queued builds remain queued until enabled again or explicitly canceled.
Existing publication requests retain their normal behavior. This switch is not a publication lock.

The preparation workload retains its provider settings, writer credentials, database, private journals, and cleanup drivers.
It reconciles expired leases without repeating discovery or launching another Pod.
Ready reviews and failed observations remain saved before provider deletion.
Inspect pending jobs and cleanup status before disabling the preparation workload, removing credentials, or uninstalling worker resources.
Do not discard an uncertain registry upload journal. Follow [registry recovery](registry-preparation.md#ownership-and-recovery).
Do not replace the journal claim during an ordinary image update.

## Local verification

```sh
.data/kubernetes-tools/venv/bin/python scripts/verify-kubernetes.py \
  --helm helm --kubeconform kubeconform --preparation-only \
  --output output/kubernetes-preparation/package
```

Six renderings cover disabled preparation, both builders, Go-only uploads, cleanup-only operation, shared journals, private registry trust, and Bearer authentication.
Twelve invalid configurations reject missing workers, missing or mutable runtimes, foreign registry references, shared claims, and incorrect credentials.
Both rendered process configurations match the provider's protected admission policy.
Native and packaged startup checks cover the selected source profile and fail before SQLite initialization or provider contact.
Evidence remains in output/kubernetes-preparation. Owned startup probe containers are removed and their absence is verified.
The UI, private trace viewer, health binary, and source runtime images are reused.
No browser, registry push, cluster, AWS, or Azure operation runs.
