import { test } from 'node:test'
import assert from 'node:assert/strict'
import { randomId } from './random-id'

test('Identifiers retain secure UUID generation when the browser exposes it', () => {
  assert.equal(randomId({ randomUUID: () => '12345678-1234-4567-8123-123456789abc', getRandomValues: () => { throw new Error('Unexpected fallback') } }), '12345678-1234-4567-8123-123456789abc')
})
test('Browsers without randomUUID receive version 4 identifiers from secure random bytes', () => {
  const source: Pick<Crypto, 'getRandomValues'> = { getRandomValues: value => { if (!(value instanceof Uint8Array)) throw new Error('Expected random bytes'); value.fill(255); return value } }
  assert.equal(randomId(source), 'ffffffff-ffff-4fff-bfff-ffffffffffff')
  const generated = new Set(Array.from({ length: 20 }, () => randomId({ getRandomValues: value => crypto.getRandomValues(value) })))
  assert.equal(generated.size, 20)
  for (const value of generated) assert.match(value, /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/)
})
test('Missing secure randomness fails explicitly without a weak random fallback', () => {
  assert.throws(() => randomId({} as Crypto), /secure random identifiers/)
})
