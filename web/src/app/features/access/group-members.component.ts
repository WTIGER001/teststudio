import { DatePipe } from '@angular/common'
import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { GroupsService } from './groups.service'
import { GroupMembersService } from './group-members.service'
import { GroupMemberEditorComponent } from './group-member-editor.component'
import { canEditMembership } from './access.utils'

@Component({
  selector: 'ts-group-members', standalone: true, imports: [DatePipe, GroupMemberEditorComponent], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="panel" aria-labelledby="group-members-heading"><div class="section-top"><div><h2 id="group-members-heading" tabindex="-1">Group members</h2><p class="small muted">Local and external provenance stay distinct.</p></div><div class="row-actions wrap-actions"><button type="button" class="button secondary" [disabled]="service.members.loading() || !service.rights().readMembers" (click)="service.loadMembers()">Refresh members</button>@if (canEdit(service.group()?.data ?? null, service.rights().manageMembers)) { <button type="button" class="button primary" [disabled]="editor.busy() || service.action.busy()" (click)="editor.open()">+ Add member</button> }</div></div>
      @if (service.group()?.data?.managedBy) { <p class="notice">External membership is read-only. Update the source provider to change membership.</p> }
      @if (service.group()?.data?.archived) { <p class="notice warning">This group is archived. Retained membership does not grant effective access.</p> }
      @if (editor.opened() && canEdit(service.group()?.data ?? null, service.rights().manageMembers)) { <ts-group-member-editor /> }
      @if (service.members.error()) { <p class="notice error" role="alert">{{ service.members.error() }}</p> }
      <div class="table-scroll"><table><caption class="sr-only">Group membership records</caption><thead><tr><th scope="col">Member</th><th scope="col">Source</th><th scope="col">Activity / expiry</th><th scope="col">Actions</th></tr></thead><tbody>
        @for (item of service.members.items(); track item.id) { <tr><td><strong>{{ item.data.name }}</strong><code class="table-subtitle wrap-anywhere">{{ item.data.userId }}</code></td><td class="wrap-anywhere">{{ item.data.source }}<span class="table-subtitle muted">{{ item.data.readOnly ? 'Externally managed · Read only' : 'Locally managed' }}</span></td><td><span class="badge" [class.success]="item.data.active">{{ item.data.active ? 'Active' : 'Inactive' }}</span>@if (item.data.expiresAt) { <time class="table-subtitle muted" [attr.datetime]="item.data.expiresAt">{{ item.data.expiresAt | date:'medium':'UTC' }} UTC</time> } @else { <span class="table-subtitle muted">No expiry</span> }</td><td>@if (canEdit(service.group()?.data ?? null, service.rights().manageMembers, item.data)) { <div class="row-actions"><button type="button" class="button small subtle" [disabled]="editor.busy() || service.action.busy()" (click)="editor.open(item)">Edit membership</button><button type="button" class="button small subtle" [disabled]="editor.busy() || service.action.busy()" (click)="editor.remove(item)">Remove member</button></div> }</td></tr> }
        @empty { @if (!service.members.loading() && !service.members.error()) { <tr><td colspan="4">No readable memberships in this page.</td></tr> } }
      </tbody></table></div>
      @if (service.members.loading()) { <p class="small" role="status">Loading members…</p> }
      @if (service.members.partial()) { <p class="small muted">This page covers a bounded scan. Continue for more readable memberships.</p> }
      @if (service.members.cursor()) { <button type="button" class="button secondary more-button" [disabled]="service.members.loading()" (click)="service.loadMembers(true)">More members</button> }
    </section>
  `
})
export class GroupMembersComponent { readonly service = inject(GroupsService); readonly editor = inject(GroupMembersService); readonly canEdit = canEditMembership }
