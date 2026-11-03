import { DatePipe } from '@angular/common'
import { ChangeDetectionStrategy, Component, input } from '@angular/core'
import type { PermissionChoice, RoleRule } from '../../core/generated/access.types'
import { permissionDescription } from './role-policy.utils'

@Component({
  selector: 'ts-role-rules', standalone: true, imports: [DatePipe], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `<div class="table-scroll" tabindex="0" role="region" aria-label="Scrollable stored rules"><table><caption class="sr-only">Stored rules for this role</caption><thead><tr><th scope="col">Permission</th><th scope="col">Effect</th><th scope="col">Exact object</th><th scope="col">Expiry</th></tr></thead><tbody>@for (rule of rules(); track $index) { <tr><th scope="row"><code class="small wrap-anywhere">{{ rule.permission }}</code><p class="small muted">{{ describe(permissions(), rule.permission) }}</p></th><td><span class="badge" [class.warning]="rule.effect === 'deny'">{{ rule.effect === 'allow' ? 'Allow' : 'Deny' }}</span></td><td><code class="small wrap-anywhere">{{ rule.object || 'Full ownership scope' }}</code></td><td>{{ rule.expiresAt ? (rule.expiresAt | date:'medium':'UTC') + ' UTC' : 'No expiry' }}</td></tr> } @empty { <tr><td colspan="4">No direct rules. Inspect inherited roles for additional policy.</td></tr> }</tbody></table></div>`,
  styles: `table { min-width: 680px; } th:first-child { min-width: 240px; }`
})
export class RoleRulesComponent { readonly rules = input.required<RoleRule[]>(); readonly permissions = input.required<PermissionChoice[]>(); readonly describe = permissionDescription }
