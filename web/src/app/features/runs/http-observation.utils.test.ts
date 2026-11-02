import assert from 'node:assert/strict'
import { test } from 'node:test'
import { certificateObservation, httpTimingRows } from './http-observation.utils.ts'

test('HTTP diagnostics distinguish unavailable phases from a valid measured zero', () => {
  const rows = httpTimingRows({ status: 200, url: 'https://example.invalid/health', durationMs: 25, timingsMs: { connect: 0, firstByte: 25 } })
  assert.deepEqual(rows.map(row => row.milliseconds), [undefined, 0, undefined, 25])
})

test('certificate lifetime remains relative to its retained observation', () => {
  const http = { status: 200, url: 'https://example.invalid/health', durationMs: 25, certificateExpires: '2026-12-02T12:00:00Z' }
  assert.equal(certificateObservation(http, '2026-11-02T12:00:00Z').remainingDays, 30)
  assert.equal(certificateObservation({ ...http, certificateExpires: undefined }, '2026-11-02T12:00:00Z').expiresAt, undefined)
  assert.equal(certificateObservation({ ...http, certificateExpires: 'invalid' }, '2026-11-02T12:00:00Z').expiresAt, undefined)
})
