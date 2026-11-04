import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { ProjectSetupComponent } from './project-setup.component'
import { RecordCreationService } from '../../core/record-creation.service'
import { RouterLink } from '@angular/router'
import { ProjectsService } from './projects.service'

@Component({
  standalone: true,
  imports: [ProjectSetupComponent, RouterLink],
  providers: [RecordCreationService, ProjectsService],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="page-heading"><div><span class="eyebrow">Your workspace</span><h1>Projects</h1><p class="muted">A home for every application, suite, and result.</p></div><span class="count-label">{{ service.workspace.projects().length }} projects</span></div>
    <div class="two-column">
      <section aria-labelledby="projects-title">
        <h2 id="projects-title" class="section-heading">Your projects</h2>
        <div class="project-grid">
          @for (project of service.workspace.projects(); track project.id) {
            <a class="project-card" [routerLink]="['/projects', project.id, 'applications']"><span class="project-icon" aria-hidden="true">▦</span><h3>{{ project.data.name }}</h3><p class="muted">{{ project.data.description || 'Choose an application and start testing.' }}</p><span class="small">Open project <span aria-hidden="true">→</span></span></a>
          } @empty {
            <div class="empty-state"><span class="empty-icon" aria-hidden="true">▦</span>@if (service.workspace.can('projects.create')) { <h3>A fresh start.</h3><p>Create your first project to organize applications and build your test catalog.</p> } @else { <h3>Access not granted</h3><p>Sign-in succeeded. A project owner can add your account and assign access.</p> }</div>
          }
        </div>
      </section>
      @if (service.workspace.can('projects.create')) {
        <section class="panel" aria-labelledby="new-project-title">
          <h2 id="new-project-title">Create a project</h2><p class="small muted">Keep related applications and test suites together.</p>
          <ts-project-setup />
        </section>
      }
    </div>
  `
})
export class ProjectsPage { readonly service = inject(ProjectsService); canLeave(): boolean { return this.service.canLeave() } }
