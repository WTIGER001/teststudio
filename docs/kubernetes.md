# Kubernetes service profile

The Helm chart in deploy/kubernetes packages the TestStudio service, identity settings, private evidence storage, and ingress configuration.
It uses the portable service image and installation-specific PostgreSQL and OIDC settings.
The current chart runs the API, UI, schedules, notifications, and native HTTP checks.
Hosted coded execution and preparation adapters remain required work. Kubernetes deployment acceptance remains open.

## Installation inputs

Select an authorized cluster context and a dedicated TestStudio namespace for deployment work.
Prepare these installation resources:

| Input | Configuration |
| --- | --- |
| Service image | Registry repository, immutable sha256 digest, and matching Linux architecture. |
| PostgreSQL | An existing namespace Secret with the dedicated TestStudio database connection. |
| Identity | Initial OIDC provider definitions and the exact issuer and subject for the first administrator. |
| Private OIDC clients | Existing Secret names and keys, mapped through approved TESTSTUDIO_OIDC_ environment references. |
| Public routes | Different HTTPS hosts for API/UI and private artifacts. |
| TLS | Existing certificate Secrets for those hosts. |
| Ingress | The installed ingress class and its actual traffic-source pod selectors. |
| Artifacts | A private persistent claim or the installation's selected object storage profile. |

The chart creates no database, identity provider, cloud account, or credential value.
It references existing Secrets through secretKeyRef. Raw database and OIDC secret values are rejected by the values schema.
The API and preparation command support --require-postgres. The chart enables it to reject a local SQLite fallback before opening storage.
The namespace Secret must contain a PostgreSQL URL. Missing Secret references prevent pod startup.

## Configure the chart

Copy deploy/kubernetes/values.example.yaml into a private installation directory.
Replace its reserved example hosts, image digest, Secret names, ingress settings, issuer, client identifier, and bootstrap subject.
Supply complete installation values on upgrades. Keep actual Secret contents in the installation's protected secret system.

The chart accepts several initial OIDC providers. Their identifiers must be unique.
The bootstrap issuer must match one configured provider. The subject names exactly one external identity.
Register each provider's callback as UI_ORIGIN/auth/callback/PROVIDER_ID.
Private clients use secretRef values such as env:TESTSTUDIO_OIDC_ORGANIZATION.
Every referenced environment name requires an explicit oidc.credentials entry with an existing Secret and key.
Mapped references are also approved for later provider management through the UI.
Public clients can omit secretRef or use an empty string.
Local password sign-in remains optional through localAccounts. Initial OIDC administration can create and manage those accounts afterward.
See [Identity providers](providers.md) and [Access](access.md) for managed sessions and authorization.

