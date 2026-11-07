# System project and application role plan

Status: Backend role and access APIs are implemented locally on 2026-11-07. API contract 0.37.0 and the generated client cover scoped assignments, replacement, explanations, and reviewed conversion. UI work and target deployment verification remain pending. The hosted release and existing assignments are unchanged.

Use System for the installation and keep Project, Application, and Group.
Offer five cumulative roles at Project and Application scope: Reader, Member, Developer, Contributor, and Owner.
Keep System Admin separate. Show the source and scope of effective access wherever membership is managed.

## Structure and ownership

| Term | Responsibility |
| --- | --- |
| System | Contains the installation, internal user directory, sign-in providers, workers, and projects. |
| Project | Contains applications, shared suites, project groups, and project access policy. |
| Application | Represents a product or service under one project and owns its environments. |
| Group | Collects users so they can receive access together. It belongs permanently to the System or one Project. |

Keep suite ownership at Project scope. Suites can link to zero, one, or several applications.
Application access controls the resources someone may use; it does not move suites or duplicate their releases.
Keep project and application identifiers, existing routes, databases, and retained history.
Use System in navigation and help. Preserve installation terminology in compatible API identifiers where renaming would break clients.

## Role definitions

Reader views permitted tests, results, trends, schedules, and evidence.
Member adds execution and scheduling of published tests.
Developer adds test authoring, publication, and source and suite management within the permitted boundary.
Contributor adds settings and membership management, with every Owner capability except deleting resources and adding, removing, or changing Owners.
Owner adds resource deletion and ownership management for that boundary.

The hierarchy is Reader, Member, Developer, Contributor, then Owner. A Developer includes Member and Reader capabilities. An Owner includes Contributor capabilities.
Explicit denies, disabled roles, inactive membership, target protections, and retention policy still constrain effective access.

| Capability within the assigned scope | Reader | Member | Developer | Contributor | Owner |
| --- | --- | --- | --- | --- | --- |
| View tests, releases, results, trends, and schedule state | Yes | Yes | Yes | Yes | Yes |
| View and download permitted run evidence | Yes | Yes | Yes | Yes | Yes |
| Execute published tests and rerun failures | No | Yes | Yes | Yes | Yes |
| Cancel runs and create, edit, pause, or archive execution schedules | No | Yes | Yes | Yes | Yes |
| Create and edit test drafts, including HTTPS probes and recordings | No | No | Yes | Yes | Yes |
| Import and export definitions, archive and restore tests | No | No | Yes | Yes | Yes |
| Publish, activate, or roll back suite releases | No | No | Yes | Yes | Yes |
| Manage suites, test sources, and repository synchronization | No | No | Yes | Yes | Yes |
| Manage project or application settings, environments, and approved credential bindings | No | No | No | Yes | Yes |
| Manage notification destinations and delivery settings | No | No | No | Yes | Yes |
| Add, remove, or change Reader, Member, Developer, and Contributor assignments | No | No | No | Yes | Yes |
| Add, remove, promote, or demote Owners and transfer ownership | No | No | No | No | Yes |
| Manage evidence pins within retention policy | No | No | No | Yes | Yes |
| Delete resources or evidence where supported and permitted by retention policy | No | No | No | No | Yes |

Members operate any permitted run or execution schedule in their scope, including work created by another user.
Repository synchronization changes executable definitions, so its configuration and schedules require Developer access.
Members can inspect notification delivery outcomes associated with permitted runs. Changing destinations, retrying deliveries, and stopping delivery work require Contributor access.

Deletion removes resource or evidence data. This restriction applies to every role below Owner, including schedule, source, environment, application, and project deletion where supported.
Reversible archive and restore remain available at the role needed to manage that resource when they preserve retained data and ownership controls.
Removing a non-Owner membership is an allowed Contributor action; it does not grant deletion authority over product resources.
Any change that triggers immediate evidence deletion or uses retention settings to force deletion requires Owner authority.

