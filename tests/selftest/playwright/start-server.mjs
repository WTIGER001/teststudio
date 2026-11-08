import { createServer } from 'node:http'
import { execFile, spawn } from 'node:child_process'
import { randomBytes } from 'node:crypto'
import { access, writeFile, unlink } from 'node:fs/promises'
import { createOIDCFixture } from './oidc-fixture.mjs'

const projectImagesWorkflow = JSON.parse(process.env.TESTSTUDIO_WORK_ITEM ?? '{}').test?.selector === '@STUDIO-PROJECTS-001'
const publicStatusWorkflow = JSON.parse(process.env.TESTSTUDIO_WORK_ITEM ?? '{}').test?.selector === '@STUDIO-STATUS-PAGES-001'
const historyWorkflow = JSON.parse(process.env.TESTSTUDIO_WORK_ITEM ?? '{}').test?.selector === '@STUDIO-HISTORY-001'
const apiAuthoringWorkflow = JSON.parse(process.env.TESTSTUDIO_WORK_ITEM ?? '{}').test?.selector === '@STUDIO-API-AUTHOR-001'
const privateTraceWorkflow = JSON.parse(process.env.TESTSTUDIO_WORK_ITEM ?? '{}').test?.selector === '@STUDIO-TRACE-001'
const diagnosticsWorkflow = ['@STUDIO-DIAGNOSTICS-001', '@STUDIO-TRACE-001'].includes(JSON.parse(process.env.TESTSTUDIO_WORK_ITEM ?? '{}').test?.selector)
const recordingWorkflow = JSON.parse(process.env.TESTSTUDIO_WORK_ITEM ?? '{}').test?.selector === '@STUDIO-CREATE-001'
const repositoryWorkflow = JSON.parse(process.env.TESTSTUDIO_WORK_ITEM ?? '{}').test?.selector === '@STUDIO-GIT-001'
const bundleWorkflow = ['@STUDIO-BUNDLES-001', '@STUDIO-EDITOR-001', '@STUDIO-CREATE-001', '@STUDIO-API-AUTHOR-001'].includes(JSON.parse(process.env.TESTSTUDIO_WORK_ITEM ?? '{}').test?.selector)
const linkingWorkflow = JSON.parse(process.env.TESTSTUDIO_WORK_ITEM ?? '{}').test?.selector === '@STUDIO-LINKS-001'
const localWorkflow = JSON.parse(process.env.TESTSTUDIO_WORK_ITEM ?? '{}').test?.selector === '@STUDIO-LOCAL-001'
const cliWorkflow = JSON.parse(process.env.TESTSTUDIO_WORK_ITEM ?? '{}').test?.selector === '@STUDIO-CLI-001'
const providerWorkflow = cliWorkflow || JSON.parse(process.env.TESTSTUDIO_WORK_ITEM ?? '{}').test?.selector === '@STUDIO-PROVIDERS-001' || linkingWorkflow || JSON.parse(process.env.TESTSTUDIO_WORK_ITEM ?? '{}').test?.selector === '@STUDIO-PENDING-001' || JSON.parse(process.env.TESTSTUDIO_WORK_ITEM ?? '{}').test?.selector === '@STUDIO-AUTH-001'
const oidcSecret = randomBytes(24).toString('hex'), oidcFixture = providerWorkflow ? createOIDCFixture(oidcSecret) : null
const token = randomBytes(24).toString('hex'), requests = [], counts = new Map()
const bindingsPath = '/tmp/teststudio-notification-bindings.json'
const providerPath = '/tmp/teststudio-password-provider.json'
const kinds = ['retry', 'failure', 'waiting']
let development = true, passwords = localWorkflow || linkingWorkflow, restarting = false
let acceptFailure = false, child, stopping = false
const helpers = new Set()
const server = createServer(async (request, response) => {
  try {
    if (stopping) { response.writeHead(503).end(); return }
    if (oidcFixture && await oidcFixture(request, response)) return
    if (publicStatusWorkflow && request.method === 'GET' && request.url.startsWith('/status-host/')) {
      const statusToken = request.url.slice('/status-host/'.length)
      if (!/^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$/.test(statusToken)) { response.writeHead(400).end(); return }
      response.setHeader('Content-Type', 'text/html; charset=utf-8')
      response.end('<!doctype html><html lang="en"><title>Independent embed host</title><h1>Customer dashboard</h1><iframe title="Embedded public status" src="http://127.0.0.1:8040/status/' + statusToken + '" style="width:100%;height:700px;border:0"></iframe></html>'); return
    }
    if (request.method === 'GET' && request.url === '/state') {
      response.setHeader('Content-Type', 'application/json'); response.end(JSON.stringify({ requests })); return
    }
    const chunks = []
    let length = 0
    for await (const chunk of request) {
      length += chunk.length
      if (length > 65536) { response.writeHead(413).end(); return }
      chunks.push(chunk)
    }
    const body = JSON.parse(Buffer.concat(chunks).toString() || '{}')
    if (stopping) { response.writeHead(503).end(); return }
    if (localWorkflow && request.method === 'POST' && request.url === '/local/mode') {
      if (typeof body.development !== 'boolean' || typeof body.enabled !== 'boolean' || restarting) { response.writeHead(400).end(); return }
      restarting = true
      try {
        await stopChild(child)
        if (stopping) { response.writeHead(503).end(); return }
        development = body.development; passwords = body.enabled; startChild()
        let ready = false
        for (let attempt = 0; attempt < 100 && !stopping; attempt++) {
          try { ready = (await fetch('http://127.0.0.1:8040/healthz')).ok } catch {}
          if (ready) break
          await new Promise(resolve => setTimeout(resolve, 50))
        }
        if (!ready) throw new Error('Restarted password fixture did not become ready')
        response.writeHead(204).end()
      } finally { restarting = false }
      return
    }
    if (request.method === 'POST' && ['/access/prepare', '/access/creator', ...(diagnosticsWorkflow ? ['/diagnostics/prepare'] : []), ...(privateTraceWorkflow ? ['/trace/expire'] : []), ...(historyWorkflow ? ['/history/prepare', '/history/remove-working'] : [])].includes(request.url)) {
      if (typeof body.project !== 'string' || !/^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$/.test(body.project)) { response.writeHead(400).end(); return }
      const fixture = await new Promise((resolve, reject) => {
        const helper = execFile('/app/access-fixture', request.url === '/access/creator' ? [body.project, 'creator'] : request.url === '/diagnostics/prepare' ? [body.project, 'diagnostics'] : request.url === '/trace/expire' ? [body.project, 'expire-viewer'] : request.url.startsWith('/history/') ? [body.project, 'history'] : [body.project], { env: { PATH: process.env.PATH, TMPDIR: '/tmp', ...(request.url === '/trace/expire' ? { TESTSTUDIO_SELFTEST_VIEWER_TOKEN: String(body.token ?? '') } : {}), ...(request.url.startsWith('/history/') ? { TESTSTUDIO_SELFTEST_HISTORY_RUN: String(body.run ?? ''), TESTSTUDIO_SELFTEST_HISTORY_REMOVE: request.url === '/history/remove-working' ? 'true' : 'false' } : {}) }, timeout: 10000, maxBuffer: 4096 }, (error, stdout) => {
          helpers.delete(helper)
          if (error) reject(new Error('Fixture preparation failed')); else resolve(stdout)
        })
        helpers.add(helper)
      })
      response.setHeader('Content-Type', 'application/json'); response.end(JSON.stringify(JSON.parse(fixture))); return
    }
    if (request.method === 'POST' && request.url === '/control') {
      acceptFailure = body.acceptFailure === true; response.writeHead(204).end(); return
    }
    const url = new URL(request.url, 'http://127.0.0.1:8050')
    const [, prefix, providedToken, kind] = url.pathname.split('/')
    if (request.method !== 'POST' || prefix !== 'receiver' || providedToken !== token || !['retry', 'failure', 'waiting'].includes(kind)) { response.writeHead(404).end(); return }
    const deliveryId = String(request.headers['x-teststudio-delivery-id'] ?? '')
    const attempt = (counts.get(deliveryId) ?? 0) + 1; counts.set(deliveryId, attempt)
    const status = kind === 'waiting' ? 503 : kind === 'failure' && !acceptFailure ? 400 : kind === 'retry' && attempt === 1 ? 503 : 204
    requests.push({ deliveryId, attempt, kind, status, runId: body.runId, counts: body.counts,
      discordConfirmed: url.searchParams.get('wait') === 'true', mentionsSuppressed: Array.isArray(body.allowed_mentions?.parse) && body.allowed_mentions.parse.length === 0,
      secretSeenInMessage: JSON.stringify(body).includes(token) })
    if (status === 503) response.setHeader('Retry-After', kind === 'waiting' ? '30' : '1')
    response.writeHead(status).end(status === 204 ? undefined : `Do not retain this provider diagnostic ${token}`)
  } catch { if (!response.headersSent && !response.destroyed) response.writeHead(400).end() }
})

