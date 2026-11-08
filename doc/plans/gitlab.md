# GitLab runner execution plan

Status: Proposed integration, captured on 2026-11-08. No GitLab execution adapter is implemented or verified.
This plan records the design discussion. It does not replace the current recording priority or authorize deployment.

## Goal

Run TestStudio tests on each team's existing GitLab runners.
Teams retain ownership of runner infrastructure, capacity, dependencies, and network access.
TestStudio retains test selection, release snapshots, permissions, timing, results, history, and private evidence.

## Execution flow

1. A user selects tests, an environment where required, and an approved GitLab execution location in TestStudio.
2. TestStudio checks current authority and freezes the selected release, inputs, timing, capture policy, and deadline.
3. A GitLab provider adapter triggers the team's approved execution pipeline.
4. GitLab schedules the job on an eligible team runner.
5. The job starts the TestStudio worker supervisor in the approved runtime.
6. The supervisor claims its execution, runs the selected test, and reports progress, results, and private evidence.
7. TestStudio records completion and confirms provider cleanup. The run links to its GitLab pipeline and job.

GitLab receives jobs through its pipeline API. TestStudio does not dispatch directly to the runner process.
GitLab supports pipeline triggers and job tags for runner selection.
See [pipeline triggers](https://docs.gitlab.com/ci/triggers/) and [runner configuration](https://docs.gitlab.com/ci/runners/configure_runners/).

## Team routing and authority

Each approved execution location binds a TestStudio project to a GitLab instance, GitLab project, controlled pipeline configuration, approved ref, runner tags, and credential reference.
It also records supported runtimes, architecture, network policy, concurrency, and queue limits.

| Execution location | GitLab project | Runner tags |
| --- | --- | --- |
| Payments private network | payments/test-execution | payments, private |
| Inventory staging | inventory/test-execution | inventory, staging |

These names are illustrative. Installation starts without example projects or runner registrations.
GitLab project and group runner scope controls access to runners. Tags select capabilities among eligible runners; tags alone are not an authorization boundary.
A runner must match every tag requested by the job.

Project users select only locations they are authorized to use.
They cannot substitute arbitrary GitLab projects, refs, tags, commands, images, or credential references at run time.
Recheck current project, suite, environment, and execution authority before dispatch and mutation.
GitLab credentials do not grant TestStudio membership or installation privileges.

## Existing foundation and missing work

The [coded execution contract](../../internal/execution/coded.go) already separates execution from the run service.
The [private worker exchange](../../docs/worker-exchange.md) binds one attempt to immutable input, an expiring ticket, a worker instance, and private result delivery.
The [worker supervisor](../../docs/job-supervisor.md) runs outside the API and preserves structured results, timing, and evidence gaps.
Its [command entry point](../../cmd/teststudio-job/main.go) starts the existing Go or Playwright runtime.

Reuse these contracts. Add a GitLab provider adapter, durable pipeline/job receipts, team configuration, runtime packaging, and operator guidance.
The current supervisor has fixed Linux runtime paths and expects an approved prepared suite image.
Using a custom runner does not make an arbitrary checkout or operating system compatible automatically.

Start with a compatible Linux Docker or Kubernetes runner and an immutable prepared suite image.
Shell runners need an explicitly supported runtime and process, filesystem, and network isolation before admission.
See [GitLab executors](https://docs.gitlab.com/runner/executors/).
GitLab repository ingestion and source preparation are separate capabilities from GitLab execution.
This first integration must not imply that GitLab source registration already exists.

## Credentials, connectivity, and targets

Store GitLab API credentials through secret references. Keep them outside uploaded test code, browser configuration, and logs.
Choose credentials that cover the required trigger, observation, and cancellation operations; a trigger token alone does not establish the full lifecycle.
GitLab API credentials and TestStudio attempt tickets serve separate purposes.

The job receives only an expiring execution credential through an approved protected delivery mechanism.
Do not assume ordinary pipeline variables are a safe secret transport. Review job visibility, log exposure, and bootstrap authentication before choosing delivery.
Preserve the supervisor's exclusion of the job ticket from the test process environment.
The job receives no TestStudio database credentials, signing keys, or user session.

The TestStudio controller must reach the GitLab API.
The runner must reach the authenticated worker endpoint over HTTPS and pull the approved runtime image.
Use the existing private worker route or a separately reviewed authenticated relay; do not expose the internal listener as a public shortcut.

Browser and API tests retain the trusted target gateway and approved origin policy.
The runner's presence inside a team network does not replace target authorization or isolation.
Go unit tests without targets must remain usable without artificial environments or target credentials.

## Lifecycle and result rules

- Persist dispatch intent and correlation before triggering. Recover ambiguous trigger responses by locating the existing pipeline before creating another.
- Persist pipeline and job IDs so controller restart or failover resumes observation and cleanup.
- Bound queue and startup time separately from measured test duration. Preserve the frozen test timeout and outer execution deadline.
- Disable automatic test execution retries. A new GitLab job instance cannot reclaim an already claimed attempt. Use explicit TestStudio retries for another attempt.
- Cancel queued or active provider work when the run is cancelled or expires. Confirm terminal state and owned resource cleanup, including gateways.
- Treat missing runners, image failures, lost jobs, and absent reports as infrastructure observations with visible evidence gaps.
- Deliver evidence through TestStudio's existing private artifact contract. Optional GitLab artifact copies must have an explicit access and retention policy.

GitLab job success is separate from a passing test.
The existing supervisor exits successfully after delivering an accepted report, including a failed test report.
Add a trusted outcome bridge so a failed test can fail the GitLab job after successful report delivery.
Keep assertion failure, test timeout, infrastructure failure, cancellation, and delivery failure distinct in TestStudio.
Use the [Jobs API](https://docs.gitlab.com/api/jobs/) for provider observation and cancellation as appropriate.

## Entry points

First support Run from TestStudio, with manual and scheduled runs using the same approved team location.
The team installs a small controlled execution pipeline; testers need no GitLab configuration for each run.

Later support a team's GitLab deployment pipeline requesting tests and waiting for the TestStudio result.
It must reuse normal run creation and authorization, retain GitLab commit and pipeline provenance, and report results into the same history.
A CI job that merely requests remote execution does not prove that tests executed on that team's runner.

## Milestones and acceptance

| Milestone | Acceptance numbers | Smallest useful verification |
| --- | --- | --- |
| 1. One-team execution proof | 31, 32, 41, 42, 43, 51 | One new generic Go test runs through normal publication and an actual team runner. Verify selected identity, immutable image, measured duration, private logs, pipeline/job links, and a failed assertion that also fails the GitLab job. |
| 2. Durable lifecycle | 39, 43, 44, 51, 52 | Focused adapter checks and actual jobs cover an ambiguous trigger response, controller restart, duplicate delivery, queued cancellation, timeout, missing report, and confirmed cleanup without duplicate execution. |
| 3. Team boundaries and target execution | 12, 13, 35, 38, 41, 45, 51, 56 | Two configured team locations reject cross-project routing. A new browser or API fixture uses the approved gateway and returns private evidence. Revoked authority blocks dispatch. |
| 4. Product and CI workflows | 21, 37, 42, 49, 50, 51, 56 | A focused normal TestStudio browser workflow selects a location, reviews and starts a run, inspects results, and cancels work. Verify help, keyboard, themes, narrow layouts, scheduling, and a GitLab-initiated run. |

Implement each milestone before its targeted verification. Keep local adapter evidence separate from actual GitLab execution and deployment checks.
Use explicit test IDs for browser verification. Stop owned jobs, gateways, browsers, and temporary services after success or failure.
Preserve the user's port 8040 demo, database, and history.

The existing [acceptance ledger](../../docs/implementation.md#acceptance-criteria) remains authoritative: 50 verified of 57.
This document closes no acceptance item. Broad cross-profile criteria stay open until their complete scope passes.
Azure work remains deferred.
