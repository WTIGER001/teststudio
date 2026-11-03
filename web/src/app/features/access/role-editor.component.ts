import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { ReactiveFormsModule } from '@angular/forms'
import { RoleEditorService } from './role-editor.service'
import { RoleRuleRowComponent } from './role-rule-row.component'
import { RolePreviewComponent } from './role-preview.component'

@Component({
  selector: 'ts-role-editor', standalone: true, imports: [ReactiveFormsModule, RoleRuleRowComponent, RolePreviewComponent], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="panel" aria-labelledby="role-editor-heading"><div class="section-top"><h2 id="role-editor-heading" tabindex="-1">{{ editor.editing() ? 'Edit role policy' : 'Create custom role' }}</h2><button type="button" class="button subtle" [disabled]="editor.busy()" (click)="editor.close()">Close role editor</button></div>
      <p class="small muted wrap-anywhere">Permanent ownership: {{ editor.roles.projectId() || 'Installation' }}. Changes affect current assignments and parent roles.</p>
      <form [formGroup]="editor.form" (ngSubmit)="editor.submit()"><fieldset class="editor-fields" [disabled]="editor.busy()"><legend class="sr-only">Custom role policy</legend>
        <div class="form-grid"><div class="field"><label for="policy-name">Role name</label><input id="policy-name" formControlName="name" required maxlength="200"></div><div class="field"><label for="policy-id">Role identifier</label><input id="policy-id" formControlName="id" required maxlength="128" [readOnly]="!!editor.editing()"></div></div>
        <div class="field"><label for="policy-description">Role description (optional)</label><textarea id="policy-description" formControlName="description" maxlength="4096"></textarea></div>
        <h3 class="subheading">Permission rules</h3><p class="small muted">Deny rules take precedence. Empty object means the full ownership scope. Exact object rules do not grant descendant access.</p>
        @for (row of editor.form.controls.rules.controls; track row; let i = $index) { <ts-role-rule-row [row]="row" [index]="i" [project]="editor.roles.projectId()" (remove)="editor.removeRule(i)" /> }
        <button type="button" class="button secondary" [disabled]="editor.form.controls.rules.length >= 100" (click)="editor.addRule()">+ Add permission rule</button>
        <h3 class="subheading">Inherited roles</h3><p class="small muted">Choose managed roles from this ownership scope. Cycles are rejected, including through disabled roles.</p>
        @for (control of editor.form.controls.inherits.controls; track control; let i = $index) { <div class="inherit-row"><div class="field"><label [for]="'inherited-role-' + i">Inherited role {{ i + 1 }}</label><select [id]="'inherited-role-' + i" [formControl]="control" required><option value="">Select a managed role</option>@for (choice of editor.inheritanceOptions()[i]; track choice.id) { <option [value]="choice.id">{{ choice.name }}</option> }</select></div><button type="button" class="button subtle" [attr.aria-label]="'Remove inherited role ' + (i + 1)" (click)="editor.removeInheritance(i)">Remove</button></div> }
        <div class="row-actions wrap-actions"><button type="button" class="button secondary" [disabled]="editor.form.controls.inherits.length >= 16" (click)="editor.addInheritance()">+ Inherit role</button>@if (editor.roles.policies.cursor()) { <button type="button" class="button secondary" [disabled]="editor.roles.policies.loading()" (click)="editor.roles.loadPolicies(true)">More inheritance choices</button> }</div>
        @if (editor.roles.policies.error()) { <p class="notice error" role="alert">{{ editor.roles.policies.error() }}</p> }
        <ts-role-preview />
        <div class="field reason"><label for="policy-reason">Policy change reason</label><textarea id="policy-reason" formControlName="reason" required maxlength="1024"></textarea></div>
        @if (editor.error()) { <p class="notice error" role="alert">{{ editor.error() }}</p> }
        @if (editor.conflict()) { <p class="notice warning">Your entries remain here. Reload the stored policy before another save. For a new-role conflict, refresh roles, then close and reopen this editor.</p> }
        <div class="row-actions wrap-actions"><button type="submit" class="button primary" [disabled]="editor.conflict() || editor.preview.busy()">Save role policy</button>@if (editor.editing()) { <button type="button" class="button secondary" [disabled]="editor.preview.busy()" (click)="editor.reload()">Reload stored role</button> }</div>
      </fieldset></form>
    </section>
  `,
  styles: `.editor-fields { border: 0; padding: 0; min-width: 0; margin: 18px 0 0; } .subheading { margin: 28px 0 12px; } .inherit-row { display: flex; align-items: center; gap: 12px; margin: 16px 0; } .inherit-row .field { flex: 1; min-width: 0; } .reason { margin-top: 28px; }`
})
export class RoleEditorComponent { readonly editor = inject(RoleEditorService) }
