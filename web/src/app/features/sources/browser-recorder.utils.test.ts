import assert from 'node:assert/strict'
import test from 'node:test'
import { desktopRecorderCommand, recordingActive, recordingFlow, recordingPairingExpired } from './browser-recorder.utils.ts'

test('Recording progress keeps connection and stopping active before review', () => {
  assert.equal(recordingFlow('companion', undefined).current, 0)
  assert.equal(recordingFlow('companion', 'pairing').current, 1)
  for (const state of ['starting', 'recording', 'stopping']) {
    assert.equal(recordingActive(state), true)
    assert.equal(recordingFlow('companion', state).current, 2)
    assert.equal(recordingFlow('local-desktop', state).current, 1)
  }
  for (const state of ['ready', 'cancelled', 'failed']) {
    assert.equal(recordingActive(state), false)
    assert.equal(recordingFlow('companion', state).current, 3)
    assert.equal(recordingFlow('local-desktop', state).current, 2)
  }
})

test('Temporary pairing codes expire at their exact deadline', () => {
  const pairing = { id: 'session', token: 'private', expiresAt: '2026-11-05T21:00:00Z' }
  assert.equal(recordingPairingExpired(pairing, Date.parse(pairing.expiresAt) - 1), false)
  assert.equal(recordingPairingExpired(pairing, Date.parse(pairing.expiresAt)), true)
  assert.equal(recordingPairingExpired({ ...pairing, expiresAt: 'invalid' }, 0), true)
  assert.equal(recordingPairingExpired(null, 0), true)
  assert.equal(desktopRecorderCommand('https://studio.example.test/ignored?secret=private'), "teststudio-recorder --server 'https://studio.example.test'")
})
