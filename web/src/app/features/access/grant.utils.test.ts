import assert from 'node:assert/strict'
import { test } from 'node:test'
import type { GrantRecord, PermissionChoice } from '../../core/generated/access.types'
import { grantDefinition, grantFields, grantReadRights, grantTarget, grantUpdate, grantValidation } from './grant.utils'

const permissions: PermissionChoice[] = [{ id: 'teststudio.catalog.read', area: 'catalog', action: 'read', description: 'Read catalog', installationOnly: false, canGrant: true }]
test('direct grant fields preserve exact expiry and updates cannot change permanent identity', () => {
  const fields = { ...grantFields(), principalId: ' member ', permission: 'teststudio.catalog.read', object: ' suite/health ', expiresAt: '2000-01-01T00:00:00.123456Z', reason: ' Review access ' }
  assert.equal(grantValidation(fields, null, permissions), '')
  assert.deepEqual(grantDefinition(fields), { principalKind: 'user', principalId: 'member', permission: 'teststudio.catalog.read', effect: 'allow', object: 'suite/health', expiresAt: fields.expiresAt, reason: 'Review access' })
  assert.deepEqual(grantUpdate(fields), { expiresAt: fields.expiresAt, reason: 'Review access' })
  const row: GrantRecord = { id: 'abc', projectId: 'alpha', revision: 8, updatedAt: '2026-01-01T00:00:00Z', data: { ...grantDefinition(fields), projectId: 'alpha', principalName: 'Member', principalActive: true, expired: true, canUpdate: true, canRemove: true } }
  assert.equal(grantFields(row).expiresAt, fields.expiresAt)
  assert.equal(grantFields(row).reason, '')
  assert.equal(grantValidation({ ...fields, principalId: '*', permission: 'invalid' }, row, []), '')
  assert.deepEqual(grantUpdate({ ...fields, expiresAt: '' }), { expiresAt: null, reason: 'Review access' })
  assert.match(grantTarget(row), /Member.*allow.*teststudio.catalog.read.*alpha.*suite\/health/)
})
test('direct grant validation rejects invalid or undelegatable rules and unsupported expiry', () => {
  const fields = { ...grantFields(), principalId: 'member', permission: 'teststudio.catalog.read', reason: 'Review' }
  for (const changed of [{ object: 'suite/*' }, { principalId: '*' }, { permission: 'teststudio.unknown.read' }]) assert.notEqual(grantValidation({ ...fields, ...changed }, null, permissions), '')
  assert.notEqual(grantValidation(fields, null, permissions.map(p => ({ ...p, canGrant: false }))), '')
  for (const expiresAt of ['0000-01-01T00:00:00Z', '2027-02-30T00:00:00Z', '2027-01-01T24:00:00Z']) assert.notEqual(grantValidation({ ...fields, expiresAt }, null, permissions), '')
  assert.notEqual(grantValidation({ ...fields, reason: ' ' }, null, permissions), '')
  assert.equal(grantReadRights({ 'teststudio.permissions.read': true }), false)
  assert.equal(grantReadRights({ 'teststudio.permissions.read': true, 'teststudio.members.read': true }), true)
})
