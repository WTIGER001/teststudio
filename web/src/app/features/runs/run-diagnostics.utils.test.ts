import assert from 'node:assert/strict'
import { test } from 'node:test'
import type { RecordValue } from '../../core/api.service'
import type { Attempt, Run } from './run.models'
import { diagnosticAttempts, diagnosticTests, failedSteps, resultSummaries, runActionFeedback, runTestLabels } from './run-diagnostics.utils'

test('diagnostic choices preserve frozen titles and bounded attempt identity', () => {
  const run = { tests: [{ id: 'A', title: 'Checkout' }, { id: 'B', title: 'Login' }], results: [{ testId: 'A', state: 'timed-out', attempt: 2 }], request: { retries: 1 } } as Run
  assert.deepEqual(runTestLabels(run), { A: 'Checkout', B: 'Login' })
  assert.deepEqual(diagnosticTests(run), [{ id: 'A', label: 'Checkout · A · timed-out' }, { id: 'B', label: 'Login · B' }])
  const rows = [{ data: { testId: 'B', attempt: 3 } }] as RecordValue<Attempt>[]
  assert.deepEqual(diagnosticAttempts(run, 'A', rows), [1, 2])
  assert.deepEqual(diagnosticAttempts(run, '', rows), [])
  assert.equal(failedSteps([{ title: 'Assertion', category: 'expect', durationMs: 20, error: 'Mismatch' }, { title: 'Cleanup', category: 'hook', durationMs: 1 }]), 1)
})

test('ambiguous rerun responses explain idempotent same-page recovery', () => {
  for (const status of [0, 500, 503]) assert.match(runActionFeedback('rerun-failures', status, 'Server error'), /reuse the same request/)
  assert.match(runActionFeedback('rerun-failures', 403, 'Forbidden'), /current permissions/)
  assert.match(runActionFeedback('cancel', 503, 'Server error'), /saved run and its results remain available/)
})

test('final result summaries stay compact while the source error remains intact', () => {
  const error = '  Assertion mismatch\nPrivate diagnostic details'
  const run = { results: [{ testId: 'A', error }] } as Run
  assert.deepEqual(resultSummaries(run), { A: 'Assertion mismatch' })
  assert.equal(run.results[0].error, error)
})
