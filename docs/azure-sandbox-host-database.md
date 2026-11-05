# Azure sandbox controller private database

## Introduction

OpenRuntime now includes SDKHostDatabase before controller registry, host, and workload admission.
It joins the selected PostgreSQL flexible server, private endpoint, NIC address, endpoint subnet rules, DNS, and reported service approvals.
The source network stage permits TCP 5432 only to that private address.
These are reported configuration checks. Native command selection and actual hosted database access remain unfinished.

## Concepts

### Required settings

host.database contains these settings:

| Field | Required binding |
| --- | --- |
| server | Selected PostgreSQL flexible server in the installation subscription. |
| endpoint, interface | Dedicated private endpoint and NIC. |
| subnet, subnetPrefix | Separate endpoint subnet within the controller VNet. |
| securityGroup | Exclusive endpoint security group. |
| address | Usable private IPv4 address inside that subnet. |
| zone | Dedicated privatelink.postgres.database.azure.com zone. |
| consumers | Optional list of up to seven other approved database endpoint identifiers. |

host.databaseHost equals SERVER_NAME.postgres.database.azure.com.
The command retains its [hostname and verified TLS binding](database-host-binding.md).
The source, registry endpoint, firewall, and database endpoint subnets must fit inside the controller VNet without overlap.
Protected resource aliases, foreign resources, duplicate consumers, missing fields, and reserved or public addresses reject configuration.
Host-specific settings retain the separate host profile and preserve the shared converted-image pool.

### Private network path

The controller VNet contains exactly four selected subnets, with matching embedded and selected observations.
The database endpoint targets the selected server through group postgresqlServer.
Its NIC binds the postgresqlServer member to the configured address.
The endpoint subnet contains only that endpoint and its exact NIC configuration.

The source security group permits TCP 5432 only to the selected database address.
The endpoint security group permits inbound TCP 5432 only from the controller subnet to that address.
Remaining endpoint traffic is denied by the fixed rules.
The database endpoint remains inside the controller VNet. It adds no public database rule or sandbox access.

The dedicated zone has one exact server-name A record and one controller-database VNet link.
Registration and public fallback remain disabled. Additional records, aliases, wildcards, links, and incomplete inventories reject admission.
The controller resolver approved list includes the public database hostname and its private-link name.
The reported subscription DNS inventory also requires the database zone and rejects unexpected links to guarded VNets.

### Reported server and approvals

The reader pins PostgreSQL metadata to API 2024-08-01.
The server must report its exact resource, region, database hostname, Ready state, disabled public network access, and enabled password authentication.
This profile selects Private Link and rejects delegated VNet-integration fields.
These fields follow the [server resource contract](https://learn.microsoft.com/en-us/rest/api/postgresql/servers/get?view=rest-postgresql-2024-08-01).

The complete private-link resource inventory contains one postgresqlServer group and member with the configured private-zone name.
The names follow the [private-link resource contract](https://learn.microsoft.com/en-us/rest/api/postgresql/private-link-resources/get?view=rest-postgresql-2024-08-01).

The complete connection inventory must approve exactly the selected endpoint and declared consumers.
The server's embedded approvals must independently associate the same endpoints with the same connection identifiers.
Missing, duplicate, foreign, pending, or incomplete approvals cannot establish ownership.
Declared consumers permit shared installation roles. Their separate network ownership remains required elsewhere in the protection composition.

Microsoft's connection-list sample reports Ready, while its enum and embedded-server sample report Succeeded.
The reader treats both as completed states, provided all ownership and approval checks pass.
This compatibility choice follows the conflicting [connection-list example and schema](https://learn.microsoft.com/en-us/rest/api/postgresql/private-endpoint-connections-part-1/list-by-server?view=rest-postgresql-2024-08-01).
Unknown and transitional states remain unavailable.
The reader retrieves no database credentials, secret contents, or SQL records.

### Bounds, cache, and packaging

The database stage observes ten selected GET paths with a five-second joined deadline.
Network resources use API 2025-03-01 and DNS resources use 2024-06-01.
Connection inventory permits at most eight entries and nine pages. Private-link resources permit one entry and two pages.
Existing response, continuation, redirect, retry, and diagnostic restrictions remain required.
Construction acquires no token and calls no service.

Creation and source import use fresh reads. Successful live evidence expires 15 seconds after observation start.
A current failure invalidates earlier success. Remaining protection and authorization stay uncached.
Runtime.Close closes the database reader after owned work joins.

HostDatabaseDefinitions returns five staged bodies: endpoint subnet, endpoint rules, private zone, link, and A record.
It creates no server, endpoint, NIC, database account, secret, or complete deployment.

## Tutorials

### Local verification

Three new checks and eleven affected checks pass with race detection in 5.631 seconds.
They cover both controller modes, declared consumers, settings, service approvals, endpoint and DNS drift, source rules, fresh failure, and shutdown.
Affected checks retain controller network, registry, host, resolver, scoped settings, and mode admission.
The central discovery composition observes 152 unique selected paths.
A focused settings recheck also passes in 1.594 seconds after tightening registry-consumer and zone alias rejection.
Evidence is output/azure-sandbox-host-database/summary.json, local.log, and settings-followup.log.
Checks use local SQLite, fake credentials, and in-memory metadata. Owned readers close through cleanup.
No Azure service, database connection, browser, container, UI build, demo restart, resource mutation, or remote push runs.

### Continue activation

Complete actual dependency access, remaining identity attachment, effective authority, deny effects, and protected writes.
Bind native HTTPS-probe target routing to registered-target and request-policy contracts.
Then complete native command selection and deployment packaging.
Verify actual SQL access, DNS, routing, certificate trust, token behavior, preview compatibility, and independent timers in the separate deployment phase.

Related guides: [source perimeter](azure-sandbox-host-network.md), [controller host](azure-sandbox-host.md), and [runtime](azure-sandbox-runtime.md).
Acceptance 51 advances. The total remains 48 verified of 57, with nine open.
