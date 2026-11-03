import fs from 'node:fs'
import path from 'node:path'
import net from 'node:net'
import { createServer } from 'node:http'
import { randomBytes, randomUUID, createHash, timingSafeEqual } from 'node:crypto'
import { spawn, execFileSync } from 'node:child_process'
import { fileURLToPath } from 'node:url'
import { startDemoTarget } from './lib/demo-target.mjs'
import { seedDemo, demoProject } from './lib/demo-seed.mjs'
import { snapshotDemoUI } from './lib/demo-ui.mjs'

const root = fileURLToPath(new URL('..', import.meta.url))
process.chdir(root)
const args = process.argv.slice(2), command = args.shift() ?? 'start'
let directory = path.join(root, '.data/demo')
while (args.length) { const argument = args.shift(); if (argument === '--data' && args.length) directory = path.resolve(args.shift()); else throw new Error('Use node scripts/demo.mjs start|stop|status [--data DIRECTORY]') }
if (!['start', 'stop', 'status'].includes(command)) throw new Error('Use node scripts/demo.mjs start|stop|status [--data DIRECTORY]')
const marker = path.join(directory, 'demo.json'), stateFile = path.join(directory, 'launcher.json'), lockFile = path.join(directory, 'launcher.lock')
function read(filename) { try { return JSON.parse(fs.readFileSync(filename, 'utf8')) } catch (error) { if (error.code === 'ENOENT') return null; throw error } }
function save(filename, value) { const temporary = `${filename}.${process.pid}.tmp`; fs.writeFileSync(temporary, JSON.stringify(value, null, 2), { mode: 0o600 }); fs.renameSync(temporary, filename) }
async function control(state, action) {
  if (!state?.control?.origin || !state.control.token || state.directory !== directory) return null
  const parsed = new URL(state.control.origin)
  if (parsed.protocol !== 'http:' || parsed.hostname !== '127.0.0.1') throw new Error('Demo control address must use loopback')
  const response = await fetch(`${parsed.origin}/${action}`, { method: action === 'stop' ? 'POST' : 'GET', headers: { Authorization: `Bearer ${state.control.token}` }, signal: AbortSignal.timeout(5000), redirect: 'error' })
  if (!response.ok) throw new Error('The saved launcher does not accept this demo ownership token')
  return response.json()
}
function print(status) {
  console.log(`TestStudio: ${status.origin}`)
  console.log(`Demo project: ${status.projectURL}`)
  console.log(`Local target: ${status.target}`)
  console.log(`Data directory: ${directory}`)
  console.log(`Stop: node scripts/demo.mjs stop --data ${JSON.stringify(directory)}`)
}
function ensureDirectory() {
  if (fs.existsSync(directory)) { if (!fs.lstatSync(directory).isDirectory() || fs.lstatSync(directory).isSymbolicLink()) throw new Error('Demo data must use a regular directory') }
  else fs.mkdirSync(directory, { recursive: true, mode: 0o700 })
  let owner = read(marker)
  if (!owner) {
    if (fs.readdirSync(directory).length) throw new Error('The selected directory contains other data; choose a new demo data directory')
    owner = { schema: 'teststudio.demo/v1', id: randomUUID(), seeded: false }; save(marker, owner)
  }
  if (owner.schema !== 'teststudio.demo/v1' || typeof owner.id !== 'string') throw new Error('The selected directory is not a TestStudio demo installation')
  return owner
}
async function freePort(preferred, excluded = new Set()) {
  for (let attempt = 0; attempt < 10; attempt++) {
    const socket = net.createServer()
    try {
      await new Promise((resolve, reject) => { socket.once('error', reject); socket.listen(attempt === 0 ? preferred : 0, '127.0.0.1', resolve) })
      const port = socket.address().port
      await new Promise(resolve => socket.close(resolve))
      if (!excluded.has(port)) return port
    } catch (error) { if (error.code !== 'EADDRINUSE') throw error }
  }
  throw new Error('No free local demo port is available')
}
async function requireFreePorts(ports) {
  for (const [name, port] of Object.entries(ports)) {
    if (!Number.isInteger(port) || port < 1 || port > 65535) throw new Error(`The saved ${name} port is invalid`)
    const socket = net.createServer()
    try { await new Promise((resolve, reject) => { socket.once('error', reject); socket.listen(port, '127.0.0.1', resolve) }) }
    catch (error) { throw new Error(`The saved ${name} port ${port} is unavailable. Stop its owner or use a new demo data directory.`, { cause: error }) }
    finally { if (socket.listening) await new Promise(resolve => socket.close(resolve)) }
  }
}
function fingerprint(roots) {
  const hash = createHash('sha256')
  const visit = filename => { const stat = fs.lstatSync(filename); if (stat.isSymbolicLink()) throw new Error('Runtime build input cannot be a symbolic link'); if (stat.isDirectory()) for (const name of fs.readdirSync(filename).sort()) visit(path.join(filename, name)); else { hash.update(filename); hash.update(fs.readFileSync(filename)) } }
  for (const name of roots) visit(name)
  return hash.digest('hex')
}
function runtimeImage() {
  const filename = path.join(root, '.data/demo-runtime-cache.json'), previous = read(filename)
  const key = fingerprint(['runners/playwright', 'scripts/build-playwright-runtime.sh'])
  if (previous?.key === key) {
    try { const image = execFileSync('docker', ['image', 'inspect', previous.image, '--format', '{{.Id}}'], { encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe'] }).trim(); if (image === previous.image) { console.log('Reusing the unchanged trusted browser runtime.'); return image } } catch {}
  }
  const base = /^FROM ([^\s]+)$/m.exec(fs.readFileSync('runners/playwright/Dockerfile', 'utf8'))?.[1]
  if (!base?.includes('@sha256:')) throw new Error('The browser base requires an immutable digest')
  try { execFileSync('docker', ['image', 'inspect', base], { stdio: 'ignore' }) } catch { execFileSync('docker', ['pull', base], { stdio: 'inherit' }) }
  execFileSync('bash', ['scripts/build-playwright-runtime.sh'], { stdio: 'inherit' })
  const image = execFileSync('docker', ['image', 'inspect', 'teststudio/playwright-source-runtime:local', '--format', '{{.Id}}'], { encoding: 'utf8' }).trim()
  save(filename, { key, image }); return image
}

async function serve() {
  const owner = ensureDirectory()
  const previous = read(stateFile)
  try { const status = await control(previous, 'status'); if (status) { console.log(status.ready ? 'The demo is already running.' : 'The demo is starting.'); print(status); return } } catch (error) { if (error.message.includes('ownership token')) throw error }
  if (fs.existsSync(lockFile)) {
    const lock = read(lockFile)
    try { process.kill(lock.pid, 0); throw new Error('A demo launcher still owns this directory. Use its stop command or another data directory.') } catch (error) { if (error.code !== 'ESRCH') throw error }
    fs.unlinkSync(lockFile)
  }
  fs.writeFileSync(lockFile, JSON.stringify({ pid: process.pid, owner: owner.id }), { flag: 'wx', mode: 0o600 })
  const children = new Set(), token = randomBytes(32).toString('hex')
  const log = fs.openSync(path.join(directory, 'processes.log'), 'a', 0o600)
  let target, rpc, stopping = false, status = { ready: false }, failure = null, finish
  const done = new Promise(resolve => { finish = resolve })
  const progress = message => { console.log(message); if (rpc) { status.message = message; save(stateFile, { ...status, control: { origin: `http://127.0.0.1:${rpc.address().port}`, token } }) } }
  async function shutdown(error = null) {
    if (stopping) return done
    stopping = true; failure = error
    if (error) console.error(error.message)
    for (const item of children) if (item.child.exitCode === null && item.child.signalCode === null) item.child.kill('SIGTERM')
    for (const item of children) { const timer = setTimeout(() => item.child.kill('SIGKILL'), 15000); try { await item.exited } finally { clearTimeout(timer) } }
    if (target) await target.close()
    if (rpc) { rpc.closeAllConnections(); await new Promise(resolve => rpc.close(resolve)) }
    fs.closeSync(log); fs.rmSync(lockFile, { force: true }); save(stateFile, { ...status, ready: false, stoppedAt: new Date().toISOString() })
    finish()
  }
  const interrupted = () => { void shutdown() }
  process.once('SIGINT', interrupted); process.once('SIGTERM', interrupted)
  function start(name, executable, arguments_, readyLine) {
    const runtimeEnvironment = Object.fromEntries(['PATH', 'HOME', 'TMPDIR', 'TEMP', 'TMP', 'DOCKER_HOST', 'DOCKER_CONTEXT', 'DOCKER_CONFIG', 'XDG_RUNTIME_DIR'].filter(key => process.env[key] !== undefined).map(key => [key, process.env[key]]))
    const child = spawn(executable, arguments_, { cwd: root, env: runtimeEnvironment, stdio: ['ignore', readyLine ? 'pipe' : log, log] })
    const item = { child, name, exited: null, listening: !readyLine }
    if (readyLine) {
      let pending = ''
      child.stdout.on('data', chunk => {
        fs.writeSync(log, chunk)
        pending += chunk.toString()
        const lines = pending.split('\n'); pending = lines.pop().slice(-4096)
        if (lines.some(line => line === readyLine)) item.listening = true
      })
    }
    item.exited = new Promise(resolve => { child.once('error', error => { resolve(); if (!stopping) void shutdown(new Error(`${name} could not start: ${error.message}`)) }); child.once('close', code => { resolve(); if (!stopping) void shutdown(new Error(`${name} exited with status ${code}. See ${path.join(directory, 'processes.log')}`)) }) })
    children.add(item); return item
  }
  try {
    progress('Preparing local server and UI builds. Docker must be running for the browser example.')
    const build = JSON.parse(execFileSync(process.execPath, ['scripts/build-selftest.mjs', '--local-only'], { encoding: 'utf8', stdio: ['ignore', 'pipe', 'inherit'] }))
    const ui = snapshotDemoUI(build.ui, directory)
    const image = runtimeImage()
    if (owner.ports) await requireFreePorts(owner.ports)
    const excluded = new Set(Object.values(owner.ports ?? {}))
    rpc = createServer((request, response) => {
      const actual = Buffer.from(request.headers.authorization ?? ''), expected = Buffer.from(`Bearer ${token}`)
      response.setHeader('Cache-Control', 'no-store')
      if (actual.length !== expected.length || !timingSafeEqual(actual, expected)) { response.writeHead(403).end(); return }
      response.setHeader('Content-Type', 'application/json')
      if (request.method === 'GET' && request.url === '/status') { response.end(JSON.stringify(status)); return }
      if (request.method === 'POST' && request.url === '/stop') { response.end(JSON.stringify({ stopping: true }), () => { void shutdown() }); return }
      response.writeHead(404).end()
    })
    let controlPort
    for (;;) { await new Promise((resolve, reject) => { rpc.once('error', reject); rpc.listen(0, '127.0.0.1', resolve) }); controlPort = rpc.address().port; if (!excluded.has(controlPort)) break; await new Promise(resolve => rpc.close(resolve)) }
    if (!owner.ports) {
      const used = new Set([controlPort]), api = await freePort(8040, used); used.add(api)
      const artifacts = await freePort(8041, used); used.add(artifacts)
      const fixture = await freePort(8060, used)
      owner.ports = { api, artifacts, fixture }; save(marker, owner)
    }
    target = await startDemoTarget(owner.ports.fixture)
    const origin = `http://127.0.0.1:${owner.ports.api}`, artifactOrigin = `http://127.0.0.1:${owner.ports.artifacts}`
    status = { ready: false, directory, pid: process.pid, owner: owner.id, origin, artifactOrigin, target: target.origin, projectURL: `${origin}/projects/${demoProject}/catalog`, startedAt: new Date().toISOString() }
    progress('Starting the local TestStudio API and separate source worker.')
    const database = path.join(directory, 'teststudio.db')
    const server = start('TestStudio server', build.server, ['--local', '--source-bundles', '--addr', `127.0.0.1:${owner.ports.api}`, '--public-url', origin, '--artifact-addr', `127.0.0.1:${owner.ports.artifacts}`, '--artifact-origin', artifactOrigin, '--database', database, '--artifact-directory', path.join(directory, 'artifacts'), '--ui', ui], `TestStudio listens at ${origin}`)
    let available = false
    for (let attempt = 0; attempt < 100 && !stopping; attempt++) { if (server.listening) { try { available = (await fetch(origin + '/healthz', { signal: AbortSignal.timeout(1000) })).ok } catch {} } if (available) break; await new Promise(resolve => setTimeout(resolve, 100)) }
    if (!available || stopping) throw new Error('The local demo API did not start; inspect its process log')
    start('Source preparation worker', build.preparation, ['--database', database, '--playwright-runtime', image, '--execution-location', 'local', '--build-directory', path.join(directory, 'source-builds')])
    await seedDemo({ root, origin, target: target.origin, owner, save: value => save(marker, value), progress })
    if (!stopping) { status.ready = true; status.browserRun = owner.browserRun; status.runs = owner.runs; progress('Demo ready. Open TestStudio and select Open local workspace.'); print(status); console.log('Press Ctrl+C to stop. The demo data and evidence stay available for the next start.') }
    await done
  } catch (error) { await shutdown(error) }
  finally { process.off('SIGINT', interrupted); process.off('SIGTERM', interrupted) }
  if (failure) process.exitCode = 1
}

try {
  if (command === 'start') await serve()
  else {
    const state = read(stateFile)
    let status
    try { status = await control(state, command === 'stop' ? 'stop' : 'status') } catch (error) { if (error.message.includes('ownership token')) throw error }
    if (!status) console.log('No running demo launcher is available for this data directory.')
    else if (command === 'status') { console.log(status.ready ? 'Demo ready.' : 'Demo starting.'); print(status) }
    else console.log('Demo shutdown requested. Its data and evidence are retained.')
  }
} catch (error) { console.error(error.message); process.exitCode = 1 }
