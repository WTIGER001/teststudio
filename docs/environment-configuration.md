# Environment configuration

An application owns its suites, catalog, and environments.
An environment represents one deployment, such as Development, Staging, or Production.
An HTTP probe combines an environment base URL with its relative path.
Selecting another environment reuses the same test definition.

## Define requirements

Open an application and select Environment configuration.
Select Edit requirements and Add requirement.
Each name identifies one Base URL, Parameter, or Secret reference.
Parameters support text, HTTPS URLs, whole numbers, and true or false values.
A parameter can supply a nonsecret default. An environment value overrides it.
Secrets have no default value and never fall back to another environment.
Required for every run applies whenever that environment is selected.
Other requirements apply when selected tests or their dependencies declare them.
Review requirements before Save configuration.
New requirements can make existing environments incomplete without deleting their saved configuration.

Tests declare `requiredParameters` and `requiredBindings`.
Parameter names must already exist in the application contract. Save or import identifies undeclared names.
API secret requirements and HTTP endpoint requirements also enter the matrix through enabled readable tests.
The matrix never includes a denied environment or test.
Current saved browser endpoints also enter the matrix and retain their existing runtime policy.

## Configure environments

The readiness matrix compares every readable environment in the application.
Choose Configure for an environment, enter its values, then review and save.
Named endpoints support applications with separate website and API addresses.
Blank entries remove that environment value. A parameter default can still apply.
Incomplete environments can be saved. Saving never starts a test or enables a schedule.
Help preserves edits. Navigation asks before discarding them.
A revision conflict retains edits and disables saving until Reload configuration.
Reload asks before replacing unsaved entries with the current stored values.

Missing means a required entry has no value.
Invalid means its declaration, value type, or endpoint policy needs correction. Removing a parameter declaration used by an enabled test keeps that requirement visible as invalid.
Unavailable means an approved secret reference cannot be found or resolved.
Ready means a nonsecret value passes configuration validation.
Configured means a secret reference exists and is approved for an environment endpoint origin.
Execution checks its current availability separately.
Neither state proves that a target accepts a credential or that a test will pass.
Use an explicit normal run to test that behavior.

## Secret boundaries

An operator registers secret values through the existing scoped secret provider.
Environment editors select approved reference names. Application-only editors see references approved for their managed environment origins; save a changed base URL before selecting a reference for that new origin. They cannot read or paste secret bytes here.
The provider resolves required references during run review, creation, dispatch, and retry authorization.
Provider diagnostics return fixed messages without private details.
Run snapshots store reference identifiers and nonsecret parameters, never resolved secret values.
Rotation retains the provider reference and resolves the new value at execution.
API credentials still use the reviewed gateway and exact-origin restrictions.
Browser login secrets and native HTTP authentication are separate capabilities; defining a requirement does not add an injection mechanism.
See [API credentials](api-credentials.md) for operator registration and supported authentication.

## Execution enforcement

Run review and enabled schedule saves validate application requirements and the selected dependency closure.
Requirements of unrelated tests do not block the selection.
Dispatch and each attempt recheck the current contract, environment policy, authority, and secret availability.
Changed effective parameter values block an earlier queued run. Create and review a new run to use them.
Earlier definitions, release pins, parameter snapshots, results, and evidence remain unchanged.
Manual and scheduled runs use the same checks.

HTTP paths and text assertions accept `{{name}}` references from `requiredParameters`.
Path and query values use their respective URL escaping. Replacement cannot change the environment origin.
Coded environment profiles receive nonsecret parameters as JSON in `TESTSTUDIO_PARAMETERS` in updated worker runtimes.
They also remain available in the immutable work item under `browser.parameters`.
Existing runtimes must be rebuilt before coded tests consume this new variable. No hosted runtime upgrade is implied by local checks.

## Existing suites

New applications create an empty Default suite atomically with the application.
Existing applications can select Create default suite here. This preserves all existing suites and tests.
In Project settings, inspect a shared or unassigned suite and choose one application owner.
Review its current tests and affected schedules, then confirm the assignment checkbox and save.
Incompatible active tests, probes, or schedules reject assignment. Resolve those resources before retrying.
An assigned owner is permanent. Retained versions and historical runs are never rewritten.
Legacy shared suites stay readable through project administration and retained history; application catalogs exclude them.

## API

Routes below `/api/v1/projects/{project}/applications/{application}`:

| Method | Route | Purpose |
| --- | --- | --- |
| GET | `/catalog` | List this application's tests with scoped counts and filters. |
| GET | `/configuration` | Read the application contract and authorized environment readiness. |
| PUT | `/configuration` | Replace requirements using the current application revision. |
| GET | `/credential-references` | List approved reference names with environment management authority. |
| PUT | `/environments/{environment}/configuration` | Replace base URLs, parameters, and secret bindings using the environment revision. |
| POST | `/default-suite` | Add an empty default suite to an existing application using its revision. |

Mutations require current application authority, Origin, CSRF protection, and a positive If-Match revision.
Environment updates also enforce the exact environment authority and application ownership.
The existing project registry APIs remain compatible for administration and aggregate API reads.
New suite writes require exactly one owner. Existing sole-owner suites cannot be reassigned.
