import type { HelpPage } from './help.models'

export const helpPages: Record<string, HelpPage> = {
  'test-history': {
    id: 'test-history', title: 'Test history',
    introduction: ['Follow a permanent test identity through its saved runs, definitions, and retained evidence. The test or suite can be archived.'],
    concepts: [
      { title: 'Active application', description: 'The sidebar Application switcher scopes Suites, Test catalog, Test runs, and Trends. The context above the page shows the active application. Choose All applications for project-wide work. Project overviews contains All applications and HTTPS probes. Switching applications keeps the main destination and clears the previous filters.' },
      { title: 'Permanent identity', description: 'The project, suite, and test identifier define this history. Identical test identifiers in other suites keep separate histories.' },
      { title: 'Saved runs', description: 'Run titles, release, outcomes, and durations come from saved snapshots. Editing or archiving a test does not rewrite them. A missing test result displays not-recorded, never a passing result.' },
      { title: 'Archived and removed definitions', description: 'Archived tests and suites keep their history. If the working definition is missing, the last saved revision provides context. History does not restore a definition or enable execution.' },
      { title: 'Evidence and permissions', description: 'Inspect run and evidence opens the original run with this test selected. Current run and artifact permissions still apply. Normal retention applies; use an evidence pin for an investigation.' },
      { title: 'Older history', description: 'History scans are bounded. Load older runs continues the saved query. An empty partial scan does not prove that older matching runs do not exist.' }
    ],
    tutorials: [{ title: 'Investigate an old test', steps: ['Open a test title or History from the catalog, a retained release, or a run result.', 'Read the lifecycle and archived-suite notices. Compare the current title with each saved run title.', 'Select Inspect run and evidence to inspect the original attempts, logs, screenshots, traces, and attachments for this test.', 'Use Return to run when this page opened from a run. Test trends opens its stacked state history.', 'Expand Advanced revision history to inspect saved definitions. Load older runs when the page has a continuation.', 'If access changes, select Refresh access. Missing or denied data must not be read as a passing test.'] }]
  },
  releases: {
    id: 'releases', title: 'Suite releases',
    introduction: ['Browse immutable suite releases, compare definitions and artifacts, and select a release for future runs.'],
    concepts: [
      { title: 'Retained releases', description: 'Each release freezes its test definitions, timing and evidence policies, execution artifact, and source provenance. Archived suites retain readable releases.' },
      { title: 'Activation and rollback', description: 'The active release supplies future runs that follow the suite. Activating an older retained release rolls the suite back. Existing runs and pinned schedules keep their own release.' },
      { title: 'Comparisons', description: 'Choose a baseline under Compare releases. The table identifies added, removed, and changed definitions. Separate notices identify execution artifact, source provenance, timing policy, and evidence policy changes.' },
      { title: 'Current authority and conflicts', description: 'Activation checks current publishing permission and the displayed suite revision. A conflict requires refreshing before another activation. Archived suites require restoration first.' }
    ],
    tutorials: [{ title: 'Inspect and compare releases', steps: ['Select Suite for retained releases. Search by identifier, author, repository, or commit when needed.', 'Select Inspect on a release. Review its publication details, definitions, image, and recorded source.', 'Choose another release under Compare selected release against. Review changed fields and artifact notices.', 'Use View recorded commit to inspect the exact GitHub commit when repository provenance is available.'] }, { title: 'Roll back a suite', steps: ['Inspect the retained release you want to select.', 'Review its definitions and compare it with the active release.', 'Select Activate selected release. This does not start tests or change existing run snapshots.', 'If a conflict appears, refresh retained releases and review the current state before activating again.'] }]
  },
  'cli-signin': {
    id: 'cli-signin', title: 'CLI sign-in',
    introduction: ['Confirm a TestStudio publisher CLI request using your current browser account. Local and configured OIDC sign-in methods use the same confirmation.'],
    concepts: [
      { title: 'Terminal code', description: 'Start sign-in in your own terminal. Review the client name and account, then enter the code printed by that terminal. A request expires after five minutes.' },
      { title: 'Separate session', description: 'Approval allows that CLI to exchange its private proof once for a separate managed session. The browser keeps its session. Provider tokens remain on the server.' },
      { title: 'Current access', description: 'The CLI uses your existing roles and project permissions. Revoked grants, disabled accounts, password changes, and provider changes follow the normal authorization rules.' },
      { title: 'Decline or expiry', description: 'Declining creates no CLI session. Expired, consumed, or cancelled requests require a new terminal sign-in. Refresh the request after an uncertain approval response.' }
    ],
    tutorials: [{ title: 'Sign in from the publisher CLI', steps: ['Run teststudio-publish login with --server and --browser. Use --no-open to copy the address manually.', 'Sign in through the browser when required. Review the displayed account and client name.', 'Enter Terminal code exactly as printed in your terminal, then select Approve CLI sign-in.', 'Return to the terminal. It saves its managed session privately. Run logout when that CLI session is no longer needed.'] }, { title: 'Reject a request', steps: ['Select Decline CLI sign-in when you do not want to connect this terminal.', 'Start a new request in your own terminal if the client name, account, or code is incorrect.', 'Use user session administration to revoke an issued CLI session when necessary.'] }]
  },
  'create-test': {
    id: 'create-test', title: 'Create test',
    introduction: ['Choose a test type before entering its definition. Browser tests begin with recording. HTTPS probes and API tests use Name, Request, and Review steps. API tests continue into a code editor. Go suites use source preparation.'],
    concepts: [
      { title: 'Playwright recording', description: 'The local recorder opens a dedicated browser and Playwright Inspector on the TestStudio server desktop. Record clicks and assertions, stop the session, then review the generated test. Sessions remain private to their recording owner until saved as a draft.' },
      { title: 'HTTPS probes', description: 'Enter a title and choose a suite. Identifiers are generated under Advanced details. Paste a complete website URL and choose the accepted status codes. HTTPS needs no port. Review target configuration and safety before saving a draft or explicitly saving and running.' },
      { title: 'API credentials', description: 'Authentication supports Bearer, Basic, and API-key headers. Enter the environment binding name, not a token or password. Operators register references and exact permitted origins. Environment registration maps those references under Advanced credentials. The gateway supplies the selected credential and removes reflected values before the response reaches test code.' },
      { title: 'API tests', description: 'Choose API test and a destination suite, then select New API draft. Configure its first request with Name, Request, and Review steps. Advanced request policy holds endpoint, path-prefix, payload, and request-count limits. Create source draft opens editable TypeScript and a needs-review catalog. Save, prepare, review, publish, and run remain separate actions.' },
      { title: 'Go suites', description: 'Upload a complete source bundle with its suite manifest, catalog, and dependency declarations. Follow preparation, review the discovered tests, and publish a release. Repository connections remain under Import and sync tests.' },
      { title: 'Review and execution', description: 'A recorded test receives a permanent tag and a needs-review safety label. Saving does not start a run. New recording conversions visibly bind start-origin navigation to the selected registered environment. Save, prepare, review, and publish separately. The local website profile permits one endpoint; the offline starter stays isolated.' }
    ],
    tutorials: [
      { title: 'Record a browser test', steps: ['Select Playwright browser test and choose a destination suite.', 'Enter the start address, test identifier, and title. Select Start browser recording.', 'Use the dedicated browser and recorder assertion controls. Return here and select Stop and keep recording.', 'Select Review recorded test. Review the visible TESTSTUDIO_BASE_URL binding, assertions, and safety metadata. Save Playwright draft.', 'Validate and prepare the draft. Review the catalog and publish an active release.', 'In Test runs, choose the suite and a worker-reachable registered environment with an api origin. Review the selection and start the run. Container localhost cannot reach your desktop server.'] },
      { title: 'Create an HTTPS probe', steps: ['Select HTTPS probe. Enter a title, choose a suite, and select Continue.', 'Paste the website URL and enter accepted status codes. Existing target settings are reused when one target matches. Use a registered environment remains available for endpoint aliases and relative paths.', 'Select Continue. Review the request, target protection, safety label, and rationale. New addresses list the target and any application that will be saved.', 'Select Save draft to open the catalog. With publication and run permissions, Save and run publishes a snapshot without changing the active release and starts a local run.'] }
    ]
  },
  sources: {
    id: 'sources', title: 'Import and sync tests',
    introduction: ['Upload native HTTP checks, prepare a Go or Playwright ZIP bundle, register a GitHub repository, or inspect a project-owned Go or Playwright image. Review its catalog and publish an immutable release without starting tests.'],
    concepts: [
      { title: 'Browser recording', description: 'Use Create test to choose a browser journey or an HTTPS probe. The native local recorder captures browser clicks and assertions, then returns generated code for review. Review the target expression before saving. Prepare and publish the draft, then select a registered environment for local execution.' },
      { title: 'Playwright editor', description: 'Save browser source drafts without creating a ZIP. Edit TypeScript, catalog metadata, configuration, and helper files. Saving does not execute code. Preparation freezes the saved revision and generates pinned package and manifest files. The starter runs offline fixtures. Recorded drafts use the separate single-origin website profile and the selected environment’s api endpoint.' },
      { title: 'Repository sources', description: 'Bind one GitHub repository to several suite manifests. Sync resolves a branch, tag, or commit once. The worker retains its exact source snapshot. Approved private access stays on the server. Unchanged commits reuse preparation results unless you request another preparation.' },
      { title: 'Automatic synchronization', description: 'Signed push events require a separate approved webhook secret and one manual sync to verify repository identity. The worker resolves the current watched branch or tag. Periodic sync accepts 60 through 86400 seconds and coalesces missed intervals and pending work. Both methods check current source authority and never start test runs.' },
      { title: 'Assembly and discovery', description: 'Some hosted installations use separate workers for image assembly and test discovery. Waiting for discovery means the image is assembled, but no catalog review is ready yet. The discovery worker checks the retained source and current access before proceeding. Cancellation remains available while waiting. Review and publication still follow successful discovery.' },
      { title: 'Source bundles', description: 'A ZIP contains the complete source context, manifest, catalog, and dependencies. A separate worker builds offline Go or Playwright bundles. Browser dependencies must match the trusted runtime. Choose the manifest path relative to the ZIP root, then review its discovered catalog.' },
      { title: 'API request policies', description: 'API-typed Playwright and Go bundles bind the registered endpoint to reviewed method, path, payload, and request-count limits. Create test provides an API setup wizard and source editor. Protected API targets require a read-only policy and safe test labels. Use relative paths with the supplied request fixture. Go source uses go-environment and the supplied TESTSTUDIO_API_BASE_URL with standard net/http clients. Origin-only browser proxying does not enforce these request limits. API diagnostics redact sensitive text and JSON before retention. Media and opaque attachments produce visible capture gaps. Bearer, Basic, and API-key credentials use approved project references through the gateway. Configure the environment binding and declare it in the API wizard. Secret values stay outside test code.' },
      { title: 'Prebuilt preparation', description: 'Enabled workers inspect immutable images available through the configured execution location. The image must belong to this project. Choose an offline suite without a target environment. Jobs retain progress and diagnostics. Discovery does not create a test run. Review and publication remain explicit actions.' },
      { title: 'Staged source', description: 'The server stores the uploaded bytes and computes their SHA-256 digest. A staged file expires after 30 minutes. Published files remain available with their release.' },
      { title: 'Review and conflicts', description: 'Review identifiers, categories, types, safety labels, and before and after definitions. Existing catalog or suite changes invalidate the preview. Create a new preview after reviewing those changes.' },
      { title: 'Publication and activation', description: 'Publication applies the reviewed definitions and freezes a release in one transaction. Activation is optional and affects future runs. Publication never starts a test run.' },
      { title: 'Current upload limits', description: 'One upload accepts up to 30 native HTTP definitions and 1 MiB of UTF-8 source. Omitted tests remain in the suite. ZIP bundles support up to 32 MiB compressed and 128 MiB expanded. GitHub sources support manual sync, signed push events, periodic sync, and offline Go and Playwright runtime profiles.' },
      { title: 'Authority', description: 'Staging requires source management and catalog import and editing permissions. Publication also requires catalog publication. Only the staging user can publish that upload.' }
    ],
    tutorials: [
      { title: 'Write a Playwright test', steps: ['Choose an active suite without a target environment. Select New Playwright draft.', 'Edit the starter test and catalog.json. Keep permanent test tags and selectors aligned. Save Playwright draft.', 'Select Validate and prepare draft. Inspect Suite preparation below. Correct source errors in a new saved revision.', 'When ready, select Review prepared source and publish with optional activation. Start a run from Test runs. Save or discard local changes before switching suites.'] },
      { title: 'Register and sync a repository', steps: ['Select Register repository. In Repository, enter a name and owner/repository. Choose public or approved private access and a reference. Advanced repository options contains the generated permanent identifier and reference type.', 'Continue to Suites. Choose each destination once, its manifest path, and its Go or Playwright runtime. Back preserves your entries.', 'Continue to Review. Inspect the source and suite bindings. Advanced synchronization options contains periodic sync and signed push settings. Review enable state and automatic publication, then save. Inspect the source and select Sync now. Synchronization never starts tests.', 'Inspect a suite preparation. Its progress refreshes while work remains active. Refresh repository preparation requests current status immediately. When ready, select Review repository preparation. Publish with optional activation, or enable automatic publication in source settings.'] },
      { title: 'Configure automatic sync', steps: ['Edit a repository source. Choose an approved Push webhook access reference and enable signed push synchronization.', 'Save and inspect the source. After one manual sync, configure its displayed Webhook address in GitHub with the matching operator secret.', 'Set Periodic sync interval to 60 through 86400 seconds when polling is needed, or use 0 to disable polling.', 'Inspect Sync history and its Trigger. Disabling repository synchronization stops automatic and manual queueing without erasing releases.'] },
      { title: 'Prepare a source ZIP', steps: ['Choose a suite without a target environment. Select a complete Source ZIP.', 'Enter its manifest path relative to the ZIP root. Select Upload and prepare bundle.', 'Follow the queued build in Preparation details. Cancel active work or get the original source ZIP when needed.', 'When ready, select Review prepared source. Review the catalog and publish with optional activation. Tests start separately.'] },
      { title: 'Prepare an image', steps: ['Choose the destination suite. Select Prebuilt runner and enter its immutable image reference.', 'Select Prepare image. Use Preparation details to follow progress or cancel active work.', 'When preparation completes, select Review prepared source. Inspect the changes and image-reported provenance.', 'Publish with optional activation. This creates no test run. Source updates preserve managed metadata and retain removed test history.'] },
      { title: 'Upload and review', steps: ['Choose Suite for sources. Use Create or manage suites if the destination does not exist.', 'Select + Upload tests. Choose a JSON or YAML source file, or enter its name, format, and document.', 'Select Validate source and preview. Inspect the test changes and resolve any validation errors.', 'Select View retained source to inspect the exact uploaded document.'] },
      { title: 'Publish a release', steps: ['Review the destination suite, digest, and discovered definitions.', 'Select Activate this release for future runs only when the new release should become active.', 'Select Publish source release. Read the immutable release identifier and confirmation.', 'Use Test runs to start checks explicitly. Tests do not run during publication.'] },
      { title: 'Recover from a conflict', steps: ['Keep the source document and inspect the current catalog and suite settings.', 'Create a new source preview. Review every change again before publication.', 'If the publication response is interrupted, inspect Upload history. A repeated publication of the same upload returns its original release.'] }
    ]
  },
  providers: {
    id: 'providers', title: 'Identity providers',
    introduction: ['Configure installation OpenID Connect providers. Sign-in establishes a user identity. Project roles and group assignments remain separate.'],
    concepts: [
      { title: 'Stored configuration', description: 'Providers have permanent identifiers and exact issuer URLs. The callback URL follows the installation origin. Disabled providers do not start sign-in.' },
      { title: 'Credentials stay on the server', description: 'Select an operator-approved credential reference. Public clients can use no client secret. The UI accepts no credential values.' },
      { title: 'Authentication changes', description: 'Every save invalidates earlier provider transactions, sessions, and memberships. Fresh validated sign-in is required. The last usable administrator remains protected.' },
      { title: 'Bounded directory', description: 'More providers continues a bounded scan. Exact provider read denies filter rows. A filtered page can be empty while more readable records remain.' }
    ],
    tutorials: [
      { title: 'Provider logout', steps: ['Open a saved provider and copy its Post-logout callback URL into the provider’s registered logout redirects.', 'Edit Provider logout endpoint to match the discovered end-session endpoint. Leave it empty for TestStudio-only sign-out.', 'Use a provider that accepts client-identifier logout without an ID-token hint. Enter a reason and save the current revision.', 'Verify sign-out locally before enabling the integration in a hosted installation.'] },
      { title: 'Register a provider', steps: ['Select + Register provider.', 'Enter Provider name, Provider identifier, Exact issuer URL, and Client identifier.', 'Select Credential reference and Requested scopes. Keep Provider disabled while preparing group mappings.', 'Choose whether first sign-in creates pending profiles. Enter Provider change reason and select Register provider.', 'Open the saved provider to copy its Callback URL and create provider groups.'] },
      { title: 'Review configuration changes', steps: ['Open a provider from the directory and select Edit provider.', 'Review the impact of invalidating earlier authentication. Enter a reason and select Save provider.', 'A conflict preserves your fields. Select Reload stored provider to review the current revision before another save.'] }
    ]
  },
  provider: {
    id: 'provider', title: 'Identity provider',
    introduction: ['Inspect one provider, manage its exact configuration, and prepare groups populated from validated claims.'],
    concepts: [
      { title: 'Exact identity', description: 'Issuer and subject identify external accounts. Changing the issuer does not transfer existing accounts or their roles. Matching email addresses never link accounts.' },
      { title: 'Provider-owned groups', description: 'Create an empty group with this provider as its permanent membership owner. Groups can belong to the installation or an existing project. Ordinary membership editors remain read-only.' },
      { title: 'Claim mappings', description: 'Map exact external group values to existing groups owned by this provider. Managing a mapping requires authority over its destination group and membership. An unmapped value grants no access.' },
      { title: 'Maximum membership age', description: 'Membership lifetime is at most 3600 seconds and never exceeds token expiry. Zero uses 1800 seconds. Incomplete group claims withdraw provider access.' },
      { title: 'Current revision', description: 'Refresh provider updates the stored view without replacing an opened editor revision. Every save requires a reason and current revision. Failed authorization clears private editors.' }
    ],
    tutorials: [
      { title: 'Connect external group access', steps: ['Use Create a provider group. Enter New group identifier, New group name, and an optional Group project identifier.', 'Select Create provider group. The new group has no roles or memberships.', 'Select Edit provider. Enter Group claim name and select + Add group mapping.', 'Enter the exact External group value and Destination group identifier. Set Membership lifetime in seconds.', 'Clear Provider disabled when ready, enter Provider change reason, and select Save provider.', 'Assign appropriate roles to the group through Project access or Installation access. Verify with a fresh provider sign-in.'] },
      { title: 'Disable or restore sign-in', steps: ['Select Edit provider. Set or clear Provider disabled.', 'Review group mappings and enter Provider change reason. Select Save provider.', 'Disabling withdraws provider sessions and memberships. Restoration requires fresh validated sign-in and does not revive earlier cookies.'] }
    ]
  },
  'effective-permissions': {
    id: 'effective-permissions', title: 'Explain access',
    introduction: ['Check a registered permission for yourself or an authorized user. Inspect the current decision and its direct grants, groups, and inherited role paths.'],
    concepts: [
      { title: 'Exact context', description: 'An empty object checks the displayed project or installation scope. An exact object checks one existing resource. A suite object does not include its tests.' },
      { title: 'Decision and source scan', description: 'The current decision uses the same permission evaluator as live requests. The source list uses bounded scans. An incomplete or empty source page does not change the decision.' },
      { title: 'Deny, expiry, and field restrictions', description: 'Active matching denies override allows. Expired matching rules remain visible but have no effect. Field restrictions follow the evaluator for the requested permission.' },
      { title: 'Assigned and inherited roles', description: 'Each role path identifies the user or group, assigned role, inherited grant owner, and project binding. Disabled roles withdraw their access paths.' },
      { title: 'Private source details', description: 'A restricted source shows its effect and expiry status. Principal, role, scope, expiry timestamp, and field details require source administration read authority.' },
      { title: 'Changing access', description: 'A decision describes access at its check time. A directory change or processed grant expiry invalidates continued scans. Refresh by checking access again.' }
    ],
    tutorials: [
      { title: 'Check your access', steps: ['Open Explain project access or Explain installation access.', 'Leave Inspect user identifier set to your account. Select Permission to check.', 'Leave Exact object empty or enter an existing resource such as suite/health. Select Check access.', 'Read Current decision. Select Scan more sources until Source scan complete appears.'] },
      { title: 'Explain another user', steps: ['Use an account with membership management and permission read authority in the displayed scope.', 'Enter the registered Inspect user identifier. Select the permission and optional exact object.', 'Select Check access. Read the assigned role, inherited grant owner, configured scopes, and expiry.', 'Use my account returns the inspection to your own user.'] },
      { title: 'Respond to changed access', steps: ['A changed input clears the earlier report. Select Check access for the new context.', 'If access changes during a scan, the page clears its report. Select Check access again.', 'If access is denied, inspect the active deny or missing allow. Use the normal role or permission pages for authorized changes.'] }
    ]
  },
  grants: {
    id: 'grants', title: 'Direct permissions',
    introduction: ['Manage direct allow and deny rules for registered users and groups in the displayed scope. These rules apply alongside roles and inherited access.'],
    concepts: [
      { title: 'Exact ownership and objects', description: 'Project grants apply only within that project. Installation grants apply across projects. Empty object means the displayed scope. Exact objects do not include descendant resources.' },
      { title: 'Allow, deny, and expiry', description: 'An active deny overrides matching allows. Expired rules remain visible and grant no access. Removing a deny can restore access from other grants or roles.' },
      { title: 'Permanent identity', description: 'Principal, permission, effect, object, and ownership are permanent. Edit expiry to change the time limit. Remove and create a grant to change its identity. Every change requires a reason.' },
      { title: 'Current authority', description: 'Controls follow current server permissions. Shared groups require installation group authority. Inactive principals permit authorized removal. The last usable administrator remains protected.' },
      { title: 'Bounded management list', description: 'This list contains grants managed through TestStudio. It does not explain all effective access. More grants continues a bounded scan, including after an empty filtered page.' }
    ],
    tutorials: [
      { title: 'Create a grant', steps: ['Read the ownership scope and select + Add direct grant.', 'Choose Principal type and enter the Registered principal identifier.', 'Leave Exact object empty or enter an existing exact resource such as suite/health. Select Refresh grant permissions after changing it.', 'Select Permission and Effect. Enter an optional full UTC expiry timestamp.', 'Enter Grant change reason and select Create grant. Inspect the saved rule.'] },
      { title: 'Change expiry', steps: ['Find the rule and select Edit expiry.', 'Enter Expiry in UTC or leave it empty for no expiry. Enter Grant change reason and select Save grant expiry.', 'A conflict preserves your values. Select Reload stored grant to review the current revision before saving again.'] },
      { title: 'Remove an exact rule', steps: ['Select Remove grant on the intended rule.', 'Read the principal, effect, permission, object, and impact. Enter a Reason and select Confirm action.', 'After a conflict, cancel, refresh permissions, and open a new action. Other grants and roles can still provide access.'] }
    ]
  },
  roles: {
    id: 'roles', title: 'Roles and permissions',
    introduction: ['Inspect predefined roles and manage custom policies in the displayed installation or project scope.'],
    concepts: [
      { title: 'Permanent ownership', description: 'A custom role belongs to one scope. Its identifier and ownership cannot change. Installation policies can grant access across projects. Project policies use one exact project.' },
      { title: 'Explicit policy', description: 'Rules allow or deny registered actions, optionally for an exact object and expiry. Denies take precedence. Object rules do not grant descendant access.' },
      { title: 'Predefined and custom roles', description: 'Predefined definitions are code managed. Installation roles provides installation-wide disabled-state controls. Project roles shows their current state. Custom roles support editable rules and inheritance. Creating a policy does not assign it to anyone.' },
      { title: 'Bounded catalog', description: 'More custom roles continues a bounded scan. A filtered page can contain no readable records and still have another page.' }
    ],
    tutorials: [{ title: 'Create a policy', steps: ['Read the ownership scope and select + Create role.', 'Enter Role name, a permanent Role identifier, and an optional Role description.', 'Use + Add permission rule. Select an action and effect. Leave Exact object empty for the ownership scope, or enter an exact resource such as suite/health.', 'Refresh rule permissions after changing an object. Enter an optional complete UTC expiry timestamp.', 'Use + Inherit role to include existing managed roles from this scope.', 'Enter a Policy change reason and select Save role policy. Open the new role to inspect it.'] }, { title: 'Inspect and assign a role', steps: ['Open a custom role from the catalog. Inspect its rules and inherited policies.', 'Use Project access or Installation access to assign an authorized role to an existing user or group.', 'Read More custom role choices when the required role is outside the first choice page.'] }, { title: 'Change a predefined role state', steps: ['Open Installation roles and find Installation-wide predefined roles.', 'Inspect the role and its code-managed permissions. State changes apply across the entire installation.', 'Select Disable predefined role or Enable predefined role. Read the target and impact.', 'Enter a Reason and select Confirm action. Existing assignments remain retained.', 'After a conflict or unconfirmed response, cancel, select Refresh predefined roles, inspect current state, and open a new action.'] }]
  },
  role: {
    id: 'role', title: 'Role details',
    introduction: ['Review this stored policy and its permanent scope. Authorized editors can change its rules or enable and disable effective access.'],
    concepts: [
      { title: 'Inheritance and authority', description: 'Inherited roles must belong to this scope. The service rejects cycles and checks affected parent roles. Custom delegation requires each rule action and delegation authority.' },
      { title: 'Disabled policy', description: 'Disabling withdraws direct and inherited access while retaining the definition and assignments. Enabling revalidates and restores the stored policy. The last usable administrator remains protected.' },
      { title: 'Access preview', description: 'Preview compares current access with the proposed policy for one registered user and optional exact object. Enabled roles are assigned only for the preview. Existing groups, roles, and denies still apply. No changes are saved.' },
      { title: 'Revision conflicts', description: 'The editor keeps the revision it opened. Refresh role updates the stored view without replacing unsaved edits. Reload stored role explicitly replaces the editor contents after review.' }
    ],
    tutorials: [{ title: 'Edit and preview', steps: ['Select Edit role policy. Adjust the name, description, rules, or inherited roles.', 'Enter Preview user identifier and an optional Preview exact object. Select Preview access.', 'Compare Current access and Proposed access. Changed editor fields make the preview out of date.', 'Enter a Policy change reason and select Save role policy. If a conflict appears, review your entries before selecting Reload stored role.'] }, { title: 'Disable or restore access', steps: ['Close the editor after saving or discarding its changes.', 'Select Disable role or Enable role. Read the target and its effect on assignments and parent roles.', 'Enter a reason and select Confirm action.', 'If the action conflicts, cancel, refresh the role, inspect its current state, and open a new action.'] }]
  },
  'project-access': {
    id: 'project-access', title: 'Project access',
    introduction: ['Manage this project’s roles for existing users and groups. Prepare pending access for users who have not signed in. No invitation is required.'],
    concepts: [
      { title: 'Exact scope', description: 'The project route supplies the exact project binding. Project administrators can delegate permitted predefined project roles. Installation administration does not appear in these project choices.' },
      { title: 'Pending project access', description: 'Prepare one allowed provider, verified email, and project role. Only provider-confirmed email at sign-in activates access. Configuration, inherited roles, and owner authority are checked again. Pending access never merges accounts or grants installation administration.' },
      { title: 'Delegation and group ownership', description: 'Role choices come from the server. Shared installation groups require installation group management authority. A visible assignment can remain read-only.' },
      { title: 'Retained access', description: 'Removing one assignment can leave access through other roles or groups. Inactive principals and disabled roles remain visible for review. The last usable installation administrator remains protected.' },
      { title: 'Conflicts and bounded pages', description: 'Removal uses the revision displayed when the action opens. Cancel stale actions, refresh access, and review the current assignment before reopening. More assignments can continue a bounded scan even when its first page is empty.' }
    ],
    tutorials: [
      { title: 'Add a role', steps: ['Inspect Role choices and expand View permissions for the intended role.', 'Select + Add role. Choose User or Group and enter the registered principal identifier.', 'Select the role and enter an Assignment reason. Review the displayed scope.', 'Select Assign role. The new permissions apply without another sign-in. This action creates no credentials or identity links.'] },
      { title: 'Prepare access before sign-in', steps: ['Select + Prepare project access. Choose Allowed sign-in provider and inspect its exact issuer.', 'Enter Required verified email. The mailbox before @ must match exactly; domain letters ignore case.', 'Choose Pending project role and enter Pending access reason. Select Prepare access.', 'After provider sign-in, select Refresh pending access. Read activated user details or the reason an approval is blocked.', 'Cancel stale or incorrect approvals with a reason, then prepare a new approval after review. Activated roles use ordinary assignment removal.'] },
      { title: 'Remove and review access', steps: ['Find the exact principal and role in Role assignments. Use More assignments to continue a bounded scan.', 'Select Remove role on the intended row and review the target and scope.', 'Enter a reason and select Confirm action. Cancel the action if its target is incorrect.', 'If a conflict appears, cancel, use Refresh access, inspect the current assignment, and open a new action. Other role and group access can still apply.'] }
    ]
  },
  'installation-access': {
    id: 'installation-access', title: 'Installation access',
    introduction: ['Review role assignments and manage access for registered users and groups in this installation. Local and OIDC identities use the same internal user identifiers.'],
    concepts: [
      { title: 'Exact scope', description: 'This page manages installation administration. Use Project access inside a workspace to assign project roles. Bootstrap and operator-managed assignments remain visible before their first API-managed change.' },
      { title: 'Delegation and group ownership', description: 'Role choices come from the server. Shared installation groups require installation group management authority. A visible assignment can remain read-only.' },
      { title: 'Retained access', description: 'Removing one assignment can leave access through other roles or groups. Inactive principals and disabled roles remain visible for review. The last usable installation administrator remains protected.' },
      { title: 'Conflicts and bounded pages', description: 'Removal uses the revision displayed when the action opens. Cancel stale actions, refresh access, and review the current assignment before reopening. More assignments can continue a bounded scan even when its first page is empty.' }
    ],
    tutorials: [
      { title: 'Add a role', steps: ['Inspect Role choices and expand View permissions for the intended role.', 'Select + Add role. Choose User or Group and enter the registered principal identifier.', 'Select the role and enter an Assignment reason. Review the displayed scope.', 'Select Assign role. The new permissions apply without another sign-in. This action creates no credentials or identity links.'] },
      { title: 'Remove and review access', steps: ['Find the exact principal and role in Role assignments. Use More assignments to continue a bounded scan.', 'Select Remove role on the intended row and review the target and scope.', 'Enter a reason and select Confirm action. Cancel the action if its target is incorrect.', 'If a conflict appears, cancel, use Refresh access, inspect the current assignment, and open a new action. Other role and group access can still apply.'] }
    ]
  },
  'password-required': {
    id: 'password-required', title: 'Required password change',
    introduction: ['Your administrator requires a new password before this password session can open the workspace.'],
    concepts: [
      { title: 'Restricted session', description: 'You can inspect this sign-in state, change the password, or sign out. Project and administration pages remain unavailable until the change succeeds.' },
      { title: 'Password policy', description: 'Use 15 through 256 characters and at most 1024 bytes. Spaces are significant. The server also rejects common values.' },
      { title: 'Session rotation', description: 'A successful change invalidates earlier sessions and gives this browser a new session. Password fields clear after a request. An uncertain result requires a session refresh before another attempt.' }
    ],
    tutorials: [{ title: 'Complete the required change', steps: ['Enter the temporary password in Current password.', 'Enter a different New password and repeat it in Confirm new password.', 'Select Change password. The requested page opens after the new session loads.', 'If session refresh fails, select Refresh session. Do not repeat a confirmed password change.'] }]
  },
  account: {
    id: 'account', title: 'Your account',
    introduction: ['Your account applies across TestStudio. Project access is assigned separately. Open your account from the global header on any page. Inspect your profile, identity links, and password state here.'],
    concepts: [
      { title: 'System-wide account', description: 'Your account belongs to this TestStudio installation. It is independent of the active project or application. Project memberships and permissions control access to individual projects.' },
      { title: 'Your identifier', description: 'Local password sign-in uses your permanent internal user identifier. Email matching never links accounts.' },
      { title: 'Independent sign-in methods', description: 'Local credentials and configured OIDC identities can share the directory. Disabled or unconfigured local credentials require installation administration. Link another sign-in requires fresh proof of both identities and your explicit confirmation. Existing accounts never merge.' },
      { title: 'Identity linking', description: 'A link request belongs to this browser session and expires after five minutes. Review the exact issuer and subject. Confirmation invalidates earlier sessions and preserves your permissions.' },
      { title: 'Password changes', description: 'A password change requires the current password. Successful changes invalidate earlier sessions and rotate this browser’s session. Password values are never saved in browser storage.' }
    ],
    tutorials: [{ title: 'Link another sign-in', steps: ['Choose New identity provider and enter Identity link reason.', 'Verify your current password or confirm your local operator account. OIDC sign-in requires a fresh current-account proof.', 'Select Verify new identity and authenticate with the target provider.', 'Review the exact issuer and subject, then select Confirm identity link. Use Cancel identity link to discard the proof.'] }, { title: 'Change your local password', steps: ['Select your name in the global header. On narrow screens, select Your account in that header.', 'Read Your local sign-in. If an enabled credential exists, enter Current password, New password, and Confirm new password.', 'Select Change password. Review the result and refreshed account state.', 'Use Refresh session if the request outcome or refreshed session is unavailable.'] }, { title: 'Inspect your external identities', steps: ['Read Your linked identities for exact issuer and subject pairs.', 'Use More linked identities to continue a bounded page.', 'Use Refresh account to load current profile and credential state.'] }]
  },
  users: {
    id: 'users', title: 'Users',
    introduction: ['Manage installation user profiles and open account details. Project permissions alone do not grant access to this directory.'],
    concepts: [
      { title: 'Pending profiles', description: 'Registration adds a profile with a permanent internal identifier. It sends no invitation, links no external identity, and grants no project access.' },
      { title: 'Email and identity', description: 'Email is profile information. Matching email addresses never merge accounts. An external identity uses its exact issuer and subject pair.' },
      { title: 'Directory pages', description: 'Pages scan a bounded portion of the directory and filter unreadable records. An empty page can still have a More users button.' }
    ],
    tutorials: [
      { title: 'Register a profile', steps: ['Select + Register user.', 'Enter a display name, permanent identifier, and optional email address.', 'Select Save user profile. The new record has no identity link or project access.', 'Open the user name to inspect account details.'] },
      { title: 'Find and inspect an account', steps: ['Read account state, email verification, and last sign-in in User directory.', 'Use More users to continue a bounded scan.', 'Open a user name to inspect linked identities, group membership, and active sessions.', 'Use Refresh users to return to a fresh first page.'] }
    ]
  },
  user: {
    id: 'user', title: 'User details',
    introduction: ['Inspect a user profile, linked identities, group membership, and active sessions. Installation user permissions control these views and actions.'],
    concepts: [
      { title: 'Profile and sign-in', description: 'An enabled profile still needs a configured sign-in identity. Editing an email clears verification and does not change identity ownership.' },
      { title: 'Account disabling', description: 'Disabling blocks sign-in and invalidates earlier sessions. Restoring the account does not revive them. The last usable installation administrator remains protected.' },
      { title: 'Local credentials', description: 'Installation administrators can set or reset a local password with a reason. The account must change it at password sign-in. Resets preserve disabled state and invalidate earlier sessions. Refresh local credential before reopening a conflicted editor.' },
      { title: 'Session revocation', description: 'Revocation removes one session without ending other sessions. Revoking Current browser signs you out. Session credentials never appear in this view.' },
      { title: 'Revision conflicts', description: 'Edits and actions use the revision you opened. A conflict keeps your edits. Reload a profile explicitly; cancel and reopen stale lifecycle actions after refreshing.' }
    ],
    tutorials: [
      { title: 'Edit or disable a user', steps: ['Select Edit profile, change the permitted fields, and select Save user profile.', 'If a revision conflict appears, review your unsaved values before selecting Reload stored profile.', 'To disable access, select Disable user, enter a reason, and select Confirm action.', 'Use Restore user with a reason when the account can sign in again. Earlier sessions remain invalid.'] },
      { title: 'Inspect access and revoke a session', steps: ['Read Linked identities for retained issuer and subject pairs.', 'Open a Group memberships link to inspect permitted local or external membership.', 'Read Active sessions and identify the provider, last activity, and expiry limits.', 'Select Revoke session on the intended row, enter a reason, and confirm.', 'Refresh a collection for current status or use its More control to continue a bounded scan.'] }
    ]
  },
  groups: {
    id: 'groups', title: 'Groups',
    introduction: ['Manage installation groups or groups owned by the selected project. Each directory shows one permanent ownership scope.'],
    concepts: [
      { title: 'Ownership scope', description: 'Installation groups and project groups use separate directories. Project administrators cannot reparent groups or change shared installation groups.' },
      { title: 'Membership source', description: 'Local groups permit authorized local membership edits. Externally managed membership is read-only and must change through its source provider.' },
      { title: 'Membership and access', description: 'Membership alone grants no project permissions. Role assignments and explicit grants determine access. Archive removes effective membership access while retaining records.' }
    ],
    tutorials: [
      { title: 'Create a local group', steps: ['Choose Installation groups, or select a workspace and open Project groups.', 'Select + New group. Read the displayed ownership scope.', 'Enter a name, permanent identifier, and optional description.', 'Select Save group. Ownership cannot change after creation.', 'Open the group to manage permitted local membership.'] },
      { title: 'Find an archived or external group', steps: ['Read Membership source to distinguish local and external groups.', 'Select Include archived groups to find retained archived records.', 'Use More groups to continue a bounded scan.', 'Open a group for its state, source, membership, and available lifecycle actions.'] }
    ]
  },
  group: {
    id: 'group', title: 'Group details',
    introduction: ['Inspect group ownership and membership. Manage permitted local members or archive and restore the group without removing its history.'],
    concepts: [
      { title: 'Local and external members', description: 'Local membership records can have an expiry. External source records remain read-only. Membership does not imply a project role.' },
      { title: 'UTC expiry', description: 'Enter membership expiry in UTC, independent of your computer’s timezone. Empty expiry means no expiry. Expired or inactive membership does not grant effective access.' },
      { title: 'Archive and restore', description: 'Archive removes effective membership access while retaining membership and assignment records. Restore can reactivate unexpired membership and associated role access.' },
      { title: 'Conflicts and reasons', description: 'Edits retain their original revision and keep unsaved values on a conflict. Lifecycle actions and member removal require a reason. The last usable administrator remains protected.' }
    ],
    tutorials: [
      { title: 'Add or update a local member', steps: ['Select + Add member and enter an existing internal user identifier.', 'Optionally enter a future UTC expiry, then select Save local membership.', 'Use Edit membership on a local row to change or clear expiry.', 'After a conflict, review your edits before selecting Reload stored membership.', 'To remove a local member, select Remove member, enter a reason, and confirm.'] },
      { title: 'Archive or restore a group', steps: ['Review ownership, membership source, and the retained members.', 'Select Archive group, enter a reason, and select Confirm action.', 'Archived members remain visible but inactive. Local membership edits require restoration.', 'To restore, review the retained membership and assigned access before selecting Restore group.', 'Enter a reason and confirm. Retained, unexpired membership becomes active again.'] }
    ]
  },
  notifications: {
    id: 'notifications', title: 'Notifications',
    introduction: ['Send completion summaries to a generic or Discord webhook. Inspect delivery history without changing test outcomes.'],
    concepts: [
      { title: 'Registered secrets', description: 'Choose a project secret reference. The installation operator registers destinations on the server. The browser receives names, never webhook URLs or credentials.' },
      { title: 'Frozen selection', description: 'Run creation freezes enabled channel definitions and scope filters. Edits affect future runs. Empty scope filters include all values. Timeouts remain separate from failures.' },
      { title: 'Delivery and uncertainty', description: 'Transient failures retry within the frozen attempt budget. Delivery retries preserve identity and never rerun tests. An interrupted checkpoint can leave remote acceptance unknown, even after later confirmation.' },
      { title: 'Pause and archive', description: 'Pause blocks later attempts. Reenabling does not revive old messages automatically. Explicit retry requires the original destination and an enabled channel. Archive keeps history; restore leaves the channel paused.' }
    ],
    tutorials: [
      { title: 'Configure a completion channel', steps: ['Open Automation, choose Notifications, then select New channel.', 'Name the channel and choose its transport and registered destination. The identifier is generated under Advanced identity options.', 'Select Continue. Choose run triggers and outcomes. Advanced filters and delivery options contains scope filters and retries.', 'Select Continue. Review the destination and scope. Choose Enable this channel when ready, then select Save notification channel.', 'Queue a new matching run. Earlier runs do not gain new notifications.'] },
      { title: 'Investigate delivery', steps: ['Read Notification deliveries on this page or one run’s details.', 'Expand Frozen run summary to inspect the captured result and destination reference.', 'Select Attempts for retained HTTP statuses and checkpoint uncertainty.', 'For stopped or waiting deliveries, correct the channel or receiver before selecting Retry delivery.', 'Use Stop retries to cancel pending work. A message already sent cannot be recalled.'] }
    ]
  },
  schedules: {
    id: 'schedules', title: 'Schedules',
    introduction: ['Repeat published tests on an interval or cron timetable. Every triggered firing creates a normal run with frozen definitions, timing criteria, results, and evidence.'],
    concepts: [
      { title: 'Cadence and timezone', description: 'Intervals use UTC and accept five seconds through thirty days. Cron uses five fields and an explicit IANA timezone. Preview shows the next five occurrences in UTC. A repeated daylight-saving hour can contain two occurrences.' },
      { title: 'Release and selection', description: 'Leave the release pin empty to follow the suite’s active release. A pin keeps a specific published release. Empty test identifiers select all eligible tests and normal dependencies.' },
      { title: 'Execution owner', description: 'The creator or latest configuration editor becomes the execution owner. Every firing checks current execution permissions, target protection, catalog state, and worker availability.' },
      { title: 'Overlap and missed ticks', description: 'Skip prevents overlap with the previous run from this schedule. Allow permits another run. After downtime, one firing represents missed occurrences and the next time moves into the future.' },
      { title: 'Firing history', description: 'Triggered firings link to runs. Skipped and blocked firings retain their reasons. Archive keeps history. Restore leaves a schedule paused. Runtime firings do not invalidate configuration edits.' }
    ],
    tutorials: [
      { title: 'Create a repeating check', steps: ['Open Automation, choose Schedules, then select New schedule.', 'Name the schedule and choose its suite and target. Advanced selection options contains identifiers, release pins, and execution location.', 'Select Continue. Choose the cadence. Advanced run options contains overlap, run limits, and retries.', 'Select Continue to review the target and timing. Preview schedule shows future times without running tests.', 'Choose Enable this schedule only when it should repeat, then select Save schedule.'] },
      { title: 'Investigate or pause a schedule', steps: ['Read the next occurrence and last firing on Project schedules.', 'Select History to inspect triggered, skipped, and blocked occurrences.', 'Follow a run link for attempts, timing, logs, and retained evidence.', 'Use Pause to stop future firings. Existing runs use their normal cancellation controls.', 'Use Edit to correct the selection or cadence. Configuration conflicts require an explicit reload.', 'Use Archive to remove normal scheduling while keeping its identifier and history. Include archived schedules to restore it.'] }
    ]
  },
  trends: {
    id: 'trends', title: 'Trends',
    introduction: ['Explore final logical results across finished runs. Stacked areas show passed, failed, timed-out, flaky, skipped, and unknown results together.'],
    concepts: [
      { title: 'Active application', description: 'The sidebar Application switcher scopes Suites, Test catalog, Test runs, and Trends. The context above the page shows the active application. Choose All applications for project-wide work. Project overviews contains All applications and HTTPS probes. Switching applications keeps the main destination and clears the previous filters.' },
      { title: 'History scope', description: 'Dates use UTC and select runs by creation time. Filter by suite, environment, category, permanent test identifier, or execution location. Test identifiers require a suite.' },
      { title: 'Logical counts and gaps', description: 'Retries remain separate attempts and do not inflate counts. Unknown results remain unknown. Striped gaps mark runs that did not select a matching test or have no usable selection snapshot.' },
      { title: 'History pages', description: 'Each page scans a bounded portion of project history. Counts and rankings cover only that page. Use Older history to continue or Newest history to return.' },
      { title: 'Failure and volatility rankings', description: 'Failure rankings combine failures and timeouts while displaying their separate counts. Volatility measures state changes between adjacent known results for the same target and location. Gaps, skipped results, and unknowns break comparisons.' }
    ],
    tutorials: [
      { title: 'Explore a category or test', steps: ['Set From date and Through date, then enter any suite or environment identifiers.', 'Enter a Category for category history. For one test, enter its suite and Permanent test identifier.', 'Select Apply scope. The URL preserves the chosen filters.', 'Focus or hover a run in the stacked chart to inspect its values. Select it to open run details.', 'Open View chart values for the accessible table and direct run links.'] },
      { title: 'Investigate recurring problems', steps: ['Review Most failed tests for repeated failures and timeouts.', 'Follow a test link to its history, or a category link to its category view.', 'Review Most volatile tests and its changes and comparisons before interpreting the percentage.', 'Check history-page notices and sample counts. Browse older history when the page does not cover the complete period.'] }
    ]
  },
  signin: {
    id: 'signin', title: 'Sign-in',
    introduction: ['Sign in to open your TestStudio workspace. Your installation controls which sign-in options appear.'],
    concepts: [
      { title: 'Your workspace', description: 'Projects organize applications, suites, test definitions, and run history. Your permissions control which actions are available.' },
      { title: 'Provider sign-out', description: 'Sign out closes the current TestStudio session first. A configured provider can offer a separate Sign out of provider action. Confirm it at the provider. Failure or cancellation leaves the TestStudio session closed.' },
      { title: 'Local sign-in', description: 'Sign in with a password uses a managed local account and its internal user identifier. Open local workspace is a separate development administrator option that appears only when explicitly enabled.' },
      { title: 'Session', description: 'If your session expires, sign in again. The page address still identifies the work you were viewing.' }
    ],
    tutorials: [
      { title: 'Open your workspace', steps: ['Choose your organization’s sign-in provider, or select Open local workspace on a local installation.', 'Complete sign-in. If a password change is required, enter the current password and a new password before the requested page opens.', 'Use the Project selector to choose a project.'] },
      { title: 'If no sign-in option appears', steps: ['Ask your installation administrator to configure a sign-in provider.', 'Reload this page after the administrator confirms the setup.'] }
    ]
  },
  projects: {
    id: 'projects', title: 'Projects',
    introduction: ['Projects keep related applications, test suites, and results together. Open a project to manage its catalog and runs.'],
    concepts: [
      { title: 'Project', description: 'A project provides the shared home for a test catalog, configuration, and run history.' },
      { title: 'Permanent identifier', description: 'Identifiers stay stable when display names change. Use letters, numbers, periods, underscores, or hyphens.' },
      { title: 'Permissions', description: 'The projects and controls you see depend on your access. Creating a project requires project creation permission. Ordinary creators receive project-scoped administration, editing, publication, and run roles together. Installation privileges do not change.' }
    ],
    tutorials: [
      { title: 'Create your first project', steps: ['Enter a project name. The permanent identifier is generated; change it under Advanced details when needed.', 'Select Continue. Name your first application and choose whether to create an empty Availability suite.', 'Select Continue. Optionally register a target URL, stage, and protection.', 'Review the selected resources, then select Create project. No tests run during setup.', 'If setup stops after creating resources, Retry remaining setup continues without creating those resources again.'] },
      { title: 'Return to existing work', steps: ['Open a project card to choose an application.', 'Use All applications for the project overview. Choose an application to scope daily testing. Use Test runs for execution history or Project settings for other configuration.', 'Use the Project selector to switch projects.'] }
    ]
  },
  catalog: {
    id: 'catalog', title: 'Test catalog',
    introduction: ['The catalog describes what each test checks and how TestStudio runs it. Draft edits become executable through a published suite release.'],
    concepts: [
      { title: 'Active application', description: 'The sidebar Application switcher scopes Suites, Test catalog, Test runs, and Trends. The context above the page shows the active application. Choose All applications for project-wide work. Project overviews contains All applications and HTTPS probes. Switching applications keeps the main destination and clears the previous filters.' },
      { title: 'Test identity and lifecycle', description: 'A permanent identifier connects a test to its history. Enabled tests can be selected. Disabled and archived tests retain their definitions.' },
      { title: 'Draft and release', description: 'Open a test title for retained run history. Revisions under More shows saved definition revisions. Editing changes the draft. Validate and preview compares it with the active release, including safety, policy, and readable schedule effects. Publish release freezes it without activation. Publish and activate changes the active release for future unpinned runs. Stale reviews require a new preview. Validation does not execute tests or prove they pass.' },
      { title: 'Safety', description: 'Each test has a safety label and rationale. Protected targets require safe definitions and the required permissions.' },
      { title: 'HTTP assertions', description: 'Native checks support GET, HEAD, accepted statuses, exact response headers, body text, JSON fields, response-time limits, and certificate lifetime. Redirects stay disabled. Expected headers describe responses, not authentication credentials.' },
      { title: 'Timing policy', description: 'Comparable clean runs build a duration baseline. Timing rules can warn or fail on slow execution and recommend bounded timeouts.' },
      { title: 'Browse the catalog', description: 'Table shows a flat list. By category groups slash-separated paths with filtered counts. Parent categories include their children. Advanced filters provides lifecycle, exact category, tag, owner, runner, safety, and source controls. Sort tests changes display order. The server supplies matching counts and category branches across all pages. Load more tests adds the next page. Filters and paging preserve selected tests; the bulk bar reports hidden selections. If the catalog or access changes between pages, Reload results requires a fresh selection.' },
      { title: 'Catalog transfer', description: 'Import validates up to 30 native HTTP definitions before changes are applied. Export downloads the full readable suite, including tests on unloaded pages. Neither action publishes or executes tests. A stale import requires Reload catalog and a fresh preview; its document remains available.' },
      { title: 'Saved selections', description: 'A suite stores explicit test identifiers or a versioned filter. Filters match published metadata and can include child categories. A resolved preview includes dependencies and freezes one release. Empty matches, invalid dependencies, or current lifecycle restrictions block execution explicitly.' }
    ],
    tutorials: [
      { title: 'Find and manage tests', steps: ['Choose the Application in the sidebar. Search by title, identifier, category, or tag, then narrow the suite or test type.', 'Choose By category. Expand a branch and select View all in a parent, or select a leaf category.', 'Use Advanced filters for exact category matches and less common metadata. Clear category or Clear filters when no tests match.', 'Choose Sort tests and use Load more tests to inspect another page. Table and category views share the same selection. Review selected tests hidden by filters. If a cursor becomes outdated, Reload results and select tests again.', 'Use Edit, Enable, Disable, or Restore on a row. More provides History, Duplicate, and Archive. Lifecycle changes keep existing releases and run history.', 'Select Validate and preview. Read additions, changes, removals, safety details, and schedules before publishing. Choose Publish release or Publish and activate. If a response is lost, retry the same action on this page; its request key recovers an existing publication.'] },
      { title: 'Create and publish an HTTPS check', steps: ['Choose Suites and create a suite if none exists.', 'Select + Create test, then choose HTTPS probe. Enter a title, choose the suite, and select Continue.', 'Paste a complete website URL. HTTPS uses port 443 automatically. Alternatively choose Use a registered environment for an endpoint alias and relative path.', 'Add optional text, response header, JSON field, response-time, and certificate lifetime assertions. HEAD cannot assert response bodies.', 'Continue to Review. Confirm any new target configuration, the safety label, and rationale, then select Save draft. Saving a draft does not contact the website.', 'Choose the suite under Publish a release. Select Validate and preview, review the result, then Publish and activate.'] },
      { title: 'Find and maintain a test', steps: ['Search by title, identifier, category, or tag. Use suite and lifecycle filters to narrow the list.', 'Use History to inspect retained definitions, or Edit to change the draft.', 'Use Archive to remove a test from normal selection. Choose Archived in the lifecycle filter to find and restore it.', 'Publish a new release when draft changes should apply to future runs.'] },
      { title: 'Save and reuse a selection', steps: ['Select tests from one suite and use Use selected tests under Saved selections, or choose Suite for saved selections and New saved selection.', 'Enter a name and permanent identifier. Choose explicit identifiers or a versioned filter. Select Save selection.', 'Select Preview saved selection and inspect the resolved tests, dependencies, and published release.', 'Open Test runs. Choose the suite and target. Under Advanced selection and run options, choose and preview the saved selection. Select Review run, then Start run.', 'Use Inspect to edit a saved selection. After a conflict, reload before saving. Archive preserves existing run snapshots; restore makes the selection available again.'] },
      { title: 'Edit metadata across tests', steps: ['Filter by suite and select between 1 and 30 tests. Choose Edit metadata under Bulk action.', 'Select Change for each field to replace, then enter its shared value. Unselected fields remain unchanged.', 'Separate tags with commas. Select an empty Owner, Tags, Description, or Operating conditions value to clear that field.', 'Select Safety rationale and enter its explanation when changing a safety label. Current publishing permission is required for safety changes.', 'Select Preview bulk changes. Review the displayed before and after values, then Apply reviewed changes.', 'After a conflict, reload the catalog and prepare a new preview. Publish separately when the new metadata should enter a release.'] }
    ]
  },
  applications: {
    id: 'applications', title: 'Applications',
    introduction: ['Applications identify the systems tested by this project. Open an application to see its environments and linked test suites.'],
    concepts: [
      { title: 'Active application', description: 'The sidebar Application switcher scopes Suites, Test catalog, Test runs, and Trends. The context above the page shows the active application. Choose All applications for project-wide work. Project overviews contains All applications and HTTPS probes. Switching applications keeps the main destination and clears the previous filters.' },
      { title: 'Ownership and links', description: 'An environment belongs to one application. Suites belong to the project and can link to several applications. A suite without an application link remains available in the project catalog.' },
      { title: 'API authentication', description: 'When registering an environment, expand Advanced credentials to map a test binding to an approved reference. The endpoint origin must match the operator approval. HTTPS is required outside local environments. Secret values stay on the execution host and gateway.' },
      { title: 'Environment targets', description: 'The environment list shows the current application only. Stage and protection describe its target. Endpoint names remain part of environment registration.' },
      { title: 'Linked suites', description: 'View tests opens the catalog filtered to a suite. Create test carries that suite into the test-type chooser. Archived suites retain their visible history and do not offer new test creation here.' },
      { title: 'Current access', description: 'Application visibility requires project read permission. Read permissions control the environment and suite lists. Management permissions control registration buttons. Missing read permission does not imply that the application has no resources.' }
    ],
    tutorials: [
      { title: 'Find an application', steps: ['Choose a project, then select All applications under Project overviews. Open an application or use the Application switcher.', 'Select Open application on its card.', 'Review Environments and Linked suites. Select View tests to inspect one suite in the catalog.'] },
      { title: 'Register related resources', steps: ['Select Add application on the application list. Enter its name and select Continue. Identifiers are generated under Advanced details.', 'Review Options, continue to Review, then select Create.', 'On its detail page, select Add environment or Add suite. The application is already selected.', 'Follow Name, Options, and Review. Use Back to change entries before selecting Create. Unsaved registration changes require confirmation before leaving.'] }
    ]
  },
  suites: {
    id: 'suites', title: 'Suites',
    introduction: ['Suites group related tests and their published versions. Choose View tests to open the catalog for one suite.'],
    concepts: [
      { title: 'Active application', description: 'The sidebar Application switcher scopes Suites, Test catalog, Test runs, and Trends. The context above the page shows the active application. Choose All applications for project-wide work. Project overviews contains All applications and HTTPS probes. Switching applications keeps the main destination and clears the previous filters.' },
      { title: 'Applications and suites', description: 'Suites belong to a project and can link to several applications. Shared suite badges identify these links. Unassigned tests inherit suite links. Shared suites keep one draft and release, including imports, exports, and saved selections.' },
      { title: 'Drafts and releases', description: 'Draft tests can change. A published release freezes definitions for runs. Ready to run means an active release exists.' },
      { title: 'Advanced settings', description: 'Suite settings provides application associations, target requirements, and archive or restore controls. These use the same settings as Project settings.' }
    ],
    tutorials: [
      { title: 'Create a suite', steps: ['Select New suite. Enter its name and select Continue.', 'Choose an application and whether tests need a target environment.', 'Select Continue, review the settings, then select Create.', 'Select Create test on the suite card. The chooser keeps this suite selected.'] },
      { title: 'Find and manage tests', steps: ['Select View tests on a suite card.', 'Use the filtered catalog to edit or publish its tests.', 'Return to Suites and open Suite settings for target requirements or lifecycle changes.'] }
    ]
  },
  configuration: {
    id: 'configuration', title: 'Project settings',
    introduction: ['Register the applications you test, the suites that group their tests, and the environments those tests can target.'],
    concepts: [
      { title: 'Application', description: 'An application identifies the system being tested. Environments belong to an application.' },
      { title: 'Suite', description: 'A suite groups test definitions and published releases. A suite can require a target environment or run without one.' },
      { title: 'Suite lifecycle', description: 'Inspect suite opens settings, current authority, and readable affected schedules. Archive blocks new dispatch and attempts. Releases, results, evidence, and schedule pins stay retained. Restore keeps the same identity and active release.' },
      { title: 'API credential references', description: 'Advanced credentials maps a binding such as api-login to a current project reference. An installation operator registers secret values and approved origins. Review shows the binding and reference without a secret. Missing references or unapproved origins produce explicit errors. Changed mappings block queued runs until you review a new run.' },
      { title: 'Environment and endpoint', description: 'An environment records a stage and named endpoint URLs. Test definitions refer to endpoint names instead of embedding a target URL.' },
      { title: 'Protection', description: 'Protection is separate from the stage name. A protected environment requires safe tests and protected-target permissions.' }
    ],
    tutorials: [
      { title: 'Prepare a target for HTTPS checks', steps: ['Choose Applications and complete Name, Options, and Review to create an application.', 'Open the application and select Add environment. Enter a name, then select Continue.', 'Enter the endpoint URL and stage. The endpoint name defaults to api under Advanced endpoint options.', 'Choose protection, then select Continue. Review the target and select Create.'] },
      { title: 'Add API authentication', steps: ['Ask an installation operator to register a project API credential reference and its exact allowed origin.', 'Register an environment. On Options, expand Advanced credentials and select Add credential binding.', 'Enter a binding name, choose an approved reference, and review the target before selecting Create.', 'In Create test, choose API test. On Request, select Authentication and enter the same environment binding name.', 'Review and save the source. Prepare and publish it, then start a run with the configured environment.'] },
      { title: 'Register a suite', steps: ['Choose Suites and enter a name. Select Continue.', 'Choose an application when the suite belongs to one.', 'Keep Requires a target environment enabled for endpoint checks. Turn it off for tests that need no target.', 'Select Continue, review the settings, and select Create.'] },
      { title: 'Manage or archive a suite', steps: ['Choose Suites and select Inspect suite on its row.', 'Edit Suite name, Suite description, applications, or target requirements. Select Save suite settings.', 'Review Affected schedules before selecting Archive suite. Current permissions control which schedules appear.', 'Select Restore suite to resume normal selection with its retained release.', 'If a revision conflict appears, select Reload suite settings and review the current values before trying again.'] }
    ]
  },
  runs: {
    id: 'runs', title: 'Test runs',
    introduction: ['Start tests from a published suite and browse earlier runs. Each run captures its definitions, target settings, and timing criteria.'],
    concepts: [
      { title: 'Active application', description: 'The sidebar Application switcher scopes Suites, Test catalog, Test runs, and Trends. The context above the page shows the active application. Choose All applications for project-wide work. Project overviews contains All applications and HTTPS probes. Switching applications keeps the main destination and clears the previous filters.' },
      { title: 'Application history', description: 'New runs retain their application context. Application history and trends count matching tests only. Shared run details show the complete saved run. Old targetless runs without application snapshots remain under All applications.' },
      { title: 'Published suite', description: 'The run form lists suites with active published releases. Draft edits do not change existing runs.' },
      { title: 'Selection and dependencies', description: 'Leave test identifiers empty to select all eligible tests. An explicit selection also includes required dependencies.' },
      { title: 'Current safety', description: 'Every dependency must remain enabled. Protected targets require safe labels in both the release and current catalog, plus protected-execution permission. Concurrent edits revalidate run creation. A changed target policy can block queued work without replacing its saved settings.' },
      { title: 'Saved selection review', description: 'Choose a saved selection and preview its tests before starting. The review lasts 15 minutes and pins the release, selection revision, and resolved identifiers. Later active releases do not replace that reviewed release. Changed selection definitions require a new review.' },
      { title: 'Time limits and retries', description: 'The overall run deadline includes queue time. Each attempt also has a timeout. Retries remain separate attempts within one logical test result.' },
      { title: 'Timing estimate', description: 'Estimates use comparable history. Missing history stays unknown. The timeout budget includes permitted retries; expected execution excludes preparation and capture.' }
    ],
    tutorials: [
      { title: 'Start a run', steps: ['Select + New run, then choose a published suite in Start a run.', 'Choose a target environment when the suite requires one. Use No target only for suites that support it.', 'Open Advanced selection and run options for saved selections, explicit test identifiers, run limits, or retries.', 'Select Estimate timing to review available history and timeout budgets.', 'Select Review run. Check the suite, target, selection, and limits, then select Start run. Open the details to follow progress and inspect evidence.'] },
      { title: 'Find an earlier run', steps: ['Use the history filters to narrow the suite, environment, state, or selection.', 'Open a run to inspect its frozen settings, results, attempts, and evidence.', 'Use the pagination controls to browse more history.'] }
    ]
  },
  run: {
    id: 'run', title: 'Run details',
    introduction: ['This page explains one run: what it used, what happened, how long tests took, and which evidence remains available.'],
    concepts: [
      { title: 'Run state and outcome', description: 'State describes execution progress. Outcome describes the result. An incomplete run retains uncertainty instead of becoming a pass.' },
      { title: 'Blocked dispatch', description: 'Each attempt checks current test lifecycle, safety, permissions, and target policy. A blocked retry keeps its earlier attempt evidence. Read the reason, correct configuration, and start a new run. The original definitions and target remain unchanged.' },
      { title: 'Results and attempts', description: 'Final counts use logical test results. Inspect test scopes attempts, logs, and artifacts to one test. Attempt evidence scope selects one retry or all attempts. The selected test uses readable attempt and artifact cards. Failed recorded steps open automatically. Timeouts stay separate. A successful retry can leave a flaky result.' },
      { title: 'Test timing', description: 'Frozen predictions explain the criteria used for this run. A runtime change blocks dispatch; create a new run to use its matching history. Worker elapsed time includes coordination. Duration unavailable is excluded from history. A measured zero remains valid and uses the absolute slowdown guard when no ratio exists.' },
      { title: 'Evidence and retention', description: 'Logs and artifacts have capture and retention policies. Capture gaps do not erase test outcomes. Select View on a trace to open this installation’s private Playwright viewer on the artifact origin. No public upload is required. Refresh trace access renews its short-lived authorization. Pins preserve evidence; deletion removes files while retaining results.' },
      { title: 'HTTP diagnostics', description: 'Expand HTTP diagnostics in attempt history to inspect network phases, certificate dates, and any project trust-bundle name and fingerprint. Operators bind private certificate authorities to one project and exact HTTPS origin. A changed bundle blocks queued work and starts separate timing history for new runs. Unavailable phases stay unavailable. Certificate lifetime uses the retained attempt finish time.' },
      { title: 'API diagnostics', description: 'API steps, assertion text, logs, and bounded JSON hide sensitive headers, tokens, cookies, and reviewed private fields. API media and binary attachments produce capture gaps without changing test outcomes. Browser tests keep their normal media capture. Earlier evidence without a redaction policy marker has no automatic redaction guarantee.' },
      { title: 'Go diagnostics', description: 'Expand Go diagnostics for the selected package, test, toolchain, platform, build duration, and process exit codes. Build errors and missing selected tests retain unknown attempts and block the run. Recorded subtests remain steps within the registered logical test.' },
      { title: 'Notification deliveries', description: 'Completion messages use channel settings captured when the run was queued. Delivery failures remain separate from test failures. Inspect attempts, retry stopped delivery, or stop waiting retries without rerunning tests.' },
      { title: 'Worker recovery', description: 'After an interruption, recovery retains diagnostics and removes owned resources without rerunning tests. Pending uploads wait for their lease, then verify stored bytes. Missing or corrupt bytes remain upload-failed; temporary storage errors retry. Recovered observations keep the original run outcome.' }
    ],
    tutorials: [
      { title: 'Investigate a failed or incomplete run', steps: ['Read the run summary and any execution or capture warning.', 'Select Inspect test beside a final result. Test evidence scope keeps its attempts, logs, and artifacts together.', 'Use Attempt evidence scope to compare an unsuccessful attempt with a retry. Failed recorded steps open automatically. Advanced log filters and Advanced execution events provide technical detail.', 'Select View beside a retained screenshot or trace. The private trace viewer provides recorded actions and DOM snapshots without contacting the original website. Close preview returns focus to its artifact button.', 'Refresh trace access after expiry. Download trace provides an authorized local fallback. HTTPS or localhost and service-worker support are required. Missing, expired, or upload-failed files keep their visible status.', 'Use Rerun failures to repeat failed, timed-out, and flaky tests in a separate run. Original run links back to the saved parent.', 'After a lost rerun response, retry Rerun failures on this same page. Refresh preserves the error and its request key. After permission loss, use Refresh access. Rejected evidence is hidden while saved records remain stored.'] },
      { title: 'Understand timing and preserve evidence', steps: ['Review Test timing for the predictions and timeout criteria frozen when this run was queued.', 'Compare measured durations, expected values, and slowdown ratios. Functional outcomes remain visible beside timing failures.', 'Use the retention controls to pin evidence that needs further review.', 'If current timing history needs a reset, open Manage comparable timing history and provide a reason. Reset requires permission and does not change historical results.'] }
    ]
  }
}
