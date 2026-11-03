import { strict as assert } from 'node:assert'
import { test } from 'node:test'
import { pendingAccessDefinition, pendingAccessValidation, pendingBlockReason } from './pending-access.utils'
import { roleObjectValid } from './role-policy.utils'

const providers = [{ id: 'one', name: 'One', issuer: 'https://identity.example', revision: 7, autoProvision: true }]
const roles = [{ id: 'viewer', name: 'Viewer', scope: 'project', permissions: [], builtIn: false, disabled: false, delegatable: true }]

test('Pending access retains its reviewed provider revision and never accepts an unlisted target', () => {
  const body = pendingAccessDefinition({ providerId: 'one', email: ' Owner@EXAMPLE.TEST ', roleId: 'viewer', reason: ' Approve access ' }, providers)
  assert.deepEqual(body, { providerId: 'one', providerRevision: 7, email: 'Owner@EXAMPLE.TEST', roleId: 'viewer', reason: 'Approve access' })
  assert.equal(pendingAccessValidation(body, roles), '')
  assert.match(pendingAccessValidation({ ...body, providerRevision: 0 }, roles), /provider/)
  assert.match(pendingAccessValidation({ ...body, roleId: 'installation-admin' }, roles), /project role/)
  assert.match(pendingAccessValidation(body, [{ ...roles[0], delegatable: false }]), /project role/)
})

test('Pending objects use the shared exact-resource pattern and blocked states explain recovery', () => {
  assert.equal(roleObjectValid('pending-access/approval-1'), true)
  assert.equal(roleObjectValid('pending-access/approval-1/extra'), false)
  assert.match(pendingBlockReason('role-changed'), /inherited policies/)
  assert.match(pendingBlockReason('authority-changed'), /current access manager/)
  assert.equal(pendingBlockReason(undefined), '')
})
