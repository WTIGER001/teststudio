# Local Kubernetes source execution

## Introduction

Fresh Go and Playwright suites now pass through the actual local Kubernetes installation.
The workflow uses normal OIDC sign-in, ZIP upload, registry assembly, isolated discovery, reviewed publication, execution, timing, and private evidence.
It copies no existing application tests, data, or infrastructure configuration.
Cancellation, test timeouts, and interrupted-controller recovery now pass actual local verification.
Independent Pod deadline enforcement also passes. Controller peer failover now passes through a separate recorded case. See [local lifecycle](local-kubernetes-lifecycle.md).
The overall acceptance count is 49 of 57. Azure implementation remains deferred.

## Concepts

### Current installation

Helm release teststudio now uses revision 9 and .data/local-k3s/controller-values.json.
The API execution worker is disabled and two separate controllers share retained evidence through NFS.
The API and preparation workload run separately against the dedicated PostgreSQL database.
Preparation uses its own service account, projected Kubernetes identity, registry writer references, and retained private journal claim.
The API receives only the registry reader references.
The source-v2 supervised Go and Playwright runtimes use immutable registry distribution digests.
The actual runtime references are in output/local-k3s/published-runtimes-v2.json.

The local registry runs Distribution 3.1.0 in namespace teststudio-registry with a separate 2 GiB persistent claim.
Its test-only TLS proxy restricts repository paths and rejects writes from the reader identity.
The Distribution listener binds only to loopback inside that Pod. The Service exposes only the TLS proxy.
Its ingress policy admits the API and preparation workloads. Registry outbound traffic is denied.
This is a development fixture, not a production registry installation.
The official [Distribution configuration](https://github.com/distribution/distribution/blob/main/docs/content/about/configuration.md) describes its filesystem and bcrypt authentication settings.

The reader and writer use separate Kubernetes Secrets in the application namespace.
The worker namespace has a separate image-pull Secret for the kubelet.
The untrusted worker receives no registry credential, Kubernetes API token, database credential, or TLS private key.
The approved runtimes contain only the public local exchange authority alongside normal system certificate trust.
The kubelet uses its own registry CA configuration and image-pull identity outside the worker container.

The owned node resolves registry.teststudio.test through its local hosts file.
The cluster uses its owned CoreDNS configuration. Neither configuration changes host DNS or another cluster.
Publishing uses the checksum-verified crane v0.21.5 tool in a temporary private node directory.
The registry credential enters through its protected configuration file, never through a process argument.
The temporary publisher, credential copies, and node archives are removed after publication.
See the official [crane installation guide](https://github.com/google/go-containerregistry/blob/main/cmd/crane/README.md).

### Verified source workflows

The new sources live in tests/kubernetes/suites/go and tests/kubernetes/suites/playwright.
The Go suite round-trips a temporary JSON record and verifies its non-root worker identity.
The browser suite renders an in-memory result card, clicks Review result, verifies Reviewed, and captures an image.
Both suites are targetless. They require no external website, browser gateway, dependency download, or application credential.
Their catalog identifiers, manifest, module metadata, tests, and browser dependency declarations are newly authored.

The Go preparation identifier is 952334ce21b07348ef57c48febf69a75ea5e0c524f5764d669f56177548e4be1.
Its run is 20261105T151433.206691439-698ad6db3525513bd828fd49.
It passes with a 0.604332 ms logical test duration, seven retained artifacts, and seven log segments.
Its private evidence download passes the stored size and SHA-256 checks.

The Playwright preparation identifier is 0b2bb4e42e4dfa7d02c1a840585c86f77c39a4b8d11afa2b3c8683bc871a3654.
Its run is 20261105T151447.417956930-cfc4c366f3c9ee354dd21551.
It passes with a 263 ms logical test duration, nine retained artifacts, and six log segments.
Screenshots and a trace remain available. A private PNG download passes its size and SHA-256 checks.
Visual inspection confirms the rendered card shows Reviewed.

Both runs have complete evidence and cleaned execution records. Both preparation records have no pending cleanup.
The worker namespace contains no remaining Pods. CRI reports no running worker containers.
No host browser or automation daemon starts. The user's existing demo remains at http://127.0.0.1:8040.

### Integration fixes

Run creation previously rejected every coded runner outside the local location.
It now admits a matching configured runner and execution location while rejecting a driver for another location.

The first Go run passed but reported zero duration because the framework rounded its elapsed value.
The source-v2 runner preserves a valid selected-test timestamp interval when that value rounds to zero.
Positive framework durations and genuine zero intervals remain unchanged. Build time stays separate.

The first browser run stopped before its test because Playwright tried to create test-results inside the read-only source directory.
The trusted source-v2 configuration now puts framework output under /output/playwright-results and applies the frozen capture policy across projects.
Uploaded tests need no directory or capture workaround. Restricted environment profiles retain their existing gateway rules.
Relative setup, teardown, server working directories, snapshot directories, and TypeScript configuration retain their original suite location.
A focused configuration check covers these paths. The current preparation runtime includes this compatibility correction.
The retained successful browser preparation keeps its original immutable runtime reference.

The earlier Go observation and incomplete browser run remain in retained history.
The corrected runtimes have new identities, so their timing baselines remain separate.
The verifier reuses checkpoints and does not run calibration loops to populate history artificially.
Targeted Go admission and event checks, four configuration checks, and the affected actual workflows provide verification.
No broad UI or browser suite runs in this phase.

### Evidence

The combined result is output/local-k3s/coded/summary.json.
Adjacent files retain each run, preparation, execution ledger, logs, artifacts, and private download.
The inspected PNG is playwright-result-card.png. Browser and worker shutdown evidence is browser-cleanup.json.
The old observations are preserved in the files with v1 in their names.
Registry setup and authentication results are registry-installation.json and registry-auth-check.json in output/local-k3s.
Private manifests, credentials, trust configuration, and TLS keys remain under .data/local-k3s with restricted permissions.
Do not commit or print private configuration files.

## Tutorials

### Reuse the recorded workflows

Inspect the saved summary and references before repeating verification:

```sh
python3 scripts/verify-kubernetes-suites.py --runner=go
python3 scripts/verify-kubernetes-suites.py --runner=playwright
```

The default checkpoint is output/local-k3s/coded/references.json.
Existing preparation, release, and run identifiers are reused. Successful invocations inspect those records and download evidence without executing tests again.
The shared local API helper follows normal OIDC redirects, keeps cookies in memory, and revokes its session afterward.
An incomplete run is terminal for this verifier. Its diagnostics are retained before the assertion fails.
An observation timeout does not authorize a replacement preparation or run. Inspect the existing identifier first.

### Maintain the local registry

An imported multi-platform archive can contain unavailable manifest children. k3d can log that error while returning a successful command status.
The verified local import uses ctr images import --platform linux/arm64 and inspects the actual imported image afterward.
The initial ctr publication attempt opened sessions without transferring content and was explicitly interrupted after its live process was confirmed.
The standard crane publisher completed both runtime publications. Native source assembly then wrote and verified each prepared image through the normal registry adapter.
Unfinished operator upload sessions require the registry's upload maintenance. They are separate from the successfully cleaned product preparation journals.
Retain registry storage while published releases or retained observations require those images.

### Continue provider lifecycle verification

Cancellation, timeout, and interrupted-controller cases now have saved references in output/local-k3s/lifecycle.
Independent deadline evidence now lives in output/local-k3s/deadline.
Separate controllers now pass peer failover. See [local controller failover](local-kubernetes-controllers.md).
Continue with the second installation-specific OIDC provider without repeating those passing cases.
Keep all worker execution on the dedicated local cluster and retain actual provider status and cleanup evidence.
Environment-backed remote gateway delivery and cloud-target verification remain open.

Related guides: [worker isolation](local-kubernetes-isolation.md), [preparation package](kubernetes-preparation.md), and [implementation ledger](implementation.md).
