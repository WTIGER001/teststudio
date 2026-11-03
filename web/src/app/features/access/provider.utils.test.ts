import test from 'node:test'
import assert from 'node:assert/strict'
import type { ProviderRecord } from '../../core/generated/access.types'
import { providerDefinition, providerFields, providerIssuerValid, providerValidation } from './provider.utils.ts'

const fields = () => ({ ...providerFields(), id: 'local-provider', name: 'Local provider', issuer: 'https://identity.example.test', clientId: 'teststudio', reason: 'Review an independent provider' })

test('Provider editing rejects duplicate external values before creating a dictionary', () => {
  const value = { ...fields(), groupClaim: 'groups', mappings: [{ external: 'reviewers', groupId: 'first' }, { external: 'reviewers', groupId: 'second' }] }
  assert.match(providerValidation(value, [], null), /unique exact external/)
  const definition = providerDefinition({ ...value, mappings: [{ external: '__proto__', groupId: 'first' }, { external: ' exact value ', groupId: 'second' }] })
  assert.equal(Object.hasOwn(definition.groupMappings, '__proto__'), true)
  assert.equal(definition.groupMappings[' exact value '], 'second')
})

test('Only the selected existing unavailable credential can be retained while disabled', () => {
  const selected = { id: 'local-provider', data: { secretRef: 'env:PREVIOUS' } } as ProviderRecord
  const value = { ...fields(), disabled: true, secretRef: 'env:PREVIOUS' }
  assert.equal(providerValidation(value, [], selected), '')
  assert.match(providerValidation({ ...value, disabled: false }, [], selected), /operator-approved/)
  assert.match(providerValidation({ ...value, secretRef: 'env:ANOTHER_SECRET' }, [], selected), /operator-approved/)
  assert.match(providerValidation(value, [], null), /operator-approved/)
})

test('Issuer forms preserve exact HTTPS or loopback contexts and reject query or credential URLs', () => {
  for (const value of ['https://identity.example.test/tenant', 'http://127.0.0.1:8050/oidc/one', 'http://127.0.0.2:8050/oidc/one', 'http://[::1]:8050/oidc/one']) assert.equal(providerIssuerValid(value), true)
  for (const value of ['http://identity.example.test', 'https://user:secret@identity.example.test', 'https://identity.example.test?', 'https://identity.example.test/#fragment', '//identity.example.test']) assert.equal(providerIssuerValid(value), false)
})
