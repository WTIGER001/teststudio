import fs from 'node:fs'
import { execFileSync } from 'node:child_process'
import { fileURLToPath } from 'node:url'

const root = fileURLToPath(new URL('..', import.meta.url))
const run = (args, cwd = root) => execFileSync('npm', args, { cwd, stdio: ['ignore', 2, 2] })
run(['--prefix', 'web', 'run', 'build:status'])
if (process.argv.includes('--consumer')) {
  run(['ci', '--ignore-scripts', '--install-links', '--no-audit', '--no-fund'], root + '/examples/angular-status')
  run(['run', 'build'], root + '/examples/angular-status')
  run(['run', 'verify:ssr'], root + '/examples/angular-status')
}
if (process.argv.includes('--pack')) {
  const output = root + '/output/angular-status'
  fs.mkdirSync(output, { recursive: true })
  run(['pack', './web/dist/status', '--pack-destination', output])
}
