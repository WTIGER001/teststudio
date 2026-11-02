import type { Routes } from '@angular/router'

export const routes: Routes = [
  { path: '', pathMatch: 'full', redirectTo: 'projects' },
  { path: 'projects', loadComponent: () => import('./features/projects/projects.page').then(m => m.ProjectsPage) },
  { path: 'projects/:projectId/catalog', loadComponent: () => import('./features/catalog/catalog.page').then(m => m.CatalogPage) },
  { path: 'projects/:projectId/configuration', loadComponent: () => import('./features/configuration/configuration.page').then(m => m.ConfigurationPage) },
  { path: '**', redirectTo: 'projects' }
]
