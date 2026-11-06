import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { RouterLink } from '@angular/router'
import { WorkspaceService } from '../../core/workspace.service'
import { ProjectCoverEditorComponent } from './project-cover-editor.component'
import { ProjectCoversService } from './project-covers.service'
import { ProjectImageCaptureService } from './project-image-capture.service'
import { ProjectImageProcessingService } from './project-image-processing.service'

@Component({
  standalone: true,
  imports: [RouterLink, ProjectCoverEditorComponent],
  providers: [ProjectCoversService, ProjectImageCaptureService, ProjectImageProcessingService],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="page-heading"><div><span class="eyebrow">Your workspace</span><h1>Projects</h1><p class="muted">A home for every application, suite, and result.</p></div><div class="row-actions">@if (workspace.canInstallation('projects.create')) { <a class="button primary" routerLink="/projects/new">Create project</a> }</div></div>
      @if (covers.listError()) { <p class="notice" role="status">{{ covers.listError() }} <button type="button" class="button secondary small" (click)="covers.load()">Retry loading images</button></p> }
      <section aria-labelledby="projects-title">
        <h2 id="projects-title" class="section-heading">Your projects</h2>
        <div class="project-grid">
          @for (project of workspace.projects(); track project.id) {
            <article class="project-card" [class.with-image]="covers.covers()[project.id]?.hasImage">
              @if (covers.covers()[project.id]?.hasImage) { <img class="project-cover" [src]="covers.imageURL(project.id)" alt="" loading="lazy"> }
              <a class="project-card-link" [routerLink]="['/projects', project.id, 'applications']"><span class="project-icon" aria-hidden="true">▦</span><div class="project-card-details"><h3>{{ project.data.name }}</h3><p>{{ project.data.description || 'Choose an application and start testing.' }}</p><span class="small">Open project <span aria-hidden="true">→</span></span></div></a>
              @if (covers.covers()[project.id]?.canManage) { <button type="button" class="button secondary small project-image-action" [attr.aria-label]="'Change image for ' + project.data.name" (click)="editor.open(project.id, project.data.name)">Image</button> }
            </article>
          } @empty {
            <div class="empty-state"><span class="empty-icon" aria-hidden="true">▦</span>@if (workspace.canInstallation('projects.create')) { <h3>A fresh start.</h3><p>Create your first project to organize applications and build your test catalog.</p> } @else { <h3>Access not granted</h3><p>Sign-in succeeded. A project owner can add your account and assign access.</p> }</div>
          }
        </div>
      </section>
      <ts-project-cover-editor #editor />
  `
})
export class ProjectsPage {
  readonly workspace = inject(WorkspaceService)
  readonly covers = inject(ProjectCoversService)
  canLeave(): boolean { return this.covers.canLeave() }
}
