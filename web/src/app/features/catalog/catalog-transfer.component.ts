import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { ReactiveFormsModule } from '@angular/forms'
import { CatalogTransferService } from './catalog-transfer.service'

@Component({
  selector: 'ts-catalog-transfer', standalone: true, imports: [ReactiveFormsModule], providers: [CatalogTransferService], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <details class="panel catalog-transfer"><summary>Import and export catalog definitions</summary>
      <p class="small muted">Import native HTTP checks with a review before apply. Each request accepts up to 30 tests and 2 MiB of JSON. Omitted tests remain in the catalog.</p>
      @if (service.loading()) { <p class="small muted" role="status">Preparing catalog transfer…</p> }
      @if (service.error()) { <p class="notice error" role="alert">{{ service.error() }}</p> }
      <form [formGroup]="service.form" (ngSubmit)="service.preview()">
        <div class="form-grid"><div class="field"><label for="transfer-suite">Destination suite</label><select id="transfer-suite" formControlName="suite" required><option value="">Choose a suite</option>@for (suite of service.catalog.suites(); track suite.id) { <option [value]="suite.id">{{ suite.data.name }}</option> }</select></div>
        <div class="field"><label for="catalog-file">Load a JSON file</label><input id="catalog-file" type="file" accept=".json,application/json" (change)="service.file($any($event.target).files?.[0])"></div></div>
        <div class="field"><label for="catalog-document">Catalog JSON document</label><textarea id="catalog-document" class="code-input" formControlName="document" rows="9" spellcheck="false" placeholder='{"schemaVersion":"teststudio.catalog/v1","tests":[]}'></textarea></div>
        <div class="row-actions wrap-actions">
          @if (service.catalog.workspace.can('catalog.export')) { <button type="button" class="button secondary" (click)="service.export()" [disabled]="service.loading() || service.form.controls.suite.invalid">Export suite JSON</button> }
          @if (service.catalog.workspace.can('catalog.import')) { <button type="submit" class="button primary" [disabled]="service.loading() || service.form.controls.suite.invalid || !service.form.controls.document.value">Validate import and preview</button> }
          @if (service.downloadURL()) { <a class="button secondary" [href]="service.downloadURL()" [download]="service.form.controls.suite.value + '.catalog.json'">Download catalog JSON</a> }
        </div>
      </form>
    </details>
  `
})
export class CatalogTransferComponent { readonly service = inject(CatalogTransferService) }
