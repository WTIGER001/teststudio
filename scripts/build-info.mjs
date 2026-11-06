import fs from 'node:fs'
import path from 'node:path'
import crypto from 'node:crypto'
import { execFileSync } from 'node:child_process'
import { fileURLToPath } from 'node:url'
import { parseVersion } from './lib/version.mjs'

process.chdir(fileURLToPath(new URL('..', import.meta.url)))
if (process.argv.slice(2).some(value => value !== '--refresh') || process.argv.length > 3) throw new Error('Use node scripts/build-info.mjs [--refresh].')
const release = parseVersion(fs.readFileSync('VERSION', 'utf8'))
for (const file of ['web/package.json', 'web/package-lock.json']) {
  const value = JSON.parse(fs.readFileSync(file, 'utf8'))
  if (value.version !== release.version || value.packages?.[''] && value.packages[''].version !== release.version) throw new Error('Version files disagree. Use scripts/version.mjs to update the release.')
}
const generated = 'web/src/app/core/generated/build-info.ts'
function relevant(file) {
  return file !== generated && !/\.(?:spec|test)\.ts$|_test\.go$/.test(file) && (['VERSION', 'go.mod', 'go.sum', '.dockerignore'].includes(file) || /^(?:web\/(?:src|public)\/|web\/(?:package(?:-lock)?\.json|angular\.json|tsconfig.*\.json)$|internal\/|adapters\/|cmd\/|scripts\/|deploy\/service\/)/.test(file))
}
const files = execFileSync('git', ['ls-files', '--cached', '--others', '--exclude-standard', '-z'], { encoding: 'utf8' }).split('\0').filter(file => file && relevant(file) && fs.existsSync(file)).sort()
const source = crypto.createHash('sha256')
for (const file of files) { source.update(file + '\0'); source.update(fs.readFileSync(file)); source.update('\0') }
const sourceHash = source.digest('hex')
const commit = execFileSync('git', ['rev-parse', 'HEAD'], { encoding: 'utf8' }).trim()
const changed = execFileSync('git', ['status', '--porcelain=v1', '--untracked-files=all', '-z'], { encoding: 'utf8' }).split('\0').filter(Boolean)
const modified = changed.some(row => relevant(row.slice(3)))
const fingerprint = crypto.createHash('sha256').update(JSON.stringify({ release, sourceHash, commit, modified })).digest('hex')
const cache = '.data/build-info.json'
let previous
try { previous = JSON.parse(fs.readFileSync(cache, 'utf8')) } catch (error) { if (error.code !== 'ENOENT' && !(error instanceof SyntaxError)) throw error }
let info = previous?.fingerprint === fingerprint && !process.argv.includes('--refresh') ? previous.info : undefined
if (!info) {
  const builtAt = new Date().toISOString()
  const buildId = crypto.createHash('sha256').update(fingerprint + builtAt).digest('hex').slice(0, 12)
  info = { ...release, buildId, builtAt, commit, sourceHash, modified }
  fs.mkdirSync('.data', { recursive: true })
  const temporary = cache + '.' + process.pid
  fs.writeFileSync(temporary, JSON.stringify({ fingerprint, info }, null, 2) + '\n')
  fs.renameSync(temporary, cache)
}
const content = 'export const buildInfo = ' + JSON.stringify(info, null, 2) + ' as const\n'
fs.mkdirSync(path.dirname(generated), { recursive: true })
if (!fs.existsSync(generated) || fs.readFileSync(generated, 'utf8') !== content) {
  const temporary = generated + '.' + process.pid
  fs.writeFileSync(temporary, content); fs.renameSync(temporary, generated)
}
console.log(JSON.stringify(info))
