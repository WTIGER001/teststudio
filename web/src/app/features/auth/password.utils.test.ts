import assert from 'node:assert/strict'
import { test } from 'node:test'
import { loginDefinition, loginValidation, passwordChangeDefinition, passwordChangeValidation, passwordPolicy } from './password.utils.ts'

test('password forms preserve significant spaces and strip confirmation or authority fields', () => {
  assert.deepEqual(loginDefinition({ userId: ' person ', password: ' secret spaces ' }), { userId: 'person', password: ' secret spaces ' })
  assert.equal(loginValidation({ userId: 'person', password: 'secret' }), '')
  assert.match(loginValidation({ userId: 'email@example.test', password: 'secret' }), /identifier/)
  const extra = { currentPassword: 'before', newPassword: 'after', confirmation: 'after', userId: 'other' }
  assert.deepEqual(passwordChangeDefinition(extra), { currentPassword: 'before', newPassword: 'after' })
  assert.match(passwordPolicy('too short'), /15 through/)
  assert.equal(passwordPolicy('cafe\u0301 blue mountain sunrise'), '')
  assert.match(passwordPolicy('\u1100\u1161'.repeat(256)), /1024 bytes/)
  assert.match(passwordPolicy('invalid\ncontrol characters'), /control/)
  assert.match(passwordChangeValidation({ currentPassword: 'café silver sunrise', newPassword: 'cafe\u0301 silver sunrise' }, 'cafe\u0301 silver sunrise'), /different/)
  assert.match(passwordChangeValidation({ currentPassword: 'before', newPassword: 'a different long secret' }, 'another'), /match/)
})
