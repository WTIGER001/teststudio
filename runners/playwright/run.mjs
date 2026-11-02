import fs from 'node:fs'
import { spawn } from 'node:child_process'

const source = fs.readFileSync(0, 'utf8')
if (Buffer.byteLength(source) > 65536) throw new Error('Worker input exceeds 64 KiB')
const input = JSON.parse(source)
if (!/^@[A-Za-z0-9][A-Za-z0-9._-]{0,127}$/.test(input.test.selector)) throw new Error('Invalid test selector')
if (!Number.isInteger(input.timeoutMs) || input.timeoutMs < 1 || input.timeoutMs > 3600000) throw new Error('Invalid test timeout')
const escaped = input.test.selector.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
const child = spawn(process.execPath, ['/runner/node_modules/@playwright/test/cli.js', 'test', '--config=/suite/playwright.config.ts', '--grep', `${escaped}(?:\\s|$)`, '--workers=1', '--retries=0', `--timeout=${input.timeoutMs}`, '--reporter=/runner/reporter.cjs'], {
  stdio: 'inherit', detached: true, env: { ...process.env, TESTSTUDIO_WORK_ITEM: JSON.stringify(input) }
})
let stopped = false
const stop = () => {
  if (stopped) return
  stopped = true
  try { process.kill(-child.pid, 'SIGTERM') } catch {}
  setTimeout(() => { try { process.kill(-child.pid, 'SIGKILL') } catch {} }, 5000).unref()
}
const timer = setTimeout(stop, input.timeoutMs + 45000)
process.on('SIGTERM', stop)
process.on('SIGINT', stop)
child.on('error', error => { console.error(error.message); clearTimeout(timer); process.exitCode = 2 })
child.on('exit', code => { clearTimeout(timer); process.exitCode = code ?? 2 })
