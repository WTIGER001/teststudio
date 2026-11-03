import { ChangeDetectionStrategy, Component, computed, input } from '@angular/core'
import type { TestDefinition } from './catalog.models'
import { metadataChanges } from './catalog-bulk-metadata.utils'

@Component({
  selector: 'ts-catalog-metadata-changes', standalone: true, changeDetection: ChangeDetectionStrategy.OnPush,
  template: `@if (changes().length) { <dl class="changes">@for (change of changes(); track change.label) { <dt>{{ change.label }}</dt><dd><span class="muted">{{ change.before }}</span> → {{ change.after }}</dd> }</dl> }`,
  styles: [`.changes { margin: 0 0 .75rem; } dt { font-weight: 600; } dd { margin: .25rem 0 .75rem; white-space: pre-wrap; overflow-wrap: anywhere; }`]
})
export class CatalogMetadataChangesComponent {
  readonly before = input<TestDefinition>()
  readonly after = input.required<TestDefinition>()
  readonly changes = computed(() => metadataChanges(this.before(), this.after()))
}
