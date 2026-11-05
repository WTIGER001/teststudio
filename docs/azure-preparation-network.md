# Azure preparation network

## Native boundary

Native Azure assembly now requires a network object alongside worker, writer, readerClientId, and host.
The publisher checks this network after the existing host and writer chain, before selecting registry publication credentials.
Every new assembly also refreshes the chain. Owned cleanup retains its existing bounded journal behavior.
This is the trusted assembly process. It executes no uploaded source and grants no admission to untrusted Azure Jobs.

## Network settings

Copy networkBinding from the separate network template into the private settings document's network field.
The fields describe these selected resources:

| Field | Requirement |
| --- | --- |
| sourceSecurityGroup | Separate NSG attached only to the preparation source subnet. |
| natGateway and natPublicIp | Dedicated Standard NAT gateway and one regional static IPv4 public address. |
| databaseIp | Usable private PostgreSQL address in the controller VNet spaces, outside source, registry endpoint, file endpoint, and worker ranges. |
| storageAccount | Exact journal account resource identifier matching host.journalAccount. |
| fileEndpoint and fileInterface | Separate journal-only file private endpoint and its one NIC. |
| fileSubnet and fileSubnetPrefix | Dedicated endpoint subnet in the controller VNet, separate from source, registry endpoints, and the core controller subnet. |
| fileSecurityGroup | Separate NSG attached only to the journal endpoint subnet. |
| fileIp | Usable private endpoint address in fileSubnetPrefix. |
| fileZone | Selected privatelink.file.core.windows.net zone. |

All resources use the installation subscription. Selected resource groups may differ.
The account name supports Azure's lowercase letter and number rules, including a leading number.
Subnet prefixes remain canonical private IPv4. File endpoint prefixes support /29 or larger networks.
The source settings retain the existing /27 or larger requirement and controller VNet boundaries.
Resource aliases, overlapping endpoint ranges, worker addresses, and reserved endpoint addresses reject before native startup.
The current adapter uses public Azure service authorities and domains.

## Source egress

The source subnet reports the selected NSG and NAT gateway, explicit disabled default outbound access, and one successful Microsoft.App/environments delegation.
Unexpected route tables, service endpoints, private endpoints, and unsupported allocations reject this profile.
The NSG retains the existing Container Apps internal, load-balancer, registry, identity, and platform DNS rules.
It additionally permits HTTPS to AzureResourceManager, AzureKeyVault, and AzureMonitor service tags.
Registry HTTPS selects only the configured registry and regional data private addresses.
PostgreSQL selects databaseIp on TCP 5432. The journal selects fileIp on TCP 445.
The final inbound and outbound rules deny remaining traffic. The policy also retains the existing explicit metadata and other-platform-DNS denials.
This permits the platform dependencies of a trusted background worker without applying that broader policy to untrusted tests.
See Microsoft's [Container Apps NSG requirements](https://learn.microsoft.com/en-us/azure/container-apps/firewall-integration).

The NAT gateway reports one public address and only the source subnet, with no address prefixes, IPv6 addresses, or additional consumers.
Its idle timeout is ten minutes. The public address reports static regional Standard IPv4 allocation and the exact NAT backlink.
The controller VNet reports its configured address spaces and uses default Azure DNS.
The direct source and file subnet reads must match their embedded VNet values. File ranges cannot overlap another reported subnet.
Custom DNS servers and a VNet default NAT gateway reject this fixed profile.

## Journal account, endpoint, and DNS

