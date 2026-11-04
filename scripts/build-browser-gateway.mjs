import fs from 'node:fs'
import path from 'node:path'
import os from 'node:os'
import crypto from 'node:crypto'
import { execFileSync } from 'node:child_process'
import { fileURLToPath } from 'node:url'

const root = fileURLToPath(new URL('..', import.meta.url))
process.chdir(root)
const architecture = execFileSync('docker', ['info', '--format', '{{.Architecture}}'], { encoding: 'utf8' }).trim()
const arch = ({ aarch64: 'arm64', arm64: 'arm64', x86_64: 'amd64', amd64: 'amd64' })[architecture]
if (!arch) throw new Error('Unsupported browser gateway architecture')
const hash = crypto.createHash('sha256').update(arch).update(execFileSync('go', ['version'])).update(fs.readFileSync('scripts/build-browser-gateway.mjs'))
for (const name of ['go.mod', 'go.sum']) hash.update(fs.readFileSync(name))
const dependencies = execFileSync('go', ['list', '-deps', '-f', '{{if .Module}}{{if .Module.Main}}{{.Dir}}{{end}}{{end}}', './cmd/teststudio-browser-gateway'], { encoding: 'utf8' }).trim().split(/\r?\n/).filter(Boolean)
for (const directory of [...dependencies, 'runners/browser-gateway']) {
  for (const name of fs.readdirSync(directory).sort()) if (!name.endsWith('_test.go')) hash.update(name).update(fs.readFileSync(path.join(directory, name)))
}
const key = hash.digest('hex'), cacheFile = '.data/browser-gateway-cache.json'
let cache
try { cache = JSON.parse(fs.readFileSync(cacheFile, 'utf8')) } catch (error) { if (error.code !== 'ENOENT' && !(error instanceof SyntaxError)) throw error }
if (cache?.key === key) {
  try {
    const actual = execFileSync('docker', ['image', 'inspect', cache.image, '--format', '{{.Id}}'], { encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe'] }).trim()
    if (actual === cache.image) { console.error('Reusing the unchanged trusted browser gateway.'); console.log(actual); process.exit(0) }
  } catch {}
}
const directory = fs.mkdtempSync(path.join(os.tmpdir(), 'teststudio-browser-gateway-'))
try {
  fs.mkdirSync(path.join(directory, 'output'), { mode: 0o777 }); fs.chmodSync(path.join(directory, 'output'), 0o777)
  execFileSync('go', ['build', '-trimpath', '-o', path.join(directory, 'teststudio-browser-gateway'), './cmd/teststudio-browser-gateway'], { env: { ...process.env, CGO_ENABLED: '0', GOOS: 'linux', GOARCH: arch }, stdio: ['ignore', 2, 2] })
  execFileSync('docker', ['build', '--pull=false', '-q', '-f', path.join(root, 'runners/browser-gateway/Dockerfile'), '-t', 'teststudio/browser-gateway:local', directory], { stdio: ['ignore', 2, 2] })
  const image = execFileSync('docker', ['image', 'inspect', 'teststudio/browser-gateway:local', '--format', '{{.Id}}'], { encoding: 'utf8' }).trim()
  if (!/^sha256:[a-f0-9]{64}$/.test(image)) throw new Error('The browser gateway image cannot be verified')
  fs.mkdirSync('.data', { recursive: true, mode: 0o700 })
  fs.writeFileSync(cacheFile, JSON.stringify({ key, image }), { mode: 0o600 })
  console.log(image)
} finally { fs.rmSync(directory, { recursive: true, force: true }) }
