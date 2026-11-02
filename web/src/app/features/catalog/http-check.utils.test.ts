import assert from 'node:assert/strict'
import { test } from 'node:test'
import { httpCheckForm, parseHTTPCheck } from './http-check.utils.ts'
import type { HTTPCheck } from './catalog.models'

test('HTTP editor preserves structured assertions and exact body text through edits', () => {
  const original: HTTPCheck = { endpoint: 'api', path: '/health?mode=ready', method: 'GET', statusCodes: [200, 204], requiredText: ' up ', forbiddenText: 'down', headers: { 'X-Ready': 'yes' }, json: { 'items.0.id': 1, enabled: true, metadata: null, tags: ['health'] }, maxResponseMs: 500, minCertificateDays: 14 }
  assert.deepEqual(parseHTTPCheck(httpCheckForm(original)).check, original)
  const parsed = parseHTTPCheck({ ...httpCheckForm(), statusCodes: '200, 204 200' })
  assert.deepEqual(parsed.check?.statusCodes, [200, 204])
})

test('HEAD conflicts remain visible and do not discard body assertions', () => {
  const form = { ...httpCheckForm(), method: 'HEAD', requiredText: 'up', json: '{"status":"up"}' }
  assert.match(parseHTTPCheck(form).errors.method!, /HEAD cannot assert/)
  assert.equal(form.requiredText, 'up')
  assert.equal(parseHTTPCheck({ ...form, requiredText: '', json: '' }).check?.method, 'HEAD')
})

test('invalid statuses, headers, JSON paths, and thresholds cannot create executable checks', () => {
  for (const change of [
    { statusCodes: '200.5' }, { statusCodes: '' }, { statusCodes: '600' }, { path: '//other/path' }, { path: '/health#fragment' },
    { headers: '{"X-Ready":true}' }, { headers: '{"X-Ready":"yes","x-ready":"no"}' }, { headers: '{"Invalid Name":"value"}' },
    { json: '[]' }, { json: '{bad' }, { json: '{"status..value":1}' }, { maxResponseMs: -1 }, { maxResponseMs: 0.5 }, { minCertificateDays: 36501 }
  ]) assert.equal(parseHTTPCheck({ ...httpCheckForm(), ...change }).check, undefined)
  assert.ok(parseHTTPCheck({ ...httpCheckForm(), requiredText: 'x'.repeat(8193) }).errors.requiredText)
  assert.ok(parseHTTPCheck({ ...httpCheckForm(), json: JSON.stringify({ value: 'x'.repeat(65536) }) }).errors.json)
})
