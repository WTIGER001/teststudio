import fs from 'node:fs'
import path from 'node:path'
import crypto from 'node:crypto'
import { execFileSync } from 'node:child_process'
import { fileURLToPath } from 'node:url'

process.chdir(fileURLToPath(new URL('..', import.meta.url)))
const args = process.argv.slice(2)
if (args.length !== 1 || !/^--image=sha256:[a-f0-9]{64}$/.test(args[0])) throw new Error('Select a cached immutable browser runtime with --image=sha256:DIGEST')
const base = args[0].slice(8)
const inspect = image => JSON.parse(execFileSync('docker', ['image', 'inspect', image], { encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe'] }))[0]
const selected = inspect(base)
if (selected.Config.Labels?.['teststudio.playwright-version'] !== '1.64.1' || selected.Os !== 'linux') throw new Error('Recording requires the trusted pinned Linux browser runtime')
const directory = path.resolve('.data/recording-worker')
fs.mkdirSync(directory, { recursive: true, mode: 0o700 })
const policy = path.join(directory, 'seccomp.json')
const hash = data => crypto.createHash('sha256').update(data).digest('hex')
const upstreamHash = 'cc3e61cabda6bbc1e53e54d27ba4d55a9d3be829b6dd1a596f4a7b31b1cc7849'
const policyHash = 'c4fef485853f4abe05f74b8a5bb8ded0f0df029d463886c1e3b3175976fefb58'
if (!fs.existsSync(policy) || hash(fs.readFileSync(policy)) !== policyHash) {
  const response = await fetch('https://raw.githubusercontent.com/microsoft/playwright/v1.64.1/utils/docker/seccomp_profile.json', { signal: AbortSignal.timeout(15000) })
  if (!response.ok) throw new Error('The pinned Chromium sandbox policy could not be downloaded')
  const bytes = Buffer.from(await response.arrayBuffer())
  if (hash(bytes) !== upstreamHash) throw new Error('The Chromium sandbox policy checksum does not match')
  const profile = JSON.parse(bytes)
  if (JSON.stringify(profile.syscalls[0].names) !== JSON.stringify(['clone', 'setns', 'unshare'])) throw new Error('The Chromium sandbox policy structure changed')
  profile.syscalls[0].names.push('chroot')
  const derived = JSON.stringify(profile, null, 2) + '\n'
  if (hash(derived) !== policyHash) throw new Error('The derived sandbox policy checksum does not match')
  fs.writeFileSync(policy, derived, { mode: 0o600 })
}
if (hash(fs.readFileSync(policy)) !== policyHash) throw new Error('The installed Chromium sandbox policy checksum does not match')
const files = ['scripts/record-browser-embedded.mjs', 'scripts/recording-proxy.mjs', 'deploy/recording/Dockerfile', 'scripts/build-recording-runtime.mjs']
const key = hash(JSON.stringify([base, ...files.map(file => hash(fs.readFileSync(file)))]))
const cachePath = path.join(directory, 'runtime.json')
let cache
try { cache = JSON.parse(fs.readFileSync(cachePath, 'utf8')) } catch {}
if (cache?.key === key) {
  try { if (inspect(cache.image).Id === cache.image) { console.error('Reusing the unchanged isolated recording runtime.'); console.log(JSON.stringify(cache)); process.exit(0) } } catch {}
}
const context = fs.mkdtempSync(path.join(directory, 'context-'))
try {
  for (const file of files.slice(0, 2)) fs.copyFileSync(file, path.join(context, path.basename(file)))
  const tag = 'teststudio/recording-base:' + base.slice(7)
  execFileSync('docker', ['tag', base, tag], { stdio: ['ignore', 'ignore', 2] })
  execFileSync('docker', ['build', '--network=none', '--pull=false', '--build-arg', 'RUNTIME_IMAGE='+tag, '-f', path.resolve('deploy/recording/Dockerfile'), '-t', 'teststudio/recording-runtime:local', context], { stdio: ['ignore', 2, 2] })
  const image = inspect('teststudio/recording-runtime:local')
  if (image.Config.User !== '1000:1000' || image.Config.Labels?.['teststudio.recording-runtime'] !== 'isolated-browser/v1') throw new Error('The recording runtime metadata does not match')
  cache = { key, image: image.Id, base, architecture: selected.Architecture, seccomp: policy, policyHash }
  fs.writeFileSync(cachePath, JSON.stringify(cache, null, 2), { mode: 0o600 })
  console.log(JSON.stringify(cache))
} finally { fs.rmSync(context, { recursive: true, force: true }) }
