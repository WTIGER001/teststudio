import { spawn } from 'node:child_process'
import { createServer } from 'node:net'
import { openSync, closeSync, mkdirSync, writeFileSync } from 'node:fs'
import path from 'node:path'
import { randomBytes } from 'node:crypto'
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
const initial = randomBytes(24).toString('base64url'), changed = randomBytes(24).toString('base64url')
let owned, cookie = '', csrf = ''
async function start(bootstrap = false, enabled = true) {
  const fd = openSync(path.join(root, 'server.log'), 'a', 0o600)
  const child = spawn(binary, [...(bootstrap ? ['--local'] : []), ...(enabled ? ['--local-accounts'] : []), '--addr', `127.0.0.1:${ports[0].port}`, '--public-url', origin,
    '--artifact-addr', `127.0.0.1:${ports[1].port}`, '--artifact-origin', artifactOrigin, '--database', path.join(root, 'teststudio.db'), '--artifact-directory', path.join(root, 'artifacts')], { env: environment, stdio: ['ignore', fd, fd] })
  closeSync(fd)
  const exited = new Promise((resolve, reject) => { child.once('exit', (code, signal) => resolve({ code, signal })); child.once('error', reject) })
  owned = { child, exited }
  for (let i = 0; i < 100; i++) {
    if (child.exitCode !== null || child.signalCode !== null) throw new Error('The isolated password server stopped during startup')
    try { if ((await fetch(origin + '/healthz')).ok) return } catch {}
    await new Promise(resolve => setTimeout(resolve, 100))
  }
  throw new Error('The isolated password server did not start')
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
async function call(method, resource, value, revision, expected = 200, selectedCookie = cookie) {
  const response = await fetch(origin + resource, { method, signal: AbortSignal.timeout(10000), headers: { Origin: origin, Cookie: selectedCookie, 'Content-Type': 'application/json', 'X-CSRF-Token': csrf, ...(revision == null ? {} : { 'If-Match': `"${revision}"` }) }, body: value === undefined ? undefined : JSON.stringify(value) })
  const result = await response.json()
  assert.equal(response.status, expected, `${method} ${resource}: ${result.message ?? response.status}`)
  const received = response.headers.getSetCookie().find(value => value.startsWith('teststudio_session='))
  if (received) cookie = received.split(';')[0]
  return result
}
async function session() { const result = await call('GET', '/auth/session'); csrf = result.session?.csrfToken ?? ''; return result }
async function login(password, required) {
  const result = await call('POST', '/auth/password', { userId: 'local-administrator', password })
  assert.equal(result.mustChangePassword, required)
  return session()
}
try {
  await start(true)
  await call('POST', '/auth/local', {})
  await session()
  const bootstrapCookie = cookie
  const credentialPath = '/api/v1/users/local-administrator/local-credential'
  const provisioned = await call('PUT', credentialPath, { password: initial, reason: 'Prepare managed local administration' }, 0)
  assert.equal(provisioned.data.mustChangePassword, true)
  assert.equal((await call('GET', '/auth/session', undefined, undefined, 200, bootstrapCookie)).user, null)
  await login(initial, true)
  const restrictedCookie = cookie
  await call('GET', '/api/v1/projects', undefined, undefined, 403)
  await call('POST', '/auth/password/change', { currentPassword: initial, newPassword: changed })
  const normal = await session()
  assert.equal(normal.session.mustChangePassword ?? false, false)
  assert.equal(cookie !== restrictedCookie, true)
  assert.equal((await call('GET', '/auth/session', undefined, undefined, 200, restrictedCookie)).user, null)
  await call('POST', '/api/v1/projects', { id: 'password-example', name: 'Password verification' })
  const activeCookie = cookie
  await stop()
  await start()
  assert.equal((await session()).user.id, 'local-administrator')
  assert.equal((await call('GET', '/auth/providers')).localEnabled, false)
  assert.equal((await call('GET', '/auth/providers')).localAccountsEnabled, true)
  const current = await call('GET', credentialPath)
  await call('POST', credentialPath + '/disable', { reason: 'Reject the final local administrator path' }, current.revision, 422)
  assert.equal((await call('GET', '/api/v1/projects')).items.length, 1)
  await stop()
  await start(false, false)
  assert.equal((await session()).user, null)
  assert.equal((await call('GET', '/auth/providers')).localAccountsEnabled, false)
  await call('POST', '/auth/password', { userId: 'local-administrator', password: changed }, undefined, 403)
  await stop()
  await start()
  assert.equal((await call('GET', '/auth/session', undefined, undefined, 200, activeCookie)).user, null)
  await login(changed, false)
  assert.equal((await call('GET', '/api/v1/projects')).items.length, 1)
  await call('POST', '/auth/logout', {})
  const wrong = await call('POST', '/auth/password', { userId: 'unknown-user', password: initial }, undefined, 401)
  const failed = await call('POST', '/auth/password', { userId: 'local-administrator', password: initial }, undefined, 401)
  assert.equal(wrong.message, failed.message)
  const output = path.resolve('output/local-accounts', path.basename(root))
  mkdirSync(output, { recursive: true })
  await stop()
  for (const args of [['--public-url', 'http://studio.example.test'], ['--addr', '0.0.0.0:0']]) {
    const child = spawn(binary, ['--local-accounts', '--database', path.join(root, 'invalid.db'), '--artifact-directory', path.join(root, 'invalid-artifacts'), ...args], { env: environment, stdio: 'ignore' })
    const timer = setTimeout(() => child.kill('SIGTERM'), 5000)
    try {
      const code = await new Promise((resolve, reject) => { child.once('exit', resolve); child.once('error', reject) })
      assert.equal(code, 1, 'Password sign-in accepted an unprotected public origin or listener')
    } finally { clearTimeout(timer) }
  }
  writeFileSync(path.join(output, 'summary.json'), JSON.stringify({ managedLocalLogin: true, requiredChangeEnforced: true, cookieRotated: true, bootstrapAndRestrictedCookiesRevoked: true, ordinaryRestartPreservedSession: true, passwordOnlyAdministration: true, lastAdministratorProtected: true, disabledModeRejectedLogin: true, reenabledModeRejectedOldCookie: true, genericLoginErrors: true, protectedTransportRequired: true, ownedControllersExited: true }, null, 2))
  console.log(`Local account verification passed: ${output}`)
  console.log(`Retained isolated installation: ${root}`)
} finally { await stop() }
