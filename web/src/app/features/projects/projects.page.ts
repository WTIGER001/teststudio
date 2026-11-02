import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { ReactiveFormsModule } from '@angular/forms'
import { RouterLink } from '@angular/router'
import { ProjectsService } from './projects.service'

@Component({
  standalone: true,
  imports: [ReactiveFormsModule, RouterLink],
  providers: [ProjectsService],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="page-heading"><div><span class="eyebrow">Your workspace</span><h1>Projects</h1><p class="muted">A home for every application, suite, and result.</p></div><span class="count-label">{{ service.workspace.projects().length }} projects</span></div>
    <div class="two-column">
      <section aria-labelledby="projects-title">
        <h2 id="projects-title" class="section-heading">Your projects</h2>
        <div class="project-grid">
          @for (project of service.workspace.projects(); track project.id) {
            <a class="project-card" [routerLink]="['/projects', project.id, 'catalog']"><span class="project-icon" aria-hidden="true">▦</span><h3>{{ project.data.name }}</h3><p class="muted">{{ project.data.description || 'Open the catalog and manage this project.' }}</p><span class="small">Open project <span aria-hidden="true">→</span></span></a>
          } @empty {
            <div class="empty-state"><span class="empty-icon" aria-hidden="true">▦</span><h3>A fresh start.</h3><p>Create your first project to organize applications and build your test catalog.</p></div>
          }
        </div>
      </section>
      @if (service.workspace.can('projects.create')) {
        <section class="panel" aria-labelledby="new-project-title">
          <h2 id="new-project-title">Create a project</h2><p class="small muted">Keep related applications and test suites together.</p>
          <form [formGroup]="service.form" (ngSubmit)="service.create()">
            <div class="field"><label for="project-name">Project name</label><input id="project-name" formControlName="name" required placeholder="My application" autocomplete="off"></div>
            <div class="field"><label for="project-id">Permanent identifier</label><input id="project-id" formControlName="id" required placeholder="my-application" pattern="[A-Za-z0-9](?:[A-Za-z0-9._]|-){0,127}" autocomplete="off"><small>Use letters, numbers, periods, underscores, or hyphens.</small></div>
            <div class="field"><label for="project-description">Description <span class="muted">(optional)</span></label><textarea id="project-description" formControlName="description" rows="3"></textarea></div>
            @if (service.error()) { <p class="notice error" role="alert">{{ service.error() }}</p> }
            <button class="button primary" type="submit" [disabled]="service.saving() || service.form.invalid">{{ service.saving() ? 'Creating…' : 'Create project' }}</button>
          </form>
        </section>
      }
    </div>
  `
})
export class ProjectsPage { readonly service = inject(ProjectsService) }
