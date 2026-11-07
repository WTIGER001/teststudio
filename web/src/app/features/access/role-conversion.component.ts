import { DatePipe } from '@angular/common'
import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { ReactiveFormsModule } from '@angular/forms'
import { RoleConversionService } from './role-conversion.service'

@Component({
  selector: 'ts-role-conversion', standalone: true, imports: [DatePipe, ReactiveFormsModule], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    @if (service.principal(); as row) {
      <section class="panel" aria-labelledby="conversion-heading"><div class="section-top"><h2 id="conversion-heading" tabindex="-1">Review role conversion</h2><button type="button" class="button subtle" [disabled]="service.busy()" (click)="service.close()">Close conversion</button></div>
        <p>{{ row.data.principalName }} · {{ service.access.scopeLabel() }}</p><p class="small muted">Replace this principal's predefined role assignments with one standard role. Custom roles, direct grants, and group membership remain. Nothing changes until you apply the review.</p>
        @if (!service.receipt()?.data?.applied) {
          <form [formGroup]="service.form" (ngSubmit)="service.preview()"><fieldset [disabled]="service.busy() || service.uncertain()"><legend class="sr-only">Role conversion</legend>
            <div class="form-grid"><div class="field"><label for="conversion-role">Target standard role</label><select id="conversion-role" formControlName="roleId"><option value="">Select a role</option>@for (role of service.access.standardChoices(); track role.id) { <option [value]="role.id" [disabled]="!role.delegatable || role.disabled">{{ role.name }}</option> }</select></div><div class="field"><label for="conversion-reason">Conversion reason</label><textarea id="conversion-reason" formControlName="reason" maxlength="1024" required></textarea></div></div>
            <button type="submit" class="button secondary">Preview conversion</button>
          </fieldset></form>
        }
        @if (service.error()) { <p class="notice error" role="alert">{{ service.error() }}</p> }
        @if (service.receipt(); as receipt) {
          <h3 id="conversion-review-heading" tabindex="-1">{{ receipt.data.applied ? 'Conversion applied' : 'Conversion review' }}</h3>
          <p>Target roles: {{ service.roleNames(receipt.data.targetRoleIds) }}. Review expires {{ receipt.data.expiresAt | date:'medium':'UTC' }} UTC.</p>
          <details><summary>Assignments replaced</summary><ul>@for (item of receipt.data.before; track item.id) { <li>{{ item.data.roleName }} · {{ item.data.principalName }}</li> }</ul></details>
          <div class="form-grid"><section><h4>Permissions added</h4><ul>@for (permission of receipt.data.addedPermissions; track permission) { <li class="wrap-anywhere">{{ permission }}</li> } @empty { <li>None</li> }</ul></section><section><h4>Permissions removed</h4><ul>@for (permission of receipt.data.removedPermissions; track permission) { <li class="wrap-anywhere">{{ permission }}</li> } @empty { <li>None</li> }</ul></section></div>
          <p class="small muted">These are role-definition changes. Current denies, other roles, and groups still determine effective access.</p>
          @if (!receipt.data.applied) { <button type="button" class="button primary" [disabled]="service.busy() || service.uncertain()" (click)="service.apply()">Apply reviewed conversion</button> }
          @else { <p class="notice success" role="status">Role conversion saved. Repeating the request uses this receipt.</p><details><summary>Advanced recovery</summary><div class="field"><label for="rollback-reason">Rollback reason</label><textarea id="rollback-reason" [formControl]="service.rollbackReason" maxlength="1024"></textarea></div><p class="small muted">Prepare a new review restoring the original role set. Current authority and usable-Owner safeguards still apply.</p><button type="button" class="button secondary" [disabled]="service.busy()" (click)="service.rollback()">Preview rollback</button></details> }
          <button type="button" class="button secondary" [disabled]="service.busy()" (click)="service.reload()">Reload conversion receipt</button>
        }
        @if (service.uncertain()) { <p class="notice warning">The save could not be confirmed. Reload the receipt before another action. Reloading this page ends the in-page recovery; inspect current membership before preparing another conversion.</p> }
      </section>
    }
  `,
  styles: `fieldset { border: 0; margin: 16px 0; padding: 0; min-width: 0; } button { margin: 8px 8px 0 0; } details { margin-top: 16px; } .form-grid > section { min-width: 0; } h4 { margin-bottom: 10px; }`
})
export class RoleConversionComponent { readonly service = inject(RoleConversionService) }
