export function accessPath(path: string, parameters: Record<string, unknown>, query: Record<string, string | number | boolean | undefined> = {}): string {
  const resolved = path.replace(/\{([A-Za-z][A-Za-z0-9]*)\}/g, (_, name: string) => {
    const value = parameters[name]
    if (typeof value !== 'string' || !value) throw new Error(`Missing access path parameter: ${name}`)
    return encodeURIComponent(value)
  })
  const search = new URLSearchParams()
  for (const [key, value] of Object.entries(query)) if (value !== undefined) search.set(key, String(value))
  return resolved + (search.size ? `?${search}` : '')
}
