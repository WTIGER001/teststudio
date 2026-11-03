import fs from 'node:fs'
import path from 'node:path'
import { createHash } from 'node:crypto'

function digestUI(root) {
  const hash = createHash('sha256')
  function visit(filename) {
    const stat = fs.lstatSync(filename)
    if (stat.isSymbolicLink()) throw new Error('Demo UI files cannot contain symbolic links')
    if (stat.isDirectory()) { for (const name of fs.readdirSync(filename).sort()) visit(path.join(filename, name)); return }
    if (!stat.isFile()) throw new Error('Demo UI assets require regular files')
    hash.update(path.relative(root, filename).split(path.sep).join('/') + '\0'); hash.update(fs.readFileSync(filename)); hash.update('\0')
  }
  if (!fs.lstatSync(root).isDirectory()) throw new Error('Demo UI build must use a directory')
  visit(root)
  if (!fs.statSync(path.join(root, 'index.html')).isFile()) throw new Error('Demo UI build requires index.html')
  return hash.digest('hex')
}

export function snapshotDemoUI(source, directory) {
  const digest = digestUI(source), base = path.join(directory, 'ui'), target = path.join(base, digest)
  if (fs.existsSync(base) && (fs.lstatSync(base).isSymbolicLink() || !fs.lstatSync(base).isDirectory())) throw new Error('Demo UI cache requires a regular directory')
  fs.mkdirSync(base, { recursive: true, mode: 0o700 })
  if (fs.existsSync(target)) {
    if (digestUI(target) !== digest) throw new Error('The saved demo UI changed; use a new demo data directory')
    return target
  }
  const temporary = fs.mkdtempSync(path.join(base, '.copy-'))
  try {
    fs.cpSync(source, temporary, { recursive: true, dereference: false, force: false, errorOnExist: true })
    if (digestUI(temporary) !== digest || digestUI(source) !== digest) throw new Error('The UI build changed during demo setup; start again after its build completes')
    fs.renameSync(temporary, target)
    return target
  } finally { fs.rmSync(temporary, { recursive: true, force: true }) }
}
