import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { FormsModule } from '@angular/forms'
import { RouterLink } from '@angular/router'
import { AccessActionService } from './access-action.service'
import { GroupsService } from './groups.service'
import { GroupEditorService } from './group-editor.service'
import { GroupEditorComponent } from './group-editor.component'
import { groupRoute } from './access.utils'

@Component({
  standalone: true, imports: [FormsModule, RouterLink, GroupEditorComponent], providers: [GroupsService, GroupEditorService, AccessActionService], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="page-heading"><div><span class="eyebrow wrap-anywhere">{{ service.projectId() ? 'Project · ' + service.projectId() : 'System administration' }}</span><h1 id="groups-heading" tabindex="-1">{{ service.projectId() ? 'Project groups' : 'System groups' }}</h1><p class="muted">Manage groups within their permanent ownership scope.</p></div>@if (service.rights().manageGroups) { <button type="button" class="button primary" (click)="editor.open()">+ New group</button> }</div>
    @if (editor.opened()) { <ts-group-editor /> }
    @if (service.message()) { <p class="notice success" role="status">{{ service.message() }}</p> }
    @if (service.error()) { <p class="notice error" role="alert">{{ service.error() }}</p> }
    <section class="panel" aria-labelledby="group-directory-heading"><div class="section-top"><h2 id="group-directory-heading">Group directory</h2><button type="button" class="button secondary" [disabled]="service.groups.loading() || service.loading() || !service.rights().readGroups" (click)="service.loadGroups()">Refresh groups</button></div>
      <label class="checkbox-field"><input type="checkbox" [ngModel]="service.archived()" (ngModelChange)="service.archived.set($event); service.loadGroups()" [disabled]="!service.rights().readGroups"> Include archived groups</label>
      @if (service.groups.error()) { <p class="notice error" role="alert">{{ service.groups.error() }}</p> }
      <div class="table-scroll"><table><caption class="sr-only">Groups in the selected ownership scope</caption><thead><tr><th scope="col">Group</th><th scope="col">State</th><th scope="col">Membership source</th><th scope="col">Description</th></tr></thead><tbody>
        @for (item of service.groups.items(); track item.id) { <tr><td><a [routerLink]="route(item.data)">{{ item.data.name }}</a><code class="table-subtitle wrap-anywhere">{{ item.id }}</code></td><td><span class="badge" [class.warning]="item.data.archived">{{ item.data.archived ? 'Archived' : 'Active' }}</span></td><td class="wrap-anywhere">{{ item.data.managedBy || 'Local' }}<span class="table-subtitle muted">{{ item.data.managedBy ? 'External membership is read-only' : 'Local membership can be managed' }}</span></td><td class="wrap-anywhere">{{ item.data.description || '—' }}</td></tr> }
        @empty { @if (!service.groups.loading() && !service.loading() && !service.groups.error() && !service.error()) { <tr><td colspan="4">No readable groups in this page. Create a group or include archived groups.</td></tr> } }
      </tbody></table></div>
      @if (service.loading() || service.groups.loading()) { <p class="small" role="status">Loading groups…</p> }
      @if (service.groups.partial()) { <p class="small muted">This page covers a bounded scan. More readable groups may appear on later pages.</p> }
      @if (service.groups.cursor()) { <button type="button" class="button secondary more-button" [disabled]="service.groups.loading()" (click)="service.loadGroups(true)">More groups</button> }
    </section>
  `
})
export class GroupsPage { readonly service = inject(GroupsService); readonly editor = inject(GroupEditorService); readonly route = groupRoute }
