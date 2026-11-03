import { spawn, execFile } from 'node:child_process'
import { readFile, stat, unlink } from 'node:fs/promises'
import { randomUUID } from 'node:crypto'
import { expect } from '@playwright/test'

export async function startPublisher(clientName: string) {
  const filename = `/tmp/teststudio-cli-${randomUUID()}.json`
  const environment = { PATH: process.env.PATH, TMPDIR: '/tmp' }
  const child = spawn('/app/teststudio-publish', ['login', '--browser', '--no-open', '--server', 'http://127.0.0.1:8040', '--client-name', clientName, '--session-file', filename], { env: environment, stdio: ['ignore', 'pipe', 'pipe'] })
  let output = '', diagnostic = '', closed = false
  child.stdout.on('data', data => { if (output.length < 65536) output += data.toString() })
  child.stderr.on('data', data => { if (diagnostic.length < 65536) diagnostic += data.toString() })
  const exited = new Promise<number | null>((resolve, reject) => { child.once('error', reject); child.once('close', code => { closed = true; resolve(code) }) })
  void exited.catch(() => {})
  const cleanup = async () => {
    if (!closed) {
      child.kill('SIGTERM')
      const force = setTimeout(() => child.kill('SIGKILL'), 3000)
      try { await exited } finally { clearTimeout(force) }
    }
    try { await readFile(filename); await new Promise<void>(resolve => execFile('/app/teststudio-publish', ['logout', '--session-file', filename], { env: environment, timeout: 5000, maxBuffer: 4096 }, () => resolve())) } catch {}
    await unlink(filename).catch(() => {})
    expect(closed, 'Owned publisher process exits').toBe(true)
  }
  try {
    await expect.poll(() => /Open this address to confirm CLI sign-in: (http[^\s]+)/.test(output) && /Terminal code: [A-F0-9]{4}-[A-F0-9]{4}/.test(output), { timeout: 10000 }).toBe(true)
  } catch (error) { await cleanup(); throw error }
  const address = /Open this address to confirm CLI sign-in: (http[^\s]+)/.exec(output)![1]
  const code = /Terminal code: ([A-F0-9]{4}-[A-F0-9]{4})/.exec(output)![1]
  return {
    address, code, cleanup,
    async finished() { const exit = await exited; expect(exit, diagnostic || 'Publisher sign-in succeeds').toBe(0); const info = await stat(filename); expect(info.mode & 0o777).toBe(0o600); const session = JSON.parse(await readFile(filename, 'utf8')); expect(output).not.toContain(session.cookie); expect(output).not.toContain(session.csrfToken); return session },
    async declined() { expect(await exited).toBe(1); expect(diagnostic).toContain('declined'); await expect(stat(filename)).rejects.toThrow() }
  }
}