Protected execution and capture overrides remain separate Advanced permissions. None of the five standard roles grants them automatically.
Owning an environment does not permit removing its protection or changing safety controls to bypass those permissions.
Allow normal authoring with safe defaults. Require distinct policy-management authority to reduce protection, weaken safety criteria, expand approved origins, or override capture safeguards; test both old and proposed resource state.
Granting these Advanced exceptions requires current authority over the action and scope. Owner alone cannot self-grant them through a custom role or direct rule.
System Admin retains its current registered administrative permissions, subject to explicit denies and normal safety validation.
Secret values never appear in membership or credential-reference responses. Approved bindings remain constrained by their target origins.
Reader access includes evidence download; use an explicit deny or a custom role when a project needs stricter evidence access.

## System administration

System Admin manages users, account recovery, sign-in providers, system groups, worker configuration, global policies, and project creation authority.
It retains the existing Installation administrator's access across projects and applications through the permission service.
Keep the existing role identifier compatible while changing its display name to System Admin.
System Admin is never assignable from a Project or Application membership endpoint.

Creating projects requires System-scoped projects.create authority. Preserve explicit delegation of this action without requiring System Admin.
Ordinary authorized creators receive one Project Owner assignment atomically with their new project.
System Admin creators keep the current behavior of using system authority without automatic project self-assignment.
Their new projects show System-managed ownership until an explicit Project Owner is assigned. Ordinary creator projects adopt the Owner model in their creation transaction.
New applications inherit Project roles and do not add separate creator grants.
Creating applications requires Project Contributor authority. Application Contributor or Owner does not authorize creating siblings.
Project Contributors and Owners manage project settings, private project images, and project groups within the deletion and ownership restrictions.
Application Contributors and Owners manage only their own application settings and environments.

Project and Application roles grant no authority over system users, passwords, provider configuration, system groups, worker registration, or global role state.
Successful local or OIDC sign-in grants no project or application membership.
Keep issuer-subject identity isolation, explicit account linking, and provider-managed membership rules.

## Scope and inheritance

Project membership applies to that project and all its current and future applications.
Application membership applies only to one application and its authorized resources.
Application-only members see enough parent-project name and navigation to open their application; they do not receive project-wide read permission.
Project lists, application lists, counts, search, overviews, and capability responses must all enforce this distinction.

Combine applicable direct and group roles by permission. Among standard roles this normally produces the highest applicable role.
A lower direct role does not cancel a higher inherited role. An applicable deny still overrides allows.
Custom roles and direct grants can produce exceptions, so a role label never replaces a current permission check.
Show inherited access as read-only and link authorized users to the scope where it can be changed.
Removing one assignment removes only that path. Explain any remaining inherited, group, custom-role, or direct-grant access.

| Example assignment | Result |
| --- | --- |
| Project Reader and Website Developer | Read the project; author and run Website tests; read sibling applications. |
| Website Reader only | Read Website and minimal parent context; no access to Admin or Search. |
| Project Developer and Website Reader | Developer access still applies to Website through the Project. |
| Website Owner only | Manage Website membership and settings; cannot manage its Project, sibling applications, or shared groups. |
| Group Member on a Project | Active group members can operate permitted tests throughout that project. |
| Website Contributor only | Manage Website tests, settings, and non-Owner members; cannot delete resources or change Owners. |

Do not add nested projects, nested groups, or application-owned groups in this delivery.
Project Contributors and Owners manage project groups subject to ownership and deletion restrictions. System group administration remains System-scoped.
Application Contributors and Owners may assign existing users directly within their delegation limits. Assigning or revoking a role for a shared group also requires authority over that group's stored ownership scope, preserving the current guard.
Group membership changes must evaluate every affected assignment. Application administration never grants permission to add oneself to a project group.
Contributors cannot change membership, archive, or disable a group when that change would add, remove, or alter an Owner access path, even if another Owner path remains.
Provider-owned membership remains read-only and expires through the existing provider rules.

## Resource boundaries

An application selector currently filters project resources. It must become backed by server authorization before application membership is offered.
Resolve stored ownership and affected resources before checking permission. Never trust a client-supplied application identifier to narrow a broader mutation.
Apply the same boundary to direct object routes, exports, background jobs, callbacks, and private artifact access.

