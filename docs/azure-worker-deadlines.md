# Independent Azure worker deadlines

## Current boundary

The Azure adapter now implements its Deadlines hook with Azure Logic Apps Consumption workflows.
Each workflow targets one owned Container Apps Job. The timer runs outside the API and preparation processes.
The adapter supports Go and Playwright execution and discovery through the same saved dispatch contract.

Azure coded execution remains disabled in native commands and infrastructure packaging.
Actual network admission, identity role binding, private exchange packaging, and native setup remain required.
Local SDK fixtures prove request and recovery behavior. They do not prove Azure timer delivery, termination latency, identity permissions, or deployment parity.
See [durable Azure execution](azure-worker-lifecycle.md) and [core Azure installation](azure.md).

## Resource and identity configuration

Construct SDKDeadlines with DeadlineOptions in the trusted controller.
Use the same durable Store, Jobs client, worker Config, and discovery cleanup authority as the execution provider.
Supply an explicit Azure TokenCredential. The adapter performs no credential discovery or resource-provider registration.

DeadlineConfig requires these settings:

| Setting | Requirement |
| --- | --- |
| worker | The exact worker configuration used by the provider. |
| resourceGroup | An existing resource group in the selected subscription, separate from the job resource group. |
| stopIdentity | An existing user-assigned identity in that subscription, distinct from the worker pull identity. |

The workflow uses the worker region. Its name combines the installation prefix, stop, and 24 reference digest characters.
Tags bind the complete reference digest, frozen specification, original deadline, and deadline configuration digest.
Names never substitute for full ownership checks.
A changed configuration cannot adopt timers created under the previous configuration. Retain the previous configuration until those jobs drain.

Grant the stop identity only Microsoft.App/jobs/stop/action on the dedicated worker job resource group or narrower job scopes.
It needs no start, secret, registry, workflow editing, or execution permissions.
Keep this identity off worker containers. Workflow editing authority can exercise its stop permission and belongs only to trusted controllers.
The controller needs workflow read, write, and delete authority in the separate deadline resource group.
It also needs the right to assign the selected managed identity to these resources.
Native packaging must create and verify these bindings before coded execution is enabled.
The current adapter validates identity references and resource definitions. It does not inspect effective Azure role assignments.

## Timer behavior

The workflow contains one built-in Recurrence trigger and one HTTP Stop action.
The trigger starts at the frozen absolute deadline, rounded up to the next whole second when necessary.
It repeats every minute afterward. Repetition catches executions that become visible or start after an earlier stop request.
The action posts to the exact job stop endpoint with management API 2025-07-01.
It stops all executions of that owned job. It cannot select another job or receive a different target from run input.
Normal provider cancellation retains its separate exact-execution stop contract.

The HTTP action uses the selected managed identity and Azure Resource Manager audience.
It contains no worker ticket, source credential, application secret, input packet, or request trigger.
Two fixed retries use ten-second intervals. DisableAsyncPattern prevents polling an arbitrary response Location endpoint.
An accepted stop response does not prove termination. The provider still observes the owned execution and preserves missing-status uncertainty.

This is an independently scheduled stop request, not an exact-second termination guarantee.
Azure scheduling, throttling, control-plane availability, and container shutdown can delay termination.
A missed recurrence can wait until a later occurrence. The API deadline gate and replica timeout remain separate controls.
Do not treat replicaTimeout as an absolute queue-plus-execution deadline.
Do not report service delay as measured runner duration.

Each execution reserves one workflow resource until cleanup completes.
Workflow quotas, control-plane throttling, orphan monitoring, and installation concurrency need capacity checks before hosted enablement.
An API outage leaves the timer active. It can continue issuing billable stop requests until cleanup resumes.
Do not disable or remove a timer while its job or execution remains unresolved.

## Creation and recovery

