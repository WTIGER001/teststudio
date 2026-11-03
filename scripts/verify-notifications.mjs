import fs from 'node:fs'
import path from 'node:path'
import http from 'node:http'
import net from 'node:net'
import crypto from 'node:crypto'
import { spawn } from 'node:child_process'
import { once } from 'node:events'

const root = process.env.TESTSTUDIO_NOTIFICATION_VERIFY_ROOT
if (!root || !fs.existsSync(path.join(root, 'teststudio'))) throw new Error('Run scripts/test-notifications.sh to create an isolated verification installation')
const project = 'notification-verification', token = crypto.randomBytes(24).toString('hex')
const received = [], counts = new Map()
let receiverError = false
let interrupted = false, controller, exit
process.on('SIGINT', () => { interrupted = true })
process.on('SIGTERM', () => { interrupted = true })
const pause = ms => new Promise(resolve => setTimeout(resolve, ms))
const receiver = http.createServer(async (request, response) => {
  if (request.method !== 'POST' || request.url !== `/receive/${token}`) { response.writeHead(404); response.end(); return }
  try {
  let text = ''
  for await (const chunk of request) { text += chunk; if (text.length > 16384) { response.writeHead(413); response.end(); return } }
  const message = JSON.parse(text), id = request.headers['x-teststudio-delivery-id']
  if (id !== message.deliveryId) { response.writeHead(400); response.end(); return }
  const attempt = (counts.get(id) ?? 0) + 1
  counts.set(id, attempt); received.push(message)
  response.writeHead(attempt === 1 ? 503 : 204, attempt === 1 ? { 'Retry-After': '1' } : {})
  response.end()
  } catch { receiverError = true; if (!response.headersSent) response.writeHead(500); response.end() }
})
async function port() {
  const server = net.createServer()
  server.listen(0, '127.0.0.1'); await once(server, 'listening')
  const value = server.address().port
  await new Promise(resolve => server.close(resolve))
  return value
}
let origin, cookie = '', csrf = ''
async function call(method, resource, body, revision) {
  if (interrupted) throw new Error('Notification verification was interrupted')
  const response = await fetch(origin + resource, { method, signal: AbortSignal.timeout(5000), headers: { Origin: origin, Cookie: cookie, 'Content-Type': 'application/json', 'X-CSRF-Token': csrf, 'Idempotency-Key': crypto.randomUUID(), ...(revision == null ? {} : { 'If-Match': `"${revision}"` }) }, body: body === undefined ? undefined : JSON.stringify(body) })
  const sessionCookie = response.headers.getSetCookie().find(value => value.startsWith('teststudio_session='))
  if (sessionCookie) cookie = sessionCookie.split(';')[0]
  const value = await response.json()
  if (!response.ok) throw new Error(`${method} ${resource}: ${value.message ?? response.status}`)
  return value
}
async function poll(operation, accept, label, limit = 30000) {
  const deadline = Date.now() + limit
  while (Date.now() < deadline) {
    if (interrupted) throw new Error('Notification verification was interrupted')
    if (receiverError) throw new Error('The local notification receiver rejected an invalid message')
    const value = await operation()
    if (accept(value)) return value
    await pause(100)
  }
  throw new Error(`${label} exceeded its verification deadline`)
}
try {
  receiver.listen(0, '127.0.0.1'); await once(receiver, 'listening')
  const secret = `http://127.0.0.1:${receiver.address().port}/receive/${token}`
  const apiPort = await port(); let artifactPort = await port(); while (artifactPort === apiPort) artifactPort = await port()
  origin = `http://127.0.0.1:${apiPort}`
  const bindings = path.join(root, 'bindings.json')
  fs.writeFileSync(bindings, JSON.stringify([{ id: 'receiver', name: 'Local verification receiver', projectId: project, purpose: 'notification-webhook', environmentVariable: 'TESTSTUDIO_VERIFY_NOTIFICATION_URL', network: 'loopback' }]), { mode: 0o600 })
  const log = fs.openSync(path.join(root, 'server.log'), 'a', 0o600)
  const environment = { PATH: process.env.PATH ?? '', TESTSTUDIO_VERIFY_NOTIFICATION_URL: secret }
  if (process.env.TMPDIR) environment.TMPDIR = process.env.TMPDIR
  controller = spawn(path.join(root, 'teststudio'), ['--local', '--addr', `127.0.0.1:${apiPort}`, '--public-url', origin, '--artifact-addr', `127.0.0.1:${artifactPort}`, '--artifact-origin', `http://127.0.0.1:${artifactPort}`, '--database', path.join(root, 'teststudio.db'), '--artifact-directory', path.join(root, 'artifacts'), '--secret-bindings', bindings], { env: environment, stdio: ['ignore', log, log] })
  fs.closeSync(log); exit = once(controller, 'exit')
  await poll(async () => { if (controller.exitCode != null) throw new Error('The verification service stopped before readiness'); try { return (await fetch(origin + '/healthz', { signal: AbortSignal.timeout(1000) })).ok } catch { return false } }, Boolean, 'Service startup')
  await call('POST', '/auth/local', {}); csrf = (await call('GET', '/auth/session')).session.csrfToken
  const base = `/api/v1/projects/${project}`
  await call('POST', '/api/v1/projects', { id: project, name: 'Notification verification' })
  await call('PUT', `${base}/applications/app`, { id: 'app', name: 'Local TestStudio' }, 0)
  await call('PUT', `${base}/environments/local`, { id: 'local', name: 'Local verification instance', applicationId: 'app', stage: 'local', endpoints: { api: origin } }, 0)
  await call('PUT', `${base}/suites/health`, { id: 'health', name: 'Local checks', applications: ['app'], requiresTarget: true }, 0)
  await call('POST', `${base}/suites/health/tests`, { id: 'HEALTH-001', title: 'Local availability', category: 'Availability', applicationId: 'app', type: 'https', runner: 'http', state: 'enabled', safety: 'safe', rationale: 'Reads this newly created local verification instance', description: '', tags: [], dependencies: [], requiredBindings: [], timeoutSeconds: 3, check: { endpoint: 'api', path: '/healthz', method: 'GET', statusCodes: [200] } })
  const draft = await call('POST', `${base}/suites/health/catalog-drafts`, {})
  await call('POST', `${base}/suites/health/catalog-drafts/${draft.id}/publish`, { activate: true }, draft.revision)
  await call('POST', `${base}/notifications/channels`, { id: 'availability', name: 'Availability delivery', enabled: true, kind: 'webhook', secretReference: 'receiver', outcomes: ['passed'], maxAttempts: 3 })
  const request = { suiteId: 'health', environmentId: 'local', timeoutSeconds: 30 }
  const manual = await call('POST', `${base}/runs`, request)
  const definition = { id: 'probe', name: 'Scheduled availability', enabled: true, kind: 'interval', intervalSeconds: 5, timezone: 'UTC', overlap: 'skip', request }
  await call('POST', `${base}/schedules`, definition)
  const fired = await poll(() => call('GET', `${base}/schedules/probe`), row => !!row.data.lastRunId, 'Scheduled run')
  await call('PUT', `${base}/schedules/probe`, { ...definition, enabled: false }, fired.revision)
  const ids = [manual.id, fired.data.lastRunId]
  for (const id of ids) await poll(() => call('GET', `${base}/runs/${id}`), row => row.data.state === 'completed', 'Normal run completion')
  const deliveries = await poll(() => call('GET', `${base}/notifications/deliveries`), page => page.items.length === 2 && page.items.every(row => row.data.state === 'delivered'), 'Durable notification retries')
  for (const row of deliveries.items) {
    const delivery = row.data
    if (!ids.includes(delivery.runId) || delivery.attempts !== 2 || delivery.message.counts.passed !== 1 || delivery.message.counts.total !== 1 || delivery.message.url !== `${origin}/projects/${project}/runs/${delivery.runId}`) throw new Error('Delivery lost normal run identity, counts, or retry history')
    const history = await call('GET', `${base}/notifications/deliveries/${row.id}/attempts`)
    if (history.items.length !== 2 || history.items[1].data.httpStatus !== 503) throw new Error('The failed delivery attempt was not retained')
  }
  const triggers = new Set(deliveries.items.map(row => row.data.message.trigger))
  if (!triggers.has('manual') || !triggers.has('schedule') || counts.size !== 2 || received.length !== 4 || [...counts.values()].some(n => n !== 2)) throw new Error('Manual and scheduled notifications did not use one stable delivery path')
  const references = await call('GET', `${base}/notifications/references`), channels = await call('GET', `${base}/notifications/channels`)
  if (JSON.stringify({ deliveries, references, channels }).includes(token)) throw new Error('A resolved webhook secret entered API data')
  const output = path.resolve('output/notifications', manual.id)
  fs.mkdirSync(output, { recursive: true }); fs.writeFileSync(path.join(output, 'summary.json'), JSON.stringify({ origin, runs: ids, deliveries: deliveries.items, received }, null, 2))
  console.log('Manual and scheduled notifications passed with retained retries and stable delivery IDs.')
  console.log(`Local verification output: ${output}`)
  console.log(`Retained installation: ${root}`)
} finally {
  if (controller && controller.exitCode == null && controller.signalCode == null) {
    controller.kill('SIGTERM')
    const ended = await Promise.race([exit.then(() => true), new Promise(resolve => { const timer = setTimeout(() => resolve(false), 10000); timer.unref() })])
    if (!ended) { controller.kill('SIGKILL'); await exit }
  }
  receiver.closeAllConnections()
  await new Promise(resolve => receiver.close(resolve))
}
