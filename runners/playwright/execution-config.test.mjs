import assert from 'node:assert/strict'
import fs from 'node:fs/promises'
import os from 'node:os'
import path from 'node:path'
import { pathToFileURL } from 'node:url'
import { test } from 'node:test'
import { executionConfig } from './execution-config.mjs'

async function prepared(t, suite, capture, overrides, options) {
  const directory = await fs.mkdtemp(path.join(os.tmpdir(), 'teststudio-execution-config-'))
  t.after(() => fs.rm(directory, { recursive: true, force: true }))
  const source = path.join(directory, 'suite.mjs')
  await fs.writeFile(source, `export default ${JSON.stringify(suite)}`)
  const config = executionConfig({ config: source }, capture, overrides, { ...options, filename: path.join(directory, 'execution.mjs') })
  return { directory, config }
}

test('execution uses writable output and declared capture across projects', async t => {
  const { directory, config } = await prepared(t, { testDir: 'checks', outputDir: './read-only', use: { locale: 'en-US', trace: 'off' }, projects: [{ name: 'chromium', testDir: 'browser', outputDir: './also-read-only', use: { screenshot: 'off' } }] }, { screenshots: 'always', trace: 'failure', video: 'off' })
  const suite = (await import(pathToFileURL(config))).default
  assert.equal(suite.testDir, path.join(directory, 'checks'))
  assert.equal(suite.outputDir, '/output/playwright-results')
  assert.equal(suite.projects[0].outputDir, '/output/playwright-results')
  assert.equal(suite.projects[0].testDir, path.join(directory, 'browser'))
  assert.deepEqual(suite.projects[0].use, { locale: 'en-US', trace: 'on', screenshot: 'on', video: 'off' })
})

test('restricted API execution removes suite proxy and disables opaque media', async t => {
  const { config } = await prepared(t, { use: { proxy: { server: 'http://unapproved.invalid' } } }, { screenshots: 'always', trace: 'always', video: 'always' }, { baseURL: 'http://127.0.0.1:1234' }, { restricted: true, environment: true })
  const suite = (await import(pathToFileURL(config))).default
  assert.equal(suite.use.proxy, undefined)
  assert.equal(suite.projects[0].use.proxy, undefined)
  assert.deepEqual(suite.use, { screenshot: 'off', trace: 'off', video: 'off', baseURL: 'http://127.0.0.1:1234' })
})

test('registered environment execution cannot start an uploaded web server', async t => {
  const { config } = await prepared(t, { webServer: { command: 'untrusted-command' } }, { screenshots: 'off', trace: 'off', video: 'off' }, {}, { environment: true })
  await assert.rejects(import(pathToFileURL(config)), /cannot start a suite web server/)
})

test('relocated configuration preserves source hooks and server working directories', async t => {
  const { directory, config } = await prepared(t, { globalSetup: './setup.mjs', globalTeardown: ['./teardown.mjs'], tsconfig: './tsconfig.json', snapshotDir: 'snapshots', webServer: { command: 'not-executed', cwd: 'tools' }, projects: [{ name: 'chromium', snapshotDir: 'browser-snapshots' }] }, { screenshots: 'off', trace: 'off', video: 'off' })
  await fs.writeFile(path.join(directory, 'setup.mjs'), 'export default () => "source-hook"')
  const suite = (await import(pathToFileURL(config))).default
  assert.equal(suite.globalSetup, path.join(directory, 'setup.mjs'))
  assert.deepEqual(suite.globalTeardown, [path.join(directory, 'teardown.mjs')])
  assert.equal(suite.tsconfig, path.join(directory, 'tsconfig.json'))
  assert.equal(suite.snapshotDir, path.join(directory, 'snapshots'))
  assert.equal(suite.projects[0].snapshotDir, path.join(directory, 'browser-snapshots'))
  assert.equal(suite.webServer.cwd, path.join(directory, 'tools'))
  assert.equal((await import(pathToFileURL(suite.globalSetup))).default(), 'source-hook')
})
