import { test } from 'node:test'
import assert from 'node:assert/strict'
import { creationMatches, suggestedIdentifier, validWebAddress } from './creation.utils'

test('Generated identifiers handle accents, non-Latin names, and length limits', () => {
  assert.equal(suggestedIdentifier('Café status', 'abcd1234'), 'cafe-status-abcd1234')
  assert.equal(suggestedIdentifier('測試', 'abcd1234'), 'item-abcd1234')
  assert.equal(suggestedIdentifier('   ', 'abcd1234'), '')
  assert.match(suggestedIdentifier('x'.repeat(200), 'abcd1234'), /^[a-z0-9-]{1,128}$/)
})
test('Uncertain saves match server defaults without accepting changed content', () => {
  assert.equal(creationMatches({ name: 'Example', archived: false, endpoints: { api: 'https://a.test' } }, { name: 'Example', endpoints: { api: 'https://a.test' }, created: true }), true)
  assert.equal(creationMatches({ endpoints: { api: 'https://a.test' } }, { endpoints: { api: 'https://b.test' } }), false)
  assert.equal(creationMatches({ items: ['one'] }, { items: ['one', 'two'] }), false)
  assert.equal(creationMatches({ items: [] }, null), false)
})
test('Target addresses allow HTTP and HTTPS without embedded credentials', () => {
  for (const address of ['http://localhost:8050', 'https://app.example.test/base']) assert.equal(validWebAddress(address), true)
  for (const address of ['ftp://example.test', 'https://example.test/#fragment', 'https://name:password@example.test', 'javascript:alert(1)', 'https://exam\nple.test']) assert.equal(validWebAddress(address), false)
})