origins.ui and origins.artifacts contain different HTTPS hosts without paths, credentials, query strings, or custom ports.
The Ingress routes the UI/API host to the API Service and the evidence host to the artifact Service.
Their target ports remain 8040 and 8041. Each host selects its declared TLS Secret.
Configure the selected controller to preserve request paths, permit product upload sizes, and retain suitable request timeouts.
Keep authenticated API, sign-in, and artifact responses outside proxy caches.
Ingress behavior depends on the selected controller. See the [Kubernetes Ingress contract](https://kubernetes.io/docs/concepts/services-networking/ingress/).

## Storage and replicas

Filesystem artifacts use /data/artifacts on a private claim.
The chart creates a retained claim by default. persistence.existingClaim selects an already provisioned TestStudio claim.
A single replica with ReadWriteOnce storage uses Recreate updates. These updates have a service interruption.
Multiple filesystem replicas require declared ReadWriteMany storage. Verify that the actual claim and storage driver support shared access.
The claim remains after Helm release removal. Back up or remove retained private evidence through explicit operator action.
Installation export covers database metadata and references. Artifact bytes require their own backup and cutover copy.
See [Database transfer](data-transfer.md) and [Artifact storage](artifact-storage.md).

For object storage, replace artifactStore with the existing S3 or Azure configuration object.
The chart mounts temporary /data storage for those profiles and creates no artifact claim.
It accepts storage settings and identity references rather than access keys.
Configure provider authorization through the installation's selected workload identity integration.
serviceAccount.annotations, podAnnotations, and podLabels support those integrations.
Core service selectors and the configuration checksum remain protected.
Object storage and shared filesystem profiles use rolling updates. More than one replica also creates a disruption budget.
No claim is made about hosted failover or provider lifecycle until the corresponding deployment verification passes.

## Process and network configuration

The pod uses user and group 65532, a read-only root filesystem, no added capabilities, and private writable mounts.
The ServiceAccount has no automatic API token or chart-created RBAC grants.
Configuration changes update a pod-template checksum and trigger rollout.
Environment Secret rotations require a controlled pod restart after the new Secret value is available.

Startup and readiness probes use /healthz, which checks database availability.
Liveness uses /livez, which checks the service process independently of database readiness.
A temporary database failure removes readiness without causing liveness restarts.
Startup probes delay normal liveness and readiness processing until startup succeeds, as described in the [Kubernetes probe documentation](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/).

The default NetworkPolicy permits incoming API and artifact traffic only from the selected namespace and proxy pods.
Choose the traffic-source pods, rather than a controller that only manages an external load balancer.
Adapt the installation's policy for host-network ingress or externally sourced traffic.
The policy covers ingress. The API still needs approved database, identity, notification, and test-target connectivity.
NetworkPolicy enforcement requires a supporting network plugin. See [Kubernetes NetworkPolicy](https://kubernetes.io/docs/concepts/services-networking/network-policies/).
Untrusted coded workers need their own isolation and dispatch contracts. They do not run inside this service pod.

Native checks run at the kubernetes execution location.
Register endpoints that are reachable from that location. Desktop localhost and pod localhost refer to different machines.
The chart contains no example projects, copied test suites, or imported history.

## Local packaging checks

Local rendering and validation contact no cluster. No deployment command runs during the current implementation session.
The current checks use Helm 4.3.1, kubeconform 0.8.0, and PyYAML 6.0.3.
Install Helm and kubeconform from their verified upstream releases.
Use the local Python helper environment:

```sh
python3 -m venv .data/kubernetes-tools/venv
.data/kubernetes-tools/venv/bin/python -m pip install -r scripts/kubernetes-check-requirements.txt
.data/kubernetes-tools/venv/bin/python scripts/verify-kubernetes.py \
  --helm helm --kubeconform kubeconform --output output/kubernetes-package/check
```

The verifier disables kubeconfig input and clears inherited Helm cluster overrides.
It lints and renders the chart, validates strict resource schemas, and checks cross-resource routing, selectors, private references, storage, and probes.
Four rendering cases cover filesystem, shared filesystem, Azure settings, S3 settings, multiple providers, and replica constraints.
Nine invalid configurations fail before deployment. They include inline secrets, shared origins, missing mappings, and unsafe single-writer replicas.
The schema source is pinned in deploy/kubernetes/schema-source.json. Public schemas download into the local check cache on first use.
Helm rendering itself runs locally; it performs no server-side validation. See [Helm template](https://helm.sh/docs/helm/helm_template/).
The chart declares Kubernetes 1.25 or later. Current local schema validation uses version 1.37.0.
Admission policy, storage behavior, ingress traffic, identity callbacks, and provider execution still require later authorized deployment verification.

## Separate deployment work

Review the complete rendered resources and private installation values before deployment.
Use an explicit authorized context rather than an inherited default:

```sh
helm template teststudio deploy/kubernetes --namespace teststudio \
  --values PRIVATE_VALUES_FILE > REVIEWED_MANIFEST_FILE
helm upgrade --install teststudio deploy/kubernetes \
  --kube-context TESTSTUDIO_CONTEXT --namespace teststudio --create-namespace \
  --values PRIVATE_VALUES_FILE
```

The deployment command is documented for the separate deployment phase. It is not run by the local verifier.
Keep the full acceptance scope in [Implementation verification](implementation.md), including hosted workers, recovery, identity, and private artifact access.
