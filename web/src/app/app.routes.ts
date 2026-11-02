import type { Routes } from '@angular/router'

export const routes: Routes = [
  { path: '', pathMatch: 'full', redirectTo: 'projects' },
  { path: 'projects', data: { helpTopic: 'projects' }, loadComponent: () => import('./features/projects/projects.page').then(m => m.ProjectsPage) },
  { path: 'projects/:projectId/catalog', data: { helpTopic: 'catalog' }, loadComponent: () => import('./features/catalog/catalog.page').then(m => m.CatalogPage) },
  { path: 'projects/:projectId/configuration', data: { helpTopic: 'configuration' }, loadComponent: () => import('./features/configuration/configuration.page').then(m => m.ConfigurationPage) },
  { path: 'projects/:projectId/runs', data: { helpTopic: 'runs' }, loadComponent: () => import('./features/runs/run-list.page').then(m => m.RunListPage) },
  { path: 'projects/:projectId/trends', data: { helpTopic: 'trends' }, loadComponent: () => import('./features/trends/trends.page').then(m => m.TrendsPage) },
  { path: 'projects/:projectId/runs/:runId', data: { helpTopic: 'run' }, loadComponent: () => import('./features/runs/run-detail.page').then(m => m.RunDetailPage) },
  { path: '**', redirectTo: 'projects' }
]
