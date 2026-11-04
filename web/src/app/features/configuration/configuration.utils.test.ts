import assert from 'node:assert/strict'
import test from 'node:test'
import { credentialBindingProblem, credentialBindings } from './configuration.utils.ts'

test('environment credentials require unique names and exact operator-approved origins', () => {
  const refs = [{ id: 'reader', name: 'API reader', allowedOrigins: ['https://approved.test'] }]
  const rows = [{ binding: 'api-login', reference: 'reader' }]
  assert.equal(credentialBindingProblem(rows, refs, 'https://approved.test'), undefined)
  assert.equal(credentialBindingProblem(rows, refs, 'https://foreign.test')?.field, 'reference')
  assert.equal(credentialBindingProblem([...rows, ...rows], refs, 'https://approved.test')?.field, 'binding')
  assert.equal(credentialBindingProblem(rows, [], 'https://approved.test')?.field, 'reference')
  assert.deepEqual(credentialBindings(rows), { 'api-login': 'reader' })
})
