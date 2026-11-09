import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { FormsModule } from '@angular/forms'
import { CatalogBulkMetadataService } from './catalog-bulk-metadata.service'

@Component({
  selector: 'ts-catalog-bulk-metadata', standalone: true, imports: [FormsModule], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="metadata-editor" aria-labelledby="bulk-metadata-title"><h3 id="bulk-metadata-title">Edit selected test metadata</h3>
      <p class="small muted">Select each field to replace across the selected tests. Unselected fields keep their values. Empty optional values clear the selected field. Source definitions and saved versions stay unchanged.</p>
      <div class="form-grid">@for (field of service.fields; track field.key) {
        <div class="field"><label class="checkbox-field"><input type="checkbox" [ngModel]="service.edits()[field.key].apply" (ngModelChange)="service.select(field.key, $event)" [attr.aria-label]="'Change ' + field.label">Change {{ field.label }}</label>
          <fieldset [disabled]="!service.edits()[field.key].apply"><label [for]="'bulk-' + field.key">Bulk {{ field.label }}</label>
            @if (field.key === 'safety') { <select [id]="'bulk-' + field.key" [ngModel]="service.edits()[field.key].value" (ngModelChange)="service.change(field.key, $event)"><option value="needs-review">Needs review</option><option value="safe">Safe</option><option value="unsafe">Unsafe</option></select> }
            @else if (field.maximum > 2000) { <textarea [id]="'bulk-' + field.key" rows="2" [maxLength]="field.maximum" [ngModel]="service.edits()[field.key].value" (ngModelChange)="service.change(field.key, $event)" [attr.aria-describedby]="'bulk-' + field.key + '-issue'"></textarea> }
            @else { <input [id]="'bulk-' + field.key" [maxLength]="field.maximum" [ngModel]="service.edits()[field.key].value" (ngModelChange)="service.change(field.key, $event)" [attr.aria-describedby]="'bulk-' + field.key + '-issue'"> }
          </fieldset>
          @if (field.key === 'tags') { <small>Separate tags with commas. Leave empty to remove all tags.</small> }
          <small [id]="'bulk-' + field.key + '-issue'" class="error-text">{{ service.fieldIssue(field.key) }}</small>
        </div>
      }</div>
      @if (service.issue()) { <p class="small muted">{{ service.issue() }}</p> }
      <p class="small muted">A safety change requires an explicit rationale and current saving permission. Review the before and after values before applying.</p>
    </section>
  `,
  styles: [`fieldset { border: 0; margin: 0; padding: 0; min-width: 0; display: grid; gap: .5rem; } .metadata-editor { margin: 1rem 0; padding: 1rem; border: 1px solid var(--border); border-radius: 8px; } .error-text { color: var(--error); }`]
})
export class CatalogBulkMetadataComponent { readonly service = inject(CatalogBulkMetadataService) }
