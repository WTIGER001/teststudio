import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { ReactiveFormsModule } from '@angular/forms'
import { GrantEditorService } from './grant-editor.service'
import { grantTarget } from './grant.utils'

@Component({
  selector: 'ts-grant-editor', standalone: true, imports: [ReactiveFormsModule], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="panel" aria-labelledby="grant-editor-heading"><div class="section-top"><h2 id="grant-editor-heading" tabindex="-1">{{ editor.selected() ? 'Edit grant expiry' : 'Create direct grant' }}</h2><button type="button" class="button subtle" [disabled]="editor.busy()" (click)="editor.close()">Close grant editor</button></div>
      <p class="small muted wrap-anywhere">Scope: {{ editor.access.projectId() || 'Installation · all projects' }}. Direct grants apply alongside roles and group membership.</p>
      @if (editor.selected(); as selected) { <p class="small wrap-anywhere">{{ target(selected) }}</p><p class="small muted">The principal, permission, effect, object, and ownership are permanent. Remove this grant and create another to change them.</p> }
      <form [formGroup]="editor.form" (ngSubmit)="editor.submit()"><fieldset [disabled]="editor.busy()"><legend class="sr-only">Direct grant fields</legend>
        @if (!editor.selected()) {
          <div class="form-grid"><div class="field"><label for="grant-kind">Principal type</label><select id="grant-kind" formControlName="principalKind"><option value="user">User</option><option value="group">Group</option></select></div><div class="field"><label for="grant-principal">Registered principal identifier</label><input id="grant-principal" formControlName="principalId" required maxlength="128" pattern="[A-Za-z0-9][A-Za-z0-9._-]{0,127}"></div></div>
          <p class="small muted">Shared installation groups require installation group authority. Local and OIDC users share permanent internal identifiers.</p>
          <div class="field"><label for="grant-object">Exact object (optional)</label><input id="grant-object" formControlName="object" maxlength="400" aria-describedby="grant-object-help"></div><p id="grant-object-help" class="small muted">Leave empty for the displayed scope. Use an existing exact resource such as suite/health or test/health/check. Wildcards are not supported.</p>
          <button type="button" class="button secondary" [disabled]="editor.permissionsLoading()" (click)="editor.loadPermissions()">Refresh grant permissions</button>
          @if (editor.permissionsLoading()) { <p class="small" role="status">Loading permission choices…</p> }
          @if (editor.permissionsError()) { <p class="notice error" role="alert">{{ editor.permissionsError() }}</p> }
          @if (!editor.permissionsCurrent() && !editor.permissionsLoading()) { <p class="small muted">Refresh grant permissions for the current exact object before saving.</p> }
          <div class="form-grid"><div class="field"><label for="grant-permission">Permission</label><select id="grant-permission" formControlName="permission" required [attr.aria-disabled]="!editor.permissionsCurrent()"><option value="">Select a permission</option>@for (choice of editor.permissions(); track choice.id) { <option [value]="choice.id" [disabled]="!choice.canGrant || !editor.permissionsCurrent()">{{ choice.id }}{{ choice.canGrant ? '' : ' · Read only' }} — {{ choice.description }}</option> }</select></div><div class="field"><label for="grant-effect">Effect</label><select id="grant-effect" formControlName="effect"><option value="allow">Allow</option><option value="deny">Deny</option></select></div></div>
          <p class="small muted">An active deny overrides matching allows, including access from roles and groups.</p>
        }
        <div class="field"><label for="grant-expiry">Expiry in UTC (optional)</label><input id="grant-expiry" formControlName="expiresAt" placeholder="2027-01-01T12:00:00Z" maxlength="40" aria-describedby="grant-expiry-help"></div><p id="grant-expiry-help" class="small muted">Enter a complete timestamp with Z or a UTC offset. Empty means no expiry. Expired grants remain visible and grant no access.</p>
        <div class="field"><label for="grant-reason">Grant change reason</label><textarea id="grant-reason" formControlName="reason" required maxlength="1024"></textarea></div>
        @if (editor.error()) { <p class="notice error" role="alert">{{ editor.error() }}</p> }
        @if (editor.conflict()) { <p class="small">The record changed or the save could not be confirmed. Your entries remain here. Review current state before another submission.</p> }
        <div class="row-actions wrap-actions"><button type="submit" class="button primary" [disabled]="editor.conflict() || (!editor.selected() && (!editor.access.canCreate() || !editor.permissionsCurrent()))">{{ editor.selected() ? 'Save grant expiry' : 'Create grant' }}</button>@if (editor.selected()) { <button type="button" class="button secondary" (click)="editor.reload()">Reload stored grant</button> } @else if (editor.conflict()) { <button type="button" class="button secondary" (click)="editor.access.refresh()">Refresh current grants</button> }</div>
      </fieldset></form>
    </section>
  `,
  styles: `fieldset { border: 0; margin: 18px 0 0; padding: 0; min-width: 0; } .field { min-width: 0; } select, input, textarea { width: 100%; min-width: 0; }`
})
export class GrantEditorComponent { readonly editor = inject(GrantEditorService); readonly target = grantTarget }
