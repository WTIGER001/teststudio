# Notification transports and secret bindings

## Server-side bindings

The local secret adapter resolves registered environment variables or mounted files.
Bindings identify one project, reference, purpose, source, and network scope.
Configuration stores source names and references. It does not store resolved secret values.
Reference listings expose only reference identifiers and display names.
A reference cannot resolve another project's secret or another purpose.
Reads reject empty values, oversized values, and non-regular files.
The resolved value does not serialize into JSON.

The notification webhook purpose is `notification-webhook`.
Its value contains the complete webhook URL, including any provider token.
The sender resolves that value on the server for each delivery attempt.
Environment and file bindings support native local deployments and mounted secret integrations.
AWS and Azure secret service adapters remain pending.

## Network rules

Native probes and notifications share address validation and pinned DNS dialing.
Public scope rejects private, loopback, link-local, multicast, unspecified, and known metadata destinations.
Private scope permits configured internal addresses. Link-local and metadata addresses remain blocked.
Loopback scope permits only local loopback addresses.
Each connection validates all resolved addresses before dialing a selected address.
The transport does not use environment proxy settings or follow redirects.
TLS uses certificate verification and a minimum version of 1.2.
Plain HTTP requires a binding with private or loopback scope.

## Generic webhook

The sender posts a versioned JSON run summary.
The body includes project and run identity, suite, target stage, outcome, counts, evidence state, and an authenticated UI link.
It includes no resolved secret, authorization header, test logs, response bodies, screenshots, or private artifact links.
`X-TestStudio-Delivery-ID` carries the stable delivery identifier.
Receivers can use that identifier to suppress retry duplicates.

Successful 2xx responses confirm delivery.
Connection uncertainty, 408, 425, 429, and server errors permit bounded retries.
Other unsuccessful responses remain permanent delivery failures.
Errors expose status numbers and fixed diagnostic text. They do not expose destination URLs or response bodies.
The transport bounds request duration, response headers, message size, and rate-limit response reads.

## Discord webhook

Discord delivery uses `wait=true` to request server confirmation.
Messages include summary counts and a run link. Mention parsing stays disabled.
The sender respects bounded `Retry-After` values and Discord's `retry_after` field.
See the [Discord webhook contract](https://docs.discord.com/developers/resources/webhook#execute-webhook) and [rate-limit contract](https://docs.discord.com/developers/topics/rate-limits).

## Verification checkpoint

Focused local checks verify project and purpose isolation, bounded secret reads, payloads, stable delivery identifiers, and redacted errors.
They also verify network scope, redirect rejection, Discord confirmation, mention suppression, and rate-limit parsing.
The durable outbox, service configuration, notification interface, and complete delivery workflow remain under implementation at this transport checkpoint.
