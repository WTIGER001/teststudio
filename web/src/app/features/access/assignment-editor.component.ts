import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { ReactiveFormsModule } from '@angular/forms'
import { AssignmentEditorService } from './assignment-editor.service'

@Component({
  selector: 'ts-assignment-editor', standalone: true, imports: [ReactiveFormsModule], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="panel" aria-labelledby="assignment-editor-heading"><div class="section-top"><h2 id="assignment-editor-heading" tabindex="-1">{{ editor.editing() ? 'Change member role' : 'Add role assignment' }}</h2><button type="button" class="button subtle" [disabled]="editor.busy()" (click)="editor.close()">Close assignment editor</button></div>
      <p class="small muted wrap-anywhere">Scope: {{ editor.access.scopeLabel() }}. Use the permanent identifier of a registered user or group.</p>
      <p class="small muted">Local and OIDC profiles use the same internal user identifiers. Adding a role does not create credentials or link accounts.</p>
      <form [formGroup]="editor.form" (ngSubmit)="editor.submit()"><fieldset [disabled]="editor.busy() || !editor.access.canAdd()"><legend class="sr-only">Role assignment</legend>
        <div class="form-grid"><div class="field"><label for="assignment-kind">Principal type</label><select id="assignment-kind" formControlName="principalKind"><option value="user">User</option><option value="group">Group</option></select></div><div class="field"><label for="assignment-principal">Registered principal identifier</label><input id="assignment-principal" formControlName="principalId" required maxlength="128" pattern="[A-Za-z0-9][A-Za-z0-9._-]{0,127}" aria-describedby="assignment-principal-help"></div></div>
        <p id="assignment-principal-help" class="small muted">Group assignments require management authority in the group's ownership scope. Shared System groups require System authority.</p>
        <div class="field"><label for="assignment-role">Role to assign</label><select id="assignment-role" formControlName="roleId" required><option value="">Select a role</option>@for (role of editor.access.standardChoices(); track role.id) { <option [value]="role.id" [disabled]="!role.delegatable || role.disabled">{{ role.name }}{{ role.disabled ? ' · Disabled' : !role.delegatable ? ' · Read only' : '' }}</option> }<optgroup label="Advanced roles">@for (role of editor.access.advancedChoices(); track role.id) { <option [value]="role.id" [disabled]="!role.delegatable || role.disabled || !!editor.editing()">{{ role.name }}{{ role.disabled ? ' · Disabled' : !role.delegatable ? ' · Read only' : '' }}</option> }</optgroup></select></div>
        @if (editor.role(); as role) { <p class="small wrap-anywhere">Assign {{ role.name }} in {{ editor.access.scopeLabel() }}. {{ editor.description(role.id) }}</p> }
        <div class="field"><label for="assignment-reason">Assignment reason</label><textarea id="assignment-reason" formControlName="reason" required maxlength="1024"></textarea></div>
        @if (editor.error()) { <p class="notice error" role="alert">{{ editor.error() }}</p> }
        @if (editor.conflict()) { <p class="small">The assignment already exists or its state changed. Your entries remain here. Refresh access to inspect current assignments, then close and reopen this editor.</p> }
        @if (editor.reviewed()) { <section class="notice" aria-labelledby="assignment-review-heading"><h3 id="assignment-review-heading" tabindex="-1">Review role assignment</h3><p>{{ editor.candidate()?.name || editor.form.getRawValue().principalId }} · {{ editor.access.scopeLabel() }}</p><p>{{ editor.editing()?.data?.roleName || 'No direct standard role' }} → {{ editor.role()?.name }}</p><p>{{ editor.description(editor.form.getRawValue().roleId) }}</p><p>Inherited and group access remain in effect. Ownership changes and the last usable Owner remain protected.</p></section> }
        <div class="row-actions wrap-actions"><button type="submit" class="button primary" [disabled]="editor.conflict()">{{ editor.role()?.id && editor.access.isStandard(editor.role()!.id) && !editor.reviewed() ? 'Review assignment' : editor.editing() ? 'Save role' : 'Assign role' }}</button>@if (editor.conflict()) { <button type="button" class="button secondary" (click)="editor.access.refresh()">Refresh current assignments</button> }</div>
      </fieldset></form>
    </section>
  `,
  styles: `fieldset { border: 0; margin: 18px 0 0; padding: 0; min-width: 0; }`
})
export class AssignmentEditorComponent { readonly editor = inject(AssignmentEditorService) }
