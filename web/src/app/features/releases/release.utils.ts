import type { SourceCatalogRelease, SourceCatalogTest } from '../../core/generated/access.types'

export interface ReleaseChange { id: string; title: string; action: string; fields: string }
export interface ReleaseComparison { changes: ReleaseChange[]; artifactChanged: boolean; timingChanged: boolean; captureChanged: boolean }

function stable(value: unknown): string {
  if (Array.isArray(value)) return `[${value.map(stable).join(',')}]`
  if (value !== null && typeof value === 'object') return `{${Object.entries(value).sort(([a], [b]) => a.localeCompare(b)).map(([key, item]) => `${JSON.stringify(key)}:${stable(item)}`).join(',')}}`
  return JSON.stringify(value) ?? 'undefined'
}
function testChanges(before: SourceCatalogTest, after: SourceCatalogTest): string {
  const oldValue = before as unknown as Record<string, unknown>, newValue = after as unknown as Record<string, unknown>
  return [...new Set([...Object.keys(oldValue), ...Object.keys(newValue)])].filter(key => key !== 'id' && stable(oldValue[key]) !== stable(newValue[key])).sort().join(', ')
}
export function compareReleases(before: SourceCatalogRelease | undefined, after: SourceCatalogRelease | null): ReleaseComparison | null {
  if (!before || !after || before.id === after.id) return null
  const previous = new Map(before.catalog.tests.map(test => [test.id, test])), next = new Map(after.catalog.tests.map(test => [test.id, test]))
  const changes: ReleaseChange[] = []
  for (const test of after.catalog.tests) {
    const old = previous.get(test.id)
    if (!old) changes.push({ id: test.id, title: test.title, action: 'Added', fields: 'New definition' })
    else { const fields = testChanges(old, test); if (fields) changes.push({ id: test.id, title: test.title, action: 'Changed', fields }) }
  }
  for (const test of before.catalog.tests) if (!next.has(test.id)) changes.push({ id: test.id, title: test.title, action: 'Removed', fields: 'Absent from selected version' })
  return { changes: changes.sort((a, b) => a.id.localeCompare(b.id)), artifactChanged: stable(before.catalog.artifact) !== stable(after.catalog.artifact), timingChanged: stable(before.catalog.timing) !== stable(after.catalog.timing), captureChanged: stable(before.catalog.capture) !== stable(after.catalog.capture) }
}
export function filterReleases(releases: SourceCatalogRelease[], query: string): SourceCatalogRelease[] {
  const term = query.trim().toLowerCase()
  return releases.filter(release => !term || [release.id, release.actor, release.catalog.artifact.provenance?.commit, release.catalog.artifact.provenance?.repository].some(value => value?.toLowerCase().includes(term)))
}
export function comparisonBase(releases: SourceCatalogRelease[], selected: string, active: string): string { return active && active !== selected ? active : releases.find(release => release.id !== selected)?.id ?? '' }
export function githubCommit(release: SourceCatalogRelease | null): string {
  const origin = release?.catalog.artifact.provenance
  if (!origin?.repository || !origin.commit || !/^(?:[a-f0-9]{40}|[a-f0-9]{64})$/.test(origin.commit)) return ''
  try { const url = new URL(origin.repository); if (url.protocol !== 'https:' || url.hostname !== 'github.com' || url.username || url.password || url.search || url.hash || !/^\/[A-Za-z0-9-]+\/[A-Za-z0-9_.-]+\/?$/.test(url.pathname)) return ''; return `${url.origin}${url.pathname.replace(/\/$/, '')}/commit/${origin.commit}` } catch { return '' }
}
