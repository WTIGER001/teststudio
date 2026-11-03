import { spawn } from 'node:child_process'
import { createServer } from 'node:net'
import { openSync, closeSync, mkdirSync, writeFileSync } from 'node:fs'
import path from 'node:path'
import assert from 'node:assert/strict'

const root = path.resolve(process.argv[2]), binary = path.join(root, 'teststudio')
const ports = []
try {
  for (let i = 0; i < 2; i++) {
    const server = createServer()
    await new Promise((resolve, reject) => { server.once('error', reject); server.listen(0, '127.0.0.1', resolve) })
    ports.push({ server, port: server.address().port })
  }
} finally { await Promise.all(ports.map(({ server }) => new Promise(resolve => server.close(resolve)))) }
const origin = `http://127.0.0.1:${ports[0].port}`, artifactOrigin = `http://127.0.0.1:${ports[1].port}`
const environment = { PATH: process.env.PATH, TMPDIR: process.env.TMPDIR ?? '/tmp' }
let owned, cookie = '', csrf = ''

async function start(recover = false) {
  const fd = openSync(path.join(root, 'server.log'), 'a', 0o600)
  const child = spawn(binary, ['--local', ...(recover ? ['--recover-local-admin'] : []), '--addr', `127.0.0.1:${ports[0].port}`, '--public-url', origin,
    '--artifact-addr', `127.0.0.1:${ports[1].port}`, '--artifact-origin', artifactOrigin, '--database', path.join(root, 'teststudio.db'), '--artifact-directory', path.join(root, 'artifacts')], { env: environment, stdio: ['ignore', fd, fd] })
  closeSync(fd)
  const exited = new Promise((resolve, reject) => { child.once('exit', (code, signal) => resolve({ code, signal })); child.once('error', reject) })
  owned = { child, exited }
  for (let i = 0; i < 100; i++) {
    if (child.exitCode !== null || child.signalCode !== null) throw new Error('The isolated access server stopped during startup')
    try { if ((await fetch(origin + '/healthz')).ok) return } catch {}
    await new Promise(resolve => setTimeout(resolve, 100))
  }
  throw new Error('The isolated access server did not start')
}

async function stop() {
  if (!owned) return
  const { child, exited } = owned
  if (child.exitCode === null && child.signalCode === null) child.kill('SIGTERM')
  const timer = setTimeout(() => child.kill('SIGKILL'), 5000)
  const result = await exited
  clearTimeout(timer); owned = undefined
  assert.equal(result.code, 0, 'Owned controller did not exit cleanly')
}

async function request(method, resource, value, revision, expected = 200, selectedCookie = cookie) {
  const response = await fetch(origin + resource, { method, signal: AbortSignal.timeout(5000), headers: { Origin: origin, Cookie: selectedCookie, 'Content-Type': 'application/json', 'X-CSRF-Token': csrf, ...(revision == null ? {} : { 'If-Match': `"${revision}"` }) }, body: value === undefined ? undefined : JSON.stringify(value) })
  const result = await response.json()
  assert.equal(response.status, expected, `${method} ${resource}: ${result.message ?? response.status}`)
  const received = response.headers.getSetCookie().find(value => value.startsWith('teststudio_session='))
  if (received) cookie = received.split(';')[0]
  return result
}

async function signIn() {
  await request('POST', '/auth/local', {})
  const session = await request('GET', '/auth/session')
  csrf = session.session.csrfToken
  assert.equal(session.user.disabled, false)
  return session
}

try {
  await start(true)
  const signedIn = await signIn()
  assert.equal(signedIn.user.sessionEpoch, 1)
  const oldCookie = cookie
  await request('POST', '/api/v1/users', { id: 'pending-reviewer', name: 'Pending reviewer', email: 'reviewer@example.test' })
  await request('POST', '/api/v1/projects', { id: 'access-example', name: 'Access example' })
  await request('POST', '/api/v1/groups', { id: 'reviewers', name: 'Reviewers', projectId: 'access-example', description: 'New isolated access verification' })
  await request('POST', '/api/v1/groups', { id: 'watchers', name: 'Watchers', projectId: 'access-example' })
  const firstPage = await request('GET', '/api/v1/groups?project=access-example&limit=1')
  assert.equal(firstPage.items[0].id, 'reviewers')
  assert.equal(typeof firstPage.cursor, 'string')
  await request('PUT', '/api/v1/groups/reviewers/members/pending-reviewer', {}, 0)
  const membership = await request('GET', '/api/v1/groups/reviewers/members')
  assert.equal(membership.items[0].data.source, 'local')
  const administrator = await request('GET', '/api/v1/users/local-administrator')
  await request('POST', '/api/v1/users/local-administrator/disable', { reason: 'Reject removal of the only administrator' }, administrator.revision, 422)
  const sessions = await request('GET', '/api/v1/users/local-administrator/sessions')
  assert.equal(sessions.items.length, 1)
  assert.equal(JSON.stringify(sessions).includes(csrf), false)
  await stop()

  await start()
  const restored = await request('GET', '/auth/session', undefined, undefined, 200, oldCookie)
  assert.equal(restored.user.sessionEpoch, 1)
  assert.equal((await request('GET', '/api/v1/groups/reviewers/members')).items.length, 1)
  const continued = await request('GET', '/api/v1/groups?project=access-example&limit=1&cursor=' + encodeURIComponent(firstPage.cursor))
  assert.equal(continued.items[0].id, 'watchers')
  await stop()

  await start(true)
  const invalidated = await request('GET', '/auth/session', undefined, undefined, 200, oldCookie)
  assert.equal(invalidated.user, null)
  assert.equal((await signIn()).user.sessionEpoch, 2)
  const users = await request('GET', '/api/v1/users')
  assert.equal(users.items.some(item => item.id === 'pending-reviewer'), true)
  await stop()

  for (const args of [['--recover-local-admin'], ['--local', '--recover-local-admin', '--addr', '0.0.0.0:8040']]) {
    const child = spawn(binary, args, { env: environment, stdio: 'ignore' })
    const code = await new Promise((resolve, reject) => { child.once('exit', resolve); child.once('error', reject) })
    assert.equal(code, 1, 'Recovery accepted an invalid listen mode')
  }
  const summary = { usersPreserved: true, membershipsPreserved: true, paginationSurvivedRestart: true, oldSessionSurvivedOrdinaryRestart: true, oldSessionRejectedAfterRecovery: true, lastAdministratorRemovalRejected: true, recoveryRequiresLoopbackLocalMode: true, ownedControllersExited: true }
  const output = path.resolve('output/access', path.basename(root))
  mkdirSync(output, { recursive: true })
  writeFileSync(path.join(output, 'summary.json'), JSON.stringify(summary, null, 2))
  console.log(`Access verification passed: ${output}`)
  console.log(`Retained isolated installation: ${root}`)
} finally { await stop() }
