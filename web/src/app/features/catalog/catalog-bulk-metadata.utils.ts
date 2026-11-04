import type { CatalogBulkMetadata } from '../../core/generated/access.types'
import type { TestDefinition } from './catalog.models'

export const bulkMetadataFields = [
  { key: 'category', label: 'Category', maximum: 200 },
  { key: 'tags', label: 'Tags', maximum: 2000 },
  { key: 'owner', label: 'Owner', maximum: 200 },
  { key: 'title', label: 'Title', maximum: 200 },
  { key: 'description', label: 'Description', maximum: 4096 },
  { key: 'safety', label: 'Safety label', maximum: 20 },
  { key: 'rationale', label: 'Safety rationale', maximum: 4096 },
  { key: 'conditions', label: 'Operating conditions', maximum: 4096 }
] as const
export type BulkMetadataField = typeof bulkMetadataFields[number]['key']
export type BulkMetadataEdits = Record<BulkMetadataField, { apply: boolean; value: string }>

export function defaultBulkMetadata(): BulkMetadataEdits {
  return Object.fromEntries(bulkMetadataFields.map(field => [field.key, { apply: false, value: field.key === 'safety' ? 'needs-review' : '' }])) as BulkMetadataEdits
}

export function bulkMetadataFieldIssue(edits: BulkMetadataEdits, field: BulkMetadataField): string {
  const edit = edits[field]
  if (!edit.apply) return ''
  const definition = bulkMetadataFields.find(item => item.key === field)!
  if (['category', 'title', 'rationale'].includes(field) && !edit.value.trim()) return `${definition.label} requires nonempty text.`
  if (new TextEncoder().encode(edit.value).length > definition.maximum) return `${definition.label} exceeds ${definition.maximum} bytes.`
  if (field === 'safety' && !['safe', 'unsafe', 'needs-review'].includes(edit.value)) return 'Choose a valid safety label.'
  if (field === 'tags') {
    const tags = metadataTags(edit.value)
    if (tags.length > 30 || new Set(tags).size !== tags.length || tags.some(tag => new TextEncoder().encode(tag).length > 64)) return 'Use up to 30 distinct tags, each at most 64 bytes.'
  }
  return ''
}

export function bulkMetadataIssue(edits: BulkMetadataEdits): string {
  if (!bulkMetadataFields.some(field => edits[field.key].apply)) return 'Choose at least one field to change.'
  for (const field of bulkMetadataFields) { const issue = bulkMetadataFieldIssue(edits, field.key); if (issue) return issue }
  if (edits.safety.apply && !edits.rationale.apply) return 'Select Safety rationale when changing a safety label.'
  return ''
}

export function bulkMetadataInput(edits: BulkMetadataEdits): CatalogBulkMetadata {
  const issue = bulkMetadataIssue(edits)
  if (issue) throw new Error(issue)
  return Object.fromEntries(bulkMetadataFields.filter(field => edits[field.key].apply).map(field => [field.key, field.key === 'tags' ? metadataTags(edits.tags.value) : edits[field.key].value]))
}

export function metadataChanges(before: TestDefinition | undefined, after: TestDefinition): { label: string; before: string; after: string }[] {
  if (!before) return []
  return bulkMetadataFields.filter(field => JSON.stringify(before[field.key]) !== JSON.stringify(after[field.key])).map(field => ({ label: field.label, before: metadataValue(before[field.key]), after: metadataValue(after[field.key]) }))
}

function metadataTags(value: string): string[] { return value.split(',').map(tag => tag.trim()).filter(Boolean) }
function metadataValue(value: string | string[] | null | undefined): string { return (Array.isArray(value) ? value.join(', ') : value) || 'Empty' }