async function stopChild(owned) {
  if (!owned?.pid || owned.exitCode !== null || owned.signalCode !== null) return
  const exited = new Promise(resolve => owned.once('exit', resolve))
  owned.kill('SIGTERM')
  const timer = setTimeout(() => owned.kill('SIGKILL'), 4000)
  await exited; clearTimeout(timer)
}

async function stop(code) {
  if (stopping) return
  stopping = true
  await Promise.all([stopChild(child), ...[...helpers].map(stopChild)])
  server.closeAllConnections()
  await new Promise(resolve => server.close(resolve))
  await unlink(bindingsPath).catch(() => {})
  await unlink(providerPath).catch(() => {})
  process.exitCode = code
}

function startChild() {
  child = spawn('/app/teststudio', [...(development ? ['--local'] : []), ...(recordingWorkflow ? ['--recording-cli', '/runner/node_modules/playwright/cli.js', '--recording-helper', '/suite/recording-fixture.mjs', '--recording-directory', '/tmp/teststudio-recordings'] : []), ...(bundleWorkflow ? ['--source-bundles'] : []), ...(repositoryWorkflow ? ['--github-sources'] : []), ...(passwords ? ['--local-accounts'] : []), ...(localWorkflow ? ['--oidc-config', providerPath] : []), ...(providerWorkflow ? ['--oidc-secret-refs', 'env:TESTSTUDIO_SELFTEST_OIDC_SECRET'] : []), '--trace-viewer-directory', '/app/trace-viewer', '--database', '/tmp/teststudio-selftest.db', '--artifact-directory', '/tmp/teststudio-artifacts', '--ui', '/app/ui', '--secret-bindings', bindingsPath], {
    env: { PATH: process.env.PATH, TMPDIR: '/tmp', ...(providerWorkflow ? { TESTSTUDIO_SELFTEST_OIDC_SECRET: oidcSecret } : {}), ...Object.fromEntries(kinds.map(kind => [`TESTSTUDIO_SELFTEST_${kind.toUpperCase()}`, `http://127.0.0.1:8050/receiver/${token}/${kind}`])) }, stdio: ['ignore', 'inherit', 'inherit']
  })
  child.once('error', () => { void stop(1) })
  child.once('exit', code => { if (!stopping && !restarting) void stop(code || 1) })
}

