import { ChangeDetectionStrategy, Component, input } from '@angular/core'
import type { PermissionChoice, Role } from '../../core/generated/access.types'
import { permissionDescription } from './role-policy.utils'

@Component({
  selector: 'ts-role-definitions', standalone: true, changeDetection: ChangeDetectionStrategy.OnPush,
  template: `<section class="panel" aria-labelledby="predefined-roles-heading"><h2 id="predefined-roles-heading">Predefined roles</h2><p class="small muted">These definitions are managed by TestStudio. Use custom roles for editable policies.</p><ul class="definitions">@for (role of roles(); track role.id) { <li><div class="section-top"><h3>{{ role.name }}</h3><span class="badge">{{ role.disabled ? 'Disabled' : 'Code managed' }}</span></div><code class="small wrap-anywhere">{{ role.id }}</code><details><summary>View permissions for {{ role.name }}</summary><ul>@for (permission of role.permissions; track permission) { <li><code class="small wrap-anywhere">{{ permission }}</code><p class="small muted">{{ describe(permissions(), permission) }}</p></li> }</ul></details></li> }</ul></section>`,
  styles: `.definitions { list-style: none; padding: 0; display: grid; gap: 14px; margin-top: 18px; } .definitions > li { padding: 16px; border: 1px solid var(--border); border-radius: 8px; min-width: 0; } h3 { margin: 0; } details { margin-top: 12px; } summary { cursor: pointer; } details ul { padding-left: 20px; margin-top: 14px; } details li { margin: 10px 0; }`
})
export class RoleDefinitionsComponent { readonly roles = input.required<Role[]>(); readonly permissions = input.required<PermissionChoice[]>(); readonly describe = permissionDescription }
