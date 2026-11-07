import { DatePipe } from '@angular/common'
import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { AccessActionComponent } from './access-action.component'
import { AccessActionService } from './access-action.service'
import { GrantEditorComponent } from './grant-editor.component'
import { GrantEditorService } from './grant-editor.service'
import { GrantsService } from './grants.service'
import { RolePermissionCatalog } from './role-permissions.service'

@Component({
  standalone: true, imports: [DatePipe, AccessActionComponent, GrantEditorComponent], providers: [GrantsService, GrantEditorService, AccessActionService, RolePermissionCatalog], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="page-heading"><div><span class="eyebrow wrap-anywhere">{{ service.projectId() ? 'Project · ' + service.projectId() : 'System administration' }}</span><h1 id="grants-heading" tabindex="-1">{{ service.projectId() ? 'Project permissions' : 'System permissions' }}</h1><p class="muted">Manage direct allow and deny rules for registered users and groups.</p></div><div class="row-actions wrap-actions"><button type="button" class="button secondary" [disabled]="service.loading() || editor.busy() || service.action.busy()" (click)="service.refresh()">Refresh permissions</button>@if (service.canCreate()) { <button type="button" class="button primary" [disabled]="editor.busy() || service.action.busy()" (click)="editor.open()">+ Add direct grant</button> }</div></div>
    @if (editor.opened()) { <ts-grant-editor /> }
    <ts-access-action />
    @if (service.message()) { <p class="notice success" role="status">{{ service.message() }}</p> }
    @if (service.error()) { <p class="notice error" role="alert">{{ service.error() }}</p> }
    <section class="panel" aria-labelledby="managed-grants-heading"><h2 id="managed-grants-heading">Managed direct grants</h2><p class="small muted">These rules apply alongside role assignments and inherited access. Expired rules remain visible. This list contains grants managed through TestStudio.</p>
      @if (service.grants.error()) { <p class="notice error" role="alert">{{ service.grants.error() }}</p> }
      <div class="table-scroll" role="region" aria-label="Managed direct grants" tabindex="0"><table><caption class="sr-only">Managed direct grants</caption><thead><tr><th scope="col">Principal</th><th scope="col">Permission and exact object</th><th scope="col">Effect</th><th scope="col">Expiry</th><th scope="col">Changed</th><th scope="col">Actions</th></tr></thead><tbody>
        @for (row of service.grants.items(); track row.id) { <tr><th scope="row"><span>{{ row.data.principalName }}</span><code class="table-subtitle wrap-anywhere">{{ row.data.principalKind }}/{{ row.data.principalId }}</code><span class="table-subtitle muted">{{ row.data.principalActive ? 'Active principal' : 'Inactive principal' }}</span></th><td><code class="wrap-anywhere">{{ row.data.permission }}</code><code class="table-subtitle wrap-anywhere">{{ row.data.object || 'Whole scope' }}</code><details class="small"><summary>Grant identity</summary><code class="wrap-anywhere">{{ row.id }}</code><span class="table-subtitle">Revision {{ row.revision }}</span></details></td><td><span class="badge" [class.warning]="row.data.effect === 'deny'">{{ row.data.effect === 'deny' ? 'Deny' : 'Allow' }}</span></td><td>@if (row.data.expiresAt) { <time class="wrap-anywhere small" [attr.datetime]="row.data.expiresAt">{{ row.data.expiresAt }}</time><span class="table-subtitle">{{ row.data.expired ? 'Expired · no access' : 'Not expired' }}</span> } @else { <span>No expiry</span> }</td><td><time class="small" [attr.datetime]="row.updatedAt">{{ row.updatedAt | date:'medium':'UTC' }} UTC</time></td><td><div class="row-actions wrap-actions">@if (row.data.canUpdate) { <button type="button" class="button secondary" [disabled]="editor.busy() || service.action.busy()" (click)="editor.open(row)">Edit expiry</button> }@if (row.data.canRemove) { <button type="button" class="button subtle" [disabled]="editor.busy() || service.action.busy()" (click)="editor.remove(row)">Remove grant</button> }@if (!row.data.canUpdate && !row.data.canRemove) { <span class="small muted">Read only</span> }</div></td></tr> }
        @empty { @if (!service.loading() && !service.grants.loading() && !service.error() && !service.grants.error()) { <tr><td colspan="6">No readable managed grants in this page.</td></tr> } }
      </tbody></table></div>
      @if (service.loading() || service.grants.loading()) { <p class="small" role="status">Loading permissions…</p> }
      @if (service.grants.cursor()) { <p class="small muted">This page covers a bounded scan. More readable grants can appear on later pages.</p><button type="button" class="button secondary more-button" [disabled]="service.grants.loading()" (click)="service.load(true)">More grants</button> }
    </section>
  `,
  styles: `.table-scroll table { min-width: 1040px; } th[scope="row"] { min-width: 180px; } td:nth-child(2) { min-width: 270px; max-width: 340px; } td:nth-child(4) { min-width: 170px; } td:last-child { min-width: 200px; } details { margin-top: 8px; }`
})
export class GrantsPage { readonly service = inject(GrantsService); readonly editor = inject(GrantEditorService) }
