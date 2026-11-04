import type { CatalogItem } from './catalog.models'

export interface CatalogFilters {
  application?: string; category?: string; descendants?: boolean; tag?: string; owner?: string; runner?: string; safety?: string; source?: string; sort?: string
}
export interface CatalogCategory { path: string; label: string; count: number; children: CatalogCategory[] }
export interface CatalogOption { value: string; label: string }
export const uncategorized = '\u0000uncategorized'

export function categoryPath(value: string): string {
  return (value ?? '').split('/').map(part => part.trim()).join('/')
}

export function matchesCatalogFilters(item: CatalogItem, filters: CatalogFilters): boolean {
  const test = item.data
  for (const [requested, actual] of [[filters.application, test.applicationId], [filters.owner, test.owner], [filters.runner, test.runner], [filters.safety, test.safety]]) {
    if (requested && requested !== actual) return false
  }
  if (filters.tag && !test.tags?.includes(filters.tag)) return false
  if (filters.source && filters.source !== test.source && filters.source !== test.definitionSourceId) return false
  if (!filters.category) return true
  const category = categoryPath(test.category)
  if (filters.category === uncategorized) return !category
  const requested = categoryPath(filters.category)
  return category === requested || filters.descendants !== false && category.startsWith(requested + '/')
}

export function sortCatalog(items: CatalogItem[], sort = 'title'): CatalogItem[] {
  return [...items].sort((a, b) => {
    let result = 0
    switch (sort) {
      case 'title-desc': result = b.data.title.localeCompare(a.data.title); break
      case 'id': result = a.id.localeCompare(b.id); break
      case 'category': result = categoryPath(a.data.category).localeCompare(categoryPath(b.data.category)); break
      case 'updated': result = b.updatedAt.localeCompare(a.updatedAt); break
      default: result = a.data.title.localeCompare(b.data.title)
    }
    return result || a.data.title.localeCompare(b.data.title) || a.suiteId.localeCompare(b.suiteId) || a.id.localeCompare(b.id)
  })
}

export function catalogCategories(items: CatalogItem[]): CatalogCategory[] {
  const roots: CatalogCategory[] = []
  for (const item of items) {
    const category = categoryPath(item.data.category)
    const parts = category ? category.split('/').every(Boolean) ? category.split('/') : [category] : ['']
    let children = roots, path = ''
    for (const part of parts) {
      path = path ? path + '/' + part : part
      const key = category ? path : uncategorized
      let node = children.find(value => value.path === key)
      if (!node) { node = { path: key, label: part || 'Uncategorized', count: 0, children: [] }; children.push(node) }
      node.count++
      children = node.children
    }
  }
  const order = (nodes: CatalogCategory[]): CatalogCategory[] => nodes.sort((a, b) => a.label.localeCompare(b.label)).map(node => ({ ...node, children: order(node.children) }))
  return order(roots)
}

export function catalogOptions(items: CatalogItem[], field: 'application' | 'tag' | 'owner' | 'runner' | 'safety' | 'source', applications: { id: string; data: { name: string } }[] = []): CatalogOption[] {
  const values = items.flatMap(item => {
    const test = item.data
    switch (field) {
      case 'application': return [test.applicationId ?? '']
      case 'tag': return test.tags ?? []
      case 'source': return [test.definitionSourceId ?? '', test.source ?? '']
      default: return [test[field] ?? '']
    }
  })
  return [...new Set(values)].filter(Boolean).sort((a, b) => a.localeCompare(b)).map(value => ({ value, label: field === 'application' ? applications.find(row => row.id === value)?.data.name ?? value : value }))
}

export function categoryLabel(path: string): string {
  return path === uncategorized ? 'Uncategorized' : path || 'All categories'
}

export function hiddenCatalogSelections(selected: CatalogItem[], visible: CatalogItem[]): number {
  const keys = new Set(visible.map(item => item.suiteId + '/' + item.id))
  return selected.filter(item => !keys.has(item.suiteId + '/' + item.id)).length
}

export function categoryInputValue(value: string): string {
  return value === uncategorized ? '' : value
}
