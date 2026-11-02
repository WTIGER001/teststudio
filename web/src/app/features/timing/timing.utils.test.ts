import assert from 'node:assert/strict'
import { test } from 'node:test'
import type { Run } from '../runs/run.models'
import { defaultTimingPolicy, liveTiming, timingPolicyIssue } from './timing.utils.ts'

function activeRun(): Run {
  return {
    startedAt: '2026-11-02T12:00:00Z', currentTestStartedAt: '2026-11-02T12:00:00Z', currentTestId: 'A',
    tests: [{ id: 'A' }, { id: 'B' }], results: [],
    timing: { tests: { A: { expectedMs: 1000 }, B: { expectedMs: 2000 } } }
  } as unknown as Run
}

test('remaining execution uses pending tests and reports unknown history honestly', () => {
  const run = activeRun(), now = Date.parse(run.startedAt!) + 500
  assert.deepEqual(liveTiming(run, now), { elapsedMs: 500, remainingMs: 2500, overdue: false })
  delete run.timing!.tests.B.expectedMs
  assert.equal(liveTiming(run, now).remainingMs, null)
})
test('an overdue attempt does not show a false zero remaining prediction', () => {
  const run = activeRun(), now = Date.parse(run.startedAt!) + 1500
  assert.deepEqual(liveTiming(run, now), { elapsedMs: 1500, remainingMs: null, overdue: true })
})
test('completed tests and finalization keep their measured elapsed time', () => {
  const run = activeRun(), now = Date.parse(run.startedAt!) + 2500
  run.results = [{ testId: 'A' }] as Run['results']
  assert.equal(liveTiming(run, now).remainingMs, 2000)
  assert.equal(liveTiming(run, now).overdue, false)
  run.finishedAt = '2026-11-02T12:00:04Z'
  assert.deepEqual(liveTiming(run, now), { elapsedMs: 4000, remainingMs: null, overdue: false })
  assert.equal(liveTiming(null, now).elapsedMs, null)
})
test('policy window must contain its minimum observation count', () => {
  const policy = defaultTimingPolicy()
  assert.equal(timingPolicyIssue(policy), '')
  policy.window = policy.minimumSamples - 1
  assert.notEqual(timingPolicyIssue(policy), '')
})
