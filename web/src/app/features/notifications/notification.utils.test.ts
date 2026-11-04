import assert from 'node:assert/strict'
import { test } from 'node:test'
import type { Channel, Delivery } from './notification.models'
import { canCancelDelivery, canRetryDelivery, channelDefinition, deliveryStatus, notificationDefinition, notificationForm, notificationValidation, notificationIdentityValidation } from './notification.utils.ts'

const channel = { id: 'alerts', name: 'Alerts', enabled: true, kind: 'webhook', secretReference: 'receiver', suiteIds: ['health'], environmentIds: ['none'], stages: ['acceptance testing'], triggers: ['schedule'], outcomes: ['timed-out'], includeEvidenceGaps: false, maxAttempts: 3, projectId: 'example', owner: 'alice', archived: false, deliveryEpoch: 8 } as Channel
const references = [{ id: 'receiver', name: 'Local receiver' }]

test('editing preserves exact scope, timeout selection, and explicit empty outcome lists without serializing server metadata', () => {
  assert.deepEqual(notificationDefinition(notificationForm(channel)), channelDefinition(channel))
  assert.equal('owner' in channelDefinition(channel), false)
  assert.equal('deliveryEpoch' in notificationDefinition(notificationForm(channel)), false)
  const onlyGaps = { ...channel, outcomes: [], includeEvidenceGaps: true }
  assert.deepEqual(notificationDefinition(notificationForm(onlyGaps)).outcomes, [])
  const unfiltered = notificationDefinition(notificationForm({ ...channel, suiteIds: null, environmentIds: null, stages: null, triggers: null, outcomes: null }))
  assert.deepEqual(unfiltered.triggers, [])
  assert.deepEqual(unfiltered.outcomes, ['failed', 'timed-out', 'flaky', 'unknown'])
  assert.deepEqual(unfiltered.suiteIds, [])
})

test('selection validates registered project references, nonempty triggers, evidence-only rules, and bounded exact filters', () => {
  const form = notificationForm(channel)
  assert.equal(notificationValidation(form, references), '')
  assert.match(notificationValidation(form, []), /registered secret/)
  assert.match(notificationValidation({ ...form, schedule: false }, references), /trigger/)
  assert.match(notificationValidation({ ...form, timedOut: false }, references), /outcome/)
  assert.equal(notificationValidation({ ...form, timedOut: false, includeEvidenceGaps: true }, references), '')
  for (const maxAttempts of [0, 11, 1.5, NaN]) assert.match(notificationValidation({ ...form, maxAttempts }, references), /attempts/)
  assert.match(notificationValidation({ ...form, suites: 'health, health' }, references), /unique/)
  assert.match(notificationValidation({ ...form, environments: 'bad/id' }, references), /valid identifiers/)
  assert.match(notificationValidation({ ...form, stages: 'é'.repeat(65) }, references), /128 bytes/)
  assert.match(notificationValidation({ ...form, name: 'é'.repeat(129) }, references), /256 bytes/)
})

test('delivery operations distinguish waiting work, permanent stops, and uncertain in-flight checkpoints', () => {
  for (const state of ['pending', 'retrying', 'failed', 'blocked', 'cancelled', 'sending', 'delivered']) {
    assert.equal(canRetryDelivery(state), ['retrying', 'failed', 'blocked', 'cancelled'].includes(state))
    assert.equal(canCancelDelivery(state), ['pending', 'retrying'].includes(state))
  }
  const delivery = { state: 'sending', claimUntil: '2026-11-02T12:00:00Z', acceptanceUnknown: true } as Delivery
  assert.equal(deliveryStatus(delivery, Date.parse('2026-11-02T12:00:01Z')), 'Recovery pending')
  assert.equal(deliveryStatus(delivery, Date.parse('2026-11-02T11:59:59Z')), 'sending')
  assert.equal(deliveryStatus({ ...delivery, state: 'delivered' }, Date.now()), 'delivered')
})

test('Destination validation lets users fix selection rules on the next step', () => {
  const form = { ...notificationForm(channel), schedule: false }
  assert.equal(notificationIdentityValidation(form, references), '')
  assert.match(notificationValidation(form, references), /trigger/)
  assert.match(notificationIdentityValidation(form, []), /registered secret/)
})
