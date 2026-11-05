# TestStudio implementation plan

Status: Resumed at the user's request on 2026-11-03. Implementation remains authorized.
Track verified delivery in `implementation.md`.
The user authorizes targeted tests and frequent feature commits. Remote pushes remain separate operations.

## Current execution direction

The user resumes implementation on 2026-11-03.
Focus on implementation, then targeted tests for the changed behavior.
Run tests only locally. AWS, Kubernetes, and Azure verification are separate work.
Do not repeat broad test suites or fixture builds without a specific unresolved failure or changed behavior.
The full product scope remains required. This direction supersedes the testing sequence in the historical checkpoint below.

## Active delivery plan after the tester walkthrough

This section is the current implementation order. Historical continuation commands below do not override it.
The local demonstration is usable and verified. The complete product remains unfinished at 48 verified items out of 57.
Ordinary project creator access and the first environment-backed local recorded website flow pass targeted verification.
Tester investigation, failure recovery, and local manual/scheduled runner parity pass targeted verification.
Continue source-workflow, access, storage, and hosted packaging gaps. Reuse passing runner evidence.
Do not add new dashboard pages or repeat completed demonstration setup before closing the remaining workflow gaps.

### Current decisions

| Area | Decision and current status |
| --- | --- |
| Repository | Keep the Go service, Angular UI, adapters, and product-owned tests in the independent TestStudio repository. No second product repository is needed. |
| Runtime boundaries | Keep API, preparation, recording, and execution responsibilities separate. Uploaded test code never runs inside the API process. |
| Ownership model | Projects own applications and suites. Environments belong to applications. Suites link to applications and contain tests. Keep this model visible in the UI. |
| Onboarding | Use guided setup and direct project membership. Support local users, OIDC users, and mixed sign-in. Invitations are not required. |
| Test creation | Use a dedicated type chooser. Browser tests begin with recording. Quick HTTP checks accept a complete URL with an optional port. |
| New target configuration | Show new application and environment records on Review before saving. Reuse matching targets and preserve their protection. |
| Test lifecycle | Keep recording, draft saving, preparation, publication, activation, and execution distinct. Combined actions explain each effect. |
| Save and run | Retain the implemented non-activating publication behavior. Run the selected test and dependencies without replacing the suite's active release. |
| Persistence | SQLite is the single-instance local default. PostgreSQL is the hosted and multi-instance default. Installation export and import preserve identifiers and access across both databases. Shared storage parity is verified under 06. Hosted profile acceptance remains separate. |
| Charts | Keep the existing accessible SVG stacked-area implementation. A chart-library migration is not required. |
| Local verification | Use local fixtures, cached builds, and targeted checks. Public customer websites are manual user targets, not verification dependencies. |
| Deployment | Retain local server, AWS ECS with CloudFront, Kubernetes, and Azure Container Apps as required profiles. Develop isolated sandbox execution for Azure. Hosted parity remains unverified. |

### Azure isolated execution direction

Continue Azure coded execution through Container Apps Sandboxes. Keep the UI and API in Container Apps.
The public sandbox contract supplies a separate execution boundary and create-time outbound controls.
The service and selected API remain preview. Complete deployment acceptance remains required.
This uses the existing plan's suitable-backend alternative. It does not add a deployment target or change the 57 acceptance numbers.
Ordinary Jobs remain disabled. Retain their request and recovery contracts as reference material.
Do not continue adding Job network inspectors as a substitute for an execution isolation boundary.

The new Go client now supports digest-based image import, image inventory, sandbox creation, status, stop, deletion, and bounded sandbox inventory.
Launches use the approved source inspector result, fixed supervisor, private VNet connection, and Full default-deny outbound inspection.
Only the exact private exchange host is allowed. Runtime environments contain only attempt-scoped worker credentials.
Recovery proofs and returned snapshots retain hashes without usable tickets. Restored proofs cannot create another execution.
Local protocol and ownership checks pass for both runners and discovery. No native configuration or deployment enables this client yet.

The durable sandbox controller now reserves one create call and reconciles lost receipts against complete inventory.
It freezes the exact exchange input hash and canonical source digest, reauthorizes initial dispatch, and asserts current exchange revisions.
Cancellation fences initial creation. Checkpointed cleanup retains uncertain creates, confirms quiescence and deletion, and removes deadlines last.
Local restart, twelve-replica contention, in-flight cancellation, input binding, and cleanup checks pass.
The controller returns resource facts. It does not turn a stopped instance or accepted report into independent command completion.

The concrete sandbox deadline SDK now reserves one independent Logic Apps timer per execution and verifies its exact owned definition.
It finds matching owned instances through the data-plane API, including late creation without a receipt.
Known timer rejection remains distinct from uncertainty. Confirmed instance absence and retained result checkpoints control timer removal.
Local SDK, database-reopen, cleanup, public-schema, and static-reference checks pass. Actual workflow execution and authority remain unverified.

Independent command completion now uses a trusted idle launch, fixed supervised invocation, and the service's explicit exit result.
The v3 proof binds a versioned image capability. Linux modes enforce UID and GID 1000, no new privileges, empty capabilities, and protected process state.
Atomic command reservation asserts active exchange work and fresh authorization. Restart never replays a reserved command.
Controller.Status requires both a valid exit receipt and quiescence before successful native completion.
Cancellation suppresses late success and checks resource ownership before interrupting a connection.
Targeted protocol, restart, scope, cancellation, OCI, and supervisor checks pass. Local Linux identity probes pass and their containers are removed.

Durable image import and converted-image lookup now use the shared database and gated discovery authorization.
One reservation controls each import. Lost acknowledgments reconcile complete inventory without replay.
Fresh image reads verify immutable source, supervisor, labels, and ownership before launch.
Only a new authorized discovery request can replace a known failed generation. Retained history prevents reuse of earlier requests.
Lookup processes cannot initiate conversion. Targeted local restart, contention, authorization, history, and protocol checks pass.
See [durable sandbox images](azure-sandbox-images.md). The verified count remains 48 of 57.

The sandbox Provider now connects pending image preparation, controller status, command execution, cancellation, cleanup, and recovery inventory.
Launch intents retain pending conversions before a sandbox exists. Their proofs freeze source, import identity, work input, and the original deadline.
The actual shared executor and worker exchange pass local protocol flows for both runners with private logs, reported timing, and completed cleanup.
Scope, restart, late creation, lost receipts, and concurrent admission updates pass targeted checks.
See [sandbox provider integration](azure-sandbox-provider.md). This does not establish deployed Azure execution.

Independent sandbox settings and OpenRuntime now assemble the concrete ACR, data-plane, timer, image, and provider clients.
Construction selects explicit control and read-only registry client identifiers without token acquisition or service calls.
Native execution and discovery modes reject control of the opposite workload. Closing the runtime prevents new provider operations.
Local settings, credential, client, mode, shutdown, and affected shared lifecycle checks pass.
See [sandbox runtime assembly](azure-sandbox-runtime.md). Commands do not load these settings yet.

The runtime now includes a concrete ARM placement and import-grant stage before its remaining protection dependencies.
Seven selected reads verify group, VNet attachment, subnet bindings, import identity, registry settings, and the reported pull grant.
Placement and protected-policy fingerprints bind native launch and source-cache profiles. The metadata reader uses a third explicit identity.
Initial and import checks stay fresh. Live placement observations have a 15-second positive cache and a three-resource shared refresh.
Sandbox-definition, permission, revocation, and additional protection checks remain uncached.
Targeted local SDK, drift, grant, cancellation, bounds, caching, and affected lifecycle checks pass.
See [sandbox placement admission](azure-sandbox-placement.md). Reported metadata does not prove complete effective authority or private routing.

The runtime now includes a selected-authority stage for control, registry inspection, metadata, and independent timer identities.
Fifteen selected reads compare four identities, five custom role definitions, and six exact assignments.
Execution and discovery control profiles differ. The timer identity receives read and stop data actions only.
Workflow administration stays in the separate timer group. Identity assignment targets only the selected timer identity.
Canonical definitions provide reviewable packaging inputs without changing Azure resources.
Authority observations share bounded positive metadata evidence. Domain authorization and remaining protection callbacks stay uncached.
Targeted local profiles, drift, scope, identity separation, cancellation, caching, composition, and affected lifecycle checks pass.
See [sandbox selected authority](azure-sandbox-authority.md). This does not establish complete effective access or host attachment.

The runtime now verifies the selected security group's complete custom rules and the selected route table's contents.
The sandbox connection policy permits only platform DNS and HTTPS to explicit private endpoint addresses, with metadata and remaining traffic denied.
It includes no ordinary Jobs platform allow rules. Actual preview-service compatibility remains deployment work.
The route table disables BGP propagation and contains only the selected default route through a private firewall address.
The placement profile binds those addresses. Canonical resource bodies support later packaging without mutation.
Initial and import checks stay fresh. Live observations share bounded positive evidence, and current failures invalidate prior success.
Targeted local policy, drift, settings, caching, runtime integration, and affected lifecycle checks pass.
See [sandbox connection rules and routes](azure-sandbox-network.md). This does not establish effective routing or endpoint ownership.

The runtime now binds the allowed addresses to two selected private endpoints and their three exact NIC IP configurations.
The registry login and regional data members target the selected registry. The exchange member targets the selected relay environment.
The endpoint subnet has an exclusive endpoint inventory and a security group permitting HTTPS only from the execution subnet.
Two selected private zones have complete bounded record and link inventories, exact A records, and no wildcard, alias, registration, or public fallback.
The placement profile binds endpoint, subnet, service, address, and zone settings.
Fresh initial and import observations and bounded live evidence now include this private stage.
Existing pure endpoint and DNS verifiers are reused without ordinary Jobs configuration.
Targeted local drift, paging, settings, caching, runtime integration, and affected lifecycle checks pass.
See [sandbox private endpoint and DNS admission](azure-sandbox-private.md). Reported records do not establish effective resolution or service ownership.

The runtime now checks the selected VNet's address space and exact embedded execution, endpoint, and firewall subnets.
Peerings, custom DNS servers, extra subnets, public NAT, and mismatched embedded subnet contents reject.
The checked firewall address binds the default route to the selected firewall, policy, subnet, and public IP attachment.
The complete policy inventory permits one fixed deny-all network group and only empty known service default groups.
No Jobs platform allow rules enter this sandbox policy. Existing resource and policy verifiers are reused through independent bindings.
The placement profile binds the added topology settings. Five resource bodies support later packaging without mutation.
Fresh initial and import checks and bounded live evidence now include topology and firewall contents.
Six new local checks and affected native and shared policy checks pass with race detection.
See [sandbox VNet and firewall admission](azure-sandbox-topology.md). Reported configuration does not prove deployed routing or traffic enforcement.

The runtime now verifies selected resolver policy, complete rules and links, and the actual VNet policy association.
Five approved exchange and registry domains and a root-domain block list use two fixed enabled rules.
Azure domain hierarchy includes descendants. Keep selected private zones, fixed records, no wildcard or alias, and no public fallback independently required.
One exact empty POST read action observes the actual association. The metadata transport otherwise remains scoped to selected GET paths.
Reported subscription inventory now checks every returned private zone and its links, rejecting additional links to the sandbox VNet.
Unrelated zones and VNets remain permitted. Inventory limits, missing pages, count gaps, and failures retain uncertainty.
The placement profile binds resolver settings. Six reviewable resource bodies support later packaging without mutation.
Fresh initial and import observations and bounded live evidence include these DNS checks.
Six new local checks and affected native, shared DNS, and inventory checks pass with race detection.
See [sandbox resolver admission](azure-sandbox-resolver.md). This does not prove effective resolution, visibility across subscriptions, or absence of deny-hidden resources.

The runtime now verifies the selected relay and trusted backend applications, private environments, authentication configuration, and complete revisions.
The relay's worker-facing endpoint approval and embedded connection inventory must agree. Certificate metadata binds its custom exchange hostname.
Fixed templates retain relay-only and worker-backend commands, immutable trusted images, probes, resources, origins, database references, and Blob configuration.
Four distinct identity pins retain pull and startup lifecycle None, with only the backend runtime identity exposed to main containers.
The service identities cannot reuse the import, timer, control, registry inspection, or metadata identity bindings.
Exactly one active healthy revision must match each trusted template. Inactive revisions must have zero replicas and traffic.
The sandbox-placement/v6 profile binds service images, identities, origins, and dependencies. Four bodies support later packaging without mutation.
Fresh initial and import checks and bounded live evidence now include these service observations.
Six new local checks and affected native and shared service checks pass with race detection.
See [trusted service admission](azure-sandbox-services.md). Metadata checks do not establish backend routing, readiness, dependency access, or effective permissions.

Complete remaining effective-authority and protected network admission, native command role wiring, and deployment packaging next.
Connect native relay-to-backend endpoint ownership and end-to-end readiness next.
Verify workload placement, identity host attachment, additional grants and deny effects, and protected write boundaries.
Keep cross-subscription link visibility and actual DNS hierarchy behavior explicit in separate deployment acceptance.
Verify actual service acceptance of custom data-action roles and narrow timer grants during the separate deployment phase.
Keep private registry conversion access and proxy trust as explicit deployment compatibility questions.
Pin the group reader and template to an explicit service contract. Current preview guidance and the versioned ARM reference describe different shapes.
Do not silently combine the pinned preview SDK with the 2026-07-01 environment-linked resource shape.
Reuse the new command and status bridge. Preserve explicit gaps after a lost acknowledgment and never replay the logical attempt.
Preserve source-queue authorization and the existing worker exchange. Keep controller credentials outside sandboxes.
Bind actual source-import identity, group authority, VNet routing, proxy trust, and effective outbound enforcement before enabling native execution.
Confirm the pinned SDK token audience during the separate deployment verification phase.
Idle suspension and accepted stop responses cannot replace the absolute deadline contract.
See [Azure isolated sandbox execution](azure-sandboxes.md) and [durable sandbox control](azure-sandbox-controller.md).
See [independent sandbox deadlines](azure-sandbox-deadlines.md) for timer construction, verification, cleanup, and remaining deployment evidence.
See [supervised sandbox commands](azure-sandbox-commands.md) for command reservation, identity enforcement, completion, and remaining native integration.
Acceptance 51 and supporting recovery under 52 advance; the total remains 48 verified of 57.

### Active application navigation correction

Prioritize the tester's reported scope confusion before further hosted adapter work.
Use one project selector, one application selector, and a persistent project and application context band.
Place Projects before those selectors. Keep account access in the global header and omit the ambiguous Workspace section.
The demo serves a private UI snapshot. After interface changes, restart it and verify the served page before reporting delivery.
Keep its database and evidence. A source build alone does not refresh an already running demo.
Application testing contains Overview, Suites, Test catalog, Test runs, and Trends with matching data filters.
Project overviews provides All applications and HTTPS probes. Keep project administration visibly separate.
Use explicit application routes. Preserve context in creation and investigation links, reload, and browser history.
Shared suites retain project ownership and one release. Explain complete-suite operations beside application filters.
Freeze application context in new runs. Keep unknown legacy context in the project-wide view.
The focused local navigation workflow and targeted scope checks now pass. See the implementation ledger for the retained run.
Keep the stable 57 acceptance numbers and the 48 verified count unchanged.

### Next implementation batches

Complete one coherent batch before adding the next. Update its user help and relevant technical guides with the behavior.

| Priority | Batch | Completion evidence |
| --- | --- | --- |
| Complete | Project creator access | Ordinary creator setup, managed scoped roles, publication, local execution, atomic rollback, and project boundaries pass on SQLite and local PostgreSQL. Acceptance 13 is verified. |
| Complete | Recorded website execution locally | New native recording, reviewed local website steps, saved preparation, publication, registered target execution, guarded direct request, and blocked off-site redirect pass with retained logs, screenshot, trace, and timing. |
| Complete | Tester operation and failure recovery | Scoped attempts, logs, screenshots, timeout and retry inspection, permission-loss recovery, and retry after a lost rerun response without another child pass. Acceptance 44 is verified. |
| 4 | Remaining catalog and evidence gaps | Private trace viewing and category browsing are implemented. Reviewed publication also passes local verification. Catalog workflows and the shared paged API now pass verification. Preparation safety and retained history now close 26 and 29. Evidence restart recovery closes 43. S3 and Azure Blob adapters now have operator selection and local protocol checks. Keep 46 open for provider verification. Local coded timing and logical counts now pass focused checks. Manual and scheduled parity closes 37 across implemented local runners. Keep 41 open across deployment profiles. Continue provider packaging and remaining acceptance work against the ledger. |
| 5 | Hosted packaging and provider adapters | Implement and document independent provider configuration, worker dispatch, private artifacts, cancellation, deadlines, and recovery. Use local contract checks first. |
| 6 | Deployment acceptance | Verify AWS, Kubernetes, and Azure in their separate deployment work. Then close the applicable remaining ledger items with profile-specific evidence. |

