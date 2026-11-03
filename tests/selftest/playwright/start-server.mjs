import { createServer } from 'node:http'
import { spawn } from 'node:child_process'
import { randomBytes } from 'node:crypto'
import { writeFile, unlink } from 'node:fs/promises'

const token = randomBytes(24).toString('hex'), requests = [], counts = new Map()
const bindingsPath = '/tmp/teststudio-notification-bindings.json'
let acceptFailure = false, child, stopping = false
const server = createServer(async (request, response) => {
  try {
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
  } catch { response.writeHead(400).end() }
})

async function stop(code) {
  if (stopping) return
  stopping = true
  if (child?.pid && child.exitCode === null && child.signalCode === null) {
    const exited = new Promise(resolve => child.once('exit', resolve))
    child.kill('SIGTERM')
    const timer = setTimeout(() => child.kill('SIGKILL'), 4000)
    await exited; clearTimeout(timer)
  }
  server.closeAllConnections()
  await new Promise(resolve => server.close(resolve))
  await unlink(bindingsPath).catch(() => {})
  process.exitCode = code
}

process.on('SIGTERM', () => { void stop(0) })
process.on('SIGINT', () => { void stop(0) })
try {
  await new Promise((resolve, reject) => { server.once('error', reject); server.listen(8050, '127.0.0.1', resolve) })
  const kinds = ['retry', 'failure', 'waiting']
  await writeFile(bindingsPath, JSON.stringify(kinds.map(kind => ({ id: `${kind}-receiver`, name: `${kind} receiver`, projectId: 'notification-selftests', purpose: 'notification-webhook', environmentVariable: `TESTSTUDIO_SELFTEST_${kind.toUpperCase()}`, network: 'loopback' }))), { mode: 0o600 })
  child = spawn('/app/teststudio', ['--local', '--database', '/tmp/teststudio-selftest.db', '--artifact-directory', '/tmp/teststudio-artifacts', '--ui', '/app/ui', '--secret-bindings', bindingsPath], {
    env: { PATH: process.env.PATH, TMPDIR: '/tmp', ...Object.fromEntries(kinds.map(kind => [`TESTSTUDIO_SELFTEST_${kind.toUpperCase()}`, `http://127.0.0.1:8050/receiver/${token}/${kind}`])) }, stdio: ['ignore', 'inherit', 'inherit']
  })
  child.once('error', () => { void stop(1) })
  child.once('exit', code => { if (!stopping) void stop(code || 1) })
} catch { await stop(1) }
