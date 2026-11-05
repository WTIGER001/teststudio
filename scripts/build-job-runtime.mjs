import fs from 'node:fs'
import path from 'node:path'
import os from 'node:os'
import crypto from 'node:crypto'
import { execFileSync } from 'node:child_process'
import { fileURLToPath } from 'node:url'
import { dockerBuildBackend } from './lib/docker-build.mjs'

const root = fileURLToPath(new URL('..', import.meta.url))
process.chdir(root)
const args = process.argv.slice(2)
if (args.some(value => !/^--image=.+$/.test(value) && !/^--buildx=.+$/.test(value)) || new Set(args.map(value => value.split('=')[0])).size !== args.length) throw new Error('Use node scripts/build-job-runtime.mjs --image=IMMUTABLE_IMAGE [--buildx=PATH]')
const reference = args.find(value => value.startsWith('--image='))?.slice(8)
if (!/^(sha256:[a-f0-9]{64}|[A-Za-z0-9][A-Za-z0-9._:/-]*@sha256:[a-f0-9]{64})$/.test(reference ?? '')) throw new Error('Select an existing immutable source runtime image')
const inspect = reference => {
  try { return JSON.parse(execFileSync('docker', ['image', 'inspect', reference], { encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe'] }))[0] }
  catch { throw new Error('The selected local image cannot be inspected') }
}
const base = inspect(reference), labels = base.Config.Labels ?? {}
const runner = labels['teststudio.runtime']?.startsWith('go') ? 'go' : labels['teststudio.runtime']?.startsWith('playwright') ? 'playwright' : ''
if (!runner || labels['teststudio.runtime-kind'] !== 'source-base/v1' || labels['teststudio.work-item-bytes'] !== '4194304' || !['amd64', 'arm64'].includes(base.Architecture) || base.Os !== 'linux') throw new Error('The selected image is not a supported immutable TestStudio source runtime')
const architecture = base.Architecture
const backend = dockerBuildBackend({ buildxPath: args.find(value => value.startsWith('--buildx='))?.slice(9), architecture, requireBuildKit: true })
const hash = value => crypto.createHash('sha256').update(value).digest('hex')
const fileHash = name => fs.existsSync(name) ? hash(fs.readFileSync(name)) : ''
const directory = path.join(root, '.data/job-runtime-build', runner + '-' + architecture + '-' + base.Id.slice(7))
fs.mkdirSync(directory, { recursive: true, mode: 0o700 })
const cachePath = path.join(directory, 'cache.json')
let cache = {}
try { cache = JSON.parse(fs.readFileSync(cachePath, 'utf8')) } catch (error) { if (error.code !== 'ENOENT' && !(error instanceof SyntaxError)) throw error }
const environment = { ...process.env, GOWORK: 'off', GOFLAGS: '', CGO_ENABLED: '0', GOOS: 'linux', GOARCH: architecture }
const flags = ['build', '-mod=readonly', '-trimpath', '-ldflags=-s -w']
const inputs = [execFileSync('go', ['version'], { encoding: 'utf8', env: environment }).trim(), architecture, flags, fileHash('go.mod'), fileHash('go.sum'), process.env.GOEXPERIMENT ?? '', process.env.GOAMD64 ?? '', process.env.GOARM64 ?? '']
const dependencies = execFileSync('go', ['list', '-mod=readonly', '-deps', '-f', '{{if .Module}}{{if .Module.Main}}{{.Dir}}{{end}}{{end}}', './cmd/teststudio-job'], { encoding: 'utf8', env: environment }).trim().split(/\r?\n/).filter(Boolean)
for (const dependency of dependencies.sort()) for (const name of fs.readdirSync(dependency).sort()) if (name.endsWith('.go') && !name.endsWith('_test.go')) inputs.push(name, fileHash(path.join(dependency, name)))
const binaryKey = hash(JSON.stringify(inputs)), binary = path.join(directory, 'teststudio-job')
if (cache.binaryKey !== binaryKey || cache.binaryHash !== fileHash(binary) || !fs.existsSync(binary)) {
  execFileSync('go', [...flags, '-o', binary, './cmd/teststudio-job'], { env: environment, stdio: ['ignore', 2, 2] })
} else console.error('Reusing the unchanged hosted supervisor binary.')
cache.binaryKey = binaryKey; cache.binaryHash = fileHash(binary)
const dockerfile = path.join(root, 'runners/job/Dockerfile')
const sourceKey = hash(JSON.stringify([base.Id, architecture, backend.version, cache.binaryHash, fileHash(dockerfile), fileHash('scripts/build-job-runtime.mjs'), fileHash('scripts/lib/docker-build.mjs')]))
const save = () => { const temporary = cachePath + '.' + process.pid + '.tmp'; fs.writeFileSync(temporary, JSON.stringify(cache, null, 2), { mode: 0o600 }); fs.renameSync(temporary, cachePath) }
save()
if (cache.sourceKey === sourceKey && cache.image) {
  try { const image = inspect(cache.image); if (image.Id === cache.image && image.Architecture === architecture && image.Config.User === '1000:1000' && image.Config.Labels?.['teststudio.task-storage'] === 'task-volumes/v1' && image.Config.Labels?.['teststudio.sandbox-supervisor'] === 'azure-sandbox-command/v1' && Object.keys(image.Config.Volumes ?? {}).sort().join(',') === '/output,/tmp') { console.error('Reusing the unchanged hosted source runtime.'); console.log(JSON.stringify({ image: cache.image, base: base.Id, runner, architecture, sourceKey })); process.exit(0) } } catch {}
}
const baseTag = 'teststudio/job-runtime-base:' + base.Id.slice(7)
execFileSync('docker', ['tag', base.Id, baseTag], { stdio: ['ignore', 'ignore', 2] })
const context = fs.mkdtempSync(path.join(os.tmpdir(), 'teststudio-job-runtime-'))
try {
  fs.copyFileSync(binary, path.join(context, 'teststudio-job'))
  execFileSync(backend.command, [...backend.prefix, '--network=none', '--pull=false', '--platform=linux/' + architecture, '--build-arg', 'RUNTIME_IMAGE=' + baseTag, '--build-arg', 'RUNTIME_DIGEST=' + base.Id, '--build-arg', 'SOURCE_KEY=' + sourceKey, '-q', '-f', dockerfile, '-t', 'teststudio/job-runtime:' + runner + '-' + architecture + '-local', context], { stdio: ['ignore', 2, 2] })
  const image = inspect('teststudio/job-runtime:' + runner + '-' + architecture + '-local')
  if (image.Architecture !== architecture || image.Config.User !== '1000:1000' || image.Config.Labels?.['teststudio.job-supervisor'] !== 'worker-exchange/v1' || image.Config.Labels?.['teststudio.discovery-supervisor'] !== 'worker-exchange/v1' || image.Config.Labels?.['teststudio.job-runtime-base'] !== base.Id || image.Config.Labels?.['teststudio.task-storage'] !== 'task-volumes/v1' || image.Config.Labels?.['teststudio.sandbox-supervisor'] !== 'azure-sandbox-command/v1' || Object.keys(image.Config.Volumes ?? {}).sort().join(',') !== '/output,/tmp') throw new Error('The hosted runtime metadata does not match its source package')
  cache.image = image.Id; cache.sourceKey = sourceKey; save()
  console.log(JSON.stringify({ image: image.Id, base: base.Id, runner, architecture, sourceKey }))
} finally { fs.rmSync(context, { recursive: true, force: true }) }
