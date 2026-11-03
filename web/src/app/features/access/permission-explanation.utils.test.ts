import test from 'node:test'
import assert from 'node:assert/strict'
import type { PermissionExplanation, PermissionSource } from '../../core/generated/access.types'
import { appendExplanation, sourceDetails, sourceTitle } from './permission-explanation.utils.ts'

test('Restricted source presentation drops administrative details even if provided', () => {
  const source: PermissionSource = { id: 'opaque', kind: 'role', effect: 'allow', status: 'matching', scopeKnown: false, restricted: true, principalId: 'private-user', roleId: 'private-role', teamScope: 'private-project', expiresAt: '2030-01-01T00:00:00Z', restrictedFields: ['private-field'] }
  assert.deepEqual(sourceDetails(source), [])
  assert.equal(sourceTitle(source), 'Restricted source')
})

test('Continuation cannot mix another context or a changed decision with an earlier report', () => {
  const first = { userId: 'member', projectId: 'alpha', object: 'suite/health', permission: 'teststudio.catalog.read', allowed: true, reason: 'allowed', cursor: 'opaque', items: [{ id: 'one' }] } as PermissionExplanation
  const next = { ...first, cursor: undefined, items: [{ id: 'one' }, { id: 'two' }] } as PermissionExplanation
  assert.deepEqual(appendExplanation(first, next).items.map(source => source.id), ['one', 'two'])
  for (const change of [{ userId: 'another' }, { projectId: 'beta' }, { object: '' }, { permission: 'teststudio.catalog.edit' }, { allowed: false }, { reason: 'explicit-deny' }]) assert.throws(() => appendExplanation(first, { ...next, ...change } as PermissionExplanation), /changed during the scan/)
  assert.throws(() => appendExplanation({ ...first, cursor: undefined }, next), /changed during the scan/)
})
