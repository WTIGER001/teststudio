import assert from 'node:assert/strict'
import { test } from 'node:test'
import { inheritanceChoices, previewFingerprint, previewRows, roleExpiryValid, roleObjectValid, rolePolicyDefinition, rolePolicyValidation } from './role-policy.utils.ts'
import type { PermissionChoice, RolePolicyRecord } from '../../core/generated/access.types'

const permissions: PermissionChoice[] = [{ id: 'teststudio.catalog.read', area: 'catalog', action: 'read', description: 'Read test catalog entries.', installationOnly: false, canGrant: true }]
const fields = { id: ' reader ', name: ' Reader ', description: ' Notes ', reason: ' Review ', rules: [{ permission: permissions[0].id, effect: 'allow' as const, object: ' suite/health ', expiresAt: '2000-01-01T00:00:00.123456Z' }], inherits: [] }
test('policy forms preserve exact expiry and strip mutable authority from existing identities', () => {
  const created = rolePolicyDefinition(fields, 'alpha', null)
  assert.equal(created.rules?.[0].expiresAt, fields.rules[0].expiresAt)
  assert.equal(created.rules?.[0].object, 'suite/health')
  assert.equal(rolePolicyValidation(created, permissions), '')
  const stored: RolePolicyRecord = { id: 'permanent', revision: 3, updatedAt: '', data: { id: 'permanent', name: 'Before', projectId: 'alpha', description: '', disabled: true, rules: [], inherits: [] } }
  const changed = rolePolicyDefinition({ ...fields, id: 'other' }, 'beta', stored)
  assert.equal(changed.id, 'permanent'); assert.equal(changed.projectId, 'alpha'); assert.equal(changed.disabled, true)
  assert.match(rolePolicyValidation({ ...created, id: 'choices' }, permissions), /identifier/)
  assert.match(rolePolicyValidation({ ...created, rules: [...created.rules!, ...created.rules!] }, permissions), /duplicate/)
  assert.equal(roleExpiryValid('2000-01-01T00:00:00.123456Z'), true)
  assert.equal(roleExpiryValid('2027-02-30T00:00:00Z'), false)
  assert.equal(roleExpiryValid('2027-01-01T00:00:00'), false)
  assert.equal(roleExpiryValid('2027-01-01T24:00:00Z'), false)
  assert.equal(roleObjectValid('suite/*'), false); assert.equal(roleObjectValid('test/health/check'), true)
})
test('preview context expires when policy or subject changes while reasons leave access unchanged', () => {
  const body = rolePolicyDefinition(fields, 'alpha', null), key = previewFingerprint(body, 0, 'member', 'suite/health')
  assert.equal(previewFingerprint({ ...body, reason: 'Different audit reason' }, 0, 'member', 'suite/health'), key)
  assert.notEqual(previewFingerprint({ ...body, rules: [] }, 0, 'member', 'suite/health'), key)
  assert.notEqual(previewFingerprint(body, 0, 'another', 'suite/health'), key)
  assert.deepEqual(inheritanceChoices([], ['outside-first-page'], 'reader'), [[{ id: 'outside-first-page', name: 'outside-first-page · Current selection' }]])
  const preview = { userId: 'member', projectId: 'alpha', object: '', userDisabled: false, decisions: [{ permission: 'changed', before: false, after: true }, { permission: 'same', before: false, after: false }] }
  assert.deepEqual(previewRows(preview).map(row => row.permission), ['changed'])
  assert.equal(previewRows(preview, true).length, 2)
})
