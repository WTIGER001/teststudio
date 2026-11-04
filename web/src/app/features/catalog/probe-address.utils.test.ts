import { test } from 'node:test'
import assert from 'node:assert/strict'
import { matchProbeAddress, parseProbeAddress } from './probe-address.utils'
import type { Environment, Suite } from '../configuration/configuration.service'
import type { RecordValue } from '../../core/api.service'

const environment: RecordValue<Environment> = { id: 'live', revision: 3, updatedAt: '', data: { id: 'live', name: 'Live', applicationId: 'app', stage: 'production', protected: true, disabled: false, endpoints: { api: 'https://shop.example.test:443' } } }

test('Website addresses need no explicit HTTPS port and retain paths and queries', () => {
  assert.deepEqual(parseProbeAddress('https://shop.example.test'), { origin: 'https://shop.example.test', path: '/', hostname: 'shop.example.test', protocol: 'https:' })
  assert.deepEqual(parseProbeAddress('https://shop.example.test/check?ready=yes'), { origin: 'https://shop.example.test', path: '/check?ready=yes', hostname: 'shop.example.test', protocol: 'https:' })
  assert.equal(parseProbeAddress('https://shop.example.test:8443')?.origin, 'https://shop.example.test:8443')
  for (const value of ['htttps://shop.example.test', 'https://user:secret@example.test', 'https://example.test/#token', '//example.test', 'file:///tmp/check']) assert.equal(parseProbeAddress(value), undefined)
})
test('URL reuse preserves registered protection and does not hide disabled targets', () => {
  const match = matchProbeAddress('https://shop.example.test/healthz', [environment])[0]
  assert.equal(match.environment, environment)
  assert.equal(match.environment.data.protected, true)
  assert.equal(match.path, '/healthz')
  assert.equal(matchProbeAddress('https://shop.example.test', [{ ...environment, data: { ...environment.data, disabled: true } }])[0].environment.data.disabled, true)
  assert.equal(matchProbeAddress('https://shop.example.test', [environment], { applications: ['another-app'] } as Suite).length, 0)
  assert.equal(matchProbeAddress('https://shop.example.test:8443', [environment]).length, 0)
})
test('Prefix endpoints keep representable paths and leave ambiguous targets for explicit choice', () => {
  const prefixed = { ...environment, id: 'prefix', data: { ...environment.data, endpoints: { api: 'https://shop.example.test/api/' } } }
  assert.equal(matchProbeAddress('https://shop.example.test/api/healthz?x=1', [prefixed])[0].path, '/healthz?x=1')
  assert.equal(matchProbeAddress('https://shop.example.test/api', [prefixed]).length, 0)
  assert.equal(matchProbeAddress('https://shop.example.test/apian', [prefixed]).length, 0)
  assert.equal(matchProbeAddress('https://shop.example.test/api/healthz', [environment, prefixed]).length, 2)
})
