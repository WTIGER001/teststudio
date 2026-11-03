import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { ReactiveFormsModule } from '@angular/forms'
import { UserEditorService } from './user-editor.service'

@Component({
  selector: 'ts-user-editor', standalone: true, imports: [ReactiveFormsModule], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="panel" aria-labelledby="user-editor-heading">
      <div class="section-top"><h2 id="user-editor-heading" tabindex="-1">{{ editor.editing() ? 'Edit user profile' : 'Register user profile' }}</h2><button type="button" class="button subtle" [disabled]="editor.busy()" (click)="editor.close()">Close editor</button></div>
      @if (editor.editing()) { <p class="small muted">Edit display information for this account. Identity links and access use its permanent internal identifier.</p> } @else { <p class="small muted">Registration adds a pending profile. It sends no invitation and grants no access. Matching email addresses do not link identities.</p> }
      <form [formGroup]="editor.form" (ngSubmit)="editor.submit()"><fieldset [disabled]="editor.busy()"><legend class="sr-only">User profile</legend>
        <div class="form-grid"><div class="field"><label for="user-name">Display name</label><input id="user-name" formControlName="name" required maxlength="256"></div><div class="field"><label for="user-id">User identifier</label><input id="user-id" formControlName="id" required [readOnly]="!!editor.editing()"><small>Permanent internal identifier. Email changes do not change it.</small></div><div class="field"><label for="user-email">Email address (optional)</label><input id="user-email" type="email" formControlName="email" maxlength="254"><small>Changing the email clears verification.</small></div></div>
        @if (editor.error()) { <p class="notice error" role="alert">{{ editor.error() }}</p> }
        @if (editor.conflict()) { <p class="small">The profile changed. Your edits remain here. Reload the stored profile before retrying.</p> }
        <div class="row-actions wrap-actions"><button type="submit" class="button primary" [disabled]="editor.conflict()">Save user profile</button>@if (editor.editing()) { <button type="button" class="button secondary" (click)="editor.reload()">Reload stored profile</button> }</div>
      </fieldset></form>
    </section>
  `,
  styles: `fieldset { border: 0; padding: 0; margin: 18px 0 0; min-width: 0; }`
})
export class UserEditorComponent { readonly editor = inject(UserEditorService) }
