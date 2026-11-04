import fs from 'node:fs'
import path from 'node:path'
import os from 'node:os'
import crypto from 'node:crypto'
import { execFileSync } from 'node:child_process'
import { fileURLToPath } from 'node:url'

const root = fileURLToPath(new URL('..', import.meta.url))
process.chdir(root)
const arguments_ = process.argv.slice(2)
if (arguments_.some(value => !/^--arch=(amd64|arm64)$/.test(value)) || arguments_.length > 1) throw new Error('Use node scripts/build-service-image.mjs [--arch=amd64|arm64]')
const hostName = execFileSync('docker', ['info', '--format', '{{.Architecture}}'], { encoding: 'utf8' }).trim()
const host = ({ aarch64: 'arm64', arm64: 'arm64', x86_64: 'amd64', amd64: 'amd64' })[hostName]
if (!host) throw new Error('Unsupported Docker host architecture')
const architecture = arguments_[0]?.split('=')[1] ?? host
const directory = path.join(root, '.data/service-build', architecture)
fs.mkdirSync(directory, { recursive: true, mode: 0o700 })
const cacheFile = path.join(directory, 'cache.json')
let cache = {}
try { cache = JSON.parse(fs.readFileSync(cacheFile, 'utf8')) } catch (error) { if (error.code !== 'ENOENT' && !(error instanceof SyntaxError)) throw error }
const digest = values => crypto.createHash('sha256').update(JSON.stringify(values)).digest('hex')
const hashFile = name => fs.existsSync(name) ? crypto.createHash('sha256').update(fs.readFileSync(name)).digest('hex') : ''
function inspect(reference) {
  try { return JSON.parse(execFileSync('docker', ['image', 'inspect', reference], { encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe'] }))[0] }
  catch (error) { if (/no such image|no such object/i.test(String(error.stderr))) return null; throw new Error('The image state cannot be verified; the build cache remains unchanged.') }
}
function save() { const temporary = `${cacheFile}.${process.pid}.tmp`; fs.writeFileSync(temporary, JSON.stringify(cache, null, 2), { mode: 0o600 }); fs.renameSync(temporary, cacheFile) }
const assets = JSON.parse(execFileSync(process.execPath, ['scripts/build-selftest.mjs', '--assets-only'], { encoding: 'utf8', stdio: ['ignore', 'pipe', 2] }))
const goEnvironment = { ...process.env, GOWORK: 'off', GOFLAGS: '', GOOS: 'linux', GOARCH: architecture, CGO_ENABLED: '0' }
const buildArguments = ['build', '-mod=readonly', '-trimpath', '-ldflags=-s -w']
const goVersion = execFileSync('go', ['version'], { encoding: 'utf8', env: goEnvironment }).trim()
cache.binaries ??= {}
for (const name of ['teststudio', 'teststudio-health']) {
  const package_ = './cmd/' + name
  const dependencies = execFileSync('go', ['list', '-mod=readonly', '-deps', '-f', '{{if .Module}}{{if .Module.Main}}{{.Dir}}{{end}}{{end}}', package_], { encoding: 'utf8', env: goEnvironment }).trim().split(/\r?\n/).filter(Boolean)
  const inputs = [goVersion, architecture, buildArguments, { GOWORK: 'off', GOFLAGS: '', CGO_ENABLED: '0', GOOS: 'linux', GOARCH: architecture, GOEXPERIMENT: process.env.GOEXPERIMENT ?? '', GOAMD64: process.env.GOAMD64 ?? '', GOARM64: process.env.GOARM64 ?? '' }, hashFile('go.mod'), hashFile('go.sum')]
  for (const dependency of dependencies.sort()) for (const file of fs.readdirSync(dependency).sort()) if (file.endsWith('.go') && !file.endsWith('_test.go')) inputs.push(file, hashFile(path.join(dependency, file)))
  const key = digest(inputs), binary = path.join(directory, name)
  if (cache.binaries[name]?.key !== key || cache.binaries[name]?.hash !== hashFile(binary) || !fs.existsSync(binary)) {
    execFileSync('go', [...buildArguments, '-o', binary, package_], { env: goEnvironment, stdio: ['ignore', 2, 2] })
    cache.binaries[name] = { key, hash: hashFile(binary) }; save()
  } else console.error(`Reusing unchanged service binary: ${name}.`)
}
const dockerfile = path.join(root, 'deploy/service/Dockerfile')
const sourceKey = digest([architecture, hashFile(dockerfile), hashFile('scripts/build-service-image.mjs'), cache.binaries, assets.uiOutput, assets.traceOutput])
if (cache.sourceKey === sourceKey && cache.image) {
  const previous = inspect(cache.image)
  if (previous?.Id === cache.image && previous.Architecture === architecture) { console.error('Reusing the unchanged service image.'); console.log(JSON.stringify({ image: cache.image, architecture, sourceKey })); process.exit(0) }
}
const base = /^FROM --platform=\$\{BUILDPLATFORM\} ([^\s]+) AS files$/m.exec(fs.readFileSync(dockerfile, 'utf8'))?.[1]
if (!base?.includes('@sha256:')) throw new Error('The service certificate source requires an immutable digest')
if (!inspect(base)) execFileSync('docker', ['pull', '--platform=linux/' + host, base], { stdio: ['ignore', 2, 2] })
const context = fs.mkdtempSync(path.join(os.tmpdir(), 'teststudio-service-image-'))
try {
  for (const name of ['teststudio', 'teststudio-health']) fs.copyFileSync(path.join(directory, name), path.join(context, name))
  fs.cpSync(assets.ui, path.join(context, 'ui'), { recursive: true, dereference: false })
  fs.cpSync(assets.traceViewer, path.join(context, 'trace-viewer'), { recursive: true, dereference: false })
  execFileSync('docker', ['build', '--network=none', '--pull=false', '--platform=linux/' + architecture, '--build-arg', 'BUILDPLATFORM=linux/' + host, '--build-arg', 'TARGETARCH=' + architecture, '--build-arg', 'SOURCE_KEY=' + sourceKey, '-q', '-f', dockerfile, '-t', 'teststudio/service:' + architecture + '-local', context], { stdio: ['ignore', 2, 2] })
  const image = inspect('teststudio/service:' + architecture + '-local')
  if (!image || image.Architecture !== architecture || !/^sha256:[a-f0-9]{64}$/.test(image.Id) || image.Config.User !== '65532:65532') throw new Error('The built service image metadata does not match the requested package')
  cache.sourceKey = sourceKey; cache.image = image.Id; save()
  console.log(JSON.stringify({ image: image.Id, architecture, sourceKey }))
} finally { fs.rmSync(context, { recursive: true, force: true }) }
