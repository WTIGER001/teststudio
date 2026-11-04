import assert from 'node:assert/strict'
import { test } from 'node:test'
import type { CatalogItem } from './catalog.models'
import { filterCatalog } from './catalog.utils.ts'
import { catalogCategories, catalogOptions, hiddenCatalogSelections, uncategorized } from './catalog-browse.utils.ts'

function item(id: string, suiteId: string, category: string, extras = {}): CatalogItem {
  return { id, suiteId, suiteName: suiteId, revision: 1, updatedAt: '2026-11-01T00:00:00Z', data: { id, title: id, category, type: 'https', runner: 'http', state: 'enabled', tags: ['smoke'], applicationId: 'web', owner: 'QA', safety: 'safe', source: 'release', definitionSourceId: 'registered', ...extras } } as CatalogItem
}
const items = [item('B', 'one', 'Availability / Local'), item('A', 'one', 'Availability'), item('C', 'two', 'AvailabilityExtra'), item('D', 'one', 'Availability/Local/Checkout', { state: 'archived', tags: [] }), item('E', 'one', '')]

test('category matching follows path boundaries, whitespace, exact scope, and lifecycle', () => {
  assert.deepEqual(filterCatalog(items, '', 'active', '', '', { category: 'Availability' }).map(x => x.id), ['A', 'B'])
  assert.deepEqual(filterCatalog(items, '', 'all', '', '', { category: ' Availability / Local ' }).map(x => x.id), ['B', 'D'])
  assert.deepEqual(filterCatalog(items, '', 'all', '', '', { category: 'Availability/Local', descendants: false }).map(x => x.id), ['B'])
  assert.deepEqual(filterCatalog(items, '', 'all', '', '', { category: uncategorized }).map(x => x.id), ['E'])
})

test('tree parents aggregate only supplied matches and retain permanent suite identities', () => {
  const tree = catalogCategories(filterCatalog(items, '', 'active'))
  assert.deepEqual(tree.map(x => [x.path, x.count]), [['Availability', 2], ['AvailabilityExtra', 1], [uncategorized, 1]])
  assert.equal(tree[0].children[0].path, 'Availability/Local')
  assert.equal(tree[0].children[0].count, 1)
  assert.equal(catalogCategories([item('x', 'one', 'A//B')])[0].path, 'A//B')
  assert.equal(hiddenCatalogSelections([items[0], item('B', 'two', 'Availability')], [items[0]]), 1)
})

test('advanced filters combine exact metadata, both source identities, text, and stable sorting', () => {
  const filters = { application: 'web', tag: 'smoke', owner: 'QA', runner: 'http', safety: 'safe', source: 'registered', sort: 'title-desc' }
  assert.deepEqual(filterCatalog(items, 'Availability', 'active', 'one', 'https', filters).map(x => x.id), ['B', 'A'])
  assert.deepEqual(filterCatalog(items, '', 'all', '', '', { tag: 'missing' }), [])
  assert.deepEqual(filterCatalog(items, '', 'all', '', '', { source: 'missing' }), [])
  assert.equal(filterCatalog(items, '', 'all', '', '', { source: 'release' }).length, 5)
  const first = item('same', 'a', 'A'), second = item('same', 'b', 'B')
  assert.deepEqual(filterCatalog([second, first], '', 'all', '', '', { sort: 'updated' }).map(x => x.suiteId), ['a', 'b'])
  assert.deepEqual(catalogOptions(items, 'application', [{ id: 'web', data: { name: 'Website' } }]), [{ value: 'web', label: 'Website' }])
})
