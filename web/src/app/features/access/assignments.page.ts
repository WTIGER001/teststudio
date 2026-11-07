import { RoleConversionService } from './role-conversion.service'
import { RoleConversionComponent } from './role-conversion.component'
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
  standalone: true, imports: [RoleConversionComponent, AssignmentEditorComponent, AssignmentListComponent, AssignmentRolesComponent, AccessActionComponent, PendingAccessComponent], providers: [RoleConversionService, AssignmentsService, AssignmentEditorService, AccessActionService], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="page-heading access-heading"><div><span class="eyebrow wrap-anywhere">{{ service.scopeLabel() }}</span><h1 id="assignments-heading" tabindex="-1">{{ service.applicationId() ? 'Application members' : service.projectId() ? 'Project members' : 'System access' }}</h1><p class="muted">{{ service.applicationId() ? 'Manage direct roles for this application. Project and group roles can add inherited access.' : 'Assign existing users and groups to roles in the displayed scope.' }}</p></div><div class="row-actions wrap-actions"><button type="button" class="button secondary" [disabled]="service.loading() || editor.busy() || conversion.busy() || service.action.busy()" (click)="service.refresh()">Refresh access</button><button type="button" class="button secondary" (click)="service.inspect()">Explain my access</button>@if (service.canAdd()) { <button type="button" class="button primary" [disabled]="editor.busy() || service.action.busy() || conversion.busy()" (click)="add()">+ Add role</button> }</div></div>
    @if (service.ownership(); as ownership) { <p class="notice" [class.warning]="ownership.state === 'owner-unavailable'" role="status">Ownership: {{ ownership.state === 'owned' ? 'A usable Owner manages this scope.' : ownership.state === 'system-managed' ? 'System-managed. A System Admin can assign the first Owner.' : 'No usable Owner is available. Ask a System Admin to restore access.' }}</p> }

    @if (service.message()) { <p class="notice success" role="status">{{ service.message() }}</p> }
    @if (service.error()) { <p class="notice error" role="alert">{{ service.error() }}</p> }
    @if (editor.opened()) { <ts-assignment-editor /> }
    <ts-access-action /><ts-role-conversion /><ts-assignment-list />@if (service.projectId() && !service.applicationId() && service.rights().read) { <ts-pending-access /> }@if (service.applicationId() && service.rights().read) {
      <section class="panel" aria-labelledby="inherited-heading"><h2 id="inherited-heading">Inherited project access</h2><p class="small muted">Project roles apply to every application. Change them in Project members. A lower direct application role does not reduce inherited access. Use Explain my access or Inspect access to include group and custom paths.</p>
        <ul>@for (item of service.inherited.items(); track item.id) { <li>{{ item.data.principalName }} · {{ item.data.roleName }} · From project · Read only here</li> }</ul>
        @if (service.inherited.cursor()) { <button type="button" class="button secondary" [disabled]="service.inherited.loading()" (click)="service.loadInherited(true)">More inherited assignments</button> }
        @if (service.inherited.error()) { <p class="notice error" role="alert">{{ service.inherited.error() }}</p> }
      </section>
    }<ts-assignment-roles />
  `,
  styleUrl: './access-detail.css'
})
export class AssignmentsPage { readonly service = inject(AssignmentsService); readonly editor = inject(AssignmentEditorService); readonly conversion = inject(RoleConversionService)
  add(): void { if (this.conversion.principal()) this.conversion.close(); if (!this.conversion.principal()) this.editor.open() }
  canLeave(): boolean { return !(this.editor.opened() && this.editor.form.dirty || this.conversion.dirty() || this.service.action.pending() && this.service.action.reason().trim()) || window.confirm('This membership page has unsaved changes. Leave without saving?') }
}
