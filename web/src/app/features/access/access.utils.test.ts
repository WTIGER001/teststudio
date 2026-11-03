import assert from 'node:assert/strict'
import { test } from 'node:test'
import type { Group, Membership, User } from '../../core/generated/access.types'
import { accessRights, canEditMembership, expiryForm, groupDefinition, groupRoute, groupValidation, membershipExpiry, reasonValidation, userDefinition, userValidation } from './access.utils.ts'

test('editable definitions strip identity state and preserve permanent group ownership', () => {
  const user = { id: ' user-one ', name: ' Example ', email: ' person@example.test ', disabled: true, emailVerified: true, sessionEpoch: 4 } as User
  assert.deepEqual(userDefinition(user), { id: 'user-one', name: 'Example', email: 'person@example.test' })
  const group = { id: 'editors', name: ' Editors ', description: 'Review', projectId: 'alpha', archived: true, managedBy: 'oidc:provider' } as Group
  assert.deepEqual(groupDefinition(group), { id: 'editors', name: 'Editors', description: 'Review', projectId: 'alpha' })
  assert.deepEqual(groupRoute(group), ['/projects', 'alpha', 'groups', 'editors'])
  assert.deepEqual(groupRoute({ id: 'shared' }), ['/settings', 'groups', 'shared'])
  assert.match(userValidation({ id: 'valid', name: 'é'.repeat(129) }), /256 bytes/)
  assert.match(groupValidation({ id: 'valid', name: 'Valid', description: 'é'.repeat(2049) }), /4096 bytes/)
  assert.match(reasonValidation('   '), /reason/)
})

test('membership expiry uses explicit UTC and rejects invalid, past, and nonexistent calendar values', () => {
  const now = Date.parse('2026-11-03T12:00:00Z')
  assert.deepEqual(membershipExpiry('', now), { expiresAt: null, error: '' })
  assert.deepEqual(membershipExpiry('2026-11-04T15:30', now), { expiresAt: '2026-11-04T15:30:00.000Z', error: '' })
  assert.equal(membershipExpiry('2026-11-04T15:30:00-05:00', now).expiresAt, '2026-11-04T20:30:00.000Z')
  assert.equal(expiryForm('2026-11-04T15:30:00-05:00'), '2026-11-04T20:30:00')
  for (const value of ['yesterday', '2026-11-03T11:59', '2027-02-30T12:00']) assert.match(membershipExpiry(value, now).error, /future UTC/)
})

test('group scope and source govern local membership controls independently from user administration', () => {
  const capabilities = { 'teststudio.groups.read': true, 'teststudio.groups.manage': true }
  assert.equal(accessRights(capabilities, '').manageMembers, true)
  assert.equal(accessRights(capabilities, 'alpha').manageMembers, false)
  assert.equal(accessRights(capabilities, 'alpha').manageUsers, false)
  const group = { id: 'editors', archived: false } as Group
  const local = { source: 'local', readOnly: false } as Membership
  assert.equal(canEditMembership(group, true, local), true)
  assert.equal(canEditMembership({ ...group, archived: true }, true, local), false)
  assert.equal(canEditMembership({ ...group, managedBy: 'oidc:provider' }, true, local), false)
  assert.equal(canEditMembership(group, true, { ...local, source: 'oidc:provider' }), false)
  assert.equal(canEditMembership(group, false, local), false)
})
