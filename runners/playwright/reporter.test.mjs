import assert from 'node:assert/strict'
import { test } from 'node:test'
import Reporter from './reporter.cjs'

test('missing requested media stays separate from the observed result', async () => {
  for (const value of [
    { state: 'passed', capture: { screenshots: 'always', trace: 'off', video: 'off' }, gaps: 1 },
    { state: 'passed', capture: { screenshots: 'failure', trace: 'failure', video: 'off' }, gaps: 0 },
    { state: 'skipped', capture: { screenshots: 'failure', trace: 'failure', video: 'off' }, gaps: 0 },
    { state: 'failed', capture: { screenshots: 'failure', trace: 'failure', video: 'off' }, gaps: 2 },
    { state: 'timed-out', capture: { screenshots: 'failure', trace: 'failure', video: 'always' }, gaps: 3 },
    { state: 'passed', capture: { screenshots: 'failure', trace: 'off', video: 'off' }, priorFailure: true, gaps: 1 }
  ]) {
    const reporter = Object.create(Reporter.prototype)
    reporter.input = { test: { id: 'MEDIA-1', selector: '@MEDIA-1' }, capture: value.capture, failedBefore: value.priorFailure }
    reporter.results = [{ result: { state: value.state, durationMs: 20 }, attachments: [] }]
    reporter.problems = []
    reporter.attachmentCount = 0
    let report
    reporter.writeReport = value => { report = value }
    await reporter.onEnd({ status: ['failed', 'timed-out'].includes(value.state) ? 'failed' : 'passed' })
    assert.equal(report.result.state, value.state)
    assert.equal(report.evidenceErrors.length, value.gaps)
    assert.ok(report.evidenceErrors.every(error => error.startsWith('Requested ')))
  }
})

test('API diagnostics omit media and opaque attachments before copying files', async () => {
  const reporter = Object.create(Reporter.prototype)
  reporter.input = { test: { id: 'API-1', type: 'api', selector: '@API-1' }, capture: { screenshots: 'off', trace: 'off', video: 'off' } }
  reporter.results = [{ result: { state: 'passed', durationMs: 5 }, attachments: [
    { name: 'trace', contentType: 'application/zip', path: '/must-not-read/private.zip' },
    { name: 'private image', contentType: 'image/png', path: '/must-not-read/private.png' },
    { name: 'opaque', contentType: 'application/octet-stream', body: Buffer.from('private-canary') }
  ] }]
  reporter.problems = []
  reporter.attachmentCount = 0
  let report
  reporter.writeReport = value => { report = value }
  await reporter.onEnd({ status: 'passed' })
  assert.equal(report.result.state, 'passed')
  assert.equal(report.attachments.length, 0)
  assert.equal(report.evidenceErrors.length, 3)
  assert.ok(report.evidenceErrors.every(value => value.startsWith('API attachment omitted:')))
})
