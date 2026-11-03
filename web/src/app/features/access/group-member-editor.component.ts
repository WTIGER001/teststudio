import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { ReactiveFormsModule } from '@angular/forms'
import { GroupMembersService } from './group-members.service'

@Component({
  selector: 'ts-group-member-editor', standalone: true, imports: [ReactiveFormsModule], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="member-editor" aria-labelledby="member-editor-heading"><div class="section-top"><h3 id="member-editor-heading" tabindex="-1">{{ editor.editing() ? 'Edit local membership' : 'Add local member' }}</h3><button type="button" class="button subtle" [disabled]="editor.busy()" (click)="editor.close()">Close membership editor</button></div>
      <form [formGroup]="editor.form" (ngSubmit)="editor.submit()"><fieldset [disabled]="editor.busy()"><legend class="sr-only">Local membership</legend>
        <div class="form-grid"><div class="field"><label for="member-user">Registered user identifier</label><input id="member-user" formControlName="user" required [readOnly]="!!editor.editing()"><small>Use an existing internal user identifier.</small></div><div class="field"><label for="member-expiry">Membership expiry (UTC, optional)</label><input id="member-expiry" type="datetime-local" step="1" formControlName="expiry"><small>Enter UTC. Leave empty for no expiry.</small></div></div>
        @if (editor.error()) { <p class="notice error" role="alert">{{ editor.error() }}</p> }
        @if (editor.conflict()) { <p class="small">Membership changed. Your edits remain here. Reload the stored membership before retrying.</p> }
        <div class="row-actions wrap-actions"><button type="submit" class="button primary" [disabled]="editor.conflict()">Save local membership</button>@if (editor.editing()) { <button type="button" class="button secondary" (click)="editor.reload()">Reload stored membership</button> }</div>
      </fieldset></form>
    </section>
  `,
  styles: `fieldset { border: 0; margin: 0; padding: 0; min-width: 0; } .member-editor { margin: 20px 0; padding: 20px 0; border-top: 1px solid var(--border); border-bottom: 1px solid var(--border); }`
})
export class GroupMemberEditorComponent { readonly editor = inject(GroupMembersService) }
