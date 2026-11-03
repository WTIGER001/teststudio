import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { RouterLink } from '@angular/router'
import { AccessActionService } from './access-action.service'
import { AccessActionComponent } from './access-action.component'
import { GroupsService } from './groups.service'
import { GroupEditorService } from './group-editor.service'
import { GroupEditorComponent } from './group-editor.component'
import { GroupMembersService } from './group-members.service'
import { GroupMembersComponent } from './group-members.component'

@Component({
  standalone: true, imports: [RouterLink, AccessActionComponent, GroupEditorComponent, GroupMembersComponent], providers: [GroupsService, GroupEditorService, GroupMembersService, AccessActionService], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <a class="small" [routerLink]="service.listRoute()">← Group directory</a>
    <div class="page-heading access-heading"><div><span class="eyebrow">{{ service.projectId() ? 'Project group' : 'Installation group' }}</span><h1 id="group-details-heading" tabindex="-1">Group details</h1><code class="small muted wrap-anywhere">{{ service.groupId() }}</code></div><div class="row-actions wrap-actions"><button type="button" class="button secondary" [disabled]="service.loading()" (click)="service.refreshDetails()">Refresh group</button>@if (service.group(); as item) { @if (service.rights().manageGroups) { @if (!item.data.archived) { <button type="button" class="button primary" [disabled]="editor.busy() || members.busy() || service.action.busy()" (click)="editor.open(item)">Edit group</button> }<button type="button" class="button secondary" [disabled]="editor.busy() || members.busy() || service.action.busy()" (click)="service.state(!item.data.archived)">{{ item.data.archived ? 'Restore group' : 'Archive group' }}</button> } }</div></div>
    @if (editor.opened()) { <ts-group-editor /> }
    <ts-access-action />
    @if (service.message()) { <p class="notice success" role="status">{{ service.message() }}</p> }
    @if (service.error()) { <p class="notice error" role="alert">{{ service.error() }}</p> }
    @if (service.loading()) { <p class="small" role="status">Loading group details…</p> }
    @if (service.group(); as item) {
      <section class="panel" aria-labelledby="group-profile-heading"><h2 id="group-profile-heading">{{ item.data.name }}</h2><dl class="detail-grid"><dt>Ownership</dt><dd>{{ item.data.projectId || 'Installation' }} · Permanent</dd><dt>State</dt><dd><span class="badge" [class.warning]="item.data.archived">{{ item.data.archived ? 'Archived' : 'Active' }}</span></dd><dt>Membership source</dt><dd>{{ item.data.managedBy || 'Local' }}{{ item.data.managedBy ? ' · Externally managed' : '' }}</dd><dt>Description</dt><dd>{{ item.data.description || 'No description' }}</dd></dl></section>
      <ts-group-members />
    }
  `,
  styleUrl: './access-detail.css'
})
export class GroupDetailPage { readonly service = inject(GroupsService); readonly editor = inject(GroupEditorService); readonly members = inject(GroupMembersService) }
