import assert from 'node:assert/strict'
import { test } from 'node:test'
import { privateTraceURL } from './trace-viewer.utils'

test('private viewer addresses stay on their artifact origin and grant scope', () => {
  const token = 'a'.repeat(43), artifact = `https://evidence.example.test/access/${token}`
  const url = new URL(`https://evidence.example.test/trace/${token}/index.html`)
  url.searchParams.set('trace', artifact); url.searchParams.set('expires', '2030-01-01T00:00:00Z')
  assert.equal(privateTraceURL(url.href, artifact), url.href)
  assert.equal(privateTraceURL(undefined, artifact), '')
  for (const change of [(value: URL) => { value.hostname = 'external.test' }, (value: URL) => { value.pathname = '/trace/other/index.html' }, (value: URL) => { value.searchParams.set('trace', 'https://external.test/trace.zip') }]) {
    const value = new URL(url); change(value)
    assert.throws(() => privateTraceURL(value.href, artifact))
  }
})
