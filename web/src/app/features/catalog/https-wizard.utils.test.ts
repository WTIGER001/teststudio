import { test } from 'node:test'
import assert from 'node:assert/strict'
import { probeTargets, probeURL } from './https-wizard.utils'
import type { RecordValue } from '../../core/api.service'
import type { Environment, Suite } from '../configuration/configuration.service'

const target = (id: string, applicationId: string, disabled = false): RecordValue<Environment> => ({ id, revision: 1, updatedAt: '', data: { id, name: id, applicationId, stage: 'local', protected: false, disabled, endpoints: { api: 'http://localhost' } } })
const suite: RecordValue<Suite> = { id: 'health', revision: 1, updatedAt: '', data: { id: 'health', name: 'Health', description: '', applications: ['app'], requiresTarget: true, archived: false } }

test('Probe targets respect linked applications and exclude disabled environments', () => {
  const targets = [target('local', 'app'), target('other', 'different'), target('disabled', 'app', true)]
  assert.deepEqual(probeTargets(targets, [suite], 'health').map(row => row.id), ['local'])
  assert.deepEqual(probeTargets(targets, [{ ...suite, data: { ...suite.data, applications: [] } }], 'health').map(row => row.id), ['local', 'other'])
  assert.equal(targets.length, 3)
})
test('Probe preview appends the path and replaces endpoint query settings like the executor', () => {
  assert.equal(probeURL('https://example.test/api/?old=1', '/healthz?ready=yes'), 'https://example.test/api/healthz?ready=yes')
  assert.equal(probeURL('http://localhost:8050/', '/healthz'), 'http://localhost:8050/healthz')
  assert.match(probeURL(undefined, '/healthz'), /resolved from the run environment/)
})
