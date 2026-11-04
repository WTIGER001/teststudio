import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { RouterLink } from '@angular/router'
import { CatalogService } from './catalog.service'
import { CatalogSelectionService } from './catalog-selection.service'
import { CatalogEntryService } from './catalog-entry.service'

@Component({
  selector: 'ts-catalog-table', standalone: true, imports: [RouterLink], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
        <div class="table-scroll"><table><caption class="sr-only">Tests in this project</caption><thead><tr>@if (selection.canSelect()) { <th scope="col"><span class="sr-only">Select</span></th> }<th scope="col">Test</th><th scope="col">Suite / category</th><th scope="col">Type</th><th scope="col">Safety</th><th scope="col">State</th><th scope="col">Actions</th></tr></thead><tbody>
          @for (item of service.visible(); track item.suiteId + '/' + item.id) {
            <tr>@if (selection.canSelect()) { <td><input class="table-checkbox" type="checkbox" [checked]="selection.contains(item)" (change)="selection.toggle(item)" [attr.aria-label]="'Select ' + item.id"></td> }<td class="test-identity"><strong><a [routerLink]="['/projects', service.projectId(), 'tests', item.suiteId, item.id, 'history']">{{ item.data.title }}</a></strong><code class="table-subtitle">{{ item.id }} · revision {{ item.revision }}</code></td><td>{{ item.suiteName }}<span class="table-subtitle muted">{{ item.data.category }}</span></td><td><span class="badge">{{ item.data.type }}</span></td><td><span class="badge" [class.success]="item.data.safety === 'safe'" [class.warning]="item.data.safety === 'needs-review'">{{ item.data.safety }}</span></td><td><span class="state-label"><span class="status-dot" [class.inactive]="item.data.state !== 'enabled'" aria-hidden="true"></span>{{ item.data.state }}</span></td>
              <td><div class="row-actions wrap-actions">@if (service.workspace.can('catalog.update')) { <button class="button small subtle" type="button" (click)="service.edit(item)">Edit</button> }
              @if (item.data.state === 'enabled' && service.workspace.can('catalog.update')) { <button class="button small subtle" type="button" [disabled]="service.saving()" (click)="service.action(item, 'disable')">Disable</button> }
              @if ((item.data.state === 'disabled' || item.data.state === 'draft') && service.workspace.can('catalog.restore')) { <button class="button small subtle" type="button" [disabled]="service.saving()" (click)="service.action(item, 'enable')">Enable</button> }
              @if (item.data.state === 'archived' && service.workspace.can('catalog.restore')) { <button class="button small subtle" type="button" [disabled]="service.saving()" (click)="service.action(item, 'restore')">Restore</button> }
              <details class="entry-actions"><summary>More</summary><div class="row-actions wrap-actions"><button class="button small subtle" type="button" (click)="entry.open(item, 'history')">Revisions</button><a class="button small subtle" [routerLink]="['/projects', service.projectId(), 'tests', item.suiteId, item.id, 'history']">History</a>@if (service.workspace.can('catalog.create')) { <button class="button small subtle" type="button" (click)="entry.open(item, 'duplicate')">Duplicate</button> }@if (service.workspace.can('catalog.archive') && item.data.state !== 'archived') { <button class="button small subtle" type="button" [disabled]="service.saving()" (click)="service.action(item, 'archive')">Archive</button> }</div></details></div></td></tr>
          }
        </tbody></table></div>
  `,
  styles: [`:host { display: block; min-width: 0; } .table-scroll { margin: 0; } table { min-width: 800px; } .test-identity { min-width: 180px; overflow-wrap: anywhere; } .entry-actions { min-width: 60px; } summary { cursor: pointer; padding: 8px; } .entry-actions .row-actions { padding: 8px 0; }`]
})
export class CatalogTableComponent { readonly service = inject(CatalogService); readonly selection = inject(CatalogSelectionService); readonly entry = inject(CatalogEntryService) }
