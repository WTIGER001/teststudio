import type { Routes } from '@angular/router'

export const routes: Routes = [
  { path: '', pathMatch: 'full', redirectTo: 'projects' },
  { path: 'projects', loadComponent: () => import('./features/projects/projects.page').then(m => m.ProjectsPage) },
  { path: 'projects/:projectId/catalog', loadComponent: () => import('./features/catalog/catalog.page').then(m => m.CatalogPage) },
  { path: 'projects/:projectId/configuration', loadComponent: () => import('./features/configuration/configuration.page').then(m => m.ConfigurationPage) },
  { path: 'projects/:projectId/runs', loadComponent: () => import('./features/runs/run-list.page').then(m => m.RunListPage) },
  { path: 'projects/:projectId/runs/:runId', loadComponent: () => import('./features/runs/run-detail.page').then(m => m.RunDetailPage) },
  { path: '**', redirectTo: 'projects' }
]
