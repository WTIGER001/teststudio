import type { ProbeOverviewPage } from './features/probes/probe-overview.page'
import type { ProbeSetupPage } from './features/probes/probe-setup.page'
import type { CatalogPage } from './features/catalog/catalog.page'
import type { SuitesPage } from './features/configuration/suites.page'
import type { ProjectsPage } from './features/projects/projects.page'
import type { CreateProjectPage } from './features/projects/create-project.page'
import type { ApplicationsPage } from './features/applications/applications.page'
import type { CreateTestPage } from './features/catalog/create-test.page'
import type { SourcesPage } from './features/sources/sources.page'
import type { StatusEditorPage } from './features/status-pages/status-editor.page'
import type { AssignmentsPage } from './features/access/assignments.page'
import type { Routes } from '@angular/router'

const projectRoutes: Routes = [
 { path: 'projects/:projectId/status-pages/new', canDeactivate: [(page: StatusEditorPage) => page.canLeave()], data: { helpTopic: 'status-pages', mode: 'create' }, loadComponent: () => import('./features/status-pages/status-editor.page').then(m => m.StatusEditorPage) },
 { path: 'projects/:projectId/status-pages/:pageId/edit', canDeactivate: [(page: StatusEditorPage) => page.canLeave()], data: { helpTopic: 'status-pages', mode: 'edit' }, loadComponent: () => import('./features/status-pages/status-editor.page').then(m => m.StatusEditorPage) },
 { path: 'projects/:projectId/status-pages/:pageId/publish', data: { helpTopic: 'status-pages', mode: 'publish' }, loadComponent: () => import('./features/status-pages/status-editor.page').then(m => m.StatusEditorPage) },
 { path: 'projects/:projectId/status-pages/:pageId', canDeactivate: [(page: StatusEditorPage) => page.canLeave()], data: { helpTopic: 'status-pages' }, loadComponent: () => import('./features/status-pages/status-editor.page').then(m => m.StatusEditorPage) },
 { path: 'projects/:projectId/status-pages', data: { helpTopic: 'status-pages' }, loadComponent: () => import('./features/status-pages/status-pages.page').then(m => m.StatusPagesPage) },
  { path: 'projects/:projectId/applications/:applicationId/members', canDeactivate: [(page: AssignmentsPage) => page.canLeave()], data: { helpTopic: 'application-members' }, loadComponent: () => import('./features/access/assignments.page').then(m => m.AssignmentsPage) },
  { path: 'projects/:projectId/applications/:applicationId/explain-access', data: { helpTopic: 'effective-permissions' }, loadComponent: () => import('./features/access/permission-explanation.page').then(m => m.PermissionExplanationPage) },
  { path: 'projects/:projectId/probes/new', canDeactivate: [(page: ProbeSetupPage) => page.canLeave()], data: { helpTopic: 'https-probes' }, loadComponent: () => import('./features/probes/probe-setup.page').then(m => m.ProbeSetupPage) },
  { path: 'projects/:projectId/probes/adopt', canDeactivate: [(page: ProbeSetupPage) => page.canLeave()], data: { helpTopic: 'https-probes', adopt: true }, loadComponent: () => import('./features/probes/probe-setup.page').then(m => m.ProbeSetupPage) },
  { path: 'projects/:projectId/probes', canDeactivate: [(page: ProbeOverviewPage) => page.canLeave()], data: { helpTopic: 'https-probes' }, loadComponent: () => import('./features/probes/probe-overview.page').then(m => m.ProbeOverviewPage) },
  { path: 'projects/:projectId/suites', canDeactivate: [(page: SuitesPage) => page.canLeave()], data: { helpTopic: 'suites' }, loadComponent: () => import('./features/configuration/suites.page').then(m => m.SuitesPage) },
  { path: 'projects/:projectId/applications', canDeactivate: [(page: ApplicationsPage) => page.canLeave()], data: { helpTopic: 'applications' }, loadComponent: () => import('./features/applications/applications.page').then(m => m.ApplicationsPage) },
  { path: 'projects/:projectId/applications/:applicationId', canDeactivate: [(page: ApplicationsPage) => page.canLeave()], data: { helpTopic: 'applications' }, loadComponent: () => import('./features/applications/applications.page').then(m => m.ApplicationsPage) },
  { path: 'projects/:projectId/tests/new', canDeactivate: [(page: CreateTestPage) => page.canLeave()], data: { helpTopic: 'create-test' }, loadComponent: () => import('./features/catalog/create-test.page').then(m => m.CreateTestPage) },
  { path: 'projects/:projectId/tests/new/:kind', canDeactivate: [(page: CreateTestPage) => page.canLeave()], data: { helpTopic: 'create-test' }, loadComponent: () => import('./features/catalog/create-test.page').then(m => m.CreateTestPage) },
  { path: 'projects/:projectId/tests/:suiteId/:testId/history', data: { helpTopic: 'test-history' }, loadComponent: () => import('./features/test-history/test-history.page').then(m => m.TestHistoryPage) },
  { path: 'projects/:projectId/releases', data: { helpTopic: 'releases' }, loadComponent: () => import('./features/releases/releases.page').then(m => m.ReleasesPage) },
  { path: 'cli-sign-in/:requestId', data: { helpTopic: 'cli-signin' }, loadComponent: () => import('./features/cli-signin/cli-signin.page').then(m => m.CLISignInPage) },
  { path: 'projects/:projectId/sources', canDeactivate: [(page: SourcesPage) => page.canLeave()], data: { helpTopic: 'sources' }, loadComponent: () => import('./features/sources/sources.page').then(m => m.SourcesPage) },
  { path: 'settings/providers', data: { helpTopic: 'providers' }, loadComponent: () => import('./features/access/providers.page').then(m => m.ProvidersPage) },
  { path: 'settings/providers/:providerId', data: { helpTopic: 'provider' }, loadComponent: () => import('./features/access/provider-detail.page').then(m => m.ProviderDetailPage) },
  { path: '', pathMatch: 'full', redirectTo: 'projects' },
  { path: 'account', data: { helpTopic: 'account' }, loadComponent: () => import('./features/account/account.page').then(m => m.AccountPage) },
  { path: 'account/access', data: { helpTopic: 'effective-permissions' }, loadComponent: () => import('./features/access/permission-explanation.page').then(m => m.PermissionExplanationPage) },
  { path: 'projects/:projectId/explain-access', data: { helpTopic: 'effective-permissions' }, loadComponent: () => import('./features/access/permission-explanation.page').then(m => m.PermissionExplanationPage) },
  { path: 'projects/new', canDeactivate: [(page: CreateProjectPage) => page.canLeave()], data: { helpTopic: 'projects' }, loadComponent: () => import('./features/projects/create-project.page').then(m => m.CreateProjectPage) },
  { path: 'projects', canDeactivate: [(page: ProjectsPage) => page.canLeave()], data: { helpTopic: 'projects' }, loadComponent: () => import('./features/projects/projects.page').then(m => m.ProjectsPage) },
  { path: 'settings/access', canDeactivate: [(page: AssignmentsPage) => page.canLeave()], data: { helpTopic: 'installation-access' }, loadComponent: () => import('./features/access/assignments.page').then(m => m.AssignmentsPage) },
  { path: 'settings/permissions', data: { helpTopic: 'grants' }, loadComponent: () => import('./features/access/grants.page').then(m => m.GrantsPage) },
  { path: 'projects/:projectId/permissions', data: { helpTopic: 'grants' }, loadComponent: () => import('./features/access/grants.page').then(m => m.GrantsPage) },
  { path: 'settings/roles', data: { helpTopic: 'roles' }, loadComponent: () => import('./features/access/role-policies.page').then(m => m.RolePoliciesPage) },
  { path: 'settings/roles/:roleId', data: { helpTopic: 'role' }, loadComponent: () => import('./features/access/role-policy-detail.page').then(m => m.RolePolicyDetailPage) },
  { path: 'projects/:projectId/roles', data: { helpTopic: 'roles' }, loadComponent: () => import('./features/access/role-policies.page').then(m => m.RolePoliciesPage) },
  { path: 'projects/:projectId/roles/:roleId', data: { helpTopic: 'role' }, loadComponent: () => import('./features/access/role-policy-detail.page').then(m => m.RolePolicyDetailPage) },
  { path: 'projects/:projectId/members', canDeactivate: [(page: AssignmentsPage) => page.canLeave()], data: { helpTopic: 'project-access' }, loadComponent: () => import('./features/access/assignments.page').then(m => m.AssignmentsPage) },
  { path: 'settings/users', data: { helpTopic: 'users' }, loadComponent: () => import('./features/access/users.page').then(m => m.UsersPage) },
  { path: 'settings/users/:userId', data: { helpTopic: 'user' }, loadComponent: () => import('./features/access/user-detail.page').then(m => m.UserDetailPage) },
  { path: 'settings/groups', data: { helpTopic: 'groups' }, loadComponent: () => import('./features/access/groups.page').then(m => m.GroupsPage) },
  { path: 'settings/groups/:groupId', data: { helpTopic: 'group' }, loadComponent: () => import('./features/access/group-detail.page').then(m => m.GroupDetailPage) },
  { path: 'projects/:projectId/groups', data: { helpTopic: 'groups' }, loadComponent: () => import('./features/access/groups.page').then(m => m.GroupsPage) },
  { path: 'projects/:projectId/groups/:groupId', data: { helpTopic: 'group' }, loadComponent: () => import('./features/access/group-detail.page').then(m => m.GroupDetailPage) },
  { path: 'projects/:projectId/catalog', canDeactivate: [(page: CatalogPage) => page.canLeave()], data: { helpTopic: 'catalog' }, loadComponent: () => import('./features/catalog/catalog.page').then(m => m.CatalogPage) },
  { path: 'projects/:projectId/configuration', data: { helpTopic: 'configuration' }, loadComponent: () => import('./features/configuration/configuration.page').then(m => m.ConfigurationPage) },
  { path: 'projects/:projectId/runs', data: { helpTopic: 'runs' }, loadComponent: () => import('./features/runs/run-list.page').then(m => m.RunListPage) },
  { path: 'projects/:projectId/trends', data: { helpTopic: 'trends' }, loadComponent: () => import('./features/trends/trends.page').then(m => m.TrendsPage) },
  { path: 'projects/:projectId/schedules', data: { helpTopic: 'schedules' }, loadComponent: () => import('./features/schedules/schedules.page').then(m => m.SchedulesPage) },
  { path: 'projects/:projectId/notifications', data: { helpTopic: 'notifications' }, loadComponent: () => import('./features/notifications/notifications.page').then(m => m.NotificationsPage) },
  { path: 'projects/:projectId/runs/:runId', data: { helpTopic: 'run' }, loadComponent: () => import('./features/runs/run-detail.page').then(m => m.RunDetailPage) },
  { path: '**', redirectTo: 'projects' }
]

export const routes: Routes = [
 { path: 'status/:token', loadComponent: () => import('./features/status-pages/public-status.page').then(m => m.PublicStatusPage) },
  ...projectRoutes.filter(route => /^projects\/:projectId\/(suites|catalog|runs|trends|tests)(\/|$)/.test(route.path ?? '')).map(route => ({ ...route, path: route.path!.replace('projects/:projectId/', 'projects/:projectId/applications/:applicationId/') })),
  ...projectRoutes
]
