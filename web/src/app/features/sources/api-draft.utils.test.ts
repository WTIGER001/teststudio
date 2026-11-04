import assert from 'node:assert/strict'
import test from 'node:test'
import { apiPathPrefix, apiStarterIssue, apiStarterProblem, configureAPIStarter, type APIStarterOptions } from './api-draft.utils.ts'

const options = (): APIStarterOptions => ({ title: 'Search orders', id: 'API-SEARCH', method: 'POST', path: '/v1/search?limit=5', prefix: '/v1/search', endpoint: 'api', status: 200, readOnly: true, body: '{"query":"new"}', maxBodyBytes: 1024, maxRequests: 10, maxResponseBytes: 65536 })

test('API wizard validates request policy and identifies invalid Advanced fields', () => {
  assert.equal(apiStarterIssue(options()), '')
  assert.equal(apiPathPrefix('/v1/items/?page=2'), '/v1/items')
  for (const value of [{ ...options(), path: 'https://outside.invalid/' }, { ...options(), path: '/v1/../admin' }, { ...options(), path: '/v1/%2fadmin' }, { ...options(), path: '/v1//admin' }, { ...options(), prefix: '/private' }, { ...options(), body: '{invalid}' }, { ...options(), maxBodyBytes: 1 }, { ...options(), method: 'DELETE' }]) assert.notEqual(apiStarterIssue(value), '')
  assert.equal(apiStarterProblem({ ...options(), endpoint: 'invalid/name' })?.field, 'endpoint')
  assert.equal(apiStarterProblem({ ...options(), prefix: '/private' })?.field, 'prefix')
  assert.notEqual(apiStarterIssue({ ...options(), method: 'GET' }), '')
  assert.equal(apiStarterIssue({ ...options(), method: 'GET', path: '/v1/search/', body: '' }), '')
})

test('API source generation preserves literal data and keeps safety separate from policy', () => {
  const starter = { id: '', name: 'API starter', manifestPath: 'teststudio-suite.json', files: [
    { path: 'checks/api.spec.ts', content: '' },
    { path: 'catalog.json', content: JSON.stringify({ tests: [{ id: 'API-001', title: 'Default', type: 'api', safety: 'needs-review' }] }) },
    { path: 'teststudio-suite.json', content: JSON.stringify({ type: 'api', browser: { endpoint: 'api' } }) }
  ] }
  const value = { ...options(), title: `Search ' "; throw new Error('untrusted'); //`, body: '{"query":"\\\"`literal`"}' }
  const result = configureAPIStarter(starter, value)
  const code = result.files[0].content
  assert.doesNotThrow(() => new Function(code.replace(/^import[^\n]*\n/, '')))
  assert.ok(code.includes(JSON.stringify(value.title)))
  const catalog = JSON.parse(result.files[1].content), manifest = JSON.parse(result.files[2].content)
  assert.equal(catalog.tests[0].id, 'API-SEARCH')
  assert.equal(catalog.tests[0].safety, 'needs-review')
  assert.equal(manifest.browser.requests.rules[0].readOnly, true)
  assert.equal(manifest.browser.requests.rules[0].maxBodyBytes, 1024)
  assert.equal(JSON.parse(starter.files[1].content).tests[0].id, 'API-001')
})
