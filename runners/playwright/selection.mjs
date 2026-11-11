import fs from 'node:fs'
import path from 'node:path'

export function selectionArguments(context, test, filename = '/tmp/teststudio-selection.txt') {
  if (!/^@[A-Za-z0-9][A-Za-z0-9._-]{0,127}$/.test(test.selector)) throw new Error('Invalid permanent test selector')
  if (!test.framework) return ['--grep', `${test.selector.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}(?:\\s|$)`]
  const mapping = test.framework
  if (!Array.isArray(mapping.titlePath) || !mapping.titlePath.length || mapping.titlePath.some(title => typeof title !== 'string' || !title || /[\r\n›>\0]/.test(title)) || typeof mapping.project !== 'string' || mapping.project.length > 200 || /[\[\]\r\n›>\0]/.test(mapping.project) || typeof test.source !== 'string' || /[\r\n›>\0\\]/.test(test.source)) throw new Error('Invalid framework test mapping')
  const source = path.resolve(context.root, test.source)
  if (!source.startsWith(context.root + path.sep)) throw new Error('Framework source escapes its context')
  fs.writeFileSync(filename, `${mapping.project ? `[${mapping.project}] › ` : ''}${source} › ${mapping.titlePath.join(' › ')}\n`, { mode: 0o600 })
  return ['--test-list', filename, '--no-deps']
}
