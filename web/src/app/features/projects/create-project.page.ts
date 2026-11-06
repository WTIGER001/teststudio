import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { RouterLink } from '@angular/router'
import { RecordCreationService } from '../../core/record-creation.service'
import { ProjectSetupComponent } from './project-setup.component'
import { ProjectsService } from './projects.service'

@Component({
  standalone: true,
  imports: [RouterLink, ProjectSetupComponent],
  providers: [RecordCreationService, ProjectsService],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="creation-page">
      <div class="page-heading"><div><span class="eyebrow">Your workspace</span><h1>Create project</h1><p class="muted">Keep related applications and test suites together.</p></div><a class="button secondary" routerLink="/projects">Back to projects</a></div>
      @if (service.workspace.canInstallation('projects.create')) {
        <section class="panel" aria-label="Project setup"><ts-project-setup /></section>
        <a class="cancel-link" routerLink="/projects">Cancel setup</a>
      } @else {
        <p class="notice" role="status">You need project creation permission to create a project. Ask an installation administrator for access.</p>
      }
    </div>
  `,
  styles: [`:host { display: block; min-width: 0; } .creation-page { max-width: 56rem; margin-inline: auto; } .panel { min-width: 0; } .cancel-link { display: inline-block; margin-top: 1rem; }`]
})
export class CreateProjectPage {
  readonly service = inject(ProjectsService)
  canLeave(): boolean { return this.service.canLeave() }
}