process.on('SIGTERM', () => { void stop(0) })
process.on('SIGINT', () => { void stop(0) })
try {
  if (projectImagesWorkflow) {
    const display = spawn('Xvfb', [':97', '-screen', '0', '1440x1000x24', '-ac', '-nolisten', 'tcp'], { stdio: 'ignore' })
    helpers.add(display)
    display.once('error', () => { void stop(1) })
    let ready = false
    for (let count = 0; count < 100 && !ready; count++) {
      ready = await access('/tmp/.X11-unix/X97').then(() => true, () => false)
      if (!ready) await new Promise(resolve => setTimeout(resolve, 25))
    }
    if (!ready) throw new Error('The isolated capture display did not start')
  }
  await new Promise((resolve, reject) => { server.once('error', reject); server.listen(8050, '127.0.0.1', resolve) })
  if (localWorkflow) await writeFile(providerPath, JSON.stringify([{ id: 'selftest-oidc', name: 'Self-test identity', issuer: 'https://identity.selftest.invalid', clientId: 'teststudio-selftest', scopes: ['openid'], autoProvision: false }]), { mode: 0o600 })
  const bindings = kinds.map(kind => ({ id: `${kind}-receiver`, name: `${kind} receiver`, projectId: 'notification-selftests', purpose: 'notification-webhook', environmentVariable: `TESTSTUDIO_SELFTEST_${kind.toUpperCase()}`, network: 'loopback' }))
  if (apiAuthoringWorkflow) bindings.push({ id: 'local-api-reader', name: 'Local API reader', projectId: 'api-auth-authoring', purpose: 'api-credential', environmentVariable: 'UNRESOLVED_API_AUTHOR_VALUE', network: 'private', allowedOrigins: ['http://127.0.0.1:8040'] })
  await writeFile(bindingsPath, JSON.stringify(bindings), { mode: 0o600 })
  startChild()
} catch { await stop(1) }