| Resource | Application access rule |
| --- | --- |
| Application and environments | Require the role's action for that exact application. Prevent moving resources into scope through an update. |
| Suite linked only to one application | Allow permitted authoring and publication when all definitions, sources, dependencies, and mutation effects stay within that application. |
| Suite linked to several applications | Permit scoped test views and eligible execution. Require Project Developer authority for complete source access, authoring, publication, activation, transfer, or link changes. |
| Suite with no application links | Require Project authority. Do not assign it to an application implicitly. |
| Run and evidence | Use frozen execution context and authorized test selection. Never grant a full mixed-application run or shared artifact because one test matches. |
| Execution schedule or probe binding | Check the full target, selected tests, dependency closure, release policy, and current owner's authority. Block firing after access loss. |
| Source or repository | Application Developers manage it only when all bound suites and preparation outputs stay in their application. Shared source changes require Project Developer authority. |
| Notification channel | Application Contributors manage application-only destinations and filters. Project-wide or mixed channels require Project Contributor authority. Deletion requires the corresponding Owner authority. |

Keep existing filtered mixed-run views for users with Project authority.
For application-only members, reject a mixed run's complete detail, logs, exports, and artifact routes in the initial delivery.
Application history may show only authorized summary fields with no mixed-run identifiers or links until a separately authorized projection exists.
Do not expose full shared suite source or release payloads through a filtered catalog view.
Members can execute an authorized subset of a shared published suite only when its dependencies and target remain authorized.

Historical resource authorization uses retained application context, so later suite relinking cannot expose older results.
Old runs without trustworthy application boundaries remain available to Project-authorized users and unavailable to application-only members.
Recheck access when issuing and consuming artifact grants, downloading prepared source, publishing, enabling schedules, and dispatching jobs.
An application-only worker request must not receive code, artifacts, or credentials outside the execution resources it requires.

## Membership and policy controls

Use one standard direct role per user or group at each Project or Application scope.
Preserve additional custom-role and direct-grant paths under Advanced permissions.
Role replacement is one revision-checked transaction; it must not temporarily remove or combine access during separate saves.
Record scope, actor, principal, previous role, new role, reason, revision, and change time in the audit trail.

Owners can assign Reader, Member, Developer, Contributor, or Owner within their own scope.
Contributors can assign Reader, Member, Developer, or Contributor and change or revoke those assignments. They cannot change an existing Owner assignment or promote anyone, including themselves, to Owner.
Project Contributors and Owners manage direct Application membership inside their project within these delegation limits.
Application Contributors and Owners cannot modify Project membership or inherited assignments.
Project Contributors and Owners manage project custom roles and direct grants only within their current delegation authority. Application Contributors and Owners delegate standard roles only in this delivery.
Contributors cannot grant deletion or ownership powers through custom roles, inheritance, direct grants, or group changes. They cannot alter a policy or deny in a way that changes an Owner access path.
Check the complete proposed permission change in the transaction, including every affected group assignment and inherited policy. Protect all Owner paths, beyond the last usable Owner safeguard.
Disabled roles remain visible and cannot receive new assignments. Global role state changes require System authority.

Preserve the existing last usable System administrator safeguard across assignments, groups, credentials, and provider changes.
Add atomic checks against removing the last usable Project Owner after a project adopts the standard Owner model.
An Application can rely on a usable Project Owner; it need not have a direct Application Owner.
Check usable access rather than counting assignment rows, including active membership, enabled accounts and roles, denies, and configured sign-in paths.
External expiry can still remove the last owner. Report the gap to authorized administrators and provide audited System Admin recovery without silently restoring access.
Legacy projects need an explicit owner-adoption step; startup must not create owners automatically.

## API and implementation design

Continue using github.com/wtiger001/go-permissions through permissions.Service for all decisions.
Extend TestStudio's managed assignment and resource-context model with explicit System, Project, and Application scope.
Keep existing project team bindings. Derive application bindings and exact objects from route context and stored ownership.
Prove the library representation for application inheritance, object denies, expiry, and revocation in the first implementation milestone before choosing its concrete grant encoding.
Do not implement a second permission evaluator or treat the five role names as authorization shortcuts.

