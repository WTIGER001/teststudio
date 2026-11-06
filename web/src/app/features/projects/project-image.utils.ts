export function websiteAddress(value: string): string {
  try {
    const url = new URL(value)
    return ['https:', 'http:'].includes(url.protocol) && !url.username && !url.password && value.length <= 2048 ? url.href : ''
  } catch { return '' }
}