The provider first reserves DeadlineAttempted in its durable dispatch journal.
The SDK hook requires that saved reference, complete specification, and absolute deadline.
An existing workflow before the hook reserves creation is rejected. The adapter does not overwrite or adopt it.
A revision-checked journal mutation reserves one creation attempt before PUT.
Current exchange authorization, expiry, revocation, and dispatch state still guard that reservation.
Concurrent controllers cannot reserve another creation attempt.

The SDK pins armlogic 1.2.0 and workflow management API 2019-05-01.
Requests use the fixed public Azure management origin and generated workflow namespace.
Automatic retries, redirects, child operations, extra queries, foreign names, and alternate API versions are rejected.
Operations have five-second contexts. Responses have a one-megabyte read limit.
Provider diagnostics map to fixed errors.

After an unknown creation response, recovery reads the resource instead of repeating PUT.
Successful verification requires the complete expected writable definition, selected identity, enabled state, and succeeded provisioning.
Only documented read-only metadata and empty outer parameters are excluded.
An evaluated recurrence is excluded only when it equals the declared recurrence.
Changed targets, actions, credentials, parameters, ownership, state, and schedule block admission.
The provider verifies the timer before Start and during execution admission.

## Cleanup

Removal requires a revoked exchange, a saved result, and complete or explicitly abandoned evidence.
Discovery uses its separate saved cleanup authority.
The hook independently checks that the job is absent before it removes the timer.
A durable removal marker fences later creation and verification.

A missing workflow after an unseen, uncertain creation remains unresolved. Missing visibility does not prove that no resource was created.
When that exact resource becomes visible, cleanup verifies ownership and removes it.
A creation attempt that never becomes visible needs operator investigation. Do not erase the journal to bypass that uncertainty.
Definitively rejected creation has no such unseen-resource requirement.
Foreign or changed resources remain untouched.

A deletion attempt reserves a thirty-second retry boundary before the request.
Concurrent controllers retain one reservation. Unknown responses retain the marker through database restart.
Cleanup requires a subsequent absent-resource response before saving the removed marker.
Removed records remain fenced against redispatch.

## Local verification

Eleven new focused checks cover both runners and both operations.
They cover lost creation responses, SQLite reopen, unready provisioning, changed definitions, exact SDK stop requests, concurrency, and cleanup ordering.
They also cover unseen-resource uncertainty, deletion backoff, lost deletion responses, configuration replacement, and bounded control requests.
Two selected existing provider checks cover journal compatibility and controller concurrency.
Only affected fixture checks repeat after fixture encoding and clock corrections.

To reproduce the deadline checks locally:

```sh
go test -race ./adapters/execution/azure -run '^TestAzureDeadline' -count=1
```

Evidence is output/azure-worker-deadlines/summary.json and the focused command results.
No Azure connection, cloud credential, deployment, registry push, UI build, browser, container, or image build is part of these checks.
Acceptance 51 and supporting recovery under 52 advance. The total remains 48 verified of 57, with nine open items.

## Primary references

[Recurrence scheduling](https://learn.microsoft.com/en-us/azure/connectors/connectors-native-recurrence) defines future start times and recurrence behavior.
[Managed identities](https://learn.microsoft.com/en-us/azure/logic-apps/authenticate-with-managed-identity) defines workflow identity and audience configuration.
[HTTP action options](https://learn.microsoft.com/en-us/azure/logic-apps/logic-apps-workflow-actions-triggers#asynchronous-patterns) defines DisableAsyncPattern.
[Job stop](https://learn.microsoft.com/en-us/rest/api/resource-manager/containerapps/jobs/stop-multiple-executions?view=rest-resource-manager-containerapps-2025-07-01) defines the exact stop request and asynchronous acceptance.
[Compute permissions](https://learn.microsoft.com/en-us/azure/role-based-access-control/permissions/compute) defines the job stop action.
[Logic Apps limits](https://learn.microsoft.com/en-us/azure/logic-apps/logic-apps-limits-and-config) defines workflow and identity quotas.
