import fs from 'node:fs'
import path from 'node:path'
import net from 'node:net'
import { spawn, execFileSync } from 'node:child_process'
import { once } from 'node:events'
import { LocalTestClient } from './selftest-api.mjs'

const image = process.env.TESTSTUDIO_SELFTEST_IMAGE
const binary = process.env.TESTSTUDIO_RECOVERY_SERVER
if (!image || !/^sha256:[a-f0-9]{64}$/.test(image) || !binary) throw new Error('Set TESTSTUDIO_SELFTEST_IMAGE and TESTSTUDIO_RECOVERY_SERVER')
fs.mkdirSync('.data', { recursive: true })
const root = fs.mkdtempSync(path.resolve('.data/worker-recovery-'))
const port = async () => { const server = net.createServer(); server.listen(0, '127.0.0.1'); await once(server, 'listening'); const selected = server.address().port; await new Promise(resolve => server.close(resolve)); return selected }
const apiPort = await port(), artifactPort = await port()
const origin = `http://127.0.0.1:${apiPort}`, artifactOrigin = `http://127.0.0.1:${artifactPort}`
const client = new LocalTestClient(origin)
let controller, run, reference
const args = ['--local', '--addr', `127.0.0.1:${apiPort}`, '--public-url', origin, '--artifact-addr', `127.0.0.1:${artifactPort}`, '--artifact-origin', artifactOrigin, '--database', path.join(root, 'teststudio.db'), '--artifact-directory', path.join(root, 'artifacts')]
const waitFor = async (check, timeout, description) => { const deadline = Date.now() + timeout; while (Date.now() < deadline) { const value = await check(); if (value) return value; await new Promise(resolve => setTimeout(resolve, 200)) } throw new Error(description) }
async function start(enabled) {
  const log = fs.openSync(path.join(root, 'server.log'), 'a')
  controller = spawn(path.resolve(binary), enabled ? [...args, '--playwright-images', image] : args, { stdio: ['ignore', log, log] })
  fs.closeSync(log)
  await waitFor(async () => { if (controller.exitCode != null) throw new Error('Controller exited before readiness'); try { return (await fetch(origin + '/healthz')).ok } catch { return false } }, 10000, 'Controller did not start')
  await client.signIn()
}
async function stop(signal) {
  if (!controller || controller.exitCode != null || controller.signalCode != null) return
  const ended = once(controller, 'exit'); controller.kill(signal); await ended
}
try {
  await start(true)
  const base = '/api/v1/projects/recovery-verification'
  await client.call('POST', '/api/v1/projects', { id: 'recovery-verification', name: 'Worker recovery verification' })
  await client.call('PUT', `${base}/suites/browser`, { id: 'browser', name: 'Interruption verification', applications: [], requiresTarget: false }, 0)
  await client.call('POST', `${base}/suites/browser/tests`, { id: 'ADAPTER-TIMEOUT', title: 'Interrupted browser work', source: 'teststudio-selftests', selector: '@ADAPTER-TIMEOUT', category: 'Recovery', type: 'browser', runner: 'playwright', state: 'enabled', safety: 'safe', rationale: 'Runs only in an isolated disposable instance', timeoutSeconds: 60, dependencies: [], requiredBindings: [], tags: [] })
  const draft = await client.call('POST', `${base}/suites/browser/catalog-drafts`, { image, discoveredSelectors: { 'ADAPTER-TIMEOUT': '@ADAPTER-TIMEOUT' } })
  await client.call('POST', `${base}/suites/browser/catalog-drafts/${draft.id}/publish`, { activate: true }, draft.revision)
  run = await client.call('POST', `${base}/runs`, { suiteId: 'browser', timeoutSeconds: 300 })
  await waitFor(async () => {
    const current = (await client.call('GET', `${base}/runs/${run.id}`)).data
    const resources = await client.call('GET', `${base}/runs/${run.id}/executions`)
    reference = resources.items[0]?.data.reference
    return current.phase === 'execution' && current.currentTestStartedAt && reference
  }, 20000, 'Browser work did not start')
  console.log(`Killing owned controller during run ${run.id}`)
  await stop('SIGKILL')
  const present = execFileSync('docker', ['ps', '--filter', `label=teststudio.execution=${reference.id}`, '--format', '{{.Names}}'], { encoding: 'utf8' }).trim()
  if (!present) throw new Error('The abrupt-stop test did not leave live browser work')
  console.log('Restarting the same installation with new Playwright dispatch disabled; waiting for the old lease to expire.')
  await start(false)
  const recovered = await waitFor(async () => {
    const resources = await client.call('GET', `${base}/runs/${run.id}/executions`)
    const record = resources.items[0]?.data
    if (record?.state === 'cleanup-failed') console.log(`Recovery retry: ${record.error}`)
    return record?.state === 'cleaned' ? record : false
  }, 120000, 'Abandoned worker resources were not reconciled')
  const final = (await client.call('GET', `${base}/runs/${run.id}`)).data
  if (final.state !== 'incomplete' || final.outcome !== 'unknown' || !recovered.resultSaved || !recovered.recovered) throw new Error('Recovery changed the unknown run outcome or lost its observation')
  const attempts = await client.call('GET', `${base}/runs/${run.id}/attempts`)
  const artifacts = await client.call('GET', `${base}/runs/${run.id}/artifacts`)
  if (!attempts.items.some(item => item.data.recovered) || !artifacts.items.some(item => item.data.kind === 'log')) throw new Error('Recovery diagnostics are missing')
  const remaining = execFileSync('docker', ['ps', '-a', '--filter', `label=teststudio.execution=${reference.id}`, '--format', '{{.Names}}'], { encoding: 'utf8' }).trim()
  if (remaining) throw new Error('The abandoned container remains')
  try { execFileSync('docker', ['volume', 'inspect', reference.volume], { stdio: 'ignore' }); throw new Error('The abandoned volume remains') } catch (error) { if (error.status == null) throw error }
  const output = path.resolve('output/worker-recovery', run.id); fs.mkdirSync(output, { recursive: true })
  fs.writeFileSync(path.join(output, 'summary.json'), JSON.stringify({ run: final, execution: recovered, attempts: attempts.items, artifacts: artifacts.items, installation: root }, null, 2))
  console.log(`Verified incomplete outcome, recovered diagnostics, and removal of the owned container and volume. Output: ${output}`)
} finally {
  await stop('SIGTERM')
  if (reference) {
    const container = execFileSync('docker', ['ps', '-a', '--filter', `label=teststudio.execution=${reference.id}`, '--format', '{{.Names}}'], { encoding: 'utf8' }).trim()
    if (container) execFileSync('docker', ['rm', '--force', container], { stdio: 'ignore' })
    try { execFileSync('docker', ['volume', 'rm', reference.volume], { stdio: 'ignore' }) } catch {}
  }
}
