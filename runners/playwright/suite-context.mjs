import fs from 'node:fs'
import path from 'node:path'

function file(root, relative, directory = false) {
  if (typeof relative !== 'string' || !relative || relative.includes('\\') || relative.includes('\0') || path.posix.isAbsolute(relative) || path.posix.normalize(relative) !== relative || relative === '..' || relative.startsWith('../')) throw new Error('Suite file is outside its source context')
  const realRoot = fs.realpathSync(root), resolved = fs.realpathSync(path.join(root, relative))
  const inside = path.relative(realRoot, resolved)
  if (inside === '..' || inside.startsWith(`..${path.sep}`) || path.isAbsolute(inside)) throw new Error('Suite file escapes its source context')
  const info = fs.statSync(resolved)
  if (directory ? !info.isDirectory() : !info.isFile()) throw new Error('Suite context requires a regular file or module directory')
  return resolved
}
function text(root, relative) {
  const filename = file(root, relative)
  if (fs.statSync(filename).size > 1 << 20) throw new Error('Suite declaration exceeds 1 MiB')
  const value = fs.readFileSync(filename)
  if (value.length > 1 << 20) throw new Error('Suite declaration exceeds 1 MiB')
  return new TextDecoder('utf-8', { fatal: true }).decode(value)
}
export function suiteContext(root = '/suite', required = false) {
  if (!fs.existsSync(path.join(root, 'teststudio-suite.json'))) {
    if (required) throw new Error('Prepared suite manifest is unavailable')
    return { root, module: root, config: path.join(root, 'playwright.config.ts'), project: '' }
  }
  const manifest = JSON.parse(text(root, 'teststudio-suite.json'))
  const origin = JSON.parse(text(root, 'teststudio-origin.json'))
  if (manifest.schemaVersion !== 1 || manifest.runner !== 'playwright' || !manifest.tests || typeof manifest.tests !== 'object') throw new Error('Playwright suite manifest is invalid')
  const module = file(root, manifest.moduleRoot, true)
  const config = file(module, manifest.tests.config || 'playwright.config.ts')
  const project = manifest.tests.project || ''
  if (typeof project !== 'string' || project.length > 200 || /[\[\]\r\n›>\0]/.test(project)) throw new Error('Playwright discovery requires one exact project identifier')
  return { root: fs.realpathSync(root), module, config, project, manifest, origin, catalogText: text(root, manifest.catalog) }
}
