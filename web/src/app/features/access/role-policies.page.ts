import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { RouterLink } from '@angular/router'
import { AccessActionService } from './access-action.service'
import { RolePoliciesService } from './role-policies.service'
import { RoleEditorService } from './role-editor.service'
import { RolePermissionCatalog } from './role-permissions.service'
import { RolePreviewService } from './role-preview.service'
import { RoleEditorComponent } from './role-editor.component'
import { RoleDefinitionsComponent } from './role-definitions.component'
import { roleRoute } from './role-policy.utils'

@Component({
  standalone: true, imports: [RouterLink, RoleEditorComponent, RoleDefinitionsComponent], providers: [RolePoliciesService, RoleEditorService, RolePermissionCatalog, RolePreviewService, AccessActionService], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="page-heading access-heading"><div><span class="eyebrow wrap-anywhere">{{ service.projectId() ? 'Project · ' + service.projectId() : 'Installation administration' }}</span><h1 id="roles-heading" tabindex="-1">Roles and permissions</h1><p class="muted">Manage exact scoped policies and inspect predefined definitions.</p></div><div class="row-actions wrap-actions"><button type="button" class="button secondary" [disabled]="service.loading() || editor.busy()" (click)="service.refresh()">Refresh roles</button>@if (service.canCreate()) { <button type="button" class="button primary" [disabled]="service.loading() || editor.busy()" (click)="editor.open()">+ Create role</button> }</div></div>
    @if (service.error()) { <p class="notice error" role="alert">{{ service.error() }}</p> }
    @if (service.message()) { <p class="notice success" role="status">{{ service.message() }}</p> }
    @if (editor.opened()) { <ts-role-editor /> }
    @if (service.rights().read) {
      <section class="panel" aria-labelledby="custom-roles-heading"><h2 id="custom-roles-heading">Custom roles</h2><p class="small muted">One permanent ownership scope per role. Open a role to inspect exact objects, allow and deny rules, and inheritance.</p>
        @if (service.policies.error()) { <p class="notice error" role="alert">{{ service.policies.error() }}</p> }
        @if (service.policies.loading()) { <p class="small" role="status">Loading custom roles…</p> }
        <div class="table-scroll" tabindex="0" role="region" aria-label="Custom role catalog"><table><caption class="sr-only">Managed custom role catalog</caption><thead><tr><th scope="col">Role</th><th scope="col">State</th><th scope="col">Direct rules</th><th scope="col">Inherited roles</th></tr></thead><tbody>@for (row of service.policies.items(); track row.id) { <tr><th scope="row"><a [routerLink]="route(service.projectId(), row.id)">{{ row.data.name }}</a><p><code class="small muted wrap-anywhere">{{ row.id }}</code></p></th><td><span class="badge" [class.warning]="row.data.disabled">{{ row.data.disabled ? 'Disabled' : 'Enabled' }}</span></td><td>{{ row.data.rules.length }}</td><td>{{ row.data.inherits.length }}</td></tr> } @empty { <tr><td colspan="4">No custom roles in this page.</td></tr> }</tbody></table></div>
        @if (service.policies.cursor()) { <button type="button" class="button secondary more-button" [disabled]="service.policies.loading()" (click)="service.loadPolicies(true)">More custom roles</button> }
      </section>
      @if (service.permissions.error()) { <p class="notice error" role="alert">{{ service.permissions.error() }}</p> }
      @if (service.builtin.error()) { <p class="notice error" role="alert">{{ service.builtin.error() }}</p> }
      <ts-role-definitions [roles]="service.builtin.items()" [permissions]="service.permissions.items()" />
    }
  `,
  styleUrl: './access-detail.css'
})
export class RolePoliciesPage { readonly service = inject(RolePoliciesService); readonly editor = inject(RoleEditorService); readonly route = roleRoute }
