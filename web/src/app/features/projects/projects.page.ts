import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { RouterLink } from '@angular/router'
import { WorkspaceService } from '../../core/workspace.service'

@Component({
  standalone: true,
  imports: [RouterLink],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="page-heading"><div><span class="eyebrow">Your workspace</span><h1>Projects</h1><p class="muted">A home for every application, suite, and result.</p></div><div class="row-actions"><span class="count-label">{{ workspace.projects().length }} {{ workspace.projects().length === 1 ? 'project' : 'projects' }}</span>@if (workspace.canInstallation('projects.create')) { <a class="button primary" routerLink="/projects/new">Create project</a> }</div></div>
      <section aria-labelledby="projects-title">
        <h2 id="projects-title" class="section-heading">Your projects</h2>
        <div class="project-grid">
          @for (project of workspace.projects(); track project.id) {
            <a class="project-card" [routerLink]="['/projects', project.id, 'applications']"><span class="project-icon" aria-hidden="true">▦</span><h3>{{ project.data.name }}</h3><p class="muted">{{ project.data.description || 'Choose an application and start testing.' }}</p><span class="small">Open project <span aria-hidden="true">→</span></span></a>
          } @empty {
            <div class="empty-state"><span class="empty-icon" aria-hidden="true">▦</span>@if (workspace.canInstallation('projects.create')) { <h3>A fresh start.</h3><p>Create your first project to organize applications and build your test catalog.</p> } @else { <h3>Access not granted</h3><p>Sign-in succeeded. A project owner can add your account and assign access.</p> }</div>
          }
        </div>
      </section>
  `
})
export class ProjectsPage { readonly workspace = inject(WorkspaceService) }
