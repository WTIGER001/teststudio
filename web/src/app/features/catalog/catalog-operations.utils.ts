import type { RecordValue } from '../../core/api.service'
import type { CatalogItem } from './catalog.models'
import type { CatalogDocument, CatalogRevision, TestRevision } from './catalog-operations.models'

export function catalogKey(item: CatalogItem): string { return `${item.suiteId}/${item.id}` }
export function selectedItems(items: CatalogItem[], keys: string[]): CatalogItem[] { return items.filter(item => keys.includes(catalogKey(item))) }
export function bulkInput(items: CatalogItem[], action: string): { suite: string; input: { action: string; tests: { id: string; revision: number }[] } } {
  if (!items.length || items.length > 30) throw new Error('Select between 1 and 30 tests for a bulk action.')
  const suite = items[0].suiteId
  if (items.some(item => item.suiteId !== suite)) throw new Error('Bulk actions use one suite at a time. Filter the catalog by suite, then select tests.')
  if (!action) throw new Error('Choose a bulk action.')
  return { suite, input: { action, tests: items.map(item => ({ id: item.id, revision: item.revision })) } }
}

export function parseCatalogDocument(text: string): CatalogDocument {
  let value: unknown
  try { value = JSON.parse(text) } catch { throw new Error('The catalog document is not valid JSON.') }
  if (!value || typeof value !== 'object' || !('tests' in value) || !Array.isArray(value.tests)) throw new Error('The catalog document must contain a tests array.')
  return value as CatalogDocument
}

export function newestRevisions(rows: RecordValue<TestRevision>[]): CatalogRevision[] { return [...rows].sort((a, b) => b.id.localeCompare(a.id)).map(row => ({ ...row, number: Number(row.id) })) }
