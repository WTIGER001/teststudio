import fs from 'node:fs'
import path from 'node:path'
import crypto from 'node:crypto'
import { fileURLToPath } from 'node:url'

const root = fileURLToPath(new URL('..', import.meta.url))
const dependency = path.join(root, 'web/node_modules/playwright-core')
const version = JSON.parse(fs.readFileSync(path.join(dependency, 'package.json'), 'utf8')).version
if (version !== '1.64.1') throw new Error('The trace viewer must match the pinned Playwright 1.64.1 runtime')
const source = path.join(dependency, 'lib/vite/traceViewer')
const target = path.join(root, 'web/dist/trace-viewer')
const files = new Map()
function collect(directory, prefix = '') {
  for (const name of fs.readdirSync(directory).sort()) {
    const filename = path.join(directory, name), info = fs.lstatSync(filename), relative = prefix + name
    if (info.isSymbolicLink()) throw new Error('Trace viewer inputs must not contain symbolic links')
    if (info.isDirectory()) collect(filename, relative + '/')
    else if (info.isFile()) files.set(relative === 'sw.bundle.js' ? 'sw.vendor.js' : relative, fs.readFileSync(filename))
    else throw new Error('Trace viewer inputs must be regular files')
  }
}
collect(source)
for (const name of ['LICENSE', 'NOTICE', 'ThirdPartyNotices.txt']) files.set(name, fs.readFileSync(path.join(dependency, name)))
files.set('sw.bundle.js', Buffer.from(fs.readFileSync(path.join(root, 'runners/trace-viewer/sw-wrapper.js'), 'utf8').replace('__TESTSTUDIO_ASSETS__', JSON.stringify([...files.keys(), 'teststudio-session.js']))))
files.set('teststudio-session.js', fs.readFileSync(path.join(root, 'runners/trace-viewer/session.js')))
files.set('index.html', Buffer.from(files.get('index.html').toString().replace('</head>', '<script type="module" src="./teststudio-session.js"></script></head>')))
const hashes = Object.fromEntries([...files].map(([name, data]) => [name, crypto.createHash('sha256').update(data).digest('hex')]))
const manifest = JSON.stringify({ schemaVersion: 1, playwrightVersion: version, files: hashes }, null, 2) + '\n'
let unchanged = false
try { unchanged = fs.readFileSync(path.join(target, 'teststudio-viewer.json'), 'utf8') === manifest && [...files].every(([name, data]) => fs.readFileSync(path.join(target, name)).equals(data)) } catch {}
if (!unchanged) {
  fs.rmSync(target, { recursive: true, force: true })
  for (const [name, data] of files) { const filename = path.join(target, name); fs.mkdirSync(path.dirname(filename), { recursive: true }); fs.writeFileSync(filename, data) }
  fs.writeFileSync(path.join(target, 'teststudio-viewer.json'), manifest)
}
console.error(unchanged ? 'Reusing the unchanged private trace viewer.' : 'Packaged the pinned private trace viewer.')
console.log(target)
