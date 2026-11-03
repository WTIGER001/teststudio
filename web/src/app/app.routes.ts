import type { Routes } from '@angular/router'

export const routes: Routes = [
  { path: '', pathMatch: 'full', redirectTo: 'projects' },
  { path: 'projects', data: { helpTopic: 'projects' }, loadComponent: () => import('./features/projects/projects.page').then(m => m.ProjectsPage) },
  { path: 'settings/users', data: { helpTopic: 'users' }, loadComponent: () => import('./features/access/users.page').then(m => m.UsersPage) },
  { path: 'settings/users/:userId', data: { helpTopic: 'user' }, loadComponent: () => import('./features/access/user-detail.page').then(m => m.UserDetailPage) },
  { path: 'settings/groups', data: { helpTopic: 'groups' }, loadComponent: () => import('./features/access/groups.page').then(m => m.GroupsPage) },
  { path: 'settings/groups/:groupId', data: { helpTopic: 'group' }, loadComponent: () => import('./features/access/group-detail.page').then(m => m.GroupDetailPage) },
  { path: 'projects/:projectId/groups', data: { helpTopic: 'groups' }, loadComponent: () => import('./features/access/groups.page').then(m => m.GroupsPage) },
  { path: 'projects/:projectId/groups/:groupId', data: { helpTopic: 'group' }, loadComponent: () => import('./features/access/group-detail.page').then(m => m.GroupDetailPage) },
  { path: 'projects/:projectId/catalog', data: { helpTopic: 'catalog' }, loadComponent: () => import('./features/catalog/catalog.page').then(m => m.CatalogPage) },
  { path: 'projects/:projectId/configuration', data: { helpTopic: 'configuration' }, loadComponent: () => import('./features/configuration/configuration.page').then(m => m.ConfigurationPage) },
  { path: 'projects/:projectId/runs', data: { helpTopic: 'runs' }, loadComponent: () => import('./features/runs/run-list.page').then(m => m.RunListPage) },
  { path: 'projects/:projectId/trends', data: { helpTopic: 'trends' }, loadComponent: () => import('./features/trends/trends.page').then(m => m.TrendsPage) },
  { path: 'projects/:projectId/schedules', data: { helpTopic: 'schedules' }, loadComponent: () => import('./features/schedules/schedules.page').then(m => m.SchedulesPage) },
  { path: 'projects/:projectId/notifications', data: { helpTopic: 'notifications' }, loadComponent: () => import('./features/notifications/notifications.page').then(m => m.NotificationsPage) },
  { path: 'projects/:projectId/runs/:runId', data: { helpTopic: 'run' }, loadComponent: () => import('./features/runs/run-detail.page').then(m => m.RunDetailPage) },
  { path: '**', redirectTo: 'projects' }
]