Separate application read and management actions from today's project-wide projects.manage check.
Create an explicit action-to-role manifest covering every registered action and scope, including safety controls and membership delegation.
Extend resource validation, capabilities, assignment cursors, effective-access explanations, role-state handling, and audits together.
Unscoped project capabilities must not be reused for application-only sessions.
Keep configured OIDC group mappings bound to their actual project and application assignment scope; an email claim never grants Owner access.

Preserve existing System and Project endpoint compatibility. Add application membership, scoped capability, and access-explanation operations to the canonical OpenAPI contract and generated identity client.
Bind cursors and revisions to actor, principal, and complete scope. Recheck authority inside mutation transactions.
Persist managed assignment state and permission-library changes atomically on SQLite and PostgreSQL.

Primary implementation areas are internal/authorization, internal/access, internal/identity, internal/projects, internal/catalog, internal/runs, internal/schedules, internal/sources, internal/httpapi, contracts/openapi.yaml, and the Angular workspace and access services.
Inventory remaining evidence, probe, notification, recorder, and worker dispatch paths before declaring application isolation complete.

## Interface and help

Rename Installation administration to System administration and Installation administrator to System Admin in visible labels.
Keep Projects, Applications, and Groups. Use Project members and Application members for standard membership pages.
Reuse the current Project members route and add an Application members route within its existing application context.
Keep custom policies, exact grants, and compatibility assignments under named Advanced sections.

Each member row shows principal, direct role, effective role or Custom access, source, scope, and current activity state.
Distinguish Direct, From project, and From group. Explain why lowering or removing one role may leave access intact.
Use server-provided management capabilities. Application Contributors and Owners get an authorized existing-user selector without access to the system user directory.
Keep the Member role distinct from the generic members list label. Show Contributor below Owner in role choices, and explain deletion and ownership restrictions in its description.
Role changes review the affected scope and new powers before saving. Preserve unsaved values, revision conflicts, and uncertain-save reconciliation.

Add Introduction, Concepts, and Tutorials to every changed membership, role, and access-explanation page.
Tutorials cover adding a Reader, granting Member execution access, assigning Developer to one application, delegating Contributor management, inspecting inheritance, and transferring ownership.
Verify keyboard focus, light and dark themes, system theme, and narrow layouts. Help must preserve edits.

## Migration and compatibility

Introduce new role identifiers without broadening the permission definitions of existing project roles.
Retain existing grants, role disables, explicit denies, provider mappings, direct access paths, assignment revisions, and audit history.
Rename the existing installation administrator display label compatibly.
New projects use the standard roles after their scoped implementation passes; existing projects keep their effective policies until reviewed conversion.

| Existing assignment | Conversion treatment |
| --- | --- |
| Viewer | Reader is a candidate when its complete effective permissions match. |
| Operator | Member is a candidate after comparing every action. Preserve any existing schedule-delete authority until an explicit conversion removes it. Contributor would grant substantially broader administration. |
| Catalog editor | Keep as a compatibility role; Developer would add running and publication. |
| Publisher | Keep as a compatibility role; Developer would add authoring and execution. |
| Project administrator | Keep as a compatibility role; Contributor would add test and execution powers, and Owner would also add deletion and ownership powers. |
| Current four-role creator bundle | Offer reviewed Owner conversion. Show every added and removed action; the new Owner definition includes additional management actions. |
| Custom roles, direct grants, and provider mappings | Preserve exact scope and semantics. Convert only through an explicit reviewed operation. |

Provide a bounded conversion preview per project with changed actions, affected principals, group paths, disabled policies, and schedules that could become blocked.
An authorized owner or System Admin reviews and applies the exact preview with revisions and a reason.
Legacy Project administrator delegation does not authorize granting the broader new Contributor or Owner roles. Require current authority over every added action or System Admin assistance for that conversion.
Treat every permission expansion as an explicit grant. Do not infer equivalence from role names or a sample of successful API calls.
Commit each conversion atomically with its audit and owner-adoption state. Preserve request identity for uncertain responses and retry.
Rollback restores recorded assignments only after showing any resulting access changes and validating current authority and owner safeguards.
Startup never repeats migration or restores revoked grants.

