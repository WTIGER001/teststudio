import assert from 'node:assert/strict'
import { test } from 'node:test'
import type { CatalogListEntry } from '../../core/generated/access.types'
import { catalogListRequest, catalogListItem, mergeCatalogItems, hiddenCatalogSelections, uncategorized } from './catalog-browse.utils.ts'

test('category navigation sends explicit uncategorized and descendant semantics to the API', () => {
  const query = catalogListRequest('health', 'active', 'suite', 'https', { category: uncategorized, descendants: false, source: 'registered', sort: 'updated' }, 'opaque')
  assert.equal(query.category, ''); assert.equal(query.uncategorized, true); assert.equal(query.descendants, false)
  assert.equal(query.source, 'registered'); assert.equal(query.limit, 30); assert.equal(query.cursor, 'opaque')
  assert.equal(catalogListRequest('', 'active', '', '', { category: 'Availability / Local' }).descendants, true)
})

test('page merging preserves independent suite identities and updates selected revisions', () => {
  const row = (suiteId: string, revision: number) => catalogListItem({ id: 'SAME', suiteId, revision, data: { id: 'SAME', title: 'Check', tags: null, dependencies: null, requiredBindings: null } } as CatalogListEntry)
  const known = [row('one', 1), row('two', 1)]
  const merged = mergeCatalogItems(known, [row('one', 2)])
  assert.equal(merged.length, 2); assert.equal(merged[0].revision, 2)
  assert.equal(hiddenCatalogSelections(merged, [merged[0]]), 1)
  assert.equal(merged[0].data.dependencies, null); assert.equal(merged[0].data.requiredBindings, null); assert.equal(merged[0].data.tags, null)
})

test('unsupported timing criteria cannot silently become an editable catalog policy', () => {
  assert.throws(() => catalogListItem({ data: { timing: { mode: 'unknown', timeoutMode: 'fixed' } } } as CatalogListEntry), /unsupported timing/)
})
