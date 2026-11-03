import { strict as assert } from 'node:assert'
import { test } from 'node:test'
import { authenticationFeedback, providerLoginPath } from './auth-navigation.utils'

test('Provider navigation preserves the exact local destination and removes prior authentication messages', () => {
  const path = providerLoginPath('issuer/name', '/projects/alpha/runs/run-1?test=one&signIn=expired&providerLogout=complete#evidence')
  const url = new URL(path, 'https://studio.example')
  assert.equal(url.pathname, '/auth/login/issuer%2Fname')
  assert.equal(url.searchParams.get('returnTo'), '/projects/alpha/runs/run-1?test=one#evidence')
})

test('Authentication feedback uses fixed messages without displaying provider diagnostics', () => {
  assert.match(authenticationFeedback('/projects?signIn=cancelled&error_description=private').message, /cancelled/)
  assert.equal(authenticationFeedback('/projects?signIn=expired').error, true)
  assert.match(authenticationFeedback('/projects?providerLogout=unconfirmed').message, /remains closed/)
  assert.deepEqual(authenticationFeedback('/projects?signIn=arbitrary-provider-message'), { message: '', error: false })
})