The journal account reports disabled public networking, default-deny network access, and no service, IP, virtual-network, or resource-instance bypass.
Its file authority must be the exact host.journalAccount file endpoint.
The account requires HTTPS, TLS 1.2, no public Blob access, and Shared Key compatibility for the platform SMB mount.
StorageV2 and FileStorage accounts are supported. SFTP, local-user, and NFS v3 features reject this profile.
One service-side private endpoint connection must select the journal endpoint, report successful provisioning, and be approved.
The exact share must report SMB, a positive quota, and live non-snapshot state.
Missing protocol or approval metadata remains unavailable; it does not silently establish SMB access.
The management SDK uses the documented [selected share GET](https://learn.microsoft.com/en-us/rest/api/storagerp/file-shares/get?view=rest-storagerp-2024-01-01), without snapshot or expansion requests.

The network-side endpoint must select that account's file group and its exact NIC, subnet, and address.
The shared endpoint and NIC verifiers reject other members, public addresses, alternate attachments, or changed approvals.
The dedicated subnet enables endpoint network policies and contains only that endpoint and its reported NIC configuration.
Its NSG permits only source-subnet SMB to fileIp, followed by fixed deny rules.

The private zone contains the one account A record and normal SOA record, with the existing bounded TTL and inventory checks.
One completed preparation-journal link selects the controller VNet, disables registration, and permits no public fallback.
The same account name is used by the host's AzureFile binding, management account, private endpoint service, and DNS record.
See [Azure Files DNS forwarding](https://learn.microsoft.com/en-us/azure/storage/files/storage-files-networking-dns) for private file naming.

## Actual journal mount

Before database access or cloud startup checks, the command inspects the current Linux process's bounded mount table.
The journal directory must be one writable CIFS mount of the configured account's canonical file share, with no subpath substitution.
An absent mount, temporary filesystem, other share, read-only mount, or ambiguous duplicate rejects startup.
The existing private directory checks still apply.

The command then creates one private probe, writes and synchronizes it, and synchronizes the containing directory.
It removes the probe and synchronizes that removal. Failure stops startup before any source job is claimed.
The probe uses no source data or credentials and does not inspect existing journals.
This verifies current mount identity and basic local synchronization behavior. It does not prove recovery on another replica.
The platform storage binding still owns the SMB credential. No storage key is loaded into the preparation settings or command arguments.

## Deployment package and sequence

[preparation-network.bicep](../deploy/azure/preparation-network.bicep) is a separate installation template.
It declares the source NSG, Standard NAT and public address, delegated source subnet, private journal subnet, account, SMB share, endpoint, and DNS resources.
The journal account has deletion protection. The template retrieves no storage keys or secret values and creates no application or test job.
It targets the resource group containing the installation-owned controller VNet. The VNet itself already exists.
The separate [worker template](azure-native-assembly.md) remains responsible for identities, role assignments, secrets, process, and mount declarations.

For the later deployment session:

1. Supply the controller VNet and distinct subnet names and prefixes. Match the existing registry private addresses and private PostgreSQL address.
2. Apply the reviewed network template. Retain its account, share, and journal data during worker replacement.
3. Create the dedicated internal Consumption environment in sourceSubnetId and its read-write AzureFile storage binding for this account and share.
4. Configure the binding's SMB credential through the platform's supported private credential mechanism. The template performs no automatic key retrieval.
5. Bootstrap the worker template with workerEnabled=false, then merge its writer, readerClientId, and host outputs with networkBinding and the existing worker and registry settings.
6. Save that complete private configuration as the named vault secret before enabling the worker.

Keep the PostgreSQL connection on the configured private address and TCP 5432, with the existing verified TLS settings.
Do not substitute the core evidence Blob account or an ephemeral journal directory.
The account lock protects the resource; it does not implement TestStudio evidence retention, quotas, or pins.
Complete source review still requires compatible isolated discovery.

Compile the network package locally without Azure CLI, account credentials, or deployment:

```sh
python3 scripts/verify-azure-package.py --profile=preparation-network \
  --bicep=.data/azure-tools/bicep \
  --output=output/azure-preparation-network-package
```

Use --compiled-template with preparation-network.json after checker-only edits. The core and worker profiles retain their existing commands.

## Bounds and remaining evidence

The native publisher owns thirteen readers. The added network and DNS readers close with the existing publisher lifecycle.
The network check has a 65-second total bound, including the host's 55-second chain.
Eleven selected GETs run concurrently, followed by the shared zone, record, and link observations.
The stage adds fourteen requests normally and at most seventeen with bounded DNS paging.
Network calls use API 2025-03-01, storage calls use 2024-01-01, and private DNS calls use 2024-06-01.
Each operation retains its five-second context and four-megabyte response limit.
Alternate resources, credential operations, lists, expansions, mutations, and redirects reject the restricted SDK transport.
The pinned armstorage v1.8.1 dependency adds the two management reads without changing existing SDK versions.

Five focused network checks and one journal mount and probe check pass locally.
Changed native and publisher regressions pass. The network, worker, and core package checks also pass, reusing existing compiled templates where applicable.
Fixtures use fake credentials and in-memory SDK transports. The journal probe check uses an owned local temporary directory and removes its files.
No real registry, identity endpoint, cloud service, image, container, browser, or deployment runs.

Reported policy does not prove complete effective routing, cross-scope DNS visibility, exclusive authority, or prevention of later changes.
Actual mount behavior, remote durability, identity enforcement, and restart recovery remain separate deployment evidence.
Untrusted Azure discovery and native execution admission remain unfinished. Acceptance 51 advances; the count remains 48 verified of 57.
