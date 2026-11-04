import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import net from 'node:net'
import { spawn } from 'node:child_process'
import { once } from 'node:events'
import { fileURLToPath } from 'node:url'
import { demoAPI } from './lib/demo-api.mjs'

const root = fileURLToPath(new URL('..', import.meta.url))
const binary = path.resolve(root, process.argv[2] ?? '.data/selftest-build/controller')
if (process.argv.length > 3 || !fs.statSync(binary).isFile()) throw new Error('Provide one built TestStudio server path')
const output = path.join(root, 'output/empty-installation')
fs.mkdirSync(output, { recursive: true })
const directory = fs.mkdtempSync(path.join(output, 'local-'))
fs.chmodSync(directory, 0o700)
const summary = { localOnly: true, directory, checks: [], processes: [], cleanup: false }
let controller, exited, interrupted = false
const pause = ms => new Promise(resolve => setTimeout(resolve, ms))
const interrupt = () => { interrupted = true; controller?.kill('SIGTERM') }
process.once('SIGINT', interrupt)
process.once('SIGTERM', interrupt)

async function port(excluded) {
  for (;;) {
    const socket = net.createServer()
    socket.listen(0, '127.0.0.1')
    await once(socket, 'listening')
    const value = socket.address().port
    await new Promise(resolve => socket.close(resolve))
    if (value !== excluded) return value
  }
}
const apiPort = await port(), artifactPort = await port(apiPort)
const origin = `http://127.0.0.1:${apiPort}`
async function stop() {
  if (!controller) return
  if (controller.exitCode === null && controller.signalCode === null) controller.kill('SIGTERM')
  let timer
  try {
    const clean = await Promise.race([exited.then(() => true), new Promise(resolve => { timer = setTimeout(() => resolve(false), 10000) })])
    if (!clean) { controller.kill('SIGKILL'); await exited }
  } finally { clearTimeout(timer) }
  assert.ok(controller.exitCode !== null || controller.signalCode !== null, 'Owned server must stop')
  controller = null
}
async function start() {
  if (interrupted) throw new Error('Verification interrupted')
  const log = fs.openSync(path.join(directory, 'server.log'), 'a', 0o600)
  const env = Object.fromEntries(['PATH', 'HOME', 'TMPDIR', 'TEMP', 'TMP'].filter(key => process.env[key] !== undefined).map(key => [key, process.env[key]]))
  try {
    controller = spawn(binary, ['--local', '--scheduler=false', '--notification-worker=false', '--addr', `127.0.0.1:${apiPort}`, '--public-url', origin, '--artifact-addr', `127.0.0.1:${artifactPort}`, '--artifact-origin', `http://127.0.0.1:${artifactPort}`, '--database', path.join(directory, 'teststudio.db'), '--artifact-directory', path.join(directory, 'artifacts'), '--ui', path.join(directory, 'ui')], { cwd: root, env, stdio: ['ignore', log, log] })
  } finally { fs.closeSync(log) }
  exited = once(controller, 'exit')
  summary.processes.push(controller.pid)
  for (let attempt = 0; attempt < 100; attempt++) {
    if (interrupted || controller.exitCode !== null || controller.signalCode !== null) throw new Error('Owned server stops before readiness')
    try {
      if ((await fetch(origin + '/healthz', { signal: AbortSignal.timeout(500) })).ok) return
    } catch {}
    await pause(100)
  }
  throw new Error('Owned server startup exceeds ten seconds')
}
async function emptyProjects() {
  const api = await demoAPI(origin)
  try { assert.equal((await api.call('GET', '/api/v1/projects')).items.length, 0) }
  finally { await api.logout() }
}
const project = '/api/v1/projects/selected-examples', suite = `${project}/suites/availability`
try {
  await start()
  await emptyProjects()
  summary.checks.push('Fresh normal startup creates no projects')
  await stop()
  await start()
  await emptyProjects()
  summary.checks.push('Restart does not seed examples')
  const api = await demoAPI(origin)
  try {
    await api.call('POST', '/api/v1/projects', { id: 'selected-examples', name: 'Selected local examples' })
    for (const resource of ['applications', 'environments', 'suites', 'runs', 'schedules']) assert.equal((await api.call('GET', `${project}/${resource}`)).items.length, 0, `${resource} starts empty`)
    await api.call('PUT', `${project}/applications/local-service`, { id: 'local-service', name: 'Local service' }, 0)
    await api.call('PUT', `${project}/environments/local`, { id: 'local', name: 'Local', applicationId: 'local-service', stage: 'local', endpoints: { api: origin } }, 0)
    await api.call('PUT', suite, { id: 'availability', name: 'Selected availability example', applications: ['local-service'], requiresTarget: true }, 0)
    assert.equal((await api.call('GET', `${suite}/tests`)).items.length, 0)
    const document = JSON.parse(fs.readFileSync(path.join(root, 'examples/http/catalog.json'), 'utf8'))
    const selected = { ...document, tests: document.tests.filter(test => test.id === 'HTTP-HEALTH-001') }
    assert.equal(selected.tests.length, 1)
    const preview = await api.call('POST', `${suite}/source-uploads`, { name: 'selected-http.json', format: 'json', content: JSON.stringify(selected) })
    assert.equal((await api.call('GET', `${suite}/tests`)).items.length, 0, 'Preview must not install definitions')
    await api.call('POST', `${suite}/source-uploads/${preview.upload.id}/publish`, { activate: true }, preview.upload.revision)
    const tests = (await api.call('GET', `${suite}/tests`)).items
    assert.deepEqual(tests.map(row => row.id), ['HTTP-HEALTH-001'])
    assert.equal((await api.call('GET', `${project}/runs`)).items.length, 0, 'Publication must not run examples')
    summary.checks.push('New project registries start empty', 'Only the selected HTTP example publishes', 'Preview and publication create no run')
  } finally { await api.logout() }
  await stop()
  await start()
  const restored = await demoAPI(origin)
  try {
    assert.deepEqual((await restored.call('GET', '/api/v1/projects')).items.map(row => row.id), ['selected-examples'])
    assert.deepEqual((await restored.call('GET', `${suite}/tests`)).items.map(row => row.id), ['HTTP-HEALTH-001'])
    assert.equal((await restored.call('GET', `${project}/runs`)).items.length, 0)
  } finally { await restored.logout() }
  summary.checks.push('Restart preserves exactly the selected definitions without demo or self-test records')
  summary.passed = true
} finally {
  await stop()
  for (const pid of summary.processes) assert.throws(() => process.kill(pid, 0), { code: 'ESRCH' })
  summary.cleanup = true
  fs.writeFileSync(path.join(directory, 'summary.json'), JSON.stringify(summary, null, 2) + '\n', { mode: 0o600 })
  process.off('SIGINT', interrupt)
  process.off('SIGTERM', interrupt)
}
console.log(`Empty startup and selected example import pass. Owned servers stop. Evidence: ${directory}`)