The creator-access batch fixes the missing initial assignments in project creation.
Ordinary creators receive the existing administrator, editor, publisher, and operator roles bound only to their new project.
The project and initial assignments commit atomically with audit records and the directory revision.
The administrator role itself retains its existing responsibilities. Full installation administrators keep existing authority without automatic project self-assignment.
Disabled required roles prevent partial creation. Explicit denies remain effective. Project edits do not restore revoked roles.
Existing projects receive no automatic access changes during upgrade.
See [initial creator access](access.md#initial-project-creator-access) for current API behavior and role policy.
SQLite, local PostgreSQL, and STUDIO-OWNER-001 evidence are recorded in the implementation ledger.

This sequence changes implementation priority, not the scope or numbering of the 57 acceptance criteria.
Acceptances 03, 08, 13, 20, 21, 25, 26, 28, 29, 34, 36, 38, 42, 43, 44, and 45 are complete. Acceptance 12 now passes the focused grant and revocation audit. Remaining requirements stay open under their stable ledger numbers.
Do not treat these batches as additional acceptance items or claim the whole product complete after a demonstration.

### Current continuation: access, storage, and provider work

Acceptances **02**, **06**, **12**, **22**, and **37** are complete. The count is **48 verified, 9 open**.
The repository editor uses Repository, Suites, and Review through the canonical wizard components.
Identifiers generate from the name. Uncommon reference and synchronization settings stay under named Advanced sections.
Review keeps enabled state, automatic publication, future activation, and test-execution consequences visible.
Inspected repository preparation updates automatically and provides an explicit refresh action.

The complete local GitHub-to-release UI journey passes against a local TLS provider fixture.
The production adapter fetches one pinned archive. Real Go preparation, browser review, publication, retained source, and explicit release execution pass.
Its authoring browser runs through normal TestStudio publication and guarded environment execution.
The check finds and fixes direct crypto.randomUUID calls on ordinary HTTP origins.
The shared randomId helper preserves secure UUID generation with a secure-random-byte fallback.
The Angular build and targeted identifier checks pass. Existing native-upload and ZIP UI evidence remains valid without repetition.
See the implementation ledger for exact runs and verified cleanup.

Acceptance 12 now passes its source audit and nine focused existing checks on SQLite and local PostgreSQL with race detection.
User, group, role, object, and project grants preserve denies, expiry, inheritance, exact revocation, and administrative boundaries.
No authorization behavior change, browser test, UI build, or runner image build is needed.

Acceptance 06 now passes the same eleven focused storage, ledger, and scheduler checks on SQLite and local PostgreSQL with race detection.
Atomic writes, revision conflicts, cursor reopen, concurrent identity, worker checkpoints, and claim recovery share verified behavior.
The owned local database container is removed. No UI rebuild, browser run, or runner-image build is needed.

Acceptance 02 now passes the distribution provenance audit against nine available legacy checkouts and the checked-in ZIP members.
Repository origin, authored fixtures, independent example configuration, and retained isolated-build and empty-start evidence support the comparison.
The audit tool records its exact scope and limits. It imports no reference material.

Core provider packaging is implemented locally. Continue hosted execution adapters for AWS, Kubernetes, and Azure through independent provider configuration.
Keep artifact storage, secrets, worker dispatch, status, cancellation, deadlines, and recovery within common product contracts.
Use local provider protocol fixtures and packaging checks only. Do not run cloud deployments in this work session.
Reuse completed local product evidence and build only changed inputs.
Keep deployment-wide timing **41**, storage-provider retention **46**, and hosted criteria **51–57** open until their required verification passes.
Do not repeat completed source, native, API, browser, or scheduled-runner fixtures without a concrete contract change or unresolved failure.
Cloud execution and storage-provider verification remain separate work.

The ECS network admission checker now reads exact VPC, routing, group, endpoint, PrivateLink, DNS, and task-interface state.
Its bounded settings loader binds network identity into timing and recovery ownership.
Use a dedicated worker VPC; VPC-wide DNS restrictions must not affect trusted controller traffic.
No Route 53 Profile overlay is permitted in that worker VPC. See [network admission](ecs-network-admission.md).
The corrected Fargate contract uses task-local volumes, an explicit 21 GiB allocation, and inspected non-root storage capability.
Both thin overlays and source metadata preservation pass focused local checks.
Scoped SDK construction, account verification, rotating ECR authentication, and optional command wiring now pass local checks.
Package the private worker network and HTTPS route afterward. Preserve the separate cloud verification phase.

The optional ECS commands now require PostgreSQL, the ecs location, a private port-8042 listener, and separate public origins.
They reject mixed providers, local runners, ambiguous JSON sources, and artifact-role execution before database creation.
The SDK factory verifies the configured calling account and fixes regional service endpoints, retries, and HTTP limits.
ECR authorization stays in trusted processes, refreshes with credential identity, and strips credentials from expected S3 blob redirects.
Native startup guards and focused SDK protocol race checks pass without credential discovery or provider calls.
The shared hosted wiring keeps execution, discovery, approval, and cleanup contracts for both ECS and Kubernetes.
See [ECS workers](ecs-workers.md) for complete settings, commands, roles, local evidence, and current deployment limits.
The staged AWS worker package now supplies the dedicated VPC, DNS lists, endpoints, PrivateLink, internal TLS listener, scoped roles, and separate preparation.
Generated private settings connect to the optional API worker listener. Private EFS journals retain source upload recovery state.
An explicit endpoint-to-worker HTTPS rule suppresses CloudFormation default egress while worker ingress remains empty.
Nine local profile previews, eleven rejected configurations, production admission alignment, and three changed-image probes pass.
The ARM64 service image rebuild reuses the UI and trace assets. Trusted source runtime bases stay unchanged.
See [AWS worker packaging](aws-workers-package.md) for ordered prepare, connect, and ready stages, DNS operator steps, and evidence.
Local rollout admission now preserves known backend startup and draining transitions while requiring at least one owned healthy target.
A single bounded EC2 request checks up to 32 backend interfaces. Existing retiring interfaces retain all ownership checks.
Exact cross-zone, health-threshold, protocol, and connection-draining attributes match the staged template and controller read policy.
Focused race checks, generated-package admission, shared admission-loss behavior, and native entrypoint compilation pass.
One package invocation uses the wrong relative fixture path; its corrected absolute path passes.
Older installations need a complete drain before applying the new attributes, read authority, and freshly built service image.
This verifies local contracts, not live API availability or uninterrupted requests during rotation.
Azure now has a pinned SDK job contract for Go and Playwright execution and discovery.
It fixes manual single-replica execution, zero platform retries, private pull-only identity, exact source images, resources, and supervisor commands.
The job definition and recovery proof contain no usable ticket. Restored proofs cannot start another execution.
Local checks cover writable drift, execution receipts, optional list metadata, provider states, and scoped SDK create, start, and stop requests.
Provider completion remains separate from numeric process exit codes, which the Azure execution model does not expose.
See [Azure job contracts](azure-worker-contract.md). Acceptance 51 advances; the total remains 48 of 57.

The Azure provider now reserves Create and Start durably, polls provisioning, reconciles lost responses, and retains exact receipts and terminal history.
Fresh exchange authorization controls Start after provisioning. Recovery proofs alone still cannot launch another execution.
Shared native completion preserves validated reports and measured timing without inventing a numeric process exit code.
Owned cancellation works even when provider status is unknown. Checkpointed cleanup retains missing-history uncertainty and bounded deletion backoff.
The scoped SDK bridge disables automatic retries and registration, rejects redirects, and binds pagination to the exact parent job.
Local race, database-reopen, concurrency, protocol, authorization, cancellation, and shared-lifecycle checks pass.
Both native entrypoint packages compile. No cloud, UI, image, browser, or source-runtime matrix verification runs.
See [Durable Azure lifecycle](azure-worker-lifecycle.md). Acceptance 51 and supporting recovery under 52 advance; the count stays 48 of 57.

Trusted ACR authentication now uses an explicit Azure credential, the registry-specific audience, and exact repository scopes.
The same OCI client retains challenge validation, immutable metadata checks, source publication, and upload journals.
Inspection and preparation use separate read and write modes. Expiring in-memory caches bind to source identity and bounded token lifetime.
Local race checks pass for both runners, publication, identity rotation, malformed responses, foreign challenges, and credential-free blob redirects.
Four selected existing OCI checks pass. No image, runtime, UI, browser, real registry, or cloud test runs.
See [Azure registry authentication](azure-registry-authentication.md). Acceptance 51 advances; the count remains 48 of 57.

The concrete Azure deadline hook now provisions a separate Logic Apps timer for each owned job.
It binds the absolute due time, exact stop target, full specification, and a distinct stop identity.
Durable creation and deletion reservations preserve lost-response uncertainty, restart recovery, and cleanup ordering.
Targeted local race checks cover both runners, discovery, concurrency, changed resources, scoped SDK requests, and absence confirmation.
Azure service scheduling and shutdown latency remain unverified. A scheduled stop request does not imply exact-second termination.
See [independent Azure deadlines](azure-worker-deadlines.md). Acceptances 51 and 52 advance; the count remains 48 of 57.

The Azure topology stage now inspects actual environments, VNets, and subnets through a scoped read-only SDK client.
It rejects shared controller networking, public worker ingress, unsafe address spaces, unexpected delegation, NAT, peering, and attachment drift.
Fresh bounded reads preserve provisioning and resource uncertainty. It does not implement complete Network admission or release worker input.
Seven new focused race checks and two selected existing SDK checks pass without Azure connections.
See [Azure worker network admission](azure-worker-network.md). Acceptance 51 advances; the count remains 48 of 57.

The Azure perimeter stage now checks the complete fixed NSG policy and one default route through the selected firewall address.
It refreshes topology first, requires connection reevaluation and disabled BGP propagation, and rejects rule, route, attachment, and ownership drift.
Validated immutable settings also produce SDK definitions for later packaging. The production reader permits only the two selected GET resources.
Eight new focused race checks, a 24-case declared traffic matrix, and two selected topology SDK checks pass locally.
This stage does not implement complete Network admission. Platform compatibility, service-tag membership, and effective packet isolation remain unverified.
See [Azure worker perimeter](azure-worker-perimeter.md). Acceptance 51 advances; the count remains 48 of 57.

The Azure firewall stage now binds the routed private IP to the selected firewall, subnet, public IP, and dedicated policy.
It rejects classic and inherited rules, populated default groups, NAT, proxies, changed application controls, and substituted resource ownership.
A fixed platform HTTPS allowlist precedes an explicit application deny that overrides the built-in infrastructure fallback.
Complete policy group references and bounded list inventory must agree. Safe scoped paging preserves hidden-resource and incomplete-inventory uncertainty.
Eight new focused race checks and two selected existing transport checks pass locally. Independent resource reads run concurrently after perimeter inspection.
See [Azure worker firewall inspection](azure-worker-firewall.md). Acceptance 51 advances; the count remains 48 of 57.

The Azure DNS stage now verifies the actual VNet association, exclusive worker link, complete rule inventory, approved names, and root-domain block.
It rejects unexpected rules, domain lists, actions, disabled blocks, bulk URLs, foreign associations, and incomplete policy state.
Explicit registry data region avoids assuming that the registry and worker share a region.
The SDK permits one exact read-only POST association action. Shared scoped paging preserves existing firewall and read-transport boundaries.
Seven new focused race checks, a nineteen-case declaration matrix, and three selected existing checks pass locally.
Native DNS hierarchy remains explicit. An approved parent can allow descendants; this is not an exact-host-only DNS guarantee.
See [Azure worker DNS policy inspection](azure-worker-dns.md). Acceptance 51 advances; the count remains 48 of 57.

The selected private DNS stage now verifies the exchange hostname zone, registry private zone, complete records, and completed worker-only links.
It binds three distinct A-record roles to the complete perimeter address allowlist and explicit registry data region.
Aliases, wildcards, IPv6, extra records, automatic registration, public fallback, and unsettled counts cannot establish admission.
The reader permits only selected-zone GET resources and complete bounded inventories. Six new focused race checks and two selected paging checks pass locally.
See [Azure worker private DNS records](azure-worker-private-dns.md). Acceptance 51 advances; the count remains 48 of 57.

The endpoint stage now binds actual registry and exchange endpoints, consumer-side approval, NIC ownership, subnet policy, and three role addresses.
Its registry NIC members distinguish login from selected regional data. A dedicated endpoint NSG restricts worker access to HTTPS.
The scoped SDK reader performs only six selected-resource GET requests. Seven new focused race checks and two existing perimeter checks pass locally.
See [Azure worker private endpoint ownership](azure-worker-endpoints.md). Acceptance 51 advances; the count remains 48 of 57.

The registry service stage now checks actual private access configuration, selected data region, required members, and complete service-side approval.
The worker registry endpoint and separate preparation endpoint must both be approved, with consistent embedded and collection inventories.
The stable SDK exposes anonymous access, task bypass, and permission mode. Inspection preserves the required platform image-pull authentication setting.
Six new focused local race checks pass. Two changed checks repeat after review fixes; no unchanged broad suite runs.
See [Azure worker registry service admission](azure-worker-registry.md). Acceptance 51 advances; the count remains 48 of 57.

A new prerequisite now supplies the worker-facing process: the standalone teststudio-exchange relay.
The existing API owns a second worker listener and retains private storage. The relay forwards only four bounded operations to one trusted HTTPS backend.
The production command links only the standard library and small wire helpers. It has no storage, identity-service, secret, or cloud-SDK dependency.
Seven new focused local race checks and two existing durable exchange checks pass. Local listener cleanup passes.
The standalone batch introduces a separate scratch-image Dockerfile. Its builder now passes local packaging checks below; provider wiring remains unfinished.
See [standalone private worker relay](worker-relay.md). Acceptance 51 advances; the count remains 48 of 57.

Relay image packaging now compiles only the relay and health commands and reuses the pinned certificate source.
The local-only builder supports explicit architecture and buildx, hashes source and binary inputs, and reuses an unchanged image.
The AMD64 package passes seven focused container checks on the current ARM64 host.
It starts non-root with read-only root, no capabilities, no network, and no writable mounts. Static binary and CA bundle checks pass.
A warm invocation reuses both binaries, certificates, and the exact image. Owned extraction and verification containers are absent.
No UI, runner, broader Go suite, registry push, or deployment runs.
See [relay packaging](worker-relay.md#packaging-boundary). Acceptances 51 and supporting packaging under 57 advance; the count remains 48 of 57.

The selected exchange stage now inspects environment readiness, service-side approval, relay specification, current auth configuration, certificate metadata, routes, and revisions.
The relay image stays outside the untrusted test repository prefix. Exact command and origin arguments, absent runtime credentials, and pull-only lifecycle None remain required.
The stage checks SNI certificate binding, SAN coverage, remaining certificate life, one healthy latest revision, and empty environment HTTP routes.
Six new focused local race checks and one existing relay configuration check pass. Production relay dependencies remain unchanged.
The reader accesses only eight selected resources and collections. It does not claim complete workload occupancy from an application lookup.
See [selected Azure exchange binding](azure-worker-exchange.md). Acceptance 51 advances; the count remains 48 of 57.

The workload inventory stage now reads the configured subscription's complete reported app and job collections within explicit bounds.
It finds the selected relay and rejects other observed exchange occupants across resource groups, duplicate identities, and conflicting placement.
Each SDK page reduces immediately to resource and environment identifiers. Unrelated configuration does not leave the reader.
Missing arrays, unknown placement, failed reads, exceeded item limits, and unfinished paging cannot establish absence.
The shared read transport now accepts registered subscription paths without assuming a resource-group segment.
Six new focused race checks and two selected existing transport checks pass locally.
See [Azure exchange workload inventory](azure-worker-workloads.md). Acceptance 51 advances; the count remains 48 of 57.

The trusted backend prerequisite now supplies --serve-role=worker with one private transport listener and local health probes.
It shares storage and canonical execution and discovery authorization without UI, sign-in, dispatch, or background worker startup.
Role options reject unrelated service settings before opening storage. The existing API worker listener remains available.
Six new focused race checks and three existing artifact or listener lifecycle checks pass locally.
The actual command preserves stored sign-in configuration and closes its owned process and port.
See [trusted worker backend](worker-backend.md). Acceptance 51 advances; the count remains 48 of 57.
No browser, UI build, service image rebuild, registry push, deployment, or cloud request runs.

The backend now supports optional direct TLS through mounted certificate and key material with one exact HTTPS origin.
It validates current leaf identity before storage opens. TLS requires the selected SNI, and requests require its exact HTTP Host.
The real command passes the existing relay's four operations through locally verified HTTPS and shared durable storage.
Two new focused race checks and three existing command or listener checks pass. Only changed certificate and end-to-end cases repeat.
Foreign authority, untrusted roots, legacy TLS, plaintext HTTP, and revoked writes reject locally.
See [direct backend TLS](worker-backend.md#direct-tls). Acceptance 51 advances; the count remains 48 of 57.
No UI, browser, image build, cloud request, registry push, or deployment runs.

A separate staged Azure backend package now defines a dedicated internal environment with public network access disabled.
Its only app uses the exact worker role command, platform HTTPS termination, and the selected private worker port.
Separate identities supply image pull, one database secret, and the shared artifact container. Existing scoped grant modules remain canonical.
Resource-bound outputs identify the backend origin, app, and environment without secret values.
Local schema compilation and six packaging checks pass. The checker inspects no live Azure state.
The core package remains unchanged. No image, UI, browser, Go fixture, registry push, deployment, or cloud request runs.
See [staged Azure trusted backend](azure-worker-backend.md). Acceptance 51 and supporting packaging under 57 advance; the count remains 48 of 57.

The backend observation stage now binds the selected worker-only app, dedicated environment, origin, storage references, identity metadata, and healthy revision.
It reuses the preceding call's transient subscription placement snapshot for reported backend occupancy without two more scans.
The backend and relay share bounded service reads and specification comparison. Missing route arrays remain unavailable rather than empty.
Four new focused race checks and four selected existing exchange or inventory checks pass locally.
Only checks affected by snapshot reuse and stronger occupant coverage repeat. The composed guard remains separate from Network and native dispatch.
See [Azure trusted backend observation](azure-worker-backend-admission.md). Acceptance 51 advances; the count remains 48 of 57.
No browser, UI build, image build, container, live service listener, cloud request, registry push, or deployment runs.

The backend endpoint stage now observes consumer and service approval, NIC and address ownership, dedicated endpoint subnet policy, and the selected relay source prefix.
It requires service-required-member consistency and a dedicated NSG allowing only relay HTTPS to the selected backend address.
The stage reuses the preceding validated backend environment observation without another GET. Shared worker endpoint verification remains canonical.
Four new focused race checks and four existing endpoint or backend regression checks pass locally.
The composed observation still implements no Network and enables no coded job.
See [private backend endpoint observation](azure-worker-backend-endpoint.md). Acceptance 51 advances; the count remains 48 of 57.
No cloud request, browser, UI build, image build, container, live service listener, registry push, or deployment runs.

The relay now checks backend readiness from its own network through GET /readyz and a dedicated three-second HTTPS request.
The backend health reply identifies the worker role. Readiness verifies normal certificate trust, hostname, minimum lifetime, and a bounded exact health schema without caller credentials.
The new optional private backend address pins readiness and worker transport while preserving Host and TLS verification. It bypasses the backend DNS lookup.
The Azure relay definition requires that address and the backend-dependent readiness probe. Endpoint observation must select the same address.
Five new focused race checks and ten selected existing relay, command, backend, and Azure definition or endpoint checks pass locally.
Only changed certificate and configuration cases repeat. The relay production dependency audit remains unchanged.
See [relay backend readiness](worker-relay-readiness.md). Acceptance 51 advances; the count remains 48 of 57.
All owned TLS listeners, request goroutines, and the command-check subprocess close. No browser, container, UI build, image build, cloud request, registry push, or deployment runs.

The relay VNet stage now checks the selected address space, DNS settings, peering, and bounded embedded subnet inventory.
It reuses the endpoint stage's current direct subnet observations and adds one VNet GET without duplicate subnet reads.
Four new focused race checks and two existing endpoint checks pass locally. No cloud operation, image build, UI build, or demo restart runs.
See [relay VNet observation](azure-relay-topology.md). Acceptance 51 advances; the count remains 48 of 57.

The relay perimeter stage now binds its dedicated source NSG, source subnet network settings, and sole default firewall route.
It shares the worker policy generator, rule and route comparison, and bounded SDK reader.
Two concurrent selected reads reuse the topology stage's current subnet observation without another VNet or subnet GET.
Four new focused race checks and five selected existing checks pass locally. No cloud operation, image build, UI build, or demo restart runs.
See [relay security rules and routing](azure-relay-perimeter.md). Acceptance 51 advances; the count remains 48 of 57.

The relay firewall stage now binds the routed address to the actual firewall, dedicated policy, subnet, and public IP association.
Worker and relay profiles share the canonical verifier, definitions, SDK, bounded inventory, and local fixtures.
The stage reuses the current validated VNet observation to compare its firewall subnet without another network GET.
The VNet inventory permits the exact reserved AzureFirewallSubnet name and preserves all other naming and parent restrictions.
Four new focused race checks and seven selected existing checks pass locally. Only failed or affected checks repeat after the naming correction.
See [relay firewall observation](azure-relay-firewall.md). Acceptance 51 advances; the count remains 48 of 57.
No cloud operation, UI build, image build, browser, container, deployment, or demo restart runs.

The relay DNS stage now observes exact lists, enabled rules, its exclusive VNet link, and the actual VNet policy association.
Worker and relay profiles share the verifier, definitions, scoped SDK reader, bounded inventories, and local fixtures.
The relay derives registry data region from the prior selected profile and omits backend and public exchange lookup declarations.
Four new focused race checks and five selected existing checks pass locally. No cloud operation, UI build, image build, browser, or demo restart runs.
See [relay DNS policy observation](azure-relay-dns.md). Acceptance 51 advances; the count remains 48 of 57.

The registry service contract now supports one explicit relay consumer while preserving exactly two consumers when that setting is absent.
The relay registry endpoint stage requires that selected third consumer and binds its endpoint, NIC members, addresses, dedicated subnet, and NSG.
It reuses current registry service approval and the validated VNet observation without duplicate registry or VNet reads.
Six new focused race checks and four selected existing checks pass locally. Only two changed checks repeat for stronger reserved-address and paging-bound coverage.
See [relay registry endpoint observation](azure-relay-registry-endpoint.md). Acceptance 51 advances; the count remains 48 of 57.
No cloud operation, browser, UI build, image build, container, registry push, deployment, or demo restart runs.

The selected relay private-zone stage now derives registry and regional data records from its verified endpoint addresses.
It requires one completed relay-only link, disabled registration, default resolution without fallback, and matching record and link counts.
Worker and relay profiles share the verifier, definitions, scoped SDK reader, and fixtures without duplicate parent reads.
Four new focused race checks and five selected existing checks pass locally.
See [relay private registry DNS observation](azure-relay-private-dns.md). Acceptance 51 advances; the count remains 48 of 57.
No cloud operation, UI build, image build, browser, container, registry push, deployment, or demo restart runs.

The reported private DNS inventory now lists zones across the configured subscription and validates complete reported child link inventories.
It detects additional reported links into worker or relay VNets while permitting unrelated valid zones and links.
The SDK uses a per-call discovered path allowlist, bounded parallel reads, inventory limits, and an atomic request budget.
Four new focused race checks and two selected existing checks pass locally. No cloud operation, UI build, image build, browser, or deployment runs.
See [reported private DNS inventory](azure-private-dns-inventory.md). Acceptance 51 advances; the count remains 48 of 57.

The Azure Private DNS FAQ confirms cross-subscription and cross-tenant zone links.
The reviewed zone-list and link-list APIs cover one subscription and one zone respectively.
This scan is therefore reported-subscription inspection. It cannot replace proof of effective cross-scope visibility or attachment authority.
Inspect actual enforcement and scope coverage before relying on policy or deployment-stack protection settings.
Keep those controls mandatory and native coded execution disabled while they remain unproven.

The selected deny stage now checks actual assignments on both VNets and the three selected private DNS zones.
It requires enforced mode, system protection, child scopes, all-principal coverage, no principal or condition bypass, and only the fixed read exceptions.
Five selected GETs reuse the current reported-inventory chain without duplicate parent observations.
The pinned preview authorization SDK retains enforcement and assignment-condition fields absent from the older stable model. Hosted API support remains unverified.
Four new focused race checks pass for the five-resource contract. Two selected existing inventory checks pass; only changed deny checks repeat.
See [selected network and DNS deny observation](azure-network-deny.md). Acceptance 51 advances; the count remains 48 of 57.
No cloud operation, UI build, image build, browser, container, registry push, deployment, or demo restart runs.

The observer identity stage now binds a separate user-assigned identity to its reported client, principal, tenant, and selected subscription-root grant.
The fixed custom role permits only management reads and the existing bodyless DNS-association observation, without exclusions, data access, or conditions.
An explicit credential factory selects the verified observed client identifier without a default-credential fallback or token request during construction.
Three concurrent selected GETs add no duplicate parent observation. Complete native credential wiring and effective authorization remain unproven.
The SDK uses the scoped role lookup. Its GetByID disambiguation query exceeds the exact transport boundary and is rejected.
Five new focused race checks and one selected existing protection regression pass locally in 4.003 seconds.
See [observer identity and selected read grant](azure-read-identity.md). Acceptance 51 advances; the count remains 48 of 57.
No cloud operation, browser, UI build, image build, container, registry push, deployment, or demo restart runs.

The bound observer session now rebuilds all 20 SDK readers with one credential for the verified observed client identifier.
Binding performs one bootstrap inspection. Later checks use only the bound credential and refresh the complete resource chain without repeating bootstrap reads.
Client construction makes no observer request. Runtime mismatches, credential failures, unavailable state, changed grants, and closed sessions block inspection.
Owned readers close in reverse order. Closing waits for active checks and prevents later requests.
Two new focused race checks pass in 2.325 seconds. One selected existing credential check runs without failure in the initial command.
The new helper corrects its audience expectation by reusing the established credential fixture. Only the two changed checks repeat.
See [bound observer sessions](azure-read-identity.md#bound-observer-session). Acceptance 51 advances; the total remains 48 of 57.
No real token, cloud request, browser, UI build, image build, container, registry push, deployment, or demo restart runs.

The reported read-deny stage now gates each bound observer check through the same selected credential.
It inspects the unfiltered subscription response, requires the five selected protections, and rejects potentially applicable read or DNS-association conflicts.
Principal exclusions, inherited scopes, explicit enforcement, conditions, wildcard actions, paging limits, and missing metadata remain explicit.
Group applicability stays conservative without directory membership queries. Successful fixtures do not establish complete effective authorization or hosted visibility.
The session now owns 21 readers. The composed check has a 130-second context and at most 2212 requests.
Normal added cost is one GET; the existing parent chain runs only after the scan passes.
Three new focused race checks and two changed session checks pass in 2.716 seconds. No unchanged broad suite repeats.
See [reported subscription read-deny inspection](azure-read-deny.md). Acceptance 51 advances; the count stays 48 of 57.
Only public documentation requests leave the environment. No real token, cloud operation, UI build, browser, container, image build, registry push, deployment, or demo restart runs.

The observer host stage now binds the selected controller app, four identities, explicit lifecycles, service image, and healthy current revision.
BindHostObserver builds a 22-reader session using the same verified observer credential and no repeated bootstrap reads.
Two concurrent host reads follow the existing chain. The composed context is 135 seconds, with at most 2216 requests.
The core package optionally declares the observer identity and its fixed subscription read role and assignment, with API-only Main exposure.
The option defaults to false and enables no native observer startup or coded job. Its outputs contain references rather than secrets.
Three new focused race checks and one selected existing session regression pass in 2.456 seconds.
The package compiles locally without warnings. Contract checks reuse the current compiled template until a real API-role hardening change requires a final compile.
No unchanged Go suite repeats for that packaging change. No UI build, browser, image build, container, cloud operation, registry push, deployment, or demo restart runs.
See [observer host attachment](azure-observer-host.md). Acceptances 51 and supporting packaging under 57 advance; the count remains 48 of 57.

The host-bound workload reader now retains reduced identity references and checks reported observer placement during its existing subscription app and job scans.
The selected host must appear with matching observer metadata. Other app or job attachments and reported client or principal aliases block inspection.
Unknown identity metadata remains unavailable. Earlier placement-only readers preserve their existing behavior.
Three new focused race checks and two selected existing checks pass in 2.410 seconds, including the complete bound host path.
No additional collection, SDK resource, permission, reader, or request is added. Session bounds remain 135 seconds and 2216 requests.
See [reported observer identity placement](azure-worker-workloads.md#observer-identity-placement). Acceptance 51 advances; the count remains 48 of 57.
No cloud operation, UI build, browser, image build, container, registry push, deployment, or demo restart runs. Only public documentation requests leave the environment.

The identity reader now requires an explicit empty federated-credential collection during bootstrap and each bound inspection.
The existing SDK adds one selected collection GET. It rejects any credential object and treats missing values, null entries, or continuations as unavailable.
No continuation is followed, and no credential metadata reaches callers. A failed observation prevents managed identity credential construction and later host reads.
Three new focused race checks and four changed identity, coverage, and session checks pass in 3.179 seconds.
No session reader, dependency, API version, role grant, or packaging build is added. One concurrent GET increases the existing request ceilings by one.
The identity and bootstrap bound is now 2197 requests, the read-deny session bound is 2213, and the host-bound session limit is 2217.
Their contexts remain 125, 130, and 135 seconds respectively. No parent scan or bootstrap inspection repeats during a bound check.
See [federated credential absence](azure-read-identity.md#federated-credential-absence). Acceptance 51 advances; the count remains 48 of 57.
No cloud operation, UI build, browser, image build, container, registry push, deployment, or demo restart runs. Only public documentation requests leave the environment.

The identity reader now requires a selected resource-scoped protection assignment on the observer identity and its child scopes.
Its fixed contract requires explicit enforcement, system protection, all-principal coverage, no conditions or exclusions, and only read and lock-delete exceptions.
The subscription scan now requires six selected protections. The existing five network and zone assignments retain their original exception contract.
The shared read-conflict check now considers a DNS-association denial only at scopes where the VNet operation can apply.
Three new focused race checks, eight changed identity and scan checks, and one shared network-protection regression pass in 4.240 seconds.
The fifth concurrent selected GET increases identity and bootstrap limits to 2198, the read-deny session bound to 2214, and host checks to 2218 requests.
Contexts remain 125, 130, and 135 seconds. The host session retains 22 readers. No extra parent scan, role grant, dependency, or packaging build is added.
See [identity write protection](azure-read-identity.md#identity-write-protection). Acceptance 51 advances; the count remains 48 of 57.
The inspector creates no assignment and attempts no forbidden mutation. Actual protection provisioning, enforcement, supported response shape, and owning-management authority remain unverified.
No cloud operation, UI build, browser, image build, container, registry push, deployment, or demo restart runs. Only public documentation requests leave the environment.

The identity stage now requires two selected deployment stacks with fixed policy and exact protected-root membership.
The network stack reports the five existing VNet and DNS roots. The distinct identity stack reports only the observer identity.
Both require Succeeded, denyWriteAndDelete, child coverage, no excluded principal, and the fixed effective exception sets.
The SDK reads only the two selected resources and drops templates, parameters, outputs, and deployment metadata from returned observations.
Failed or changed protection blocks credential construction and later host reads. Successful state is never cached.

Three new focused race checks and six changed identity, scan, and session checks pass in 3.668 seconds.
After correcting missing-metadata error categories, only the two affected checks repeat and pass in 1.925 seconds.
Local checks first exposed value-shaped SDK responses and an incompatible resource-group-only helper; both are corrected before the passing runs.
Seven concurrent identity-stage observations now yield ceilings of 2200 bootstrap, 2216 read-deny, and 2220 host requests.
Contexts remain 125, 130, and 135 seconds. The host session retains 22 readers and repeats no parent scan or bootstrap during a bound check.
The stable armdeploymentstacks/v2 v2.0.0 adds one module and two hashes without upgrading other dependencies.

The API exposes no exact deny-assignment link. These checks establish reported consistency, not direct ownership or stack-management authority.
The optional core package creates no protection arrangement. Actual enforcement, provenance, other consumers, complete host admission, isolation, and native wiring remain open.
See [reported protection stacks](azure-protection-stacks.md). Acceptance 51 advances; the total remains 48 verified of 57, with nine open.
No cloud operation, UI build, browser, image build, container, registry push, deployment, or demo restart runs.
External requests retrieve only public documentation and the public SDK module.

The observer host now requires explicit runtime settings and an independently constructed expected core API definition.
App and active revision must match that definition, even when both report the same changed arguments or environment.
The contract binds API role, PostgreSQL, separate origins, Azure Blob runtime client, Key Vault and OIDC references, bootstrap identity, and local-account policy.
It rejects command overrides, additional arguments or variables, alternate storage, inline credentials, extra mounts, substituted ingress, probe drift, and replica changes.
The existing image, identity exposure, lifecycle, placement, and healthy revision checks remain required.
Optional OIDC client references follow the two existing package slots. Public clients need no secret.

The optional observerBinding package output now includes matching non-secret runtime settings from the API deployment parameters.
It exposes references and bootstrap identifiers, not secret values or tokens. The observer remains disabled by default, with no native worker startup.
Four new focused race checks and three changed host regressions pass in 2.918 seconds.
Four directly affected federation, protection, and identity-placement host checks pass in two targeted selections, taking 2.172 and 2.039 seconds.
The first dependent selection matches two tests; the two other intended checks run afterward under their exact current names.
The changed output compiles once with pinned Bicep 0.48.1 and passes local package contracts without warnings or registry restore.
No unchanged Go suite, UI build, browser, image, container, provider operation, registry push, deployment, or demo restart runs.

The guard reuses the existing two host GETs, 22-reader session, 135-second context, and 2220-request ceiling.
No new dependency, grant, scan, credential, or observation cache is added.
Reported configuration does not prove secret contents, actual image behavior, platform authentication settings, effective authorization, or management authority.
See [fixed host runtime contract](azure-observer-host-runtime.md). Acceptances 51 and supporting packaging under 57 advance; the count remains 48 verified, nine open.

The host reader now requires the selected API app's authConfigs/current with an explicit false platform enabled flag.
It uses the existing SDK and returns only resource identity and that flag. Provider, registration, encryption, and login metadata do not reach callers.
Missing resources and unknown state remain unavailable. A changed enabled setting rejects the next bound inspection, while restored disabled state can pass.
The core app module now declares disabled platform authentication for both API and artifact roles, regardless of the optional observer setting.
TestStudio retains its own OIDC, local-account, session, and private-report authorization contracts.

Three new focused race checks and seven affected regressions run locally. Nine pass initially; one exposes a reader's missing-resource error category.
After that normalization is corrected, only the SDK privacy check and bound authentication check repeat and pass in 1.972 seconds.
The initial targeted selection takes 3.503 seconds. The changed Bicep module compiles once with pinned 0.48.1 and passes package contracts without warnings.
Three concurrent host reads replace two, increasing only the host ceiling to 2221 requests. Its context remains 135 seconds with 22 readers.
Bootstrap remains 125 seconds and 2200 requests; diagnostic read-deny sessions remain 130 seconds and 2216 requests.
No role grant, dependency, repeated parent scan, observation cache, UI build, browser, image, container, provider operation, deployment, registry push, or demo restart is added.
See [reported host platform authentication](azure-observer-host-auth.md). Acceptance 51 and supporting packaging under 57 advance; the total remains 48 verified, nine open.
Reported state does not establish deployed authentication behavior, other identity consumers, protection-management authority, execution isolation, or complete native admission.

The host observer now requires an explicit empty controller-environment HTTP route collection on every successful Check.
Its mandatory RoutesAbsent method reads one selected page through the existing SDK client and observer credential.
Any route object rejects ownership. Missing or null values, null-only rows, failed reads, and incomplete empty pages remain unavailable.
It never follows continuations, even to the same collection, and returns no route names, hosts, targets, rules, or response bodies.
The staged profile supports no environment routing overrides, including routes for unrelated apps. The inspector changes no existing route.
The core package declares no such override. Installation tooling must select a suitable controller environment.

Three new focused race checks and seven affected host or protection regressions pass in 4.929 seconds.
They cover explicit absence, present and unknown routes, same and foreign continuations, exact scope, rejected queries and mutations, cancellation, freshness, and parent gating.
Four concurrent host reads now yield four ordinary GETs and at most six host-specific requests with revision paging.
Only the host ceiling increases to 2222 requests; its context remains 135 seconds with 22 readers.
Bootstrap remains 125 seconds and 2200 requests; diagnostic read-deny remains 130 seconds and 2216 requests.
No dependency, grant, additional reader, repeated parent scan, observation cache, package build, UI, browser, image, container, provider call, deployment, registry push, or demo restart is added.
See [reported controller route absence](azure-observer-host-routes.md). Acceptance 51 advances; the total remains 48 verified, nine open.
Reported absence does not prove effective visibility, prevent future route writes, establish external proxy behavior, or complete native execution admission.

A separate PreparationRegistryGuard now checks the selected trusted registry path after the existing registry service and worker network observation chain.
It binds the approved preparation endpoint to its NIC, exact registry and data members, dedicated endpoint subnet and security rules, source subnet, controller VNet, and separate private zone.
Fresh VNet reads preserve worker separation, prefix containment, core-subnet separation, and direct-versus-embedded subnet agreement.
The fixed security rules allow exactly the selected private controller VNet prefixes to the two IPs on TCP 443, followed by inbound and outbound deny rules.
This preserves trusted controller and backend platform image pulls alongside preparation; writer authority remains separate.
Configured controller spaces copy and sort into the bound settings, match fresh VNet metadata, and cannot overlap workers.
The private zone has exact preparation addresses and one completed controller link, with no registration or public fallback under the existing DNS contract.
The two explicit-credential readers reuse the network and DNS SDKs. Static SDK definitions support installation review without a cloud mutation.

Four new targeted race checks pass after the final policy correction in 2.070 seconds.
Review adds fresh address-space separation, containment, caller-copy protection, and the existing canonical single-prefix representation helper.
Earlier verification repeats only changed branches. The final controller-prefix source policy receives all four directly affected checks.
No existing Go check, broad suite, UI, browser, package, image, container, provider call, publication, deployment, registry push, or demo restart runs.
The standalone guard has a 40-second context, including its existing 30-second registry parent, with nine ordinary additional requests and a twelve-request additional paging ceiling.
It repeats no complete parent inspection and caches no success. Parent failure blocks preparation reads; network failure blocks DNS.
The existing host-bound observer and its 22-reader limits remain unchanged.

The source queue also requires isolated discovery. These read-only checks do not enable publication, a preparation queue, native coded execution, or complete Network admission.
Actual preparation workload placement, effective DNS and egress, separate writer identity grants, protection-management authority, and native wiring remain required.
See [trusted preparation registry path](azure-preparation-registry.md). Acceptance 51 advances; the total remains 48 verified, nine open.

The preparation service now supports an optional assembly-only role and a durable handoff to the normal discovery queue.
The writer publishes a validated immutable image, releases its owned journal without deleting content, and commits the handoff, lease removal, and audit atomically.
Assembly and discovery retain separate references. Claim checks now bind the exact reference as well as worker identity.
The API can discover the handed image and complete review, approval, publication, and owned discovery cleanup without a builder or registry writer.
Assembly-only workers run no discovery. Compatible discovery workers recheck retained source, expiry, current access, and normal artifact provenance.
Confirmed interrupted assembly resumes only its handoff; concurrent writers and repeated recovery create no duplicate build or discovery index.
Waiting cancellation, expiry, revocation, and cancellation during handoff block dispatch. Old jobs and the default combined workflow remain compatible.

The native preparation command supports --assembly-only for existing ECS and Kubernetes settings and separate Azure assembly settings.
It rejects local mode before creating state. This queue boundary permits the trusted Azure writer without enabling native coded execution.
OpenAPI 0.34 and generated client types include the two optional assembly fields.
Preparation and repository views show Waiting for discovery; page help explains when review becomes available.

Six new behavior checks, eight targeted existing source regressions, generated-contract synchronization, and the Angular production build pass locally.
The first source selection exposes missing expiry validation after queue handoff. Retained source now revalidates before discovery, including before retention sweeping.
Only the four new source checks repeat after that fix, passing in 4.914 seconds. Registry preservation passes for both runners in 1.426 seconds; native rejection passes in 1.410 seconds.
Contract synchronization passes in 0.269 seconds. The final UI build passes in 2.830 seconds and reuses the private viewer.
A premature UI build after a script path error and a later one-line component formatting error are corrected before the final build; neither is used as final UI evidence.
No existing browser workflow repeats: the UI changes only label the new state and add help, with no new controls or layout. Existing keyboard, theme, and narrow-layout evidence is retained.
No cloud operation, image build, container, registry push, deployment, browser, or demo restart runs. Local TLS fixture servers close through test teardown.
See [separate source assembly and discovery](source-assembly-handoff.md). Acceptance 51 advances; the total remains 48 verified, nine open.

Azure preparation library settings now compose private networking, a selected writer identity, its registry-scoped AcrPush grant, and an explicit managed-identity publishing client.
The factory reuses the existing registry chain and owns ten SDK readers. Each client opening refreshes the chain and selected grant.
Four new local checks and one affected observer regression pass. See [Azure preparation writer](azure-preparation-writer.md).
This advances 51 without closing an item; the total remains 48 verified, nine open.
The [native Azure assembly profile](azure-native-assembly.md) now configures explicit reader and writer credentials, PostgreSQL, registry runtimes, and fresh admission before each build.
It registers both recovery builders without a discovery inspector. Owned cleanup and confirmed handoff do not require another control-plane observation.
The separate headless Bicep template declares four identities, scoped grants, vault references, an always-on process, and a private persistent journal mount.
The worker defaults to disabled so bootstrap can establish identity bindings before the private settings secret is populated.
Four new Go checks and two affected regressions pass. The template compiles once and passes six local packaging checks after checker corrections.
The core checker reuses its saved compiled template. No image rebuild, container, browser, provider call, or deployment runs.
Native assembly advances 51; the verified count remains 48 of 57.
Native startup now checks the selected app, environment, storage binding, reader identity, and selected custom read grant before claiming jobs.
The fixed headless runtime and independent revision comparison include command, images, source runtimes, vault references, mount options, identities, and scaling.
The host-bound publisher owns eleven readers and refreshes its chain once before selecting the writer credential or starting a new build.
Startup waits at most three minutes for unavailable platform state and stops on a definite mismatch or cancellation.
The Bicep binding now includes the complete host object. Native settings require it and reject mismatched runtime flags or journal paths before state creation.
Five new targeted checks and affected shared-runtime, identity, native, and publisher regressions pass locally, with one corrected error-category expectation.
The package compiles once. Checker correction reuses its compiled output; no image build, live service call, browser, or demo restart runs.
This advances 51 and preserves 48 verified items out of 57.
The [preparation network profile](azure-preparation-network.md) now binds source egress, its selected NAT, private PostgreSQL address, and the private journal account, share, endpoint, NIC, subnet, and DNS.
Native settings require this network binding. The full publisher owns thirteen readers and checks the host chain once before the added network observations.
The separate network template declares those resources without fetching keys or enabling coded jobs.
Native startup also checks its actual writable CIFS mount and probes file and directory synchronization before database access.
Six new focused checks and affected native and publisher regressions pass locally. Network, worker, and core packaging checks pass with existing templates reused where applicable.
The initial network compile reports a DNS suffix warning; using the environment storage suffix resolves it. The final compile has no warnings.
Only the journal and uncertainty checks repeat after their probe-cleanup and missing-approval scenarios change. Checker refinements reuse the compiled template.
This advances 51; the count remains 48 verified of 57. No cloud request, image build, browser, container, or demo restart runs.
Next complete native Azure discovery and execution configuration against the existing job, supervisor, evidence, and admission contracts.
Keep actual cloud verification separate. Do not repeat the completed local source and interface workflows for provider-only configuration work.
Actual source egress, filesystem behavior, effective read authority, and deployment recovery remain separate deployment evidence.
The selected grant check does not establish absence of additional authority or reuse of the writer by other workloads.
Keep complete Azure discovery and coded execution admission separate until implemented.
Reported app and job checks do not establish exclusive use across all Azure resources or subscriptions.
Keep hosted read coverage, complete effective authorization, and cross-scope visibility separate until authoritative deployment evidence exists.
Wire the bound execution observer into native commands only after complete execution admission exists.
Verify actual resource-scoped assignment provisioning and enforcement, and authority to change the owning protection arrangement.
Establish authoritative existing cross-scope visibility or clean installation provenance. Observed deny assignments do not remove pre-existing foreign links.
Continue environment attachment control, remaining perimeter protection, and trusted preparation networking.
A completed list response does not prove authorization coverage or prevent later attachments. Keep those native installation controls mandatory.
Use the worker backend role to verify its restricted route, TLS authority, and placement before completing service admission.
The staged Azure backend selects platform HTTPS termination. Other approved profiles can select direct TLS.
Selected origin, app, environment, command, identity metadata, and revision observation now pass local fixtures.
The selected backend endpoint, NIC, address, approval, source prefix, and endpoint subnet policy now pass local fixtures.
Pinned relay backend routing, TLS trust, and worker-role readiness now pass local checks from owned TLS fixtures.
The selected relay VNet now passes local address-space, DNS-setting, peering, and subnet-consistency checks.
The selected relay source NSG and default route now pass local binding, rule, attachment, and bypass checks.
The selected relay firewall, dedicated policy, public IP backlink, and firewall subnet consistency now pass local checks.
The selected relay resolver policy, exact lists, enabled rules, exclusive link, and actual VNet association now pass local checks.
The selected relay registry endpoint now passes local approval, NIC ownership, member-address, dedicated subnet, and security-rule checks.
The selected relay registry records, complete inventory counts, completed exclusive link, disabled registration, and no-fallback settings now pass local checks.
Verify authoritative effective VNet-wide DNS visibility, trusted preparation networking, and deployed certificate trust next.
The direct subscription inventory now detects reported unexpected zone links. It does not establish the complete effective view or prevent later changes.
The selected deny observer now checks enforced protection metadata for both VNets and three selected zones, with no join or record-write exception.
Verify actual enforcement and protection-management authority. A matching response does not establish clean creation provenance or complete existing cross-scope coverage.
The registry service guard now permits the explicit relay consumer alongside worker and preparation endpoints while rejecting unknown consumers.
Inspect effective identity grants and backend infrastructure subnet controls.
Backend hostname lookup is unnecessary in pinned mode. Preserve worker, exchange, registry, and platform DNS controls.
Keep the staged backend separate from core deployment and coded worker packaging until admission and private relay access are complete.
Resolve effective VNet-wide private-zone visibility and attachment controls. Selected-zone inspection cannot rule out other shadowing zones.
Keep that proof separate from configured record correctness and private endpoint ownership.
Provide trusted preparation with a separate private registry endpoint and DNS path. Do not add controller links to worker zones or peer worker and controller networks.
Do not treat the configured public exchange hostname or allowed DNS descendants as a complete private boundary.
Check live platform aliases and CNAME compatibility in the separate hosted phase. Preserve the root-domain block.
Then verify the isolated exchange service before composing complete admission.
Verify unrelated workload inventory, cross-job isolation, and actual execution placement before composing Network.
Bind topology and all perimeter settings into networkProfile. Do not substitute a partial topology digest.
Keep coded jobs disabled until those controls pass. Ordinary jobs must meet the execution isolation contract or use a suitable Azure backend.
Bind the stop-only identity role, workflow management authority, capacity limits, and orphan monitoring during infrastructure packaging.
Then bind actual managed identities and registry roles in native setup and infrastructure packaging. The authentication adapter alone enables no worker.
Keep network and deadline hooks mandatory. A local fake does not establish a deployable Azure isolation boundary.
Replica timeout alone does not cover queue delay or satisfy the absolute deadline contract.
Keep inspected non-root images mandatory. Azure lacks the ECS root-filesystem and capability overrides; verify its own isolation arrangement.
Complete private HTTPS relay ingress, the restricted trusted worker backend, and preparation wiring before enabling native coded execution or adding coded-worker Bicep resources.
Preserve the later remote gateway and recording requirements. Keep application secrets in trusted gateways.
Run only targeted local fixtures during these batches. Keep live MMDev and Azure deployment verification separate.
Acceptances 41, 46, and 51–57 remain open.

### Current hosted packaging checkpoint

The portable service image now packages the API, UI, private trace viewer, and bounded health command.
API and preparation processes accept database environment references without exposing their values in arguments or initialization errors.
The final image uses a non-root user, a read-only root filesystem, separate writable mounts, trusted CA certificates, and embedded timezone data.
The producer reuses the canonical static assets and hashes compiler, binary, and package inputs. No registry push or deployment occurs.
The local arm64 package passes startup, health, storage persistence, origin separation, and database-canary checks.
The implementation ledger records exact evidence and verified cleanup. The count remains **48 verified, 9 open**.

Kubernetes service packaging now includes a Helm chart with private references, separate origins, persistent artifacts, replica constraints, and lifecycle probes.
Local lint, four strict-schema renderings, and nine rejected configurations pass without contacting a cluster.
The image adds independent liveness and an explicit hosted PostgreSQL guard. The count remains 48 verified and 9 open.
See [Kubernetes service profile](kubernetes.md) for operator settings and the verified local evidence.

AWS ECS with CloudFront core packaging now defines 31 resources and passes local schema and cross-resource checks.
Its independent stack references PostgreSQL and initial OIDC settings through Secrets Manager.
Separate UI and artifact distributions preserve identity forwarding, private evidence authorization, and zero-duration error caching without HTML error substitution.
Private Fargate tasks use the immutable service image, startup-only secret access, scoped evidence permissions, process liveness, and database readiness.
The image adds a non-root temporary volume declaration. Strict bounded environment JSON supports hosted configuration without a shell or writable configuration files.
Initial provider lists validate completely before database initialization. Existing file configuration remains available.
The current image passes targeted configuration and volume startup. Earlier checks from the same batch cover the complete 12-check local package contract.
No AWS operation, browser run, source-runtime rebuild, or unchanged UI build occurs.
See [AWS core service profile](aws.md) and the implementation ledger for exact evidence and remaining scope.
The count remains **48 verified, 9 open**. No hosted acceptance item closes from local packaging checks.

Azure Container Apps core packaging now uses separate API and artifact service roles with shared PostgreSQL and private Blob storage.
Its Bicep modules compile locally and pass seven packaging checks without Azure account access.
The profile separates runtime Blob permissions from platform-only registry and named-secret identities.
The artifact role starts no background workers, exposes no sign-in or UI routes, and preserves current password-authentication state.
Focused local checks cover health, listener startup failure, shutdown, password-backed artifact access, and later authentication revocation.
The AMD64 service image now builds through buildx and passes targeted local role and configuration checks.
The cross-architecture producer fixes the legacy builder failure and verifies certificate-extraction cleanup.
See [Azure core service profile](azure.md) for operator inputs, platform limits, and exact local evidence.
The count remains **48 verified, 9 open**. Core packaging does not establish complete hosted execution or deployment parity.

The common private worker exchange now transports frozen work, start observations, immutable reports, and checksummed attachments.
It binds one active controller-owned attempt to one worker instance and reconstructs its ticket after controller restart.
Current run authority, leases, cancellation, project state, and deadlines control worker access.
Private staging uses project and run reservations, per-file upload leases, and checkpoint-gated cleanup.
An HTTPS client rejects redirects and excludes private diagnostics. The optional private listener is disabled by default.
Focused service, live-run TLS, report compatibility, export barrier, and native listener checks pass locally.
See [Private hosted worker exchange](worker-exchange.md) for exact limits and integration order.
Acceptance **51** advances. The count stays **48 verified, 9 open**.
The transport and supervisor now connect to the optional Kubernetes provider described below.

The trusted hosted runner supervisor now fetches frozen work and invokes the fixed Go or Playwright runner outside the API.
It strips ambient credentials from the child environment, forwards bounded progress, and snapshots owned files for exact upload retries.
Runner process exit, observed test outcome, provider delivery, and evidence completeness remain separate.
Missing or invalid reports produce explicit unmeasured infrastructure observations with available private logs.
A shared capture decision preserves failure-only evidence for timing-policy failures.
Thin supervised source runtimes reuse the cached Go and Playwright bases without rebuilding toolchains or browser dependencies.
Local TLS/process checks and narrow runtime package probes pass. Acceptance 51 advances; the count remains 48 verified and 9 open.
See [Hosted runner supervisor](job-supervisor.md) for its job contract, packaging, and remaining provider controls.

The common hosted lifecycle driver now connects approved immutable images, frozen work, provider status, and the normal managed execution hooks.
It validates ownership, deadlines, and exit status. It stops owned jobs on cancellation and recovers ambiguous creation without another dispatch.
Ticket revocation precedes checkpoint publication. Ledger gates precede provider deletion and exchange staging cleanup.
Temporary evidence read failures stay retryable. Restart input preserves the original attempt capture flag.
Focused local fixture and race checks pass. The driver now has an optional Kubernetes adapter described below.
Acceptance 51 advances; the count remains 48 verified and 9 open.
See [Hosted execution lifecycle](hosted-execution.md) for the provider contract and current limits.

The OCI registry inspector now verifies approved distribution metadata for Go and Playwright without a Docker daemon or cloud SDK.
It binds one explicit registry namespace, selects the configured platform, and checks digest, size, runtime, supervisor, and source ownership metadata.
Scoped registry authentication supports credential rotation. Approved configuration-blob redirects strip registry credentials.
Focused local TLS race checks pass. Provider enablement remains explicit and optional.
See [Hosted registry inspection](registry-inspection.md) for the contract and bounds.
Acceptance 51 advances; the count remains 48 verified and 9 open.

The first concrete provider adapter now dispatches one owned Kubernetes Pod per offline Go or Playwright attempt.
A durable reservation prevents another controller from submitting the same creation. Lost acknowledgement leads to discovery instead of another launch.
The adapter checks Pod ownership, immutable settings, namespace protection, service account, RuntimeClass, and exact network policy.
Cancellation shortens the Pod runtime deadline. Deletion waits for terminal state and uses UID and revision preconditions.
Lost live Pods remain unresolved instead of being treated as confirmed stopped workers.
Timing identity now includes the provider execution pool so class or node-selection changes start a separate baseline.

The shared exchange now gates executable input until the controller verifies provider creation and activates that exact attempt.
Unexpected admission mutations never receive uploaded code. Recovery does not activate an uncertain pending job.
Already accepted reports still recover after later checkpoint failure without another launch.
The gate rechecks current run authority. It has no worker HTTP activation route.

The server accepts bounded file or environment Kubernetes worker settings and requires PostgreSQL, a private listener, and its Kubernetes location.
The Helm profile adds an optional dedicated restricted namespace, worker-only service account, narrow controller permissions, rotating API credentials, and private ingress.
Local race checks, rendered settings comparison, two schema profiles, four rejected worker configurations, a generated Pod schema, and five startup guards pass.
No cluster, cloud account, registry push, source-runtime rebuild, browser suite, or unchanged UI rebuild is used.
See [Kubernetes execution workers](kubernetes-workers.md) and the implementation ledger for exact local evidence.
Acceptance 51 advances. Acceptance 41 gains execution-pool identity support. The count remains 48 verified and 9 open.

Registry source assembly and immutable OCI publication now have a separate builder adapter.
It appends a deterministic bounded source layer to an inspected supervised runtime without a Docker daemon or uploaded build commands.
Private upload journals retain exact ownership, cancellation, lost acknowledgements, and interrupted session checkpoints.
Local TLS and race checks cover Go and Playwright. Registry inspection and local assembly retain their existing behavior.
The API, preparation-worker, and publisher packages compile. No live registry, image rebuild, UI build, or cloud operation occurs.
See [Registry source assembly](registry-preparation.md) for its exact scope and recovery limits.
The preparation worker now enables registry assembly through explicit Kubernetes settings and immutable runtime flags.
Hosted discovery uses a distinct gated packet and report. It creates no test result or timing baseline.
The normal build checkpoint, source review, exact artifact approval, and publication accept immutable registry references.
Provider success remains required alongside a valid discovery report.
Source leases and current permissions control worker authority. Saved preparation checkpoints control provider and exchange cleanup.
Cleanup-only recovery remains available when new builds are disabled. It never repeats discovery or provider dispatch.
The service image includes the separate preparation command. Both thin runtime overlays advertise the discovery supervisor capability.
Targeted local race, TLS/process, native startup, and package checks pass. Trusted runtime bases and trace assets are reused.
The source help reflects the configured execution location, and the Angular production build passes.
See [Hosted preparation](hosted-discovery.md) for exact native setup and retained evidence.
Acceptances 51 and supporting preparation isolation in 52 advance. The count remains 48 verified and 9 open.

The optional Helm preparation package now includes a separate workload, projected provider identity, bounded resources, and private durable journal claim.
API registry reads and preparation registry writes use separate Secret references and process environments.
The API advertises only configured source builders. A Go-only preparation worker no longer advertises Playwright ZIP staging.
Cleanup-only mode disables new staging and build flags while retaining provider settings, registry credentials, and durable journals.
Single-writer claims use Recreate. Multiple preparation replicas require shared ReadWriteMany journals.
Six local schema profiles, twelve rejected settings, both rendered provider configurations, and nine native or packaged startup guards pass.
The source-profile parser passes targeted checks. UI assets, the trace viewer, and trusted source runtimes are reused.
See [Kubernetes source preparation](kubernetes-preparation.md) for installation, drain, recovery, limits, and exact local evidence.
This advances 51, supporting failover work in 52, and deployment packaging in 57. The total remains 48 verified and 9 open.

AWS worker task and deadline contracts now build immutable SDK requests for offline Go and Playwright execution and discovery.
They bind exact ownership, fixed non-root task settings, resource class, private launch configuration, and a separate stop schedule.
Recovery proofs omit worker tickets and cannot generate another launch request.
Targeted local race and signed TLS wire checks pass. Current API, preparation, and supervisor commands compile without enabling ECS workers.
See [ECS task and deadline contracts](ecs-worker-contract.md) for current scope, verified AWS constraints, and exact remaining work.
Acceptance 51 advances. The total remains 48 verified and 9 open.

The durable ECS provider now reserves registration and launch, recovers lost acknowledgements without redispatch, observes exact tasks, and verifies independent deadlines.
Task and definition receipts survive stop-time tag loss. Saved terminal observations survive later task-history expiry.
Checkpointed cleanup verifies the owned schedule and definition. Asynchronous definition deletion retains its result and uses a one-minute read backoff.
The shared lifecycle waits for provider admission before opening executable input and stops work after admission loss.
Local race, database-reopen, mutation, concurrency, missing-history, and targeted compatibility checks pass.
See [Durable ECS lifecycle](ecs-worker-lifecycle.md) for exact evidence, ownership, cleanup, and uncertainty rules.
This advances 51 and supporting replica recovery under 52. The total remains 48 verified and 9 open.

The production network checker, bounded settings loader, and supported task storage now pass focused local checks.
The scoped SDK factory, ECR credential callback, and optional API and preparation flags now pass focused local checks.
Keep actual task and pool network admission mandatory. Private worker infrastructure and independent controller permissions now have staged packaging.
Rolling target-health admission now passes local verification. ECS server wiring and private infrastructure are also implemented.
Continue the Azure provider and remote gateway gaps. AWS deployment verification remains separate.
Do not recreate uncertain tasks after the finite RunTask idempotency window. Missing live tasks remain unresolved without a saved terminal observation.
Keep exact image approval, gated dispatch, provider ownership, cancellation, deadlines, and checkpointed cleanup.
Use local protocol fixtures and package rendering. Do not run AWS, cluster, or Azure deployment checks in this session.
Keep uploaded source outside the API and retain exact source approval, project ownership, catalog review, and publication boundaries.
Reuse trusted supervised runtimes. Do not execute uploaded Dockerfiles or add privileged building inside the API.
Preserve the activation gate, cancellation, deadline, evidence, and cleanup contracts for preparation work.
Then continue Azure workers and remote gateway delivery through the same managed lifecycle.
Use the frozen work packet as the runner's existing bounded stdin contract.
Preserve immutable source approval, timing, result, capture-policy, and gateway contracts.
Keep real target credentials in the trusted gateway. Untrusted workers receive neither application sessions nor provider SDK credentials.
Keep resolved gateway secret versions available for restart redaction without sending their values to the test worker.
Preserve the configured private Kubernetes route. Add private worker HTTPS routing to the remaining provider profiles before enabling their dispatch.
Azure needs its own private service arrangement for the worker listener rather than an additional public TCP port.
Use local protocol fixtures for the remaining provider dispatch, status, cancellation, deadline, and restart reconciliation paths.
Preparation must publish and inspect immutable artifacts outside the API before hosted code can execute.
Do not substitute fire-and-forget job creation for these lifecycle and recovery requirements.
Preserve separate API/UI and artifact origins. Require installation-specific PostgreSQL, identity, storage, and secret configuration.
Keep untrusted test execution outside the API. Do not promise hosted coded execution before its network, artifact, deadline, and recovery contracts exist.
Use local schema, protocol, and packaging checks. Cloud deployment verification remains separate work.
Do not repeat passing source, browser, native probe, grant, storage, or provenance suites for packaging prose or unrelated changes.

### Recorded browser execution scope

The native recorder, saved source editor, and first environment-backed local browser profile now work together.
The offline profile stays separate. Recorded drafts prepare through the single-origin playwright-environment profile.
See [Website execution](browser-execution.md) for its reviewed source binding, request policy, limits, and operator setup.
The implemented runtime binds reviewed navigation to one registered endpoint. The default endpoint identifier is api.
It retains network=none on browser workers and uses an owned Unix-socket gateway.
The profile permits all methods to that endpoint. It does not claim read-only behavior or encrypted payload inspection.
Additional origins, injected credential bindings, custom request policies, and hosted browser execution remain later profile work.
Preserve the original recording and source provenance. Do not silently rewrite arbitrary code or remove its target restriction.
Keep dependencies inside trusted preparation workers. Add bounded dependency preparation only when the selected profile needs it.

Enforce allowed target origins, current permissions, protected-target safety, and dependency safety before dispatch and each attempt.
Apply network and request policies to browser navigation and direct API clients used by the suite.
Define redirects, subresources, external origins, and credential bindings explicitly. Do not make the worker network unrestricted.
Never copy a recorder's personal browser session or cookies into execution by default.
Keep credentials and sensitive browser evidence within the existing secret, capture, retention, and private-access contracts.
Retain normal timeout status, inclusive failure reruns, one logical result per test, timing observations, and explicit evidence gaps.
No hosted recorder or cloud adapter is required to verify the first local website journey.

Hosted recording remains a separate delivery after local execution works.
Choose a local companion or streamed browser design before promising hosted recording support.
Remote service localhost, worker localhost, and the user's browser localhost are different network locations.
Show target reachability and reject incompatible execution locations before creating an executable run.
Document container-to-host addressing and the selected target policy alongside the implementation.

### Lessons to retain in subsequent work

- Verify the normal tester and project-owner roles. Local administrator success alone does not prove usable access boundaries.
- A new URL can require environment management authority. Explain that requirement and retain the option to reuse an authorized target.
- Keep stage and protection independent. Do not infer safety from a hostname, HTTP method, or a name such as Production.
- Keep current redirect behavior explicit. HTTP probes reject unexpected redirect statuses and do not follow redirects automatically.
- Improve response and redirect diagnostics before adding a guarded redirect option. Never silently accept an unexpected response as a pass.
- Continue distinguishing observed test outcome, execution completeness, timing assessment, and evidence availability.
- Preserve saved resources and explain retry behavior after a partial save. Describe reload limits honestly.
- Keep the existing demo data and user's browser work intact. Refresh the demo without reseeding or closing unrelated sessions.
- Use observed workflow durations to improve estimates and timeout behavior. Do not invent completion percentages or dates from code size.

## Usability and documentation are continuing requirements

Keep the product easy to use as the feature set grows.
Apply the shared [UI contract](ui.md#usability-and-documentation-contract) to each changed workflow.
Use [Page help](help.md) for contextual guide structure and maintenance.
These requirements apply throughout implementation. They are not a final polish phase.
The current UI contract supersedes older navigation sketches and form layouts in historical checkpoints.
Keep one canonical pattern for each interaction. Improve and reuse it across related pages.

Use tester language and a clear task sequence. Preserve project, application, suite, and target context.
Use steppers for configuration with several decisions. Keep simple tasks short.
Generate identifiers, retain defaults, and put uncommon controls under named Advanced sections.
Keep protection, safety, enable state, and execution or delivery consequences visible before submission.
Preserve entries through Back, help, and lookup refreshes. Explain validation, conflicts, partial saves, and recovery.
Document supported workflows, defaults, dependencies, permissions, side effects, and current limits.
Update documentation in the same delivery as behavior. Keep visible labels aligned across the UI and guides.
Maintain the README guide index so testers, API users, operators, and contributors can find the correct instructions.

## Workflow completion checklist

Apply this checklist to the changed workflow before reporting it complete:

- The tester can find the task through the intended navigation and retain its project, suite, and target context.
- Common controls are visible. The stepper, Advanced options, defaults, review, and action labels follow the shared UI contract.
- Relevant loading, empty, denied, validation, conflict, and partial-save states explain the next available action.
- Current-page help and the affected user guide match actual behavior. API and operator documentation changes accompany affected contracts or operations.
- Supported behavior and known limits are explicit. Documentation links and guide keys resolve. Planned features do not appear as available.
- Targeted local checks cover the changed behavior. Record evidence and cleanup in the implementation ledger. Reuse relevant passing evidence.

For documentation-only changes, check consistency, references, and links. Do not rebuild the UI or repeat browser workflows.
Keep AWS, Kubernetes, and Azure verification separate from current local implementation work.
Report remaining usability or documentation gaps with the feature status.
These requirements do not add or renumber acceptance items. The stable ledger remains 57 items.
Documentation updates alone do not change verified acceptance totals.

## Continuing usability and documentation work

Use small improvements as each affected page changes. Do not start a broad rewrite or repeat unrelated tests.
Keep these follow-up tasks visible:

- Apply the shared patterns to remaining complex editors as their feature work proceeds.
- Reconcile route help, feature guides, API contracts, and operator instructions when their behavior changes.
- Use tester walkthrough feedback to improve project setup, quick checks, run review, and failure-evidence discovery.
- Document recorded browser execution with its implementation. Keep its present execution limits explicit until verified.
- Complete provider-specific installation, upgrade, backup, restore, and recovery instructions with the relevant deployment work.

## Immediate delivery: usable local demonstration

The user prioritizes a usable, demonstrable application on 2026-11-03.
Finish a small local demo delivery before adding more features from the complete product backlog.
The full objective and all 57 acceptance items remain required. They do not all gate the first local demonstration.
See demo-readiness.md for the immediate delivery checklist.
The current user priority is a guided walkthrough of the running local demo.
Keep that demo available while the user works through its pages.
The user requests grouped sidebar sections during the walkthrough.
The sidebar separates Workspace, Testing, Automation, Advanced testing, Project administration, and Installation administration.
Automation, Advanced testing, and administration collapse by default. The current page opens its section.
The user also requests Applications in the sidebar.
Applications now opens a dedicated project list and detail route with environment and linked suite views.
Those pages reuse the existing registration workflow and preserve the current data ownership model.
View tests opens a suite-filtered catalog. Create test preserves the selected suite through the type chooser.
See applications.md for routes, current access, and registration behavior.
The refreshed local demo serves a private UI snapshot and retains its existing database and evidence.
Future builds do not replace assets used by this running demo.
The user also requests in-app Playwright authoring during the walkthrough.
Import and sync tests now provides saved browser workspaces and the common preparation, review, and publication flow.
The editor generates package and manifest files. It keeps source editing separate from catalog metadata management.
See playwright-editor.md for the workflow, API, revision rules, and current isolated profile limits.
The user chooses browser recording first and requests a dedicated typed creation route.
Create test now opens a chooser for Playwright browser tests, HTTPS probes, and Go suites.
Playwright creation leads with native recording and reviewed generated source.
The demo enables the local desktop recorder. Hosted recording needs a separate companion or streamed browser.
See create-tests.md and browser-recording.md for the implemented flows and limits.
Retained Playwright bundles can also open as editable source copies without changing their original releases.
The user next requests steppers and progressive disclosure during the walkthrough.
Project setup now groups the first application, optional target, and empty suite behind a final review.
Resource registration uses Name, Options, and Review. HTTPS creation uses Name, Request, and Review.
Run creation uses Selection and Review. Named Advanced sections hold uncommon controls.
HTTPS Save and run explicitly publishes a snapshot without replacing the active release and starts a local run.
Continue through the active delivery sequence after this interface delivery.
The offline profile stays targetless. The separate single-origin browser profile now executes reviewed recorded drafts against registered environments.

Item **01** now passes a build from an isolated copy of the current source tree.
Go uses no workspace or local module replacements. A clean offline npm installation and Angular build also pass.
Evidence and cleanup are recorded in implementation.md.
The five-item local demonstration checklist is complete.
The explicit launcher, new examples, walkthrough, and targeted local verification are recorded in demo-readiness.md.
Suite settings and lifecycle controls also pass focused local verification after the demonstration delivery.
Project settings supports edits, revision conflicts, archive, restoration, and readable affected schedules.
Exact suite read and management checks protect both lifecycle routes and the existing PUT route.
Run history, artifact downloads, retained releases, and schedule pins survive archive and restoration.
The tester sidebar now exposes Suites beside Applications, the catalog, runs, and trends.
Schedules and Notifications use guided configuration under Automation.
Imports and retained releases stay under Advanced testing.
Quick HTTPS probes now accept a complete URL directly, including addresses without an explicit port.
The wizard reuses matching registered targets and preserves their protection.
Review lists new target configuration and any new application before explicit saving.
Follow the active delivery sequence above. Creator access and the first local recorded execution flow are complete.
Tester investigation now uses shared test and attempt scopes with readable failure and artifact cards.
Next, complete retention adapters and timing parity. Keep hosted verification separate.
The next implementation batches follow the full ledger without adding new gates to the usable demonstration.

## Current implementation checkpoint

The acceptance ledger has **43 verified, 14 open, 57 total** items.
Private trace viewing now passes targeted local verification. Acceptances **42** and **45** are complete.
Run details expose recorded attempts, steps, context, logs, screenshots, traces, and configured attachments without HTML reports.
The pinned installation-hosted viewer uses the separate artifact origin and short-lived grants with current project authorization.
Recorded resources replay from the archive. Expiry closes the view; refresh and authorized local downloads provide recovery.
See [Private trace viewing](trace-viewer.md) for setup, limits, isolation, and operator requirements.
STUDIO-TRACE-001 and focused access checks pass. The implementation ledger records exact evidence and cleanup.
Category navigation, metadata filters, stable sorting, and visible lifecycle actions now pass focused local verification.
Reviewed publication now shows changes, safety, policies, and readable schedule effects before explicit publication or activation.
Stale reviews and current permission loss reject publication. Same-key retry recovers a committed release after response loss.
Acceptance **25** is complete. Full source metadata preservation and revision conflicts pass focused local checks.
See [Publication review](publication-review.md) for API, user choices, expiry, and recovery.
Acceptance **21** is complete. The UI and API share authorized catalog filtering, sorting, counts, category branches, and opaque paging.
Complete JSON export, reviewed file import, conflict recovery, and normal creation, publication, and execution pass targeted local checks.
See [Catalog lists](catalog-list.md) for API queries, pagination, exact permission checks, and limits.
Acceptance **26** is complete. Bounded preparation recovery and terminal replay pass on SQLite and local PostgreSQL.
Acceptance **29** is complete. Retained history and private evidence remain available through archive and working-definition removal.
See [Retained test history](test-history.md) for tester steps, API permissions, paging, and retention limits.
Acceptance **43** is complete. Durable upload recovery preserves identities, reservations, and original run outcomes.
SQLite, local PostgreSQL, and the owned controller restart scenario pass. See [Evidence recovery](evidence.md#interrupted-upload-recovery).
Acceptance **46** advances. Filesystem, S3, and Azure Blob adapters now share the evidence service through operator-selected storage.
SDK protocol fixtures and common lifecycle checks pass locally. Hosted storage identities and lifecycle behavior remain unverified.
See [Artifact storage](artifact-storage.md) for configuration, migration, and remaining acceptance scope.
Acceptance **36** is complete. Go package events, aggregates, retries, and API steps preserve logical result counts.
Acceptance **41** advances with runtime binding, valid zero observations, and censored missing measurements on SQLite and local PostgreSQL.
The actual local runtime inspection and focused normal self-tests pass. Cross-deployment timing verification remains open.
See [Timing compatibility](timing.md#runtime-compatibility-and-zero-measurements) and [Logical counts](results.md#logical-count-boundaries). Acceptance **43** is complete. Durable upload recovery and repeated private evidence reads pass after local controller restart.
Acceptance **08** is complete. Private installation archives transfer SQLite and PostgreSQL metadata and native permission state without changing public identifiers.
Focused local checks preserve access, credentials, sessions, run history, grant high-water marks, and artifact references.
Damaged input rolls back and nonempty destinations reject restoration. Artifact files and runtime configuration require a separate offline cutover copy.
See [Installation backup and database transfer](data-transfer.md) for commands and limits.
Acceptance **03** is complete. Fresh server startup and restart preserve empty testing registries.
A selected native HTTP example publishes without loading omitted examples or creating a run.
The focused real-server check stops all owned processes. No browser or container is needed.
See [First installation](first-installation.md) for optional examples and [Self-testing](self-testing.md) for the focused check.
Acceptance **20** is complete. OIDC credential resolution and discovery failures exclude upstream private text from application errors.
Local canary checks verify server-only token exchange, empty-secret rejection, private provider data, and readable generic browser failures.
See [Provider credentials](providers.md#server-credentials) for the boundary and focused verification.
Acceptances **28** and **38** are complete. Atomic admission guards the reviewed suite, release, environment, and selected working tests.
Current lifecycle, network policy, permissions, protection, and dependency safety block queued dispatch and retries without rewriting snapshots.
Focused local SQLite and PostgreSQL checks preserve actual manual and scheduled HTTP runs, earlier attempt evidence, and large guarded selections.
Acceptance **37** is complete across implemented local runners. See [Dispatch safety](results.md#dispatch-safety-and-frozen-runs).
Acceptance **34** is complete. Manual and scheduled native HTTPS checks share normal result, evidence, assertion, TLS, and response-time history contracts.
Project and exact-origin private CA bundles preserve certificate validation and network policy. Trust rotation blocks queued work and separates timing history.
New local TLS fixtures pass on SQLite and PostgreSQL. The production UI and native server build pass without a browser or runner-image rebuild.
See [Private HTTPS trust](http-checks.md#private-certificate-authorities) for operator configuration and retained diagnostics.
Required native HTTP follow-ups remain authentication secret bindings, bounded permitted redirects, and bounded redacted response excerpts.
Acceptance **35** advances with method, path, payload, redirect, and request-budget enforcement in the existing trusted gateway.
The API starter endpoint and exact-revision preparation pass focused local checks. Protected API targets require reviewed read-only rules.
Older runner and gateway images fail capability checks. Source and runtime policy snapshots must match.
The API creation choice and Name, Request, Review wizard now use the canonical source editor.
One newly authored API suite passes actual protected manual and scheduled execution with normal history and trend counts.
The targeted authoring UI workflow passes focus, preservation, Help, discard protection, persistence, and narrow dark layout.
The API worker harness double-deletion is corrected; exact resource absence is verified without repeating successful execution.
Acceptances **35** and **37** are complete for their implemented local scope.
API diagnostic redaction now protects retained assertion text, steps, logs, and bounded JSON during normal execution and recovery.
API media and opaque files produce explicit omissions. Browser media remains available under its capture rules.
Environment credential bindings now support Bearer, Basic, and API-key headers through the trusted gateway.
Operator-approved project references require exact origins and intersect the selected network policy.
The environment and API creation wizards expose references and binding names without secret values.
Authenticated manual and scheduled runs retain normal results and redacted evidence.
Environment-backed Go API clients now share the registered gateway and normal result, evidence, timing, history, and trend contracts.
Focused authenticated manual and scheduled Go runs pass, including a large reviewed policy and direct-network denial.
The recorded website and targetless contract checks complete local scheduled runner parity under **37**.
See [Coded API request policies](api-testing.md) for the implemented foundation and current limits.
Continue remaining storage parity, source workflows, HTTP follow-ups, adapter retention, timing, and provider packaging work.

Bulk metadata edits now support reviewed category, tags, owner, title, description, safety, rationale, and condition changes.
The API and UI preserve source-owned definitions and use atomic revision-controlled preview plans.
Safety changes recheck current publishing permission. Stored previews recheck exact read and editing authority.
Focused catalog checks and STUDIO-BULK-METADATA-001 pass locally. The implementation ledger records evidence and cleanup.
Saved explicit selections and versioned filters now work through catalog API and UI controls.
Their previews pin a release and dependency closure before normal run creation.
Runs retain saved definition snapshots. Later active releases or saved selection edits cannot rewrite those runs.
Focused API checks and STUDIO-SELECTIONS-001 pass locally. The implementation ledger records evidence and cleanup.
The catalog now provides category navigation, combined metadata filters, stable sorting, and preserved selections with hidden-selection counts.
Common controls stay visible. Advanced filters and row More disclosures contain less frequent options.
Enable, Disable, Archive, Restore, Duplicate, and History pass the targeted STUDIO-BROWSE-001 workflow.
The UI now uses server-side filtering and snapshot-bound cursors. Exact test denies protect rows and every facet.
Import, export, lifecycle, and publication workflows pass current local verification. Acceptance 21 is complete.
OpenAPI 0.28 and the generated client describe the implemented catalog operations.
Item **23** now passes a real local repository snapshot with two suites beside shared application source.
Both suites build separate images, publish, and pass normal runs with one result and complete evidence.
Item **24** previously passes two-image publication and execution without restarting the server.
Item **27** now passes UI activation and rollback with immutable existing runs, future active selection, and explicit run and schedule pins.
The Suite releases page provides retained definitions, comparisons, immutable source links, conflict handling, and archived-suite history.
Focused catalog checks and STUDIO-RELEASES-001 pass locally. The implementation ledger records evidence and cleanup.

Go ZIP uploads, prebuilt images, and registered GitHub sources use durable preparation and the common catalog review.
The Import and sync tests UI manages repository settings, multiple bindings, manual sync, diagnostics, history, and publication policy.
The separate worker uses a provider-neutral port and repository-scoped GitHub App credentials for private source.
It pins a commit and repository identity, shares one archive, and retains exact source provenance.
Unchanged commits reuse work. Ready reviews refresh metadata without repeating assembly or discovery.
Generation, current authority, worker leases, and latest-sync guards fence stale work.
Source-object denies protect preparations, reviews, and retained archives.

Focused backend checks pass. The real two-suite local workflow passes in about 22 seconds.
The Angular build and targeted STUDIO-GIT-001 UI self-test pass.
The implementation ledger records exact evidence and cleanup.
The cached runtime remains available. Owned verification images, containers, volumes, and contexts are removed.
The self-test launcher reuses unchanged UI builds, binaries, and immutable runner images.
Use --test=ID to run only the affected workflow. Do not repeat the broad matrix without a specific integration concern.
No live GitHub or cloud deployment checks run for this milestone.

Signed push events and durable periodic sync now use the same source queue and current authority checks.
The Import and sync tests UI manages both policies, next due time, webhook address, and trigger history.
Focused signature, delivery replay, delayed push, scheduler contention, coalescing, and revocation checks pass locally.
Offline Playwright bundles now use the same builder, discovery, publication, and execution workflow.
A real local browser bundle and the expanded STUDIO-BUNDLES-001 self-test pass.
Workers accept separate Go and Playwright runtimes and leave other runtime work queued.
Browser-confirmed publisher login now supports local and OIDC accounts through the ordinary browser sign-in screen.
It issues one independent, proof-bound managed CLI session and retains current authorization and expiry bounds.
The actual CLI binary passes STUDIO-CLI-001 with local and local-fixture OIDC accounts, denial, help, and mobile checks.
See cli-sign-in.md and the implementation ledger for exact evidence and continuation details.
Acceptance 22 now passes complete GitHub-to-UI publication against a local TLS provider fixture. Continue explicit dependency profiles only when required by a supported execution profile.
The cached browser runtime is teststudio/playwright-source-runtime:local. The cached Go runtime is teststudio/go-source-runtime:local.
Do not repeat the passing real Go or Playwright image integrations without a builder or execution contract change.
Use the targeted source checks or affected self-test selector after implementation changes.
Source-management milestone commits include 6b5a9f1 and e1354ef. Use git log for the current feature history.
The current source builder supports offline Go and Playwright bundles with trusted runtimes and fixed copy instructions.
Items **22**, **25**, **26**, and **29** advance but remain open for their broader scope.
Items **21** and **29** also advance through retained release controls.
Keep the full 57-item plan required. Do not narrow the goal to this milestone.
Implement coherent batches, then run the smallest relevant local checks. Reuse existing evidence.
AWS, Kubernetes, and Azure verification remain separate work.

## Historical shutdown checkpoint and continuation

The user requests the earlier pause to power off the computer.
The user resumes this task with the implementation-first direction and active delivery sequence above.
This checkpoint records earlier work. Do not resume its old numbered steps as the current task queue.
The complete product plan and all acceptance criteria remain required.

### Checkout and verified completion

- Repository: `/Users/johnbauer/Dev/Personal/teststudio`.
- Branch: `codex/implementation`.
- Latest committed milestone: `0451138`, native JSON/YAML uploads with atomic catalog publication and retained source snapshots.
- Acceptance ledger: **22 verified, 35 pending, 57 total** in `implementation.md`.
- The inspection work below remains intentionally uncommitted. Preserve it when resuming.
- The environment can open in the Mapag checkout. Always use the explicit TestStudio working directory.

### Current saved work

The working tree contains a prepared-suite inspection milestone:

- `internal/definitions`: shared bounded JSON/YAML decoding. JSON scanning rejects duplicate keys before typed decoding.
- `internal/suitespec`: versioned manifests, origin metadata, safe file reads, catalog matching, and discovery report validation.
- `contracts/suite.schema.json` and `contracts/source-origin.schema.json`: canonical manifest and provenance schemas.
- `internal/preparation`: an artifact inspection port separate from test scheduling and future source builds.
- `adapters/execution/localcontainer/discovery.go`: isolated inspection, bounded report collection, ownership checks, and scoped cleanup.
- Go runtime discovery: compile declared packages and inspect runnable source declarations without executing initialization, `TestMain`, or test bodies.
- Go discovery marks dynamic child selectors as `parentOnly`. Execution must still prove that the selected child exists.
- Go normal execution accepts prepared module roots and retains the existing catalog format for earlier TestStudio suites.
- Playwright discovery: list tests through a dedicated reporter, retain inherited tags, and select one exact project.
- Playwright normal execution uses the prepared module and configuration when a suite manifest exists.
- `cmd/teststudio-publish`: local `inspect` and `cleanup` commands. Server publication is **not implemented** by this CLI yet.
- The CLI saves private cleanup references before dispatch. Inspection results use new private files and never overwrite existing paths.
- New independent fixture images and `scripts/test-discovery.sh` exercise both runtimes and their normal execution paths.
- `.dockerignore`, the self-test image, README, and prepared-suite guides contain the related updates.

The local inspector accepts only enabled immutable images already present in Docker.
It supports `go-module` and `playwright-node` without request policies or execution bindings.
It does not pull images, install dependencies, or build containers inside the API.
Image-provided source provenance is recorded, not independently attested against a remote repository.
The API does not call the inspection port yet.

### Verification evidence

These checks pass for the saved work:

```sh
go test -race ./internal/definitions ./internal/suitespec ./internal/sources ./runners/go/runner ./cmd/teststudio-publish ./adapters/execution/localcontainer
bash scripts/test-discovery.sh
```

The standalone Go command skips container checks unless the fixture image variables are set.
The discovery script builds the fixture images and runs those container checks with actual workers.
It verifies both runners, module roots, source locations, project selection, isolation, failure cleanup, and CLI output privacy.
It also executes inspected Go and Playwright fixtures through the existing managed execution contract.
Go checks reject missing declarations, lower-case helper names, `TestMain`, non-running examples, and compilation failures.

Latest retained CLI evidence:

- `.data/discovery-verification.WqyUvr/go.json`: `inspected`, two catalog entries.
- `.data/discovery-verification.WqyUvr/playwright.json`: `inspected`, two catalog entries.
- Both inspection runs finish successfully on 2026-11-03.

At shutdown, no owned inspection processes, discovery containers, or discovery volumes remain.
The checks include the `discovery-tests` and `cli-discovery-checks` project labels.
No remote push or cloud deployment occurs in this checkpoint.

### Exact continuation steps

1. Read this checkpoint and `AGENTS.md`. Check the working tree before editing. Preserve all saved inspection files.
2. Add a real in-progress cancellation check. Current cancellation coverage includes cancellation before dispatch.
3. Verify that a duplicate inspection reference cannot stop or remove an existing inspection owned by the first caller.
4. Run the smallest checks for any changes. Use `scripts/test-discovery.sh` for real container verification and scoped cleanup.
5. Run the existing source workflow regression: `bash scripts/test-self.sh --test=STUDIO-SOURCES-001`.
6. Verify native source and transfer API behavior after the shared decoder extraction, including local PostgreSQL where relevant.
7. Update `implementation.md` with the completed inspection milestone and precise evidence. Commit verified work locally; do not push.
8. Implement durable prebuilt preparation jobs, cancellation, recovery, project-scoped artifact admission, and retained diagnostics.
9. Connect successful server inspection to the existing immutable source snapshot, catalog, release, and explicit activation contracts.
10. Add actual CLI publication and the corresponding API/UI workflows. Do not treat client inspection files as trusted server verification.
11. Allow test-only publication without restarting or deploying the API/UI for each image. Keep artifact authority scoped to its project.
12. Continue ZIP and GitHub preparation through a separate isolated build backend. The inspector itself is not a source builder.
13. Continue the full acceptance ledger, including metadata ownership, evidence, timing parity, database transfer, authorization, recovery, and deployment targets.

Keep initial functional verification local.
Use the authorized MMDev account for later AWS verification.
AWS ECS/CloudFront, Kubernetes, Azure Container Apps, and local-server acceptance remain required.
Do not mark broad source, deployment, or independence criteria complete from the inspection checks alone.


TestStudio is an independent testing product for teams with multiple projects.
The repository name is `github.com/wtiger001/teststudio`. Private visibility remains the proposal.
The repository contains the Go server, Angular UI, contracts, generic runners, and deployment definitions.
The product name is TestStudio. The repository and command name is `teststudio`.
The UI hostname remains an open decision. The GitHub owner is `wtiger001`.
Consumer tests can remain in their own application repositories or separate suite repositories.
No second product repository is required initially.

### Product independence

Start with an empty project registry and fresh infrastructure configuration.
Copy no existing tests, fixtures, test catalogs, credentials, environments, schedules, reports, or history.
Include only newly written generic examples and product validation tests.
MarketMaker is not a required first project, dependency, deployment, or acceptance target.

Use the current system as a behavioral reference for generic workflows.
Any reused product implementation must have no application-specific dependencies.
Do not depend on MarketMaker modules, shared libraries, authentication, notification transports, or deployment scripts.
Do not inherit application identifiers, permission groups, account identifiers, AWS profiles, hostnames, or sibling checkout paths.
Do not add legacy record adapters, catalog aliases, redirects, compatibility wrappers, or a MarketMaker cutover workflow.
The existing MarketMaker deployment remains a separate system outside this plan.
The distribution provenance audit under acceptance 02 is complete. See [Independent product material](product-provenance.md) for evidence and audit limits.

Applications, suites, endpoints, stages, credentials, permissions, and notification policies are configuration.
Use provider-neutral domain models and service contracts.
Cloud resource identifiers belong only in provider configuration and adapters.

### Deployment targets

Support AWS, Kubernetes, Azure, and a local server as required deployment targets.
Each target runs the complete product, including UI, API, scheduling, results, and private artifacts.
Each target supports isolated suite execution through its execution adapter.
A deployment target hosts the service. An execution location hosts a test worker.
A tested environment identifies the application under test. Keep these three concepts separate.
A remotely hosted service can register a local worker without moving its own deployment.

| Target | Go service and UI | Isolated suite execution | Artifacts and secrets |
| --- | --- | --- | --- |
| AWS | ECS service. Private S3 UI origin behind CloudFront. Route API requests to ECS. | ECS tasks with configured resource limits and identities. | S3 artifacts and Secrets Manager references. |
| Kubernetes | Deployments, Services, and configured ingress or Gateway. Serve static UI through its own container or the Go service. | Kubernetes Jobs with configured service accounts and resource limits. | Configured object storage or persistent volumes. Secret references use the cluster integration. |
| Azure | Container Apps service. Serve UI from the service or a separate static distribution. | Container Apps Jobs started through the execution adapter. | Blob Storage artifacts and Key Vault references. |
| Local server | Native Go service or Docker Compose. Serve the built Angular UI locally. | Local container worker. Registered native execution is an explicit trusted option. | Local persistent directories and local secret bindings. |

Azure Container Apps is the selected planning baseline for Azure. Its deployment implementation and acceptance remain open.
It fits the continuous Go API and scheduler without requiring a Functions-specific service architecture.
Use Container Apps Jobs for finite isolated executions.
Azure Functions is not required for the baseline. Evaluate it separately if a later event handler needs it.
Keep at least one scheduler-capable instance active when schedules are enabled.
A static distribution is optional outside the required AWS CloudFront profile.

Define ports for durable metadata, artifacts, secrets, identity, builds, and execution.
Keep provider SDK types outside domain models and shared contracts.
Local, Kubernetes, and Azure deployments require no AWS account, credentials, or services.
The local server supports durable runs, schedules, results, and reports without a cloud connection.
External Git imports and dependency downloads can still require network access.

Use PostgreSQL for hosted and multi-instance deployments.
Use SQLite as the default for development and a single-instance local server.
Local deployments can select PostgreSQL when they need the hosted database configuration.
Both databases support the same catalog, run, schedule, result, and evidence contracts.
Select managed or self-hosted PostgreSQL through deployment configuration.
Artifact interfaces support S3, Azure Blob Storage, and filesystem storage.
Keep private artifact authorization independent of CloudFront signing.
Use authorization checks before issuing a provider-specific download grant or serving a local artifact.
Serve generated HTML reports on a separate origin from the authenticated UI.

Use one scheduler implementation with durable claims and execution leases across all targets.
Replicas can recover abandoned work without starting duplicate logical runs.
Provider retries must reconcile an existing execution before starting another worker.
Application schedules do not require EventBridge, Kubernetes CronJobs, or Azure timer functions.
See [Durable schedules](schedules.md) for the implemented local scheduler, project interface, and atomic firing contract.
Configure concurrency by project, execution location, and workload.
Define environment exclusivity explicitly. Frequent health probes must not inherit a global browser-run lock.

### Required behavior

Provide these generic workflows:

- Manual execution, scheduled execution, cancellation, deadlines, and failed-test reruns.
- Durable run records, configurable execution limits, exclusive-run locks, and idempotent requests.
- In-process Go scheduling. Do not introduce EventBridge.
- Immutable suite releases and run snapshots.
- Suite publication without a testing API deployment.
- Progress polling every five seconds, provisional counts, and runner heartbeat reporting.
- Private artifacts, screenshots, result details, and access checks on every deployment target.
- Full-width result cards under collapsible categories.
- Application, environment, stage, category, test, and date filters.
- Stacked area charts for per-run result states, with visible gaps for missing history and explicit incomplete results.
- Most failed and most volatile test rankings.
- TestStudio-owned Playwright self-tests that run through its normal catalog, release, execution, timing, and evidence workflows.
- Durable completion notifications with retry handling.
- Production safety labels, request guards, and analytics suppression.

Safety labels remain `safe`, `unsafe`, and `needs-review`.
Safety labels do not require separate approval records or renewal steps.
Protected targets accept only tests labeled `safe`, including their dependencies.
Project-specific request guards continue to enforce the declared operating conditions.

### Domain model

Use the following ownership structure:

```text
Project
  Applications
    Environments
  Suites
    Catalog drafts and test revisions
    Releases
      Test catalog
      Runner configuration
  Test sources
  Builds
  Runs
    Attempts and steps
    Events and log streams
    Evidence manifests and artifacts
  Schedules
  Notification settings
```

A project groups related applications, suites, history, and settings.
An application has an arbitrary permanent identifier and a display name.
An environment belongs to an application and has a permanent identifier, display name, endpoints, stage, and protection setting.
Display names such as Production can repeat across applications.
A suite belongs to a project and declares the applications it supports.
A test source belongs to a project and supplies uploaded content, a registered repository, or UI-managed definitions.
One repository can supply multiple suites through separate module roots, test paths, and manifests.
A build records source preparation and artifact validation. It is separate from a test execution.
A release identifies its catalog, source revision, adapter, execution configuration, and immutable image or bundle.
A run freezes its project, suite release, test selection, environment, endpoints, and execution configuration.
Later configuration edits do not change that snapshot.

Separate test type from runner in the new contracts.
Type describes the check: browser, HTTPS health, API, unit, integration, accessibility, visual, or performance.
Runner describes its implementation: Playwright, Go, or native HTTP.
A Go API test therefore has type `api` and runner `go`.
An execution location is separate from both fields. It identifies an ECS, Kubernetes, Azure, or local worker.

Suites declare whether they require a target environment.
Pure unit suites require no web URL or API credentials.
Their runs appear under All and an explicit No target filter, rather than a production stage.
API and integration suites declare their required endpoint and credential bindings.

Test identity includes project, suite, and permanent test identifier.
Browser or framework variants remain separate result dimensions.
Two projects can use the same local test identifier without sharing history.
Test identifiers remain stable across releases when they represent the same check.

Stages include production, beta, development, local, and custom values.
Protection is independent of stage.
A development application that reads production data can require production-safe tests without sending production-stage notifications.
Environment records store explicit stages. Do not infer stages from their names.

### Users, groups, and permissions

Use `github.com/wtiger001/go-permissions` as the authorization engine.
Pin a reviewed library version or commit and verify its Go toolchain requirements before implementation.
Use `permissions.Service` for permission checks, role management, assignments, grants, and effective-permission reports.
Do not create a second authorization evaluator in TestStudio.
The library handles authorization. TestStudio handles identities, group membership, sessions, local sign-in, and OIDC sign-in.

#### Users and groups

Store permanent internal user identifiers independently of email addresses and identity providers.
An external identity uses the exact OIDC issuer and subject pair as its unique key.
A user can link multiple external identities through an authenticated, explicit account-linking workflow.
Require fresh authentication and proof of control of the identity being linked.
Do not link accounts automatically because their email addresses match.
Store display name, email verification state, lifecycle state, creation time, and last sign-in time.
Disabling a user blocks new requests and sessions while preserving ownership, run history, and audit records.

Use direct project membership instead of invitation messages, acceptance links, or invitation expiry.
Project owners with membership authority add existing users and choose permitted project roles.
Support pending project access for users who have not signed in.
A pending access record stores an allowed provider, verified email requirement, project, and permitted role.
Apply pending access only after that provider confirms the verified email during sign-in.
Do not use pending access to link existing internal accounts or grant installation administration.
Audit pending access creation, cancellation, activation, and the resulting role assignments.
Support configurable user provisioning at first OIDC sign-in.
Successful sign-in alone grants no project access.
Do not grant installation administration from an email domain or first-login order.

Groups have permanent identifiers, names, descriptions, ownership scope, lifecycle state, and membership provenance.
Support local groups and groups populated by explicit OIDC claim mappings.
A user can belong to multiple groups. A group can receive different role assignments in different authorized projects.
Use flat user-to-group membership initially. Nested directory groups require an explicit resolver with cycle and expansion limits.
Local membership edits and external synchronization remain distinguishable in the UI and audit history.
Externally managed membership is read-only in ordinary local membership editors.
Archiving a group removes its effective access without erasing past assignments or audit records.

Installation administrators manage installation-owned users and groups.
Project administrators manage only project-owned groups and project assignments they can delegate.
Project-owned groups cannot receive assignments outside their project.
Membership changes to shared installation groups require installation-level authority.

Implement the library's `IdentityProvider` interface through TestStudio's identity repositories.
Resolve `GetUserGroups`, `GetGroupMembers`, and `IsUserInGroup` from authoritative active membership records.
Implement `GetUserTeams` and `IsUserInTeam` for the pinned interface without inventing implicit project membership.
Initially return no team membership unless a team integration is explicitly configured.
Use user and group principals for initial role assignment workflows.
The permission library does not own or synchronize the user directory.

#### Permission scopes and evaluation

Use installation scope for user administration, provider configuration, global workers, and product settings.
Map project identifiers to the library's `Request.TeamID` for project checks.
Map a resource to a canonical object identifier containing project, resource type, and permanent resource identifier.
Load the resource's authoritative project before constructing an authorization request.
Do not trust a browser-supplied project identifier to establish resource ownership.
Object-scoped grants are explicit. Parent access does not imply descendant access without a documented grant.

Register stable permission identifiers in the `teststudio` namespace.
Register predefined role identifiers with the required `builtin.` prefix.
Project role grants use explicit project bindings rather than wildcard project scopes.
For reusable project roles, validate `?team` binding resolution through the pinned library's grant and assignment contracts.
Keep missing bindings and unauthorized wildcard scopes as errors.
Custom roles contain only permitted registered actions and validated installation, project, or object scopes.

| Area | Proposed permission identifiers |
| --- | --- |
| Catalog | `teststudio.catalog.{read,create,update,import,export,publish,archive,restore,delete}`. |
| Execution | `teststudio.runs.{read,create,cancel,execute-protected}`. |
| Evidence | `teststudio.evidence.{read,download,retain,delete}`. |
| Schedules | `teststudio.schedules.{read,create,update,delete}`. |
| Project configuration | Separate read and manage permissions for environments, sources, credential bindings, notifications, and project settings. |
| Access management | `teststudio.members.{read,manage}`, `teststudio.roles.{read,manage}`, and `teststudio.permissions.grant`. |
| Installation | Separate manage permissions for users, installation groups, OIDC providers, global workers, and system settings. |

Braces expand into separate permission identifiers. Use full identifiers in contracts and stored grants.
Credential-binding permissions expose references and configuration, not resolved secret values.
Protected execution requires both ordinary run creation and the protected-execution permission.
It also requires the suite safety policy. No role bypasses safety labels or request guards.

| Predefined role | Default responsibilities |
| --- | --- |
| `builtin.teststudio.viewer` | Read authorized catalogs, results, trends, and evidence. |
| `builtin.teststudio.operator` | View, start and cancel runs, and manage ordinary schedules. Protected execution requires a separate grant. |
| `builtin.teststudio.editor` | Edit catalog drafts and declarative definitions. Publication requires a separate grant. |
| `builtin.teststudio.publisher` | Publish and activate validated releases and manage registered test sources. |
| `builtin.teststudio.project-admin` | Manage one project's settings, members, groups, and permitted role assignments. |
| `builtin.teststudio.installation-admin` | Manage installation identity, providers, workers, and explicitly configured installation-wide grants. |

Built-in definitions remain code-managed. Custom roles, assignments, grants, and disabled-role overrides are persistent.
Do not give `builtin.public` or `builtin.authenticated` project permissions by default.
Use a dedicated internal administrator group for any configured synthetic administrator role.
Do not bind an arbitrary external group claim directly to synthetic administration.
Retain the library's explicit-deny precedence and expired-grant behavior.
Fail closed on permission-store or identity-resolution errors.
Use authentication middleware to identify callers, then check `HasPermission` inside handlers or domain services after parsing resource context.
Apply checks to lists, searches, pagination, counts, charts, exports, downloads, and mutations.
Filter unauthorized projects before computing aggregate results.
The UI displays server-computed capabilities; it does not infer authority from an OIDC group name.

Granting roles requires explicit delegation authority for the target project, role, and scope.
Project administrators cannot grant installation roles, widen bindings, or change shared groups to elevate their own access.
Validate role inheritance and scoped bindings before enabling custom inheritance workflows.
Permission checks remain authoritative even when the UI hides an action.

Refresh authorization after membership, assignment, grant, or user-state changes.
Invalidate local caches immediately and propagate changes across instances within a documented maximum of five seconds.
Verify library caches and product caches together. Do not rely on a new OIDC login to remove local access.
User disabling and session revocation take effect on the next authenticated request.
Existing short-lived artifact grants expire within their configured lifetime.
Schedules record an execution principal and recheck its authority before dispatch.
Revocation blocks future starts without rewriting past run snapshots.

#### Access management API and UI

Expose identity and access management through the same versioned API used by the UI.
Use `/api/v1/me` for the current profile and linked identities.
Expose effective capabilities and permission explanations for the current authorized project and resource context.
Restrict inspection of another user's effective access to authorized access managers.

| Route family | Purpose |
| --- | --- |
| `/api/v1/users` | List, create, inspect, disable, and restore users with installation authority. |
| `/api/v1/groups` | Manage permitted groups and their local memberships. Show synchronized membership separately. |
| `/api/v1/roles` | List predefined roles and manage authorized custom roles. |
| `/api/v1/projects/{projectId}/members` | Inspect project access and manage scoped member or group assignments. |
| `/api/v1/projects/{projectId}/role-assignments` | Assign and revoke permitted roles with validated project and object bindings. |
| `/api/v1/projects/{projectId}/permissions` | Inspect effective grants and manage permitted explicit grants or denies. |
| `/api/v1/identity-providers` | Configure OIDC providers and explicit external-group mappings with installation authority. |

Document single-assignment revocation and grant removal operations in OpenAPI.
Use revision checks and audit records for access changes, including actor, subject, scope, prior value, new value, and time.
Prevent an ordinary UI action from removing the last usable installation administrator.
Provide an operator-controlled recovery procedure independent of a failed provider or group mapping.

Add Users, Groups, Roles and Permissions, and Sign-in Providers to authorized Settings pages.
See [Identity administration](access.md) for implemented user, group, membership, profile, and session APIs, revision rules, and local recovery.
User details show linked identities, local and external groups, project assignments, effective permissions, and active sessions.
Group details show membership origin, affected projects, and role assignments.
Role editors show action descriptions, scope bindings, explicit denies, and an effective-access preview.
Show the reason for an access decision, including its role or group source, to authorized managers.
Return distinct unauthenticated and unauthorized responses without exposing another project's private data.

### Local and OIDC sign-in

Each installation can enable local accounts, OIDC providers, or both.
Local accounts and external identities use the same internal user directory, sessions, groups, and project permissions.
An authorized installation administrator creates local credentials for an internal user.
Store password hashes on the server with a maintained password-hashing implementation and per-account salts.
Never return password hashes or credentials in directory responses, audit records, or logs.
Support password changes, administrator resets, required changes after a reset, and disabling local sign-in for one account.
Password reset and credential disabling revoke affected sessions.
Apply bounded sign-in attempts, generic credential errors, and request-forgery protection to local authentication.
Local account creation does not automatically grant project roles.
The sign-in screen presents the enabled local and provider choices.
Linking local and external sign-in methods requires fresh authentication and proof of control.
Matching email addresses do not link accounts.
Protect the last usable administrator when local credentials, providers, user state, or access assignments change.
Operator recovery remains explicit and distinct from ordinary account management.

### OIDC sign-in and sessions

Support one or more explicitly configured OpenID Connect providers on every deployment profile.
Use provider discovery from an allowed issuer and support configurable scopes and claim mappings.
Configure each provider's issuer, client identifier, server-side credential reference when required, callback, and logout settings.
Providers can include enterprise or self-hosted installations. No provider or tenant is hardcoded.
A provider outage does not require changing catalog or permission records.

Use the authorization code flow with PKCE S256, state, and nonce validation.
The Go service handles the callback and validates issuer, audience, signature, expiry, nonce, and token-use requirements.
Use verified discovery metadata and signing keys with bounded refresh and key-rotation handling.
Validate the provider and callback binding before exchanging an authorization code.
Do not accept an ID token as a general API access token.
Keep provider secrets, refresh tokens, and access tokens on the server.
The browser receives an opaque session cookie rather than provider tokens in browser storage.

Use HttpOnly cookies, Secure on HTTPS deployments, and SameSite settings appropriate for the callback flow.
Protect cookie-authenticated mutations against cross-site request forgery.
Rotate the session after sign-in and privilege changes.
Store session expiry, revocation, authentication provider, and user status durably in the selected database.
Support idle and absolute expiry, current-session logout, and administrator session revocation.
Clear the local session even when provider logout fails. Provider logout is an explicit optional integration.
Keep callback redirects and post-login return routes within configured destinations.

Identity-provider sign-in establishes the internal user identity, not its project roles.
Apply configured external-group mappings only from validated claims for that issuer.
Use stable provider group identifiers when available and record mapping provenance.
An unmapped external group grants no access. External role strings are not executable permission names.
Distinguish complete membership claims from missing, partial, or provider-overflow group responses.
Use an explicit provider integration for additional membership retrieval when required.
Never preserve outdated elevated external membership as a silent fallback after an incomplete refresh.
Recheck authoritative external membership on a configured refresh interval and impose a maximum membership age.
If required membership cannot be refreshed, deny dependent access or require sign-in again.
Local group membership remains independent of provider-managed membership.

Expose `/auth/providers`, `/auth/login/{providerId}`, `/auth/callback/{providerId}`, `/auth/session`, and `/auth/logout`.
Publish only provider display information and enabled login choices to unauthenticated clients.
The sign-in screen shows configured providers and readable cancellation, failure, and expired-session states.
Preserve a permitted return route and show Access not granted when authentication succeeds without project access.
Use the same login and permission contracts for AWS, Kubernetes, Azure, and local hosted deployments.
Static UI hosting routes authentication and session API requests to the Go service without caching private responses.

Provision the initial administrator through an operator-controlled bootstrap bound to an exact issuer and subject.
An explicit loopback-only development mode can create a local principal without an external provider.
That mode still uses the permission engine and configured local groups. It never silently activates for remote clients.
Shared or remotely accessible local servers use configured local credentials, OIDC, or both.
Automation clients use revocable, expiring project-scoped credentials. Workers retain separately scoped run-ingestion credentials.
Neither credential type reuses a browser cookie or receives installation administration by default.

### Database and durable storage

PostgreSQL stores catalog revisions, releases, run snapshots, attempts, steps, schedule claims, artifact metadata, and audit events.
It also stores users, linked identities, memberships, sessions, custom roles, assignments, grants, and provider configuration references.
SQLite stores the same logical records for development and single-instance local operation.
Keep database access behind focused repositories with shared domain rules.
Use versioned migrations for each database dialect.
Use the library's PostgreSQL permission store within a product-owned schema and a controlled migration lifecycle.
Implement a SQLite adapter for the pinned `permissions.PermissionStore` interface.
The reviewed library provides PostgreSQL storage; this plan does not assume a built-in SQLite backend.
Both stores use the same `permissions.Service` evaluator and TestStudio identity adapter.
Persist role inheritance, assignment bindings, grants, expiry, and disabled built-in overrides on both databases.
SQLite mode does not fall back to permissive checks or lose assignments on restart.
Add shared store checks for explicit denies, expiration, revocation, scope binding, and built-in role overrides.
Do not require PostgreSQL extensions for the core product.

Use SQLite WAL mode on a local persistent disk, with bounded transactions and a configured busy timeout.
Support one local service instance. Workers communicate through the service API rather than opening its database file.
Use PostgreSQL for multiple API or scheduler instances.
Do not place a SQLite WAL database on a shared network filesystem.
Verify revision conflicts, idempotency, execution claims, lease recovery, and retention behavior against both databases.
Use atomic conditional updates for claims. Keep provider-specific locking outside domain services.

Store screenshots, traces, videos, reports, attachments, and log segment bodies in configured artifact storage.
Store their ownership, content type, size, checksum, status, capture policy, and storage reference in the database.
Logs use immutable ordered segments with a searchable metadata index and cursor-based reads.
Do not put large binaries or unrestricted log bodies in relational rows.
The local profile stores artifacts under its persistent data directory.
PostgreSQL operation does not require object storage when a filesystem adapter meets the deployment requirements.

Back up database state and artifact manifests with documented consistency rules.
Test restoration with artifact checksums and missing-file detection.
Provide an explicit export and import path from SQLite to PostgreSQL when the installation grows.
Preserve identifiers, revisions, run snapshots, audit history, and artifact references during that transfer.
Changing the database connection alone does not move existing data.

### Catalog management through the API and UI

The service owns the test catalog and exposes all catalog workflows through its versioned API.
The Angular UI uses that same API. CLI publishers and integrations use the same contracts and permissions.
A project has a searchable catalog across its suites.
A suite owns permanent test identities, catalog drafts, published revisions, and release membership.
Use the project, suite, and test identifier together for identity.

Each test records these fields:

- Permanent identifier, display title, description, application, suite, category, tags, and optional owner.
- Test type, runner, source origin, source revision, and executable selector or declarative definition.
- Required endpoint and credential bindings, dependencies, timeout, and registered execution profile.
- Safety label, rationale, operating conditions, lifecycle state, and capture policy.
- Revision, creation and update timestamps, actor, and optional external reference links.

Treat latest outcomes, failure rates, and artifact counts as derived observations, not editable catalog metadata.
Keep catalog revision separate from the latest execution result.

#### Editing and publication

Create and edit catalog entries in a draft.
Declarative HTTPS and API checks use structured editors and API definitions.
Code-backed tests reference discovered selectors from a pinned source snapshot or immutable artifact.
Creating a code-backed catalog entry does not create executable test code.
Use the source workflow to add or change its implementation.

Keep discovered source facts separate from service-managed metadata.
Record the authority for each editable field. Show source-owned fields as read-only with a source link.
Store service-managed titles, categories, tags, owners, and descriptions as versioned metadata.
A source synchronization previews additions, removals, and changes before applying the configured publication policy.
Do not silently overwrite service-managed metadata during discovery.
A safety change requires a rationale and publishing permission. It does not add a separate approval workflow.
Reassess safety when source, helpers, dependencies, or request policy changes.

Validate identifiers, selector mappings, definitions, dependency existence, dependency cycles, required bindings, and permitted execution profiles.
Code-backed publication verifies that referenced tests exist in the prepared artifact.
Return field-specific errors and a release comparison before publication.
Use revision tokens and If-Match for edits and bulk mutations.
Reject stale edits with a conflict response rather than losing another user's changes.

Publishing freezes the catalog revision and its execution artifact in an immutable release.
Activation selects a ready release. Failed validation leaves the active release unchanged.
Metadata edits do not silently change a published release or a historical run.
New runs record the exact release, catalog revision, selected identifiers, definitions, and effective capture policy.

Support draft, enabled, disabled, and archived states.
Archive removes a test from normal selection while preserving its identity, revisions, history, and evidence.
Restore preserves the identifier. Duplicating a test creates a new identifier and independent history.
Disabled or archived tests cannot start new attempts, including through an old release or queued request.
Recheck admission immediately before dispatch. Report blocked selections and affected schedules explicitly.
An already started attempt follows the cancellation policy; a catalog edit does not silently terminate it.
Permanent deletion is a separate administrator action and cannot remove records required by retained runs or releases.

Saved selections can contain explicit identifiers or a versioned filter expression.
Preview the resolved selection and freeze its identifiers when creating a run.
Show removed or disabled dependencies as configuration errors instead of silently omitting them.
Bulk edits preview affected records and use atomic changes within the documented request limit.
Return explicit validation failures. Do not report partially applied edits as complete success.
Record actor, revision, timestamp, and changes for create, edit, import, publish, disable, archive, and restore operations.

#### Catalog API

Use `/api/v1/projects/{projectId}` as the project route prefix.
Use `/suites/{suiteId}` below that prefix for suite-owned operations.
The exact schemas and operation identifiers belong in OpenAPI.

| Route below the project prefix | Operations |
| --- | --- |
| `/tests` | Search and paginate the authorized project catalog across suites. |
| `/suites/{suiteId}/tests` | List or create suite test identities and draft entries. |
| `/suites/{suiteId}/tests/{testId}` | Read details, patch an editable draft with If-Match, and inspect history links. |
| `/suites/{suiteId}/tests/{testId}/revisions` | Read retained revisions and compare definitions or metadata. |
| `/suites/{suiteId}/tests/{testId}/actions/{action}` | Disable, enable, archive, restore, or duplicate with revision checks. |
| `/suites/{suiteId}/catalog-drafts` | Create, inspect, and revise a catalog draft. |
| `/suites/{suiteId}/catalog-drafts/{draftId}/validate` | Validate the complete draft and preview resolved execution bindings. |
| `/suites/{suiteId}/catalog-drafts/{draftId}/publish` | Create an immutable release with an idempotency key. |
| `/suites/{suiteId}/catalog-imports` | Validate and preview an import, then apply an explicit import plan. |
| `/suites/{suiteId}/catalog-exports` | Export a versioned catalog definition and metadata without secret values. |

Expose bulk edits and saved selections through documented operations on the same API.
Use deterministic sorting and opaque cursors for catalog lists.
Filters include text, suite, application, category, tag, type, runner, lifecycle state, safety label, and source.
Text search includes identifier, title, and description.
Allow optional latest-outcome filters without treating tests with no runs as passing.
Separate viewing, editing, publishing, running, source management, and destructive administration permissions.
Apply project authorization to every detail, revision, import, export, and history operation.

#### Catalog UI

Provide table and category-tree views with search, filters, sorting, saved selections, and bulk actions.
Show draft changes and the active release separately.
Test detail shows metadata, definition or source, revisions, dependencies, safety, latest result, history, and evidence links.
Provide Create test, Edit, Duplicate, Disable, Archive, Restore, Import, Export, Validate, and Publish actions according to permissions.
Show field validation beside its control and stale-edit conflicts with reload or review choices.
The publication screen compares additions, edits, removals, safety changes, and affected schedules.
Keep code source editing in the registered source workflow. The catalog UI manages metadata and declarative definitions.

### Repository structure

Keep the generic service and its dedicated UI in one repository.
Consumer test suites remain outside the product except for newly written generic examples.

```text
teststudio/
  cmd/
    teststudio/
  internal/
    domain/
    identity/
    authorization/
    sessions/
    projects/
    applications/
    environments/
    suites/
    catalog/
    evidence/
    sources/
    builds/
    healthchecks/
    runs/
    schedules/
    results/
    analytics/
    notifications/
    httpapi/
  adapters/
    auth/oidc/
    permissions/postgres/
    permissions/sqlite/
    sources/github/
    builds/
    execution/ecs/
    execution/kubernetes/
    execution/azure/
    execution/local/
    storage/postgres/
    storage/sqlite/
    artifacts/s3/
    artifacts/azure/
    artifacts/filesystem/
    secrets/aws/
    secrets/azure/
    secrets/kubernetes/
    secrets/local/
    notifications/discord/
  web/
    src/app/
      shell/
      core/auth/
      core/api/
      features/projects/
      features/applications/
      features/environments/
      features/suites/
      features/catalog/
      features/sources/
      features/runs/
      features/schedules/
      features/trends/
      features/settings/
      features/users/
      features/groups/
      features/permissions/
  contracts/
    openapi.yaml
    catalog.schema.json
    execution.schema.json
    progress.schema.json
    results.schema.json
    https-check.schema.json
    api-check.schema.json
    go-suite.schema.json
    suite.schema.json
    source.schema.json
    catalog-draft.schema.json
    run-event.schema.json
    artifact.schema.json
    evidence.schema.json
    identity.schema.json
    authorization.schema.json
    oidc-provider.schema.json
  packages/
    runner-sdk/
      typescript/
      go/
  runners/
    playwright/
    go/
    http/
  examples/
    sample-app/
    sample-suite/
  infrastructure/
    aws/
    kubernetes/
    azure/
    local/
  scripts/
    publish-suite/
    deploy/
    validate-deployment/
  docs/
```

Use standalone Angular components, small services, and pure utilities.
Keep product UI primitives and notification implementations inside this repository.
Do not create a shared library that requires changes to an unrelated application.

### Execution and publication contracts

Define versioned catalog, execution, progress, result, and artifact contracts before implementing adapters.
The service accepts project, suite release, optional environment, selection, and execution location identifiers.
Suites declare required bindings. Targetless suites omit the environment instead of using a placeholder.
Provider-specific task details belong to registered execution configuration.
The server does not accept arbitrary shell commands from the browser.

Plan adapters for Playwright browser tests, native HTTPS checks, Go suites, and REST API suites.
Their adapters send the same progress and final result records used by other adapters.
Reserve further adapter identifiers for accessibility, visual, and performance suites.
These additional adapters do not require empty implementations in the first release.
Generated HTML reports remain optional artifacts. Stored results remain the source for trends.

Use immutable container publication for coded suites initially.
Publish portable OCI images through configured registries. Record supported operating systems and processor architectures.
Register and activate releases per project and suite.
Validate release ownership, image digests, permitted execution configuration, and catalog references.
The server executes published releases. It does not clone Git repositories during test execution.
Record source repository and revision metadata for review and traceability.

Container releases require an immutable image reference.
Native HTTPS releases reference versioned check definitions and require no browser image or provider-specific task definition.

A later publication improvement can separate test bundles from pinned browser runtime images.
That work needs dependency verification and digest checks before activation.
It is not required for the initial product release.

### Run details and evidence

A run captures execution context and diagnostic evidence independently of optional HTML reports.
Expose the same details through authenticated API responses and the dedicated UI.
Results remain readable when report generation or an individual artifact upload fails.

Record project, suite, release, catalog revision, source commit, selected tests, trigger, actor, and parent rerun.
Record the resolved environment, endpoint snapshot, execution location, image digest, and toolchain or browser versions.
Record capture policy, worker identity, operating system, architecture, start and end times, exit status, and cancellation reason.
Store credential references and binding names without storing resolved secret values in the run snapshot.
Distinguish queue, preparation, execution, reporting, and artifact upload durations.
Use timestamps from the service and worker with clear clock origin. Do not imply a precise global ordering across workers.

#### Predictive timing and regression criteria

Time every test attempt and retain total execution time across retries.
Keep functional outcomes separate from duration regression assessments.
Compare only matching tests, environments, execution locations, runtime profiles, definitions, and target configurations.
Use clean first-attempt passes to estimate typical durations and guide bounded adaptive timeouts.
Keep slow, failed, retried, and interrupted observations outside the baseline.
Require enough comparable observations before enforcing slowdown rules.
Default to a failure at twice the median, with an absolute noise guard and a historical tail guard.
Support configurable warning, failure, and disabled modes, sample windows, history age, and timeout controls.
Freeze predictions and thresholds when each run enters the queue.
Show pre-run estimates, baseline readiness, attempt durations, slowdown ratios, and timeout budgets through the API and UI.
Make incomplete predictions explicit. Keep queue, preparation, and capture time separate.
Support audited baseline resets with revisions and generation guards for queued work.
See [Test timing](timing.md) for the initial native HTTP policy and contracts.

#### Attempts, steps, and capture

Each test result has its own attempts, assertions, steps, timing, errors, and artifact links.
Timeouts use a separate `timed-out` status and `timedOut` count across runners, API responses, and the UI.
Rerun failures includes failed, timed-out, and flaky tests. Timeout evidence retains unsuccessful-attempt capture and retention rules.
The run deadline preserves a known active-test timeout when the worker can save its result.
See [Test results](results.md) for status, aggregation, compatibility, and rerun behavior.
An explicit retry creates a separate attempt. A passing retry remains flaky.
Keep setup, teardown, worker, and build diagnostics identifiable even when no test starts.
Run-level output remains run-level when the adapter cannot attribute it reliably to a test.

| Evidence | Required behavior |
| --- | --- |
| Structured events | Record lifecycle transitions, worker heartbeats, step boundaries, retries, cancellation, and infrastructure failures. |
| Process logs | Capture timestamped stdout and stderr, severity when available, worker, stream, attempt, and ordered sequence. |
| Browser diagnostics | Capture console errors, page errors, failed requests, and bounded redacted request metadata. |
| Screenshots | Capture failed browser attempts and explicit test attachments. Allow configured checkpoint images and masking. |
| Traces | Retain Playwright traces for failed attempts by default. Collect them from the first attempt, even when retries are disabled. |
| Video | Support opt-in recording with always or failure-only retention. Keep recording disabled by default. |
| Assertions and API details | Capture expected and actual values, stack traces, bounded redacted request and response excerpts, and assertion timing. |
| Other attachments | Support typed attachments, optional coverage files, generated reports, and adapter-specific diagnostics. |

Default to logs for every run, screenshots on browser failure, retained failure traces, and no video.
Allow project defaults and suite overrides with optional test-specific capture settings.
Explicit run overrides require permission and remain within project capture and size limits.
Freeze the effective policy in the run snapshot.
Where supported, capture on assertion failure before teardown removes useful state.
Record unavailable evidence explicitly when a browser crashes or a worker terminates before capture.

Do not record authorization headers, cookies, secret values, or complete response bodies by default.
Redact configured sensitive fields in logs and structured diagnostics before publication.
Use configured screenshot masks and capture exclusions where binary evidence can contain sensitive application data.
Apply capture restrictions before collecting an attachment. Do not assume text redaction sanitizes an image, video, or trace.

#### Collection and publication protocol

Workers send structured events, log segments, and artifact manifests through versioned ingestion contracts.
Use credentials scoped to the assigned run, worker, permitted operations, and expiry.
A worker cannot attach evidence to another project, run, or attempt.
Uploads use the artifact adapter or a short-lived scoped upload grant.
Completion verifies expected ownership, size, content type, and checksum before making an artifact available.
Publish no arbitrary worker-supplied storage path or external URL as a trusted artifact reference.

Identify events and log segments by worker session, stream, and sequence for idempotent ingestion.
Preserve ordering within a stream. Expose missing sequence ranges and truncated output explicitly.
Send logs periodically during execution rather than only when a run finishes.
Use five-second flush and UI polling intervals initially, with configurable bounded batches.
The log API supports cursor-based reads, filtering by attempt and stream, and resumable downloads.
Keep acknowledged segments durable across service restarts.
A future streaming transport can reuse the event and cursor contracts.

Flush final events and upload available artifacts on success, failure, timeout, and cancellation when the worker can still execute cleanup.
Reconcile available evidence after unexpected worker loss.
Use separate states for test outcome, run execution, and evidence completeness.
Artifact failures do not convert an observed assertion failure or pass into a different assertion result.
Missing required result records keep execution incomplete. Missing optional attachments mark evidence incomplete.
Show missing or failed captures even when the test result is available.
Artifact upload retries do not rerun tests.

An artifact manifest records identifier, kind, filename, content type, bytes, checksum, capture time, attempt, step, and storage reference.
States include pending, available, upload-failed, missing, expired, and deleted.
Only available artifacts receive view or download grants.
The manifest also records capture omissions, redaction policy version, and size or retention limits reached.

#### Evidence API and UI

Use the project route prefix and project-authorized run routes.
Expose `/runs/{runId}` for run details.
Expose attempts, events, logs, and artifacts at `/runs/{runId}/{collection}`.
Name the collections `attempts`, `events`, `logs`, and `artifacts` in OpenAPI.
Use `/runs/{runId}/artifacts/{artifactId}/access` to obtain a short-lived view or download grant.
Document worker ingestion separately from user-facing read APIs.
The UI and external API clients have access to the same result and evidence metadata.

Run detail provides Summary, Tests, Timeline, Logs, and Artifacts views.
Test cards link directly to their attempts, failure screenshots, trace, related logs, and attachments.
Logs support live following, pause, text search, severity filters, stream filters, and download.
Provide screenshot thumbnails, image previews, video playback, and attachment metadata.
Use an installation-hosted trace viewer on the isolated artifact origin, with an authenticated download fallback.
Trace viewing must not require making evidence public or sending it to an external service.
Filter large artifact lists by test, attempt, kind, and state.
Show capture progress, failed uploads, missing evidence, and expired artifacts as distinct states.

#### Retention and access

Set project storage quotas, per-file and per-run limits, and separate retention periods for logs, images, traces, videos, and reports.
Allow longer retention for failed or flaky attempts and explicit retention pins for investigated runs.
Keep compact results and trends after large evidence files expire.
The UI and API show expiry times and retained metadata rather than broken download links.
Release pinning preserves execution artifacts; evidence retention pins preserve captured run artifacts.
Do not confuse these independent retention policies.

Authorize every artifact access request against its project and run.
Keep generated HTML and active attachment formats on an isolated origin.
Record access grants and administrative deletion actions in the audit history.
Retention cleanup respects pins and uses retryable storage deletion with explicit database state.
Report storage usage and quota failures without hiding the test outcome.

### TestStudio self-testing

Maintain a new Playwright suite in this repository. TestStudio runs this suite through its normal APIs and workers.
Use permanent test identifiers and immutable releases. Preserve comparable timing history when the same installation runs a newer release.
Cover catalog workflows, execution, evidence, timing controls, themes, and narrow layouts.
Extend the suite as new product areas become available.
Use separate intentional failures, timeouts, and cancellation checks to verify the runner contract.
A failed worker or missing report must never become a passing test.

Start with an explicitly enabled local container profile and disposable TestStudio instances.
Each test owns its database and target process. Do not reuse MarketMaker tests, resources, or data.
Keep self-test registration explicit. A normal installation starts with empty registries.
The first profile accepts targetless images with internal fixtures and no external network access.
Cloud and external-target profiles remain separate required work.
Keep stored attempts, steps, screenshots, traces, logs, and capture gaps visible through the standard result and evidence views.
See [Self-testing](self-testing.md) for local execution and retained verification output.

### Uploads, Git sources, and test management

Support uploads and registered Git repositories through the same release lifecycle.
Use Git for routine code maintenance and uploads for direct imports or source bundles.
Tests can live in an application repository or a separate test repository.
The service does not require every application to create a dedicated test repository.
Provide CLI publication as another way to register a release.

The common flow is:

```text
Upload files or sync a registered repository
  -> store an immutable source snapshot
  -> validate definitions and prepare the execution artifact
  -> discover and validate the test catalog
  -> create a ready suite release
  -> activate or pin the release according to suite policy
  -> execute manual and scheduled runs
```

Ready means the release is executable and its contracts are valid.
It does not mean that its tests passed against every environment.
Source synchronization and build validation do not silently execute production tests.
An optional run-after-publication setting must name its environment and selection explicitly.

#### Upload options

| Upload | Contents | Processing |
| --- | --- | --- |
| Declarative definitions | HTTPS checks or supported REST definitions in JSON or YAML. | Validate the schema and create a versioned native release. No container build is needed. |
| Source bundle | A ZIP containing the suite manifest, tests, helpers, policy, and dependency files. | Prepare an isolated build and discover its catalog. |
| Prebuilt release | A permitted immutable image or bundle reference plus catalog and provenance metadata. | Verify ownership, digests, runner compatibility, and contracts before registration. |

The upload wizard selects the destination project and suite.
It previews discovered identifiers, categories, types, safety labels, and validation errors.
New suites can be created during import.
Updating an existing suite creates a release rather than overwriting its current files.

A source bundle must include the dependencies needed to interpret its tests.
Playwright bundles include their package manifest, lockfile, fixtures, and required helpers.
Go bundles include the required module context, dependency metadata, and test catalog mapping.
A single source file can use an existing registered runtime only when its required context is explicit.
Do not assume an uploaded `.spec.ts` or `_test.go` file is self-contained.

Upload through a short-lived, project-scoped staging location.
The service computes and records the stored source digest.
Validate archive paths, entry types, file counts, and size limits before extraction.
Reject entries that escape the suite context.
Failed or abandoned staging uploads expire according to configured retention.
Published source snapshots remain immutable while retained releases reference them.

#### Registered Git repositories

Start with GitHub, including private repositories.
Keep the source provider interface independent of GitHub for later Git provider support.
Prefer a GitHub App with access limited to selected repositories and required read permissions.
Store connection credentials as server-side secret references.
Do not place personal access tokens in repository URLs, browser configuration, or build logs.

Repository registration records:

- Project and source display name.
- Provider connection and repository identity.
- Watched branch or tag, or an explicitly pinned commit.
- Suite paths, module roots, manifest locations, and permitted build profiles.
- Manual sync, push-triggered sync, or periodic sync policy.
- Automatic activation policy and any environment or schedule release pins.

Manual Sync now resolves the requested reference to a commit and stores that exact source snapshot.
Push synchronization validates the webhook signature and repository identity before queuing work.
Deduplicate preparation by source, resolved commit, manifest, and build profile.
Duplicate webhook deliveries must not produce duplicate builds or test executions.
Periodic synchronization uses the service scheduler when webhooks are unavailable.
GitHub Actions is not required.

Always execute a published release, not a branch checkout fetched during each run.
Record both the human-readable reference and resolved commit with the release.
Branch updates, tag changes, force pushes, and repository outages do not alter existing run snapshots.
A monorepo can register multiple suites without copying them into another repository.
Changes to shared fixtures, dependencies, or manifests must trigger preparation for affected suites.

#### Suite manifest

Use a small, versioned manifest with runner-specific configuration.
It identifies suite roots, catalog metadata, execution profiles, required bindings, and request policy.
Keep permanent test identifiers in source metadata or an explicit mapping file.
The service validates source declarations against registered project permissions and execution profiles.

An example proposed Go API suite manifest is:

```yaml
schemaVersion: 1
suiteId: api-regression
type: api
runner: go
applications: [service-api]
moduleRoot: .
tests:
  packages: [./tests/api]
catalog: testing/catalog.yaml
requestPolicy: testing/read-policy.yaml
buildProfile: go-module
```

Manifest validation is not a new production approval workflow.
Safety labels remain the production execution policy.

#### Builds and release activation

Persist source preparation and build jobs with progress, timestamps, logs, cancellation, and failure reasons.
The API and scheduler enqueue work. They do not run dependency installation or container builds in their own process.
Use isolated builders with the registered toolchain and bounded resource limits.
Build workers do not receive production test credentials or report-signing keys.
Repository access, private dependency access, and execution credentials remain separate bindings.

Select one supported build backend before delivering automatic source-bundle and Git preparation.
The product CLI publisher can register externally prepared releases initially.
Build backends can use local isolated workers, Kubernetes build Jobs, or configured cloud build services.
An AWS managed builder remains an AWS option. It is not a required service for other deployment targets.
The build backend remains a separate interface rather than a dependency of test scheduling.
Do not assume that the testing API container can run a Docker daemon.

Use observable preparation stages:

```text
received -> validating -> preparing -> discovering -> ready
                                      -> failed
                                      -> cancelled
```

A failed new build keeps the previous active release available.
Show that source is newer than the active release when preparation fails.
Do not report a preparation failure as a failed production health probe that never executed.
Older builds that finish late must not overwrite a newer active release.
Use activation revisions or source generations to prevent that race.

Support automatic activation of validated releases as suite policy.
Optional Activate, Pin, and Roll back actions select an existing immutable release.
These controls do not require per-test approval files or repeated confirmation for tests labeled safe.
Runs freeze their release when created.
Schedules either use the latest active release at firing time or an explicit release pin.
Changing the active release never changes tests already running.

#### Management UI and retention

The Sources area lists connection state, repository or upload origin, reference, last sync, and latest source version.
It provides Upload, Register repository, Sync now, source settings, and connection diagnostics.
The Suites area shows manifests, catalogs, build history, active release, pinned releases, and source links.
Build details expose progress and readable validation or compiler failures.
Git source links use the recorded commit rather than the current branch head.

Release comparisons show added, removed, and changed catalog entries and safety labels.
Source file changes remain inspectable without pretending that metadata alone proves behavior equivalence.
Stable identifiers preserve history when a test is updated or renamed.
Removed tests remain visible in historical runs.
Schedules that refer to removed tests show a clear configuration problem rather than silently dropping those checks.

Disabling a source stops synchronization. It does not erase retained releases or history.
Disabling a suite prevents new executions and identifies affected schedules.
Repository outages do not prevent execution of retained active releases.
Archiving a suite keeps its run history and reports accessible.
Retirement and restoration preserve permanent identifiers.
Preserve source snapshots and execution artifacts required by retained runs, release pins, or the rollback window.
Captured evidence follows the separate evidence retention and pinning policy.
Source and build retention are separate from test result and report retention.

Project permissions distinguish viewing, running, publishing, managing sources, and changing activation policy.
Record the actor and configuration version for upload, sync, activation, pin, rollback, and archive actions.

#### Source-management delivery sequence

1. Define the suite manifest, source snapshot, build, release, and activation contracts.
2. Support declarative uploads and product CLI or prebuilt release registration.
3. Add source archive uploads and GitHub registration with manual sync through one isolated build backend.
4. Add signed push synchronization, periodic sync, release comparisons, and activation policies.
5. Verify updates, failed builds, late build completion, rollback, retirement, and retained history.

No step requires a testing API deployment for each new suite release.

### HTTPS health-check test type

Add `https` as a first-class test type alongside `playwright` and `api`.
These tests check endpoint availability, response expectations, response time, and TLS health.
Each check has a permanent test identifier, project, application, category, safety label, and environment binding.
The UI creates and edits declarative check definitions without a test repository or application deployment.
Each edit creates a definition version. Existing runs keep their original version and resolved target snapshot.
Checks can also be imported through the publication API.

The first implementation supports these settings:

| Setting | Planned behavior |
| --- | --- |
| Target | Select a registered environment endpoint and relative path, or an explicitly registered project URL. |
| Method | GET by default. HEAD is an explicit alternative. Mutating methods belong to API suites. |
| Status | Configure accepted status codes. The default is 200. |
| Redirects | Disabled by default, so a login redirect cannot hide a failed endpoint. Optional redirects have a bounded limit and permitted destinations. |
| Response assertions | Optional required or forbidden text, selected headers, and JSON field expectations. No arbitrary JavaScript expressions. |
| Time limits | Configure an overall timeout and optional maximum response time. Use a proposed 10-second timeout initially. |
| TLS | Validate trust, hostname, and certificate dates. Optionally require a minimum remaining certificate lifetime. |
| Credentials | Optional secret references for authentication headers or basic authentication. The browser does not receive secret values. |
| Response capture | Store a bounded, redacted diagnostic excerpt when needed. Do not store complete response bodies by default. |

HEAD checks cannot require a response body.
Do not silently replace HEAD with GET when a server rejects HEAD.
Redirect destinations must pass target validation at every hop.
Do not forward credentials to a different origin or downgrade an HTTPS request to HTTP.
Explicitly configured private endpoints can use a project trust bundle. Certificate verification remains enabled.
Plain HTTP is an explicit local-development option, rather than the production default.

Example definition for the proposed contract:

```yaml
id: HTTPS-0001
type: https
application: service-api
category: health
target:
  endpoint: api
  path: /health
request:
  method: GET
  timeoutMs: 10000
  followRedirects: false
expect:
  statusCodes: [200]
  maxResponseTimeMs: 1500
  json:
    - path: status
      equals: up
  tls:
    minCertificateValidityDays: 14
recommendation: safe
rationale: Reads the registered health endpoint without changing application data.
```

Use a lightweight Go HTTP executor with a bounded worker pool in the service deployment.
Do not launch a browser or isolated container job for every health probe.
Keep execution outside HTTP request handlers and the scheduler loop.
Use durable execution claims, deadlines, cancellation, and recovery, as for other run types.
The same run creation path handles manual and scheduled execution.
An HTTP response is successful only when all configured assertions pass.
Capture every retry attempt. A passing retry remains flaky rather than a clean pass.

The executor checks only registered targets from its configured network location.
Target and redirect validation prevent access to cloud metadata services and unregistered destinations.
Private network probes require an explicitly permitted execution location.
Localhost probes require a worker that can reach the developer's application.
The same production-safety labels apply. GET and HEAD do not automatically establish that an endpoint has no side effects.

Record the following information:

- Resolved target, check version, execution location, timestamp, and attempt.
- HTTP status, final permitted URL, redirect count, and assertion failures.
- Request duration and optional DNS, connection, TLS, and time-to-first-byte timings.
- Certificate expiry date and remaining lifetime.
- Distinct failure reasons for DNS, connection, timeout, TLS, HTTP status, and content assertions.

Request duration measures probe execution, rather than queue delay or browser startup.
Run elapsed time remains a separate value.
Record unavailable timing phases as unavailable rather than zero.
Success rates describe observed checks. Missing executions do not imply successful checks or measured uptime.

The dedicated UI provides an HTTPS check editor and a test-type filter.
HTTPS cards appear under the same collapsible categories as other test results.
Cards show status, response time, TLS details, and readable failures.
History and trends support the existing project, application, stage, environment, category, and test filters.
Add response-time charts alongside pass and failure history for this type.
Include HTTPS checks in failure and volatility rankings.
An HTML report is optional. Native run details must work without an Allure report.

Use the in-process scheduler for frequent probes as well as nightly checks.
Apply configured concurrency and request limits to prevent a slow endpoint from blocking other projects.
Keep notification policy configurable per suite.
HTTPS suites can select every completion, failures and recovery, or a summary policy.
Any recovery or suppression state must survive service restarts.

Add this type during contract and registry work, before the deployment acceptance milestone.
Verify successful responses, unexpected status codes, timeouts, redirects, invalid TLS, approaching certificate expiry, and body assertion failures.
Verify that manual and scheduled checks produce the same run, result, trend, and notification records.
Adding or editing a health check requires no API deployment, browser image build, or service rollout.
See [Native HTTP checks](http-checks.md) for the implemented editor, assertion limits, publication rules, and retained diagnostics.

### Go test suites

Register Go suites with runner `go` and an appropriate type such as `unit`, `integration`, or `api`.
Keep existing Go tests beside their application source when appropriate.
Publish a registered test artifact from a pinned source revision and toolchain.
The service runs that artifact in an isolated execution worker, not inside the API process.
The publisher can include source and dependencies in a runner image or publish registered test binaries with the required adapter.

Suite metadata defines permitted packages, test selectors, categories, stable test identifiers, and supported execution profiles.
The UI selects catalog entries. It does not submit arbitrary shell commands or Go flags.
An example registered execution is:

```sh
go test -json -count=1 -timeout=5m -run '^TestExampleEndpoint$' ./tests/api
```

Disable Go result caching for scheduled and on-demand executions.
Parse structured events by package and test name, including interleaved parallel execution.
Map run, pass, fail, skip, output, and elapsed-time events into the common result protocol.
Store bounded test logs as diagnostics and retain the process exit status.
Record build failures, panics, timeouts, and worker termination as explicit run problems.
A package-level pass is not an additional passed test.
No selected tests executed must not appear as a successful run.

Map permanent test identifiers to package and function selectors in published metadata.
A function rename can update the selector while preserving the permanent identifier and its history.
Show dynamic subtests as children of their registered parent test by default.
Independently registered subtests receive their own identifiers.
Do not count parent aggregates and their registered children twice.
Explicit retries use separate attempts. A passing retry remains flaky.

Pure unit suites do not need target endpoints or application credentials.
Integration suites bind explicitly to their environment, services, data stores, and permitted operations.
Choose package allowlists rather than assuming `go test ./...` is safe for every repository.
Consumer repositories can contain tests that access live cloud resources.
Production classification must assess those operations before execution.
Optional race and coverage profiles use registered runner images and resource limits.
Coverage is an optional artifact, rather than a replacement for individual test outcomes.

The UI groups Go results by category or package and shows test names, subtests, attempts, duration, and failures.
Use the normal run cards, trends, rankings, schedules, and notifications.
Go tests require no Playwright or Allure dependency to produce usable results.

See [Local Go suites](go-testing.md) for the implemented source worker, manifest, timing, diagnostics, and verification contract.

### REST API test suites

Use type `api` for API behavior checks, independently of implementation language.
Support these implementation choices:

| Mode | Runner | Use |
| --- | --- | --- |
| Declarative requests | Native HTTP | Status, headers, content, JSON fields, schemas, and bounded request sequences. |
| Coded TypeScript | Playwright API request context | Reuse TypeScript fixtures and reporters for API workflows without launching a browser. |
| Existing Go integration tests | Go | Reuse application tests and Go clients for API contracts and workflows. |

Start coded API support with Playwright and the Go adapter.
Add the declarative editor for bounded cases after the shared HTTP execution and definition contracts are stable.
HTTPS checks remain the simpler availability and TLS check type.
API suites cover endpoint behavior, validation, authorization, pagination, and related request sequences.

Example read-only checks include:

- Search returns expected status, structure, and matching entries.
- A detail request returns the identifier and fields requested by the caller.
- An unknown resource returns the expected not-found response.
- Missing credentials produce the expected authorization response.
- Pagination follows the API contract without duplicate or omitted fixture records.

Resolve API base URLs and credential references from the selected environment.
Request sequences can extract values and pass them to later steps through bounded declarative bindings.
Do not accept arbitrary scripts in UI-managed request definitions.
Redact authorization headers, cookies, credentials, and sensitive response fields from diagnostics.

Browser route interception does not protect direct API clients.
Playwright API fixtures and Go HTTP clients need their own guarded transport or request wrapper.
Each request must satisfy the suite's target, method, path, payload, and redirect policy before transmission.
Production execution requires the `safe` label and reviewed read-only operations, including any read-only POST endpoints.
General Go code is trusted published code. A framework wrapper alone is not a sandbox for arbitrary code.
Restrict worker credentials and execution network access to the registered suite requirements.

Create, update, and delete workflows target permitted non-production environments.
Use dedicated test data and unique resource identifiers.
Cleanup runs after assertion failures where the worker remains available.
Record cleanup failures and remaining resource references explicitly.
Tests that change application data remain ineligible for protected production targets.

Each API test produces a permanent test identifier, category, final outcome, attempts, elapsed time, and readable assertion details.
Request-level steps can appear within its card without counting each step as another independent test.
Manual and scheduled execution use the same run, progress, result, artifact, trend, and notification contracts.
Suite publication does not require a testing service deployment.

### Adapter delivery milestones

1. Freeze the shared result contracts, test type and runner fields, stable identifiers, and environment capabilities.
2. Implement the generic Playwright adapter and native HTTPS checks with newly written example suites.
3. Add the Go event adapter and coded API suites, including direct-client request policies.
4. Add the declarative REST editor for bounded request sequences.
5. Verify browser, HTTPS, Go unit, and API suites in two newly configured generic projects across all deployment profiles.

### Local execution

Remote cloud and Kubernetes workers cannot reach a developer's localhost through a localhost URL.
The UI must show which execution target has network access to the selected environment.
Reject an incompatible execution target before creating an executable run.

Provide local execution through a local service or worker with access to the application.
Local, ECS, Kubernetes, and Azure execution use the same run registration, progress, result, and artifact contracts.
Local CLI executions must appear in service history when registered with the service.
Document container-to-host networking when the local executor uses Docker.
Keep the request guard enabled for local targets that read production APIs.

### Dedicated UI

The UI has its own application shell, authentication, design tokens, and runtime configuration.
It builds from this repository alone.
Use generic interaction patterns without importing another application's controls or styles.

#### Current stack and dependency policy

Use a standalone Angular application under `web/` in the service repository.
Keep the implemented Angular 22 standalone application. Use web/package.json and its lockfile for exact dependency versions.
Add packages only when a concrete requirement needs them and targeted checks justify the change.
No other application requires an upgrade or a shared library build.

Use these building blocks:

| Layer | Current approach |
| --- | --- |
| Components and state | Standalone Angular components, signals, focused services, and pure utilities. |
| Accessible interactions | Native controls now. Add Angular ARIA or CDK only when a new complex interaction requires it. |
| Styling | App-owned CSS and shared design tokens for spacing, typography, borders, focus, and status colors. |
| Charts | Existing app-owned accessible SVG stacked-area charts and pure chart utilities. |
| API client | Typed client generated from the service OpenAPI contract, with a small authentication adapter. |
| Authentication | Local credentials and OIDC authorization code sign-in through Go-managed sessions. Authorization uses `wtiger001/go-permissions`. |

Create a small set of shared UI primitives for buttons, form fields, selectors, dialogs, cards, status badges, and empty states.
Reuse these primitives across feature pages.
Keep test execution and source management out of visual components.
Keep styles scoped to app-owned classes so document elements cannot introduce unrelated width or alignment constraints.

#### Layout and visual direction

Use a compact operational dashboard with a persistent desktop sidebar and fluid main workspace.
Start the sidebar with Projects, followed by the Project and Application selectors.
Keep environment filters in the relevant page toolbar.
Use Start run for reviewed execution and Save and run for the explicit combined test-creation action.
Make the chosen project, target, and protected state visible before execution.

Do not use Workspace as an ownership heading.
Account access belongs in the global header. Accounts apply across the installation; project memberships grant project access.
Application testing contains Application overview, Suites, Test catalog, Test runs, and Trends.
Project overviews contains All applications and HTTPS probes.
Automation contains Schedules and Notifications.
Advanced testing contains Import and sync tests and Suite releases.
Project administration and Installation administration keep access controls separate from daily testing.
Automation, Advanced testing, and administration collapse by default and open for their current page.
Application environments remain in application context. Do not require testers to discover them through administration.
See [the sidebar contract](ui.md#sidebar-navigation) for current labels and behavior.
The page inventory below does not require a separate sidebar item for every page.

Use neutral light backgrounds, white cards, dark text, subtle borders, and one navigation accent.
Reserve green for success, red for failures, amber for flaky or incomplete results, and neutral colors for pending states.
Use blue for timeout status. Keep timeout counts separate from assertion failures.
Always show status text as well as color.
Use readable body text, compact metadata, and consistent control heights.
Keep desktop content gutters small and cards at the available content width.
Collapse the sidebar on narrow screens and wrap filters and card metadata.
Honor reduced-motion settings and visible keyboard focus.

#### Primary screens

The planned Overview page prioritizes running work, recent failures, health-check status, and upcoming schedules.
It links directly to the relevant run or test history.
Use tables for scanning many runs or sources, and cards for detailed test results.

The Test catalog provides its table, editing workflows, validation, publication, and revision history.
Category-tree navigation remains planned.
It includes test-type and runner filters, safety labels, and suite or application context.
Run selection supports categories, individual tests, and explicit dependencies.
Native HTTPS checks and REST definitions use structured editors.
Go and Playwright tests display their source and release metadata.

Import and sync tests provides uploads, source preparation, and registered repository workflows.
Source detail shows sync state, the resolved revision, build progress, failures, catalog changes, and active releases.
Keep source preparation status distinct from the outcome of a test execution.

Run details retain the current progress indicator, rotating activity icon, category summaries, collapsible categories, and full-width test cards.
Keep attempts and error output readable across the card width.
Provide the timeline, live logs, screenshots, traces, and artifact states described in the evidence contract.
Test and category links open their history while retaining context.

Trends use stacked area charts for result states. Retain failure and volatility rankings.
Stack passed, failed, timed-out, flaky, skipped, and unknown or incomplete counts within the same chart.
Use a stable stack order and consistent state colors across overall, category, and test views.
The default vertical axis shows counts. Each run contributes one final logical result per selected test.
Keep retries in attempt history. They do not add extra counts to the stacked areas.
Show each state's count and the total in the tooltip and an accessible data table.
Missing history creates visible gaps. Do not interpolate missing observations or count incomplete results as passed.
Project and environment scope, period, sample counts, and partial-history notices remain visible.
HTTPS check views add response-time history and certificate information.
See [Result trends](trends.md) for paging, logical counts, gaps, rankings, and the current project trend API.

Use dedicated routes for substantial authoring, including typed test creation and source workflows.
Focused configuration wizards can remain within their list or application page when this preserves context.
Use dialogs or compact inline forms for short actions. Follow the current shared UI contract.
Persist shareable filters in URLs.
Keep loading, empty, unavailable, and permission-denied states distinct.
An unavailable API must not appear as an empty healthy project.

#### Page help

Provide Help in the shared header on every page, including sign-in.
Open a drawer that slides in from the right and shows help for the current page.
Each guide contains Introduction, Concepts, and Tutorials with numbered task steps.
Keep guide content aligned with available controls and behavior.
Support keyboard navigation, Escape dismissal, focus restoration, and independent content scrolling.
Preserve unsaved page edits when opening and closing help.
Use shared light and dark theme tokens. Respect reduced motion and narrow viewports.
Verify the workflow through TestStudio's own Playwright suite.

#### Hosting and delivery approach

Build the UI as static assets with independent versioned releases.
Support the four deployment profiles defined above.
Cloud profiles can deploy the UI separately from the Go service.
Local and container distributions can serve the same UI build through the Go service.
Route authenticated API traffic through a configured origin or development proxy.
Do not cache authenticated API responses in a CDN.
Authorize private artifact access through the selected storage provider.
Keep generated reports separate from the authenticated UI origin on every profile.
Keep service addresses and identity settings in runtime configuration.
UI-only deployments do not require an API rollout when the contract remains compatible.

Review wireframes for Overview, Source import and build status, Run details, and Test history.
Validate the tree, cards, charts, and login flow with newly written product fixtures.
Test the complete workflows independently of any existing application or deployment.

Current navigation and related page inventory:

| Area | Purpose |
| --- | --- |
| Workspace | Projects and Your account. The Project selector changes the current project. |
| Applications | Application registration, environments, and linked suites. |
| Suites | Test groups, scoped catalog links, creation, and suite settings. |
| Test catalog | Definitions, typed test creation, validation, publication, and revision history. |
| Test runs | Run selection, review, execution history, and filters. |
| Run details | Progress, results, attempts, logs, evidence, cancellation, and reruns. |
| Trends | Result-state stacked areas and failure or volatility rankings. |
| Automation | Schedules, notification channels, and their retained histories. |
| Advanced testing | Import and sync tests, source preparation, and retained suite releases. |
| Project administration | Project settings, access, permissions, groups, roles, and access explanations. |
| Installation administration | Users, identity providers, installation access, permissions, groups, roles, and access explanations. |

Overview, cross-project rollups, and broader worker configuration remain planned work.
Do not add inactive sidebar links for planned pages.
The Project selector supports All projects for project discovery. Broader history scope remains separate feature work.
Execution requires one explicit project and an eligible suite.
Require an environment only when the suite declares target bindings. Targetless unit suites use No target.
Within a project, keep All, All Production, All Beta, All Dev, and individual environment filters.
Cross-project views keep project identity visible in chart points, links, and rankings.
Volatility compares outcomes within the same environment and test configuration.
Incomplete runs and skipped tests do not become successful observations.
Partial-history notices and sample counts remain visible.

Use the OIDC sign-in, session, user, group, and permission contracts defined above.
No identity provider, tenant, pool, account, or group is inherited from another product.
The API resolves an internal user and evaluates project and resource permissions through `permissions.Service`.
Project checks cover catalogs, runs, environments, evidence, schedules, publication, and access management.
The browser receives no cloud keys, provider tokens, or client secret.

### Notifications

Own notification transport implementations inside the product repository.
Providers use a common delivery contract and configurable project settings.
Notification delivery uses durable retries and at-least-once semantics.
A crash after remote acceptance can produce a retry duplicate.
Run links use the configured product UI origin and project routes.
Store webhook and provider credentials through the selected secret adapter.
Store only secret references in project settings.
No notification integration requires a MarketMaker library or compatibility wrapper.
See [Durable completion notifications](notifications.md) for the outbox, scoped API, channel editor, delivery history, and local verification contract.

### Delivery sequence

1. **Define independent contracts.** Specify projects, identities, memberships, scoped permissions, OIDC sessions, suites, targets, provider ports, and execution claims.
2. **Build the local product.** Deliver SQLite storage, the permissions adapter, user and group management, OIDC sign-in, catalog management, artifacts, and worker packaging.
3. **Add generic test capabilities.** Implement browser, HTTPS, Go, and API execution with evidence capture and newly written examples.
4. **Add deployment profiles.** Implement PostgreSQL, AWS ECS and CloudFront, Kubernetes, and Azure Container Apps with their execution adapters.
5. **Add source management.** Deliver uploads, GitHub connections, isolated preparation, immutable releases, and activation controls.
6. **Validate independent projects.** Register two generic projects through configuration. Verify isolated identities, data, permissions, and history.
7. **Validate deployment parity.** Run the same product acceptance scenarios on local, AWS, Kubernetes, and Azure deployments.
8. **Complete operational coverage.** Verify installation, upgrades, backups, restoration, worker recovery, and provider configuration documentation.

These steps describe implementation order. Usability and documentation work accompanies every step.
All four deployment profiles remain required product scope.
Use fresh product-owned resources or resources explicitly configured for this independent installation.
There is no existing-test import, historical data migration, scheduler cutover, or Admin navigation replacement.
Upgrade and rollback guarantees apply to this product's own versioned contracts and records.
Namespace metadata, artifact keys, idempotency, locks, schedules, and notifications by project and suite where applicable.
Verify schedule ownership, execution recovery, and notification retries when replicas restart or deployment revisions overlap.

### Completion criteria

- A new project, application, environment, and suite can be registered without editing Go or TypeScript source.
- Users, groups, membership, custom roles, assignments, and effective permissions are manageable through authorized API and UI workflows.
- `wtiger001/go-permissions` evaluates access on PostgreSQL and SQLite through the same service and identity contracts.
- User, group, role, object, and project grant checks cover explicit denies, expired grants, inheritance, and assignment revocation.
- Project administrators cannot grant installation privileges or affect another project's groups, assignments, results, or evidence.
- Membership and grant revocation reach all instances within the documented limit. Disabled users and revoked sessions fail their next request.
- Local accounts, OIDC accounts, and mixed sign-in work through the same directory, permission checks, and managed sessions.
- Local credential changes, resets, disabling, and recovery preserve audit records and the last usable administrator.
- OIDC sign-in supports configured providers with code flow, PKCE, state, nonce, issuer and audience validation, key rotation, and logout.
- Duplicate email addresses from different issuer-subject pairs do not merge accounts or transfer access.
- Unmapped, missing, incomplete, or outdated external groups do not silently preserve elevated project access.
- Browser sessions keep provider tokens server-side. Expiry, revocation, callback rejection, and request-forgery protection pass focused checks.
- SQLite restarts preserve users, memberships, assignments, custom grants, disabled roles, and sessions without a PostgreSQL service.
- Two newly configured OIDC providers work with the same internal directory and project-scoped permissions across deployment profiles.
- Catalog creation, editing, validation, publication, import, export, bulk actions, archive, and restoration work through both API and UI.
- Concurrent catalog edits produce explicit conflicts. Source synchronization preserves service-managed metadata.
- Catalog changes do not change existing run snapshots. Disabled tests and invalid dependencies block new dispatch explicitly.
- Run details expose attempts, steps, context, logs, screenshots, traces, and configured attachments without requiring HTML reports.
- Available evidence survives service restarts. Duplicate events and upload retries do not create duplicate attachments or rerun tests.
- Capture gaps, failed uploads, expired files, and incomplete evidence remain visible beside the observed test outcome.
- Artifact access respects project permissions. Private trace viewing requires no public upload or third-party viewer.
- Retention, quotas, and evidence pins work across local, S3, and Azure artifact adapters.
- PostgreSQL and SQLite pass the same storage behavior checks, including claims, revision conflicts, and restart recovery.
- Local SQLite installation requires no database server. Export and import to PostgreSQL preserve identifiers and artifact references.
- Declarative uploads, complete source bundles, and registered GitHub repositories can produce versioned releases through the UI.
- One repository can provide multiple suites, including suites beside existing application source.
- Failed or duplicate preparation events preserve the active release and do not create unintended test runs.
- Runs retain their release across updates, activation changes, and rollbacks.
- Removed tests and archived suites retain their history and required artifacts.
- Newly written generic suites complete runs and produce progress, history, reports, and trends on each deployment target.
- Result-state trends use stacked area charts with consistent state colors, final logical counts, accessible values, and visible missing-history gaps.
- HTTPS checks run manually and on schedules, with configurable assertions, TLS checks, response-time history, and normal result records.
- Adding an HTTPS check requires no browser or service deployment.
- Go suites preserve test identities, disable cached execution, and distinguish build failures from test assertion failures.
- Go unit suites run without artificial web targets or production credentials.
- API suites run through guarded direct clients and produce the same history and trends as browser tests.
- Package events, subtest aggregates, retries, and API steps do not inflate test counts.
- TestStudio executes its own new Playwright suite through normal publication, execution, timing, result, and evidence contracts.
- The Go service and UI build from this repository without MarketMaker imports, packages, shared resources, or sibling checkouts.
- No existing tests, fixtures, catalog records, configuration, credentials, reports, or history are copied into the product.
- Installation starts with empty registries and can load only explicitly selected new examples.
- AWS ECS and CloudFront, Kubernetes, Azure Container Apps, and local-server profiles pass the common acceptance scenarios.
- Local, Kubernetes, and Azure installations operate without AWS services or credentials.
- Local durable operation works without a cloud account.
- Provider adapters support execution status, cancellation, deadlines, artifacts, and recovery consistently.
- Hosted identity and private report access work through installation-specific configuration.
- Local and scheduled executions produce the same logical run and result records.
- Protected environments reject unsafe selections and unsafe dependencies.
- Manual and scheduled notifications use the same configured completion path.
- Replica failover creates no duplicate logical run. Notification retries retain their documented at-least-once behavior.
- Test-only publication requires no API or UI deployment.
- Provider credentials remain outside browser configuration and application logs.

### Remaining deployment and product decisions

The repository, current UI stack, databases, local recorder, and existing source workflow are already selected.
Do not reopen those choices without a concrete implementation problem.
Use go.mod and web/package-lock.json as the exact dependency records.
PostgreSQL and SQLite now pass shared storage parity and installation transfer verification. Hosted deployment acceptance remains separate.
Provider-specific choices below remain installation configuration or later implementation decisions:

- Public hostname, repository visibility, and hosted release origins.
- Actual OIDC providers, group mappings, session limits, and approved secret references for each installation.
- Packaging of the selected local service, preparation worker, browser runtime, and any future companion recorder.
- Azure static UI distribution when it is separate from Container Apps. Azure Functions is optional future integration work.
- Kubernetes ingress, artifact storage, secret integration, worker identity, and runtime requirements.
- Provider execution policy, concurrency limits, cancellation, deadlines, and recovery behavior.
- Remaining exact role and delegation boundaries, including initial project administrator assignment.
- Operator defaults for schedule cadence, response and certificate thresholds, capture, quotas, and retention.
- Additional dependency preparation profiles and hosted source credentials. The existing offline builder and GitHub workflow remain in use.
- Private trace-viewer packaging and credential redaction across logs, screenshots, traces, and attachments.
- Cross-project overview and history scope after the primary tester workflows are complete.

Record a decision when its implementation requires it. Do not invent account identifiers, credentials, or production resource settings.
The MMDev AWS account is the authorized deployment verification destination. It is not a runtime dependency or a reason to test AWS now.
Azure subscription and Kubernetes cluster selection remain open for their later deployment work.

### Protocol references

- [Playwright reporter interface](https://playwright.dev/docs/api/class-reporter) supports custom result and progress reporting for the first adapter.
- [Go HTTP client](https://pkg.go.dev/net/http#Client) documents client deadlines and redirect policy for native probes.
- [Go TLS connection state](https://pkg.go.dev/crypto/tls#ConnectionState) exposes verified connection and certificate information.
- [Go test event format](https://go.dev/cmd/test2json/) describes structured package and test results.
- [Go test flags](https://pkg.go.dev/cmd/go/internal/test) documents uncached execution and test selection.
- [Playwright API testing](https://playwright.dev/docs/api-testing) describes request fixtures for coded REST suites.
- [GitHub App installation tokens](https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/generating-an-installation-access-token-for-a-github-app) support repository-scoped source access.
- [GitHub webhook validation](https://docs.github.com/en/webhooks/using-webhooks/validating-webhook-deliveries) describes signed push delivery validation.
- [ECS standalone tasks](https://docs.aws.amazon.com/AmazonECS/latest/developerguide/standalone-tasks.html) describe isolated AWS executions.
- [Kubernetes Jobs](https://kubernetes.io/docs/concepts/workloads/controllers/job/) describe finite container workloads.
- [Azure Container Apps and Jobs](https://learn.microsoft.com/en-us/azure/container-apps/jobs) describe continuous services and finite executions.
- [Angular ARIA](https://angular.dev/guide/aria/overview) describes headless accessible interaction patterns for the proposed UI.
- [SQLite WAL](https://www.sqlite.org/wal.html) describes local filesystem and concurrency requirements.
- [Playwright recording options](https://playwright.dev/docs/test-use-options#recording-options) describe screenshot, trace, and video capture modes.
- [Playwright Trace Viewer](https://playwright.dev/docs/trace-viewer) describes inspection of captured browser execution details.
- [go-permissions](https://github.com/wtiger001/go-permissions) defines the selected authorization library and integration guides.
- [Permission and identity interfaces](https://github.com/wtiger001/go-permissions/blob/main/store.go) define the adapter contracts to verify when pinning a version.
- [OpenID Connect Core](https://openid.net/specs/openid-connect-core-1_0.html) defines sign-in claims and token validation.
- [OpenID Connect Discovery](https://openid.net/specs/openid-connect-discovery-1_0.html) defines provider metadata discovery.
- [OAuth security practices](https://www.rfc-editor.org/rfc/rfc9700.html) define current authorization-code and redirect protections.

## Light and dark interface

Support light, dark, and system theme preferences throughout the application.
Persist the explicit preference locally and apply it before the first application render.
Use shared semantic color tokens for surfaces, text, borders, focus, charts, and status.
Use native HTML landmarks, headings, labels, buttons, forms, tables, details, and dialogs where appropriate.
Use native controls for current simple patterns. Add Angular ARIA or CDK when the changed interaction needs their capabilities.
Support keyboard navigation, visible focus, reduced motion, and narrow viewports in both themes.
Every status includes text or an icon as well as color.
Verify relevant themes within the changed catalog or run workflow. Reuse passing evidence for unchanged controls.

## Local-first verification

Run initial functional testing locally.
Use SQLite and a local PostgreSQL container for persistence and authorization checks.
Use local application fixtures, browser sessions, workers, and artifact storage for complete workflows.
Use local provider contract checks where they directly verify changed adapter behavior.
Do not run AWS, Kubernetes, or Azure deployment checks during the current implementation session.
Run targeted tests after changes. Add broader checks only when feature integration requires them.
When separate AWS verification begins, use the MMDev account after local workflows are stable.
Keep account and profile values in installation configuration.
Cloud smoke tests supplement local functional coverage. They do not replace it.
