import type { SavedSelection, SavedSelectionFilter } from '../../core/generated/access.types'
import type { CatalogItem } from './catalog.models'

export interface SavedSelectionForm {
  suite: string; id: string; name: string; description: string; kind: string; tests: string
  query: string; category: string; descendants: boolean; tags: string; owner: string; application: string; type: string; runner: string; safety: string; source: string
}

export function emptySavedSelectionForm(suite = ''): SavedSelectionForm {
  return { suite, id: '', name: '', description: '', kind: 'explicit', tests: '', query: '', category: '', descendants: false, tags: '', owner: '', application: '', type: '', runner: '', safety: '', source: '' }
}

export function savedSelectionDefinition(value: SavedSelectionForm): SavedSelection {
  const filter: SavedSelectionFilter = { query: value.query, category: value.category, includeCategoryDescendants: value.descendants, tags: value.tags.split(',').map(tag => tag.trim()).filter(Boolean), owner: value.owner, applicationId: value.application, type: value.type, runner: value.runner, safety: value.safety, source: value.source }
  return { id: value.id, name: value.name, description: value.description, archived: false, spec: { schemaVersion: 1, kind: value.kind === 'filter' ? 'filter' : 'explicit', ...(value.kind === 'filter' ? { filter } : { testIds: value.tests.split(/[\s,]+/).filter(Boolean) }) } }
}

export function savedSelectionForm(suite: string, definition: SavedSelection): SavedSelectionForm {
  const filter = definition.spec.filter
  return { ...emptySavedSelectionForm(suite), id: definition.id, name: definition.name, description: definition.description, kind: definition.spec.kind, tests: definition.spec.testIds?.join(', ') ?? '', query: filter?.query ?? '', category: filter?.category ?? '', descendants: filter?.includeCategoryDescendants ?? false, tags: filter?.tags?.join(', ') ?? '', owner: filter?.owner ?? '', application: filter?.applicationId ?? '', type: filter?.type ?? '', runner: filter?.runner ?? '', safety: filter?.safety ?? '', source: filter?.source ?? '' }
}

export function savedSelectionFromItems(items: CatalogItem[]): { suite: string; tests: string } {
  if (!items.length || items.length > 1000) throw new Error('Select between 1 and 1000 tests to save an explicit selection.')
  if (items.some(item => item.suiteId !== items[0].suiteId)) throw new Error('A saved selection uses one suite. Filter by suite before selecting tests.')
  return { suite: items[0].suiteId, tests: items.map(item => item.id).sort().join(', ') }
}
