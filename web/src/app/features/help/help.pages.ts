import type { HelpPage } from './help.models'

export const helpPages: Record<string, HelpPage> = {
  'project-access': {
    id: 'project-access', title: 'Project access',
    introduction: ['Review role assignments and manage access for registered users and groups in this project. Local and OIDC identities use the same internal user identifiers.'],
    concepts: [
      { title: 'Exact scope', description: 'The project route supplies the exact project binding. Project administrators can delegate permitted predefined project roles. Installation administration does not appear in these project choices.' },
      { title: 'Delegation and group ownership', description: 'Role choices come from the server. Shared installation groups require installation group management authority. A visible assignment can remain read-only.' },
      { title: 'Retained access', description: 'Removing one assignment can leave access through other roles or groups. Inactive principals and disabled roles remain visible for review. The last usable installation administrator remains protected.' },
      { title: 'Conflicts and bounded pages', description: 'Removal uses the revision displayed when the action opens. Cancel stale actions, refresh access, and review the current assignment before reopening. More assignments can continue a bounded scan even when its first page is empty.' }
    ],
    tutorials: [
      { title: 'Add a role', steps: ['Inspect Role choices and expand View permissions for the intended role.', 'Select + Add role. Choose User or Group and enter the registered principal identifier.', 'Select the role and enter an Assignment reason. Review the displayed scope.', 'Select Assign role. The new permissions apply without another sign-in. This action creates no credentials or identity links.'] },
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
      { title: 'Configure a completion channel', steps: ['Select + New channel and enter a name and permanent identifier.', 'Choose a transport and registered secret reference.', 'Optionally filter suites, environments, and stages. Select triggers and outcomes, or include evidence gaps.', 'Set the automatic attempt budget. Select Enable this channel when ready, then Save notification channel.', 'Queue a new matching run. Earlier runs do not gain new notifications.'] },
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
      { title: 'Create a repeating check', steps: ['Select + New schedule and enter a name and permanent identifier.', 'Choose a published suite, its required target, and an execution location.', 'Optionally select test identifiers or pin a release.', 'Choose an interval or cron expression, timezone, overlap rule, run limit, and retries.', 'Select Preview schedule to inspect future times and timing readiness.', 'Select Enable this schedule when it should run, then Save schedule.'] },
      { title: 'Investigate or pause a schedule', steps: ['Read the next occurrence and last firing on Project schedules.', 'Select History to inspect triggered, skipped, and blocked occurrences.', 'Follow a run link for attempts, timing, logs, and retained evidence.', 'Use Pause to stop future firings. Existing runs use their normal cancellation controls.', 'Use Edit to correct the selection or cadence. Configuration conflicts require an explicit reload.', 'Use Archive to remove normal scheduling while keeping its identifier and history. Include archived schedules to restore it.'] }
    ]
  },
  trends: {
    id: 'trends', title: 'Trends',
    introduction: ['Explore final logical results across finished runs. Stacked areas show passed, failed, timed-out, flaky, skipped, and unknown results together.'],
    concepts: [
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
      { title: 'Local access', description: 'Open local workspace appears only when local administrator sign-in is enabled for this installation.' },
      { title: 'Session', description: 'If your session expires, sign in again. The page address still identifies the work you were viewing.' }
    ],
    tutorials: [
      { title: 'Open your workspace', steps: ['Choose your organization’s sign-in provider, or select Open local workspace on a local installation.', 'Complete sign-in. TestStudio opens the requested page.', 'Use the Workspace selector to choose a project.'] },
      { title: 'If no sign-in option appears', steps: ['Ask your installation administrator to configure a sign-in provider.', 'Reload this page after the administrator confirms the setup.'] }
    ]
  },
  projects: {
    id: 'projects', title: 'Projects',
    introduction: ['Projects keep related applications, test suites, and results together. Open a project to manage its catalog and runs.'],
    concepts: [
      { title: 'Project', description: 'A project provides the shared home for a test catalog, configuration, and run history.' },
      { title: 'Permanent identifier', description: 'Identifiers stay stable when display names change. Use letters, numbers, periods, underscores, or hyphens.' },
      { title: 'Permissions', description: 'The projects and controls you see depend on your access. Creating a project requires project creation permission.' }
    ],
    tutorials: [
      { title: 'Create your first project', steps: ['Enter a project name and permanent identifier in Create a project.', 'Add an optional description, then select Create project.', 'Open the project and use Project settings to register a suite.'] },
      { title: 'Return to existing work', steps: ['Open a project card to view its test catalog.', 'Use Test runs for execution history or Project settings for applications, suites, and environments.', 'Use the Workspace selector to switch projects.'] }
    ]
  },
  catalog: {
    id: 'catalog', title: 'Test catalog',
    introduction: ['The catalog describes what each test checks and how TestStudio runs it. Draft edits become executable through a published suite release.'],
    concepts: [
      { title: 'Test identity and lifecycle', description: 'A permanent identifier connects a test to its history. Enabled tests can be selected. Disabled and archived tests retain their definitions.' },
      { title: 'Draft and release', description: 'Editing changes the draft. Publication freezes a release. Validation checks definitions; it does not execute tests or prove they pass.' },
      { title: 'Safety', description: 'Each test has a safety label and rationale. Protected targets require safe definitions and the required permissions.' },
      { title: 'HTTP assertions', description: 'Native checks support GET, HEAD, accepted statuses, exact response headers, body text, JSON fields, response-time limits, and certificate lifetime. Redirects stay disabled. Expected headers describe responses, not authentication credentials.' },
      { title: 'Timing policy', description: 'Comparable clean runs build a duration baseline. Timing rules can warn or fail on slow execution and recommend bounded timeouts.' },
      { title: 'Catalog transfer', description: 'Import validates a JSON catalog before changes are applied. Export downloads definitions. Neither action executes tests.' }
    ],
    tutorials: [
      { title: 'Create and publish an HTTPS check', steps: ['Create a suite in Project settings if none exists.', 'Select + Create test. Choose the suite and enter an identifier, title, and category.', 'Enter the registered endpoint name and relative path. Choose GET or HEAD and the accepted status codes.', 'Add optional text, response header, JSON field, response-time, and certificate lifetime assertions. HEAD cannot assert response bodies.', 'Set the timeout ceiling, safety label, and rationale. Select Save draft changes.', 'Choose the suite under Publish a release. Select Validate and preview, review the result, then Publish and activate.'] },
      { title: 'Find and maintain a test', steps: ['Search by title, identifier, category, or tag. Use suite and lifecycle filters to narrow the list.', 'Use History to inspect retained definitions, or Edit to change the draft.', 'Use Archive to remove a test from normal selection. Choose Archived in the lifecycle filter to find and restore it.', 'Publish a new release when draft changes should apply to future runs.'] }
    ]
  },
  configuration: {
    id: 'configuration', title: 'Project settings',
    introduction: ['Register the applications you test, the suites that group their tests, and the environments those tests can target.'],
    concepts: [
      { title: 'Application', description: 'An application identifies the system being tested. Environments belong to an application.' },
      { title: 'Suite', description: 'A suite groups test definitions and published releases. A suite can require a target environment or run without one.' },
      { title: 'Environment and endpoint', description: 'An environment records a stage and named endpoint URLs. Test definitions refer to endpoint names instead of embedding a target URL.' },
      { title: 'Protection', description: 'Protection is separate from the stage name. A protected environment requires safe tests and protected-target permissions.' }
    ],
    tutorials: [
      { title: 'Prepare a target for HTTPS checks', steps: ['Choose Applications. Enter a name and permanent identifier, then select Create.', 'Choose Environments. Select the application, enter a name and identifier, and choose the stage.', 'Enter an endpoint name and URL. Use this endpoint name in test definitions.', 'Choose whether the environment needs production-safe tests, then select Create.'] },
      { title: 'Register a suite', steps: ['Choose Suites and enter a name and permanent identifier.', 'Choose an application when the suite belongs to one.', 'Keep Requires a target environment enabled for checks that use registered endpoints. Turn it off for tests that need no target.', 'Select Create, then open Test catalog to add and publish definitions.'] }
    ]
  },
  runs: {
    id: 'runs', title: 'Test runs',
    introduction: ['Start tests from a published suite and browse earlier runs. Each run captures its definitions, target settings, and timing criteria.'],
    concepts: [
      { title: 'Published suite', description: 'The run form lists suites with active published releases. Draft edits do not change existing runs.' },
      { title: 'Selection and dependencies', description: 'Leave test identifiers empty to select all eligible tests. An explicit selection also includes required dependencies.' },
      { title: 'Time limits and retries', description: 'The overall run deadline includes queue time. Each attempt also has a timeout. Retries remain separate attempts within one logical test result.' },
      { title: 'Timing estimate', description: 'Estimates use comparable history. Missing history stays unknown. The timeout budget includes permitted retries; expected execution excludes preparation and capture.' }
    ],
    tutorials: [
      { title: 'Start a run', steps: ['Select + New run, then choose a published suite in Start a run.', 'Choose a target environment when the suite requires one. Use No target only for suites that support it.', 'Optionally enter test identifiers. Set the run time limit and retries per test.', 'Select Estimate timing to review available history and timeout budgets.', 'Select Start run. Open the resulting details to follow progress and inspect evidence.'] },
      { title: 'Find an earlier run', steps: ['Use the history filters to narrow the suite, environment, state, or selection.', 'Open a run to inspect its frozen settings, results, attempts, and evidence.', 'Use the pagination controls to browse more history.'] }
    ]
  },
  run: {
    id: 'run', title: 'Run details',
    introduction: ['This page explains one run: what it used, what happened, how long tests took, and which evidence remains available.'],
    concepts: [
      { title: 'Run state and outcome', description: 'State describes execution progress. Outcome describes the result. An incomplete run retains uncertainty instead of becoming a pass.' },
      { title: 'Results and attempts', description: 'Final counts use logical test results. Attempt history keeps each retry. Timeouts have a separate timed-out status and count. A successful retry can leave a flaky result.' },
      { title: 'Test timing', description: 'Frozen predictions explain the criteria used for this run. Worker elapsed time includes coordination. Duration unavailable means no valid test measurement exists.' },
      { title: 'Evidence and retention', description: 'Logs and artifacts have capture and retention policies. Capture gaps do not erase test outcomes. Pins preserve evidence; deletion removes files while retaining results.' },
      { title: 'HTTP diagnostics', description: 'Expand HTTP diagnostics in attempt history to inspect network phases and certificate data. Unavailable phases stay unavailable. Certificate lifetime uses the retained attempt finish time.' },
      { title: 'Go diagnostics', description: 'Expand Go diagnostics for the selected package, test, toolchain, platform, build duration, and process exit codes. Build errors and missing selected tests retain unknown attempts and block the run. Recorded subtests remain steps within the registered logical test.' },
      { title: 'Notification deliveries', description: 'Completion messages use channel settings captured when the run was queued. Delivery failures remain separate from test failures. Inspect attempts, retry stopped delivery, or stop waiting retries without rerunning tests.' },
      { title: 'Worker recovery', description: 'After an interruption, recovery retains available diagnostics and removes owned resources. Recovered observations keep the original run outcome.' }
    ],
    tutorials: [
      { title: 'Investigate a failed or incomplete run', steps: ['Read the run summary and any execution or capture warning.', 'Compare final results with Attempt history. Expand recorded steps when available.', 'Inspect Run logs and Execution events. Filter logs by stream or test identifier when needed.', 'Open retained screenshots or other artifacts. Check Worker resources for pending cleanup or recovery gaps.', 'Use Rerun failures to repeat failed, timed-out, and flaky tests in a separate run. The original record remains unchanged.'] },
      { title: 'Understand timing and preserve evidence', steps: ['Review Test timing for the predictions and timeout criteria frozen when this run was queued.', 'Compare measured durations, expected values, and slowdown ratios. Functional outcomes remain visible beside timing failures.', 'Use the retention controls to pin evidence that needs further review.', 'If current timing history needs a reset, open Manage comparable timing history and provide a reason. Reset requires permission and does not change historical results.'] }
    ]
  }
}
