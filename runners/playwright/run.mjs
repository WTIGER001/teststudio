import fs from 'node:fs'
import { spawn } from 'node:child_process'
import { suiteContext } from './suite-context.mjs'

const discovery = process.argv.length === 3 && process.argv[2] === '--discover'
if (process.argv.length > 2 && !discovery) throw new Error('Unknown Playwright worker operation')
const context = suiteContext('/suite', discovery)
const source = discovery ? '{}' : fs.readFileSync(0, 'utf8')
if (Buffer.byteLength(source) > 65536) throw new Error('Worker input exceeds 64 KiB')
const input = JSON.parse(source)
if (!discovery && !/^@[A-Za-z0-9][A-Za-z0-9._-]{0,127}$/.test(input.test.selector)) throw new Error('Invalid test selector')
if (!discovery && (!Number.isInteger(input.timeoutMs) || input.timeoutMs < 1 || input.timeoutMs > 3600000)) throw new Error('Invalid test timeout')
const escaped = (input.test?.selector ?? '').replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
const arguments_ = discovery
  ? ['test', `--config=${context.config}`, '--list', '--workers=1', '--reporter=/runner/discovery-reporter.cjs']
  : ['test', `--config=${context.config}`, '--grep', `${escaped}(?:\\s|$)`, '--workers=1', '--retries=0', `--timeout=${input.timeoutMs}`, '--reporter=/runner/reporter.cjs']
if (context.project) arguments_.push(`--project=${context.project}`)
const child = spawn(process.execPath, ['/runner/node_modules/@playwright/test/cli.js', ...arguments_], {
  cwd: context.module, stdio: 'inherit', detached: true, env: { ...process.env, TESTSTUDIO_WORK_ITEM: JSON.stringify(input) }
})
let stopped = false
const stop = () => {
  if (stopped) return
  stopped = true
  try { process.kill(-child.pid, 'SIGTERM') } catch {}
  setTimeout(() => { try { process.kill(-child.pid, 'SIGKILL') } catch {} }, 5000).unref()
}
const timer = setTimeout(stop, discovery ? 120000 : input.timeoutMs + 45000)
process.on('SIGTERM', stop)
process.on('SIGINT', stop)
child.on('error', error => { console.error(error.message); clearTimeout(timer); process.exitCode = 2 })
child.on('exit', code => { clearTimeout(timer); process.exitCode = code ?? 2 })
