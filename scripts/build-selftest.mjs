import fs from 'node:fs'
import path from 'node:path'
import crypto from 'node:crypto'
import { execFileSync } from 'node:child_process'
import { fileURLToPath } from 'node:url'

const root = fileURLToPath(new URL('..', import.meta.url))
process.chdir(root)
const directory = '.data/selftest-build'
fs.mkdirSync(directory, { recursive: true, mode: 0o700 })
const filename = `${directory}/build-cache.json`
let cache = {}
try { cache = JSON.parse(fs.readFileSync(filename, 'utf8')) } catch (error) { if (error.code !== 'ENOENT' && !(error instanceof SyntaxError)) throw error }
const digest = values => crypto.createHash('sha256').update(JSON.stringify(values)).digest('hex')
const fileHash = filename => fs.existsSync(filename) ? crypto.createHash('sha256').update(fs.readFileSync(filename)).digest('hex') : ''
function fingerprint(roots, omitGoTests = false, omitTypeScriptTests = false) {
  const hash = crypto.createHash('sha256')
  function visit(filename) {
    if (!fs.existsSync(filename)) return
    const stat = fs.lstatSync(filename)
    if (stat.isSymbolicLink()) throw new Error(`Self-test build input must not be a symbolic link: ${filename}`)
    if (stat.isDirectory()) { if (path.basename(filename) === 'node_modules') return; for (const name of fs.readdirSync(filename).sort()) visit(path.join(filename, name)); return }
    if (!stat.isFile() || omitGoTests && filename.endsWith('_test.go') || omitTypeScriptTests && /\.(?:test|spec)\.ts$/.test(filename)) return
    hash.update(filename + '\0'); hash.update(fs.readFileSync(filename)); hash.update('\0')
  }
  for (const filename of roots) visit(filename)
  return hash.digest('hex')
}
function run(bin, args, options = {}) { execFileSync(bin, args, { stdio: ['ignore', 2, 2], ...options }) }
function inspect(image) {
  try { return execFileSync('docker', ['image', 'inspect', image, '--format', '{{.Id}} {{.Architecture}}'], { encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe'] }).trim().split(' ') }
  catch (error) { if (/no such image/i.test(String(error.stderr))) return null; throw new Error('Docker image state cannot be verified; build cache was not changed.') }
}
function save() {
  const temporary = `${filename}.${process.pid}.tmp`
  fs.writeFileSync(temporary, JSON.stringify(cache, null, 2), { mode: 0o600 })
  fs.renameSync(temporary, filename)
}

const assetsOnly = process.argv.includes('--assets-only')
const localOnly = assetsOnly || process.argv.includes('--local-only')
let base = '', baseImage = '', architecture = process.arch === 'arm64' ? 'arm64' : 'amd64'
if (!localOnly) {
  const dockerfile = fs.readFileSync('tests/selftest/Dockerfile', 'utf8')
  base = /^FROM ([^\s]+)$/m.exec(dockerfile)?.[1]
  if (!base?.includes('@sha256:')) throw new Error('The self-test base must use an immutable digest')
  let baseMetadata = inspect(base)
  if (!baseMetadata) { run('docker', ['pull', base]); baseMetadata = inspect(base) }
  if (!baseMetadata) throw new Error('The pinned self-test base is unavailable')
  ;[baseImage, architecture] = baseMetadata
  if (!['amd64', 'arm64'].includes(architecture)) throw new Error('Unsupported self-test runtime architecture')
}

const uiKey = fingerprint(['web/src', 'web/public', 'web/angular.json', 'web/tsconfig.json', 'web/tsconfig.app.json', 'web/package.json', 'web/package-lock.json'], false, true)
const uiDirectory = 'web/dist/web/browser'
let uiOutput = fs.existsSync(`${uiDirectory}/index.html`) ? fingerprint([uiDirectory]) : ''
if (cache.uiKey !== uiKey || !uiOutput || cache.uiOutput !== uiOutput) {
  if (!fs.existsSync('web/node_modules/playwright-core/package.json')) run('npm', ['--prefix', 'web', 'ci'])
  run('npm', ['--prefix', 'web', 'run', 'build'])
  uiOutput = fingerprint([uiDirectory]); cache.uiKey = uiKey; cache.uiOutput = uiOutput
  save()
} else console.error('Reusing the unchanged self-test UI build.')

const traceViewer = execFileSync(process.execPath, ['scripts/build-trace-viewer.mjs'], { encoding: 'utf8', stdio: ['ignore', 'pipe', 2] }).trim()
const traceOutput = fingerprint([traceViewer])
if (assetsOnly) { console.log(JSON.stringify({ ui: path.resolve(uiDirectory), uiOutput, traceViewer, traceOutput })); process.exit(0) }

const goVersion = execFileSync('go', ['version'], { encoding: 'utf8' }).trim()
const goKey = digest([goVersion, fingerprint(['go.mod', 'go.sum', 'internal', 'adapters', 'cmd', 'tests/selftest/seed-access'], true)])
const targets = localOnly ? [
  { id: 'controller', package: './cmd/teststudio', platform: `${process.platform}-${process.arch}`, env: {} },
  { id: 'preparation-worker', package: './cmd/teststudio-prepare', platform: `${process.platform}-${process.arch}`, env: {} }
] : [
  { id: 'controller', package: './cmd/teststudio', platform: `${process.platform}-${process.arch}`, env: {} },
  { id: 'teststudio', package: './cmd/teststudio', platform: `linux-${architecture}-cgo0`, env: { CGO_ENABLED: '0', GOOS: 'linux', GOARCH: architecture } },
  { id: 'teststudio-publish', package: './cmd/teststudio-publish', platform: `linux-${architecture}-cgo0`, env: { CGO_ENABLED: '0', GOOS: 'linux', GOARCH: architecture } },
  { id: 'access-fixture', package: './tests/selftest/seed-access', platform: `linux-${architecture}-cgo0`, env: { CGO_ENABLED: '0', GOOS: 'linux', GOARCH: architecture } }
]
cache.binaries ??= {}
for (const target of targets) {
  const output = `${directory}/${target.id}`, key = digest([goKey, target.package, target.platform])
  const previous = cache.binaries[target.id], actual = fileHash(output)
  if (!actual || previous?.key !== key || previous?.hash !== actual) {
    run('go', ['build', '-o', output, target.package], { env: { ...process.env, ...target.env } })
    cache.binaries[target.id] = { key, hash: fileHash(output) }; save()
  } else console.error(`Reusing unchanged self-test binary: ${target.id}.`)
}
if (localOnly) {
  console.log(JSON.stringify({ server: path.resolve(directory, 'controller'), preparation: path.resolve(directory, 'preparation-worker'), traceViewer, ui: path.resolve(uiDirectory) }))
  process.exit(0)
}
const imageKey = digest([baseImage, architecture, uiOutput, traceOutput, cache.binaries.teststudio.hash, cache.binaries['access-fixture'].hash, cache.binaries['teststudio-publish'].hash, fingerprint(['tests/selftest/Dockerfile', 'tests/selftest/playwright', 'runners/playwright'])])
if (cache.imageKey !== imageKey || !cache.image || !inspect(cache.image)) {
  run('docker', ['build', '--pull=false', '--build-arg', `TARGETARCH=${architecture}`, '-q', '-f', 'tests/selftest/Dockerfile', '-t', 'teststudio/selftests:local', '.'])
  const image = inspect('teststudio/selftests:local')?.[0]
  if (!image || !/^sha256:[a-f0-9]{64}$/.test(image)) throw new Error('The built self-test image cannot be verified')
  cache.imageKey = imageKey; cache.image = image; save()
} else console.error('Reusing the unchanged immutable self-test image.')
console.log(cache.image)
