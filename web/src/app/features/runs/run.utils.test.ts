import assert from 'node:assert/strict'
import { test } from 'node:test'
import type { Run } from './run.models'
import { canRerun } from './run.utils.ts'

test('failure reruns include timeouts and flaky results only after the run finishes', () => {
  for (const state of ['failed', 'timed-out', 'flaky']) {
    const run = { state: 'completed', results: [{ state }] } as Run
    assert.equal(canRerun(run), true, state)
    run.state = 'running'
    assert.equal(canRerun(run), false, state)
  }
  assert.equal(canRerun({ state: 'completed', results: [{ state: 'passed' }, { state: 'skipped' }] } as Run), false)
})
