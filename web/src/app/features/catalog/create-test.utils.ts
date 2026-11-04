export type CreationKind = 'playwright' | 'https' | 'go' | 'api'

export const creationChoices: { id: CreationKind; title: string; description: string; symbol: string }[] = [
  { id: 'playwright', title: 'Playwright browser test', description: 'Record a browser journey, add assertions, and review the generated test.', symbol: '▷' },
  { id: 'https', title: 'HTTPS probe', description: 'Check an endpoint, response content, status codes, and TLS conditions.', symbol: '↗' },
  { id: 'api', title: 'API test', description: 'Configure a guarded request, then add coded request steps and assertions.', symbol: '⇄' },
  { id: 'go', title: 'Go suite', description: 'Prepare Go tests from a complete source bundle or registered repository.', symbol: '⌘' }
]

export function creationKind(value: string | null): CreationKind | '' | 'unknown' {
  if (!value) return ''
  return value === 'playwright' || value === 'https' || value === 'go' || value === 'api' ? value : 'unknown'
}
