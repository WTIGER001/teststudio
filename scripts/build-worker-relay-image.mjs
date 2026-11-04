import fs from 'node:fs'
import path from 'node:path'
import os from 'node:os'
import crypto from 'node:crypto'
import { execFileSync } from 'node:child_process'
import { fileURLToPath } from 'node:url'
import { dockerBuildBackend } from './lib/docker-build.mjs'

const root = fileURLToPath(new URL('..', import.meta.url))
process.chdir(root)
const arguments_ = process.argv.slice(2)
if (arguments_.some(value => !/^--arch=(amd64|arm64)$/.test(value) && !/^--buildx=.+$/.test(value) && value !== '--local-only') || new Set(arguments_.map(value => value.split('=')[0])).size !== arguments_.length) throw new Error('Use node scripts/build-worker-relay-image.mjs [--arch=amd64|arm64] [--buildx=PATH] [--local-only]')
const localOnly = arguments_.includes('--local-only')
const run = (command, args, options = {}) => execFileSync(command, args, { encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe'], ...options })
const contextName = run('docker', ['context', 'show']).trim()
const contextEndpoint = run('docker', ['context', 'inspect', contextName, '--format', '{{.Endpoints.docker.Host}}']).trim()
if (localOnly && [contextEndpoint, process.env.DOCKER_HOST].filter(Boolean).some(value => !/^(unix|npipe):\/\//.test(value))) throw new Error('Local-only relay builds require a local Docker socket.')
const host = ({ aarch64: 'arm64', arm64: 'arm64', x86_64: 'amd64', amd64: 'amd64' })[run('docker', ['info', '--format', '{{.Architecture}}']).trim()]
if (!host) throw new Error('Unsupported Docker host architecture')
const architecture = arguments_.find(value => value.startsWith('--arch='))?.split('=')[1] ?? host
const backend = dockerBuildBackend({ buildxPath: arguments_.find(value => value.startsWith('--buildx='))?.slice('--buildx='.length), architecture, host })
const directory = path.join(root, '.data/worker-relay-build', architecture)
fs.mkdirSync(directory, { recursive: true, mode: 0o700 })
const cacheFile = path.join(directory, 'cache.json')
let cache = {}
try { cache = JSON.parse(fs.readFileSync(cacheFile, 'utf8')) } catch (error) { if (error.code !== 'ENOENT' && !(error instanceof SyntaxError)) throw error }
const digest = values => crypto.createHash('sha256').update(JSON.stringify(values)).digest('hex')
const hashFile = name => fs.existsSync(name) ? crypto.createHash('sha256').update(fs.readFileSync(name)).digest('hex') : ''
function save() {
  const temporary = `${cacheFile}.${process.pid}.tmp`
  fs.writeFileSync(temporary, JSON.stringify(cache, null, 2), { mode: 0o600 })
  fs.renameSync(temporary, cacheFile)
}
function inspect(reference) {
  try { return JSON.parse(run('docker', ['image', 'inspect', reference]))[0] }
  catch (error) { if (/no such image|no such object/i.test(String(error.stderr))) return null; throw new Error('Cannot inspect the relay image state; cached ownership remains unchanged.') }
}
const goSettings = { GOENV: 'off', GOWORK: 'off', GOFLAGS: '', GOTOOLCHAIN: 'local', GOOS: 'linux', GOARCH: architecture, CGO_ENABLED: '0', GOAMD64: 'v1', GOARM64: 'v8.0', GOEXPERIMENT: '' }
const goEnvironment = { ...process.env, ...goSettings }
const dependencyAudit = JSON.parse(run('python3', ['scripts/verify-worker-relay-dependencies.py'], { env: goEnvironment }))
const buildArguments = ['build', '-mod=readonly', '-trimpath', '-buildvcs=false', '-ldflags=-s -w']
const goVersion = run('go', ['version'], { env: goEnvironment }).trim()
const binaries = {}
const rebuiltBinaries = []
const reusedBinaries = []
cache.binaries ??= {}
const template = '{{if not .Standard}}{{.Dir}}{{"\t"}}{{join .GoFiles ","}}{{"\t"}}{{join .EmbedFiles ","}}{{"\t"}}{{join .SFiles ","}}{{"\t"}}{{join .SysoFiles ","}}{{end}}'
for (const name of ['teststudio-exchange', 'teststudio-health']) {
  const package_ = './cmd/' + name
  const rows = run('go', ['list', '-mod=readonly', '-deps', '-f', template, package_], { env: goEnvironment }).split(/\r?\n/).filter(Boolean)
  const inputs = [goVersion, goSettings, buildArguments, hashFile('go.mod'), hashFile('go.sum')]
  for (const row of rows.sort()) {
    const [folder, ...groups] = row.split('\t')
    for (const name of groups.flatMap(value => value.split(',')).filter(Boolean).sort()) {
      const file = path.resolve(folder, name)
      const relative = path.relative(root, file)
      if (relative === '..' || relative.startsWith('..' + path.sep) || path.isAbsolute(relative)) throw new Error('Relay source dependencies must remain inside the product repository.')
      inputs.push(path.relative(root, file), hashFile(file))
    }
  }
  const key = digest(inputs), binary = path.join(directory, name)
  if (cache.binaries[name]?.key !== key || cache.binaries[name]?.hash !== hashFile(binary) || !fs.existsSync(binary)) {
    run('go', [...buildArguments, '-o', binary, package_], { env: goEnvironment, stdio: ['ignore', 2, 2] })
    cache.binaries[name] = { key, hash: hashFile(binary) }
    rebuiltBinaries.push(name)
    save()
  } else reusedBinaries.push(name)
  binaries[name] = { hash: hashFile(binary) }
}
const certificateSource = JSON.parse(fs.readFileSync('deploy/service/certificate-source.json', 'utf8'))
if (!/^[^\s]+@sha256:[a-f0-9]{64}$/.test(certificateSource.image) || !/^linux\/(amd64|arm64)$/.test(certificateSource.platform) || certificateSource.path !== '/etc/ssl/certs/ca-certificates.crt') throw new Error('The relay requires the approved immutable service certificate source.')
const certificateFile = path.join(directory, 'ca-certificates.crt')
let certificatesReused = false
if (cache.certificates?.source === digest(certificateSource) && cache.certificates.hash === hashFile(certificateFile) && fs.existsSync(certificateFile)) certificatesReused = true
else {
  if (!inspect(certificateSource.image)) {
    if (localOnly) throw new Error('The pinned certificate image is unavailable locally. No image is pulled in local-only mode.')
    run('docker', ['pull', '--platform=' + certificateSource.platform, certificateSource.image], { stdio: ['ignore', 2, 2] })
  }
  const name = 'teststudio-relay-certificates-' + crypto.randomUUID()
  let created = false
  try {
    created = true
    run('docker', ['create', '--name', name, '--platform=' + certificateSource.platform, '--network=none', '--read-only', '--entrypoint=/bin/true', certificateSource.image])
    run('docker', ['cp', name + ':' + certificateSource.path, certificateFile])
    const bundle = fs.readFileSync(certificateFile, 'utf8')
    if (!bundle.includes('-----BEGIN CERTIFICATE-----')) throw new Error('The extracted relay certificate bundle is invalid.')
    new crypto.X509Certificate(bundle)
    fs.chmodSync(certificateFile, 0o600)
    cache.certificates = { source: digest(certificateSource), hash: hashFile(certificateFile) }
    save()
  } finally {
    if (created) {
      run('docker', ['rm', '--force', '--volumes', name])
      try { run('docker', ['container', 'inspect', name]); throw new Error('The owned relay certificate container remains.') }
      catch (error) { if (!/no such/i.test(String(error.stderr))) throw error }
    }
  }
}
const dockerfile = path.join(root, 'deploy/worker-relay/Dockerfile')
const sourceKey = digest([architecture, backend.version, binaries, certificateSource, hashFile(certificateFile), hashFile(dockerfile), hashFile('scripts/build-worker-relay-image.mjs'), hashFile('scripts/lib/docker-build.mjs'), hashFile('scripts/verify-worker-relay-dependencies.py')])
const validEnvironment = env => env == null || env.length <= 1 && env.every(value => value === 'PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin')
const validImage = image => image && validEnvironment(image.Config.Env) && /^sha256:[a-f0-9]{64}$/.test(image.Id) && image.Architecture === architecture && image.Config.User === '65532:65532' && image.Config.Labels?.['teststudio.package'] === 'worker-relay/v1' && image.Config.Labels?.['teststudio.source-key'] === sourceKey && Object.keys(image.Config.Volumes ?? {}).length === 0 && JSON.stringify(image.Config.Entrypoint) === JSON.stringify(['/app/teststudio-exchange', '--addr=0.0.0.0:8043'])
const report = (image, imageReused) => console.log(JSON.stringify({ image: image.Id, architecture, sourceKey, imageReused, rebuiltBinaries, reusedBinaries, certificatesReused, localOnly, dependencyAudit, artifacts: { binaries, certificateHash: hashFile(certificateFile) } }))
if (cache.sourceKey === sourceKey && cache.image) {
  const image = inspect(cache.image)
  if (validImage(image)) { report(image, true); process.exit(0) }
}
const context = fs.mkdtempSync(path.join(os.tmpdir(), 'teststudio-relay-image-'))
try {
  for (const name of ['teststudio-exchange', 'teststudio-health']) fs.copyFileSync(path.join(directory, name), path.join(context, name))
  fs.copyFileSync(certificateFile, path.join(context, 'ca-certificates.crt'))
  fs.chmodSync(path.join(context, 'ca-certificates.crt'), 0o644)
  const tag = 'teststudio/worker-relay:' + architecture + '-local'
  run(backend.command, [...backend.prefix, '--network=none', '--pull=false', '--platform=linux/' + architecture, '--build-arg', 'TARGETARCH=' + architecture, '--build-arg', 'SOURCE_KEY=' + sourceKey, '-q', '-f', dockerfile, '-t', tag, context], { stdio: ['ignore', 2, 2] })
  const image = inspect(tag)
  if (!validImage(image)) throw new Error('The built relay image does not match its package identity.')
  cache.sourceKey = sourceKey
  cache.image = image.Id
  save()
  report(image, false)
} finally {
  fs.rmSync(context, { recursive: true, force: true })
}
