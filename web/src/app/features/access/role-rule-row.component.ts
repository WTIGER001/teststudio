import { ChangeDetectionStrategy, Component, inject, input, OnChanges, output } from '@angular/core'
import { ReactiveFormsModule } from '@angular/forms'
import type { RoleRuleForm } from './role-policy.forms'
import { permissionDescription } from './role-policy.utils'
import { RoleRulePermissions } from './role-permissions.service'

@Component({
  selector: 'ts-role-rule-row', standalone: true, imports: [ReactiveFormsModule], providers: [RoleRulePermissions], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <fieldset class="rule-row" [formGroup]="row()"><legend>Permission rule {{ index() + 1 }}</legend>
      <div class="section-top"><p class="small muted">Allow or deny one action in this role's scope.</p><button type="button" class="button subtle" (click)="remove.emit()" [attr.aria-label]="'Remove rule ' + (index() + 1)">Remove rule</button></div>
      <div class="form-grid"><div class="field"><label [for]="'rule-object-' + index()">Exact object (optional)</label><input [id]="'rule-object-' + index()" formControlName="object" placeholder="suite/health" maxlength="400"></div><div class="field"><label [for]="'rule-effect-' + index()">Effect</label><select [id]="'rule-effect-' + index()" formControlName="effect"><option value="allow">Allow</option><option value="deny">Deny</option></select></div></div>
      <div class="row-actions wrap-actions"><button type="button" class="button secondary" [disabled]="permissions.loading()" (click)="permissions.load(true)">Refresh rule permissions</button><span class="small muted wrap-anywhere">Choices for {{ permissions.loadedObject() || 'the full ownership scope' }}. Refresh after changing the object.</span></div>
      @if (permissions.error()) { <p class="notice error" role="alert">{{ permissions.error() }}</p> }
      @if (permissions.loading()) { <p class="small" role="status">Loading rule permissions…</p> }
      <div class="field"><label [for]="'rule-permission-' + index()">Permission</label><select [id]="'rule-permission-' + index()" formControlName="permission" required><option value="">Select a permission</option>@for (choice of permissions.choices(); track choice.id) { <option [value]="choice.id" [disabled]="!choice.canGrant">{{ choice.id }}{{ choice.canGrant ? '' : ' · Cannot delegate' }}</option> }</select><p class="hint">{{ describe(permissions.choices(), row().controls.permission.value) }}</p></div>
      <div class="field"><label [for]="'rule-expiry-' + index()">Expiry in UTC (optional)</label><input [id]="'rule-expiry-' + index()" formControlName="expiresAt" placeholder="2027-01-01T00:00:00Z" maxlength="64"><p class="hint">Use a complete timestamp with Z or a timezone offset. Expired rules remain visible and grant no access.</p></div>
    </fieldset>
  `,
  styles: `.rule-row { border: 1px solid var(--border); padding: 18px; border-radius: 8px; margin: 18px 0; min-width: 0; } legend { padding: 0 6px; font-size: 13px; font-weight: 600; } .field { margin-top: 14px; } .hint { font-size: 12px; color: var(--muted); line-height: 1.6; margin: 9px 0 0; }`
})
export class RoleRuleRowComponent implements OnChanges {
  readonly row = input.required<RoleRuleForm>()
  readonly index = input.required<number>()
  readonly project = input.required<string>()
  readonly remove = output<void>()
  readonly permissions = inject(RoleRulePermissions)
  readonly describe = permissionDescription
  ngOnChanges(): void { this.permissions.select(this.project(), this.row()) }
}
