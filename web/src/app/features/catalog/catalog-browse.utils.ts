import type { AccessRequests, CatalogCategoryNode, CatalogListEntry, CatalogListOptions, CatalogListPage } from '../../core/generated/access.types'
import type { CatalogItem } from './catalog.models'
import type { TimingPolicy } from '../timing/timing.models'

export interface CatalogFilters { application?: string; category?: string; descendants?: boolean; tag?: string; owner?: string; runner?: string; safety?: string; source?: string; sort?: string }
export type CatalogCategory = CatalogCategoryNode
export interface CatalogOption { value: string; label: string }
export const uncategorized = '\u0000uncategorized'

export function catalogListRequest(query: string, state: string, suite: string, type: string, filters: CatalogFilters, cursor = ''): NonNullable<AccessRequests['listProjectCatalog']['query']> {
  return { query, state, suite, type, application: filters.application, category: filters.category === uncategorized ? '' : filters.category, uncategorized: filters.category === uncategorized, descendants: filters.descendants !== false, tag: filters.tag, owner: filters.owner, runner: filters.runner, safety: filters.safety, source: filters.source, sort: filters.sort || 'title', limit: 30, cursor: cursor || undefined }
}

export function catalogListItem(entry: CatalogListEntry): CatalogItem {
  let timing: TimingPolicy | undefined
  if (entry.data.timing) {
    const policy = entry.data.timing
    if (!['off', 'warn', 'fail'].includes(policy.mode) || !['fixed', 'adaptive'].includes(policy.timeoutMode)) throw new Error('The catalog contains an unsupported timing policy.')
    timing = { ...policy, mode: policy.mode as TimingPolicy['mode'], timeoutMode: policy.timeoutMode as TimingPolicy['timeoutMode'] }
  }
  return { ...entry, data: { ...entry.data, timing } }
}

export function mergeCatalogItems(known: CatalogItem[], incoming: CatalogItem[]): CatalogItem[] {
  const items = new Map(known.map(item => [item.suiteId + '/' + item.id, item]))
  for (const item of incoming) items.set(item.suiteId + '/' + item.id, item)
  return [...items.values()]
}

export function catalogFacetOptions(facets: CatalogListOptions, applications: { id: string; data: { name: string } }[]) {
  const options = (values: string[]): CatalogOption[] => values.map(value => ({ value, label: value }))
  return { application: facets.application.map(value => ({ value, label: applications.find(row => row.id === value)?.data.name ?? value })), tag: options(facets.tag), owner: options(facets.owner), runner: options(facets.runner), safety: options(facets.safety), source: options(facets.source) }
}
export function catalogTypeOptions(values: string[]): CatalogOption[] {
  return values.map(value => ({ value, label: value === 'https' || value === 'api' ? value.toUpperCase() : value.charAt(0).toUpperCase() + value.slice(1) }))
}
export function emptyCatalogPage(): CatalogListPage { return { items: [], total: 0, matched: 0, counts: { total: 0, enabled: 0, safe: 0, suites: 0 }, categories: [], options: { application: [], tag: [], owner: [], runner: [], safety: [], source: [], type: [] } } }
export function categoryLabel(path: string): string { return path === uncategorized ? 'Uncategorized' : path || 'All categories' }
export function categoryInputValue(value: string): string { return value === uncategorized ? '' : value }
export function hiddenCatalogSelections(selected: CatalogItem[], visible: CatalogItem[]): number { const keys = new Set(visible.map(item => item.suiteId + '/' + item.id)); return selected.filter(item => !keys.has(item.suiteId + '/' + item.id)).length }