## Delivery milestones and verification

The user authorizes API implementation on 2026-11-07. Backend work for milestones 1 through 3 now has focused SQLite and PostgreSQL evidence.
The [access guide](access.md#standard-project-and-application-roles) describes the implemented endpoints and compatibility rules.
Milestone 4 remains UI work. Milestone 5 remains release publication and target verification.

This planning milestone maps primarily to acceptance 10, 12, and 13, with follow-on coverage for 05, 07, 14, 18, 38, 45, 49, 50, 54, and 55.
The authoritative ledger stays at 50 verified of 57. Scope changes require fresh targeted evidence before retaining the affected acceptance claims for the new behavior.

| Milestone | Deliverable and exit condition | Acceptance | Smallest useful local verification |
| --- | --- | --- | --- |
| 1 | Define the action manifest, five cumulative roles, System labels, scoped assignment representation, and compatibility policy. Prove application scope through permissions.Service before exposing it. | 10, 12, 13 | Focused authorization and access checks for role inclusion, Member execution, Contributor management, Owner-only deletion and ownership, scope isolation, inherited paths, denies, role disables, and expiry on SQLite and PostgreSQL. |
| 2 | Enforce Application boundaries across registries, authoring, source preparation, runs, evidence, schedules, probes, notifications, and dispatch. No application membership UI ships before this passes. | 12, 13, 14, 18, 38, 45 | Exact authorized and rejected API/service cases for application-only users, mixed resources, forged scope, relinking, dependency closure, revocation, artifact reuse, and next schedule firing. |
| 3 | Deliver atomic scoped membership APIs, owner safeguards, creator Owner assignment, compatible migration preview and apply, and generated contracts. | 05, 07, 10, 12, 13, 14 | Focused checks on both databases for Contributor changes to lower roles, rejected direct and indirect Owner changes, last-owner guards, restart persistence, competing changes, conversion retries and rollback, and generated-client synchronization. |
| 4 | Deliver System labels, Project and Application members, inherited access explanations, Advanced compatibility controls, and contextual help. | 10, 49, 50 | UI build and the changed STUDIO-ROLES-001 and STUDIO-PERMISSIONS-001 workflows through normal local execution. Add a cataloged STUDIO-APPLICATION-ACCESS-001 workflow for application-only navigation and rejected sibling access. |
| 5 | Update operator and feature guides, create a versioned release from committed source, and verify the selected target's identity and permission boundaries. | 54, 55 and affected scoped criteria | Reuse passing local evidence. On MMDev, verify one generic project with two applications, one inherited member, one application-only member, one group path, and evidence revocation. Keep cross-profile claims separate. |

Select individual existing browser IDs only when their behavior changes; STUDIO-ACCESS-001, STUDIO-POLICY-001, STUDIO-GRANTS-001, STUDIO-BUILTIN-001, and STUDIO-PROVIDERS-001 remain available for focused follow-up.
Register the new application workflow before selecting it. Do not run the complete catalog by default.
No UI build, browser run, hosted membership conversion, or cloud mutation is needed for the API-only implementation request.
Choose the product version through the version helper before publishing the changed runtime. API contract version 0.37.0 remains separate.
Preserve the user's port 8040 demo, database, history, and all unrelated workloads. Azure implementation and verification stay deferred.

## Workflow completion checklist

- Map every registered action and every resource family before enabling application-only membership.
- Implement server enforcement, canonical API schemas, generated client types, and UI capabilities together.
- Update access, applications, first-installation, source, scheduling, evidence, UI, and help guides for each changed workflow.
- Verify positive access, rejected access, direct routes, expiry, revocation, revisions, and recovery with explicit local selections.
- Record local evidence separately from target verification. Shut down owned verification resources.
- Commit verified implementation, use the version helper before publication, and build the release from committed source.
- Update continuation notes and the 57-item ledger without closing broad cross-profile requirements prematurely.

Related current behavior: [Access administration](access.md), [Applications](applications.md), [UI rules](ui.md), [Self testing](self-testing.md), and [Versioning](versioning.md).
