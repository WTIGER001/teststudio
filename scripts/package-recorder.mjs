import fs from 'node:fs'
import path from 'node:path'
import crypto from 'node:crypto'
import { execFileSync } from 'node:child_process'
import { fileURLToPath } from 'node:url'

const workspace = fileURLToPath(new URL('..', import.meta.url))
const options = { platform: process.platform, architecture: process.arch === 'x64' ? 'amd64' : process.arch, output: path.join(workspace, 'output/recorder') }
for (let index = 2; index < process.argv.length; index += 2) {
  const key = process.argv[index].replace(/^--/, '')
  if (!['platform', 'architecture', 'output'].includes(key) || !process.argv[index].startsWith('--') || !process.argv[index + 1]) throw new Error('Use --platform darwin|linux, --architecture arm64|amd64, and optional --output DIRECTORY')
  options[key] = process.argv[index + 1]
}
if (!['darwin', 'linux'].includes(options.platform) || !['arm64', 'amd64'].includes(options.architecture)) throw new Error('The recorder package supports macOS and Linux on arm64 or amd64')
const read = filename => fs.readFileSync(path.join(workspace, filename))
const pkg = JSON.parse(read('runners/playwright/package.json'))
const lock = JSON.parse(read('runners/playwright/package-lock.json'))
const version = pkg.dependencies['@playwright/test']
const declared = /const PlaywrightVersion = "([^"]+)"/.exec(read('internal/recording/models.go').toString())?.[1]
if (!/^\d+\.\d+\.\d+$/.test(version) || declared !== version || lock.packages[''].dependencies['@playwright/test'] !== version) throw new Error('Recorder and locked Playwright versions must match')
fs.mkdirSync(path.resolve(options.output), { recursive: true })
const output = fs.mkdtempSync(path.join(path.resolve(options.output), 'package-'))
const name = `teststudio-recorder-${options.platform}-${options.architecture}`
const directory = path.join(output, name)
fs.mkdirSync(directory, { mode: 0o755 })
const files = []
function put(relative, bytes, mode = 0o644) {
  const target = path.join(directory, relative)
  fs.mkdirSync(path.dirname(target), { recursive: true, mode: 0o755 })
  fs.writeFileSync(target, bytes, { mode })
  files.push({ path: relative, bytes: bytes.length, sha256: crypto.createHash('sha256').update(bytes).digest('hex') })
}
const binary = path.join(directory, 'teststudio-recorder')
execFileSync('go', ['build', '-mod=readonly', '-trimpath', '-o', binary, './cmd/teststudio-recorder'], { cwd: workspace, stdio: ['ignore', 2, 2], env: { ...process.env, GOWORK: 'off', CGO_ENABLED: '0', GOOS: options.platform, GOARCH: options.architecture } })
fs.chmodSync(binary, 0o755)
put('teststudio-recorder', fs.readFileSync(binary), 0o755)
for (const file of ['scripts/record-browser.mjs', 'runners/playwright/package.json', 'runners/playwright/package-lock.json']) put(file, read(file))
put('setup-recorder.mjs', Buffer.from(`import { execFileSync } from 'node:child_process'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

if (Number(process.versions.node.split('.')[0]) < 22) throw new Error('Install Node.js 22 or later before setting up the recorder')
const directory = path.dirname(fileURLToPath(import.meta.url))
const npm = process.platform === 'win32' ? 'npm.cmd' : 'npm'
execFileSync(npm, ['ci', '--prefix', 'runners/playwright', '--ignore-scripts', '--no-audit', '--no-fund'], { cwd: directory, stdio: 'inherit' })
const cli = path.join(directory, 'runners/playwright/node_modules/playwright/cli.js')
const version = execFileSync(process.execPath, [cli, '--version'], { encoding: 'utf8' }).trim()
if (version !== ${JSON.stringify('Version '+version)}) throw new Error('Installed Playwright version does not match the trusted recorder')
execFileSync(process.execPath, [cli, 'install', 'chromium'], { stdio: 'inherit' })
console.log('Setup complete. From this directory, run: ./teststudio-recorder --server https://your-teststudio.example')
`))
put('README.txt', Buffer.from(`TestStudio desktop recorder (${options.platform}/${options.architecture})

Use this package only when it comes from your trusted TestStudio operator.
Use the package that matches your desktop operating system and processor.

1. Extract the archive and open a terminal in this directory.
2. Install Node.js 22 or later, including npm. Go is not required.
3. Run: node setup-recorder.mjs
   Setup installs the locked Playwright ${version} dependencies and Chromium.
   Linux desktops can need operator-installed browser system libraries.
4. In TestStudio, choose Create test, Playwright browser test, and a suite.
   Configure the title and start address. Select Continue to desktop connection.
5. Run the displayed ./teststudio-recorder command from this directory.
6. Paste Connection code into its prompt. Do not pass the code as an argument.
7. Record actions and assertions in the dedicated browser and Inspector.
8. Select Stop and keep recording in TestStudio. Review and save the returned source.

Run ./teststudio-recorder --help for Advanced command settings.
Use --resume SESSION_ID to recover retained source without opening another browser.
The companion stops on connection failure, lost authority, cancellation, or deadline.
It preserves unrelated browser windows. Credentials and source stay in private files.
Saving a draft, preparation, publication, and execution remain separate actions.

manifest.json records the checksums of package files before dependency installation.
Keep scripts/record-browser.mjs and runners/playwright beside the command.
`))
const manifest = {
  schema: 1, platform: options.platform, architecture: options.architecture, playwrightVersion: version,
  sourceRevision: execFileSync('git', ['rev-parse', 'HEAD'], { cwd: workspace, encoding: 'utf8' }).trim(),
  sourceDirty: Boolean(execFileSync('git', ['status', '--porcelain'], { cwd: workspace, encoding: 'utf8' }).trim()),
  compiler: execFileSync('go', ['version'], { encoding: 'utf8' }).trim(), files
}
fs.writeFileSync(path.join(directory, 'manifest.json'), JSON.stringify(manifest, null, 2)+'\n')
const archive = path.join(output, `${name}.tar.gz`)
execFileSync('tar', ['-czf', archive, '-C', output, name], { env: { ...process.env, COPYFILE_DISABLE: '1' } })
const checksum = crypto.createHash('sha256').update(fs.readFileSync(archive)).digest('hex')
fs.writeFileSync(archive+'.sha256', checksum+'  '+path.basename(archive)+'\n')
console.log(JSON.stringify({ directory, archive, checksum, manifest: path.join(directory, 'manifest.json') }))
