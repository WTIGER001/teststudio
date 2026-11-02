import fs from 'node:fs'
import path from 'node:path'
import crypto from 'node:crypto'
import { execFileSync } from 'node:child_process'

const origin = new URL(process.env.TESTSTUDIO_ORIGIN ?? 'http://127.0.0.1:8040').origin
if (!['127.0.0.1', '[::1]', 'localhost'].includes(new URL(origin).hostname)) throw new Error('The local self-test registrar requires a loopback TestStudio origin')
const image = process.env.TESTSTUDIO_SELFTEST_IMAGE
if (!image || !/^sha256:[a-f0-9]{64}$/.test(image)) throw new Error('Set TESTSTUDIO_SELFTEST_IMAGE to the immutable local image ID')
const diagnostic = process.argv.includes('--diagnostics')
const timeoutProbe = process.argv.includes('--timeout-probe')
const catalog = JSON.parse(fs.readFileSync(new URL('../tests/selftest/catalog.json', import.meta.url), 'utf8'))
class APIError extends Error { constructor(status, message) { super(message); this.status = status } }
let cookie = '', csrf = ''
async function call(method, resource, value, revision) {
  const response = await fetch(origin + resource, { method, headers: {
    Origin: origin, Accept: 'application/json', 'Content-Type': 'application/json', 'X-CSRF-Token': csrf,
    Cookie: cookie, 'Idempotency-Key': crypto.randomUUID(), ...(revision == null ? {} : { 'If-Match': `"${revision}"` })
  }, body: value === undefined ? undefined : JSON.stringify(value) })
  const sessionCookie = response.headers.getSetCookie().find(value => value.startsWith('teststudio_session='))
  if (sessionCookie) cookie = sessionCookie.split(';')[0]
  const body = await response.json()
  if (!response.ok) throw new APIError(response.status, `${method} ${resource}: ${body.message ?? response.status}`)
  return body
}
await call('POST', '/auth/local', {})
csrf = (await call('GET', '/auth/session')).session.csrfToken
const project = process.env.TESTSTUDIO_SELFTEST_PROJECT ?? 'teststudio-selftests'
if (!/^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$/.test(project)) throw new Error('Invalid self-test project identifier')
const suite = diagnostic || timeoutProbe ? 'adapter-diagnostics' : 'browser-workflows'
const base = `/api/v1/projects/${project}`
try { await call('GET', base) }
catch (error) { if (error.status !== 404) throw error; await call('POST', '/api/v1/projects', { id: project, name: 'TestStudio self-tests', description: 'Explicitly registered TestStudio browser verification' }) }
const suiteRows = await call('GET', `${base}/suites`)
const previousSuite = suiteRows.items.find(item => item.id === suite)
const suiteDescription = 'Managed by the TestStudio self-test registrar.'
if (previousSuite && (previousSuite.data.archived || previousSuite.data.requiresTarget || previousSuite.data.description !== suiteDescription)) throw new Error('The existing self-test suite needs review before registration')
if (!previousSuite) await call('PUT', `${base}/suites/${suite}`, { id: suite, name: suite === 'browser-workflows' ? 'TestStudio browser workflows' : 'Runner diagnostics', description: suiteDescription, applications: [], requiresTarget: false }, 0)
const selected = diagnostic ? catalog.diagnostics.filter(test => test.id === 'ADAPTER-FAIL') : timeoutProbe ? catalog.diagnostics.filter(test => test.id === 'ADAPTER-TIMEOUT') : catalog.tests
const existing = (await call('GET', `${base}/suites/${suite}/tests`)).items
const discovered = Object.fromEntries([...catalog.tests, ...catalog.diagnostics].map(test => [test.id, test.selector])), ids = []
for (const definition of selected) {
  discovered[definition.id] = definition.selector
  const old = existing.find(item => item.id === definition.id)
  if (old && old.data.source !== catalog.source) throw new Error(`Test ${definition.id} belongs to another source`)
  const test = {
    ...definition, source: catalog.source, type: 'browser', runner: 'playwright', state: 'enabled', safety: 'safe',
    rationale: 'Runs against a new isolated TestStudio instance with no external network access', description: '', tags: ['selftest'], dependencies: [], requiredBindings: [], timeoutSeconds: timeoutProbe ? 3 : 60,
    capture: { logs: true, screenshots: 'always', trace: 'failure', video: 'off', maxBytes: 104857600, maxFileBytes: 67108864, retentionDays: 7 },
    ...old?.data, source: catalog.source, selector: definition.selector, runner: 'playwright', type: 'browser'
  }
  await call(old ? 'PUT' : 'POST', `${base}/suites/${suite}/tests${old ? '/' + definition.id : ''}`, test, old?.revision)
  if (test.state === 'enabled') ids.push(test.id)
}
if (!ids.length) throw new Error('The self-test selection has no enabled tests')
const artifact = { image, sourceRevision: process.env.TESTSTUDIO_SELFTEST_REVISION ?? image, discoveredSelectors: discovered }
const draft = await call('POST', `${base}/suites/${suite}/catalog-drafts`, artifact)
await call('POST', `${base}/suites/${suite}/catalog-drafts/${draft.id}/publish`, { activate: true }, draft.revision)
const run = await call('POST', `${base}/runs`, { suiteId: suite, testIds: ids, executionLocation: 'local', timeoutSeconds: 600, retries: 0 })
const url = `${origin}/projects/${project}/runs/${run.id}`
console.log(`Self-test run: ${url}`)
const deadline = Date.now() + 610000
let current
while (Date.now() < deadline) {
  current = (await call('GET', `${base}/runs/${run.id}`)).data
  if (['completed', 'cancelled', 'timed-out', 'blocked', 'incomplete'].includes(current.state)) break
  await new Promise(resolve => setTimeout(resolve, 500))
}
if (!current || !['completed', 'cancelled', 'timed-out', 'blocked', 'incomplete'].includes(current.state)) {
  await call('POST', `${base}/runs/${run.id}/cancel`, {})
  throw new Error('Self-test verification exceeded its deadline; cancellation was requested')
}
console.log(`${current.state}: ${current.outcome}; ${current.counts.passed} passed, ${current.counts.failed} failed`)
const attempts = await call('GET', `${base}/runs/${run.id}/attempts`)
const artifacts = await call('GET', `${base}/runs/${run.id}/artifacts`)
const summary = { url, run: current, attempts: attempts.items, artifacts: artifacts.items }
const output = path.resolve('output/selftest', run.id)
fs.mkdirSync(output, { recursive: true })
fs.writeFileSync(path.join(output, 'summary.json'), JSON.stringify(summary, null, 2))
for (const item of artifacts.items) {
  const a = item.data
  if (a.kind !== 'screenshot' && a.kind !== 'trace') continue
  const grant = await call('POST', `${base}/runs/${run.id}/artifacts/${a.id}/access`, { action: 'download' })
  const response = await fetch(grant.url)
  if (!response.ok) throw new Error('Self-test evidence download failed')
  fs.writeFileSync(path.join(output, `${a.id}-${path.basename(a.filename)}`), Buffer.from(await response.arrayBuffer()))
}
console.log(`Local verification output: ${output}`)
const remaining = execFileSync('docker', ['ps', '-a', '--filter', `label=teststudio.run=${run.id}`, '--format', '{{.Names}}'], { encoding: 'utf8' }).trim()
if (remaining) throw new Error('Self-test containers remain after completion: ' + remaining)
if (diagnostic || timeoutProbe) {
  const expected = timeoutProbe ? 'timeout' : 'assertion'
  if (current.state !== 'completed' || current.outcome !== 'failed' || current.results[0]?.failureKind !== expected) throw new Error('The intentional failure did not retain the expected outcome')
  if (!artifacts.items.some(item => item.data.kind === 'trace') || !artifacts.items.some(item => item.data.kind === 'screenshot')) throw new Error('Failure evidence is missing')
} else if (current.state !== 'completed' || current.outcome !== 'passed' || current.evidenceState !== 'complete') {
  console.error(current.results.map(result => `${result.testId}: ${result.state} ${result.error ?? ''}`).join('\n'))
  throw new Error('TestStudio self-tests did not complete successfully')
}
