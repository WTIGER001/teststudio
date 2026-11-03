export function sourceFormat(name: string): 'json' | 'yaml' { return /\.ya?ml$/i.test(name) ? 'yaml' : 'json' }
export function sourceDefinition(value: unknown): string { return JSON.stringify(value, null, 2) }
