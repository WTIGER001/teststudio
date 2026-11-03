import type { EditorFile } from '../../core/generated/access.types'

export function replaceEditorFile(files: EditorFile[], path: string, content: string): EditorFile[] {
  return files.map(file => file.path === path ? { ...file, content } : file)
}

export function editorPathError(path: string, files: EditorFile[]): string {
  if (!path || path.length > 512 || /[\\:\x00-\x1f\x7f]/.test(path) || path.split('/').some(part => !part || part === '.' || part === '..')) return 'Use a relative file path, such as checks/another.spec.ts.'
  const key = path.toLowerCase()
  if (['package.json', 'package-lock.json', 'teststudio-suite.json', 'teststudio-origin.json', 'node_modules'].includes(key) || key.startsWith('node_modules/')) return 'TestStudio manages package and manifest files.'
  if (files.some(file => { const existing = file.path.toLowerCase(); return existing === key || existing.startsWith(`${key}/`) || key.startsWith(`${existing}/`) })) return 'That path already exists or conflicts with another file.'
  if (files.length >= 50) return 'A draft supports up to 50 files.'
  return ''
}
