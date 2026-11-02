import assert from 'node:assert/strict'
import { test } from 'node:test'
import type { CatalogItem } from './catalog.models'
import { catalogTypes, filterCatalog } from './catalog.utils.ts'

test('catalog types combine with suite, lifecycle, and search without conflating runner and type', () => {
  const items = [
    { id: 'health', suiteId: 'checks', suiteName: 'Checks', data: { id: 'health', title: 'Health', type: 'https', runner: 'http', state: 'enabled' } },
    { id: 'api', suiteId: 'checks', suiteName: 'Checks', data: { id: 'api', title: 'API', type: 'api', runner: 'http', state: 'enabled' } },
    { id: 'old', suiteId: 'checks', suiteName: 'Checks', data: { id: 'old', title: 'Old', type: 'https', runner: 'http', state: 'archived' } },
    { id: 'browser', suiteId: 'ui', suiteName: 'UI', data: { id: 'browser', title: 'Browser', type: 'browser', runner: 'playwright', state: 'enabled' } }
  ] as CatalogItem[]
  assert.deepEqual(filterCatalog(items, '', 'active', 'checks', 'https').map(item => item.id), ['health'])
  assert.deepEqual(filterCatalog(items, 'old', 'all', 'checks', 'https').map(item => item.id), ['old'])
  assert.deepEqual(filterCatalog(items, '', 'active', 'ui', 'https'), [])
  assert.deepEqual(catalogTypes(items).map(option => option.label), ['API', 'Browser', 'HTTPS'])
})
