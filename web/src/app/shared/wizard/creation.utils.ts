export function suggestedIdentifier(name: string, suffix: string): string {
  const slug = name.normalize('NFKD').replace(/[\u0300-\u036f]/g, '').toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '').slice(0, 96).replace(/-$/g, '')
  return name.trim() ? `${slug || 'item'}-${suffix}` : ''
}

export function creationMatches(expected: unknown, actual: unknown): boolean {
  if (actual === undefined && (expected === false || expected === 0 || expected === '')) return true
  if (expected === undefined) return true
  if (Array.isArray(expected)) return Array.isArray(actual) && expected.length === actual.length && expected.every((item, index) => creationMatches(item, actual[index]))
  if (expected !== null && typeof expected === 'object') return actual !== null && typeof actual === 'object' && Object.entries(expected).every(([key, value]) => creationMatches(value, (actual as Record<string, unknown>)[key]))
  return expected === actual
}

export function validWebAddress(value: string): boolean {
  try { const address = new URL(value); return ['https:', 'http:'].includes(address.protocol) && !!address.hostname && !address.username && !address.password && !address.hash && !/[\x00-\x1f\x7f]/.test(value) && value.length <= 2048 }
  catch { return false }
}
