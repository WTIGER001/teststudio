import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { RouterLink } from '@angular/router'
import { WorkspaceService } from './workspace.service'

@Component({
  selector: 'ts-scope-context', standalone: true, imports: [RouterLink], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `@if (workspace.projectId()) {
    <nav class="scope-context" aria-label="Current scope">
      <div><span class="eyebrow">Project</span><a [routerLink]="['/projects', workspace.projectId(), 'applications']">{{ workspace.project()?.name || workspace.projectId() }}</a></div>
      <span aria-hidden="true">›</span>
      <div><span class="eyebrow">{{ workspace.applicationId() ? 'Active application' : 'Project scope' }}</span>
        @if (workspace.applicationId()) { <a [routerLink]="['/projects', workspace.projectId(), 'applications', workspace.applicationId()]">{{ workspace.application()?.name || workspace.applicationId() }}</a> }
        @else { <strong>All applications</strong> }
      </div>
      <p>{{ workspace.applicationId() ? 'Suites, tests, runs, and trends follow this application.' : 'Choose an application for its suites and test catalog.' }}</p>
    </nav>
  }`,
  styles: [`:host { display: block; } .scope-context { display: flex; align-items: center; gap: 16px; border-bottom: 1px solid var(--border); padding-bottom: 16px; margin-bottom: 24px; flex-wrap: wrap; } .scope-context div { display: grid; gap: 4px; } a, strong { font-size: 14px; font-weight: 600; } p { margin: 0 0 0 auto; color: var(--muted); font-size: 12px; } @media(max-width: 700px) { p { flex-basis: 100%; margin: 0; } }`]
})
export class ScopeContextComponent { readonly workspace = inject(WorkspaceService) }
