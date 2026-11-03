import { computed, inject, Injectable, signal } from '@angular/core'
import { CatalogPlanService } from './catalog-plan.service'
import { bulkMetadataFieldIssue, bulkMetadataFields, bulkMetadataInput, bulkMetadataIssue, defaultBulkMetadata, type BulkMetadataField } from './catalog-bulk-metadata.utils'

@Injectable()
export class CatalogBulkMetadataService {
  private readonly plans = inject(CatalogPlanService)
  readonly fields = bulkMetadataFields
  readonly edits = signal(defaultBulkMetadata())
  readonly issue = computed(() => bulkMetadataIssue(this.edits()))

  select(field: BulkMetadataField, apply: boolean): void { this.plans.clear(); this.edits.update(edits => ({ ...edits, [field]: { ...edits[field], apply } })) }
  change(field: BulkMetadataField, value: string): void { this.plans.clear(); this.edits.update(edits => ({ ...edits, [field]: { ...edits[field], value } })) }
  reset(): void { this.edits.set(defaultBulkMetadata()) }
  fieldIssue(field: BulkMetadataField): string { return bulkMetadataFieldIssue(this.edits(), field) }
  input() { return bulkMetadataInput(this.edits()) }
}
