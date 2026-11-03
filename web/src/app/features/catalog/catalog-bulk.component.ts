import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { FormsModule } from '@angular/forms'
import { CatalogSelectionService } from './catalog-selection.service'
import { CatalogBulkMetadataComponent } from './catalog-bulk-metadata.component'

@Component({
  selector: 'ts-catalog-bulk', standalone: true, imports: [FormsModule, CatalogBulkMetadataComponent], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    @if (service.canSelect()) {
      <div class="catalog-bulk" aria-label="Bulk catalog actions"><span class="small muted">{{ service.items().length }} selected</span>
        <button class="button small subtle" type="button" (click)="service.selectVisible()">Select visible tests</button><button class="button small subtle" type="button" (click)="service.clear()">Clear selection</button>
        <label class="sr-only" for="bulk-action">Bulk action</label><select id="bulk-action" [ngModel]="service.action()" (ngModelChange)="service.selectAction($event)"><option value="">Choose a bulk action</option>@if (service.catalog.workspace.can('catalog.update')) { <option value="metadata">Edit metadata</option><option value="disable">Disable</option> }@if (service.catalog.workspace.can('catalog.restore')) { <option value="enable">Enable</option><option value="restore">Restore</option> }@if (service.catalog.workspace.can('catalog.archive')) { <option value="archive">Archive</option> }</select>
        <button class="button secondary" type="button" [disabled]="!service.items().length || !service.action() || service.loading() || (service.action() === 'metadata' && !!service.metadata.issue())" (click)="service.preview()">Preview bulk changes</button>
      </div>
      @if (service.action() === 'metadata') { <ts-catalog-bulk-metadata /> }
      @if (service.error()) { <p class="notice error" role="alert">{{ service.error() }}</p> }
    }
  `
})
export class CatalogBulkComponent { readonly service = inject(CatalogSelectionService) }
