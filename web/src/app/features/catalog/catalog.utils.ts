import type { CatalogItem } from './catalog.models'

export function filterCatalog(items: CatalogItem[], query: string, state: string, suite = '', type = ''): CatalogItem[] {
  const term = query.trim().toLowerCase()
  return items.filter(item => {
    const test = item.data
    const stateMatches = state === 'all' || (state === 'active' ? test.state !== 'archived' : test.state === state)
    const text = [test.id, test.title, test.description, test.category, item.suiteName, ...(test.tags || [])].join(' ').toLowerCase()
    return stateMatches && (!suite || item.suiteId === suite) && (!type || test.type === type) && (!term || text.includes(term))
  }).sort((a, b) => a.data.title.localeCompare(b.data.title) || a.suiteId.localeCompare(b.suiteId) || a.id.localeCompare(b.id))
}

export function catalogTypes(items: CatalogItem[]): { value: string; label: string }[] {
  return [...new Set(items.map(item => item.data.type))].filter(Boolean).sort().map(value => ({ value, label: value === 'https' || value === 'api' ? value.toUpperCase() : value.charAt(0).toUpperCase() + value.slice(1) }))
}

export function catalogCounts(items: CatalogItem[]): { total: number; enabled: number; safe: number; suites: number } {
  const active = items.filter(item => item.data.state !== 'archived')
  return { total: active.length, enabled: active.filter(item => item.data.state === 'enabled').length, safe: active.filter(item => item.data.safety === 'safe').length, suites: new Set(active.map(item => item.suiteId)).size }
}
