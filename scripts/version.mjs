import fs from 'node:fs'
import { fileURLToPath } from 'node:url'
import { nextVersion } from './lib/version.mjs'

process.chdir(fileURLToPath(new URL('..', import.meta.url)))
const [action, tag, ...extra] = process.argv.slice(2)
if (extra.length || (action !== 'tag' && tag !== undefined)) throw new Error('Use node scripts/version.mjs major|minor|patch or node scripts/version.mjs tag TAG.')
const version = nextVersion(fs.readFileSync('VERSION', 'utf8'), action, tag)
const package_ = JSON.parse(fs.readFileSync('web/package.json', 'utf8'))
const lock = JSON.parse(fs.readFileSync('web/package-lock.json', 'utf8'))
package_.version = lock.version = lock.packages[''].version = version
fs.writeFileSync('VERSION', version + '\n')
fs.writeFileSync('web/package.json', JSON.stringify(package_, null, 2) + '\n')
fs.writeFileSync('web/package-lock.json', JSON.stringify(lock, null, 2) + '\n')
console.log(version)
