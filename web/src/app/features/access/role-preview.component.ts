import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { ReactiveFormsModule } from '@angular/forms'
import { RoleEditorService } from './role-editor.service'

@Component({
  selector: 'ts-role-preview', standalone: true, imports: [ReactiveFormsModule], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section aria-labelledby="role-preview-heading"><h3 id="role-preview-heading">Preview effective access</h3><p class="small muted">Compare a registered user's current access with this proposed policy. An enabled role is assigned only inside a preview that is rolled back. Existing roles, groups, and denies remain in effect.</p>
      <div class="form-grid" [formGroup]="editor.preview.form"><div class="field"><label for="preview-user">Preview user identifier</label><input id="preview-user" formControlName="userId" maxlength="128"></div><div class="field"><label for="preview-object">Preview exact object (optional)</label><input id="preview-object" formControlName="object" placeholder="suite/health" maxlength="400"></div></div>
      <button type="button" class="button secondary" [disabled]="editor.preview.busy() || editor.busy() || editor.conflict()" (click)="editor.inspect()">Preview access</button>
      @if (editor.preview.error()) { <p class="notice error" role="alert">{{ editor.preview.error() }}</p> }
      @if (editor.preview.busy()) { <p class="small" role="status">Evaluating proposed access…</p> }
      @if (editor.preview.result(); as result) {
        @if (!editor.previewCurrent()) { <p class="notice warning" role="status">Preview is out of date. Run it again for the current fields.</p> }
        @else {
          <p class="notice success" role="status">Preview complete. No policy, assignment, or audit change was saved.</p>
          @if (result.userDisabled) { <p class="notice warning">This account is disabled. Its requests remain denied.</p> }
          <label class="checkbox-field"><input type="checkbox" [checked]="editor.preview.showUnchanged()" (change)="editor.preview.showUnchanged.set($any($event.target).checked)">Show unchanged permissions</label>
          @if (!editor.comparisons().length) { <p class="notice">No access changes in this context.</p> }
          <div class="table-scroll preview-table" tabindex="0" role="region" aria-label="Access preview results"><table><caption class="sr-only">Access before and after the proposed policy</caption><thead><tr><th scope="col">Permission</th><th scope="col">Current access</th><th scope="col">Proposed access</th></tr></thead><tbody>@for (row of editor.comparisons(); track row.permission) { <tr><th scope="row"><code class="small wrap-anywhere">{{ row.permission }}</code></th><td>{{ row.before ? 'Allowed' : 'Denied' }}</td><td><span class="badge" [class.success]="row.after" [class.warning]="!row.after">{{ row.after ? 'Allowed' : 'Denied' }}</span></td></tr> }</tbody></table></div>
        }
      }
    </section>
  `,
  styles: `section { margin-top: 28px; padding-top: 22px; border-top: 1px solid var(--border); } h3 { margin-bottom: 12px; } .form-grid { margin: 16px 0; } .notice { margin-top: 16px; } .preview-table { max-height: 400px; margin: 16px 0; border: 1px solid var(--border); border-radius: 7px; } .preview-table th:first-child { min-width: 230px; }`
})
export class RolePreviewComponent { readonly editor = inject(RoleEditorService) }
