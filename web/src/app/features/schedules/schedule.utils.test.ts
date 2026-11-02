import test from 'node:test'
import assert from 'node:assert/strict'
import { definitionFromForm, scheduleDefinition, scheduleForm, scheduleRows } from './schedule.utils.ts'

test('Schedule edits retain request options and send configuration fields only', () => {
  const original = { id: 'probe', name: 'Probe', enabled: true, kind: 'interval' as const, intervalSeconds: 5, timezone: 'UTC', overlap: 'skip' as const,
    request: { suiteId: 'health', environmentId: 'local', executionLocation: 'local', timeoutSeconds: 60, retries: 1, capture: { logs: true } }, owner: 'internal-owner', generation: 5, claimUntil: 'server-state' }
  const value = scheduleForm(original)
  value.name = ' Renamed '; value.ids = 'CHECK-1, CHECK-2'; value.kind = 'cron'; value.cron = ' 0 2 * * * '; value.timezone = 'America/New_York'
  const definition = definitionFromForm(value, original)
  assert.equal(definition.name, 'Renamed'); assert.equal(definition.intervalSeconds, undefined); assert.equal(definition.cron, '0 2 * * *')
  assert.deepEqual(definition.request.testIds, ['CHECK-1', 'CHECK-2']); assert.deepEqual((definition.request as any).capture, { logs: true })
  const wire = scheduleDefinition(original)
  assert.equal('owner' in wire, false); assert.equal('generation' in wire, false); assert.equal('claimUntil' in wire, false)
})
test('Interval edits use UTC and preserve unknown timing instead of inventing a run estimate', () => {
  const form = scheduleForm(); form.id = 'probe'; form.name = 'Probe'; form.suite = 'health'; form.timezone = 'America/New_York'; form.cron = 'ignored'
  const definition = definitionFromForm(form)
  assert.equal(definition.enabled, false); assert.equal(definition.timezone, 'UTC'); assert.equal(definition.cron, undefined)
  assert.equal(definition.request.releaseId, undefined); assert.deepEqual(definition.request.testIds, [])
  assert.equal(scheduleRows([{ id: 'probe', revision: 1, updatedAt: '', data: { ...definition, owner: 'owner', projectId: 'project', generation: 1, archived: false } }])[0].status, 'Paused')
})
