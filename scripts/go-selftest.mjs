import fs from 'node:fs'
import path from 'node:path'
import crypto from 'node:crypto'
import { execFileSync } from 'node:child_process'

const origin = new URL(process.env.TESTSTUDIO_ORIGIN ?? 'http://127.0.0.1:8040').origin
if (!['127.0.0.1', '[::1]', 'localhost'].includes(new URL(origin).hostname)) throw new Error('Go self-tests require a loopback controller')
const image = process.env.TESTSTUDIO_GO_IMAGE
if (!/^sha256:[a-f0-9]{64}$/.test(image ?? '')) throw new Error('Set TESTSTUDIO_GO_IMAGE to the immutable local image ID')
const manifest = JSON.parse(fs.readFileSync(new URL('../tests/selftest/go/teststudio-go.json', import.meta.url), 'utf8'))
let cookie = '', csrf = ''
async function call(method, resource, value, revision) {
  const response = await fetch(origin + resource, { method, headers: { Origin: origin, Cookie: cookie, 'Content-Type': 'application/json', 'X-CSRF-Token': csrf, 'Idempotency-Key': crypto.randomUUID(), ...(revision == null ? {} : { 'If-Match': `"${revision}"` }) }, body: value === undefined ? undefined : JSON.stringify(value) })
  const sessionCookie = response.headers.getSetCookie().find(value => value.startsWith('teststudio_session='))
  if (sessionCookie) cookie = sessionCookie.split(';')[0]
  const result = await response.json()
  if (!response.ok) throw new Error(`${method} ${resource}: ${result.message ?? response.status}`)
  return result
}
await call('POST', '/auth/local', {})
csrf = (await call('GET', '/auth/session')).session.csrfToken
const project = 'teststudio-go-selftests', suite = 'go-source', base = `/api/v1/projects/${project}`
await call('POST', '/api/v1/projects', { id: project, name: 'TestStudio Go self-tests', description: 'New isolated source checks and runner diagnostics' })
await call('PUT', `${base}/suites/${suite}`, { id: suite, name: 'Go source checks', applications: [], requiresTarget: false }, 0)
for (const entry of manifest.tests) {
  await call('POST', `${base}/suites/${suite}/tests`, {
    id: entry.id, title: entry.test, description: 'New Go runner verification', category: entry.package === './unit' ? 'unit' : 'diagnostics', tags: ['selftest'], runner: 'go', type: 'unit', state: 'enabled',
    safety: 'safe', rationale: 'Uses no target, network, credentials, or host directories', source: 'teststudio-go-selftests', selector: `${entry.package}::${entry.test}`, dependencies: [], requiredBindings: [], timeoutSeconds: 1,
    capture: { logs: true, screenshots: 'off', trace: 'off', video: 'off', maxBytes: 10485760, maxFileBytes: 4194304, retentionDays: 7 }
  })
}
const draft = await call('POST', `${base}/suites/${suite}/catalog-drafts`, { image, sourceRevision: image, discoveredSelectors: Object.fromEntries(manifest.tests.map(e => [e.id, `${e.package}::${e.test}`])) })
await call('POST', `${base}/suites/${suite}/catalog-drafts/${draft.id}/publish`, { activate: true }, draft.revision)
const verification = []
const terminal = ['completed', 'blocked', 'incomplete', 'cancelled', 'timed-out']
async function finish(run) {
  const deadline = Date.now() + 180000
  let current
  while (Date.now() < deadline) {
    current = (await call('GET', `${base}/runs/${run.id}`)).data
    if (terminal.includes(current.state)) break
    await new Promise(resolve => setTimeout(resolve, 250))
  }
  if (!current || !terminal.includes(current.state)) {
    await call('POST', `${base}/runs/${run.id}/cancel`, {})
    throw new Error('Go verification exceeded its deadline; cancellation was requested')
  }
  const attempts = (await call('GET', `${base}/runs/${run.id}/attempts`)).items
  const artifacts = (await call('GET', `${base}/runs/${run.id}/artifacts`)).items
  const executions = (await call('GET', `${base}/runs/${run.id}/executions`)).items
  if (current.evidenceState !== 'complete' || !executions.length || executions.some(e => e.data.state !== 'cleaned' || !e.data.resultSaved || !e.data.evidenceComplete)) throw new Error('Go observation or resource cleanup remains incomplete')
  for (const kind of ['container', 'volume']) {
    const args = kind === 'container' ? ['ps', '-a'] : ['volume', 'ls']
    const remaining = execFileSync('docker', [...args, '--filter', `label=teststudio.run=${run.id}`, '--format', kind === 'container' ? '{{.Names}}' : '{{.Name}}'], { encoding: 'utf8' }).trim()
    if (remaining) throw new Error(`Go ${kind} resources remain after completion`)
  }
  for (const attempt of attempts) {
    if (!attempt.data.go?.toolchain || !attempt.data.timing || attempt.data.timing.baseline.context.runner !== 'go') throw new Error('Go diagnostics or timing contract is missing')
  }
  if (!artifacts.some(a => a.data.kind === 'log' && a.data.stream.startsWith('go.'))) throw new Error('Go evidence is missing')
  const summary = { url: `${origin}/projects/${project}/runs/${run.id}`, run: current, attempts, artifacts, executions }
  verification.push(summary)
  console.log(`${current.id}: ${current.state} / ${current.outcome}; ${attempts.length} attempts`)
  return summary
}
async function execute(ids, retries = 0) {
  const request = { suiteId: suite, testIds: ids, executionLocation: 'local', timeoutSeconds: 180, retries }
  const estimate = await call('POST', `${base}/runs/estimate`, request)
  if (estimate.totalTests !== ids.length) throw new Error('Go estimate inflated logical counts')
  return finish(await call('POST', `${base}/runs`, request))
}
const normal = await execute(['GO-SUM-001', 'GO-ROWS-001', 'GO-CHILD-001', 'GO-CHILD-002'])
if (normal.run.state !== 'completed' || normal.run.counts.passed !== 4 || normal.run.counts.total !== 4 || normal.attempts.length !== 4 || normal.attempts.find(a => a.data.testId === 'GO-ROWS-001')?.data.steps.length !== 3) throw new Error('Go subtest aggregates inflated counts or lost steps')
const learned = await call('POST', `${base}/runs/estimate`, { suiteId: suite, testIds: ['GO-SUM-001'], executionLocation: 'local', timeoutSeconds: 180 })
if (learned.tests['GO-SUM-001'].sampleCount !== 1 || !learned.tests['GO-SUM-001'].context.profile.includes('go1.26.8-linux-')) throw new Error('Clean Go timing did not train its runtime baseline')
for (const [id, state, kind, retries] of [
  ['GO-FAIL-001', 'failed', 'assertion', 1], ['GO-PANIC-001', 'failed', 'panic', 0], ['GO-TIMEOUT-001', 'timed-out', 'timeout', 1],
  ['GO-SKIP-001', 'skipped', '', 2], ['GO-MISSING-001', 'unknown', 'discovery', 2], ['GO-BUILD-001', 'unknown', 'build', 2]
]) {
  const summary = await execute([id], retries)
  const expectedAttempts = state === 'unknown' || state === 'skipped' ? 1 : retries + 1
  if (summary.attempts.length !== expectedAttempts || summary.attempts.some(a => a.data.state !== state || (a.data.failureKind ?? '') !== kind)) throw new Error(`Wrong Go diagnostic classification for ${id}`)
  if (state === 'unknown' && (summary.run.state !== 'blocked' || summary.run.counts.failed !== 0 || summary.attempts.some(a => !a.data.durationUnavailable || a.data.durationMs !== 0))) throw new Error('Go preparation became an assertion failure or duration sample')
  if (state === 'timed-out') {
    if (summary.run.counts.timedOut !== 1 || summary.run.counts.failed !== 0) throw new Error('Go timeout was counted as an assertion failure')
    const child = await finish(await call('POST', `${base}/runs/${summary.run.id}/rerun-failures`, {}))
    if (child.run.request.parentRunId !== summary.run.id || child.run.release.id !== summary.run.release.id || child.run.counts.timedOut !== 1) throw new Error('Failure rerun lost the Go timeout selection or release')
  }
}
const output = path.resolve('output/go-selftest', normal.run.id)
fs.mkdirSync(output, { recursive: true })
fs.writeFileSync(path.join(output, 'summary.json'), JSON.stringify(verification, null, 2))
console.log(`Go verification output: ${output}`)
