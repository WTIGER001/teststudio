# Publisher CLI sign-in

The publisher CLI supports browser confirmation, local passwords, and the optional loopback administrator mode.
Every method creates a normal managed TestStudio session. Existing roles and project permissions still apply.

## Browser confirmation

Use browser confirmation for local accounts or configured OIDC accounts:

```sh
go build -o bin/teststudio-publish ./cmd/teststudio-publish
bin/teststudio-publish login --server https://teststudio.example \
  --browser --client-name "Publisher on my workstation"
```

The CLI opens the installation in the system browser and prints a terminal code.
Use no-open when you need to open the address manually:

```sh
bin/teststudio-publish login --server http://127.0.0.1:8040 \
  --browser --no-open --client-name "Local publisher"
```

The browser uses the installation's normal sign-in screen.
Choose a local password, enabled loopback access, or a configured OIDC provider.
OIDC returns to the original confirmation page, including its client name.

Review the account and client name. Enter the code printed in the terminal.
Select Approve CLI sign-in. The terminal saves its new session file with mode 0600.
Select Decline CLI sign-in to issue no session.
The request expires after five minutes. Start a new terminal request after expiry or cancellation.
The confirmation page includes contextual help and preserves its code while help is open.

## Session behavior

Browser approval creates a separate CLI session. It does not replace or close the browser session.
The CLI session retains the account, authentication method, provider revision, credential revision, and authentication expiry.
Its absolute expiry cannot exceed the browser authentication expiry or the original session expiry.
Account disabling, credential changes, provider changes, and permission revocation follow the existing authorization rules.

The terminal keeps a random private proof. The browser URL contains its bound challenge and client name.
The proof binds the request time and client name. It never appears in the browser URL or terminal output.
Anonymous polling creates no durable authorization records.
The server stores a request only after an authenticated browser approves or declines it.
Approval requires the current browser session, exact request origin, and CSRF token.
The server consumes the request atomically when it issues one managed CLI session.
Competing exchanges cannot issue multiple sessions. Replayed proofs return no credential.

Provider access tokens, ID tokens, refresh tokens, and client secrets stay outside the CLI.
The CLI stores only the installation origin, managed session cookie, CSRF token, and expiry.
Its session file uses the existing private publisher session schema.
Use session-file to keep distinct files for separate installations.
A successful browser login replaces the selected CLI session file and attempts to revoke its previous session at the same origin.
If session-file saving fails, the CLI cancels the request and attempts to revoke its issued session.
Expired confirmation records are removed by service maintenance.

## Existing sign-in methods

The local administrator command remains available only against an explicitly enabled loopback installation:

```sh
bin/teststudio-publish login --server http://127.0.0.1:8040 --local
```

For managed local passwords, use user and password-env as described in the prepared suite guide.
Password values do not appear in command arguments.
Browser confirmation also supports these accounts without placing their passwords in the CLI process.
A required password change must finish in the browser before approval becomes available.

## End or revoke a session

```sh
bin/teststudio-publish logout
```

Logout revokes that CLI session and removes its private session file.
It does not close the browser session that approves it.
Authorized user session administration shows the CLI client name and supports normal session revocation.
Revoking the approving browser before exchange blocks CLI issuance.
After issuance, browser and CLI sessions remain independently revocable.

## Local verification

STUDIO-CLI-001 starts the actual publisher binary inside an isolated local TestStudio self-test worker.
It verifies local browser approval, real OIDC browser sign-in against a new local fixture, denial, private files, and independent sessions.
It also checks the help drawer, retained form input, dark theme, mobile width, and owned process cleanup.
Focused Go checks verify replay, concurrent exchange, proof tampering, expiry, revocation, private output, and failed file delivery.
The existing local CLI publication check still passes.
No cloud deployment, live identity provider, or broad browser matrix is required for these checks.
