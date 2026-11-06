export type ThemePreference = 'light' | 'dark' | 'system'

export function themePreference(value: string | null): ThemePreference {
  return value === 'light' || value === 'dark' ? value : 'system'
}

export function resolvedTheme(preference: ThemePreference, systemDark: boolean): 'light' | 'dark' {
  return preference === 'system' ? (systemDark ? 'dark' : 'light') : preference
}

export function projectFromURL(url: string): string {
  if (/^\/projects\/new(?:[?#]|$)/.test(url)) return ''
  const match = /^\/projects\/([A-Za-z0-9._-]+)(?:\/|$)/.exec(url)
  return match?.[1] ?? ''
}
