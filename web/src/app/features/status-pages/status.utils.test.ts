import assert from 'node:assert/strict'
import { test } from 'node:test'
import { statusTextColor, statusWindows, monitoredLabel } from './status.utils.ts'

test('public status text preserves contrast across configured backgrounds', () => {
  assert.equal(statusTextColor('#ffffff'), '#111827')
  for (const background of ['#101827', '#000000', '#0000ff']) assert.equal(statusTextColor(background), '#ffffff')
  for (const background of ['#ffff00', '#eeeeee', '#00ff00']) assert.equal(statusTextColor(background), '#111827')
  assert.deepEqual(statusWindows.map(window => window.id), ['24h', '7d', '1m', '1y', 'all'])
})

test('small monitored coverage is visible rather than rounded to zero', () => {
  assert.equal(monitoredLabel(.001), '<0.1%')
  assert.equal(monitoredLabel(0), '0.0%')
  assert.equal(monitoredLabel(42.35), '42.4%')
})
