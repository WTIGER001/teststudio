import test from 'node:test'
import assert from 'node:assert/strict'
import { parseVersion, nextVersion, buildLinkerFlags } from './version.mjs'

test('Display tags preserve canonical semantic release numbers', () => {
  assert.equal(parseVersion('0.1.0-dev').displayVersion, '0.1.0.dev')
  assert.equal(parseVersion('1.2.3').displayVersion, '1.2.3.stable')
  assert.equal(parseVersion('1.2.3-rc1').tag, 'rc1')
  for (const version of ['01.2.3', '1.2', '1.2.3.4', '1.2.3-stable', '1.2.3-Dev', '1.2.3-']) assert.throws(() => parseVersion(version))
})

test('Release bumps reset lower numbers and retain the selected channel', () => {
  assert.equal(nextVersion('0.1.9-dev', 'patch'), '0.1.10-dev')
  assert.equal(nextVersion('0.1.9-dev', 'minor'), '0.2.0-dev')
  assert.equal(nextVersion('0.1.9-dev', 'major'), '1.0.0-dev')
  assert.equal(nextVersion('0.1.9-dev', 'tag', 'beta'), '0.1.9-beta')
  assert.equal(nextVersion('0.1.9-dev', 'tag', 'stable'), '0.1.9')
  assert.throws(() => nextVersion('0.1.0-dev', 'other'))
})

test('The Go linker receives the same complete manifest without shell escaping', () => {
  const info = { version: '0.1.0-dev', buildId: '0123456789ab', builtAt: '2026-11-06T14:00:00.000Z', modified: true }
  const value = buildLinkerFlags(info).split('Manifest=')[1]
  assert.deepEqual(JSON.parse(Buffer.from(value, 'base64').toString('utf8')), info)
})
