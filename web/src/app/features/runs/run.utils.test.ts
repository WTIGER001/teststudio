import assert from 'node:assert/strict'
import { test } from 'node:test'
import type { Run } from './run.models'
import { canRerun, previewKind } from './run.utils.ts'

test('failure reruns include timeouts and flaky results only after the run finishes', () => {
  for (const state of ['failed', 'timed-out', 'flaky']) {
    const run = { state: 'completed', results: [{ state }] } as Run
    assert.equal(canRerun(run), true, state)
    run.state = 'running'
    assert.equal(canRerun(run), false, state)
  }
  assert.equal(canRerun({ state: 'completed', results: [{ state: 'passed' }, { state: 'skipped' }] } as Run), false)
})

test('Go JSON event evidence opens as text without changing binary previews', () => {
  assert.equal(previewKind('application/x-ndjson'), 'text')
  assert.equal(previewKind('image/png'), 'image')
  assert.equal(previewKind('application/zip'), 'download')
})
