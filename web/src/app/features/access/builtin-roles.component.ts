import { DatePipe } from '@angular/common'
import { ChangeDetectionStrategy, Component, inject, input } from '@angular/core'
import type { PermissionChoice } from '../../core/generated/access.types'
import { AccessActionComponent } from './access-action.component'
import { AccessActionService } from './access-action.service'
import { BuiltinRolesService } from './builtin-roles.service'
import { permissionDescription } from './role-policy.utils'

@Component({
  selector: 'ts-builtin-roles', standalone: true, imports: [DatePipe, AccessActionComponent], providers: [BuiltinRolesService, AccessActionService], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="panel" aria-labelledby="builtin-roles-heading"><div class="section-top"><h2 id="builtin-roles-heading" tabindex="-1">Installation-wide predefined roles</h2><button type="button" class="button secondary" [disabled]="service.records.loading() || service.action.busy()" (click)="service.refresh()">Refresh predefined roles</button></div>
      <p class="small muted">Definitions stay code managed. Disabled state applies across the entire installation. Project assignments remain retained and can grant access again after enabling.</p>
      <ts-access-action />
      @if (service.error()) { <p class="notice error" role="alert">{{ service.error() }}</p> }
      @if (service.records.error()) { <p class="notice error" role="alert">{{ service.records.error() }}</p> }
      @if (service.records.loading()) { <p class="small" role="status">Loading predefined roles…</p> }
      <div class="definitions">@for (row of service.records.items(); track row.id) { <section class="definition" [attr.aria-label]="row.data.name"><div class="section-top"><h3>{{ row.data.name }}</h3><span class="badge" [class.warning]="row.data.disabled">{{ row.data.disabled ? 'Disabled' : 'Enabled' }}</span></div><code class="small wrap-anywhere">{{ row.id }}</code><p class="small muted">{{ row.data.scope === 'project' ? 'Reusable project role · state applies to all projects' : 'System administration role' }} · Revision {{ row.revision }}</p>
        @if (row.updatedAt) { <p class="small muted">Last managed change: <time [attr.datetime]="row.updatedAt">{{ row.updatedAt | date:'medium':'UTC' }} UTC</time></p> } @else { <p class="small muted">No managed state history</p> }
        <details><summary>View permissions for {{ row.data.name }}</summary><ul>@for (permission of row.data.permissions; track permission) { <li><code class="small wrap-anywhere">{{ permission }}</code><p class="small muted">{{ describe(permissions(), permission) }}</p></li> }</ul></details>
        @if (row.data.canManage) { <button type="button" class="button secondary state-button" [disabled]="service.action.busy() || service.records.loading() || !!service.records.error()" (click)="service.state(row)">{{ row.data.disabled ? 'Enable predefined role' : 'Disable predefined role' }}</button> } @else { <p class="small muted">Read only. State changes require installation administration and authority for the role's default actions.</p> }
      </section> } @empty { @if (!service.records.loading() && !service.records.error() && !service.error()) { <p class="small muted">No readable predefined roles.</p> } }</div>
    </section>
  `,
  styles: `.definitions { display: grid; gap: 14px; margin-top: 18px; } .definition { padding: 16px; border: 1px solid var(--border); border-radius: 8px; min-width: 0; } h3 { margin: 0; } details { margin-top: 12px; } summary { cursor: pointer; } details ul { padding-left: 20px; margin-top: 14px; } details li { margin: 10px 0; } .state-button { margin-top: 14px; } .section-top { flex-wrap: wrap; }`
})
export class BuiltinRolesComponent { readonly service = inject(BuiltinRolesService); readonly permissions = input.required<PermissionChoice[]>(); readonly describe = permissionDescription }
