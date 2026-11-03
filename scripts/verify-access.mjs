import { spawn } from 'node:child_process'
import { createServer } from 'node:net'
import { openSync, closeSync, mkdirSync, writeFileSync } from 'node:fs'
import path from 'node:path'
import assert from 'node:assert/strict'
import { randomBytes } from 'node:crypto'

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
const reviewerInitial = randomBytes(24).toString('base64url'), reviewerPassword = randomBytes(24).toString('base64url')

async function start(recover = false) {
  const fd = openSync(path.join(root, 'server.log'), 'a', 0o600)
  const child = spawn(binary, ['--local', '--local-accounts', ...(recover ? ['--recover-local-admin'] : []), '--addr', `127.0.0.1:${ports[0].port}`, '--public-url', origin,
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
  const assignmentBase = '/api/v1/projects/access-example/role-assignments'
  const assignmentPath = assignmentBase + '/user/pending-reviewer/builtin.teststudio.editor'
  const assigned = await request('POST', assignmentBase, { principalKind: 'user', principalId: 'pending-reviewer', roleId: 'builtin.teststudio.editor', reason: 'Verify durable scoped access' })
  await request('POST', assignmentBase, { principalKind: 'user', principalId: 'pending-reviewer', roleId: 'builtin.teststudio.operator', reason: 'Verify retained predefined execution authority' })
  const operatorRolePath = '/api/v1/predefined-roles/builtin.teststudio.operator'
  const operatorDefault = await request('GET', operatorRolePath)
  const disabledOperator = await request('PUT', operatorRolePath, { disabled: true, reason: 'Verify a durable installation-wide disabled override' }, operatorDefault.revision)
  for (const id of ['health', 'another']) await request('PUT', '/api/v1/projects/access-example/suites/' + id, { id, name: id, applications: [], requiresTarget: false }, 0)
  const role = { id: 'health-reader', name: 'Health reader', description: 'New exact-object verification', projectId: 'access-example', disabled: false, rules: [{ permission: 'teststudio.catalog.read', effect: 'allow', object: 'suite/health' }], inherits: [], reason: 'Verify durable object rules' }
  const reader = await request('POST', '/api/v1/roles', role)
  const composite = await request('POST', '/api/v1/roles', { ...role, id: 'composite-reader', name: 'Composite reader', rules: [], inherits: [role.id] })
  await request('POST', assignmentBase, { principalKind: 'group', principalId: 'reviewers', roleId: composite.id, reason: 'Verify inherited group access' })
  const grantsPath = '/api/v1/projects/access-example/permissions'
  const grantDefinition = { principalKind: 'user', principalId: 'pending-reviewer', permission: 'teststudio.runs.read', effect: 'allow', reason: 'Verify durable direct access' }
  const direct = await request('POST', grantsPath, grantDefinition)
  const denied = await request('POST', grantsPath, { ...grantDefinition, principalKind: 'group', principalId: 'reviewers', effect: 'deny', reason: 'Verify durable deny precedence' })
  await request('PUT', '/api/v1/users/pending-reviewer/local-credential', { password: reviewerInitial, reason: 'Verify through an actual managed session' }, 0)
  const rootAssignment = '/api/v1/role-assignments/user/local-administrator/builtin.teststudio.installation-admin'
  const rootRole = await request('GET', rootAssignment)
  await request('DELETE', rootAssignment, { reason: 'Reject removal of the only usable role' }, rootRole.revision, 422)
  const administrator = await request('GET', '/api/v1/users/local-administrator')
  await request('POST', '/api/v1/users/local-administrator/disable', { reason: 'Reject removal of the only administrator' }, administrator.revision, 422)
  const sessions = await request('GET', '/api/v1/users/local-administrator/sessions')
  assert.equal(sessions.items.length, 1)
  assert.equal(JSON.stringify(sessions).includes(csrf), false)
  const rootCookie = cookie, rootCSRF = csrf
  cookie = ''; csrf = ''
  await request('POST', '/auth/password', { userId: 'pending-reviewer', password: reviewerInitial })
  csrf = (await request('GET', '/auth/session')).session.csrfToken
  await request('POST', '/auth/password/change', { currentPassword: reviewerInitial, newPassword: reviewerPassword })
  const reviewerCookie = cookie
  cookie = rootCookie; csrf = rootCSRF
  assert.equal((await request('GET', '/api/v1/me/capabilities?project=access-example', undefined, undefined, 200, reviewerCookie))['teststudio.runs.cancel'], false)
  const explanationQuery = '?project=access-example&object=suite%2Fhealth&permission=teststudio.catalog.read'
  const healthExplanation = '/api/v1/users/pending-reviewer/effective-permissions' + explanationQuery
  const firstExplanation = await request('GET', healthExplanation)
  assert.equal(firstExplanation.allowed, true)
  const inheritedSource = firstExplanation.items.find(item => item.roleId === role.id && item.assignedRoleId === composite.id)
  assert.ok(inheritedSource)
  assert.equal(inheritedSource.principalId, 'reviewers')
  const ownExplanation = await request('GET', '/api/v1/me/effective-permissions' + explanationQuery, undefined, undefined, 200, reviewerCookie)
  assert.equal(ownExplanation.allowed, true)
  assert.equal(ownExplanation.canInspectOthers, false)
  await stop()

  await start()
  const restored = await request('GET', '/auth/session', undefined, undefined, 200, oldCookie)
  assert.equal(restored.user.sessionEpoch, 1)
  assert.equal((await request('GET', '/api/v1/groups/reviewers/members')).items.length, 1)
  const retainedOperator = await request('GET', operatorRolePath)
  assert.equal(retainedOperator.revision, disabledOperator.revision)
  assert.equal(retainedOperator.data.disabled, true)
  assert.equal((await request('GET', '/api/v1/me/capabilities?project=access-example', undefined, undefined, 200, reviewerCookie))['teststudio.runs.cancel'], false)
  const continued = await request('GET', '/api/v1/groups?project=access-example&limit=1&cursor=' + encodeURIComponent(firstPage.cursor))
  assert.equal(continued.items[0].id, 'watchers')
  const retainedAssignment = await request('GET', assignmentPath)
  assert.equal(retainedAssignment.id, assigned.id)
  assert.equal(retainedAssignment.revision, assigned.revision)
  assert.equal((await request('GET', '/api/v1/roles/' + role.id)).revision, reader.revision)
  assert.deepEqual((await request('GET', '/api/v1/roles/' + composite.id)).data.inherits, [role.id])
  await request('DELETE', assignmentPath, { reason: 'Revoke before the next restart' }, assigned.revision)
  const retainedExplanation = await request('GET', healthExplanation)
  assert.equal(retainedExplanation.items.find(item => item.roleId === role.id).id, inheritedSource.id)
  const runHistory = '/api/v1/projects/access-example/runs'
  assert.equal((await request('GET', grantsPath + '/' + direct.id)).revision, direct.revision)
  await request('GET', runHistory, undefined, undefined, 403, reviewerCookie)
  const expiredDeny = await request('PUT', grantsPath + '/' + denied.id, { expiresAt: '2000-01-01T00:00:00.123456Z', reason: 'Verify expired deny after restart' }, denied.revision)
  assert.equal(expiredDeny.data.expired, true)
  await request('GET', runHistory, undefined, undefined, 200, reviewerCookie)
  await request('DELETE', grantsPath + '/' + direct.id, { reason: 'Revoke the current direct grant' }, direct.revision)
  await request('GET', runHistory, undefined, undefined, 403, reviewerCookie)
  const recreatedDirect = await request('POST', grantsPath, grantDefinition)
  assert.equal(recreatedDirect.id, direct.id)
  assert.ok(recreatedDirect.revision > direct.revision)
  await request('DELETE', grantsPath + '/' + direct.id, { reason: 'Reject a stale direct removal' }, direct.revision, 409)
  await request('GET', runHistory, undefined, undefined, 200, reviewerCookie)
  const healthDrafts = '/api/v1/projects/access-example/suites/health/catalog-drafts'
  await request('GET', healthDrafts, undefined, undefined, 200, reviewerCookie)
  await request('GET', '/api/v1/projects/access-example/suites/another/catalog-drafts', undefined, undefined, 403, reviewerCookie)
  await request('GET', '/api/v1/roles/' + composite.id, undefined, undefined, 403, reviewerCookie)
  const disabledRole = await request('PUT', '/api/v1/roles/' + composite.id, { ...composite.data, disabled: true, reason: 'Disable inherited access before restart' }, composite.revision)
  await request('GET', healthDrafts, undefined, undefined, 403, reviewerCookie)
  const disabledExplanation = await request('GET', healthExplanation)
  assert.equal(disabledExplanation.allowed, false)
  assert.equal(disabledExplanation.reason, 'no-matching-allow')
  assert.equal(disabledExplanation.items.length, 0)
  await stop()

  await start(true)
  const invalidated = await request('GET', '/auth/session', undefined, undefined, 200, oldCookie)
  assert.equal(invalidated.user, null)
  assert.equal((await signIn()).user.sessionEpoch, 2)
  assert.equal((await request('GET', grantsPath + '/' + direct.id)).revision, recreatedDirect.revision)
  const retainedDeny = await request('GET', grantsPath + '/' + denied.id)
  assert.equal(retainedDeny.revision, expiredDeny.revision)
  assert.equal(retainedDeny.data.expired, true)
  await request('GET', runHistory, undefined, undefined, 200, reviewerCookie)
  const retainedRole = await request('GET', '/api/v1/roles/' + composite.id)
  assert.equal(retainedRole.revision, disabledRole.revision)
  assert.equal(retainedRole.data.disabled, true)
  assert.deepEqual(retainedRole.data.inherits, [role.id])
  await request('GET', healthDrafts, undefined, undefined, 403, reviewerCookie)
  await request('PUT', '/api/v1/roles/' + composite.id, { ...retainedRole.data, disabled: false, reason: 'Restore retained inherited policy' }, retainedRole.revision)
  await request('GET', healthDrafts, undefined, undefined, 200, reviewerCookie)
  const restoredExplanation = await request('GET', healthExplanation)
  assert.equal(restoredExplanation.allowed, true)
  assert.equal(restoredExplanation.items.find(item => item.roleId === role.id).id, inheritedSource.id)
  await request('GET', assignmentPath, undefined, undefined, 404)
  const recreated = await request('POST', assignmentBase, { principalKind: 'user', principalId: 'pending-reviewer', roleId: 'builtin.teststudio.editor', reason: 'Recreate with a new revision' })
  assert.equal(recreated.id, assigned.id)
  assert.ok(recreated.revision > assigned.revision)
  await request('DELETE', assignmentPath, { reason: 'Reject stale removal' }, assigned.revision, 409)
  await request('DELETE', assignmentPath, { reason: 'Remove the current assignment' }, recreated.revision)
  const users = await request('GET', '/api/v1/users')
  assert.equal(users.items.some(item => item.id === 'pending-reviewer'), true)
  await request('PUT', operatorRolePath, { disabled: false, reason: 'Restore retained predefined execution access after restart' }, retainedOperator.revision)
  assert.equal((await request('GET', '/api/v1/me/capabilities?project=access-example', undefined, undefined, 200, reviewerCookie))['teststudio.runs.cancel'], true)
  const completePermissions = (await request('GET', '/api/v1/permissions')).items.map(item => ({ permission: item.id, effect: 'allow' }))
  await request('POST', '/api/v1/roles', { id: 'complete-recovery-administrator', name: 'Complete recovery administrator', projectId: '', rules: completePermissions, reason: 'Retain a usable local account while testing predefined administrator recovery' })
  await request('POST', '/api/v1/role-assignments', { principalKind: 'user', principalId: 'pending-reviewer', roleId: 'complete-recovery-administrator', reason: 'Retain complete alternate administration' })
  const builtinAdminPath = '/api/v1/predefined-roles/builtin.teststudio.installation-admin'
  const builtinAdmin = await request('GET', builtinAdminPath)
  const disabledAdmin = await request('PUT', builtinAdminPath, { disabled: true, reason: 'Verify explicit operator recovery of a disabled administrator role' }, builtinAdmin.revision)
  assert.equal(disabledAdmin.data.canManage, false)
  const beforeRoleRecoveryCookie = cookie
  assert.equal((await request('GET', '/api/v1/me/capabilities'))['teststudio.system.manage'], false)
  await stop()

  await start()
  assert.equal((await request('GET', '/api/v1/me/capabilities'))['teststudio.system.manage'], false)
  await stop()

  await start(true)
  assert.equal((await request('GET', '/auth/session', undefined, undefined, 200, beforeRoleRecoveryCookie)).user, null)
  assert.equal((await signIn()).user.sessionEpoch, 3)
  const recoveredAdmin = await request('GET', builtinAdminPath)
  assert.equal(recoveredAdmin.data.disabled, false)
  assert.ok(recoveredAdmin.revision > disabledAdmin.revision)
  await request('PUT', builtinAdminPath, { disabled: false, reason: 'Reject a stale role override revision after explicit recovery' }, disabledAdmin.revision, 409)
  await stop()

  for (const args of [['--recover-local-admin'], ['--local', '--recover-local-admin', '--addr', '0.0.0.0:8040']]) {
    const child = spawn(binary, args, { env: environment, stdio: 'ignore' })
    const code = await new Promise((resolve, reject) => { child.once('exit', resolve); child.once('error', reject) })
    assert.equal(code, 1, 'Recovery accepted an invalid listen mode')
  }
  const summary = { permissionExplanationsPreserved: true, inactiveInheritedAccessExplained: true, ownExactObjectExplanationVerified: true, builtinOverridesPreserved: true, retainedBuiltinAccessRestored: true, ordinaryStartupPreservesDisabledBuiltin: true, explicitRecoveryRestoresBuiltinAdministrator: true, builtinRecoveryAdvancesRevision: true, directGrantsPreserved: true, directDenyPrecedenceVerified: true, expiredDeniesPreserved: true, directRevocationVerified: true, recreatedDirectRevisionAdvanced: true, customRoleRulesPreserved: true, customRoleInheritancePreserved: true, exactObjectAccessVerified: true, disabledInheritedAccessPreserved: true, enabledInheritedAccessRestored: true, assignmentsPreserved: true, assignmentRevocationPreserved: true, recreatedAssignmentRevisionAdvanced: true, usersPreserved: true, membershipsPreserved: true, paginationSurvivedRestart: true, oldSessionSurvivedOrdinaryRestart: true, oldSessionRejectedAfterRecovery: true, lastAdministratorRemovalRejected: true, recoveryRequiresLoopbackLocalMode: true, ownedControllersExited: true }
  const output = path.resolve('output/access', path.basename(root))
  mkdirSync(output, { recursive: true })
  writeFileSync(path.join(output, 'summary.json'), JSON.stringify(summary, null, 2))
  console.log(`Access verification passed: ${output}`)
  console.log(`Retained isolated installation: ${root}`)
} finally { await stop() }
