# Independent Azure sandbox deadlines

## Introduction

SDKDeadlines supplies the independent deadline interface used by the [sandbox controller](azure-sandbox-controller.md).
It manages one Azure Logic Apps workflow for each logical execution attempt.
The workflow starts at the absolute ticket deadline and repeats stop requests independently of TestStudio's API process.
It can find an owned sandbox without a successful create receipt.

Local SDK, restart, schema, and static-reference checks pass.
Native command selection and deployment packaging remain unfinished.
Actual scheduling, identity authority, service compatibility, and stop enforcement still require separate deployment verification.
Local schema validation does not execute the workflow.

## Concepts

### Configuration

| Field | Requirement |
| --- | --- |
| Worker | The validated sandbox Config. |
| ResourceGroup | A dedicated timer group in the worker subscription. It differs from the worker group. |
| NamePrefix | Two to 16 lowercase letters, digits, or hyphens. Start with a letter and end with a letter or digit. |
| StopIdentity | An explicit user-assigned identity in the worker subscription. |

Use a stop identity separate from source publication, image pull, controller management, and runtime identities.
The installation must verify its selected sandbox-group read and stop grants and protect workflow management and identity attachment.
The SDK compares identity metadata. It does not prove effective role grants.

DeadlineOptions also requires the actual shared worker exchange, Instances client, and an explicit Azure credential.
Construction performs no provider request or credential discovery.
ARM calls use that management credential and armlogic's pinned 2019-05-01 API.
Automatic retries and resource-provider registration remain disabled.
Requests stay within the selected subscription, timer group, and generated workflow names.
The owned client uses HTTPS, TLS 1.2 or later, no environment proxy, and no redirect following.
Requests have five-second limits. Bodies close and have a one-MiB limit. Close releases idle connections.

Workflow data-plane calls use the selected stop identity.
Their audience is https://dynamicsessions.io, derived from the pinned SDK's https://dynamicsessions.io/.default scope.
This differs from the ARM audience. Confirm actual compatibility during deployment verification because some public examples use another audience.
Runtime workers receive neither management credentials nor the stop identity.

### Timer behavior

Names combine the prefix, stop, and 24 hexadecimal reference characters.
Tags retain the full reference hash, execution specification, deadline profile, and absolute deadline.
The profile binds configuration, API version, and token scope. Definitions contain no usable worker ticket.

The recurrence starts at the ticket deadline rounded up to the next whole second and repeats every minute.
It permits one active run and one waiting run.
Rounding adds less than one second. Platform scheduling and request latency still need deployment verification.
Idle suspension does not replace this timer.

Inventory uses the selected regional endpoint, group, API version, and exact ownership-label filter.
Returned items must also match all expected labels before receiving a stop request.
These labels bind installation, project, execution, reference, configuration, specification, and deadline.
All matching instances receive stop requests, including late or duplicated matches.
Logical ownership controls this safety stop even if another execution setting changes.
Normal admission and cleanup retain their complete configuration checks.

The workflow accepts arrays and value envelopes.
It permits eight pages, two minutes per loop, 500 items per page, and 1,000 total items.
Continuations retain the exact endpoint, collection, API version, and ownership filter, with one permitted nonempty continuation argument.
Foreign scopes, changed queries, cycles, fragments, and oversized links cannot produce another inventory request.
Only a bounded ASCII instance identifier enters the encoded stop URI under the selected group.
Unsafe or incomplete inventory produces a failed workflow run.

List and stop actions disable asynchronous Location polling and configure bounded retries and action time limits.
Inventory, selection, and stop actions hide sensitive inputs and outputs from run history.
Raw runtime environments are removed before items enter workflow variables.
Accepted stop responses remain acknowledgments. They do not prove quiescence or passing tests.
The timer repeats until checkpointed removal. Native packaging still needs capacity limits, failure monitoring, and orphan cleanup.

### Creation and recovery

