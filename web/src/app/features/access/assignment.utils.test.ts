import assert from 'node:assert/strict'
import { test } from 'node:test'
import type { AssignmentDefinition, Role } from '../../core/generated/access.types'
import { assignableRoles, assignmentDefinition, assignmentRights, assignmentValidation } from './assignment.utils.ts'

const roles = [{ id: 'viewer', disabled: false, delegatable: true }, { id: 'root', disabled: false, delegatable: false }, { id: 'paused', disabled: true, delegatable: true }] as Role[]
test('assignment forms require current server choices and strip caller-owned scope fields', () => {
  const input = { principalKind: 'user', principalId: ' person ', roleId: 'viewer', reason: ' Review ', bindingValues: { team: '*' }, revision: 9 } as AssignmentDefinition
  const definition = assignmentDefinition(input)
  assert.deepEqual(definition, { principalKind: 'user', principalId: 'person', roleId: 'viewer', reason: 'Review' })
  assert.deepEqual(assignableRoles(roles).map(role => role.id), ['viewer'])
  assert.equal(assignmentValidation(definition, roles), '')
  for (const roleId of ['root', 'paused', 'unknown']) assert.match(assignmentValidation({ ...definition, roleId }, roles), /available role/)
  assert.match(assignmentValidation({ ...definition, principalId: '../another' }, roles), /registered/)
  assert.match(assignmentValidation({ ...definition, reason: 'é'.repeat(513) }, roles), /1024 bytes/)
  assert.equal(assignmentRights({ 'teststudio.members.read': true }).read, false)
  assert.equal(assignmentRights({ 'teststudio.members.manage': true }).manage, false)
  assert.equal(assignmentRights({ 'teststudio.members.manage': true, 'teststudio.permissions.grant': true }).manage, true)
})
