export function bundleFileError(file: { name: string; size: number } | null): string {
  if (!file) return 'Choose a complete ZIP source bundle.'
  if (!file.name.toLowerCase().endsWith('.zip')) return 'Choose a ZIP file.'
  if (file.size < 1 || file.size > 32 * 1024 * 1024) return 'The ZIP must contain data and cannot exceed 32 MiB.'
  return ''
}
