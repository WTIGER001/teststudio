import { spawn } from 'node:child_process'

const [cli, output, address] = process.argv.slice(2)
if (!cli || !output || !address) throw new Error('Recording requires an installed Playwright CLI, private output file, and start address')
const child = spawn(process.execPath, [cli, 'codegen', '--target=playwright-test', '--browser=chromium', '--block-service-workers', '--timeout=30000', '--output', output, address], { stdio: ['ignore', 'inherit', 'inherit'] })
let stopping = false
function stop() {
  if (stopping || child.exitCode !== null || child.signalCode !== null) return
  stopping = true
  child.kill('SIGINT')
  const force = setTimeout(() => child.kill('SIGKILL'), 8000)
  force.unref()
}
process.stdin.resume()
process.stdin.on('end', stop)
process.on('SIGTERM', stop)
process.on('SIGINT', stop)
const deadline = setTimeout(stop, 20 * 60 * 1000)
child.once('error', () => { clearTimeout(deadline); process.exit(1) })
child.once('exit', code => { clearTimeout(deadline); process.exit(stopping ? 0 : code ?? 1) })
