import { DatePipe } from '@angular/common'
import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { RouterLink } from '@angular/router'
import { AccessActionService } from './access-action.service'
import { AccessActionComponent } from './access-action.component'
import { RolePoliciesService } from './role-policies.service'
import { RoleEditorService } from './role-editor.service'
import { RolePermissionCatalog } from './role-permissions.service'
import { RolePreviewService } from './role-preview.service'
import { RoleEditorComponent } from './role-editor.component'
import { RoleRulesComponent } from './role-rules.component'
import { roleRoute } from './role-policy.utils'

@Component({
  standalone: true, imports: [DatePipe, RouterLink, AccessActionComponent, RoleEditorComponent, RoleRulesComponent], providers: [RolePoliciesService, RoleEditorService, RolePermissionCatalog, RolePreviewService, AccessActionService], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <a class="small" [routerLink]="service.listRoute()">← Roles and permissions</a>
    <div class="page-heading access-heading"><div><span class="eyebrow wrap-anywhere">{{ service.projectId() ? 'Project · ' + service.projectId() : 'Installation administration' }}</span><h1 id="role-details-heading" tabindex="-1">Role details</h1><code class="small muted wrap-anywhere">{{ service.roleId() }}</code></div><div class="row-actions wrap-actions"><button type="button" class="button secondary" [disabled]="service.loading() || editor.busy() || service.action.busy()" (click)="service.refresh()">Refresh role</button>@if (service.detail(); as row) { @if (row.canManage) { <button type="button" class="button primary" [disabled]="service.loading() || editor.busy() || service.action.busy()" (click)="editor.open(row)">Edit role policy</button><button type="button" class="button secondary" [disabled]="service.loading() || editor.opened() || service.action.busy()" (click)="service.state(!row.data.disabled)">{{ row.data.disabled ? 'Enable role' : 'Disable role' }}</button> } }</div></div>
    @if (service.error()) { <p class="notice error" role="alert">{{ service.error() }}</p> }
    @if (service.message()) { <p class="notice success" role="status">{{ service.message() }}</p> }
    @if (service.loading()) { <p class="small" role="status">Loading role details…</p> }
    @if (editor.opened()) { <ts-role-editor /> }
    <ts-access-action />
    @if (service.detail(); as row) {
      <section class="panel" aria-labelledby="role-profile-heading"><h2 id="role-profile-heading">{{ row.data.name }}</h2><dl class="detail-grid"><dt>Ownership</dt><dd>{{ row.data.projectId || 'Installation' }} · Permanent</dd><dt>State</dt><dd><span class="badge" [class.warning]="row.data.disabled">{{ row.data.disabled ? 'Disabled' : 'Enabled' }}</span></dd><dt>Assignment</dt><dd>{{ row.delegatable ? 'Can delegate to authorized principals' : 'Read only with current authority' }}</dd><dt>Description</dt><dd>{{ row.data.description || 'No description' }}</dd><dt>Last change</dt><dd>{{ row.updatedAt | date:'medium':'UTC' }} UTC</dd></dl><p class="small muted guidance">A disabled role retains its definition and assignments. Its direct and inherited access is withdrawn.</p></section>
      @if (service.permissions.error()) { <p class="notice error" role="alert">{{ service.permissions.error() }}</p> }
      <section class="panel" aria-labelledby="role-rules-heading"><h2 id="role-rules-heading">Stored permission rules</h2><ts-role-rules [rules]="row.data.rules" [permissions]="service.permissions.items()" /></section>
      <section class="panel" aria-labelledby="role-inherits-heading"><h2 id="role-inherits-heading">Inherited role policies</h2><p class="small muted">Each link uses this role's permanent ownership scope. Inspect child rules for their exact objects and effects.</p><ul class="inherited-links">@for (id of row.data.inherits; track id) { <li><a [routerLink]="route(service.projectId(), id)">{{ id }}</a></li> } @empty { <li>No inherited roles.</li> }</ul></section>
    }
  `,
  styleUrl: './access-detail.css'
})
export class RolePolicyDetailPage { readonly service = inject(RolePoliciesService); readonly editor = inject(RoleEditorService); readonly route = roleRoute }
