# Applications

Choose a project, then select All applications under Project overviews.
The page lists the applications registered in that project.
Each card shows readable environment and linked suite counts.
Select Open application to make it the active application.
The sidebar starts with Projects, followed by the Project and Application selectors.
Use the Application selector to switch quickly.
Your account stays in the global header. It applies across TestStudio, with project access assigned separately.
The context above the page shows the project and active application.
Suites, Test catalog, Test runs, and Trends follow this selection.
Choose All applications for project-wide work. HTTPS probes provides a separate project overview.

## Relationships

Applications belong to projects. Environments belong to applications.
Suites belong to projects and can link to zero, one, or several applications.
An application's Linked suites section shows those associations without changing suite ownership.
An unassigned suite remains available under All applications.
A test with an explicit application appears only in that application.
A test without an explicit application inherits its suite links.
Shared suites keep one draft and release. Publication and transfer affect the complete suite.
Archived suites retain their visible history and show an Archived badge.

Select View tests to open the catalog filtered to that suite.
Select Create test to carry an active suite into the test-type chooser.
The chooser preserves the suite while its data loads.

## Registration

Select Add application on the list page. Follow Name, Options, and Review, then select Create.
The new application's detail page opens after successful registration.
Select Add environment or Add suite to register a related resource.
The current application is preselected and fixed in these forms.
Complete the wizard and select Create on its Review step.
These wizards share the Project settings registration component and service.
Project settings remains available for the existing configuration and suite lifecycle workflows.

Unsaved registration changes require confirmation before navigation or cancellation.
Project or application changes clear the previous registration view.
Delayed creation responses do not navigate an unrelated project or application.

## Access

Environment registration provides Advanced credentials for optional API authentication.
Map a binding name to a current project's approved reference. Review the mapping and target before creating the environment.
The operator controls secret values and permitted origins. See [API credentials](api-credentials.md).

Application lists require project read permission. Application creation requires project management permission.
Environment and suite read permissions control the related lists independently.
Missing read permission displays an explanation instead of an empty resource claim.
The client does not request registries that its current capabilities do not permit.
The existing API still checks current permissions for each read and registration request.
Management permissions control the environment and suite registration buttons.

## Routes

| Route | Purpose |
| --- | --- |
| /projects/{project}/applications | Application list and registration. |
| /projects/{project}/applications/{application} | Active application overview, environments, and linked suites. |
| /projects/{project}/applications/{application}/suites | Suites linked to the active application. |
| /projects/{project}/applications/{application}/catalog | Tests and summary counts for the active application. |
| /projects/{project}/applications/{application}/runs | Application run history and creation. |
| /projects/{project}/applications/{application}/trends | Application result trends. |
| /projects/{project}/probes | HTTPS probes across project applications. |

These pages provide Introduction, Concepts, and Tutorials in the shared help drawer.
Application routes retain context after reload or browser Back. Switching applications clears old filters.
Unsaved-work confirmation can cancel the switch. The selector then keeps the original application.

## Guided registration

Application, suite, and environment registration share Name, Options, and Review steps.
The name generates a permanent identifier. Advanced details contains its editor.
Environment options keep the target URL, stage, and protection visible.
The endpoint alias defaults to api. Advanced endpoint options contains its editor.
Completed step buttons and Back preserve entries. Create remains the final action.

## First project setup

Projects provides Project, Application, Target, and Review steps.
Users can include a first application, target environment, and empty Availability suite.
Review lists the selected resources before creation. Setup creates no tests and starts no runs.
Each resource uses the existing API and current permissions.
If setup stops, successful resources remain. Retry remaining setup skips confirmed successful steps.
Entries lock after project creation. Open the created project provides access to existing resources.
Reloading ends the in-page continuation. Users can manage retained resources from Applications or Project settings.

## Project creator access

Project creation now gives ordinary authorized creators project administration, test editing, publishing, and run roles for the new project.
Creation saves the project and initial access together. It grants no installation administration.
The project wizard explains this access on Review before creation.
This lets a permitted creator complete the first application, target, and suite setup without an installation administrator session.
If a required predefined role is disabled, creation fails without saving partial project access.
Existing installation administrators use their existing authority without automatic project self-assignment.
Project edits do not restore revoked roles. Existing projects receive no automatic access changes.
See [Initial creator access](access.md#initial-project-creator-access) for API behavior, policy, and recovery.
