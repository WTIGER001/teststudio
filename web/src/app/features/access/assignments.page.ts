import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { PendingAccessComponent } from './pending-access.component'
import { AssignmentsService } from './assignments.service'
import { AssignmentEditorService } from './assignment-editor.service'
import { AssignmentEditorComponent } from './assignment-editor.component'
import { AssignmentListComponent } from './assignment-list.component'
import { AssignmentRolesComponent } from './assignment-roles.component'
import { AccessActionService } from './access-action.service'
import { AccessActionComponent } from './access-action.component'

@Component({
  standalone: true, imports: [AssignmentEditorComponent, AssignmentListComponent, AssignmentRolesComponent, AccessActionComponent, PendingAccessComponent], providers: [AssignmentsService, AssignmentEditorService, AccessActionService], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="page-heading access-heading"><div><span class="eyebrow wrap-anywhere">{{ service.projectId() ? 'Project · ' + service.projectId() : 'Installation administration' }}</span><h1 id="assignments-heading" tabindex="-1">{{ service.projectId() ? 'Project access' : 'Installation access' }}</h1><p class="muted">{{ service.projectId() ? 'Manage existing roles and prepare access before first sign-in.' : 'Review roles and manage access for registered users and groups.' }}</p></div><div class="row-actions wrap-actions"><button type="button" class="button secondary" [disabled]="service.loading() || editor.busy() || service.action.busy()" (click)="service.refresh()">Refresh access</button>@if (service.canAdd()) { <button type="button" class="button primary" [disabled]="editor.busy() || service.action.busy()" (click)="editor.open()">+ Add role</button> }</div></div>
    @if (service.message()) { <p class="notice success" role="status">{{ service.message() }}</p> }
    @if (service.error()) { <p class="notice error" role="alert">{{ service.error() }}</p> }
    @if (editor.opened()) { <ts-assignment-editor /> }
    <ts-access-action /><ts-assignment-list />@if (service.projectId() && service.rights().read) { <ts-pending-access /> }<ts-assignment-roles />
  `,
  styleUrl: './access-detail.css'
})
export class AssignmentsPage { readonly service = inject(AssignmentsService); readonly editor = inject(AssignmentEditorService) }
