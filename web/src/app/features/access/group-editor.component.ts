import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { ReactiveFormsModule } from '@angular/forms'
import { GroupEditorService } from './group-editor.service'

@Component({
  selector: 'ts-group-editor', standalone: true, imports: [ReactiveFormsModule], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="panel" aria-labelledby="group-editor-heading"><div class="section-top"><h2 id="group-editor-heading" tabindex="-1">{{ editor.editing() ? 'Edit group' : 'Create group' }}</h2><button type="button" class="button subtle" [disabled]="editor.busy()" (click)="editor.close()">Close editor</button></div>
      <p class="small muted wrap-anywhere">Ownership: {{ editor.groups.projectId() || 'Installation' }}. Ownership scope remains permanent. New groups are locally managed.</p>
      <form [formGroup]="editor.form" (ngSubmit)="editor.submit()"><fieldset [disabled]="editor.busy()"><legend class="sr-only">Group configuration</legend>
        <div class="form-grid"><div class="field"><label for="group-name">Group name</label><input id="group-name" formControlName="name" required maxlength="256"></div><div class="field"><label for="group-id">Group identifier</label><input id="group-id" formControlName="id" required [readOnly]="!!editor.editing()"></div></div><div class="field"><label for="group-description">Description (optional)</label><textarea id="group-description" formControlName="description" maxlength="4096"></textarea></div>
        @if (editor.error()) { <p class="notice error" role="alert">{{ editor.error() }}</p> }
        @if (editor.conflict()) { <p class="small">The group changed. Your edits remain here. Reload stored configuration before retrying.</p> }
        <div class="row-actions wrap-actions"><button type="submit" class="button primary" [disabled]="editor.conflict()">Save group</button>@if (editor.editing()) { <button type="button" class="button secondary" (click)="editor.reload()">Reload stored group</button> }</div>
      </fieldset></form>
    </section>
  `,
  styles: `fieldset { border: 0; margin: 18px 0 0; padding: 0; min-width: 0; }`
})
export class GroupEditorComponent { readonly editor = inject(GroupEditorService) }
