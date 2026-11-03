import { strict as assert } from 'node:assert'
import { test } from 'node:test'
import { identityLinkAuthorizePath, identityLinkDefinition, identityLinkTerminal, identityLinkValidation } from './identity-link.utils'

test('Link input sends credentials only for the actual source method', () => {
  const input = { providerId: 'new', reason: ' Add sign-in ', password: 'do not retain', confirmLocal: true }
  assert.deepEqual(identityLinkDefinition(input, 'oidc'), { providerId: 'new', reason: 'Add sign-in' })
  assert.deepEqual(identityLinkDefinition(input, 'local'), { providerId: 'new', reason: 'Add sign-in', confirmLocal: true })
  assert.deepEqual(identityLinkDefinition(input, 'password'), { providerId: 'new', reason: 'Add sign-in', password: 'do not retain' })
  assert.match(identityLinkValidation({ providerId: 'new', reason: 'Link' }, 'password'), /password/)
  assert.match(identityLinkValidation({ providerId: 'new', reason: 'Link' }, 'local'), /Confirm/)
  assert.equal(identityLinkValidation({ providerId: 'new', reason: 'Link' }, 'oidc'), '')
})

test('Only active operations allow another proof or confirmation', () => {
  for (const state of ['complete', 'failed', 'expired', 'cancelled'] as const) assert.equal(identityLinkTerminal(state), true)
  for (const state of ['authenticate-source', 'prove-target', 'review'] as const) assert.equal(identityLinkTerminal(state), false)
  assert.equal(identityLinkAuthorizePath({ id: '../identity?value=1' } as any), '/auth/identity-links/..%2Fidentity%3Fvalue%3D1/authorize')
})
