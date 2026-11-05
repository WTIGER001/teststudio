# PostgreSQL destination binding

## Introduction

The service and preparation commands accept --database-host to bind a protected database URL to one expected DNS hostname.
Use it when a deployment must reject a changed destination or a TLS downgrade in its database secret.
The option validates configuration before storage opens. It does not establish database reachability.
Omitting the option preserves existing local and hosted behavior.

## Concepts

### Bound connection contract

The selected hostname must be a lowercase DNS name with at least two labels.
IP literals, trailing dots, embedded ports, and multiple hosts reject configuration.
The protected variable must contain one postgres or postgresql URL with a username, nonempty password, and database name.
Its authority must match the selected hostname, with the default port or explicit port 5432.
The URL must select sslmode=verify-full.
The driver then verifies the server certificate against that hostname without a plaintext fallback.

Only these query options are accepted:

| Option | Contract |
| --- | --- |
| sslmode | Required verify-full. |
| sslrootcert | Optional system or a clean absolute certificate-file path. |
| connect_timeout | Optional integer from one through 30 seconds. |
| application_name | Optional nonempty value up to 64 characters. |

Duplicate options, unknown options, malformed query values, fragments, socket destinations, alternate-host fields, and service-file selectors reject configuration.
Nonempty environment variables whose names start with PG also reject configuration when the binding is enabled.
This prevents ambient PostgreSQL driver settings from changing the admitted connection.
Empty PG variables are permitted. The commands do not modify the process environment.

The validator accepts at most 16 KiB of connection data and reads no certificate, service, password, or configuration file.
The driver loads configured certificate trust during normal database initialization after validation.
Validation errors contain no connection value, credentials, or private parser diagnostics.
Keep the connection URL in its protected environment reference.

### Command roles

The service accepts the option for its existing roles, including the dedicated execution controller and worker backend.
The preparation command accepts it for normal, assembly-only, and discovery-only operation.
Existing mode, provider, location, and flag restrictions still apply.
The binding does not enable a provider or change which process owns execution or discovery.

### Staged Azure controllers

Sandbox host settings now require host.databaseHost.
Both expected controller commands include --database-host with that value.
Reported command drift rejects host admission, including replacement with another otherwise valid database hostname.
The controller resolver approved list includes the selected database name.
The host profile binds the setting without changing the shared worker or converted-image profile.

The [private database stage](azure-sandbox-host-database.md) now checks reported server, endpoint, DNS, and TCP 5432 network bindings.
Native sandbox command selection, actual database access, and full deployment packaging remain unfinished.
This configuration guard does not replace endpoint ownership, DNS records, source rules, routing, or effective dependency permissions.

## Tutorials

### Configure a bound service

Supply a protected TESTSTUDIO_DATABASE URL with the required verified TLS setting, then select the expected hostname:

```sh
bin/teststudio --database-env=TESTSTUDIO_DATABASE \
  --database-host=database.example.test --require-postgres
```

Use the same options with bin/teststudio-prepare and its existing provider and mode flags.
For a separate execution controller, retain --serve-role=controller, its explicit address, artifacts, and matching execution profile.
Remove conflicting PG environment settings from that process rather than adding routing overrides to the URL.
Configure private network access separately before enabling a hosted worker.

### Local verification

Four new checks and ten affected checks pass with race detection across four packages.
Runtime configuration takes 1.275 seconds, the server command 7.627 seconds, preparation 1.990 seconds, and sandbox contracts 2.230 seconds.
Checks verify admitted driver hostname, port, TLS verification, certificate name, and absence of fallbacks without making a database connection.
Invalid command configurations reject before storage, listeners, provider settings, and preparation directories are opened.
Affected checks retain existing database defaults, role restrictions, controller host definitions, resolver policy, and scoped settings.
The actual local controller still completes one queued HTTPS probe through SQLite and an owned loopback TLS target.
Its child process, listener, and clients close through the existing cleanup.
Evidence is output/database-host-binding/summary.json and local.log.
No cloud service, browser, container, UI build, demo restart, resource mutation, or remote push runs.

Related guides: [service image](service-image.md), [headless controller](execution-controller.md), and [sandbox host](azure-sandbox-host.md).
Acceptance 51 advances. The total remains 48 verified of 57, with nine open.
