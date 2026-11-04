# API credential bindings

An installation operator registers a project credential reference and its exact permitted origins.
An environment maps a test binding name to that reference. The gateway resolves it for each authenticated attempt.
Secret values remain outside source code, browser configuration, and test worker input.

## Register a reference

Add an entry to the operator's secret binding file:

```json
[
  {
    "id": "service-reader",
    "name": "Service API reader",
    "projectId": "example-project",
    "purpose": "api-credential",
    "file": "/run/secrets/service-api-reader",
    "allowedOrigins": ["https://api.example.test"],
    "network": "public"
  }
]
```

Start the server with --secret-bindings pointing to this file.
Use environmentVariable instead of file when the operator supplies the secret through the server environment.
Choose exactly one source. Configuration contains references and source locations, not secret bytes.
Mounted files contain only the credential value, without a trailing newline.
The API profile limits values to 4096 bytes and rejects control characters.

References belong to one project and the api-credential purpose. Notification and repository credentials remain separate.
Each reference requires 1 through 16 exact canonical origins. Wildcards, embedded credentials, duplicates, and trailing slashes are rejected.
Network public permits public addresses only. Network private also permits explicitly selected private targets.
The effective scope intersects the credential scope and environment policy. Metadata and link-local addresses remain blocked.
API credentials do not accept network loopback references.

HTTPS is required outside environments whose explicit stage is local.
For local verification, approve the complete worker-reachable HTTP origin, including its port, and choose network private.
Container localhost cannot reach the desktop server. Use an approved reachable address such as host.docker.internal.
See [API transport](api-testing.md#direct-request-transport) for worker and endpoint restrictions.

## Configure an environment

1. Open an application and select Add environment, or open Project settings and choose Environments.
2. Enter a name, then select Continue. Choose the application, stage, endpoint, and protection.
3. Expand Advanced credentials. Select Add credential binding.
4. Keep api-login or enter another stable binding name. Choose an Approved credential reference.
5. Continue to Review. Inspect the mapping, target, and protection before selecting Create.

The form accepts up to 16 unique bindings. References come from the current project's approved catalog.
Unapproved origins and references produce explicit field errors. Invalid Advanced fields open and receive focus.
Back preserves entries. Review contains no secret value.
No available references means an operator must register one. Never paste a token or password into this form.

The environment API uses credentialBindings as a binding-name to reference-ID map:

```json
{"credentialBindings": {"api-login": "service-reader"}}
```

Environment writes retain their normal revision, Origin, session, and environments.manage checks.
The server validates reference project, purpose, identifiers, and allowed endpoint origin before saving.
Changed mappings block queued dispatch and retries that retain earlier configuration. Create a new run after reviewing the change.

GET /api/v1/projects/{project}/api-credential-references requires environments.manage.
Its items contain id, name, and allowedOrigins. It resolves no secret and returns no source location or environment-variable name.

## Choose authentication

In Create test, choose API test and select New API draft.
On Request, choose Authentication and enter the Environment credential binding name.
This name matches the selected environment at run time. It is not a reference ID or a credential value.

| Authentication | Operator secret value | Injected header |
| --- | --- | --- |
| Bearer token | Token without a Bearer prefix. | Authorization: Bearer followed by the token. |
| Basic authentication | username:password. | Authorization: Basic followed by the encoded credential. |
| API key header | Complete API-key value. | The reviewed X- header or Api-Key. |

Custom credential headers cannot change routing or override the request method.
After policy validation, the gateway replaces a caller-supplied value for the selected authentication header.
Credentials apply only to the selected origin and reviewed methods and paths. Redirects remain disabled.

Manifest requests.authentication declares binding, scheme, and an optional header.
Manifest requiredBindings and authenticated test requiredBindings declare the same single binding. Preparation validates these declarations together.
Save, prepare, review, publish, and run remain separate operations.
A missing-credential test can omit its test-level requiredBindings while retaining the manifest authentication declaration.
That test receives no injected credential and can assert the expected unauthorized response.

## Runtime and recovery

Credential execution requires a gateway labeled teststudio.api-credentials=gateway-injection/v1. Older gateways reject authenticated admission.
The source runtime retains request-policy/v1. Rebuild the gateway and restart the configured server to enable this profile.

The execution host resolves the reference immediately before gateway startup.
Provider failures return fixed errors that exclude private upstream diagnostics.
The gateway receives its target policy and optional header through a one-use private socket initialized through Docker exec stdin.
The socket lives in private tmpfs outside the worker output volume. Initialization removes its path.
Secrets enter no process arguments, Docker environment, source image, work-item JSON, or stored run snapshot.
Run snapshots retain binding names and environment reference IDs.

The gateway removes literal and common encoded credential reflections from response bodies and headers before the worker receives them.
Nonsensitive fields and status codes remain available for assertions. Reflected credentials become [REDACTED].
The collector also filters known values before retaining diagnostics.
Recovery imports already filtered responses without resolving the current credential again.
Rotation therefore requires no persisted copy of earlier resolved secrets.
Cancellation, deadlines, and failed initialization remove the owned gateway and its private memory filesystem.

Use dedicated scoped credentials and reviewed data. Deliberately encoded or unlabeled private application data remains outside this conventional filter.
See [Diagnostic redaction](api-testing.md#diagnostic-redaction) for evidence formats, limits, and omissions.

## Current scope

This profile supports request-restricted Playwright and Go API tests through the local gateway.
Browser session credentials, native HTTP-check authentication, and hosted verification remain separate work.
Personal recording cookies and local browser sessions do not become execution credentials automatically.
