import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import { createRequire } from 'node:module'
import { spawnSync } from 'node:child_process'
import { fileURLToPath } from 'node:url'

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const sourceRoot = path.join(root, 'web/src/app')
const require = createRequire(path.join(root, 'web/package.json'))
const ts = require('typescript')
const tests = process.argv.slice(2).map(file => path.resolve(root, file))
if (!tests.length || tests.some(file => !file.startsWith(sourceRoot + path.sep) || !file.endsWith('.test.ts'))) throw new Error('Select one or more web/src/app/*.test.ts files')
const output = fs.mkdtempSync(path.join(os.tmpdir(), 'teststudio-ui-utils-'))
const compiled = new Map()
function compile(file) {
  if (!file.startsWith(sourceRoot + path.sep)) throw new Error('Utility dependencies must stay inside web/src/app')
  if (compiled.has(file)) return compiled.get(file)
  const target = path.join(output, path.relative(sourceRoot, file).replace(/\.ts$/, '.cjs'))
  compiled.set(file, target)
  const result = ts.transpileModule(fs.readFileSync(file, 'utf8'), { compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022, esModuleInterop: true }, fileName: file })
  const code = result.outputText.replace(/require\((["'])(.*?)\1\)/g, (match, quote, name) => {
    if (name.startsWith('node:')) return match
    if (!name.startsWith('.')) throw new Error(`Pure utilities cannot load runtime package ${name}`)
    const dependency = path.resolve(path.dirname(file), name)
    const source = fs.existsSync(dependency) ? dependency : dependency + '.ts'
    const built = compile(source)
    return `require(${JSON.stringify('./' + path.relative(path.dirname(target), built))})`
  })
  fs.mkdirSync(path.dirname(target), { recursive: true })
  fs.writeFileSync(target, code)
  return target
}
try {
  const result = spawnSync(process.execPath, ['--test', ...tests.map(compile)], { cwd: root, stdio: 'inherit' })
  if (result.error) throw result.error
  process.exitCode = result.status ?? 1
} finally { fs.rmSync(output, { recursive: true, force: true }) }
