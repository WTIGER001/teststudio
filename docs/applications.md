# Applications

Choose a project, then select Applications in the Testing sidebar section.
The page lists the applications registered in that project.
Each card shows readable environment and linked suite counts.
Select Open application to view its detail page.

## Relationships

Applications belong to projects. Environments belong to applications.
Suites belong to projects and can link to zero, one, or several applications.
An application's Linked suites section shows those associations without changing suite ownership.
An unassigned suite remains available in the project catalog.
Archived suites retain their visible history and show an Archived badge.

Select View tests to open the catalog filtered to that suite.
Select Create test to carry an active suite into the test-type chooser.
The chooser preserves the suite while its data loads.

## Registration

Select Add application on the list page. Enter its name and permanent identifier, then select Create.
The new application's detail page opens after successful registration.
Select Add environment or Add suite to register a related resource.
The current application is preselected and fixed in these forms.
Complete the existing registration fields and select Create.
These forms share the Project settings registration component and service.
Project settings remains available for the existing configuration and suite lifecycle workflows.

Unsaved registration changes require confirmation before navigation or cancellation.
Project or application changes clear the previous registration view.
Delayed creation responses do not navigate an unrelated project or application.

## Access

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
| /projects/{project}/applications/{application} | Application environments and linked suites. |

Both routes provide Introduction, Concepts, and Tutorials in the shared help drawer.
