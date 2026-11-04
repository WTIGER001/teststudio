import { DatePipe, JsonPipe } from '@angular/common'
import { ChangeDetectionStrategy, Component, input } from '@angular/core'
import type { CatalogRevision } from './catalog-operations.models'

@Component({
  selector: 'ts-test-revisions', standalone: true, imports: [DatePipe, JsonPipe], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `@for (revision of revisions(); track revision.id) { <details class="revision-entry"><summary>Revision {{ revision.number }} · {{ revision.data.at | date:'medium' }} · {{ revision.data.actor }}</summary><pre class="log-view">{{ revision.data.test | json }}</pre></details> } @empty { <p class="small muted">No saved revisions are listed.</p> }`,
  styles: [`:host { display: block; min-width: 0; } summary { cursor: pointer; overflow-wrap: anywhere; }`]
})
export class TestRevisionsComponent { readonly revisions = input.required<CatalogRevision[]>() }
