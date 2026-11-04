# Artifact storage profiles

The server supports filesystem, S3, and Azure Blob evidence storage.
All three use the same evidence service for project permissions, manifests, quotas, pins, retention, and interrupted upload recovery.
Storage selection is an operator setting. Testers use the existing Run details controls.
The separate artifact origin continues to serve private access grants and traces.
The server does not expose bucket credentials, account keys, or provider download links to the browser.

## Select storage

Without a configuration file, the server uses the existing --artifact-directory setting.
The default directory is .data/artifacts. Local startup requires no cloud credentials or account.
Keep filesystem evidence outside the public UI directory.

Select another profile with this flag:

```sh
bin/teststudio --artifact-store-config /etc/teststudio/artifact-store.json
```

The file contains one JSON object and accepts up to 64 KiB.
Container operators can supply the same JSON through an environment reference:

```sh
bin/teststudio --artifact-store-env TESTSTUDIO_ARTIFACT_CONFIGURATION
```

The environment option conflicts with --artifact-store-config. Missing, blank, malformed, and oversized selected values fail before storage opens.
Both forms reject unknown fields and trailing JSON. Errors exclude configuration values.
Filesystem directories in environment configuration must be absolute.
Storage JSON contains provider settings and references. AWS credentials and Azure account keys remain outside this document.
Unknown fields, mixed profiles, embedded credentials, and noncanonical object prefixes fail validation.
The service does not create buckets, storage accounts, containers, roles, or provider lifecycle rules.
Provision those resources separately before enabling their profile.

A filesystem configuration can select a private persistent directory:

```json
{
  "driver": "filesystem",
  "directory": "/var/lib/teststudio/artifacts"
}
```

Relative directories in a configuration file resolve beside that file.
The existing --artifact-directory value applies only when neither configuration source is supplied.
Mount durable storage when using a container. Multiple instances require shared object storage or a suitable shared filesystem.

## S3

```json
{
  "driver": "s3",
  "s3": {
    "region": "us-east-1",
    "bucket": "teststudio-evidence",
    "prefix": "teststudio/installation-one",
    "expectedOwner": "123456789012"
  }
}
```

Replace the sample region, bucket, prefix, and account identifier with your installation values.
The region and bucket are required. Prefix and expectedOwner are optional.
Use expectedOwner to reject operations against a bucket in a different account.
Assign a distinct prefix to installations that share a bucket.
An optional kmsKeyId selects a customer-managed KMS key for uploads.
Without that field, the bucket's configured encryption applies.

The AWS SDK resolves credentials from its standard configuration chain.
Use the configured ECS task role, workload identity, or an explicitly selected local profile.
The storage configuration accepts no access key or secret access key.
Grant GetObject, PutObject, and DeleteObject for the selected bucket prefix.
A selected KMS key also requires the matching key policy and cryptographic permissions.
Keep the bucket private. The adapter sets no public object ACL.
Uploads provide their size, SHA-256 checksum, private cache policy, and hash metadata.

This profile uses the regional S3 endpoint. Custom S3-compatible endpoints are not exposed by the operator configuration.
See the official [AWS SDK credential configuration](https://docs.aws.amazon.com/sdk-for-go/v2/developer-guide/configure-gosdk.html).

## Azure Blob

```json
{
  "driver": "azure",
  "azure": {
    "serviceUrl": "https://example.blob.core.windows.net",
    "container": "teststudio-evidence",
    "prefix": "teststudio/installation-one"
  }
}
```

Replace the sample service origin and container with your installation values.
Service URL and container are required. Prefix is optional.
The service URL requires HTTPS and permits no credentials, path, query, or fragment.
SAS tokens and account keys do not belong in this file.

The profile uses DefaultAzureCredential when managedIdentityClientId is absent.
Use the deployment's workload identity or managed identity, or a configured local developer identity.
An optional managedIdentityClientId selects that exact user-assigned managed identity instead of the default chain.
Assign Storage Blob Data Contributor on the required container or a narrower custom role with the necessary data operations.
The server requires blob read, write, and delete operations. It does not need permission to create a storage account.
Keep anonymous container access disabled.
The adapter uses block blob uploads, transfer checksums, private cache policy, and SHA-256 metadata.
Azure configuration does not load AWS credentials or call AWS services.
See the official [Azure Blob identity setup](https://learn.microsoft.com/en-us/azure/storage/blobs/storage-blob-go-get-started).

## Retention, failures, and moving storage

Object paths retain the existing project, run, artifact, and checksum components beneath the configured installation prefix.
Quotas track reserved logical bytes in PostgreSQL or SQLite. The provider's billing usage can include additional retained object versions.
Pins prevent TestStudio cleanup. They cannot override a provider rule that deletes objects independently.
Align provider lifecycle rules, version retention, soft deletion, backups, and encryption with the installation's evidence policy.

Missing objects map to the existing missing-content state.
A missing bucket or container remains a storage configuration error. It does not imply that an individual artifact expired.
Upload or deletion failures retain their normal manifest and quota state for recovery.
Interrupted upload recovery reads from the selected provider and verifies recorded size and SHA-256 before completing its manifest.
See [Evidence](evidence.md) for upload leases, retry limits, pins, and expiry processing.

Changing the backend, bucket, container, account, directory, or prefix does not copy existing bytes.
Copy retained objects with their exact relative keys before selecting a different location for existing metadata.
Preserve metadata and storage together during backup and restore. Keep all instances on the same installation profile.
Use [Database transfer](data-transfer.md) to move metadata and native permission state between SQLite and PostgreSQL.
Source bundle staging and prepared execution images remain under their separate source and runtime storage configuration.

Cloud uploads stage a bounded source in a private temporary file before sending it.
Provide sufficient private temporary disk space. A successful or failed call removes its staging file.
Upload size checks complete before a provider write. Provider calls retain the caller's cancellation and deadline.
Artifact keys reject traversal, ambiguous segments, control separators, and keys beyond the supported length.

## Verification status

Focused local checks use real SDK clients against in-process HTTPS protocol fixtures with test-only identities.
They verify request signing or bearer authentication, namespaced keys, size checks, checksums, reads, idempotent deletion, and missing-object mapping.
Common evidence checks verify duplicate admission, quotas, pins, expiry, and retained manifests on SQLite and local PostgreSQL.
Configuration checks preserve credential-free local defaults and reject mixed profiles or credential-bearing settings.
The private temporary-file check verifies bounded reads, cancellation, file permissions, and cleanup.

These checks do not verify a deployed S3 bucket, Azure account, workload identity, KMS policy, or provider lifecycle configuration.
Acceptance 46 remains open until the required provider verification is complete.
AWS, Kubernetes, and Azure deployment checks remain separate work. No cloud resources are contacted by these local fixtures.
