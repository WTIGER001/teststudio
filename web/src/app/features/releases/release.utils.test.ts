import assert from 'node:assert/strict'
import { test } from 'node:test'
import type { SourceCatalogRelease } from '../../core/generated/access.types'
import { compareReleases, comparisonBase, filterReleases, githubCommit } from './release.utils.ts'

test('release comparison preserves permanent identities and separates artifact changes from definitions', () => {
  const before = { id: 'old', actor: 'publisher', catalog: { artifact: { image: 'old-image' }, timing: { minimumSamples: 3 }, capture: { logs: true }, tests: [
    { id: 'one', title: 'One', check: { headers: { first: 'a', second: 'b' } } }, { id: 'removed', title: 'Removed' }
  ] } } as unknown as SourceCatalogRelease
  const after = { id: 'new', actor: 'publisher', catalog: { artifact: { image: 'new-image' }, timing: { minimumSamples: 3 }, capture: { logs: true }, tests: [
    { id: 'one', title: 'One', check: { headers: { second: 'b', first: 'a' } } }, { id: 'added', title: 'Added' }
  ] } } as unknown as SourceCatalogRelease
  const result = compareReleases(before, after)!
  assert.deepEqual(result.changes.map(change => [change.id, change.action]), [['added', 'Added'], ['removed', 'Removed']])
  assert.equal(result.artifactChanged, true); assert.equal(result.timingChanged, false); assert.equal(result.captureChanged, false)
  after.catalog.tests[0].title = 'New one'
  assert.equal(compareReleases(before, after)!.changes.find(change => change.id === 'one')?.fields, 'title')
  assert.equal(compareReleases(before, before), null)
  assert.equal(comparisonBase([before, after], 'old', 'new'), 'new')
  assert.deepEqual(filterReleases([before, after], 'publisher').map(release => release.id), ['old', 'new'])
})

test('recorded commit links use immutable GitHub provenance and reject credential-bearing URLs', () => {
  const release = { catalog: { artifact: { provenance: { repository: 'https://github.com/owner/repository', commit: 'a'.repeat(40) } } } } as unknown as SourceCatalogRelease
  assert.equal(githubCommit(release), `https://github.com/owner/repository/commit/${'a'.repeat(40)}`)
  release.catalog.artifact.provenance!.repository = 'https://private-secret@github.com/owner/repository'
  assert.equal(githubCommit(release), '')
})