The controller records deadline intent before calling the SDK.
The SDK checks for a name collision, then atomically saves TimerProfile and TimerCreateAttempted against the gated exchange revision.
An existing unreserved workflow remains an ownership conflict. It is not overwritten or adopted.
Only the winning reservation calls CreateOrUpdate.
A lost acknowledgment permits verification, never another blind create.

Verification compares all pinned SDK writable fields, selected identity, tags, enabled state, and successful provisioning.
Only declared provider metadata is removed. Empty identity objects retain their keys.
Changed scope, policy, timing, identity, or history protection rejects ownership.
Successful readback saves TimerSeen.
Known rejection remains distinct from uncertainty and rejects dispatch.
A missing timer means unavailable coverage. It never proves the worker absent.

### Removal

Removal requires revoked work, the canonical result and evidence checkpoint, and settled instance ownership.
A saved instance receipt requires another authoritative absence check.
The SDK saves cancellation and timer-removal barriers before destructive control.
It retains an uncertain timer create that has never produced owned readback, so delayed creation remains recoverable.

An existing timer must pass ownership verification before deletion.
Successful observation and a 30-second deletion backoff are recorded before the request.
Deletion stays pending until an exact GET confirms absence.
A lost deletion response can finish after restart without another delete when absence is known.
The controller excludes the dispatch from active inventory only after instance and timer cleanup both complete.

## Tutorials

### Run targeted local checks

```sh
go test -race ./adapters/execution/azuresandbox -run '^TestSandboxDeadline' -count=1
```

These checks use the production ARM SDK and transport with an in-memory endpoint and explicit fake credential.
They use the real exchange and SQLite database, including close and reopen.
They cover both runners, discovery, uncertain and rejected creation, unreserved timers, metadata drift, redirects, and interrupted deletion.
No actual provider, registry, identity endpoint, listener, browser, container, or deployment runs.

### Check the generated workflow

Set TESTSTUDIO_SANDBOX_DEADLINE_OUTPUT to an ignored directory when running the workflow contract check.
It writes workflow.json for independent schema validation.
Run scripts/verify_azure_sandbox_deadline.py with jsonschema 4.23.0 in a separate local tool environment.
Supply the workflow, downloaded public schema, expected schema SHA-256, and result output path.
The script checks schema, expression delimiters, initialized variables, references, dependency cycles, and HTTP history protection.
It does not evaluate expressions or execute the workflow.

The selected schema has SHA-256 7795118ae8a8dcfe7bdf0b84229be08cde8bc7ebd9eeac9e6c372746c520499d.
Evidence is output/azure-sandbox-deadlines/summary.json, schema-validation.json, and the retained command results.
The current workflow has 37 actions, six initialized variables, and 23 expressions.

### Continue native integration

Bind actual read and stop grants, protected identity attachment, and workflow management before native selection.
Package the timer group, identity, capacity controls, and failure monitoring.
Confirm audience, filtering, pagination, scheduling, history protection, redirect behavior, and effective stop during separate Azure verification.
Independent command completion, durable source import, actual admission, and the complete Provider bridge remain required.

## Primary references

- [Workflow Definition Language schema](https://schema.management.azure.com/providers/Microsoft.Logic/schemas/2016-06-01/workflowdefinition.json)
- [Microsoft trigger and action reference](https://learn.microsoft.com/en-us/azure/logic-apps/logic-apps-workflow-actions-triggers)
- [Microsoft loop limits and failures](https://learn.microsoft.com/en-us/azure/logic-apps/logic-apps-control-flow-loops)
- [Microsoft history protection](https://learn.microsoft.com/en-us/azure/logic-apps/set-up-security-permissions)
- [Pinned sandbox SDK](https://pypi.org/project/azure-containerapps-sandbox/0.1.0b4/)

This advances 51 and supporting cleanup recovery under 52. The total remains 48 verified of 57.
